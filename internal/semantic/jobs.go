package semantic

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
	"github.com/HappyQuQu/foliopath/internal/jobs"
)

const MaximumSemanticJobAttempts = 3

var (
	ErrInvalidSemanticJob   = errors.New("invalid semantic job")
	ErrSemanticJobNotFound  = errors.New("semantic job not found")
	ErrSemanticJobConflict  = errors.New("semantic job conflict")
	ErrSemanticJobCancelled = errors.New("semantic job cancellation requested")
	semanticKeyPattern      = regexp.MustCompile(`^[A-Za-z0-9._~-]{8,128}$`)
)

type JobMode string

const (
	JobMissing JobMode = "missing"
	JobAll     JobMode = "all"
)

type JobState string

const (
	JobQueued     JobState = "queued"
	JobRunning    JobState = "running"
	JobCancelling JobState = "cancelling"
	JobSucceeded  JobState = "succeeded"
	JobFailed     JobState = "failed"
	JobCancelled  JobState = "cancelled"
)

type BackfillJob struct {
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

type BackfillAdmission struct {
	IdempotencyKeyHash string
	RequestHash        string
	EligibleCount      int64
	Job                BackfillJob
}

type BackfillCandidate struct {
	AssetID           int64
	SourceFingerprint string
}

type BackfillCandidatePage struct {
	Items      []BackfillCandidate
	Checkpoint int64
	HasMore    bool
}

type BackfillCandidateCounts struct {
	Eligible int64
	Pending  int64
}

type BackfillCatalog interface {
	CountSemanticBackfillCandidates(context.Context, int64, string, JobMode) (BackfillCandidateCounts, error)
	ListSemanticBackfillCandidates(context.Context, int64, string, JobMode, int64, int) (BackfillCandidatePage, error)
}

type BackfillResult struct {
	Job       BackfillJob
	Created   bool
	Replayed  bool
	Coalesced bool
}

type BackfillQueue interface {
	FindSemanticBackfill(context.Context, string) (BackfillAdmission, bool, error)
	CreateSemanticBackfill(context.Context, BackfillAdmission) (BackfillAdmission, bool, bool, error)
	ClaimSemanticBackfill(context.Context, time.Time, time.Duration) (BackfillJob, bool, error)
	RefreshSemanticBackfillLease(context.Context, BackfillJob, time.Time, time.Duration) (bool, error)
	CancelSemanticBackfill(context.Context, string, int64, time.Time) (BackfillJob, error)
	CancelSemanticBackfillOperation(context.Context, string, int64, time.Time) (BackfillJob, error)
	FinishSemanticBackfill(context.Context, BackfillJob, JobState, string, time.Time) (BackfillJob, error)
	RecoverExpiredSemanticBackfills(context.Context, time.Time) (jobs.RecoverySummary, error)
}

func (service *BackfillService) CancelOperation(ctx context.Context, operationID string, revision int64) (BackfillJob, error) {
	if operationID == "" || revision < 1 {
		return BackfillJob{}, ErrInvalidSemanticJob
	}
	return service.queue.CancelSemanticBackfillOperation(ctx, operationID, revision, service.now().UTC())
}

type BackfillService struct {
	queue   BackfillQueue
	catalog BackfillCatalog
	now     func() time.Time
	newID   func(string) (string, error)
	wake    interface{ Wake() }
}

func NewBackfillService(queue BackfillQueue, catalog BackfillCatalog, wake interface{ Wake() }, now func() time.Time, newID func(string) (string, error)) (*BackfillService, error) {
	if queue == nil || catalog == nil || wake == nil {
		return nil, errors.New("semantic backfill dependencies are required")
	}
	if now == nil {
		now = time.Now
	}
	if newID == nil {
		newID = randomSemanticID
	}
	return &BackfillService{queue: queue, catalog: catalog, wake: wake, now: now, newID: newID}, nil
}

func (service *BackfillService) Request(ctx context.Context, libraryID int64, generationID string, mode JobMode, idempotencyKey string) (BackfillResult, error) {
	if libraryID < 1 || len(generationID) < 8 || len(generationID) > 128 || !validJobMode(mode) || !semanticKeyPattern.MatchString(idempotencyKey) {
		return BackfillResult{}, ErrInvalidSemanticJob
	}
	keyHash := digestSemanticValue("foliopath:idempotency-key:v1\x00" + idempotencyKey)
	requestHash := BackfillRequestHash(libraryID, generationID, mode)
	if existing, found, err := service.queue.FindSemanticBackfill(ctx, keyHash); err != nil {
		return BackfillResult{}, err
	} else if found {
		if existing.RequestHash != requestHash {
			return BackfillResult{}, ErrSemanticJobConflict
		}
		return BackfillResult{Job: existing.Job, Replayed: true}, nil
	}
	counts, err := service.catalog.CountSemanticBackfillCandidates(ctx, libraryID, generationID, mode)
	if err != nil {
		return BackfillResult{}, err
	}
	if counts.Eligible < 0 || counts.Pending < 0 || counts.Pending > counts.Eligible {
		return BackfillResult{}, ErrInvalidSemanticJob
	}
	jobID, err := service.newID("semjob")
	if err != nil {
		return BackfillResult{}, err
	}
	operationID, err := service.newID("aio")
	if err != nil {
		return BackfillResult{}, err
	}
	now := service.now().UTC()
	admission, created, coalesced, err := service.queue.CreateSemanticBackfill(ctx, BackfillAdmission{
		IdempotencyKeyHash: keyHash,
		RequestHash:        requestHash,
		EligibleCount:      counts.Eligible,
		Job: BackfillJob{
			ID: jobID, LibraryID: libraryID, GenerationID: generationID, OperationID: operationID,
			Mode: mode, State: JobQueued, RequestedRevision: 1, OperationRevision: 1,
			TotalItems: counts.Pending, CreatedAt: now, UpdatedAt: now,
		},
	})
	if err != nil {
		return BackfillResult{}, err
	}
	if admission.RequestHash != requestHash {
		return BackfillResult{}, ErrSemanticJobConflict
	}
	if created {
		service.wake.Wake()
	}
	return BackfillResult{Job: admission.Job, Created: created, Replayed: !created && !coalesced, Coalesced: coalesced}, nil
}

func BackfillRequestHash(libraryID int64, generationID string, mode JobMode) string {
	return digestSemanticValue(fmt.Sprintf("foliopath:semantic-backfill:v1\x00%d\x00%s\x00%s", libraryID, generationID, mode))
}

func ValidateBackfillAdmission(value BackfillAdmission) error {
	job := value.Job
	if len(value.IdempotencyKeyHash) != 64 || len(value.RequestHash) != 64 || value.EligibleCount < 0 ||
		job.ID == "" || job.OperationID == "" || job.LibraryID < 1 || len(job.GenerationID) < 8 ||
		len(job.GenerationID) > 128 || !validJobMode(job.Mode) || job.State != JobQueued ||
		job.CheckpointID != 0 || job.RequestedRevision != 1 || job.ClaimedRevision != 0 ||
		job.AttemptCount != 0 || job.LeaseExpiresAt != nil || job.ErrorCode != "" ||
		job.OperationRevision != 1 || job.CompletedItems != 0 || job.TotalItems < 0 || job.TotalItems > value.EligibleCount ||
		job.CreatedAt.IsZero() || !job.UpdatedAt.Equal(job.CreatedAt) {
		return ErrInvalidSemanticJob
	}
	if _, err := hex.DecodeString(value.IdempotencyKeyHash); err != nil {
		return ErrInvalidSemanticJob
	}
	if _, err := hex.DecodeString(value.RequestHash); err != nil {
		return ErrInvalidSemanticJob
	}
	return nil
}

func ValidateBackfillCandidateQuery(libraryID int64, generationID string, mode JobMode, checkpoint int64, limit int) error {
	if libraryID < 1 || len(generationID) < 8 || len(generationID) > 128 || !validJobMode(mode) ||
		checkpoint < 0 || limit < 1 || limit > 1000 {
		return ErrInvalidSemanticJob
	}
	return nil
}

func (job BackfillJob) OperationKind() aimodel.OperationKind {
	if job.Mode == JobAll {
		return aimodel.OperationSemanticRebuild
	}
	return aimodel.OperationSemanticMissing
}

func validJobMode(mode JobMode) bool { return mode == JobMissing || mode == JobAll }

func digestSemanticValue(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}

func randomSemanticID(prefix string) (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return prefix + "_" + hex.EncodeToString(value), nil
}
