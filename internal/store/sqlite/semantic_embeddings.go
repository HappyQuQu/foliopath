package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/HappyQuQu/foliopath/internal/semantic"
)

func (s *Store) PutSemanticEmbeddingBatch(ctx context.Context, batch semantic.EmbeddingBatch) error {
	if err := semantic.ValidateEmbeddingBatch(batch, s.maxBatchSize); err != nil {
		return err
	}
	dimension, state, err := s.semanticGenerationContract(ctx, s.db, batch.GenerationID)
	if err != nil {
		return err
	}
	if !writableSemanticGeneration(state) {
		return semantic.ErrSemanticGenerationUnavailable
	}
	for _, item := range batch.Items {
		if _, err := semantic.DecodeEmbedding(item.Vector, dimension); err != nil {
			return semantic.ErrInvalidEmbeddingRecord
		}
	}

	err = s.withWriteTx(ctx, func(tx *sql.Tx) error {
		currentDimension, currentState, err := s.semanticGenerationContract(ctx, tx, batch.GenerationID)
		if err != nil {
			return err
		}
		if currentDimension != dimension || !writableSemanticGeneration(currentState) {
			return semantic.ErrSemanticGenerationUnavailable
		}
		return putSemanticEmbeddingItems(ctx, tx, batch)
	})
	if err != nil {
		return fmt.Errorf("put semantic embedding batch: %w", err)
	}
	return nil
}

func (s *Store) GetSemanticEmbeddingProgress(
	ctx context.Context,
	generationID string,
	libraryID int64,
) (semantic.EmbeddingProgress, bool, error) {
	if len(generationID) < 8 || len(generationID) > 128 || libraryID < 1 {
		return semantic.EmbeddingProgress{}, false, semantic.ErrInvalidEmbeddingRecord
	}
	value, err := scanSemanticEmbeddingProgress(s.db.QueryRowContext(ctx, semanticEmbeddingProgressSelect,
		generationID, libraryID))
	if errors.Is(err, sql.ErrNoRows) {
		return semantic.EmbeddingProgress{}, false, nil
	}
	if err != nil {
		return semantic.EmbeddingProgress{}, false, fmt.Errorf("get semantic embedding progress: %w", err)
	}
	return value, true, nil
}

func (s *Store) GetSemanticGenerationRuntime(ctx context.Context, generationID string) (semantic.GenerationRuntime, error) {
	if len(generationID) < 8 || len(generationID) > 128 {
		return semantic.GenerationRuntime{}, semantic.ErrSemanticGenerationUnavailable
	}
	var value semantic.GenerationRuntime
	err := s.db.QueryRowContext(ctx, `
        SELECT id, model_id, embedding_dimension, state
        FROM semantic_generations WHERE id = ?`, generationID).Scan(
		&value.ID, &value.ModelID, &value.EmbeddingDimension, &value.State,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return semantic.GenerationRuntime{}, semantic.ErrSemanticGenerationUnavailable
	}
	if err != nil {
		return semantic.GenerationRuntime{}, fmt.Errorf("get semantic generation runtime: %w", err)
	}
	return value, nil
}

func (s *Store) CommitSemanticEmbeddingProgress(
	ctx context.Context,
	commit semantic.EmbeddingProgressCommit,
) (semantic.EmbeddingProgress, error) {
	if err := semantic.ValidateEmbeddingProgressCommit(commit, s.maxBatchSize); err != nil {
		return semantic.EmbeddingProgress{}, err
	}
	dimension, state, err := s.semanticGenerationContract(ctx, s.db, commit.Batch.GenerationID)
	if err != nil {
		return semantic.EmbeddingProgress{}, err
	}
	if !writableSemanticGeneration(state) {
		return semantic.EmbeddingProgress{}, semantic.ErrSemanticGenerationUnavailable
	}
	for _, item := range commit.Batch.Items {
		if _, err := semantic.DecodeEmbedding(item.Vector, dimension); err != nil {
			return semantic.EmbeddingProgress{}, semantic.ErrInvalidEmbeddingRecord
		}
	}

	var updated semantic.EmbeddingProgress
	err = s.withWriteTx(ctx, func(tx *sql.Tx) error {
		var jobGenerationID, jobState, operationID, operationState string
		var jobLibraryID, jobCheckpoint, claimedRevision, operationCompleted int64
		var operationTotal sql.NullInt64
		if err := tx.QueryRowContext(ctx, `
            SELECT job.generation_id, job.library_id, job.state, job.checkpoint_id,
                   job.claimed_revision, job.operation_id, operation.state,
                   operation.completed_items, operation.total_items
            FROM semantic_jobs AS job
            JOIN ai_model_operations AS operation ON operation.id = job.operation_id
            WHERE job.id = ?`, commit.JobID).Scan(
			&jobGenerationID, &jobLibraryID, &jobState, &jobCheckpoint, &claimedRevision,
			&operationID, &operationState, &operationCompleted, &operationTotal,
		); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return semantic.ErrSemanticProgressConflict
			}
			return err
		}
		if jobGenerationID != commit.Batch.GenerationID || jobLibraryID != commit.Batch.LibraryID ||
			(jobState != "running" && jobState != "cancelling") ||
			(operationState != "running" && operationState != "cancelling") ||
			jobCheckpoint != commit.ExpectedCheckpointID || claimedRevision != commit.ClaimedRevision {
			return semantic.ErrSemanticProgressConflict
		}
		currentDimension, currentState, err := s.semanticGenerationContract(ctx, tx, commit.Batch.GenerationID)
		if err != nil {
			return err
		}
		if currentDimension != dimension || !writableSemanticGeneration(currentState) {
			return semantic.ErrSemanticGenerationUnavailable
		}
		current, err := scanSemanticEmbeddingProgress(tx.QueryRowContext(ctx, semanticEmbeddingProgressSelect,
			commit.Batch.GenerationID, commit.Batch.LibraryID))
		if err != nil || current.Revision != commit.ExpectedProgressRevision ||
			current.CheckpointID != commit.ExpectedCheckpointID {
			if errors.Is(err, sql.ErrNoRows) || err == nil {
				return semantic.ErrSemanticProgressConflict
			}
			return err
		}
		processed := int64(len(commit.Batch.Items)) + commit.FailedCount + commit.StaleCount
		if current.CompletedCount+current.FailedCount+current.StaleCount+processed > current.EligibleCount ||
			operationTotal.Valid && operationCompleted+processed > operationTotal.Int64 {
			return semantic.ErrInvalidEmbeddingRecord
		}
		if err := putSemanticEmbeddingItems(ctx, tx, commit.Batch); err != nil {
			return err
		}
		result, err := tx.ExecContext(ctx, `
            UPDATE semantic_library_progress SET
                completed_count = completed_count + ?,
                failed_count = failed_count + ?,
                stale_count = stale_count + ?,
                checkpoint_id = ?, revision = revision + 1, updated_at_ms = ?
            WHERE generation_id = ? AND library_id = ? AND revision = ? AND checkpoint_id = ?`,
			len(commit.Batch.Items), commit.FailedCount, commit.StaleCount, commit.NextCheckpointID,
			commit.UpdatedAt.UTC().UnixMilli(), commit.Batch.GenerationID, commit.Batch.LibraryID,
			commit.ExpectedProgressRevision, commit.ExpectedCheckpointID)
		if err != nil {
			return err
		}
		if rows, err := result.RowsAffected(); err != nil || rows != 1 {
			return semantic.ErrSemanticProgressConflict
		}
		result, err = tx.ExecContext(ctx, `
            UPDATE semantic_jobs SET checkpoint_id = ?, updated_at_ms = ?
            WHERE id = ? AND checkpoint_id = ? AND claimed_revision = ?`,
			commit.NextCheckpointID, commit.UpdatedAt.UTC().UnixMilli(), commit.JobID,
			commit.ExpectedCheckpointID, commit.ClaimedRevision)
		if err != nil {
			return err
		}
		if rows, err := result.RowsAffected(); err != nil || rows != 1 {
			return semantic.ErrSemanticProgressConflict
		}
		result, err = tx.ExecContext(ctx, `
            UPDATE ai_model_operations SET completed_items = completed_items + ?,
                revision = revision + 1, updated_at_ms = ?
            WHERE id = ? AND state IN ('running', 'cancelling')`,
			processed, commit.UpdatedAt.UTC().UnixMilli(), operationID)
		if err != nil {
			return err
		}
		if rows, err := result.RowsAffected(); err != nil || rows != 1 {
			return semantic.ErrSemanticProgressConflict
		}
		updated, err = scanSemanticEmbeddingProgress(tx.QueryRowContext(ctx, semanticEmbeddingProgressSelect,
			commit.Batch.GenerationID, commit.Batch.LibraryID))
		if err != nil {
			return err
		}
		state := "building"
		if updated.CompletedCount+updated.FailedCount+updated.StaleCount == updated.EligibleCount {
			state = "ready"
			if updated.FailedCount > 0 || updated.StaleCount > 0 {
				state = "degraded"
			}
		}
		_, err = tx.ExecContext(ctx, `
            UPDATE ai_library_settings SET state=?, coverage_revision=coverage_revision+1, updated_at_ms=?
            WHERE library_id=? AND enabled=1 AND state <> 'clearing'`,
			state, commit.UpdatedAt.UTC().UnixMilli(), commit.Batch.LibraryID)
		return err
	})
	if err != nil {
		return semantic.EmbeddingProgress{}, fmt.Errorf("commit semantic embedding progress: %w", err)
	}
	return updated, nil
}

func putSemanticEmbeddingItems(ctx context.Context, tx *sql.Tx, batch semantic.EmbeddingBatch) error {
	for _, item := range batch.Items {
		_, err := tx.ExecContext(ctx, `
            INSERT INTO semantic_embeddings(
                generation_id, library_id, asset_id, source_fingerprint, vector, created_at_ms
            ) VALUES(?, ?, ?, ?, ?, ?)
            ON CONFLICT(generation_id, library_id, asset_id) DO UPDATE SET
                source_fingerprint = excluded.source_fingerprint,
                vector = excluded.vector,
                created_at_ms = excluded.created_at_ms
            WHERE semantic_embeddings.source_fingerprint <> excluded.source_fingerprint
               OR semantic_embeddings.vector <> excluded.vector`,
			batch.GenerationID, batch.LibraryID, item.AssetID, item.SourceFingerprint,
			item.Vector, item.CreatedAt.UTC().UnixMilli())
		if err != nil {
			return err
		}
	}
	return nil
}

const semanticEmbeddingProgressSelect = `
    SELECT generation_id, library_id, eligible_count, completed_count, failed_count,
           stale_count, checkpoint_id, revision, updated_at_ms
    FROM semantic_library_progress
    WHERE generation_id = ? AND library_id = ?`

type semanticEmbeddingProgressScanner interface {
	Scan(...any) error
}

func scanSemanticEmbeddingProgress(row semanticEmbeddingProgressScanner) (semantic.EmbeddingProgress, error) {
	var value semantic.EmbeddingProgress
	var updatedAt int64
	err := row.Scan(&value.GenerationID, &value.LibraryID, &value.EligibleCount, &value.CompletedCount,
		&value.FailedCount, &value.StaleCount, &value.CheckpointID, &value.Revision, &updatedAt)
	if err != nil {
		return semantic.EmbeddingProgress{}, err
	}
	value.UpdatedAt = time.UnixMilli(updatedAt).UTC()
	return value, nil
}

var _ semantic.GenerationRuntimeRepository = (*Store)(nil)

func (s *Store) GetSemanticEmbedding(
	ctx context.Context,
	generationID string,
	libraryID, assetID int64,
) (semantic.StoredEmbedding, bool, error) {
	if len(generationID) < 8 || len(generationID) > 128 || libraryID < 1 || assetID < 1 {
		return semantic.StoredEmbedding{}, false, semantic.ErrInvalidEmbeddingRecord
	}
	var value semantic.StoredEmbedding
	var createdAt int64
	err := s.db.QueryRowContext(ctx, `
        SELECT generation_id, library_id, asset_id, source_fingerprint, vector, created_at_ms
        FROM semantic_embeddings
        WHERE generation_id = ? AND library_id = ? AND asset_id = ?`,
		generationID, libraryID, assetID,
	).Scan(&value.GenerationID, &value.LibraryID, &value.AssetID, &value.SourceFingerprint, &value.Vector, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return semantic.StoredEmbedding{}, false, nil
	}
	if err != nil {
		return semantic.StoredEmbedding{}, false, fmt.Errorf("get semantic embedding: %w", err)
	}
	value.CreatedAt = time.UnixMilli(createdAt).UTC()
	return value, true, nil
}

func (s *Store) DeleteSemanticEmbeddingIfSourceChanged(
	ctx context.Context,
	generationID string,
	libraryID, assetID int64,
	currentFingerprint string,
) (bool, error) {
	if len(generationID) < 8 || len(generationID) > 128 || libraryID < 1 || assetID < 1 ||
		len(currentFingerprint) < 1 || len(currentFingerprint) > 256 {
		return false, semantic.ErrInvalidEmbeddingRecord
	}
	deleted := false
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `
            DELETE FROM semantic_embeddings
            WHERE generation_id = ? AND library_id = ? AND asset_id = ? AND source_fingerprint <> ?`,
			generationID, libraryID, assetID, currentFingerprint)
		if err != nil {
			return err
		}
		rows, err := result.RowsAffected()
		if err != nil {
			return err
		}
		deleted = rows == 1
		return nil
	})
	if err != nil {
		return false, fmt.Errorf("delete stale semantic embedding: %w", err)
	}
	return deleted, nil
}

type semanticGenerationQuerier interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func (*Store) semanticGenerationContract(
	ctx context.Context,
	query semanticGenerationQuerier,
	generationID string,
) (int, string, error) {
	var dimension int
	var state string
	err := query.QueryRowContext(ctx, `
        SELECT embedding_dimension, state FROM semantic_generations WHERE id = ?`, generationID,
	).Scan(&dimension, &state)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, "", semantic.ErrSemanticGenerationUnavailable
	}
	if err != nil {
		return 0, "", err
	}
	return dimension, state, nil
}

func writableSemanticGeneration(state string) bool {
	return state == "building" || state == "ready" || state == "active"
}

var _ semantic.EmbeddingRepository = (*Store)(nil)
