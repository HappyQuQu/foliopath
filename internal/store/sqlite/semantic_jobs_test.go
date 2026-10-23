package sqlite

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/media"
	"github.com/HappyQuQu/foliopath/internal/semantic"
)

type semanticWakeCounter struct{ count int }

func (wake *semanticWakeCounter) Wake() { wake.count++ }

func TestSemanticBackfillAdmissionHashesKeyReplaysAndCoalesces(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	generationID := seedEmbeddingGeneration(t, store, 2)
	seedSemanticLibrarySettings(t, store, libraryID)
	now := time.Date(2026, 8, 28, 1, 0, 0, 0, time.UTC)
	ids := []string{"semjob_first_request", "aio_first_request", "semjob_coalesced_request", "aio_coalesced_request", "semjob_conflict_request", "aio_conflict_request"}
	wake := &semanticWakeCounter{}
	service, err := semantic.NewBackfillService(store, store, wake, func() time.Time { return now }, func(string) (string, error) {
		id := ids[0]
		ids = ids[1:]
		return id, nil
	})
	if err != nil {
		t.Fatal(err)
	}

	created, err := service.Request(context.Background(), libraryID, generationID, semantic.JobMissing, "semantic-key-001")
	if err != nil || !created.Created || created.Replayed || created.Coalesced || wake.count != 1 {
		t.Fatalf("created = %#v, wake %d, err %v", created, wake.count, err)
	}
	replayed, err := service.Request(context.Background(), libraryID, generationID, semantic.JobMissing, "semantic-key-001")
	if err != nil || !replayed.Replayed || replayed.Job.ID != created.Job.ID || wake.count != 1 {
		t.Fatalf("replayed = %#v, wake %d, err %v", replayed, wake.count, err)
	}
	coalesced, err := service.Request(context.Background(), libraryID, generationID, semantic.JobMissing, "semantic-key-002")
	if err != nil || !coalesced.Coalesced || coalesced.Replayed || coalesced.Job.ID != created.Job.ID || wake.count != 1 {
		t.Fatalf("coalesced = %#v, wake %d, err %v", coalesced, wake.count, err)
	}
	if _, err := service.Request(context.Background(), libraryID, generationID, semantic.JobAll, "semantic-key-003"); !errors.Is(err, semantic.ErrSemanticJobConflict) {
		t.Fatalf("conflicting active mode error = %v", err)
	}
	var plaintextCount int
	if err := store.db.QueryRowContext(context.Background(), `
        SELECT COUNT(*) FROM semantic_job_requests WHERE idempotency_key_hash IN (?, ?, ?)`,
		"semantic-key-001", "semantic-key-002", "semantic-key-003").Scan(&plaintextCount); err != nil || plaintextCount != 0 {
		t.Fatalf("plaintext key count = %d, err %v", plaintextCount, err)
	}
	var requestCount, jobCount, operationCount int
	if err := store.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM semantic_job_requests`).Scan(&requestCount); err != nil {
		t.Fatal(err)
	}
	if err := store.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM semantic_jobs`).Scan(&jobCount); err != nil {
		t.Fatal(err)
	}
	if err := store.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM ai_model_operations WHERE kind='semantic_missing'`).Scan(&operationCount); err != nil {
		t.Fatal(err)
	}
	if requestCount != 2 || jobCount != 1 || operationCount != 1 {
		t.Fatalf("rows = requests %d jobs %d operations %d", requestCount, jobCount, operationCount)
	}
}

func TestSemanticBackfillClaimLeaseCancellationAndFinishAreCASBound(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	generationID := seedEmbeddingGeneration(t, store, 2)
	seedSemanticLibrarySettings(t, store, libraryID)
	now := time.Date(2026, 8, 28, 2, 0, 0, 0, time.UTC)
	job := admitSemanticBackfill(t, store, libraryID, generationID, now, "semantic-key-claim")

	claimed, found, err := store.ClaimSemanticBackfill(context.Background(), now.Add(time.Second), 2*time.Minute)
	if err != nil || !found || claimed.ID != job.ID || claimed.State != semantic.JobRunning ||
		claimed.ClaimedRevision != 2 || claimed.AttemptCount != 1 || claimed.OperationRevision != 2 {
		t.Fatalf("claimed = %#v, found %v, err %v", claimed, found, err)
	}
	if _, err := store.CancelSemanticBackfill(context.Background(), claimed.ID, 1, now.Add(2*time.Second)); !errors.Is(err, semantic.ErrSemanticJobConflict) {
		t.Fatalf("stale cancel error = %v", err)
	}
	cancelling, err := store.CancelSemanticBackfill(context.Background(), claimed.ID, claimed.OperationRevision, now.Add(2*time.Second))
	if err != nil || cancelling.State != semantic.JobCancelling || cancelling.OperationRevision != 3 {
		t.Fatalf("cancelling = %#v, err %v", cancelling, err)
	}
	cancelRequested, err := store.RefreshSemanticBackfillLease(context.Background(), claimed, now.Add(3*time.Second), 2*time.Minute)
	if err != nil || !cancelRequested {
		t.Fatalf("refresh cancel = %v, %v", cancelRequested, err)
	}
	finished, err := store.FinishSemanticBackfill(context.Background(), claimed, semantic.JobCancelled, "", now.Add(4*time.Second))
	if err != nil || finished.State != semantic.JobCancelled || finished.ErrorCode != "cancelled" || finished.LeaseExpiresAt != nil {
		t.Fatalf("finished = %#v, err %v", finished, err)
	}
	if _, err := store.FinishSemanticBackfill(context.Background(), claimed, semantic.JobCancelled, "", now.Add(5*time.Second)); !errors.Is(err, semantic.ErrSemanticJobConflict) {
		t.Fatalf("stale finish error = %v", err)
	}
}

func TestSemanticBackfillLifecycleClampsTimeAcrossClockRollback(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	generationID := seedEmbeddingGeneration(t, store, 2)
	seedSemanticLibrarySettings(t, store, libraryID)
	var settingsUpdatedAt int64
	if err := store.db.QueryRowContext(context.Background(), `SELECT updated_at_ms FROM ai_library_settings WHERE library_id=?`, libraryID).Scan(&settingsUpdatedAt); err != nil {
		t.Fatal(err)
	}
	now := time.UnixMilli(settingsUpdatedAt).Add(-time.Minute)
	job := admitSemanticBackfill(t, store, libraryID, generationID, now, "semantic-backfill-clock-rollback")
	now = now.Add(-time.Minute)
	claimed, found, err := store.ClaimSemanticBackfill(context.Background(), now, time.Minute)
	if err != nil || !found || claimed.State != semantic.JobRunning {
		t.Fatalf("claimed=%+v found=%v err=%v", claimed, found, err)
	}
	if claimed.LeaseExpiresAt == nil || claimed.LeaseExpiresAt.Before(claimed.CreatedAt.Add(time.Minute)) {
		t.Fatalf("claimed created=%v lease=%v", claimed.CreatedAt, claimed.LeaseExpiresAt)
	}
	now = now.Add(-time.Minute)
	cancelling, err := store.CancelSemanticBackfill(context.Background(), job.ID, claimed.OperationRevision, now)
	if err != nil || cancelling.State != semantic.JobCancelling {
		t.Fatalf("cancelling=%+v err=%v", cancelling, err)
	}
	cancelRequested, err := store.RefreshSemanticBackfillLease(context.Background(), claimed, now.Add(-time.Minute), time.Minute)
	if err != nil || !cancelRequested {
		t.Fatalf("refresh cancel=%v err=%v", cancelRequested, err)
	}
	finished, err := store.FinishSemanticBackfill(context.Background(), claimed, semantic.JobCancelled, "", now.Add(-2*time.Minute))
	if err != nil || finished.State != semantic.JobCancelled {
		t.Fatalf("finished=%+v err=%v", finished, err)
	}
	for _, check := range []struct {
		query string
		id    string
	}{
		{`SELECT 1 FROM semantic_jobs WHERE id=? AND updated_at_ms>=created_at_ms`, claimed.ID},
		{`SELECT 1 FROM ai_model_operations WHERE id=? AND updated_at_ms>=created_at_ms AND finished_at_ms>=created_at_ms`, claimed.OperationID},
	} {
		var valid int
		if err := store.db.QueryRowContext(context.Background(), check.query, check.id).Scan(&valid); err != nil || valid != 1 {
			t.Fatalf("time invariant query=%q valid=%d err=%v", check.query, valid, err)
		}
	}
}

func TestSemanticBackfillRecoveryRequeuesThenFailsAtAttemptLimit(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	generationID := seedEmbeddingGeneration(t, store, 2)
	seedSemanticLibrarySettings(t, store, libraryID)
	now := time.Date(2026, 8, 28, 3, 0, 0, 0, time.UTC)
	admitSemanticBackfill(t, store, libraryID, generationID, now, "semantic-key-recover")

	var previousClaim semantic.BackfillJob
	for attempt := 1; attempt <= semantic.MaximumSemanticJobAttempts; attempt++ {
		claimAt := now.Add(time.Duration(attempt) * time.Minute)
		claimed, found, err := store.ClaimSemanticBackfill(context.Background(), claimAt, time.Second)
		if err != nil || !found || claimed.AttemptCount != attempt {
			t.Fatalf("attempt %d claim = %#v, %v, %v", attempt, claimed, found, err)
		}
		if attempt > 1 && claimed.ClaimedRevision <= previousClaim.ClaimedRevision {
			t.Fatalf("claim revision did not advance: previous %d current %d", previousClaim.ClaimedRevision, claimed.ClaimedRevision)
		}
		previousClaim = claimed
		summary, err := store.RecoverExpiredSemanticBackfills(context.Background(), claimAt.Add(2*time.Second))
		if err != nil {
			t.Fatal(err)
		}
		if attempt < semantic.MaximumSemanticJobAttempts && (summary.Requeued != 1 || summary.Interrupted != 0) {
			t.Fatalf("attempt %d recovery = %#v", attempt, summary)
		}
		if attempt == semantic.MaximumSemanticJobAttempts && (summary.Requeued != 0 || summary.Interrupted != 1) {
			t.Fatalf("final recovery = %#v", summary)
		}
	}
	stored, err := scanSemanticJob(store.db.QueryRowContext(context.Background(), semanticJobSelect+` WHERE job.id = ?`, previousClaim.ID))
	if err != nil || stored.State != semantic.JobFailed || stored.ErrorCode != "operation_interrupted" {
		t.Fatalf("recovered job = %#v, err %v", stored, err)
	}
}

func TestSemanticBackfillClaimGivesUnservedLibraryPriority(t *testing.T) {
	store, _ := openTestStore(t)
	firstLibrary := createWorkerLibrary(t, store, "Semantic First", "semantic-first")
	secondLibrary := createWorkerLibrary(t, store, "Semantic Second", "semantic-second")
	generationID := seedEmbeddingGeneration(t, store, 2)
	seedSemanticLibrarySettings(t, store, firstLibrary.ID)
	seedSemanticLibrarySettings(t, store, secondLibrary.ID)

	now := time.Date(2026, 8, 28, 3, 30, 0, 0, time.UTC)
	ids := []string{
		"semjob_fair_first", "aio_fair_first",
		"semjob_fair_first_again", "aio_fair_first_again",
		"semjob_fair_second", "aio_fair_second",
	}
	service, err := semantic.NewBackfillService(store, store, &semanticWakeCounter{}, func() time.Time { return now }, func(string) (string, error) {
		id := ids[0]
		ids = ids[1:]
		return id, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Request(context.Background(), firstLibrary.ID, generationID, semantic.JobMissing, "semantic-fair-first-001"); err != nil {
		t.Fatal(err)
	}
	claimed, found, err := store.ClaimSemanticBackfill(context.Background(), now.Add(time.Second), time.Minute)
	if err != nil || !found || claimed.LibraryID != firstLibrary.ID {
		t.Fatalf("initial claim = %#v, found %v, err %v", claimed, found, err)
	}
	if _, err := store.FinishSemanticBackfill(context.Background(), claimed, semantic.JobSucceeded, "", now.Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}

	now = now.Add(3 * time.Second)
	if _, err := service.Request(context.Background(), firstLibrary.ID, generationID, semantic.JobMissing, "semantic-fair-first-002"); err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Second)
	if _, err := service.Request(context.Background(), secondLibrary.ID, generationID, semantic.JobMissing, "semantic-fair-second-001"); err != nil {
		t.Fatal(err)
	}

	claimed, found, err = store.ClaimSemanticBackfill(context.Background(), now.Add(time.Second), time.Minute)
	if err != nil || !found || claimed.LibraryID != secondLibrary.ID {
		t.Fatalf("fair claim = %#v, found %v, err %v", claimed, found, err)
	}
}

func TestSemanticBackfillCandidatesUseBoundedAssetIDKeysetAndFingerprint(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	generationID := seedEmbeddingGeneration(t, store, 2)
	now := time.Date(2026, 8, 28, 4, 0, 0, 0, time.UTC)

	counts, err := store.CountSemanticBackfillCandidates(context.Background(), libraryID, generationID, semantic.JobAll)
	if err != nil || counts.Eligible < 2 || counts.Pending != counts.Eligible {
		t.Fatalf("all counts = %#v, err %v", counts, err)
	}
	total := counts.Eligible
	seen := make(map[int64]struct{})
	checkpoint := int64(0)
	for {
		page, err := store.ListSemanticBackfillCandidates(context.Background(), libraryID, generationID, semantic.JobAll, checkpoint, 1)
		if err != nil || len(page.Items) == 0 {
			t.Fatalf("page after %d = %#v, err %v", checkpoint, page, err)
		}
		item := page.Items[0]
		if item.AssetID <= checkpoint {
			t.Fatalf("non-monotonic candidate %d after %d", item.AssetID, checkpoint)
		}
		if _, duplicate := seen[item.AssetID]; duplicate {
			t.Fatalf("duplicate candidate %d", item.AssetID)
		}
		seen[item.AssetID] = struct{}{}
		checkpoint = page.Checkpoint
		if !page.HasMore {
			break
		}
	}
	if int64(len(seen)) != total {
		t.Fatalf("keyset rows = %d, count %d", len(seen), total)
	}

	assetID := catalogAssetID(t, store, "photo-10.jpg")
	var fingerprint string
	if err := store.db.QueryRowContext(context.Background(), `SELECT source_fingerprint FROM assets WHERE library_id=? AND id=?`, libraryID, assetID).Scan(&fingerprint); err != nil {
		t.Fatal(err)
	}
	vector, _ := semantic.EncodeEmbedding([]float32{3, 4}, 2)
	if err := store.PutSemanticEmbeddingBatch(context.Background(), semantic.EmbeddingBatch{
		GenerationID: generationID, LibraryID: libraryID,
		Items: []semantic.EmbeddingItem{{AssetID: assetID, SourceFingerprint: fingerprint, Vector: vector, CreatedAt: now}},
	}); err != nil {
		t.Fatal(err)
	}
	counts, err = store.CountSemanticBackfillCandidates(context.Background(), libraryID, generationID, semantic.JobMissing)
	if err != nil || counts.Eligible != total || counts.Pending != total-1 {
		t.Fatalf("missing counts = %#v, want eligible %d pending %d, err %v", counts, total, total-1, err)
	}
	if _, err := store.db.ExecContext(context.Background(), `UPDATE assets SET source_fingerprint='v1:999:999' WHERE library_id=? AND id=?`, libraryID, assetID); err != nil {
		t.Fatal(err)
	}
	counts, err = store.CountSemanticBackfillCandidates(context.Background(), libraryID, generationID, semantic.JobMissing)
	if err != nil || counts.Eligible != total || counts.Pending != total {
		t.Fatalf("stale fingerprint counts = %#v, want %d/%d, err %v", counts, total, total, err)
	}
}

func TestSemanticBackfillProcessorCommitsBoundedPagesAndTerminalSuccess(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	generationID := seedEmbeddingGeneration(t, store, 2)
	seedSemanticLibrarySettings(t, store, libraryID)
	now := time.Date(2026, 8, 28, 5, 0, 0, 0, time.UTC)
	admitSemanticBackfill(t, store, libraryID, generationID, now, "semantic-key-process")
	job, found, err := store.ClaimSemanticBackfill(context.Background(), now.Add(time.Second), time.Minute)
	if err != nil || !found {
		t.Fatalf("claim = %#v, %v, %v", job, found, err)
	}
	processor, err := semantic.NewBackfillProcessor(
		store,
		semanticAssetSourceStub{store: store},
		semanticPreprocessorStub{},
		semanticEncoderStub{},
		store,
		store,
		func() time.Time { now = now.Add(time.Millisecond); return now },
		1,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := processor.Process(context.Background(), job); err != nil {
		t.Fatal(err)
	}
	stored, err := scanSemanticJob(store.db.QueryRowContext(context.Background(), semanticJobSelect+` WHERE job.id=?`, job.ID))
	if err != nil || stored.State != semantic.JobSucceeded || stored.CompletedItems != stored.TotalItems {
		t.Fatalf("stored job = %#v, err %v", stored, err)
	}
	var embeddingCount int64
	if err := store.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM semantic_embeddings WHERE generation_id=? AND library_id=?`, generationID, libraryID).Scan(&embeddingCount); err != nil {
		t.Fatal(err)
	}
	if embeddingCount != stored.TotalItems {
		t.Fatalf("embedding count = %d, total %d", embeddingCount, stored.TotalItems)
	}
	progress, found, err := store.GetSemanticEmbeddingProgress(context.Background(), generationID, libraryID)
	if err != nil || !found || progress.CompletedCount != stored.TotalItems || progress.FailedCount != 0 || progress.StaleCount != 0 {
		t.Fatalf("progress = %#v, found %v, err %v", progress, found, err)
	}
	settings, err := store.GetSemanticLibrarySettings(context.Background(), libraryID)
	if err != nil || settings.State != semantic.LibraryReady {
		t.Fatalf("ready settings = %#v, err %v", settings, err)
	}
}

func TestSemanticBackfillProcessorFailsClosedWhenEncoderUnavailable(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	generationID := seedEmbeddingGeneration(t, store, 2)
	seedSemanticLibrarySettings(t, store, libraryID)
	now := time.Date(2026, 8, 28, 6, 0, 0, 0, time.UTC)
	admitSemanticBackfill(t, store, libraryID, generationID, now, "semantic-key-no-encoder")
	job, found, err := store.ClaimSemanticBackfill(context.Background(), now.Add(time.Second), time.Minute)
	if err != nil || !found {
		t.Fatalf("claim = %#v, %v, %v", job, found, err)
	}
	processor, err := semantic.NewBackfillProcessor(store, semanticAssetSourceStub{store: store},
		semanticPreprocessorStub{}, semanticEncoderStub{err: semantic.ErrImageEncoderUnavailable},
		store, store, func() time.Time { return now.Add(2 * time.Second) }, 2)
	if err != nil {
		t.Fatal(err)
	}
	if err := processor.Process(context.Background(), job); !errors.Is(err, semantic.ErrImageEncoderUnavailable) {
		t.Fatalf("processor error = %v", err)
	}
	stored, err := scanSemanticJob(store.db.QueryRowContext(context.Background(), semanticJobSelect+` WHERE job.id=?`, job.ID))
	if err != nil || stored.State != semantic.JobFailed || stored.ErrorCode != "model_unavailable" || stored.CompletedItems != 0 {
		t.Fatalf("stored job = %#v, err %v", stored, err)
	}
}

type semanticReadSeekCloser struct{ *bytes.Reader }

func (semanticReadSeekCloser) Close() error { return nil }

type semanticAssetSourceStub struct{ store *Store }

func (source semanticAssetSourceStub) OpenSemanticAsset(ctx context.Context, libraryID, assetID int64) (semantic.SemanticAsset, error) {
	var fingerprint, format string
	if err := source.store.db.QueryRowContext(ctx, `SELECT source_fingerprint, media_format FROM assets WHERE library_id=? AND id=?`, libraryID, assetID).Scan(&fingerprint, &format); err != nil {
		return semantic.SemanticAsset{}, err
	}
	return semantic.SemanticAsset{File: semanticReadSeekCloser{bytes.NewReader([]byte{byte(assetID)})},
		Format: media.Format(format), SourceFingerprint: fingerprint}, nil
}

type semanticPreprocessorStub struct{}

func (semanticPreprocessorStub) PrepareSemanticImage(_ context.Context, source io.ReadSeeker, _ media.Format) ([]float32, error) {
	value := []byte{0}
	if _, err := source.Read(value); err != nil {
		return nil, err
	}
	return []float32{float32(value[0])}, nil
}

type semanticEncoderStub struct{ err error }

func (stub semanticEncoderStub) EncodeSemanticImage(context.Context, string, []float32) ([]float32, error) {
	if stub.err != nil {
		return nil, stub.err
	}
	return []float32{3, 4}, nil
}

func seedSemanticLibrarySettings(t *testing.T, store *Store, libraryID int64) {
	t.Helper()
	now := time.Date(2026, 8, 28, 0, 0, 0, 0, time.UTC).UnixMilli()
	if _, err := store.db.ExecContext(context.Background(), `
        INSERT INTO ai_library_settings(library_id, enabled, state, revision, coverage_revision, created_at_ms, updated_at_ms)
        VALUES(?, 1, 'building', 1, 1, ?, ?)`, libraryID, now, now); err != nil {
		t.Fatal(err)
	}
}

func admitSemanticBackfill(t *testing.T, store *Store, libraryID int64, generationID string, now time.Time, key string) semantic.BackfillJob {
	t.Helper()
	wake := &semanticWakeCounter{}
	ids := []string{"semjob_admitted_work", "aio_admitted_work"}
	service, err := semantic.NewBackfillService(store, store, wake, func() time.Time { return now }, func(string) (string, error) {
		id := ids[0]
		ids = ids[1:]
		return id, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.Request(context.Background(), libraryID, generationID, semantic.JobMissing, key)
	if err != nil || !result.Created || wake.count != 1 {
		t.Fatalf("admit = %#v, wake %d, err %v", result, wake.count, err)
	}
	return result.Job
}
