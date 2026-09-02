package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
	"github.com/HappyQuQu/foliopath/internal/face"
)

func TestFaceSettingsRemainFailClosedWithoutActiveModelAndPreserveManualState(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	var libraryCreatedAt int64
	if err := store.db.QueryRow(`SELECT created_at_ms FROM libraries WHERE id=?`, libraryID).Scan(&libraryCreatedAt); err != nil {
		t.Fatal(err)
	}
	now := time.UnixMilli(libraryCreatedAt).UTC().Add(-time.Minute)
	settings, err := store.GetFaceLibrarySettings(context.Background(), libraryID)
	if err != nil || settings.Enabled || settings.Revision != 1 {
		t.Fatalf("settings=%+v err=%v", settings, err)
	}
	if _, err := store.UpdateFaceLibrarySettings(context.Background(), libraryID, true, 1, now); !errors.Is(err, face.ErrFaceModelUnavailable) {
		t.Fatalf("enable err=%v", err)
	}
	seedFaceGeneration(t, store, "face_generation_settings", 2, "active", now)
	enabled, err := store.UpdateFaceLibrarySettings(context.Background(), libraryID, true, 1, now)
	if err != nil || !enabled.Enabled || enabled.State != "building" || enabled.Revision != 2 || enabled.ActiveGenerationID != "face_generation_settings" {
		t.Fatalf("enabled=%+v err=%v", enabled, err)
	}
	var settingsUpdatedAt int64
	if err := store.db.QueryRow(`SELECT updated_at_ms FROM face_library_settings WHERE library_id=?`, libraryID).Scan(&settingsUpdatedAt); err != nil || settingsUpdatedAt != libraryCreatedAt {
		t.Fatalf("settings updated_at=%d library created_at=%d err=%v", settingsUpdatedAt, libraryCreatedAt, err)
	}
	if _, err := store.db.Exec(`INSERT INTO people(id,name,created_at_ms,updated_at_ms) VALUES('person_settings_1','保留',?,?)`, now.UnixMilli(), now.UnixMilli()); err != nil {
		t.Fatal(err)
	}
	disabled, err := store.UpdateFaceLibrarySettings(context.Background(), libraryID, false, 2, now.Add(30*time.Second))
	if err != nil || disabled.Enabled || disabled.State != "disabled" || disabled.Revision != 3 {
		t.Fatalf("disabled=%+v err=%v", disabled, err)
	}
	var count int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM people WHERE id='person_settings_1'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("people=%d err=%v", count, err)
	}
	if _, err := store.UpdateFaceLibrarySettings(context.Background(), libraryID, true, 2, now.Add(time.Minute)); !errors.Is(err, face.ErrFaceSettingsConflict) {
		t.Fatalf("conflict err=%v", err)
	}
}

func TestDisablingFaceAnalysisAtomicallyCancelsQueuedWork(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	now := time.Date(2026, 9, 2, 20, 0, 0, 0, time.UTC)
	seedFaceGeneration(t, store, "face_generation_disable_queued", 2, "active", now)
	seedFaceReadySettings(t, store, libraryID, "face_generation_disable_queued", now)
	jobs, err := face.NewJobService(store, store, &faceClearWaker{}, func() time.Time { return now }, faceClearIDs())
	if err != nil {
		t.Fatal(err)
	}
	requested, err := jobs.Request(context.Background(), libraryID, "face_generation_disable_queued", face.JobAll, "face-disable-queued-001")
	if err != nil {
		t.Fatal(err)
	}
	disabled, err := store.UpdateFaceLibrarySettings(context.Background(), libraryID, false, 1, now.Add(-time.Minute))
	if err != nil || disabled.Enabled || disabled.State != "disabled" {
		t.Fatalf("disabled=%+v err=%v", disabled, err)
	}
	stored, found, err := store.FindFaceJob(context.Background(), faceDigestForTest("face-disable-queued-001"))
	if err != nil || !found || stored.Job.State != "cancelled" || stored.Job.ErrorCode != "cancelled" || stored.Job.OperationRevision != 2 ||
		stored.Job.UpdatedAt.Before(stored.Job.CreatedAt) {
		t.Fatalf("stored=%+v found=%t err=%v", stored, found, err)
	}
	operation, err := store.GetAIOperation(context.Background(), requested.Job.OperationID)
	if err != nil || operation.State != aimodel.OperationCancelled || operation.ErrorCode != "cancelled" || operation.FinishedAt == nil ||
		operation.UpdatedAt.Before(operation.CreatedAt) || operation.FinishedAt.Before(operation.CreatedAt) {
		t.Fatalf("operation=%+v err=%v", operation, err)
	}
	if claimed, found, err := store.ClaimFaceJob(context.Background(), now.Add(2*time.Second), time.Minute); err != nil || found {
		t.Fatalf("claimed=%+v found=%t err=%v", claimed, found, err)
	}
}

func TestDisablingFaceAnalysisCooperativelyCancelsRunningWork(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	now := time.Date(2026, 9, 2, 21, 0, 0, 0, time.UTC)
	seedFaceGeneration(t, store, "face_generation_disable_running", 2, "active", now)
	seedFaceReadySettings(t, store, libraryID, "face_generation_disable_running", now)
	jobs, err := face.NewJobService(store, store, &faceClearWaker{}, func() time.Time { return now }, faceClearIDs())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := jobs.Request(context.Background(), libraryID, "face_generation_disable_running", face.JobAll, "face-disable-running-001"); err != nil {
		t.Fatal(err)
	}
	claimed, found, err := store.ClaimFaceJob(context.Background(), now.Add(time.Second), time.Minute)
	if err != nil || !found {
		t.Fatalf("claimed=%+v found=%t err=%v", claimed, found, err)
	}
	if _, err := store.UpdateFaceLibrarySettings(context.Background(), libraryID, false, 1, now.Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}
	cancelled, err := store.RefreshFaceJobLease(context.Background(), claimed, now.Add(3*time.Second), time.Minute)
	if err != nil || !cancelled {
		t.Fatalf("cancelled=%t err=%v", cancelled, err)
	}
	finished, err := store.FinishFaceJob(context.Background(), claimed, false, "cancelled", now.Add(4*time.Second))
	if err != nil || finished.State != "cancelled" {
		t.Fatalf("finished=%+v err=%v", finished, err)
	}
	settings, err := store.GetFaceLibrarySettings(context.Background(), libraryID)
	if err != nil || settings.Enabled || settings.State != "disabled" {
		t.Fatalf("settings=%+v err=%v", settings, err)
	}
}
