package sqlite

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/HappyQuQu/foliopath/internal/face"
	"testing"
	"time"
)

type scriptedFaceAnalyzer struct {
	calls int
}

type failingFaceAnalyzer struct{ calls int }

type unavailableFaceAnalyzer struct{ calls int }

type offlineAfterFirstFaceAnalyzer struct {
	store     *Store
	libraryID int64
	calls     int
}

type unavailableAfterFirstFaceAnalyzer struct {
	store     *Store
	libraryID int64
	now       time.Time
	calls     int
}

type disablingFaceClusterRebuilder struct {
	store     *Store
	libraryID int64
	now       time.Time
}

type cancellingFaceClusterRebuilder struct {
	store       *Store
	operationID string
	now         time.Time
}

func TestCancelFaceJobDoesNotMisclassifyStorageFailure(t *testing.T) {
	store, _ := openTestStore(t)
	if err := store.db.Close(); err != nil {
		t.Fatal(err)
	}
	_, err := store.CancelFaceJobOperation(context.Background(), "face_operation_storage_failure", 1, time.Now().UTC())
	if err == nil || errors.Is(err, face.ErrFaceJobNotFound) {
		t.Fatalf("storage failure=%v", err)
	}
}

func TestFaceJobLifecycleClampsTimeAcrossClockRollback(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	var libraryCreatedAt int64
	if err := store.db.QueryRowContext(context.Background(), `SELECT created_at_ms FROM libraries WHERE id=?`, libraryID).Scan(&libraryCreatedAt); err != nil {
		t.Fatal(err)
	}
	seedAt := time.UnixMilli(libraryCreatedAt)
	const generationID = "face_generation_clock_rollback"
	seedFaceGeneration(t, store, generationID, 2, "active", seedAt)
	seedFaceReadySettings(t, store, libraryID, generationID, seedAt)
	now := seedAt.Add(-time.Minute)
	service, err := face.NewJobService(store, store, &faceClearWaker{}, func() time.Time { return now }, faceClearIDs())
	if err != nil {
		t.Fatal(err)
	}
	requested, err := service.Request(context.Background(), libraryID, generationID, face.JobMissing, "face-job-lifecycle-clock-rollback")
	if err != nil || !requested.Created {
		t.Fatalf("requested=%+v err=%v", requested, err)
	}
	now = now.Add(-time.Minute)
	claimed, found, err := store.ClaimFaceJob(context.Background(), now, time.Minute)
	if err != nil || !found || claimed.State != "running" {
		t.Fatalf("claimed=%+v found=%v err=%v", claimed, found, err)
	}
	if claimed.LeaseExpiresAt == nil || claimed.LeaseExpiresAt.Before(claimed.CreatedAt.Add(time.Minute)) {
		t.Fatalf("claimed created=%v lease=%v", claimed.CreatedAt, claimed.LeaseExpiresAt)
	}
	now = now.Add(-time.Minute)
	cancelled, err := store.RefreshFaceJobLease(context.Background(), claimed, now, time.Minute)
	if err != nil || cancelled {
		t.Fatalf("refresh cancelled=%v err=%v", cancelled, err)
	}
	finished, err := store.FinishFaceJob(context.Background(), claimed, false, "internal_error", now)
	if err != nil || finished.State != "failed" {
		t.Fatalf("finished=%+v err=%v", finished, err)
	}
	assertFaceJobTimeInvariants(t, store, libraryID, claimed.ID, claimed.OperationID)

	requested, err = service.Request(context.Background(), libraryID, generationID, face.JobAll, "face-job-cancel-clock-rollback")
	if err != nil || !requested.Created {
		t.Fatalf("cancel request=%+v err=%v", requested, err)
	}
	now = now.Add(-time.Minute)
	cancelledJob, err := service.Cancel(context.Background(), requested.Job.OperationID, 1)
	if err != nil || cancelledJob.State != "cancelled" {
		t.Fatalf("cancelled=%+v err=%v", cancelledJob, err)
	}
	assertFaceJobTimeInvariants(t, store, libraryID, requested.Job.ID, requested.Job.OperationID)
}

func assertFaceJobTimeInvariants(t *testing.T, store *Store, libraryID int64, jobID, operationID string) {
	t.Helper()
	for _, check := range []struct {
		query string
		args  []any
	}{
		{`SELECT 1 FROM face_library_settings WHERE library_id=? AND updated_at_ms>=created_at_ms`, []any{libraryID}},
		{`SELECT 1 FROM face_analysis_jobs WHERE id=? AND updated_at_ms>=created_at_ms`, []any{jobID}},
		{`SELECT 1 FROM ai_model_operations WHERE id=? AND updated_at_ms>=created_at_ms AND (finished_at_ms IS NULL OR finished_at_ms>=created_at_ms)`, []any{operationID}},
	} {
		var valid int
		if err := store.db.QueryRowContext(context.Background(), check.query, check.args...).Scan(&valid); err != nil || valid != 1 {
			t.Fatalf("time invariant query=%q valid=%d err=%v", check.query, valid, err)
		}
	}
}

func (rebuilder *disablingFaceClusterRebuilder) RebuildFaceClusters(context.Context, string, int64, string, int64, face.ClusterProfile, time.Time) error {
	_, err := rebuilder.store.UpdateFaceLibrarySettings(context.Background(), rebuilder.libraryID, false, 1, rebuilder.now)
	return err
}

func (rebuilder *cancellingFaceClusterRebuilder) RebuildFaceClusters(ctx context.Context, generationID string, libraryID int64, jobID string, claimedRevision int64, profile face.ClusterProfile, updatedAt time.Time) error {
	var operationRevision int64
	if err := rebuilder.store.db.QueryRowContext(ctx, `SELECT revision FROM ai_model_operations WHERE id=?`, rebuilder.operationID).Scan(&operationRevision); err != nil {
		return err
	}
	if _, err := rebuilder.store.CancelFaceJobOperation(ctx, rebuilder.operationID, operationRevision, rebuilder.now); err != nil {
		return err
	}
	return rebuilder.store.RebuildFaceClusters(ctx, generationID, libraryID, jobID, claimedRevision, profile, updatedAt)
}

func (analyzer *failingFaceAnalyzer) Analyze(context.Context, int64, int64, string) ([]face.Observation, error) {
	analyzer.calls++
	return nil, errors.New("source offline")
}

func (analyzer *unavailableFaceAnalyzer) Analyze(context.Context, int64, int64, string) ([]face.Observation, error) {
	analyzer.calls++
	return nil, face.ErrRuntimeUnavailable
}

func (analyzer *offlineAfterFirstFaceAnalyzer) Analyze(_ context.Context, _, _ int64, source string) ([]face.Observation, error) {
	analyzer.calls++
	if analyzer.calls == 1 {
		if _, err := analyzer.store.db.Exec(`UPDATE libraries SET status='offline',revision=revision+1 WHERE id=?`, analyzer.libraryID); err != nil {
			return nil, err
		}
	}
	return []face.Observation{{Box: face.Box{Width: .2, Height: .2}, Detection: 1, Quality: 1, SourceFingerprint: source, Embedding: []float32{1, 0}}}, nil
}

func (analyzer *unavailableAfterFirstFaceAnalyzer) Analyze(_ context.Context, _, _ int64, source string) ([]face.Observation, error) {
	analyzer.calls++
	if analyzer.calls == 1 {
		if _, err := analyzer.store.db.Exec(`UPDATE face_library_settings SET state='awaiting_model',revision=revision+1,updated_at_ms=? WHERE library_id=?`, analyzer.now.UnixMilli(), analyzer.libraryID); err != nil {
			return nil, err
		}
	}
	return []face.Observation{{Box: face.Box{Width: .2, Height: .2}, Detection: 1, Quality: 1, SourceFingerprint: source, Embedding: []float32{1, 0}}}, nil
}

func (a *scriptedFaceAnalyzer) Analyze(_ context.Context, _, _ int64, source string) ([]face.Observation, error) {
	a.calls++
	switch a.calls {
	case 1:
		return []face.Observation{
			{Box: face.Box{X: .1, Y: .1, Width: .2, Height: .2}, Detection: .99, Quality: .9, SourceFingerprint: source, Embedding: []float32{1, 0}},
			{Box: face.Box{X: .5, Y: .1, Width: .2, Height: .2}, Detection: .98, Quality: .8, SourceFingerprint: source, Embedding: []float32{.999, .001}},
		}, nil
	case 2:
		return nil, nil
	default:
		return nil, fmt.Errorf("synthetic runtime failure")
	}
}

func TestFaceJobAdmissionCoalescesAndRecoversWithoutPurgingReliableRows(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	now := time.Date(2026, 9, 1, 3, 0, 0, 0, time.UTC)
	seedFaceGeneration(t, store, "face_generation_jobs", 2, "active", now)
	seedFaceReadySettings(t, store, libraryID, "face_generation_jobs", now)
	waker := &faceClearWaker{}
	service, err := face.NewJobService(store, store, waker, func() time.Time { return now }, faceClearIDs())
	if err != nil {
		t.Fatal(err)
	}
	first, err := service.Request(context.Background(), libraryID, "face_generation_jobs", face.JobMissing, "face-job-key-001")
	if err != nil || !first.Created || waker.count != 1 {
		t.Fatalf("first=%+v wake=%d err=%v", first, waker.count, err)
	}
	replayed, err := service.Request(context.Background(), libraryID, "face_generation_jobs", face.JobMissing, "face-job-key-001")
	if err != nil || !replayed.Replayed {
		t.Fatalf("replayed=%+v err=%v", replayed, err)
	}
	coalesced, err := service.Request(context.Background(), libraryID, "face_generation_jobs", face.JobMissing, "face-job-key-002")
	if err != nil || !coalesced.Coalesced || coalesced.Job.ID != first.Job.ID {
		t.Fatalf("coalesced=%+v err=%v", coalesced, err)
	}
	if _, err := service.Request(context.Background(), libraryID, "face_generation_jobs", face.JobAll, "face-job-key-001"); !errors.Is(err, face.ErrFaceJobConflict) {
		t.Fatalf("idempotency conflict=%v", err)
	}
	claimed, found, err := store.ClaimFaceJob(context.Background(), now.Add(time.Minute), time.Minute)
	if err != nil || !found {
		t.Fatalf("claimed=%+v found=%v err=%v", claimed, found, err)
	}
	summary, err := store.RecoverExpiredFaceJobs(context.Background(), now.Add(3*time.Minute))
	if err != nil || summary.Requeued != 1 || summary.Interrupted != 0 {
		t.Fatalf("summary=%+v err=%v", summary, err)
	}
	var assets int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM assets WHERE library_id=?`, libraryID).Scan(&assets); err != nil || assets == 0 {
		t.Fatalf("assets=%d err=%v", assets, err)
	}
	for attempt := 2; attempt <= 3; attempt++ {
		claimed, found, err = store.ClaimFaceJob(context.Background(), now.Add(time.Duration(attempt*3)*time.Minute), time.Minute)
		if err != nil || !found {
			t.Fatalf("attempt=%d claimed=%+v found=%v err=%v", attempt, claimed, found, err)
		}
		summary, err = store.RecoverExpiredFaceJobs(context.Background(), now.Add(time.Duration(attempt*3+2)*time.Minute))
		if err != nil {
			t.Fatal(err)
		}
	}
	if summary.Interrupted != 1 || summary.Requeued != 0 {
		t.Fatalf("terminal summary=%+v", summary)
	}
	settings, err := store.GetFaceLibrarySettings(context.Background(), libraryID)
	if err != nil || settings.State != "degraded" || !settings.Enabled {
		t.Fatalf("settings=%+v err=%v", settings, err)
	}
}

func TestQueuedFaceJobCancellationPreservesExistingObservations(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	assetID := catalogAssetID(t, store, "photo-10.jpg")
	now := time.Date(2026, 9, 1, 4, 0, 0, 0, time.UTC)
	seedFaceGeneration(t, store, "face_generation_cancel", 2, "active", now)
	seedFaceReadySettings(t, store, libraryID, "face_generation_cancel", now)
	vector, _ := face.EncodeEmbedding([]float32{1, 0}, 2)
	if err := store.ReplaceFaceObservations(context.Background(), face.ObservationBatch{GenerationID: "face_generation_cancel", LibraryID: libraryID, AssetID: assetID, UpdatedAt: now, Items: []face.ObservationItem{{ID: "face_cancel_001", Box: face.Box{Width: .2, Height: .2}, Detection: 1, Quality: 1, SourceFingerprint: "source-v1", Vector: vector, CreatedAt: now}}}); err != nil {
		t.Fatal(err)
	}
	service, _ := face.NewJobService(store, store, &faceClearWaker{}, func() time.Time { return now }, faceClearIDs())
	requested, err := service.Request(context.Background(), libraryID, "face_generation_cancel", face.JobAll, "face-job-cancel-001")
	if err != nil {
		t.Fatal(err)
	}
	cancelled, err := service.Cancel(context.Background(), requested.Job.OperationID, 1)
	if err != nil || cancelled.State != "cancelled" {
		t.Fatalf("cancelled=%+v err=%v", cancelled, err)
	}
	var observations int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM face_observations WHERE id='face_cancel_001'`).Scan(&observations); err != nil || observations != 1 {
		t.Fatalf("observations=%d err=%v", observations, err)
	}
}

func TestFaceJobAdmissionDistinguishesOfflineFromModelUnavailable(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	now := time.Date(2026, 9, 2, 13, 0, 0, 0, time.UTC)
	seedFaceGeneration(t, store, "face_generation_admission_offline", 2, "active", now)
	seedFaceReadySettings(t, store, libraryID, "face_generation_admission_offline", now)
	if _, err := store.db.Exec(`UPDATE libraries SET status='offline',revision=revision+1 WHERE id=?`, libraryID); err != nil {
		t.Fatal(err)
	}
	service, _ := face.NewJobService(store, store, &faceClearWaker{}, func() time.Time { return now }, faceClearIDs())
	if _, err := service.Request(context.Background(), libraryID, "face_generation_admission_offline", face.JobMissing, "face-job-admission-offline-001"); !errors.Is(err, face.ErrFaceLibraryOffline) {
		t.Fatalf("expected offline error, got %v", err)
	}
	var jobs, operations int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM face_analysis_jobs`).Scan(&jobs); err != nil {
		t.Fatal(err)
	}
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM ai_model_operations WHERE kind IN ('face_missing','face_rebuild')`).Scan(&operations); err != nil {
		t.Fatal(err)
	}
	if jobs != 0 || operations != 0 {
		t.Fatalf("offline admission persisted jobs=%d operations=%d", jobs, operations)
	}
}

func TestFaceJobAdmissionRejectsAwaitingModelSettingsState(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	now := time.Date(2026, 9, 2, 13, 30, 0, 0, time.UTC)
	const generation = "face_generation_admission_not_ready"
	seedFaceGeneration(t, store, generation, 2, "active", now)
	seedFaceReadySettings(t, store, libraryID, generation, now)
	if _, err := store.db.Exec(`UPDATE face_library_settings SET state='awaiting_model',revision=revision+1 WHERE library_id=?`, libraryID); err != nil {
		t.Fatal(err)
	}
	service, _ := face.NewJobService(store, store, &faceClearWaker{}, func() time.Time { return now }, faceClearIDs())
	if _, err := service.Request(context.Background(), libraryID, generation, face.JobMissing, "face-job-admission-not-ready-001"); !errors.Is(err, face.ErrFaceModelUnavailable) {
		t.Fatalf("expected model-unavailable error, got %v", err)
	}
	var jobs, operations int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM face_analysis_jobs`).Scan(&jobs); err != nil {
		t.Fatal(err)
	}
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM ai_model_operations WHERE kind IN ('face_missing','face_rebuild')`).Scan(&operations); err != nil {
		t.Fatal(err)
	}
	if jobs != 0 || operations != 0 {
		t.Fatalf("not-ready admission persisted jobs=%d operations=%d", jobs, operations)
	}
}

func TestFaceJobTerminalDoesNotOverwriteNewerSettingsState(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	now := time.Date(2026, 9, 2, 13, 45, 0, 0, time.UTC)
	const generation = "face_generation_terminal_settings"
	seedFaceGeneration(t, store, generation, 2, "active", now)
	seedFaceReadySettings(t, store, libraryID, generation, now)
	service, _ := face.NewJobService(store, store, &faceClearWaker{}, func() time.Time { return now }, faceClearIDs())
	if _, err := service.Request(context.Background(), libraryID, generation, face.JobAll, "face-job-terminal-settings-001"); err != nil {
		t.Fatal(err)
	}
	claimed, found, err := store.ClaimFaceJob(context.Background(), now.Add(time.Second), time.Minute)
	if err != nil || !found {
		t.Fatalf("claimed=%+v found=%v err=%v", claimed, found, err)
	}
	if _, err := store.db.Exec(`UPDATE face_library_settings SET state='awaiting_model',revision=revision+1,updated_at_ms=? WHERE library_id=?`, now.Add(2*time.Second).UnixMilli(), libraryID); err != nil {
		t.Fatal(err)
	}
	finished, err := store.FinishFaceJob(context.Background(), claimed, false, "model_unavailable", now.Add(3*time.Second))
	if err != nil || finished.State != "failed" {
		t.Fatalf("finished=%+v err=%v", finished, err)
	}
	settings, err := store.GetFaceLibrarySettings(context.Background(), libraryID)
	if err != nil || settings.State != "awaiting_model" || !settings.Enabled {
		t.Fatalf("settings=%+v err=%v", settings, err)
	}
}

func TestFaceJobProcessorCommitsMultipleZeroAndFailedResults(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	now := time.Date(2026, 9, 1, 5, 0, 0, 0, time.UTC)
	seedFaceGeneration(t, store, "face_generation_process", 2, "active", now)
	seedFaceReadySettings(t, store, libraryID, "face_generation_process", now)
	service, _ := face.NewJobService(store, store, &faceClearWaker{}, func() time.Time { return now }, faceClearIDs())
	requested, err := service.Request(context.Background(), libraryID, "face_generation_process", face.JobAll, "face-job-process-001")
	if err != nil || !requested.Created {
		t.Fatalf("requested=%+v err=%v", requested, err)
	}
	claimed, found, err := store.ClaimFaceJob(context.Background(), now.Add(time.Second), time.Minute)
	if err != nil || !found {
		t.Fatalf("claimed=%+v found=%v err=%v", claimed, found, err)
	}
	analyzer := &scriptedFaceAnalyzer{}
	processor, err := face.NewJobProcessor(store, store, analyzer, store, face.ClusterProfile{CoreSimilarity: .9, EdgeSimilarity: .8, MinCoreSize: 2}, func() time.Time { return now.Add(2 * time.Second) }, 2)
	if err != nil {
		t.Fatal(err)
	}
	if err := processor.Process(context.Background(), claimed); err != nil {
		t.Fatal(err)
	}
	progress, found, err := store.GetFaceJobProgress(context.Background(), "face_generation_process", libraryID)
	if err != nil || !found || progress.Eligible != 3 || progress.Completed != 2 || progress.Failed != 1 || progress.Stale != 0 {
		t.Fatalf("progress=%+v found=%v err=%v", progress, found, err)
	}
	var results, observations, clusters int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM face_asset_results WHERE library_id=?`, libraryID).Scan(&results); err != nil {
		t.Fatal(err)
	}
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM face_observations WHERE library_id=?`, libraryID).Scan(&observations); err != nil {
		t.Fatal(err)
	}
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM face_clusters WHERE library_id=?`, libraryID).Scan(&clusters); err != nil {
		t.Fatal(err)
	}
	if results != 2 || observations != 2 || clusters != 1 {
		t.Fatalf("results=%d observations=%d clusters=%d", results, observations, clusters)
	}
	stored, found, err := store.FindFaceJob(context.Background(), faceDigestForTest("face-job-process-001"))
	if err != nil || !found || stored.Job.State != "succeeded" {
		t.Fatalf("stored=%+v found=%v err=%v", stored, found, err)
	}
	settings, err := store.GetFaceLibrarySettings(context.Background(), libraryID)
	if err != nil || settings.State != "degraded" {
		t.Fatalf("settings=%+v err=%v", settings, err)
	}
	counts, err := store.CountFaceJobCandidates(context.Background(), libraryID, "face_generation_process", face.JobMissing)
	if err != nil || counts.Pending != 1 {
		t.Fatalf("counts=%+v err=%v", counts, err)
	}
}

func TestFaceJobRuntimeUnavailableFailsClosedBeforeAdvancingProgress(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	now := time.Date(2026, 9, 2, 5, 30, 0, 0, time.UTC)
	const generation = "face_generation_runtime_unavailable"
	seedFaceGeneration(t, store, generation, 2, "active", now)
	seedFaceReadySettings(t, store, libraryID, generation, now)
	service, _ := face.NewJobService(store, store, &faceClearWaker{}, func() time.Time { return now }, faceClearIDs())
	if _, err := service.Request(context.Background(), libraryID, generation, face.JobAll, "face-job-runtime-unavailable-001"); err != nil {
		t.Fatal(err)
	}
	claimed, found, err := store.ClaimFaceJob(context.Background(), now.Add(time.Second), time.Minute)
	if err != nil || !found {
		t.Fatalf("claimed=%+v found=%v err=%v", claimed, found, err)
	}
	analyzer := &unavailableFaceAnalyzer{}
	processor, err := face.NewJobProcessor(store, store, analyzer, store, face.ClusterProfile{CoreSimilarity: .9, EdgeSimilarity: .8, MinCoreSize: 2}, func() time.Time { return now.Add(2 * time.Second) }, 3)
	if err != nil {
		t.Fatal(err)
	}
	if err := processor.Process(context.Background(), claimed); !errors.Is(err, face.ErrRuntimeUnavailable) {
		t.Fatalf("process error=%v", err)
	}
	stored, found, err := store.FindFaceJob(context.Background(), faceDigestForTest("face-job-runtime-unavailable-001"))
	if err != nil || !found || stored.Job.State != "failed" || stored.Job.ErrorCode != "model_unavailable" || analyzer.calls != 1 {
		t.Fatalf("stored=%+v found=%v calls=%d err=%v", stored, found, analyzer.calls, err)
	}
	progress, found, err := store.GetFaceJobProgress(context.Background(), generation, libraryID)
	if err != nil || !found || progress.Completed != 0 || progress.Failed != 0 || progress.Stale != 0 || progress.CheckpointID != 0 {
		t.Fatalf("progress=%+v found=%v err=%v", progress, found, err)
	}
	settings, err := store.GetFaceLibrarySettings(context.Background(), libraryID)
	if err != nil || !settings.Enabled || settings.State != "awaiting_model" {
		t.Fatalf("settings=%+v err=%v", settings, err)
	}
	var builds int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM face_cluster_builds WHERE library_id=?`, libraryID).Scan(&builds); err != nil || builds != 0 {
		t.Fatalf("cluster builds=%d err=%v", builds, err)
	}
}

func TestFaceJobProgressRecordsSourceChangeWithoutReplacingReliableRows(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	now := time.Date(2026, 9, 1, 6, 0, 0, 0, time.UTC)
	seedFaceGeneration(t, store, "face_generation_stale", 2, "active", now)
	seedFaceReadySettings(t, store, libraryID, "face_generation_stale", now)
	service, _ := face.NewJobService(store, store, &faceClearWaker{}, func() time.Time { return now }, faceClearIDs())
	if _, err := service.Request(context.Background(), libraryID, "face_generation_stale", face.JobAll, "face-job-stale-001"); err != nil {
		t.Fatal(err)
	}
	claimed, found, err := store.ClaimFaceJob(context.Background(), now.Add(time.Second), time.Minute)
	if err != nil || !found {
		t.Fatalf("claimed=%+v found=%v err=%v", claimed, found, err)
	}
	progress, found, err := store.GetFaceJobProgress(context.Background(), claimed.GenerationID, libraryID)
	if err != nil || !found {
		t.Fatal(err)
	}
	page, err := store.ListFaceJobCandidates(context.Background(), libraryID, claimed.GenerationID, face.JobAll, 0, 1)
	if err != nil || len(page.Items) != 1 {
		t.Fatalf("page=%+v err=%v", page, err)
	}
	candidate := page.Items[0]
	if _, err := store.db.Exec(`UPDATE assets SET source_fingerprint='v1:999:999' WHERE library_id=? AND id=?`, libraryID, candidate.AssetID); err != nil {
		t.Fatal(err)
	}
	updated, err := store.CommitFaceJobProgress(context.Background(), face.JobProgressCommit{
		JobID: claimed.ID, ClaimedRevision: claimed.ClaimedRevision,
		ExpectedProgressRevision: progress.Revision, ExpectedCheckpointID: progress.CheckpointID,
		NextCheckpointID: candidate.AssetID, SourceFingerprint: candidate.SourceFingerprint,
		Batch:      face.ObservationBatch{GenerationID: claimed.GenerationID, LibraryID: libraryID, AssetID: candidate.AssetID, UpdatedAt: now.Add(2 * time.Second)},
		StaleCount: 1, UpdatedAt: now.Add(2 * time.Second),
	})
	if err != nil || updated.Stale != 1 || updated.Completed != 0 {
		t.Fatalf("updated=%+v err=%v", updated, err)
	}
	var results int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM face_asset_results WHERE library_id=? AND asset_id=?`, libraryID, candidate.AssetID).Scan(&results); err != nil || results != 0 {
		t.Fatalf("results=%d err=%v", results, err)
	}
}

func TestFaceJobSourceOfflineStopsNewAnalysisAndPreservesLastReliableObservations(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	assetID := catalogAssetID(t, store, "photo-10.jpg")
	now := time.Date(2026, 9, 2, 14, 0, 0, 0, time.UTC)
	seedFaceGeneration(t, store, "face_generation_offline", 2, "active", now)
	seedFaceReadySettings(t, store, libraryID, "face_generation_offline", now)
	vector, _ := face.EncodeEmbedding([]float32{1, 0}, 2)
	if err := store.ReplaceFaceObservations(context.Background(), face.ObservationBatch{GenerationID: "face_generation_offline", LibraryID: libraryID, AssetID: assetID, UpdatedAt: now, Items: []face.ObservationItem{{ID: "face_offline_0001", Box: face.Box{Width: .2, Height: .2}, Detection: 1, Quality: 1, SourceFingerprint: "source-v1", Vector: vector, CreatedAt: now}}}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(`INSERT INTO face_asset_results(generation_id,library_id,asset_id,source_fingerprint,face_count,updated_at_ms) VALUES('face_generation_offline',?,?, 'source-v1',1,?)`, libraryID, assetID, now.UnixMilli()); err != nil {
		t.Fatal(err)
	}
	service, _ := face.NewJobService(store, store, &faceClearWaker{}, func() time.Time { return now }, faceClearIDs())
	if _, err := service.Request(context.Background(), libraryID, "face_generation_offline", face.JobAll, "face-job-offline-001"); err != nil {
		t.Fatal(err)
	}
	claimed, found, err := store.ClaimFaceJob(context.Background(), now.Add(time.Second), time.Minute)
	if err != nil || !found {
		t.Fatalf("claimed=%+v found=%v err=%v", claimed, found, err)
	}
	if _, err := store.db.Exec(`UPDATE libraries SET status='offline',revision=revision+1 WHERE id=?`, libraryID); err != nil {
		t.Fatal(err)
	}
	analyzer := &failingFaceAnalyzer{}
	processor, err := face.NewJobProcessor(store, store, analyzer, store, face.ClusterProfile{CoreSimilarity: .9, EdgeSimilarity: .8, MinCoreSize: 2}, func() time.Time { return now.Add(2 * time.Second) }, 2)
	if err != nil {
		t.Fatal(err)
	}
	if err := processor.Process(context.Background(), claimed); err != nil {
		t.Fatal(err)
	}
	var observations, reliableResults int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM face_observations WHERE id='face_offline_0001'`).Scan(&observations); err != nil {
		t.Fatal(err)
	}
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM face_asset_results WHERE generation_id='face_generation_offline' AND library_id=? AND asset_id=?`, libraryID, assetID).Scan(&reliableResults); err != nil {
		t.Fatal(err)
	}
	if observations != 1 || reliableResults != 1 || analyzer.calls != 0 {
		t.Fatalf("observations=%d results=%d calls=%d", observations, reliableResults, analyzer.calls)
	}
	stored, found, err := store.FindFaceJob(context.Background(), faceDigestForTest("face-job-offline-001"))
	if err != nil || !found || stored.Job.State != "failed" || stored.Job.ErrorCode != "library_offline" || stored.Job.CompletedItems != 0 {
		t.Fatalf("stored=%+v found=%v err=%v", stored, found, err)
	}
	settings, err := store.GetFaceLibrarySettings(context.Background(), libraryID)
	if err != nil || settings.State != "degraded" || !settings.Enabled {
		t.Fatalf("settings=%+v err=%v", settings, err)
	}
}

func TestFaceJobStopsBetweenCandidatesWhenLibraryGoesOffline(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	now := time.Date(2026, 9, 2, 15, 0, 0, 0, time.UTC)
	seedFaceGeneration(t, store, "face_generation_mid_offline", 2, "active", now)
	seedFaceReadySettings(t, store, libraryID, "face_generation_mid_offline", now)
	service, _ := face.NewJobService(store, store, &faceClearWaker{}, func() time.Time { return now }, faceClearIDs())
	if _, err := service.Request(context.Background(), libraryID, "face_generation_mid_offline", face.JobAll, "face-job-mid-offline-001"); err != nil {
		t.Fatal(err)
	}
	claimed, found, err := store.ClaimFaceJob(context.Background(), now.Add(time.Second), time.Minute)
	if err != nil || !found {
		t.Fatalf("claimed=%+v found=%v err=%v", claimed, found, err)
	}
	analyzer := &offlineAfterFirstFaceAnalyzer{store: store, libraryID: libraryID}
	processor, err := face.NewJobProcessor(store, store, analyzer, store, face.ClusterProfile{CoreSimilarity: .9, EdgeSimilarity: .8, MinCoreSize: 2}, func() time.Time { return now.Add(2 * time.Second) }, 3)
	if err != nil {
		t.Fatal(err)
	}
	if err := processor.Process(context.Background(), claimed); err != nil {
		t.Fatal(err)
	}
	stored, found, err := store.FindFaceJob(context.Background(), faceDigestForTest("face-job-mid-offline-001"))
	if err != nil || !found || stored.Job.State != "failed" || stored.Job.ErrorCode != "library_offline" || stored.Job.CompletedItems != 1 || analyzer.calls != 1 {
		t.Fatalf("stored=%+v found=%v calls=%d err=%v", stored, found, analyzer.calls, err)
	}
	progress, found, err := store.GetFaceJobProgress(context.Background(), claimed.GenerationID, libraryID)
	if err != nil || !found || progress.Completed != 1 || progress.Failed != 0 || progress.Stale != 0 {
		t.Fatalf("progress=%+v found=%v err=%v", progress, found, err)
	}
}

func TestFaceJobStopsBetweenCandidatesWhenModelBecomesUnavailable(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	now := time.Date(2026, 9, 2, 15, 30, 0, 0, time.UTC)
	const generation = "face_generation_mid_model_unavailable"
	seedFaceGeneration(t, store, generation, 2, "active", now)
	seedFaceReadySettings(t, store, libraryID, generation, now)
	service, _ := face.NewJobService(store, store, &faceClearWaker{}, func() time.Time { return now }, faceClearIDs())
	if _, err := service.Request(context.Background(), libraryID, generation, face.JobAll, "face-job-mid-model-unavailable-001"); err != nil {
		t.Fatal(err)
	}
	claimed, found, err := store.ClaimFaceJob(context.Background(), now.Add(time.Second), time.Minute)
	if err != nil || !found {
		t.Fatalf("claimed=%+v found=%v err=%v", claimed, found, err)
	}
	analyzer := &unavailableAfterFirstFaceAnalyzer{store: store, libraryID: libraryID, now: now.Add(2 * time.Second)}
	processor, err := face.NewJobProcessor(store, store, analyzer, store, face.ClusterProfile{CoreSimilarity: .9, EdgeSimilarity: .8, MinCoreSize: 2}, func() time.Time { return now.Add(3 * time.Second) }, 3)
	if err != nil {
		t.Fatal(err)
	}
	if err := processor.Process(context.Background(), claimed); err != nil {
		t.Fatal(err)
	}
	stored, found, err := store.FindFaceJob(context.Background(), faceDigestForTest("face-job-mid-model-unavailable-001"))
	if err != nil || !found || stored.Job.State != "failed" || stored.Job.ErrorCode != "model_unavailable" || stored.Job.CompletedItems != 1 || analyzer.calls != 1 {
		t.Fatalf("stored=%+v found=%v calls=%d err=%v", stored, found, analyzer.calls, err)
	}
	settings, err := store.GetFaceLibrarySettings(context.Background(), libraryID)
	if err != nil || settings.State != "awaiting_model" || !settings.Enabled {
		t.Fatalf("settings=%+v err=%v", settings, err)
	}
}

func TestFaceJobProcessorHonorsRunningCancellationBeforeAnalysis(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	now := time.Date(2026, 9, 2, 16, 0, 0, 0, time.UTC)
	seedFaceGeneration(t, store, "face_generation_cancel_running", 2, "active", now)
	seedFaceReadySettings(t, store, libraryID, "face_generation_cancel_running", now)
	service, _ := face.NewJobService(store, store, &faceClearWaker{}, func() time.Time { return now }, faceClearIDs())
	requested, err := service.Request(context.Background(), libraryID, "face_generation_cancel_running", face.JobAll, "face-job-cancel-running-001")
	if err != nil {
		t.Fatal(err)
	}
	claimed, found, err := store.ClaimFaceJob(context.Background(), now.Add(time.Second), time.Minute)
	if err != nil || !found {
		t.Fatalf("claimed=%+v found=%v err=%v", claimed, found, err)
	}
	if _, err := service.Cancel(context.Background(), requested.Job.OperationID, claimed.OperationRevision); err != nil {
		t.Fatal(err)
	}
	analyzer := &failingFaceAnalyzer{}
	processor, err := face.NewJobProcessor(store, store, analyzer, store, face.ClusterProfile{CoreSimilarity: .9, EdgeSimilarity: .8, MinCoreSize: 2}, func() time.Time { return now.Add(2 * time.Second) }, 3)
	if err != nil {
		t.Fatal(err)
	}
	if err := processor.Process(context.Background(), claimed); err != nil {
		t.Fatal(err)
	}
	stored, found, err := store.FindFaceJob(context.Background(), faceDigestForTest("face-job-cancel-running-001"))
	if err != nil || !found || stored.Job.State != "cancelled" || stored.Job.ErrorCode != "cancelled" || analyzer.calls != 0 {
		t.Fatalf("stored=%+v found=%v calls=%d err=%v", stored, found, analyzer.calls, err)
	}
}

func TestFaceJobProcessorHonorsDisableDuringClusterBuild(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	now := time.Date(2026, 9, 2, 18, 0, 0, 0, time.UTC)
	const generation = "face_generation_disable_cluster"
	seedFaceGeneration(t, store, generation, 2, "active", now)
	seedFaceReadySettings(t, store, libraryID, generation, now)
	service, _ := face.NewJobService(store, store, &faceClearWaker{}, func() time.Time { return now }, faceClearIDs())
	if _, err := service.Request(context.Background(), libraryID, generation, face.JobAll, "face-job-disable-cluster-001"); err != nil {
		t.Fatal(err)
	}
	claimed, found, err := store.ClaimFaceJob(context.Background(), now.Add(time.Second), time.Minute)
	if err != nil || !found {
		t.Fatalf("claimed=%+v found=%v err=%v", claimed, found, err)
	}
	rebuilder := &disablingFaceClusterRebuilder{store: store, libraryID: libraryID, now: now.Add(3 * time.Second)}
	processor, err := face.NewJobProcessor(store, store, &scriptedFaceAnalyzer{}, rebuilder, face.ClusterProfile{CoreSimilarity: .9, EdgeSimilarity: .8, MinCoreSize: 2}, func() time.Time { return now.Add(2 * time.Second) }, 3)
	if err != nil {
		t.Fatal(err)
	}
	if err := processor.Process(context.Background(), claimed); err != nil {
		t.Fatal(err)
	}
	stored, found, err := store.FindFaceJob(context.Background(), faceDigestForTest("face-job-disable-cluster-001"))
	if err != nil || !found || stored.Job.State != "cancelled" || stored.Job.ErrorCode != "cancelled" {
		t.Fatalf("stored=%+v found=%v err=%v", stored, found, err)
	}
	settings, err := store.GetFaceLibrarySettings(context.Background(), libraryID)
	if err != nil || settings.Enabled || settings.State != "disabled" {
		t.Fatalf("settings=%+v err=%v", settings, err)
	}
}

func TestFaceJobProcessorCancellationPreventsClusterActivation(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	now := time.Date(2026, 9, 2, 18, 30, 0, 0, time.UTC)
	const generation = "face_generation_cancel_cluster"
	seedFaceGeneration(t, store, generation, 2, "active", now)
	seedFaceReadySettings(t, store, libraryID, generation, now)
	service, _ := face.NewJobService(store, store, &faceClearWaker{}, func() time.Time { return now }, faceClearIDs())
	requested, err := service.Request(context.Background(), libraryID, generation, face.JobAll, "face-job-cancel-cluster-001")
	if err != nil {
		t.Fatal(err)
	}
	claimed, found, err := store.ClaimFaceJob(context.Background(), now.Add(time.Second), time.Minute)
	if err != nil || !found {
		t.Fatalf("claimed=%+v found=%v err=%v", claimed, found, err)
	}
	rebuilder := &cancellingFaceClusterRebuilder{store: store, operationID: requested.Job.OperationID, now: now.Add(3 * time.Second)}
	processor, err := face.NewJobProcessor(store, store, &scriptedFaceAnalyzer{}, rebuilder, face.ClusterProfile{CoreSimilarity: .9, EdgeSimilarity: .8, MinCoreSize: 2}, func() time.Time { return now.Add(2 * time.Second) }, 3)
	if err != nil {
		t.Fatal(err)
	}
	if err := processor.Process(context.Background(), claimed); err != nil {
		t.Fatal(err)
	}
	stored, found, err := store.FindFaceJob(context.Background(), faceDigestForTest("face-job-cancel-cluster-001"))
	if err != nil || !found || stored.Job.State != "cancelled" || stored.Job.ErrorCode != "cancelled" {
		t.Fatalf("stored=%+v found=%v err=%v", stored, found, err)
	}
	var activeBuilds int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM face_cluster_builds WHERE library_id=? AND state='active'`, libraryID).Scan(&activeBuilds); err != nil {
		t.Fatal(err)
	}
	if activeBuilds != 0 {
		t.Fatalf("cancelled cluster build published %d active builds", activeBuilds)
	}
}

func TestFaceClusterRebuildRejectsStaleWorkerClaim(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	now := time.Date(2026, 9, 2, 19, 0, 0, 0, time.UTC)
	const generation = "face_generation_stale_cluster_claim"
	seedFaceGeneration(t, store, generation, 2, "active", now)
	seedFaceReadySettings(t, store, libraryID, generation, now)
	service, _ := face.NewJobService(store, store, &faceClearWaker{}, func() time.Time { return now }, faceClearIDs())
	if _, err := service.Request(context.Background(), libraryID, generation, face.JobAll, "face-job-stale-cluster-claim-001"); err != nil {
		t.Fatal(err)
	}
	claimed, found, err := store.ClaimFaceJob(context.Background(), now.Add(time.Second), time.Minute)
	if err != nil || !found {
		t.Fatalf("claimed=%+v found=%v err=%v", claimed, found, err)
	}
	err = store.RebuildFaceClusters(context.Background(), generation, libraryID, claimed.ID, claimed.ClaimedRevision-1, face.ClusterProfile{CoreSimilarity: .9, EdgeSimilarity: .8, MinCoreSize: 2}, now.Add(2*time.Second))
	if !errors.Is(err, face.ErrFaceJobConflict) {
		t.Fatalf("expected stale claim rejection, got %v", err)
	}
	var builds int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM face_cluster_builds WHERE library_id=?`, libraryID).Scan(&builds); err != nil {
		t.Fatal(err)
	}
	if builds != 0 {
		t.Fatalf("stale worker created %d cluster builds", builds)
	}
}

func faceDigestForTest(key string) string {
	sum := sha256.Sum256([]byte("foliopath:face-job-key:v1\x00" + key))
	return hex.EncodeToString(sum[:])
}
