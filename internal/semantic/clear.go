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

const DefaultSemanticClearBatchSize = 500

var (
	ErrInvalidSemanticClear  = errors.New("invalid semantic clear")
	ErrSemanticClearConflict = errors.New("semantic clear conflict")
	ErrSemanticClearNotFound = errors.New("semantic clear not found")
)

type ClearJob struct {
	ID                string
	LibraryID         int64
	OperationID       string
	State             JobState
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

type ClearAdmission struct {
	IdempotencyKeyHash       string
	RequestHash              string
	ExpectedSettingsRevision int64
	Job                      ClearJob
}

type ClearResult struct {
	Job       ClearJob
	Created   bool
	Replayed  bool
	Coalesced bool
}

type ClearQueue interface {
	FindSemanticClear(context.Context, string) (ClearAdmission, bool, error)
	CreateSemanticClear(context.Context, ClearAdmission) (ClearAdmission, bool, bool, error)
	ClaimSemanticClear(context.Context, time.Time, time.Duration) (ClearJob, bool, error)
	RefreshSemanticClearLease(context.Context, ClearJob, time.Time, time.Duration) (bool, error)
	DeleteSemanticClearBatch(context.Context, ClearJob, int, time.Time) (int64, bool, error)
	FinishSemanticClear(context.Context, ClearJob, JobState, string, time.Time) (ClearJob, error)
	CancelSemanticClearOperation(context.Context, string, int64, time.Time) (ClearJob, error)
	RecoverExpiredSemanticClears(context.Context, time.Time) (jobs.RecoverySummary, error)
}

type ClearService struct {
	queue ClearQueue
	now   func() time.Time
	newID func(string) (string, error)
	wake  interface{ Wake() }
}

type ClearProcessor struct {
	queue     ClearQueue
	now       func() time.Time
	batchSize int
}

func NewClearProcessor(queue ClearQueue, now func() time.Time, batchSize int) (*ClearProcessor, error) {
	if queue == nil {
		return nil, errors.New("semantic clear queue is required")
	}
	if now == nil {
		now = time.Now
	}
	if batchSize == 0 {
		batchSize = DefaultSemanticClearBatchSize
	}
	if batchSize < 1 || batchSize > 1000 {
		return nil, ErrInvalidSemanticClear
	}
	return &ClearProcessor{queue: queue, now: now, batchSize: batchSize}, nil
}

func (processor *ClearProcessor) Process(ctx context.Context, job ClearJob) error {
	if job.ID == "" || job.LibraryID < 1 || job.OperationID == "" || job.State != JobRunning || job.ClaimedRevision < 1 {
		return ErrInvalidSemanticClear
	}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		_, done, err := processor.queue.DeleteSemanticClearBatch(ctx, job, processor.batchSize, processor.now().UTC())
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			_, finishErr := processor.queue.FinishSemanticClear(ctx, job, JobFailed, "internal_error", processor.now().UTC())
			return errors.Join(err, finishErr)
		}
		if done {
			_, err = processor.queue.FinishSemanticClear(ctx, job, JobSucceeded, "", processor.now().UTC())
			return err
		}
	}
}

func NewClearService(queue ClearQueue, wake interface{ Wake() }, now func() time.Time, newID func(string) (string, error)) (*ClearService, error) {
	if queue == nil || wake == nil {
		return nil, errors.New("semantic clear dependencies are required")
	}
	if now == nil {
		now = time.Now
	}
	if newID == nil {
		newID = randomSemanticID
	}
	return &ClearService{queue: queue, wake: wake, now: now, newID: newID}, nil
}

func (service *ClearService) Request(ctx context.Context, libraryID, expectedSettingsRevision int64, idempotencyKey string) (ClearResult, error) {
	if libraryID < 1 || expectedSettingsRevision < 1 || !semanticKeyPattern.MatchString(idempotencyKey) {
		return ClearResult{}, ErrInvalidSemanticClear
	}
	keyHash := digestSemanticValue("foliopath:semantic-clear-idempotency:v1\x00" + idempotencyKey)
	requestHash := digestSemanticValue(fmt.Sprintf("foliopath:semantic-clear:v1\x00%d\x00%d", libraryID, expectedSettingsRevision))
	if existing, found, err := service.queue.FindSemanticClear(ctx, keyHash); err != nil {
		return ClearResult{}, err
	} else if found {
		if existing.RequestHash != requestHash {
			return ClearResult{}, ErrSemanticClearConflict
		}
		return ClearResult{Job: existing.Job, Replayed: true}, nil
	}
	jobID, err := service.newID("semclear")
	if err != nil {
		return ClearResult{}, err
	}
	operationID, err := service.newID("aio")
	if err != nil {
		return ClearResult{}, err
	}
	now := service.now().UTC()
	stored, created, coalesced, err := service.queue.CreateSemanticClear(ctx, ClearAdmission{
		IdempotencyKeyHash: keyHash, RequestHash: requestHash, ExpectedSettingsRevision: expectedSettingsRevision,
		Job: ClearJob{ID: jobID, LibraryID: libraryID, OperationID: operationID, State: JobQueued,
			RequestedRevision: 1, OperationRevision: 1, CreatedAt: now, UpdatedAt: now},
	})
	if err != nil {
		return ClearResult{}, err
	}
	if stored.RequestHash != requestHash {
		return ClearResult{}, ErrSemanticClearConflict
	}
	if created {
		service.wake.Wake()
	}
	return ClearResult{Job: stored.Job, Created: created, Replayed: !created && !coalesced, Coalesced: coalesced}, nil
}

func (service *ClearService) CancelOperation(ctx context.Context, operationID string, revision int64) (ClearJob, error) {
	if operationID == "" || revision < 1 {
		return ClearJob{}, ErrInvalidSemanticClear
	}
	return service.queue.CancelSemanticClearOperation(ctx, operationID, revision, service.now().UTC())
}

func (job ClearJob) OperationKind() aimodel.OperationKind { return aimodel.OperationSemanticClear }

func ValidateClearAdmission(value ClearAdmission) error {
	job := value.Job
	if len(value.IdempotencyKeyHash) != 64 || len(value.RequestHash) != 64 || value.ExpectedSettingsRevision < 1 ||
		job.ID == "" || job.LibraryID < 1 || job.OperationID == "" || job.State != JobQueued ||
		job.RequestedRevision != 1 || job.ClaimedRevision != 0 || job.AttemptCount != 0 || job.LeaseExpiresAt != nil ||
		job.ErrorCode != "" || job.OperationRevision != 1 || job.CompletedItems != 0 || job.TotalItems != 0 ||
		job.CreatedAt.IsZero() || !job.UpdatedAt.Equal(job.CreatedAt) {
		return ErrInvalidSemanticClear
	}
	if _, err := hex.DecodeString(value.IdempotencyKeyHash); err != nil {
		return ErrInvalidSemanticClear
	}
	if _, err := hex.DecodeString(value.RequestHash); err != nil {
		return ErrInvalidSemanticClear
	}
	return nil
}
