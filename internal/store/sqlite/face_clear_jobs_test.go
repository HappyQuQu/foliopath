package sqlite

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/face"
)

type faceClearWaker struct{ count int }

func (w *faceClearWaker) Wake() { w.count++ }
func faceClearIDs() func(string) (string, error) {
	next := 0
	return func(prefix string) (string, error) { next++; return fmt.Sprintf("%s_test_%03d", prefix, next), nil }
}

func TestFaceClearAdmissionCreatesMissingSettingsWithoutMaskingLibraryState(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	now := time.Date(2026, 9, 2, 20, 0, 0, 0, time.UTC)
	service, err := face.NewClearService(store, &faceClearWaker{}, func() time.Time { return now }, faceClearIDs())
	if err != nil {
		t.Fatal(err)
	}
	requested, err := service.RequestDerived(context.Background(), libraryID, 1, "face-clear-missing-settings")
	if err != nil || !requested.Created {
		t.Fatalf("requested=%+v err=%v", requested, err)
	}
	settings, err := store.GetFaceLibrarySettings(context.Background(), libraryID)
	if err != nil || settings.Enabled || settings.State != "clearing" || settings.Revision != 2 {
		t.Fatalf("settings=%+v err=%v", settings, err)
	}
}

func TestFaceClearAdmissionClampsSettingsTimeAcrossClockRollback(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	var libraryCreatedAt int64
	if err := store.db.QueryRowContext(context.Background(), `SELECT created_at_ms FROM libraries WHERE id=?`, libraryID).Scan(&libraryCreatedAt); err != nil {
		t.Fatal(err)
	}
	now := time.UnixMilli(libraryCreatedAt).Add(-time.Minute)
	service, err := face.NewClearService(store, &faceClearWaker{}, func() time.Time { return now }, faceClearIDs())
	if err != nil {
		t.Fatal(err)
	}
	requested, err := service.RequestDerived(context.Background(), libraryID, 1, "face-clear-clock-rollback")
	if err != nil || !requested.Created {
		t.Fatalf("requested=%+v err=%v", requested, err)
	}
	var createdAt, updatedAt int64
	if err := store.db.QueryRowContext(context.Background(), `SELECT created_at_ms,updated_at_ms FROM face_library_settings WHERE library_id=?`, libraryID).Scan(&createdAt, &updatedAt); err != nil {
		t.Fatal(err)
	}
	if createdAt != libraryCreatedAt || updatedAt < createdAt {
		t.Fatalf("settings created_at_ms=%d updated_at_ms=%d library_created_at_ms=%d", createdAt, updatedAt, libraryCreatedAt)
	}
}

func TestFaceClearLifecycleClampsTimeAcrossClockRollback(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	var libraryCreatedAt int64
	if err := store.db.QueryRowContext(context.Background(), `SELECT created_at_ms FROM libraries WHERE id=?`, libraryID).Scan(&libraryCreatedAt); err != nil {
		t.Fatal(err)
	}
	now := time.UnixMilli(libraryCreatedAt).Add(-time.Minute)
	service, err := face.NewClearService(store, &faceClearWaker{}, func() time.Time { return now }, faceClearIDs())
	if err != nil {
		t.Fatal(err)
	}
	requested, err := service.RequestDerived(context.Background(), libraryID, 1, "face-clear-lifecycle-clock-rollback")
	if err != nil || !requested.Created {
		t.Fatalf("requested=%+v err=%v", requested, err)
	}
	now = now.Add(-time.Minute)
	claimed, found, err := store.ClaimFaceClear(context.Background(), now, time.Minute)
	if err != nil || !found || claimed.State != "running" {
		t.Fatalf("claimed=%+v found=%v err=%v", claimed, found, err)
	}
	var jobCreatedAt, jobUpdatedAt, leaseExpiresAt int64
	if err := store.db.QueryRowContext(context.Background(), `SELECT created_at_ms,updated_at_ms,lease_expires_ms FROM face_clear_jobs WHERE id=?`, claimed.ID).Scan(&jobCreatedAt, &jobUpdatedAt, &leaseExpiresAt); err != nil {
		t.Fatal(err)
	}
	if jobUpdatedAt < jobCreatedAt || leaseExpiresAt < jobCreatedAt+time.Minute.Milliseconds() {
		t.Fatalf("claimed job created=%d updated=%d lease=%d", jobCreatedAt, jobUpdatedAt, leaseExpiresAt)
	}
	now = now.Add(-time.Minute)
	cancelled, err := store.RefreshFaceClearLease(context.Background(), claimed, now, time.Minute)
	if err != nil || cancelled {
		t.Fatalf("refresh cancelled=%v err=%v", cancelled, err)
	}
	deleted, done, err := store.DeleteFaceClearBatch(context.Background(), claimed, 1, now)
	if err != nil || deleted != 0 || !done {
		t.Fatalf("delete deleted=%d done=%v err=%v", deleted, done, err)
	}
	finished, err := store.FinishFaceClear(context.Background(), claimed, true, "", now)
	if err != nil || finished.State != "succeeded" {
		t.Fatalf("finished=%+v err=%v", finished, err)
	}
	assertFaceClearTimeInvariants(t, store, libraryID, claimed.ID, claimed.OperationID)

	requested, err = service.RequestDerived(context.Background(), libraryID, 3, "face-clear-cancel-clock-rollback")
	if err != nil || !requested.Created {
		t.Fatalf("cancel request=%+v err=%v", requested, err)
	}
	now = now.Add(-time.Minute)
	cancelledJob, err := service.Cancel(context.Background(), requested.Job.OperationID, 1)
	if err != nil || cancelledJob.State != "cancelled" {
		t.Fatalf("cancelled=%+v err=%v", cancelledJob, err)
	}
	assertFaceClearTimeInvariants(t, store, libraryID, requested.Job.ID, requested.Job.OperationID)
}

func assertFaceClearTimeInvariants(t *testing.T, store *Store, libraryID int64, jobID, operationID string) {
	t.Helper()
	for _, check := range []struct {
		query string
		args  []any
	}{
		{`SELECT 1 FROM face_library_settings WHERE library_id=? AND updated_at_ms>=created_at_ms`, []any{libraryID}},
		{`SELECT 1 FROM face_clear_jobs WHERE id=? AND updated_at_ms>=created_at_ms`, []any{jobID}},
		{`SELECT 1 FROM ai_model_operations WHERE id=? AND updated_at_ms>=created_at_ms AND (finished_at_ms IS NULL OR finished_at_ms>=created_at_ms)`, []any{operationID}},
	} {
		var valid int
		if err := store.db.QueryRowContext(context.Background(), check.query, check.args...).Scan(&valid); err != nil || valid != 1 {
			t.Fatalf("time invariant query=%q valid=%d err=%v", check.query, valid, err)
		}
	}
}

func TestCancelFaceClearDoesNotMisclassifyStorageFailure(t *testing.T) {
	store, _ := openTestStore(t)
	if err := store.db.Close(); err != nil {
		t.Fatal(err)
	}
	_, err := store.CancelFaceClearOperation(context.Background(), "face_clear_operation_storage_failure", 1, time.Now().UTC())
	if err == nil || errors.Is(err, face.ErrFaceClearConflict) {
		t.Fatalf("storage failure=%v", err)
	}
}

func TestDerivedFaceClearPreservesManualStateAndOriginalAssetMetadata(t *testing.T) {
	originalPath := filepath.Join(t.TempDir(), "synthetic-original.jpg")
	if err := os.WriteFile(originalPath, []byte("synthetic immutable original"), 0o600); err != nil {
		t.Fatal(err)
	}
	fixedMTime := time.Date(2026, 8, 1, 12, 0, 0, 123, time.UTC)
	if err := os.Chtimes(originalPath, fixedMTime, fixedMTime); err != nil {
		t.Fatal(err)
	}
	beforeBytes, err := os.ReadFile(originalPath)
	if err != nil {
		t.Fatal(err)
	}
	beforeHash := sha256.Sum256(beforeBytes)
	beforeStat, err := os.Stat(originalPath)
	if err != nil {
		t.Fatal(err)
	}
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	assetID := catalogAssetID(t, store, "photo-10.jpg")
	now := time.Date(2026, 9, 1, 1, 0, 0, 0, time.UTC)
	seedFaceGeneration(t, store, "face_generation_clear", 2, "active", now)
	seedFaceReadySettings(t, store, libraryID, "face_generation_clear", now)
	vector, _ := face.EncodeEmbedding([]float32{1, 0}, 2)
	batch := face.ObservationBatch{GenerationID: "face_generation_clear", LibraryID: libraryID, AssetID: assetID, UpdatedAt: now, Items: []face.ObservationItem{{ID: "face_clear_0001", Box: face.Box{Width: .2, Height: .2}, Detection: 1, Quality: 1, SourceFingerprint: "source-v1", Vector: vector, CreatedAt: now}, {ID: "face_clear_0002", Box: face.Box{X: .3, Width: .2, Height: .2}, Detection: 1, Quality: 1, SourceFingerprint: "source-v1", Vector: vector, CreatedAt: now}}}
	if err := store.ReplaceFaceObservations(context.Background(), batch); err != nil {
		t.Fatal(err)
	}
	if err := store.ReplaceFaceClusters(context.Background(), face.ClusterBatch{GenerationID: batch.GenerationID, LibraryID: libraryID, UpdatedAt: now, Clusters: []face.Cluster{{ID: "face_clear_group", Role: "core", Members: []face.ClusterMember{{FaceID: "face_clear_0001", Role: "core", Confidence: 1}, {FaceID: "face_clear_0002", Role: "core", Confidence: 1}}}}}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreatePerson(context.Background(), face.CreatePersonCommand{ID: "person_clear_01", Name: "保留人物", CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AssignFace(context.Background(), face.AssignFaceCommand{EventID: "event_clear_001", RequestHash: testFaceRequestHash, AnchorID: "anchor_clear_001", FaceID: "face_clear_0001", PersonID: "person_clear_01", ExpectedFaceRevision: 1, ExpectedPersonRevision: 1, CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	var beforeFingerprint string
	var beforeMTime, size int64
	if err := store.db.QueryRow(`SELECT source_fingerprint,mtime_ns,size_bytes FROM assets WHERE library_id=? AND id=?`, libraryID, assetID).Scan(&beforeFingerprint, &beforeMTime, &size); err != nil {
		t.Fatal(err)
	}
	waker := &faceClearWaker{}
	service, err := face.NewClearService(store, waker, func() time.Time { return now.Add(time.Minute) }, faceClearIDs())
	if err != nil {
		t.Fatal(err)
	}
	requested, err := service.RequestDerived(context.Background(), libraryID, 1, "face-derived-clear-001")
	if err != nil || !requested.Created || waker.count != 1 {
		t.Fatalf("requested=%+v wake=%d err=%v", requested, waker.count, err)
	}
	replayed, err := service.RequestDerived(context.Background(), libraryID, 1, "face-derived-clear-001")
	if err != nil || !replayed.Replayed {
		t.Fatalf("replayed=%+v err=%v", replayed, err)
	}
	claimed, found, err := store.ClaimFaceClear(context.Background(), now.Add(2*time.Minute), time.Minute)
	if err != nil || !found {
		t.Fatalf("claimed=%+v found=%v err=%v", claimed, found, err)
	}
	processor, _ := face.NewClearProcessor(store, func() time.Time { return now.Add(3 * time.Minute) }, 1)
	if err := processor.Process(context.Background(), claimed); err != nil {
		t.Fatal(err)
	}
	var observations, clusters, people, anchors int
	checks := []struct {
		query  string
		args   []any
		target *int
	}{
		{`SELECT COUNT(*) FROM face_observations WHERE library_id=?`, []any{libraryID}, &observations},
		{`SELECT COUNT(*) FROM face_clusters WHERE library_id=?`, []any{libraryID}, &clusters},
		{`SELECT COUNT(*) FROM people WHERE id='person_clear_01'`, nil, &people},
		{`SELECT COUNT(*) FROM person_face_anchors WHERE id='anchor_clear_001' AND state='needs_review' AND current_face_id IS NULL`, nil, &anchors},
	}
	for _, check := range checks {
		if err := store.db.QueryRow(check.query, check.args...).Scan(check.target); err != nil {
			t.Fatal(err)
		}
	}
	if observations != 0 || clusters != 0 || people != 1 || anchors != 1 {
		t.Fatalf("observations=%d clusters=%d people=%d anchors=%d", observations, clusters, people, anchors)
	}
	var afterFingerprint string
	var afterMTime, afterSize int64
	if err := store.db.QueryRow(`SELECT source_fingerprint,mtime_ns,size_bytes FROM assets WHERE library_id=? AND id=?`, libraryID, assetID).Scan(&afterFingerprint, &afterMTime, &afterSize); err != nil {
		t.Fatal(err)
	}
	if beforeFingerprint != afterFingerprint || beforeMTime != afterMTime || size != afterSize {
		t.Fatal("derived clear changed asset truth")
	}
	afterBytes, err := os.ReadFile(originalPath)
	if err != nil {
		t.Fatal(err)
	}
	afterHash := sha256.Sum256(afterBytes)
	afterStat, err := os.Stat(originalPath)
	if err != nil {
		t.Fatal(err)
	}
	if beforeHash != afterHash || !beforeStat.ModTime().Equal(afterStat.ModTime()) {
		t.Fatal("face clear changed original media hash or mtime")
	}
}

func TestManualFaceClearRequiresExactImpactAndRetainsDerivedAndPeople(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	assetID := catalogAssetID(t, store, "photo-10.jpg")
	now := time.Date(2026, 9, 1, 2, 0, 0, 0, time.UTC)
	seedFaceGeneration(t, store, "face_generation_manual", 2, "active", now)
	seedFaceReadySettings(t, store, libraryID, "face_generation_manual", now)
	vector, _ := face.EncodeEmbedding([]float32{1, 0}, 2)
	if err := store.ReplaceFaceObservations(context.Background(), face.ObservationBatch{GenerationID: "face_generation_manual", LibraryID: libraryID, AssetID: assetID, UpdatedAt: now, Items: []face.ObservationItem{{ID: "face_manual_001", Box: face.Box{Width: .2, Height: .2}, Detection: 1, Quality: 1, SourceFingerprint: "source-v1", Vector: vector, CreatedAt: now}}}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreatePerson(context.Background(), face.CreatePersonCommand{ID: "person_manual_1", Name: "空人物保留", CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AssignFace(context.Background(), face.AssignFaceCommand{EventID: "event_manual_01", RequestHash: testFaceRequestHash, AnchorID: "anchor_manual_1", FaceID: "face_manual_001", PersonID: "person_manual_1", ExpectedFaceRevision: 1, ExpectedPersonRevision: 1, CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	service, _ := face.NewClearService(store, &faceClearWaker{}, func() time.Time { return now.Add(time.Minute) }, faceClearIDs())
	if _, err := service.RequestManual(context.Background(), libraryID, 1, "face-manual-clear-bad", face.ManualClearCounts{}); err == nil {
		t.Fatal("incorrect impact counts succeeded")
	}
	requested, err := service.RequestManual(context.Background(), libraryID, 1, "face-manual-clear-001", face.ManualClearCounts{People: 1, Assignments: 1, Constraints: 0})
	if err != nil || !requested.Created {
		t.Fatalf("requested=%+v err=%v", requested, err)
	}
	claimed, found, err := store.ClaimFaceClear(context.Background(), now.Add(2*time.Minute), time.Minute)
	if err != nil || !found {
		t.Fatalf("claimed=%+v found=%v err=%v", claimed, found, err)
	}
	processor, _ := face.NewClearProcessor(store, func() time.Time { return now.Add(3 * time.Minute) }, 1)
	if err := processor.Process(context.Background(), claimed); err != nil {
		t.Fatal(err)
	}
	var observations, people, anchors, audits int
	queries := []struct {
		q string
		v *int
	}{{`SELECT COUNT(*) FROM face_observations WHERE library_id=?`, &observations}, {`SELECT COUNT(*) FROM people WHERE id='person_manual_1'`, &people}, {`SELECT COUNT(*) FROM person_face_anchors WHERE library_id=?`, &anchors}, {`SELECT COUNT(*) FROM face_audit_events WHERE library_id=?`, &audits}}
	for _, item := range queries {
		if err := store.db.QueryRow(item.q, libraryID).Scan(item.v); err != nil {
			t.Fatal(err)
		}
	}
	if observations != 1 || people != 1 || anchors != 0 || audits != 0 {
		t.Fatalf("observations=%d people=%d anchors=%d audits=%d", observations, people, anchors, audits)
	}
}

func TestRunningFaceClearCancellationIsCooperativeAndFailClosed(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	now := time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)
	seedFaceGeneration(t, store, "face_generation_cancel_clear", 2, "active", now)
	seedFaceReadySettings(t, store, libraryID, "face_generation_cancel_clear", now)
	service, _ := face.NewClearService(store, &faceClearWaker{}, func() time.Time { return now.Add(time.Minute) }, faceClearIDs())
	requested, err := service.RequestDerived(context.Background(), libraryID, 1, "face-clear-cancel-001")
	if err != nil {
		t.Fatal(err)
	}
	claimed, found, err := store.ClaimFaceClear(context.Background(), now.Add(2*time.Minute), time.Minute)
	if err != nil || !found {
		t.Fatalf("claimed=%+v found=%v err=%v", claimed, found, err)
	}
	var operationRevision int64
	if err := store.db.QueryRow(`SELECT revision FROM ai_model_operations WHERE id=?`, requested.Job.OperationID).Scan(&operationRevision); err != nil {
		t.Fatal(err)
	}
	cancelling, err := service.Cancel(context.Background(), requested.Job.OperationID, operationRevision)
	if err != nil || cancelling.State != "cancelling" {
		t.Fatalf("job=%+v err=%v", cancelling, err)
	}
	processor, _ := face.NewClearProcessor(store, func() time.Time { return now.Add(3 * time.Minute) }, 1)
	if err := processor.Process(context.Background(), claimed); err != nil {
		t.Fatal(err)
	}
	stored, found, err := store.FindFaceClear(context.Background(), faceDigestForTestClear("face-clear-cancel-001"))
	if err != nil || !found || stored.Job.State != "cancelled" {
		t.Fatalf("stored=%+v found=%v err=%v", stored, found, err)
	}
	settings, err := store.GetFaceLibrarySettings(context.Background(), libraryID)
	if err != nil || settings.Enabled || settings.State != "disabled" {
		t.Fatalf("settings=%+v err=%v", settings, err)
	}
}

func TestExpiredFaceClearRequeuesWithoutLosingCommittedProgress(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	assetID := catalogAssetID(t, store, "photo-10.jpg")
	now := time.Date(2026, 9, 1, 9, 0, 0, 0, time.UTC)
	seedFaceGeneration(t, store, "face_generation_recover_clear", 2, "active", now)
	seedFaceReadySettings(t, store, libraryID, "face_generation_recover_clear", now)
	vector, _ := face.EncodeEmbedding([]float32{1, 0}, 2)
	if err := store.ReplaceFaceObservations(context.Background(), face.ObservationBatch{GenerationID: "face_generation_recover_clear", LibraryID: libraryID, AssetID: assetID, UpdatedAt: now, Items: []face.ObservationItem{{ID: "face_recover_clear_1", Box: face.Box{Width: .2, Height: .2}, Detection: 1, Quality: 1, SourceFingerprint: "source-v1", Vector: vector, CreatedAt: now}, {ID: "face_recover_clear_2", Box: face.Box{X: .3, Width: .2, Height: .2}, Detection: 1, Quality: 1, SourceFingerprint: "source-v1", Vector: vector, CreatedAt: now}}}); err != nil {
		t.Fatal(err)
	}
	if err := store.ReplaceFaceClusters(context.Background(), face.ClusterBatch{GenerationID: "face_generation_recover_clear", LibraryID: libraryID, UpdatedAt: now, Clusters: []face.Cluster{{ID: "face_recover_group", Role: "core", Members: []face.ClusterMember{{FaceID: "face_recover_clear_1", Role: "core", Confidence: 1}, {FaceID: "face_recover_clear_2", Role: "core", Confidence: 1}}}}}); err != nil {
		t.Fatal(err)
	}
	service, _ := face.NewClearService(store, &faceClearWaker{}, func() time.Time { return now }, faceClearIDs())
	if _, err := service.RequestDerived(context.Background(), libraryID, 1, "face-clear-recover-001"); err != nil {
		t.Fatal(err)
	}
	claimed, found, err := store.ClaimFaceClear(context.Background(), now.Add(time.Minute), time.Minute)
	if err != nil || !found {
		t.Fatal(err)
	}
	deleted, done, err := store.DeleteFaceClearBatch(context.Background(), claimed, 1, now.Add(90*time.Second))
	if err != nil || done || deleted != 1 {
		t.Fatalf("deleted=%d done=%v err=%v", deleted, done, err)
	}
	summary, err := store.RecoverExpiredFaceClears(context.Background(), now.Add(3*time.Minute))
	if err != nil || summary.Requeued != 1 {
		t.Fatalf("summary=%+v err=%v", summary, err)
	}
	retry, found, err := store.ClaimFaceClear(context.Background(), now.Add(4*time.Minute), time.Minute)
	if err != nil || !found {
		t.Fatal(err)
	}
	processor, _ := face.NewClearProcessor(store, func() time.Time { return now.Add(5 * time.Minute) }, 1)
	if err := processor.Process(context.Background(), retry); err != nil {
		t.Fatal(err)
	}
	if retry.AttemptCount != 2 {
		t.Fatalf("attempts=%d", retry.AttemptCount)
	}
	var state string
	var deletedTotal int64
	if err := store.db.QueryRow(`SELECT state,deleted_count FROM face_clear_jobs WHERE id=?`, retry.ID).Scan(&state, &deletedTotal); err != nil || state != "succeeded" || deletedTotal < 1 {
		t.Fatalf("state=%s deleted=%d err=%v", state, deletedTotal, err)
	}
}

func faceDigestForTestClear(key string) string {
	sum := sha256.Sum256([]byte("foliopath:face-clear-key:v1\x00" + key))
	return fmt.Sprintf("%x", sum[:])
}

func seedFaceReadySettings(t *testing.T, store *Store, libraryID int64, generation string, now time.Time) {
	t.Helper()
	if _, err := store.db.Exec(`INSERT INTO face_library_settings(library_id,enabled,state,active_generation_id,revision,coverage_revision,created_at_ms,updated_at_ms) VALUES(?,1,'ready',?,1,1,?,?)`, libraryID, generation, now.UnixMilli(), now.UnixMilli()); err != nil {
		t.Fatal(err)
	}
}
