package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
)

func TestAIModelRepositoryRegistersIdempotentlyAndChecksRevision(t *testing.T) {
	store, _ := openTestStore(t)
	now := time.Date(2026, 8, 27, 13, 0, 0, 0, time.UTC)
	model := aimodel.Model{
		ID: "aim_repository_model",
		Package: aimodel.VerifiedPackage{
			PackageID:       "semantic-test-v1",
			Purpose:         aimodel.PurposeSemanticImageText,
			Version:         "1.0.0",
			Architecture:    "arm64",
			ContentHash:     "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			LicenseID:       "Apache-2.0",
			PackageSizeByte: 2048,
		},
		StorageMode:          aimodel.StorageManaged,
		State:                aimodel.StateAvailable,
		SourceIdentity:       "managed:sha256:test",
		AvailabilityRevision: 1,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	created, wasCreated, err := store.RegisterAIModel(context.Background(), model)
	if err != nil || !wasCreated || created.ID != model.ID {
		t.Fatalf("create = %#v, %v, %v", created, wasCreated, err)
	}
	model.ID = "aim_duplicate_model"
	repeated, wasCreated, err := store.RegisterAIModel(context.Background(), model)
	if err != nil || wasCreated || repeated.ID != created.ID {
		t.Fatalf("repeat = %#v, %v, %v", repeated, wasCreated, err)
	}
	snapshot, err := store.ListAIModels(context.Background())
	if err != nil || snapshot.Revision != 2 || len(snapshot.Items) != 1 {
		t.Fatalf("snapshot = %#v, %v", snapshot, err)
	}
	now = now.Add(time.Minute)
	unavailable, err := store.SetAIModelAvailability(context.Background(), created.ID, 1, aimodel.StateUnavailable, now)
	if err != nil || unavailable.State != aimodel.StateUnavailable || unavailable.AvailabilityRevision != 2 {
		t.Fatalf("unavailable = %#v, %v", unavailable, err)
	}
	if _, err := store.SetAIModelAvailability(context.Background(), created.ID, 1, aimodel.StateAvailable, now); !errors.Is(err, aimodel.ErrPreconditionFailed) {
		t.Fatalf("stale update error = %v", err)
	}
}

func TestAISemanticMigrationConstraintsAndLibraryCascade(t *testing.T) {
	store, _ := openTestStore(t)
	for _, table := range []string{
		"ai_models", "semantic_generations", "ai_model_state", "ai_model_operations", "ai_model_install_requests", "ai_model_activation_requests",
		"ai_library_settings", "semantic_library_progress", "semantic_embeddings", "semantic_jobs",
		"semantic_clear_jobs", "semantic_clear_requests",
	} {
		var count int
		if err := store.db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("table %s count = %d", table, count)
		}
	}
	var version int
	if err := store.db.QueryRow(`SELECT MAX(version_id) FROM goose_db_version WHERE is_applied = 1`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != 32 {
		t.Fatalf("migration version = %d, want 32", version)
	}
}
