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

var (
	ErrInvalidTagJob  = errors.New("invalid semantic tag job")
	ErrTagJobNotFound = errors.New("semantic tag job not found")
	ErrTagJobConflict = errors.New("semantic tag job conflict")
)

type TagJob struct {
	ID, GenerationID, VocabularySnapshotID, OperationID string
	LibraryID, CheckpointID                             int64
	Mode                                                JobMode
	State                                               JobState
	RequestedRevision, ClaimedRevision                  int64
	AttemptCount                                        int
	LeaseExpiresAt                                      *time.Time
	ErrorCode                                           string
	OperationRevision, CompletedItems, TotalItems       int64
	CreatedAt, UpdatedAt                                time.Time
}

func (job TagJob) OperationKind() aimodel.OperationKind {
	if job.Mode == JobAll {
		return aimodel.OperationTagSuggestionRebuild
	}
	return aimodel.OperationTagSuggestionMissing
}

type TagJobAdmission struct {
	IdempotencyKeyHash, RequestHash string
	EligibleCount                   int64
	Job                             TagJob
}

type TagJobCandidate struct{ AssetID int64 }
type TagJobCandidatePage struct {
	Items      []TagJobCandidate
	Checkpoint int64
	HasMore    bool
}
type TagJobCandidateCounts struct{ Eligible, Pending int64 }

type TagJobCatalog interface {
	CountTagJobCandidates(context.Context, int64, string, string, JobMode) (TagJobCandidateCounts, error)
	ListTagJobCandidates(context.Context, int64, string, string, JobMode, int64, int) (TagJobCandidatePage, error)
}

type TagJobProgress struct {
	GenerationID, VocabularySnapshotID       string
	LibraryID                                int64
	Eligible, Ready, Degraded, Failed, Stale int64
	CheckpointID, Revision                   int64
	UpdatedAt                                time.Time
}

type TagSuggestionPlan struct {
	GenerationID, VocabularySnapshotID, SourceFingerprint string
	LibraryID, AssetID                                    int64
	Suggestions                                           []TagSuggestion
}

type TagJobProgressCommit struct {
	JobID                                     string
	ClaimedRevision, ExpectedProgressRevision int64
	ExpectedCheckpointID, NextCheckpointID    int64
	Plan                                      *TagSuggestionPlan
	DegradedCount, FailedCount, StaleCount    int64
	UpdatedAt                                 time.Time
}

type TagJobQueue interface {
	FindTagJob(context.Context, string) (TagJobAdmission, bool, error)
	CreateTagJob(context.Context, TagJobAdmission) (TagJobAdmission, bool, bool, error)
	ClaimTagJob(context.Context, time.Time, time.Duration) (TagJob, bool, error)
	RefreshTagJobLease(context.Context, TagJob, time.Time, time.Duration) (bool, error)
	GetTagJobProgress(context.Context, string, int64, string) (TagJobProgress, bool, error)
	CommitTagJobProgress(context.Context, TagJobProgressCommit) (TagJobProgress, error)
	CancelTagJobOperation(context.Context, string, int64, time.Time) (TagJob, error)
	FinishTagJob(context.Context, TagJob, JobState, string, time.Time) (TagJob, error)
	RecoverExpiredTagJobs(context.Context, time.Time) (jobs.RecoverySummary, error)
}

type TagJobResult struct {
	Job                          TagJob
	Created, Replayed, Coalesced bool
}

type TagJobService struct {
	queue   TagJobQueue
	catalog TagJobCatalog
	wake    interface{ Wake() }
	now     func() time.Time
	newID   func(string) (string, error)
}

func NewTagJobService(queue TagJobQueue, catalog TagJobCatalog, wake interface{ Wake() }, now func() time.Time, newID func(string) (string, error)) (*TagJobService, error) {
	if queue == nil || catalog == nil || wake == nil {
		return nil, errors.New("tag job dependencies are required")
	}
	if now == nil {
		now = time.Now
	}
	if newID == nil {
		newID = randomSemanticID
	}
	return &TagJobService{queue: queue, catalog: catalog, wake: wake, now: now, newID: newID}, nil
}

func (s *TagJobService) Request(ctx context.Context, libraryID int64, generationID, snapshotID string, mode JobMode, key string) (TagJobResult, error) {
	if libraryID < 1 || len(generationID) < 8 || len(snapshotID) < 8 || !validJobMode(mode) || !semanticKeyPattern.MatchString(key) {
		return TagJobResult{}, ErrInvalidTagJob
	}
	keyHash := digestSemanticValue("foliopath:tag-job-key:v1\x00" + key)
	requestHash := digestSemanticValue(fmt.Sprintf("foliopath:tag-job:v1\x00%d\x00%s\x00%s\x00%s", libraryID, generationID, snapshotID, mode))
	if existing, found, err := s.queue.FindTagJob(ctx, keyHash); err != nil {
		return TagJobResult{}, err
	} else if found {
		if existing.RequestHash != requestHash {
			return TagJobResult{}, ErrTagJobConflict
		}
		return TagJobResult{Job: existing.Job, Replayed: true}, nil
	}
	counts, err := s.catalog.CountTagJobCandidates(ctx, libraryID, generationID, snapshotID, mode)
	if err != nil {
		return TagJobResult{}, err
	}
	if counts.Eligible < 0 || counts.Pending < 0 || counts.Pending > counts.Eligible {
		return TagJobResult{}, ErrInvalidTagJob
	}
	jobID, err := s.newID("tagjob")
	if err != nil {
		return TagJobResult{}, err
	}
	opID, err := s.newID("aio")
	if err != nil {
		return TagJobResult{}, err
	}
	now := s.now().UTC()
	stored, created, coalesced, err := s.queue.CreateTagJob(ctx, TagJobAdmission{IdempotencyKeyHash: keyHash, RequestHash: requestHash, EligibleCount: counts.Eligible,
		Job: TagJob{ID: jobID, LibraryID: libraryID, GenerationID: generationID, VocabularySnapshotID: snapshotID, OperationID: opID, Mode: mode, State: JobQueued,
			RequestedRevision: 1, OperationRevision: 1, TotalItems: counts.Pending, CreatedAt: now, UpdatedAt: now}})
	if err != nil {
		return TagJobResult{}, err
	}
	if stored.RequestHash != requestHash {
		return TagJobResult{}, ErrTagJobConflict
	}
	if created {
		s.wake.Wake()
	}
	return TagJobResult{Job: stored.Job, Created: created, Replayed: !created && !coalesced, Coalesced: coalesced}, nil
}

func (s *TagJobService) CancelOperation(ctx context.Context, id string, revision int64) (TagJob, error) {
	if id == "" || revision < 1 {
		return TagJob{}, ErrInvalidTagJob
	}
	return s.queue.CancelTagJobOperation(ctx, id, revision, s.now().UTC())
}

func ValidateTagJobAdmission(v TagJobAdmission) error {
	j := v.Job
	if len(v.IdempotencyKeyHash) != 64 || len(v.RequestHash) != 64 || v.EligibleCount < 0 || j.ID == "" || j.LibraryID < 1 || len(j.GenerationID) < 8 || len(j.VocabularySnapshotID) < 8 || j.OperationID == "" ||
		!validJobMode(j.Mode) || j.State != JobQueued || j.CheckpointID != 0 || j.RequestedRevision != 1 || j.ClaimedRevision != 0 || j.AttemptCount != 0 || j.LeaseExpiresAt != nil || j.ErrorCode != "" ||
		j.OperationRevision != 1 || j.CompletedItems != 0 || j.TotalItems < 0 || j.TotalItems > v.EligibleCount || j.CreatedAt.IsZero() || !j.UpdatedAt.Equal(j.CreatedAt) {
		return ErrInvalidTagJob
	}
	if _, err := hex.DecodeString(v.IdempotencyKeyHash); err != nil {
		return ErrInvalidTagJob
	}
	if _, err := hex.DecodeString(v.RequestHash); err != nil {
		return ErrInvalidTagJob
	}
	return nil
}

func ValidateTagJobCandidateQuery(libraryID int64, generationID, snapshotID string, mode JobMode, checkpoint int64, limit int) error {
	if libraryID < 1 || len(generationID) < 8 || len(snapshotID) < 8 || !validJobMode(mode) || checkpoint < 0 || limit < 1 || limit > 1000 {
		return ErrInvalidTagJob
	}
	return nil
}
