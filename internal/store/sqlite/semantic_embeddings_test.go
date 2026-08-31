package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/semantic"
)

func TestSemanticEmbeddingRepositoryIsBoundedIdempotentAndInvalidatesSource(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	generationID := seedEmbeddingGeneration(t, store, 2)
	firstID := catalogAssetID(t, store, "photo-10.jpg")
	secondID := catalogAssetID(t, store, "photo-2.jpg")
	vector, err := semantic.EncodeEmbedding([]float32{3, 4}, 2)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 27, 21, 0, 0, 0, time.UTC)
	batch := semantic.EmbeddingBatch{GenerationID: generationID, LibraryID: libraryID, Items: []semantic.EmbeddingItem{
		{AssetID: firstID, SourceFingerprint: "v1:10:10", Vector: vector, CreatedAt: now},
		{AssetID: secondID, SourceFingerprint: "v1:20:20", Vector: vector, CreatedAt: now},
	}}
	if err := store.PutSemanticEmbeddingBatch(context.Background(), batch); err != nil {
		t.Fatal(err)
	}
	batch.Items[0].CreatedAt = now.Add(time.Hour)
	if err := store.PutSemanticEmbeddingBatch(context.Background(), batch); err != nil {
		t.Fatal(err)
	}
	stored, found, err := store.GetSemanticEmbedding(context.Background(), generationID, libraryID, firstID)
	if err != nil || !found || !stored.CreatedAt.Equal(now) || stored.SourceFingerprint != "v1:10:10" {
		t.Fatalf("stored = %#v, %v, %v", stored, found, err)
	}
	if deleted, err := store.DeleteSemanticEmbeddingIfSourceChanged(context.Background(), generationID, libraryID, firstID, "v1:10:10"); err != nil || deleted {
		t.Fatalf("unchanged delete = %v, %v", deleted, err)
	}
	if deleted, err := store.DeleteSemanticEmbeddingIfSourceChanged(context.Background(), generationID, libraryID, firstID, "v1:11:11"); err != nil || !deleted {
		t.Fatalf("stale delete = %v, %v", deleted, err)
	}
	if _, found, err := store.GetSemanticEmbedding(context.Background(), generationID, libraryID, firstID); err != nil || found {
		t.Fatalf("deleted lookup = %v, %v", found, err)
	}
	if _, err := store.db.ExecContext(context.Background(), `DELETE FROM assets WHERE library_id = ? AND id = ?`, libraryID, secondID); err != nil {
		t.Fatal(err)
	}
	if _, found, err := store.GetSemanticEmbedding(context.Background(), generationID, libraryID, secondID); err != nil || found {
		t.Fatalf("cascade lookup = %v, %v", found, err)
	}
}

func TestSemanticEmbeddingRepositoryRejectsMalformedOrRetiredWrites(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	generationID := seedEmbeddingGeneration(t, store, 2)
	assetID := catalogAssetID(t, store, "photo-10.jpg")
	now := time.Date(2026, 8, 27, 22, 0, 0, 0, time.UTC)
	invalid := semantic.EmbeddingBatch{GenerationID: generationID, LibraryID: libraryID, Items: []semantic.EmbeddingItem{
		{AssetID: assetID, SourceFingerprint: "v1:10:10", Vector: []byte{0, 0}, CreatedAt: now},
	}}
	if err := store.PutSemanticEmbeddingBatch(context.Background(), invalid); !errors.Is(err, semantic.ErrInvalidEmbeddingRecord) {
		t.Fatalf("malformed error = %v", err)
	}
	if _, err := store.db.ExecContext(context.Background(), `UPDATE semantic_generations SET state = 'retired' WHERE id = ?`, generationID); err != nil {
		t.Fatal(err)
	}
	valid, _ := semantic.EncodeEmbedding([]float32{1, 1}, 2)
	invalid.Items[0].Vector = valid
	if err := store.PutSemanticEmbeddingBatch(context.Background(), invalid); !errors.Is(err, semantic.ErrSemanticGenerationUnavailable) {
		t.Fatalf("retired error = %v", err)
	}
}

func TestSemanticEmbeddingProgressCommitIsAtomicAndRejectsStaleWorker(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	generationID := seedEmbeddingGeneration(t, store, 2)
	assetID := catalogAssetID(t, store, "photo-10.jpg")
	now := time.Date(2026, 8, 27, 23, 0, 0, 0, time.UTC)
	seedEmbeddingJob(t, store, generationID, libraryID, now, 2)
	vector, _ := semantic.EncodeEmbedding([]float32{3, 4}, 2)
	commit := semantic.EmbeddingProgressCommit{
		JobID: "aij_embedding_job", ClaimedRevision: 7,
		ExpectedProgressRevision: 1, ExpectedCheckpointID: 0, NextCheckpointID: assetID,
		Batch: semantic.EmbeddingBatch{GenerationID: generationID, LibraryID: libraryID, Items: []semantic.EmbeddingItem{
			{AssetID: assetID, SourceFingerprint: "v1:10:10", Vector: vector, CreatedAt: now},
		}}, UpdatedAt: now,
	}
	progress, err := store.CommitSemanticEmbeddingProgress(context.Background(), commit)
	if err != nil || progress.CompletedCount != 1 || progress.Revision != 2 || progress.CheckpointID != assetID {
		t.Fatalf("progress = %#v, %v", progress, err)
	}
	assertEmbeddingJobState(t, store, assetID, 1, 2)
	if _, found, err := store.GetSemanticEmbedding(context.Background(), generationID, libraryID, assetID); err != nil || !found {
		t.Fatalf("embedding = %v, %v", found, err)
	}
	if _, err := store.CommitSemanticEmbeddingProgress(context.Background(), commit); !errors.Is(err, semantic.ErrSemanticProgressConflict) {
		t.Fatalf("stale commit error = %v", err)
	}
	assertEmbeddingJobState(t, store, assetID, 1, 2)

	commit.ExpectedProgressRevision = 2
	commit.ExpectedCheckpointID = assetID
	commit.NextCheckpointID = assetID + 1
	commit.Batch.Items[0].AssetID = 999_999
	if _, err := store.CommitSemanticEmbeddingProgress(context.Background(), commit); err == nil {
		t.Fatal("foreign-key failure commit succeeded")
	}
	progress, found, err := store.GetSemanticEmbeddingProgress(context.Background(), generationID, libraryID)
	if err != nil || !found || progress.CompletedCount != 1 || progress.Revision != 2 || progress.CheckpointID != assetID {
		t.Fatalf("progress after rollback = %#v, %v, %v", progress, found, err)
	}
	assertEmbeddingJobState(t, store, assetID, 1, 2)
}

func TestSemanticEmbeddingProgressCanCommitFailureWithoutVector(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	generationID := seedEmbeddingGeneration(t, store, 2)
	now := time.Date(2026, 8, 28, 0, 0, 0, 0, time.UTC)
	seedEmbeddingJob(t, store, generationID, libraryID, now, 1)
	progress, err := store.CommitSemanticEmbeddingProgress(context.Background(), semantic.EmbeddingProgressCommit{
		JobID: "aij_embedding_job", ClaimedRevision: 7,
		ExpectedProgressRevision: 1, ExpectedCheckpointID: 0, NextCheckpointID: 99,
		Batch:       semantic.EmbeddingBatch{GenerationID: generationID, LibraryID: libraryID},
		FailedCount: 1, UpdatedAt: now,
	})
	if err != nil || progress.FailedCount != 1 || progress.CompletedCount != 0 || progress.Revision != 2 {
		t.Fatalf("failed progress = %#v, %v", progress, err)
	}
	assertEmbeddingJobState(t, store, 99, 1, 2)
}

func seedEmbeddingGeneration(t *testing.T, store *Store, dimension int) string {
	t.Helper()
	now := time.Date(2026, 8, 27, 20, 0, 0, 0, time.UTC)
	model := activationModelFixture(now)
	if _, _, err := store.RegisterAIModel(context.Background(), model); err != nil {
		t.Fatal(err)
	}
	const generationID = "aig_embedding_generation"
	if _, err := store.db.ExecContext(context.Background(), `
        INSERT INTO semantic_generations(
            id, model_id, transform_version, output_schema_version, index_format_version,
            embedding_dimension, state, created_at_ms, activated_at_ms, updated_at_ms
        ) VALUES(?, ?, 1, 1, 1, ?, 'active', ?, ?, ?)`,
		generationID, model.ID, dimension, now.UnixMilli(), now.UnixMilli(), now.UnixMilli()); err != nil {
		t.Fatal(err)
	}
	return generationID
}

func seedEmbeddingJob(
	t *testing.T,
	store *Store,
	generationID string,
	libraryID int64,
	now time.Time,
	eligible int64,
) {
	t.Helper()
	if _, err := store.db.ExecContext(context.Background(), `
        INSERT INTO ai_model_operations(
            id, kind, state, phase, model_id, library_id, completed_items, total_items,
            revision, created_at_ms, updated_at_ms
        ) SELECT 'aio_embedding_operation', 'semantic_missing', 'running', 'building',
                 model_id, ?, 0, ?, 1, ?, ?
          FROM semantic_generations WHERE id = ?`,
		libraryID, eligible, now.UnixMilli(), now.UnixMilli(), generationID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(context.Background(), `
        INSERT INTO semantic_jobs(
            id, library_id, generation_id, operation_id, mode, state, checkpoint_id,
            requested_revision, claimed_revision, created_at_ms, updated_at_ms
        ) VALUES('aij_embedding_job', ?, ?, 'aio_embedding_operation', 'missing', 'running', 0, 1, 7, ?, ?)`,
		libraryID, generationID, now.UnixMilli(), now.UnixMilli()); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(context.Background(), `
        INSERT INTO semantic_library_progress(
            generation_id, library_id, eligible_count, completed_count, failed_count,
            stale_count, checkpoint_id, revision, updated_at_ms
        ) VALUES(?, ?, ?, 0, 0, 0, 0, 1, ?)`,
		generationID, libraryID, eligible, now.UnixMilli()); err != nil {
		t.Fatal(err)
	}
}

func assertEmbeddingJobState(t *testing.T, store *Store, checkpoint, completed, operationRevision int64) {
	t.Helper()
	var jobCheckpoint, operationCompleted, revision int64
	if err := store.db.QueryRowContext(context.Background(), `
        SELECT job.checkpoint_id, operation.completed_items, operation.revision
        FROM semantic_jobs AS job
        JOIN ai_model_operations AS operation ON operation.id = job.operation_id
        WHERE job.id = 'aij_embedding_job'`).Scan(&jobCheckpoint, &operationCompleted, &revision); err != nil {
		t.Fatal(err)
	}
	if jobCheckpoint != checkpoint || operationCompleted != completed || revision != operationRevision {
		t.Fatalf("job state = checkpoint %d completed %d revision %d", jobCheckpoint, operationCompleted, revision)
	}
}
