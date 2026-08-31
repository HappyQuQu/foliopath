package semantic

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
	"github.com/HappyQuQu/foliopath/internal/jobs"
)

const DefaultTagReviewClearBatchSize = 500

var (
	ErrInvalidTagReviewClear  = errors.New("invalid tag review clear")
	ErrTagReviewClearConflict = errors.New("tag review clear conflict")
	ErrTagReviewClearNotFound = errors.New("tag review clear not found")
)

type TagReviewClearJob struct {
	ID                     string
	LibraryID              int64
	OperationID            string
	ExpectedReviewRevision int64
	State                  JobState
	DeletedCount           int64
	RequestedRevision      int64
	ClaimedRevision        int64
	AttemptCount           int
	LeaseExpiresAt         *time.Time
	ErrorCode              string
	OperationRevision      int64
	CompletedItems         int64
	TotalItems             int64
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

type TagReviewClearAdmission struct {
	IdempotencyKeyHash string
	RequestHash        string
	Job                TagReviewClearJob
}

type TagReviewClearResult struct {
	Job       TagReviewClearJob
	Created   bool
	Replayed  bool
	Coalesced bool
}

type TagReviewClearQueue interface {
	FindTagReviewClear(context.Context, string) (TagReviewClearAdmission, bool, error)
	CreateTagReviewClear(context.Context, TagReviewClearAdmission) (TagReviewClearAdmission, bool, bool, error)
	ClaimTagReviewClear(context.Context, time.Time, time.Duration) (TagReviewClearJob, bool, error)
	RefreshTagReviewClearLease(context.Context, TagReviewClearJob, time.Time, time.Duration) (bool, error)
	DeleteTagReviewClearBatch(context.Context, TagReviewClearJob, int, time.Time) (int64, bool, error)
	FinishTagReviewClear(context.Context, TagReviewClearJob, JobState, string, time.Time) (TagReviewClearJob, error)
	CancelTagReviewClearOperation(context.Context, string, int64, time.Time) (TagReviewClearJob, error)
	RecoverExpiredTagReviewClears(context.Context, time.Time) (jobs.RecoverySummary, error)
}

type TagReviewClearService struct {
	queue TagReviewClearQueue
	wake  interface{ Wake() }
	now   func() time.Time
	newID func(string) (string, error)
}

func NewTagReviewClearService(queue TagReviewClearQueue, wake interface{ Wake() }, now func() time.Time, newID func(string) (string, error)) (*TagReviewClearService, error) {
	if queue == nil || wake == nil {
		return nil, errors.New("tag review clear dependencies are required")
	}
	if now == nil {
		now = time.Now
	}
	if newID == nil {
		newID = randomSemanticID
	}
	return &TagReviewClearService{queue: queue, wake: wake, now: now, newID: newID}, nil
}

func (service *TagReviewClearService) Request(ctx context.Context, libraryID, expectedRevision int64, key string) (TagReviewClearResult, error) {
	if libraryID < 1 || expectedRevision < 1 || !semanticKeyPattern.MatchString(key) {
		return TagReviewClearResult{}, ErrInvalidTagReviewClear
	}
	keyHash := digestSemanticValue("foliopath:tag-review-clear-key:v1\x00" + key)
	requestHash := digestSemanticValue(fmt.Sprintf("foliopath:tag-review-clear:v1\x00%d\x00%d", libraryID, expectedRevision))
	if existing, found, err := service.queue.FindTagReviewClear(ctx, keyHash); err != nil {
		return TagReviewClearResult{}, err
	} else if found {
		if existing.RequestHash != requestHash {
			return TagReviewClearResult{}, ErrTagReviewClearConflict
		}
		return TagReviewClearResult{Job: existing.Job, Replayed: true}, nil
	}
	jobID, err := service.newID("tagclear")
	if err != nil {
		return TagReviewClearResult{}, err
	}
	operationID, err := service.newID("aio")
	if err != nil {
		return TagReviewClearResult{}, err
	}
	now := service.now().UTC()
	stored, created, coalesced, err := service.queue.CreateTagReviewClear(ctx, TagReviewClearAdmission{
		IdempotencyKeyHash: keyHash, RequestHash: requestHash,
		Job: TagReviewClearJob{ID: jobID, LibraryID: libraryID, OperationID: operationID,
			ExpectedReviewRevision: expectedRevision, State: JobQueued, RequestedRevision: 1,
			OperationRevision: 1, CreatedAt: now, UpdatedAt: now},
	})
	if err != nil {
		return TagReviewClearResult{}, err
	}
	if stored.RequestHash != requestHash {
		return TagReviewClearResult{}, ErrTagReviewClearConflict
	}
	if created {
		service.wake.Wake()
	}
	return TagReviewClearResult{Job: stored.Job, Created: created, Replayed: !created && !coalesced, Coalesced: coalesced}, nil
}

func (service *TagReviewClearService) CancelOperation(ctx context.Context, operationID string, revision int64) (TagReviewClearJob, error) {
	if operationID == "" || revision < 1 {
		return TagReviewClearJob{}, ErrInvalidTagReviewClear
	}
	return service.queue.CancelTagReviewClearOperation(ctx, operationID, revision, service.now().UTC())
}

type TagReviewClearProcessor struct {
	queue     TagReviewClearQueue
	now       func() time.Time
	batchSize int
}

func NewTagReviewClearProcessor(queue TagReviewClearQueue, now func() time.Time, batchSize int) (*TagReviewClearProcessor, error) {
	if queue == nil {
		return nil, errors.New("tag review clear queue is required")
	}
	if now == nil {
		now = time.Now
	}
	if batchSize == 0 {
		batchSize = DefaultTagReviewClearBatchSize
	}
	if batchSize < 1 || batchSize > 1000 {
		return nil, ErrInvalidTagReviewClear
	}
	return &TagReviewClearProcessor{queue: queue, now: now, batchSize: batchSize}, nil
}

func (processor *TagReviewClearProcessor) Process(ctx context.Context, job TagReviewClearJob) error {
	if job.ID == "" || job.LibraryID < 1 || job.OperationID == "" || job.State != JobRunning || job.ClaimedRevision < 1 {
		return ErrInvalidTagReviewClear
	}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		_, done, err := processor.queue.DeleteTagReviewClearBatch(ctx, job, processor.batchSize, processor.now().UTC())
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			_, finishErr := processor.queue.FinishTagReviewClear(ctx, job, JobFailed, "internal_error", processor.now().UTC())
			return errors.Join(err, finishErr)
		}
		if done {
			_, err = processor.queue.FinishTagReviewClear(ctx, job, JobSucceeded, "", processor.now().UTC())
			return err
		}
	}
}

func (job TagReviewClearJob) OperationKind() aimodel.OperationKind {
	return aimodel.OperationTagReviewClear
}

func ValidateTagReviewClearAdmission(value TagReviewClearAdmission) error {
	job := value.Job
	if len(value.IdempotencyKeyHash) != 64 || len(value.RequestHash) != 64 || job.ID == "" || job.LibraryID < 1 ||
		job.OperationID == "" || job.ExpectedReviewRevision < 1 || job.State != JobQueued || job.DeletedCount != 0 ||
		job.RequestedRevision != 1 || job.ClaimedRevision != 0 || job.AttemptCount != 0 || job.LeaseExpiresAt != nil ||
		job.ErrorCode != "" || job.OperationRevision != 1 || job.CompletedItems != 0 || job.TotalItems != 0 ||
		job.CreatedAt.IsZero() || !job.UpdatedAt.Equal(job.CreatedAt) {
		return ErrInvalidTagReviewClear
	}
	if _, err := hex.DecodeString(value.IdempotencyKeyHash); err != nil {
		return ErrInvalidTagReviewClear
	}
	if _, err := hex.DecodeString(value.RequestHash); err != nil {
		return ErrInvalidTagReviewClear
	}
	return nil
}
