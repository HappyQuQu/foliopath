package sqlite

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
)

func TestAIModelActivationAtomicallyReplacesOldGeneration(t *testing.T) {
	store, _ := openTestStore(t)
	now := time.Date(2026, 8, 27, 18, 0, 0, 0, time.UTC)
	model := activationModelFixture(now)
	if _, _, err := store.RegisterAIModel(context.Background(), model); err != nil {
		t.Fatal(err)
	}
	seedActiveGeneration(t, store, model.ID, "aig_old_generation", now.Add(-time.Hour))
	work := activationWorkFixture(model, now)
	if _, created, err := store.CreateAIModelActivation(context.Background(), work); err != nil || !created {
		t.Fatalf("create = %v, %v", created, err)
	}
	claimed, found, err := store.ClaimAIModelActivation(context.Background(), now.Add(time.Second))
	if err != nil || !found || claimed.Operation.Phase != aimodel.PhaseLoading {
		t.Fatalf("claim = %#v, %v, %v", claimed, found, err)
	}
	activated := now.Add(2 * time.Second)
	operation, err := store.CommitAIModelActivation(context.Background(), aimodel.ActivationCommit{
		OperationID: claimed.Operation.ID, ExpectedRevision: claimed.Operation.Revision,
		ExpectedAvailabilityRevision: claimed.ExpectedAvailabilityRevision,
		Generation: aimodel.Generation{
			ID: "aig_new_generation", ModelID: model.ID,
			TransformVersion: aimodel.SemanticTransformVersion, OutputSchemaVersion: aimodel.SemanticOutputSchemaVersion,
			IndexFormatVersion: aimodel.SemanticIndexFormatVersion, EmbeddingDimension: 768,
			State: aimodel.GenerationActive, CreatedAt: activated, ActivatedAt: &activated, UpdatedAt: activated,
		}, UpdatedAt: activated,
	})
	if err != nil || operation.State != aimodel.OperationSucceeded {
		t.Fatalf("commit = %#v, %v", operation, err)
	}
	var activeModel, activeGeneration string
	if err := store.db.QueryRow(`SELECT active_model_id, active_generation_id FROM ai_model_state WHERE singleton_key = 1`).Scan(&activeModel, &activeGeneration); err != nil {
		t.Fatal(err)
	}
	if activeModel != model.ID || activeGeneration != "aig_new_generation" {
		t.Fatalf("active = %q/%q", activeModel, activeGeneration)
	}
	var oldState string
	if err := store.db.QueryRow(`SELECT state FROM semantic_generations WHERE id = 'aig_old_generation'`).Scan(&oldState); err != nil || oldState != "retired" {
		t.Fatalf("old state = %q, %v", oldState, err)
	}
}

func TestAIModelActivationFailurePreservesOldActiveGeneration(t *testing.T) {
	store, _ := openTestStore(t)
	now := time.Date(2026, 8, 27, 19, 0, 0, 0, time.UTC)
	model := activationModelFixture(now)
	if _, _, err := store.RegisterAIModel(context.Background(), model); err != nil {
		t.Fatal(err)
	}
	seedActiveGeneration(t, store, model.ID, "aig_old_generation", now.Add(-time.Hour))
	work := activationWorkFixture(model, now)
	if _, _, err := store.CreateAIModelActivation(context.Background(), work); err != nil {
		t.Fatal(err)
	}
	claimed, _, err := store.ClaimAIModelActivation(context.Background(), now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	activated := now.Add(2 * time.Second)
	_, err = store.CommitAIModelActivation(context.Background(), aimodel.ActivationCommit{
		OperationID: claimed.Operation.ID, ExpectedRevision: claimed.Operation.Revision,
		ExpectedAvailabilityRevision: claimed.ExpectedAvailabilityRevision,
		Generation: aimodel.Generation{ID: "aig_old_generation", ModelID: model.ID,
			TransformVersion: 1, OutputSchemaVersion: 1, IndexFormatVersion: 1, EmbeddingDimension: 768,
			State: aimodel.GenerationActive, CreatedAt: activated, ActivatedAt: &activated, UpdatedAt: activated}, UpdatedAt: activated,
	})
	if err == nil {
		t.Fatal("invalid generation commit succeeded")
	}
	var activeGeneration, oldState string
	if err := store.db.QueryRow(`SELECT active_generation_id FROM ai_model_state WHERE singleton_key = 1`).Scan(&activeGeneration); err != nil {
		t.Fatal(err)
	}
	if err := store.db.QueryRow(`SELECT state FROM semantic_generations WHERE id = 'aig_old_generation'`).Scan(&oldState); err != nil {
		t.Fatal(err)
	}
	if activeGeneration != "aig_old_generation" || oldState != "active" {
		t.Fatalf("fallback changed = %q/%q", activeGeneration, oldState)
	}
}

func TestAIModelActivationRejectsAvailabilityRevisionChangedDuringLoad(t *testing.T) {
	store, _ := openTestStore(t)
	now := time.Date(2026, 8, 27, 19, 30, 0, 0, time.UTC)
	model := activationModelFixture(now)
	if _, _, err := store.RegisterAIModel(context.Background(), model); err != nil {
		t.Fatal(err)
	}
	seedActiveGeneration(t, store, model.ID, "aig_old_generation", now.Add(-time.Hour))
	work := activationWorkFixture(model, now)
	if _, _, err := store.CreateAIModelActivation(context.Background(), work); err != nil {
		t.Fatal(err)
	}
	claimed, found, err := store.ClaimAIModelActivation(context.Background(), now.Add(time.Second))
	if err != nil || !found {
		t.Fatalf("claim = %#v found=%v err=%v", claimed, found, err)
	}
	if _, err := store.db.Exec(`UPDATE ai_models SET availability_revision=availability_revision+1 WHERE id=?`, model.ID); err != nil {
		t.Fatal(err)
	}
	activated := now.Add(2 * time.Second)
	_, err = store.CommitAIModelActivation(context.Background(), aimodel.ActivationCommit{
		OperationID: claimed.Operation.ID, ExpectedRevision: claimed.Operation.Revision,
		ExpectedAvailabilityRevision: claimed.ExpectedAvailabilityRevision,
		Generation: aimodel.Generation{
			ID: "aig_stale_generation", ModelID: model.ID,
			TransformVersion: aimodel.SemanticTransformVersion, OutputSchemaVersion: aimodel.SemanticOutputSchemaVersion,
			IndexFormatVersion: aimodel.SemanticIndexFormatVersion, EmbeddingDimension: 768,
			State: aimodel.GenerationActive, CreatedAt: activated, ActivatedAt: &activated, UpdatedAt: activated,
		}, UpdatedAt: activated,
	})
	if !errors.Is(err, aimodel.ErrPreconditionFailed) {
		t.Fatalf("stale availability commit error = %v", err)
	}
	var activeGeneration, oldState string
	if err := store.db.QueryRow(`SELECT active_generation_id FROM ai_model_state WHERE singleton_key=1`).Scan(&activeGeneration); err != nil {
		t.Fatal(err)
	}
	if err := store.db.QueryRow(`SELECT state FROM semantic_generations WHERE id='aig_old_generation'`).Scan(&oldState); err != nil {
		t.Fatal(err)
	}
	var staleCount int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM semantic_generations WHERE id='aig_stale_generation'`).Scan(&staleCount); err != nil {
		t.Fatal(err)
	}
	if activeGeneration != "aig_old_generation" || oldState != "active" || staleCount != 0 {
		t.Fatalf("stale commit changed active state = %q/%q stale=%d", activeGeneration, oldState, staleCount)
	}
}

func activationModelFixture(now time.Time) aimodel.Model {
	return aimodel.Model{ID: "aim_activation_model", Package: aimodel.VerifiedPackage{
		PackageID: "semantic-activation-v1", Purpose: aimodel.PurposeSemanticImageText, Version: "1.0.0",
		Architecture: "arm64", ContentHash: strings.Repeat("e", 64), LicenseID: "Apache-2.0", PackageSizeByte: 3,
	}, StorageMode: aimodel.StorageManaged, State: aimodel.StateAvailable, SourceIdentity: "managed:activation",
		AvailabilityRevision: 1, CreatedAt: now, UpdatedAt: now}
}

func activationWorkFixture(model aimodel.Model, now time.Time) aimodel.ActivationWork {
	return aimodel.ActivationWork{IdempotencyKey: "activate-request-123", ModelID: model.ID,
		ExpectedAvailabilityRevision: model.AvailabilityRevision,
		RequestHash:                  aimodel.ActivationRequestHash(model.ID, model.AvailabilityRevision),
		Operation: aimodel.Operation{ID: "aio_activation_operation", Kind: aimodel.OperationModelActivate,
			State: aimodel.OperationQueued, Phase: aimodel.PhaseQueued, ModelID: model.ID,
			Revision: 1, CreatedAt: now, UpdatedAt: now}}
}

func seedActiveGeneration(t *testing.T, store *Store, modelID, generationID string, now time.Time) {
	t.Helper()
	if _, err := store.db.Exec(`INSERT INTO semantic_generations(
        id, model_id, transform_version, output_schema_version, index_format_version,
        embedding_dimension, state, created_at_ms, activated_at_ms, updated_at_ms
    ) VALUES(?, ?, 1, 1, 1, 768, 'active', ?, ?, ?)`, generationID, modelID, now.UnixMilli(), now.UnixMilli(), now.UnixMilli()); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(`UPDATE ai_model_state SET active_model_id = ?, active_generation_id = ? WHERE singleton_key = 1`, modelID, generationID); err != nil {
		t.Fatal(err)
	}
}
