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

const MaximumVideoJobAttempts = 3

var (
	ErrInvalidVideoJob   = errors.New("invalid video semantic job")
	ErrVideoJobNotFound  = errors.New("video semantic job not found")
	ErrVideoJobConflict  = errors.New("video semantic job conflict")
	ErrVideoJobCancelled = errors.New("video semantic job cancellation requested")
)

type VideoJob struct {
	ID                string
	LibraryID         int64
	GenerationID      string
	OperationID       string
	Mode              JobMode
	State             JobState
	CheckpointID      int64
	RequestedRevision int64
	ClaimedRevision   int64
	AttemptCount      int
	LeaseExpiresAt    *time.Time
	ErrorCode         string
	OperationRevision int64
	CompletedItems    int64
	TotalItems        int64
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (job VideoJob) OperationKind() aimodel.OperationKind {
	if job.Mode == JobAll {
		return aimodel.OperationVideoSemanticRebuild
	}
	return aimodel.OperationVideoSemanticMissing
}

type VideoJobAdmission struct {
	IdempotencyKeyHash string
	RequestHash        string
	EligibleCount      int64
	Job                VideoJob
}

type VideoJobCandidate struct {
	AssetID int64
}

type VideoJobCandidatePage struct {
	Items      []VideoJobCandidate
	Checkpoint int64
	HasMore    bool
}

type VideoJobCandidateCounts struct {
	Eligible int64
	Pending  int64
}

type VideoJobCatalog interface {
	CountVideoJobCandidates(context.Context, int64, string, JobMode) (VideoJobCandidateCounts, error)
	ListVideoJobCandidates(context.Context, int64, string, JobMode, int64, int) (VideoJobCandidatePage, error)
}

type VideoJobQueue interface {
	FindVideoJob(context.Context, string) (VideoJobAdmission, bool, error)
	CreateVideoJob(context.Context, VideoJobAdmission) (VideoJobAdmission, bool, bool, error)
	ClaimVideoJob(context.Context, time.Time, time.Duration) (VideoJob, bool, error)
	RefreshVideoJobLease(context.Context, VideoJob, time.Time, time.Duration) (bool, error)
	GetVideoJobProgress(context.Context, string, int64) (VideoJobProgress, bool, error)
	CommitVideoJobProgress(context.Context, VideoJobProgressCommit) (VideoJobProgress, error)
	CancelVideoJobOperation(context.Context, string, int64, time.Time) (VideoJob, error)
	FinishVideoJob(context.Context, VideoJob, JobState, string, time.Time) (VideoJob, error)
	RecoverExpiredVideoJobs(context.Context, time.Time) (jobs.RecoverySummary, error)
}

type VideoJobProgress struct {
	GenerationID string
	LibraryID    int64
	Eligible     int64
	Ready        int64
	Degraded     int64
	Failed       int64
	Stale        int64
	CheckpointID int64
	Revision     int64
	UpdatedAt    time.Time
}

type VideoJobProgressCommit struct {
	JobID                    string
	ClaimedRevision          int64
	ExpectedProgressRevision int64
	ExpectedCheckpointID     int64
	NextCheckpointID         int64
	Plan                     *VideoEmbeddingPlan
	DegradedCount            int64
	FailedCount              int64
	StaleCount               int64
	UpdatedAt                time.Time
}

func ValidateVideoJobProgressCommit(value VideoJobProgressCommit, dimension int) error {
	processed := value.DegradedCount + value.FailedCount + value.StaleCount
	if value.Plan != nil {
		processed++
	}
	if len(value.JobID) < 8 || len(value.JobID) > 128 || value.ClaimedRevision < 1 ||
		value.ExpectedProgressRevision < 1 || value.ExpectedCheckpointID < 0 ||
		value.NextCheckpointID <= value.ExpectedCheckpointID || value.DegradedCount < 0 ||
		value.FailedCount < 0 || value.StaleCount < 0 || processed != 1 || value.UpdatedAt.IsZero() {
		return ErrInvalidVideoJob
	}
	if value.Plan != nil {
		if value.Plan.AssetID != value.NextCheckpointID || ValidateVideoEmbeddingPlan(*value.Plan, dimension) != nil {
			return ErrInvalidVideoJob
		}
	}
	return nil
}

type VideoJobResult struct {
	Job       VideoJob
	Created   bool
	Replayed  bool
	Coalesced bool
}

type VideoJobService struct {
	queue   VideoJobQueue
	catalog VideoJobCatalog
	wake    interface{ Wake() }
	now     func() time.Time
	newID   func(string) (string, error)
}

func NewVideoJobService(queue VideoJobQueue, catalog VideoJobCatalog, wake interface{ Wake() }, now func() time.Time, newID func(string) (string, error)) (*VideoJobService, error) {
	if queue == nil || catalog == nil || wake == nil {
		return nil, errors.New("video semantic job dependencies are required")
	}
	if now == nil {
		now = time.Now
	}
	if newID == nil {
		newID = randomSemanticID
	}
	return &VideoJobService{queue: queue, catalog: catalog, wake: wake, now: now, newID: newID}, nil
}

func (service *VideoJobService) Request(ctx context.Context, libraryID int64, generationID string, mode JobMode, idempotencyKey string) (VideoJobResult, error) {
	if libraryID < 1 || len(generationID) < 8 || len(generationID) > 128 || !validJobMode(mode) || !semanticKeyPattern.MatchString(idempotencyKey) {
		return VideoJobResult{}, ErrInvalidVideoJob
	}
	keyHash := digestSemanticValue("foliopath:video-idempotency-key:v1\x00" + idempotencyKey)
	requestHash := VideoJobRequestHash(libraryID, generationID, mode)
	if existing, found, err := service.queue.FindVideoJob(ctx, keyHash); err != nil {
		return VideoJobResult{}, err
	} else if found {
		if existing.RequestHash != requestHash {
			return VideoJobResult{}, ErrVideoJobConflict
		}
		return VideoJobResult{Job: existing.Job, Replayed: true}, nil
	}
	counts, err := service.catalog.CountVideoJobCandidates(ctx, libraryID, generationID, mode)
	if err != nil {
		return VideoJobResult{}, err
	}
	if counts.Eligible < 0 || counts.Pending < 0 || counts.Pending > counts.Eligible {
		return VideoJobResult{}, ErrInvalidVideoJob
	}
	jobID, err := service.newID("vidjob")
	if err != nil {
		return VideoJobResult{}, err
	}
	operationID, err := service.newID("aio")
	if err != nil {
		return VideoJobResult{}, err
	}
	now := service.now().UTC()
	admission, created, coalesced, err := service.queue.CreateVideoJob(ctx, VideoJobAdmission{
		IdempotencyKeyHash: keyHash, RequestHash: requestHash, EligibleCount: counts.Eligible,
		Job: VideoJob{ID: jobID, LibraryID: libraryID, GenerationID: generationID, OperationID: operationID,
			Mode: mode, State: JobQueued, RequestedRevision: 1, OperationRevision: 1,
			TotalItems: counts.Pending, CreatedAt: now, UpdatedAt: now},
	})
	if err != nil {
		return VideoJobResult{}, err
	}
	if admission.RequestHash != requestHash {
		return VideoJobResult{}, ErrVideoJobConflict
	}
	if created {
		service.wake.Wake()
	}
	return VideoJobResult{Job: admission.Job, Created: created, Replayed: !created && !coalesced, Coalesced: coalesced}, nil
}

func (service *VideoJobService) CancelOperation(ctx context.Context, operationID string, revision int64) (VideoJob, error) {
	if operationID == "" || revision < 1 {
		return VideoJob{}, ErrInvalidVideoJob
	}
	return service.queue.CancelVideoJobOperation(ctx, operationID, revision, service.now().UTC())
}

func VideoJobRequestHash(libraryID int64, generationID string, mode JobMode) string {
	return digestSemanticValue(fmt.Sprintf("foliopath:video-semantic-job:v1\x00%d\x00%s\x00%s", libraryID, generationID, mode))
}

func ValidateVideoJobAdmission(value VideoJobAdmission) error {
	job := value.Job
	if len(value.IdempotencyKeyHash) != 64 || len(value.RequestHash) != 64 || value.EligibleCount < 0 ||
		job.ID == "" || job.OperationID == "" || job.LibraryID < 1 || len(job.GenerationID) < 8 || len(job.GenerationID) > 128 ||
		!validJobMode(job.Mode) || job.State != JobQueued || job.CheckpointID != 0 || job.RequestedRevision != 1 ||
		job.ClaimedRevision != 0 || job.AttemptCount != 0 || job.LeaseExpiresAt != nil || job.ErrorCode != "" ||
		job.OperationRevision != 1 || job.CompletedItems != 0 || job.TotalItems < 0 || job.TotalItems > value.EligibleCount ||
		job.CreatedAt.IsZero() || !job.UpdatedAt.Equal(job.CreatedAt) {
		return ErrInvalidVideoJob
	}
	if _, err := hex.DecodeString(value.IdempotencyKeyHash); err != nil {
		return ErrInvalidVideoJob
	}
	if _, err := hex.DecodeString(value.RequestHash); err != nil {
		return ErrInvalidVideoJob
	}
	return nil
}

func ValidateVideoJobCandidateQuery(libraryID int64, generationID string, mode JobMode, checkpoint int64, limit int) error {
	if libraryID < 1 || len(generationID) < 8 || len(generationID) > 128 || !validJobMode(mode) || checkpoint < 0 || limit < 1 || limit > 1000 {
		return ErrInvalidVideoJob
	}
	return nil
}
