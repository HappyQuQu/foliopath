package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/semantic"
)

func TestSemanticClearAdmissionWorksWithoutAvailableModelAndIsIdempotent(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	generationID := seedEmbeddingGeneration(t, store, 2)
	seedSemanticLibrarySettings(t, store, libraryID)
	assetID := catalogAssetID(t, store, "photo-10.jpg")
	vector, err := semantic.EncodeEmbedding([]float32{1, 0}, 2)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 28, 13, 0, 0, 0, time.UTC)
	if err := store.PutSemanticEmbeddingBatch(context.Background(), semantic.EmbeddingBatch{
		GenerationID: generationID, LibraryID: libraryID,
		Items: []semantic.EmbeddingItem{{AssetID: assetID, SourceFingerprint: "v1:10:10", Vector: vector, CreatedAt: now}},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(context.Background(), `UPDATE ai_models SET state='unavailable'`); err != nil {
		t.Fatal(err)
	}
	wake := &semanticWakeCounter{}
	ids := []string{"semclear_admitted", "aio_clear_admitted"}
	service, err := semantic.NewClearService(store, wake, func() time.Time { return now }, func(string) (string, error) {
		id := ids[0]
		ids = ids[1:]
		return id, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	first, err := service.Request(context.Background(), libraryID, 1, "semantic-clear-request-001")
	if err != nil || !first.Created || first.Job.TotalItems != 1 || wake.count != 1 {
		t.Fatalf("first=%#v wake=%d err=%v", first, wake.count, err)
	}
	var enabled int
	var state string
	var revision int64
	if err := store.db.QueryRowContext(context.Background(), `SELECT enabled, state, revision FROM ai_library_settings WHERE library_id=?`, libraryID).Scan(&enabled, &state, &revision); err != nil {
		t.Fatal(err)
	}
	if enabled != 0 || state != "clearing" || revision != 2 {
		t.Fatalf("settings enabled=%d state=%s revision=%d", enabled, state, revision)
	}
	replayed, err := service.Request(context.Background(), libraryID, 1, "semantic-clear-request-001")
	if err != nil || !replayed.Replayed || replayed.Job.ID != first.Job.ID || wake.count != 1 {
		t.Fatalf("replayed=%#v wake=%d err=%v", replayed, wake.count, err)
	}
	claimed, found, err := store.ClaimSemanticClear(context.Background(), now.Add(time.Minute), time.Minute)
	if err != nil || !found || claimed.State != semantic.JobRunning || claimed.ClaimedRevision < 1 {
		t.Fatalf("claimed=%#v found=%v err=%v", claimed, found, err)
	}
	processor, err := semantic.NewClearProcessor(store, func() time.Time { return now.Add(2 * time.Minute) }, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := processor.Process(context.Background(), claimed); err != nil {
		t.Fatal(err)
	}
	var embeddings int
	if err := store.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM semantic_embeddings WHERE library_id=?`, libraryID).Scan(&embeddings); err != nil {
		t.Fatal(err)
	}
	if err := store.db.QueryRowContext(context.Background(), `SELECT enabled, state, revision FROM ai_library_settings WHERE library_id=?`, libraryID).Scan(&enabled, &state, &revision); err != nil {
		t.Fatal(err)
	}
	if embeddings != 0 || enabled != 0 || state != "disabled" || revision != 3 {
		t.Fatalf("after clear embeddings=%d enabled=%d state=%s revision=%d", embeddings, enabled, state, revision)
	}
	var operationState string
	var completed, total int64
	if err := store.db.QueryRowContext(context.Background(), `SELECT state, completed_items, total_items FROM ai_model_operations WHERE id=?`, claimed.OperationID).Scan(&operationState, &completed, &total); err != nil {
		t.Fatal(err)
	}
	if operationState != "succeeded" || completed != 1 || total != 1 {
		t.Fatalf("operation state=%s completed=%d total=%d", operationState, completed, total)
	}
}

func TestSemanticClearAdmissionClampsSettingsTimeAcrossClockRollback(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	var libraryCreatedAt int64
	if err := store.db.QueryRowContext(context.Background(), `SELECT created_at_ms FROM libraries WHERE id=?`, libraryID).Scan(&libraryCreatedAt); err != nil {
		t.Fatal(err)
	}
	now := time.UnixMilli(libraryCreatedAt).Add(-time.Minute)
	ids := []string{"semclear_clock_rollback", "aio_clear_clock_rollback"}
	service, err := semantic.NewClearService(store, &semanticWakeCounter{}, func() time.Time { return now }, func(string) (string, error) {
		id := ids[0]
		ids = ids[1:]
		return id, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	requested, err := service.Request(context.Background(), libraryID, 1, "semantic-clear-clock-rollback")
	if err != nil || !requested.Created {
		t.Fatalf("requested=%+v err=%v", requested, err)
	}
	var createdAt, updatedAt int64
	if err := store.db.QueryRowContext(context.Background(), `SELECT created_at_ms,updated_at_ms FROM ai_library_settings WHERE library_id=?`, libraryID).Scan(&createdAt, &updatedAt); err != nil {
		t.Fatal(err)
	}
	if createdAt != libraryCreatedAt || updatedAt < createdAt {
		t.Fatalf("settings created_at_ms=%d updated_at_ms=%d library_created_at_ms=%d", createdAt, updatedAt, libraryCreatedAt)
	}
}

func TestSemanticClearLifecycleClampsTimeAcrossClockRollback(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	var libraryCreatedAt int64
	if err := store.db.QueryRowContext(context.Background(), `SELECT created_at_ms FROM libraries WHERE id=?`, libraryID).Scan(&libraryCreatedAt); err != nil {
		t.Fatal(err)
	}
	now := time.UnixMilli(libraryCreatedAt).Add(-time.Minute)
	ids := []string{"semclear_clock_lifecycle", "aio_clear_clock_lifecycle", "semclear_clock_cancel", "aio_clear_clock_cancel"}
	service, err := semantic.NewClearService(store, &semanticWakeCounter{}, func() time.Time { return now }, func(string) (string, error) {
		id := ids[0]
		ids = ids[1:]
		return id, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	requested, err := service.Request(context.Background(), libraryID, 1, "semantic-clear-lifecycle-clock-rollback")
	if err != nil || !requested.Created {
		t.Fatalf("requested=%+v err=%v", requested, err)
	}
	now = now.Add(-time.Minute)
	claimed, found, err := store.ClaimSemanticClear(context.Background(), now, time.Minute)
	if err != nil || !found || claimed.State != semantic.JobRunning {
		t.Fatalf("claimed=%+v found=%v err=%v", claimed, found, err)
	}
	if claimed.LeaseExpiresAt == nil || claimed.LeaseExpiresAt.Before(claimed.CreatedAt.Add(time.Minute)) {
		t.Fatalf("claimed created=%v lease=%v", claimed.CreatedAt, claimed.LeaseExpiresAt)
	}
	now = now.Add(-time.Minute)
	cancelled, err := store.RefreshSemanticClearLease(context.Background(), claimed, now, time.Minute)
	if err != nil || cancelled {
		t.Fatalf("refresh cancelled=%v err=%v", cancelled, err)
	}
	deleted, done, err := store.DeleteSemanticClearBatch(context.Background(), claimed, 1, now)
	if err != nil || deleted != 0 || !done {
		t.Fatalf("delete deleted=%d done=%v err=%v", deleted, done, err)
	}
	finished, err := store.FinishSemanticClear(context.Background(), claimed, semantic.JobSucceeded, "", now)
	if err != nil || finished.State != semantic.JobSucceeded {
		t.Fatalf("finished=%+v err=%v", finished, err)
	}
	assertSemanticClearTimeInvariants(t, store, libraryID, claimed.ID, claimed.OperationID)

	requested, err = service.Request(context.Background(), libraryID, 3, "semantic-clear-cancel-clock-rollback")
	if err != nil || !requested.Created {
		t.Fatalf("cancel request=%+v err=%v", requested, err)
	}
	now = now.Add(-time.Minute)
	cancelledJob, err := service.CancelOperation(context.Background(), requested.Job.OperationID, 1)
	if err != nil || cancelledJob.State != semantic.JobCancelled {
		t.Fatalf("cancelled=%+v err=%v", cancelledJob, err)
	}
	assertSemanticClearTimeInvariants(t, store, libraryID, requested.Job.ID, requested.Job.OperationID)
}

func assertSemanticClearTimeInvariants(t *testing.T, store *Store, libraryID int64, jobID, operationID string) {
	t.Helper()
	for _, check := range []struct {
		query string
		args  []any
	}{
		{`SELECT 1 FROM ai_library_settings WHERE library_id=? AND updated_at_ms>=created_at_ms`, []any{libraryID}},
		{`SELECT 1 FROM semantic_clear_jobs WHERE id=? AND updated_at_ms>=created_at_ms`, []any{jobID}},
		{`SELECT 1 FROM ai_model_operations WHERE id=? AND updated_at_ms>=created_at_ms AND (finished_at_ms IS NULL OR finished_at_ms>=created_at_ms)`, []any{operationID}},
	} {
		var valid int
		if err := store.db.QueryRowContext(context.Background(), check.query, check.args...).Scan(&valid); err != nil || valid != 1 {
			t.Fatalf("time invariant query=%q valid=%d err=%v", check.query, valid, err)
		}
	}
}

func TestSemanticClearAdmissionRejectsStaleSettingsAndActiveBackfill(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	generationID := seedEmbeddingGeneration(t, store, 2)
	seedSemanticLibrarySettings(t, store, libraryID)
	now := time.Date(2026, 8, 28, 14, 0, 0, 0, time.UTC)
	service, err := semantic.NewClearService(store, &semanticWakeCounter{}, func() time.Time { return now }, func(prefix string) (string, error) {
		return prefix + "_conflict_value", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Request(context.Background(), libraryID, 9, "semantic-clear-stale-001"); !errors.Is(err, semantic.ErrSemanticRevisionConflict) {
		t.Fatalf("stale revision error = %v", err)
	}
	admitSemanticBackfill(t, store, libraryID, generationID, now, "semantic-backfill-active-001")
	if _, err := service.Request(context.Background(), libraryID, 1, "semantic-clear-conflict-001"); !errors.Is(err, semantic.ErrSemanticClearConflict) {
		t.Fatalf("active backfill error = %v", err)
	}
}

func TestSemanticClearQueuedCancellationPreservesPartialDataAsDegraded(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	seedSemanticLibrarySettings(t, store, libraryID)
	now := time.Date(2026, 8, 28, 15, 0, 0, 0, time.UTC)
	ids := []string{"semclear_cancelled", "aio_clear_cancelled"}
	service, err := semantic.NewClearService(store, &semanticWakeCounter{}, func() time.Time { return now }, func(string) (string, error) {
		id := ids[0]
		ids = ids[1:]
		return id, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.Request(context.Background(), libraryID, 1, "semantic-clear-cancel-001")
	if err != nil {
		t.Fatal(err)
	}
	cancelled, err := service.CancelOperation(context.Background(), result.Job.OperationID, 1)
	if err != nil || cancelled.State != semantic.JobCancelled {
		t.Fatalf("cancelled=%#v err=%v", cancelled, err)
	}
	var enabled int
	var state string
	if err := store.db.QueryRowContext(context.Background(), `SELECT enabled, state FROM ai_library_settings WHERE library_id=?`, libraryID).Scan(&enabled, &state); err != nil {
		t.Fatal(err)
	}
	if enabled != 0 || state != "degraded" {
		t.Fatalf("settings enabled=%d state=%s", enabled, state)
	}
}

func TestSemanticClearResumesAfterLeaseExpiryWithoutLosingProgress(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	generationID := seedEmbeddingGeneration(t, store, 2)
	seedSemanticLibrarySettings(t, store, libraryID)
	page, err := store.ListSemanticBackfillCandidates(context.Background(), libraryID, generationID, semantic.JobAll, 0, 10)
	if err != nil || len(page.Items) < 2 {
		t.Fatalf("candidates=%#v err=%v", page, err)
	}
	now := time.Date(2026, 8, 28, 16, 0, 0, 0, time.UTC)
	vector, _ := semantic.EncodeEmbedding([]float32{1, 0}, 2)
	items := make([]semantic.EmbeddingItem, 0, len(page.Items))
	for _, candidate := range page.Items {
		items = append(items, semantic.EmbeddingItem{AssetID: candidate.AssetID, SourceFingerprint: candidate.SourceFingerprint, Vector: vector, CreatedAt: now})
	}
	if err := store.PutSemanticEmbeddingBatch(context.Background(), semantic.EmbeddingBatch{GenerationID: generationID, LibraryID: libraryID, Items: items}); err != nil {
		t.Fatal(err)
	}
	ids := []string{"semclear_resumable", "aio_clear_resumable"}
	service, err := semantic.NewClearService(store, &semanticWakeCounter{}, func() time.Time { return now }, func(string) (string, error) {
		id := ids[0]
		ids = ids[1:]
		return id, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	admitted, err := service.Request(context.Background(), libraryID, 1, "semantic-clear-resume-001")
	if err != nil {
		t.Fatal(err)
	}
	firstClaim, found, err := store.ClaimSemanticClear(context.Background(), now, time.Minute)
	if err != nil || !found {
		t.Fatalf("first claim=%#v found=%v err=%v", firstClaim, found, err)
	}
	deleted, done, err := store.DeleteSemanticClearBatch(context.Background(), firstClaim, 1, now.Add(10*time.Second))
	if err != nil || deleted != 1 || done {
		t.Fatalf("first batch deleted=%d done=%v err=%v", deleted, done, err)
	}
	summary, err := store.RecoverExpiredSemanticClears(context.Background(), now.Add(2*time.Minute))
	if err != nil || summary.Requeued != 1 || summary.Interrupted != 0 {
		t.Fatalf("recovery=%#v err=%v", summary, err)
	}
	secondClaim, found, err := store.ClaimSemanticClear(context.Background(), now.Add(3*time.Minute), time.Minute)
	if err != nil || !found || secondClaim.ID != admitted.Job.ID || secondClaim.AttemptCount != 2 || secondClaim.CompletedItems != 1 {
		t.Fatalf("second claim=%#v found=%v err=%v", secondClaim, found, err)
	}
	processor, err := semantic.NewClearProcessor(store, func() time.Time { return now.Add(3*time.Minute + time.Second) }, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := processor.Process(context.Background(), secondClaim); err != nil {
		t.Fatal(err)
	}
	var remaining int
	var operationState string
	var completed, total int64
	if err := store.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM semantic_embeddings WHERE library_id=?`, libraryID).Scan(&remaining); err != nil {
		t.Fatal(err)
	}
	if err := store.db.QueryRowContext(context.Background(), `SELECT state, completed_items, total_items FROM ai_model_operations WHERE id=?`, admitted.Job.OperationID).Scan(&operationState, &completed, &total); err != nil {
		t.Fatal(err)
	}
	if remaining != 0 || operationState != "succeeded" || completed != int64(len(items)) || total != int64(len(items)) {
		t.Fatalf("remaining=%d operation=%s completed=%d total=%d", remaining, operationState, completed, total)
	}
}

func TestRunningSemanticClearCancellationStopsAfterCommittedBatch(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	generationID := seedEmbeddingGeneration(t, store, 2)
	seedSemanticLibrarySettings(t, store, libraryID)
	page, err := store.ListSemanticBackfillCandidates(context.Background(), libraryID, generationID, semantic.JobAll, 0, 10)
	if err != nil || len(page.Items) < 2 {
		t.Fatalf("candidates=%#v err=%v", page, err)
	}
	now := time.Date(2026, 8, 28, 18, 0, 0, 0, time.UTC)
	vector, _ := semantic.EncodeEmbedding([]float32{1, 0}, 2)
	items := make([]semantic.EmbeddingItem, 0, len(page.Items))
	for _, candidate := range page.Items {
		items = append(items, semantic.EmbeddingItem{AssetID: candidate.AssetID, SourceFingerprint: candidate.SourceFingerprint, Vector: vector, CreatedAt: now})
	}
	if err := store.PutSemanticEmbeddingBatch(context.Background(), semantic.EmbeddingBatch{GenerationID: generationID, LibraryID: libraryID, Items: items}); err != nil {
		t.Fatal(err)
	}
	ids := []string{"semclear_running_cancel", "aio_clear_running_cancel"}
	service, err := semantic.NewClearService(store, &semanticWakeCounter{}, func() time.Time { return now }, func(string) (string, error) {
		id := ids[0]
		ids = ids[1:]
		return id, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	admitted, err := service.Request(context.Background(), libraryID, 1, "semantic-clear-running-cancel-001")
	if err != nil {
		t.Fatal(err)
	}
	claimed, found, err := store.ClaimSemanticClear(context.Background(), now, time.Minute)
	if err != nil || !found {
		t.Fatalf("claim=%#v found=%v err=%v", claimed, found, err)
	}
	if deleted, done, err := store.DeleteSemanticClearBatch(context.Background(), claimed, 1, now.Add(10*time.Second)); err != nil || deleted != 1 || done {
		t.Fatalf("batch deleted=%d done=%v err=%v", deleted, done, err)
	}
	var operationRevision int64
	if err := store.db.QueryRowContext(context.Background(), `SELECT revision FROM ai_model_operations WHERE id=?`, admitted.Job.OperationID).Scan(&operationRevision); err != nil {
		t.Fatal(err)
	}
	cancelling, err := service.CancelOperation(context.Background(), admitted.Job.OperationID, operationRevision)
	if err != nil || cancelling.State != semantic.JobCancelling {
		t.Fatalf("cancelling=%#v err=%v", cancelling, err)
	}
	if _, err := store.RecoverExpiredSemanticClears(context.Background(), now.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	var remaining int
	var operationState, settingsState string
	if err := store.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM semantic_embeddings WHERE library_id=?`, libraryID).Scan(&remaining); err != nil {
		t.Fatal(err)
	}
	if err := store.db.QueryRowContext(context.Background(), `SELECT state FROM ai_model_operations WHERE id=?`, admitted.Job.OperationID).Scan(&operationState); err != nil {
		t.Fatal(err)
	}
	if err := store.db.QueryRowContext(context.Background(), `SELECT state FROM ai_library_settings WHERE library_id=?`, libraryID).Scan(&settingsState); err != nil {
		t.Fatal(err)
	}
	if remaining != len(items)-1 || operationState != "cancelled" || settingsState != "degraded" {
		t.Fatalf("remaining=%d operation=%s settings=%s", remaining, operationState, settingsState)
	}
}
