package face

import (
	"context"
	"errors"
	"fmt"
	"github.com/HappyQuQu/foliopath/internal/aimodel"
	"github.com/HappyQuQu/foliopath/internal/jobs"
	"time"
)

const MaximumFaceJobAttempts = 3

var (
	ErrInvalidFaceJob  = errors.New("invalid face analysis job")
	ErrFaceJobConflict = errors.New("face analysis job conflict")
	ErrFaceJobNotFound = errors.New("face analysis job not found")
)

type JobLibraryState string

const (
	JobLibraryReady            JobLibraryState = "ready"
	JobLibraryOffline          JobLibraryState = "offline"
	JobLibraryNotReady         JobLibraryState = "not_ready"
	JobLibraryDisabled         JobLibraryState = "disabled"
	JobLibraryModelUnavailable JobLibraryState = "model_unavailable"
)

type JobMode string

const (
	JobMissing JobMode = "missing"
	JobAll     JobMode = "all"
)

type AnalysisJob struct {
	ID                                               string
	LibraryID                                        int64
	GenerationID, OperationID                        string
	Mode                                             JobMode
	State                                            string
	CheckpointID, RequestedRevision, ClaimedRevision int64
	AttemptCount                                     int
	LeaseExpiresAt                                   *time.Time
	ErrorCode                                        string
	OperationRevision, CompletedItems, TotalItems    int64
	CreatedAt, UpdatedAt                             time.Time
}

func (j AnalysisJob) OperationKind() aimodel.OperationKind {
	if j.Mode == JobAll {
		return aimodel.OperationFaceRebuild
	}
	return aimodel.OperationFaceMissing
}

type JobAdmission struct {
	IdempotencyKeyHash, RequestHash string
	EligibleCount                   int64
	Job                             AnalysisJob
}
type JobCandidateCounts struct{ Eligible, Pending int64 }
type JobCandidate struct {
	AssetID           int64
	SourceFingerprint string
}
type JobCandidatePage struct {
	Items   []JobCandidate
	HasMore bool
}
type JobCatalog interface {
	FaceJobLibraryState(context.Context, int64, string) (JobLibraryState, error)
	CountFaceJobCandidates(context.Context, int64, string, JobMode) (JobCandidateCounts, error)
	ListFaceJobCandidates(context.Context, int64, string, JobMode, int64, int) (JobCandidatePage, error)
}
type JobProgress struct {
	GenerationID                                                          string
	LibraryID, Eligible, Completed, Failed, Stale, CheckpointID, Revision int64
	UpdatedAt                                                             time.Time
}
type JobProgressCommit struct {
	JobID                                                                             string
	ClaimedRevision, ExpectedProgressRevision, ExpectedCheckpointID, NextCheckpointID int64
	SourceFingerprint                                                                 string
	Batch                                                                             ObservationBatch
	FailedCount, StaleCount                                                           int64
	UpdatedAt                                                                         time.Time
}
type JobQueue interface {
	FindFaceJob(context.Context, string) (JobAdmission, bool, error)
	CreateFaceJob(context.Context, JobAdmission) (JobAdmission, bool, bool, error)
	ClaimFaceJob(context.Context, time.Time, time.Duration) (AnalysisJob, bool, error)
	RefreshFaceJobLease(context.Context, AnalysisJob, time.Time, time.Duration) (bool, error)
	GetFaceJobProgress(context.Context, string, int64) (JobProgress, bool, error)
	CommitFaceJobProgress(context.Context, JobProgressCommit) (JobProgress, error)
	FinishFaceJob(context.Context, AnalysisJob, bool, string, time.Time) (AnalysisJob, error)
	CancelFaceJobOperation(context.Context, string, int64, time.Time) (AnalysisJob, error)
	RecoverExpiredFaceJobs(context.Context, time.Time) (jobs.RecoverySummary, error)
}
type JobAnalyzer interface {
	Analyze(context.Context, int64, int64, string) ([]Observation, error)
}
type ClusterRebuilder interface {
	RebuildFaceClusters(context.Context, string, int64, string, int64, ClusterProfile, time.Time) error
}
type JobResult struct {
	Job                          AnalysisJob
	Created, Replayed, Coalesced bool
}
type JobService struct {
	queue   JobQueue
	catalog JobCatalog
	wake    interface{ Wake() }
	now     func() time.Time
	newID   func(string) (string, error)
}
type JobProcessor struct {
	queue    JobQueue
	catalog  JobCatalog
	analyzer JobAnalyzer
	clusters ClusterRebuilder
	profile  ClusterProfile
	now      func() time.Time
	pageSize int
}

func NewJobService(queue JobQueue, catalog JobCatalog, wake interface{ Wake() }, now func() time.Time, newID func(string) (string, error)) (*JobService, error) {
	if queue == nil || catalog == nil || wake == nil {
		return nil, errors.New("face job dependencies are required")
	}
	if now == nil {
		now = time.Now
	}
	if newID == nil {
		newID = randomFaceID
	}
	return &JobService{queue: queue, catalog: catalog, wake: wake, now: now, newID: newID}, nil
}
func (s *JobService) Request(ctx context.Context, libraryID int64, generationID string, mode JobMode, key string) (JobResult, error) {
	if libraryID < 1 || len(generationID) < 8 || len(generationID) > 128 || (mode != JobMissing && mode != JobAll) || !reviewIdempotencyKeyPattern.MatchString(key) {
		return JobResult{}, ErrInvalidFaceJob
	}
	keyHash := faceDigest("foliopath:face-job-key:v1\x00" + key)
	requestHash := faceDigest(fmt.Sprintf("foliopath:face-job-request:v1\x00%d\x00%s\x00%s", libraryID, generationID, mode))
	if existing, found, err := s.queue.FindFaceJob(ctx, keyHash); err != nil {
		return JobResult{}, err
	} else if found {
		if existing.RequestHash != requestHash {
			return JobResult{}, ErrFaceJobConflict
		}
		return JobResult{Job: existing.Job, Replayed: true}, nil
	}
	libraryState, err := s.catalog.FaceJobLibraryState(ctx, libraryID, generationID)
	if err != nil {
		return JobResult{}, err
	}
	switch libraryState {
	case JobLibraryReady:
	case JobLibraryOffline:
		return JobResult{}, ErrFaceLibraryOffline
	case JobLibraryNotReady:
		return JobResult{}, ErrFaceNotReady
	case JobLibraryDisabled:
		return JobResult{}, ErrFaceDisabled
	case JobLibraryModelUnavailable:
		return JobResult{}, ErrFaceModelUnavailable
	default:
		return JobResult{}, ErrInvalidFaceJob
	}
	counts, err := s.catalog.CountFaceJobCandidates(ctx, libraryID, generationID, mode)
	if err != nil {
		return JobResult{}, err
	}
	if counts.Eligible < 0 || counts.Pending < 0 || counts.Pending > counts.Eligible {
		return JobResult{}, ErrInvalidFaceJob
	}
	jobID, err := s.newID("facejob")
	if err != nil {
		return JobResult{}, err
	}
	operationID, err := s.newID("aio")
	if err != nil {
		return JobResult{}, err
	}
	now := s.now().UTC()
	stored, created, coalesced, err := s.queue.CreateFaceJob(ctx, JobAdmission{IdempotencyKeyHash: keyHash, RequestHash: requestHash, EligibleCount: counts.Eligible, Job: AnalysisJob{ID: jobID, LibraryID: libraryID, GenerationID: generationID, OperationID: operationID, Mode: mode, State: "queued", RequestedRevision: 1, OperationRevision: 1, TotalItems: counts.Pending, CreatedAt: now, UpdatedAt: now}})
	if err != nil {
		return JobResult{}, err
	}
	if stored.RequestHash != requestHash {
		return JobResult{}, ErrFaceJobConflict
	}
	if created {
		s.wake.Wake()
	}
	return JobResult{Job: stored.Job, Created: created, Replayed: !created && !coalesced, Coalesced: coalesced}, nil
}
func (s *JobService) Cancel(ctx context.Context, operationID string, revision int64) (AnalysisJob, error) {
	if operationID == "" || revision < 1 {
		return AnalysisJob{}, ErrInvalidFaceJob
	}
	return s.queue.CancelFaceJobOperation(ctx, operationID, revision, s.now().UTC())
}

func NewJobProcessor(queue JobQueue, catalog JobCatalog, analyzer JobAnalyzer, clusters ClusterRebuilder, profile ClusterProfile, now func() time.Time, pageSize int) (*JobProcessor, error) {
	if queue == nil || catalog == nil || analyzer == nil || clusters == nil || profile.MinCoreSize < 2 || pageSize < 1 || pageSize > 1000 {
		return nil, ErrInvalidFaceJob
	}
	if now == nil {
		now = time.Now
	}
	return &JobProcessor{queue: queue, catalog: catalog, analyzer: analyzer, clusters: clusters, profile: profile, now: now, pageSize: pageSize}, nil
}

func (p *JobProcessor) Process(ctx context.Context, job AnalysisJob) error {
	if job.ID == "" || job.State != "running" || job.ClaimedRevision < 1 {
		return ErrInvalidFaceJob
	}
	progress, found, err := p.queue.GetFaceJobProgress(ctx, job.GenerationID, job.LibraryID)
	if err != nil || !found {
		if err == nil {
			err = ErrFaceJobConflict
		}
		return err
	}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		page, err := p.catalog.ListFaceJobCandidates(ctx, job.LibraryID, job.GenerationID, job.Mode, progress.CheckpointID, p.pageSize)
		if err != nil {
			return p.fail(ctx, job, err)
		}
		if len(page.Items) == 0 {
			stopped, err := p.refreshControl(ctx, job)
			if err != nil || stopped {
				return err
			}
			if rebuildErr := p.clusters.RebuildFaceClusters(ctx, job.GenerationID, job.LibraryID, job.ID, job.ClaimedRevision, p.profile, p.now().UTC()); rebuildErr != nil {
				// A disable or offline transition can race the bounded cluster build. Let
				// the canonical job state machine settle that transition before treating
				// the adapter error as an internal failure.
				stopped, controlErr := p.refreshControl(ctx, job)
				if stopped || controlErr != nil {
					return controlErr
				}
				return p.fail(ctx, job, rebuildErr)
			}
			// Recheck after the potentially long cluster build so cancellation wins
			// before the operation is published as successful.
			stopped, err = p.refreshControl(ctx, job)
			if err != nil || stopped {
				return err
			}
			_, err = p.queue.FinishFaceJob(ctx, job, true, "", p.now().UTC())
			return err
		}
		for _, candidate := range page.Items {
			stopped, err := p.refreshControl(ctx, job)
			if err != nil || stopped {
				return err
			}
			now := p.now().UTC()
			observations, analyzeErr := p.analyzer.Analyze(ctx, job.LibraryID, candidate.AssetID, candidate.SourceFingerprint)
			if ctx.Err() != nil {
				// A process shutdown or caller cancellation must leave the durable claim
				// recoverable; do not turn it into an ordinary per-asset failure.
				return ctx.Err()
			}
			if errors.Is(analyzeErr, ErrRuntimeUnavailable) {
				// A missing or faulted generation-bound runtime affects the whole job.
				// Stop before advancing the checkpoint or activating a partial cluster
				// build, and let the canonical terminal transition mark the library as
				// awaiting a model.
				return p.failWithCode(ctx, job, "model_unavailable", analyzeErr)
			}
			commit := JobProgressCommit{JobID: job.ID, ClaimedRevision: job.ClaimedRevision,
				ExpectedProgressRevision: progress.Revision, ExpectedCheckpointID: progress.CheckpointID,
				NextCheckpointID: candidate.AssetID, SourceFingerprint: candidate.SourceFingerprint, UpdatedAt: now,
				Batch: ObservationBatch{GenerationID: job.GenerationID, LibraryID: job.LibraryID, AssetID: candidate.AssetID, UpdatedAt: now}}
			if analyzeErr != nil {
				if errors.Is(analyzeErr, ErrSourceChanged) {
					commit.StaleCount = 1
				} else {
					commit.FailedCount = 1
				}
			} else {
				commit.Batch.Items = make([]ObservationItem, len(observations))
				for index, observation := range observations {
					vector, encodeErr := EncodeEmbedding(observation.Embedding, len(observation.Embedding))
					if encodeErr != nil {
						return p.fail(ctx, job, encodeErr)
					}
					commit.Batch.Items[index] = ObservationItem{ID: ObservationID(job.GenerationID, job.LibraryID, candidate.AssetID, observation),
						Box: observation.Box, Detection: observation.Detection, Quality: observation.Quality,
						SourceFingerprint: observation.SourceFingerprint, Vector: vector, CreatedAt: now}
				}
			}
			progress, err = p.queue.CommitFaceJobProgress(ctx, commit)
			if err != nil {
				return p.fail(ctx, job, err)
			}
		}
	}
}

func (p *JobProcessor) refreshControl(ctx context.Context, job AnalysisJob) (bool, error) {
	cancelled, err := p.queue.RefreshFaceJobLease(ctx, job, p.now().UTC(), 2*time.Minute)
	if err != nil {
		return true, err
	}
	if cancelled {
		_, err = p.queue.FinishFaceJob(ctx, job, false, "cancelled", p.now().UTC())
		return true, err
	}
	return p.stopIfLibraryUnavailable(ctx, job)
}

func (p *JobProcessor) stopIfLibraryUnavailable(ctx context.Context, job AnalysisJob) (bool, error) {
	state, err := p.catalog.FaceJobLibraryState(ctx, job.LibraryID, job.GenerationID)
	if err != nil {
		return true, p.fail(ctx, job, err)
	}
	switch state {
	case JobLibraryReady:
		return false, nil
	case JobLibraryOffline:
		_, err = p.queue.FinishFaceJob(ctx, job, false, "library_offline", p.now().UTC())
		return true, err
	case JobLibraryNotReady:
		_, err = p.queue.FinishFaceJob(ctx, job, false, "library_not_ready", p.now().UTC())
		return true, err
	case JobLibraryDisabled:
		_, err = p.queue.FinishFaceJob(ctx, job, false, "face_disabled", p.now().UTC())
		return true, err
	case JobLibraryModelUnavailable:
		_, err = p.queue.FinishFaceJob(ctx, job, false, "model_unavailable", p.now().UTC())
		return true, err
	default:
		return true, p.fail(ctx, job, ErrInvalidFaceJob)
	}
}

func (p *JobProcessor) fail(ctx context.Context, job AnalysisJob, cause error) error {
	return p.failWithCode(ctx, job, "internal_error", cause)
}

func (p *JobProcessor) failWithCode(ctx context.Context, job AnalysisJob, code string, cause error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	_, finishErr := p.queue.FinishFaceJob(ctx, job, false, code, p.now().UTC())
	return errors.Join(cause, finishErr)
}
