package sqlite

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/face"
	"github.com/HappyQuQu/foliopath/internal/library"
)

func seedFaceLifecycleState(t *testing.T, store *Store) (int64, int64) {
	t.Helper()
	libraryID := seedBrowseCatalog(t, store)
	assetID := catalogAssetID(t, store, "photo-10.jpg")
	now := time.Date(2026, 9, 2, 8, 0, 0, 0, time.UTC)
	const generation = "face_generation_lifecycle"
	seedFaceGeneration(t, store, generation, 2, "active", now)
	seedFaceReadySettings(t, store, libraryID, generation, now)
	vector, err := face.EncodeEmbedding([]float32{1, 0}, 2)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.ReplaceFaceObservations(context.Background(), face.ObservationBatch{
		GenerationID: generation,
		LibraryID:    libraryID,
		AssetID:      assetID,
		UpdatedAt:    now,
		Items: []face.ObservationItem{{
			ID: "face_lifecycle_01", Box: face.Box{X: .1, Y: .2, Width: .3, Height: .4},
			Detection: 1, Quality: 1, SourceFingerprint: "source-lifecycle-v1", Vector: vector, CreatedAt: now,
		}},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreatePerson(context.Background(), face.CreatePersonCommand{ID: "person_lifecycle_1", Name: "保留空人物", CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreatePerson(context.Background(), face.CreatePersonCommand{ID: "person_lifecycle_2", Name: "原本就是空人物", CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AssignFace(context.Background(), face.AssignFaceCommand{
		EventID: "event_lifecycle_1", RequestHash: testFaceRequestHash, AnchorID: "anchor_lifecycle_1",
		FaceID: "face_lifecycle_01", PersonID: "person_lifecycle_1", ExpectedFaceRevision: 1,
		ExpectedPersonRevision: 1, CreatedAt: now.Add(time.Minute),
	}); err != nil {
		t.Fatal(err)
	}
	return libraryID, assetID
}

func TestFaceStateSurvivesSQLiteBackupAndRestore(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID, assetID := seedFaceLifecycleState(t, store)
	backup := filepath.Join(t.TempDir(), "foliopath-face-backup.db")
	if _, err := store.db.ExecContext(context.Background(), `VACUUM INTO ?`, backup); err != nil {
		t.Fatalf("create SQLite-consistent backup: %v", err)
	}
	restored, err := Open(context.Background(), backup, Options{MaxBatchSize: 16})
	if err != nil {
		t.Fatalf("open restored backup: %v", err)
	}
	t.Cleanup(func() { _ = restored.Close() })

	for table, want := range map[string]int{
		"face_library_settings": 1,
		"face_observations":     1,
		"person_face_anchors":   1,
		"people":                2,
		"face_audit_events":     1,
	} {
		var got int
		if err := restored.db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM "+table).Scan(&got); err != nil {
			t.Fatalf("count restored %s: %v", table, err)
		}
		if got != want {
			t.Fatalf("restored %s=%d, want %d", table, got, want)
		}
	}
	var restoredLibraryID, restoredAssetID int64
	var faceID, state string
	if err := restored.db.QueryRowContext(context.Background(), `SELECT library_id,asset_id,current_face_id,state FROM person_face_anchors WHERE id='anchor_lifecycle_1'`).Scan(&restoredLibraryID, &restoredAssetID, &faceID, &state); err != nil {
		t.Fatal(err)
	}
	if restoredLibraryID != libraryID || restoredAssetID != assetID || faceID != "face_lifecycle_01" || state != "bound" {
		t.Fatalf("restored anchor=(%d,%d,%q,%q)", restoredLibraryID, restoredAssetID, faceID, state)
	}
	var integrity string
	if err := restored.db.QueryRowContext(context.Background(), `PRAGMA integrity_check`).Scan(&integrity); err != nil || integrity != "ok" {
		t.Fatalf("restored integrity=%q err=%v", integrity, err)
	}
}

func TestLibraryRemovalCascadesFaceStateAndPreservesOrphanedPeople(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID, _ := seedFaceLifecycleState(t, store)
	details, err := store.GetLibraryDetails(context.Background(), libraryID)
	if err != nil {
		t.Fatal(err)
	}
	requested, err := store.RequestLibraryRemoval(context.Background(), library.RemoveCommand{
		LibraryID: libraryID, ExpectedRevision: details.Revision, KeyHash: [32]byte{71}, RequestHash: [32]byte{72}, RetentionMS: 86_400_000,
	})
	if err != nil {
		t.Fatal(err)
	}
	claimed, found, err := store.ClaimNextLibraryRemoval(context.Background())
	if err != nil || !found || claimed.ID != requested.Removal.ID {
		t.Fatalf("claim=%+v found=%t err=%v", claimed, found, err)
	}
	done := false
	for attempt := 0; attempt < 128 && !done; attempt++ {
		done, err = store.CleanupLibraryRemovalBatch(context.Background(), claimed.ID, 1)
		if err != nil {
			t.Fatalf("cleanup attempt %d: %v", attempt, err)
		}
	}
	if !done {
		t.Fatal("bounded library removal did not finish")
	}
	if _, err := store.GetLibraryDetails(context.Background(), libraryID); !errors.Is(err, library.ErrNotFound) {
		t.Fatalf("removed library err=%v", err)
	}
	for _, table := range []string{"face_library_settings", "face_library_progress", "face_asset_results", "face_observations", "face_cluster_builds", "face_clusters", "face_cluster_members", "person_face_anchors", "face_exclusions", "face_cannot_links", "face_audit_events", "face_analysis_jobs", "face_clear_jobs"} {
		var got int
		if err := store.db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM "+table).Scan(&got); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		if got != 0 {
			t.Fatalf("%s retained %d library-scoped rows", table, got)
		}
	}
	for _, id := range []string{"person_lifecycle_1", "person_lifecycle_2"} {
		person, err := store.GetPerson(context.Background(), id)
		if err != nil {
			t.Fatalf("orphaned person %s: %v", id, err)
		}
		if person.ConfirmedFaceCount != 0 || person.AssetCount != 0 {
			t.Fatalf("orphaned person=%+v", person)
		}
	}
}
