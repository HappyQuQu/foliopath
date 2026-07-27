package thumbnail

import (
	"context"
	"errors"
	"time"

	"github.com/HappyQuQu/foliopath/internal/media"
)

const (
	MediaWorkerCount = 2
	MaxJobAttempts   = 3
)

var (
	ErrInvalidJob   = errors.New("invalid thumbnail job")
	ErrJobNotActive = errors.New("thumbnail job is not active")
)

type Job struct {
	ID                int64
	LibraryID         int64
	AssetID           int64
	TransformVersion  int
	SourceFingerprint media.SourceFingerprint
	Attempt           int
}

type JobOutcome string

const (
	JobSucceeded JobOutcome = "succeeded"
	JobPermanent JobOutcome = "permanent_failure"
	JobRetry     JobOutcome = "retry"
	JobStale     JobOutcome = "stale"
)

type JobErrorCode string

const (
	JobErrorInvalidMedia     JobErrorCode = "invalid_media"
	JobErrorUnsupportedMedia JobErrorCode = "unsupported_media"
	JobErrorProcessing       JobErrorCode = "media_processing_failed"
	JobErrorTimeout          JobErrorCode = "media_processing_timeout"
	JobErrorSource           JobErrorCode = "source_unavailable"
	JobErrorCache            JobErrorCode = "cache_unavailable"
)

type JobResult struct {
	Outcome    JobOutcome
	Code       JobErrorCode
	RetryDelay time.Duration
}

type JobCompletionRepository interface {
	FinishMediaJob(context.Context, Job, JobResult) error
}

type ClaimedProcessor struct {
	service    *Service
	repository JobCompletionRepository
}

func NewClaimedProcessor(
	service *Service,
	repository JobCompletionRepository,
) (*ClaimedProcessor, error) {
	if service == nil || repository == nil {
		return nil, errors.New("claimed thumbnail processor dependencies are required")
	}
	return &ClaimedProcessor{service: service, repository: repository}, nil
}

func (processor *ClaimedProcessor) Process(ctx context.Context, job Job) error {
	if job.TransformVersion != GridTransformVersion {
		return processor.repository.FinishMediaJob(
			ctx, job, JobResult{Outcome: JobStale},
		)
	}
	err := processor.service.Process(ctx, job.AssetID)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	result := classifyJobResult(err)
	if finishErr := processor.repository.FinishMediaJob(ctx, job, result); finishErr != nil {
		return errors.Join(err, finishErr)
	}
	return err
}

func classifyJobResult(err error) JobResult {
	switch {
	case err == nil:
		return JobResult{Outcome: JobSucceeded}
	case errors.Is(err, ErrSourceChanged):
		return JobResult{Outcome: JobStale}
	case errors.Is(err, media.ErrInvalidMedia):
		return JobResult{Outcome: JobPermanent, Code: JobErrorInvalidMedia}
	case errors.Is(err, media.ErrUnsupportedMedia):
		return JobResult{Outcome: JobPermanent, Code: JobErrorUnsupportedMedia}
	case errors.Is(err, context.DeadlineExceeded),
		errors.Is(err, media.ErrProcessingTimedOut):
		return JobResult{
			Outcome: JobRetry, Code: JobErrorTimeout, RetryDelay: 5 * time.Second,
		}
	case errors.Is(err, ErrSourceUnavailable):
		return JobResult{
			Outcome: JobRetry, Code: JobErrorSource, RetryDelay: 5 * time.Second,
		}
	case errors.Is(err, ErrPublishFailed),
		errors.Is(err, ErrCacheCapacity):
		return JobResult{
			Outcome: JobRetry, Code: JobErrorCache, RetryDelay: 5 * time.Second,
		}
	default:
		return JobResult{
			Outcome: JobRetry, Code: JobErrorProcessing, RetryDelay: 5 * time.Second,
		}
	}
}
