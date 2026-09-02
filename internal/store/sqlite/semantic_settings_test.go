package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/semantic"
)

func TestSemanticLibrarySettingsDefaultsAndRequiresAvailableActiveModel(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	settings, err := store.GetSemanticLibrarySettings(context.Background(), libraryID)
	if err != nil || settings.Enabled || settings.State != semantic.LibraryDisabled || settings.Revision != 1 || settings.Coverage.Revision != 1 {
		t.Fatalf("default settings = %#v, err %v", settings, err)
	}
	now := time.Date(2026, 8, 28, 8, 0, 0, 0, time.UTC)
	if _, err := store.UpdateSemanticLibrarySettings(context.Background(), libraryID, true, 1, now); !errors.Is(err, semantic.ErrSemanticGenerationUnavailable) {
		t.Fatalf("enable without model error = %v", err)
	}
	generationID := seedEmbeddingGeneration(t, store, 2)
	if _, err := store.db.ExecContext(context.Background(), `
        UPDATE ai_model_state SET active_model_id=(SELECT model_id FROM semantic_generations WHERE id=?),
            active_generation_id=?, revision=revision+1 WHERE singleton_key=1`, generationID, generationID); err != nil {
		t.Fatal(err)
	}
	settings, err = store.UpdateSemanticLibrarySettings(context.Background(), libraryID, true, 1, now)
	if err != nil || !settings.Enabled || settings.State != semantic.LibraryBuilding || settings.Revision != 2 || settings.ActiveGenerationID != generationID {
		t.Fatalf("enabled settings = %#v, err %v", settings, err)
	}
	if _, err := store.UpdateSemanticLibrarySettings(context.Background(), libraryID, false, 1, now.Add(time.Second)); !errors.Is(err, semantic.ErrSemanticRevisionConflict) {
		t.Fatalf("stale update error = %v", err)
	}
	settings, err = store.UpdateSemanticLibrarySettings(context.Background(), libraryID, false, 2, now.Add(time.Second))
	if err != nil || settings.Enabled || settings.State != semantic.LibraryDisabled || settings.Revision != 3 {
		t.Fatalf("disabled settings = %#v, err %v", settings, err)
	}
}

func TestSemanticLibrarySettingsRejectUnknownLibrary(t *testing.T) {
	store, _ := openTestStore(t)
	if _, err := store.GetSemanticLibrarySettings(context.Background(), 999); !errors.Is(err, semantic.ErrSemanticLibraryNotFound) {
		t.Fatalf("get error = %v", err)
	}
}

func TestSemanticLibrarySettingsClampTimeAcrossClockRollback(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	var libraryCreatedAt int64
	if err := store.db.QueryRow(`SELECT created_at_ms FROM libraries WHERE id=?`, libraryID).Scan(&libraryCreatedAt); err != nil {
		t.Fatal(err)
	}
	generationID := seedEmbeddingGeneration(t, store, 2)
	if _, err := store.db.Exec(`UPDATE ai_model_state SET active_model_id=(SELECT model_id FROM semantic_generations WHERE id=?),active_generation_id=? WHERE singleton_key=1`, generationID, generationID); err != nil {
		t.Fatal(err)
	}
	rollback := time.UnixMilli(libraryCreatedAt).Add(-time.Hour)
	if _, err := store.UpdateSemanticLibrarySettings(context.Background(), libraryID, true, 1, rollback); err != nil {
		t.Fatal(err)
	}
	assertSemanticSettingsTime := func(wantRevision int64) {
		t.Helper()
		var createdAt, updatedAt, revision int64
		if err := store.db.QueryRow(`SELECT created_at_ms,updated_at_ms,revision FROM ai_library_settings WHERE library_id=?`, libraryID).Scan(&createdAt, &updatedAt, &revision); err != nil {
			t.Fatal(err)
		}
		if createdAt < libraryCreatedAt || updatedAt < createdAt || revision != wantRevision {
			t.Fatalf("settings times/revision = %d/%d/%d, library created %d", createdAt, updatedAt, revision, libraryCreatedAt)
		}
	}
	assertSemanticSettingsTime(2)
	if _, err := store.UpdateSemanticLibrarySettings(context.Background(), libraryID, false, 2, rollback.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	assertSemanticSettingsTime(3)
}
