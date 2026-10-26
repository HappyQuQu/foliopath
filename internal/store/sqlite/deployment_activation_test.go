package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
	"github.com/HappyQuQu/foliopath/internal/semantic"
)

func TestDeploymentActivationAtomicSpaceBinding(t *testing.T) {
	for _, scenario := range []string{"success", "wrong-space", "wrong-dimension", "stale-model", "legacy-entry", "late-binding-failure", "file-model"} {
		t.Run(scenario, func(t *testing.T) {
			ctx := context.Background()
			store, _ := openTestStore(t)
			now := time.Now().UTC()
			old := activationModelFixture(now)
			if _, _, err := store.RegisterAIModel(ctx, old); err != nil {
				t.Fatal(err)
			}
			seedActiveGeneration(t, store, old.ID, "aig_old_deployment", now)
			library := seedBrowseCatalog(t, store)
			vector := make([]float32, 768)
			vector[0] = 1
			encoded, err := semantic.EncodeEmbedding(vector, 768)
			if err != nil {
				t.Fatal(err)
			}
			if err := store.PutSemanticEmbeddingBatch(ctx, semantic.EmbeddingBatch{GenerationID: "aig_old_deployment", LibraryID: library, Items: []semantic.EmbeddingItem{{AssetID: catalogAssetID(t, store, "photo-10.jpg"), SourceFingerprint: "v1:10:10", Vector: encoded, CreatedAt: now}}}); err != nil {
				t.Fatal(err)
			}
			identity := spaceFixture()
			fingerprint, err := identity.Fingerprint()
			if err != nil {
				t.Fatal(err)
			}
			model := activationModelFixture(now)
			model.ID = "aim_deployment_new"
			model.Package.PackageID = "service-deployment"
			model.StorageMode = aimodel.StorageService
			model.Package.ContentHash = fingerprint
			model.SourceIdentity = fingerprint
			if scenario == "file-model" {
				model.StorageMode = aimodel.StorageManaged
			}
			if _, _, err := store.RegisterAIModel(ctx, model); err != nil {
				t.Fatal(err)
			}
			work := activationWorkFixture(model, now)
			// Direct queue setup is a test fixture, not a new production admission path.
			if _, _, err := store.CreateAIModelActivation(ctx, work); err != nil {
				t.Fatal(err)
			}
			claimed, found, err := store.ClaimAIModelActivation(ctx, now.Add(time.Second))
			if err != nil || !found {
				t.Fatalf("claim=%v %v", found, err)
			}
			activated := now.Add(2 * time.Second)
			commit := semantic.DeploymentActivationCommit{Identity: identity, Activation: aimodel.ActivationCommit{
				OperationID: claimed.Operation.ID, ExpectedRevision: claimed.Operation.Revision, ExpectedAvailabilityRevision: model.AvailabilityRevision,
				Generation: aimodel.Generation{ID: "aig_service_deployment", ModelID: model.ID, TransformVersion: 1, OutputSchemaVersion: 1, IndexFormatVersion: 1,
					EmbeddingDimension: 512, State: aimodel.GenerationActive, CreatedAt: activated, ActivatedAt: &activated, UpdatedAt: activated}, UpdatedAt: activated}}
			switch scenario {
			case "wrong-space":
				commit.Identity.Model = "different-model"
			case "wrong-dimension":
				commit.Activation.Generation.EmbeddingDimension = 768
			case "stale-model":
				commit.Activation.ExpectedAvailabilityRevision++
			case "late-binding-failure":
				if _, err := store.db.Exec(`CREATE TRIGGER test_block_binding BEFORE INSERT ON semantic_generation_spaces BEGIN SELECT RAISE(ABORT,'synthetic binding failure'); END;`); err != nil {
					t.Fatal(err)
				}
			}
			var result aimodel.Operation
			if scenario == "legacy-entry" {
				result, err = store.CommitAIModelActivation(ctx, commit.Activation)
			} else {
				result, err = store.CommitDeploymentActivation(ctx, commit)
			}
			if scenario == "success" {
				if err != nil || result.State != aimodel.OperationSucceeded {
					t.Fatalf("activation=%+v %v", result, err)
				}
				generation, err := store.GetSemanticGenerationRuntime(ctx, commit.Activation.Generation.ID)
				if err != nil || generation.State != "active" || generation.SpaceFingerprint != fingerprint {
					t.Fatalf("unbound activation=%+v %v", generation, err)
				}
				if _, err := store.CommitDeploymentActivation(ctx, commit); err == nil {
					t.Fatal("stale commit replay succeeded")
				}
			} else if err == nil {
				t.Fatal("invalid activation succeeded")
			}
			var active, oldState string
			if err := store.db.QueryRow(`SELECT active_generation_id FROM ai_model_state WHERE singleton_key=1`).Scan(&active); err != nil {
				t.Fatal(err)
			}
			if err := store.db.QueryRow(`SELECT state FROM semantic_generations WHERE id='aig_old_deployment'`).Scan(&oldState); err != nil {
				t.Fatal(err)
			}
			if scenario == "success" {
				if active != commit.Activation.Generation.ID || oldState != "retired" {
					t.Fatal("active pointer did not switch atomically")
				}
			} else {
				if active != "aig_old_deployment" || oldState != "active" {
					t.Fatal("failed activation lost old active generation")
				}
				var count int
				if err := store.db.QueryRow(`SELECT COUNT(*) FROM semantic_generations WHERE id=?`, commit.Activation.Generation.ID).Scan(&count); err != nil || count != 0 {
					t.Fatal("failed activation left partial generation")
				}
				operation, err := store.GetAIOperation(ctx, claimed.Operation.ID)
				if err != nil || operation.State != aimodel.OperationRunning {
					t.Fatal("failed commit changed operation")
				}
			}
			var count int
			if err := store.db.QueryRow(`SELECT COUNT(*) FROM semantic_embeddings WHERE generation_id='aig_old_deployment'`).Scan(&count); err != nil || count != 1 {
				t.Fatal("activation removed old vectors")
			}
		})
	}
}
