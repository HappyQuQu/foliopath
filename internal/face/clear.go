package face

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/HappyQuQu/foliopath/internal/aimodel"
	"github.com/HappyQuQu/foliopath/internal/jobs"
	"time"
)

const (
	DefaultFaceClearBatchSize = 500
	MaximumFaceClearAttempts  = 3
)

var (
	ErrInvalidFaceClear       = errors.New("invalid face clear")
	ErrFaceClearConflict      = errors.New("face clear conflict")
	ErrFaceClearCountConflict = errors.New("face clear impact count conflict")
)

type ClearKind string

const (
	ClearDerived ClearKind = "derived"
	ClearManual  ClearKind = "manual"
)

type ManualClearCounts struct{ People, Assignments, Constraints int64 }
type ClearJob struct {
	ID                                               string
	LibraryID                                        int64
	OperationID                                      string
	Kind                                             ClearKind
	ExpectedSettingsRevision                         int64
	ExpectedCounts                                   *ManualClearCounts
	State                                            string
	DeletedCount, RequestedRevision, ClaimedRevision int64
	AttemptCount                                     int
	CreatedAt, UpdatedAt                             time.Time
}
type ClearAdmission struct {
	IdempotencyKeyHash, RequestHash string
	Job                             ClearJob
}
type ClearResult struct {
	Job               ClearJob
	Created, Replayed bool
}
type ClearQueue interface {
	FindFaceClear(context.Context, string) (ClearAdmission, bool, error)
	CreateFaceClear(context.Context, ClearAdmission) (ClearAdmission, bool, error)
	ClaimFaceClear(context.Context, time.Time, time.Duration) (ClearJob, bool, error)
	RefreshFaceClearLease(context.Context, ClearJob, time.Time, time.Duration) (bool, error)
	DeleteFaceClearBatch(context.Context, ClearJob, int, time.Time) (int64, bool, error)
	FinishFaceClear(context.Context, ClearJob, bool, string, time.Time) (ClearJob, error)
	CancelFaceClearOperation(context.Context, string, int64, time.Time) (ClearJob, error)
	RecoverExpiredFaceClears(context.Context, time.Time) (jobs.RecoverySummary, error)
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

func NewClearService(queue ClearQueue, wake interface{ Wake() }, now func() time.Time, newID func(string) (string, error)) (*ClearService, error) {
	if queue == nil || wake == nil {
		return nil, errors.New("face clear dependencies are required")
	}
	if now == nil {
		now = time.Now
	}
	if newID == nil {
		newID = randomFaceID
	}
	return &ClearService{queue: queue, wake: wake, now: now, newID: newID}, nil
}
func NewClearProcessor(queue ClearQueue, now func() time.Time, batchSize int) (*ClearProcessor, error) {
	if queue == nil {
		return nil, errors.New("face clear queue is required")
	}
	if now == nil {
		now = time.Now
	}
	if batchSize == 0 {
		batchSize = DefaultFaceClearBatchSize
	}
	if batchSize < 1 || batchSize > 1000 {
		return nil, ErrInvalidFaceClear
	}
	return &ClearProcessor{queue: queue, now: now, batchSize: batchSize}, nil
}
func (s *ClearService) RequestDerived(ctx context.Context, libraryID, expectedRevision int64, key string) (ClearResult, error) {
	return s.request(ctx, libraryID, expectedRevision, key, ClearDerived, nil)
}
func (s *ClearService) RequestManual(ctx context.Context, libraryID, expectedRevision int64, key string, counts ManualClearCounts) (ClearResult, error) {
	return s.request(ctx, libraryID, expectedRevision, key, ClearManual, &counts)
}
func (s *ClearService) Cancel(ctx context.Context, operationID string, expectedRevision int64) (ClearJob, error) {
	if operationID == "" || expectedRevision < 1 {
		return ClearJob{}, ErrInvalidFaceClear
	}
	return s.queue.CancelFaceClearOperation(ctx, operationID, expectedRevision, s.now().UTC())
}
func (s *ClearService) request(ctx context.Context, libraryID, expectedRevision int64, key string, kind ClearKind, counts *ManualClearCounts) (ClearResult, error) {
	if libraryID < 1 || expectedRevision < 1 || !reviewIdempotencyKeyPattern.MatchString(key) || (kind == ClearManual && (counts == nil || counts.People < 0 || counts.Assignments < 0 || counts.Constraints < 0)) || (kind == ClearDerived && counts != nil) {
		return ClearResult{}, ErrInvalidFaceClear
	}
	keyHash := faceDigest("foliopath:face-clear-key:v1\x00" + key)
	requestHash := faceDigest(fmt.Sprintf("foliopath:face-clear-request:v1\x00%d\x00%d\x00%s\x00%v", libraryID, expectedRevision, kind, counts))
	if existing, found, err := s.queue.FindFaceClear(ctx, keyHash); err != nil {
		return ClearResult{}, err
	} else if found {
		if existing.RequestHash != requestHash {
			return ClearResult{}, ErrFaceClearConflict
		}
		return ClearResult{Job: existing.Job, Replayed: true}, nil
	}
	jobID, err := s.newID("faceclear")
	if err != nil {
		return ClearResult{}, err
	}
	operationID, err := s.newID("aio")
	if err != nil {
		return ClearResult{}, err
	}
	now := s.now().UTC()
	stored, created, err := s.queue.CreateFaceClear(ctx, ClearAdmission{IdempotencyKeyHash: keyHash, RequestHash: requestHash, Job: ClearJob{ID: jobID, LibraryID: libraryID, OperationID: operationID, Kind: kind, ExpectedSettingsRevision: expectedRevision, ExpectedCounts: counts, State: "queued", RequestedRevision: 1, CreatedAt: now, UpdatedAt: now}})
	if err != nil {
		return ClearResult{}, err
	}
	if stored.RequestHash != requestHash {
		return ClearResult{}, ErrFaceClearConflict
	}
	if created {
		s.wake.Wake()
	}
	return ClearResult{Job: stored.Job, Created: created, Replayed: !created}, nil
}
func (p *ClearProcessor) Process(ctx context.Context, job ClearJob) error {
	if job.ID == "" || job.State != "running" || job.ClaimedRevision < 1 {
		return ErrInvalidFaceClear
	}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		now := p.now().UTC()
		cancelled, err := p.queue.RefreshFaceClearLease(ctx, job, now, 2*time.Minute)
		if err != nil {
			return err
		}
		if cancelled {
			_, err = p.queue.FinishFaceClear(ctx, job, false, "cancelled", now)
			return err
		}
		_, done, err := p.queue.DeleteFaceClearBatch(ctx, job, p.batchSize, now)
		if err != nil {
			_, finishErr := p.queue.FinishFaceClear(ctx, job, false, "internal_error", p.now().UTC())
			return errors.Join(err, finishErr)
		}
		if done {
			_, err = p.queue.FinishFaceClear(ctx, job, true, "", p.now().UTC())
			return err
		}
	}
}
func (j ClearJob) OperationKind() aimodel.OperationKind {
	switch j.Kind {
	case ClearDerived:
		return aimodel.OperationFaceDerivedClear
	case ClearManual:
		return aimodel.OperationFaceManualClear
	default:
		return ""
	}
}
func faceDigest(value string) string    { sum := sha256Sum(value); return hex.EncodeToString(sum) }
func sha256Sum(value string) []byte     { sum := sha256Bytes([]byte(value)); return sum[:] }
func sha256Bytes(value []byte) [32]byte { return sha256.Sum256(value) }
func randomFaceID(prefix string) (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return prefix + "_" + hex.EncodeToString(value), nil
}
