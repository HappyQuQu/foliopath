package semantic

import (
	"context"
	"errors"
	"time"
)

var (
	ErrInvalidEmbeddingRecord        = errors.New("invalid semantic embedding record")
	ErrSemanticGenerationUnavailable = errors.New("semantic generation unavailable")
	ErrSemanticProgressConflict      = errors.New("semantic progress conflict")
)

type EmbeddingItem struct {
	AssetID           int64
	SourceFingerprint string
	Vector            []byte
	CreatedAt         time.Time
}

type EmbeddingBatch struct {
	GenerationID string
	LibraryID    int64
	Items        []EmbeddingItem
}

type StoredEmbedding struct {
	GenerationID      string
	LibraryID         int64
	AssetID           int64
	SourceFingerprint string
	Vector            []byte
	CreatedAt         time.Time
}

type EmbeddingProgress struct {
	GenerationID   string
	LibraryID      int64
	EligibleCount  int64
	CompletedCount int64
	FailedCount    int64
	StaleCount     int64
	CheckpointID   int64
	Revision       int64
	UpdatedAt      time.Time
}

type EmbeddingProgressCommit struct {
	JobID                    string
	ClaimedRevision          int64
	ExpectedProgressRevision int64
	ExpectedCheckpointID     int64
	NextCheckpointID         int64
	Batch                    EmbeddingBatch
	FailedCount              int64
	StaleCount               int64
	UpdatedAt                time.Time
}

type EmbeddingRepository interface {
	PutSemanticEmbeddingBatch(context.Context, EmbeddingBatch) error
	GetSemanticEmbedding(context.Context, string, int64, int64) (StoredEmbedding, bool, error)
	DeleteSemanticEmbeddingIfSourceChanged(context.Context, string, int64, int64, string) (bool, error)
	GetSemanticEmbeddingProgress(context.Context, string, int64) (EmbeddingProgress, bool, error)
	CommitSemanticEmbeddingProgress(context.Context, EmbeddingProgressCommit) (EmbeddingProgress, error)
}

func ValidateEmbeddingProgressCommit(commit EmbeddingProgressCommit, maximumItems int) error {
	processed := int64(len(commit.Batch.Items)) + commit.FailedCount + commit.StaleCount
	if len(commit.JobID) < 8 || len(commit.JobID) > 128 || commit.ClaimedRevision < 1 ||
		commit.ExpectedProgressRevision < 1 || commit.ExpectedCheckpointID < 0 ||
		commit.NextCheckpointID <= commit.ExpectedCheckpointID || commit.FailedCount < 0 ||
		commit.StaleCount < 0 || commit.UpdatedAt.IsZero() || maximumItems < 1 ||
		len(commit.Batch.GenerationID) < 8 || len(commit.Batch.GenerationID) > 128 ||
		commit.Batch.LibraryID < 1 || processed < 1 || processed > int64(maximumItems) {
		return ErrInvalidEmbeddingRecord
	}
	if len(commit.Batch.Items) == 0 {
		return nil
	}
	return ValidateEmbeddingBatch(commit.Batch, maximumItems)
}

func ValidateEmbeddingBatch(batch EmbeddingBatch, maximumItems int) error {
	if len(batch.GenerationID) < 8 || len(batch.GenerationID) > 128 || batch.LibraryID < 1 ||
		maximumItems < 1 || len(batch.Items) < 1 || len(batch.Items) > maximumItems {
		return ErrInvalidEmbeddingRecord
	}
	seen := make(map[int64]struct{}, len(batch.Items))
	for _, item := range batch.Items {
		if item.AssetID < 1 || len(item.SourceFingerprint) < 1 || len(item.SourceFingerprint) > 256 ||
			len(item.Vector) < 2 || len(item.Vector)%2 != 0 || item.CreatedAt.IsZero() {
			return ErrInvalidEmbeddingRecord
		}
		if _, exists := seen[item.AssetID]; exists {
			return ErrInvalidEmbeddingRecord
		}
		seen[item.AssetID] = struct{}{}
	}
	return nil
}
