package sqlite

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
	"github.com/HappyQuQu/foliopath/internal/face"
)

func TestFaceDeploymentActivationAtomicIdentity(t *testing.T) {
	for _, scenario := range []string{"success", "wrong-identity", "wrong-component", "wrong-threshold", "stale-operation", "stale-model", "cancelled", "late-failure", "file-model", "legacy-entry"} {
		t.Run(scenario, func(t *testing.T) {
			ctx := context.Background()
			store, _ := openTestStore(t)
			now := time.Now().UTC()
			old := faceActivationModelFixture(now)
			if _, _, err := store.RegisterAIModel(ctx, old); err != nil {
				t.Fatal(err)
			}
			seedActiveFaceGeneration(t, store, old, "face_previous", now)
			semantic := activationModelFixture(now)
			if _, _, err := store.RegisterAIModel(ctx, semantic); err != nil {
				t.Fatal(err)
			}
			seedActiveGeneration(t, store, semantic.ID, "aig_semantic_untouched", now)
			library := seedBrowseCatalog(t, store)
			if _, err := store.db.Exec(`INSERT INTO face_library_settings(library_id,enabled,state,active_generation_id,created_at_ms,updated_at_ms) VALUES(?,1,'ready','face_previous',?,?)`, library, now.UnixMilli(), now.UnixMilli()); err != nil {
				t.Fatal(err)
			}
			identity := face.DeploymentIdentity{Version: 1, Protocol: "foliopath-face-v1", Model: "buffalo_l", Dimension: 512,
				RuntimeSHA256: strings.Repeat("a", 64), CacheManifestSHA256: strings.Repeat("b", 64), PreprocessingSHA256: strings.Repeat("c", 64),
				DetectorSHA256: strings.Repeat("d", 64), EmbedderSHA256: strings.Repeat("e", 64),
				Threshold: face.ThresholdProfile{SchemaVersion: 1, ProfileID: "test-only", CoreSimilarity: .8, EdgeSimilarity: .6, MinCoreSize: 2, QualitySummarySHA256: strings.Repeat("f", 64)}}
			fingerprint, err := identity.Fingerprint()
			if err != nil {
				t.Fatal(err)
			}
			model := old
			model.ID = "aim_face_deployment"
			model.Package.PackageID = "buffalo_l"
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
			if _, _, err := store.CreateAIModelActivation(ctx, work); err != nil {
				t.Fatal(err)
			}
			claimed, found, err := store.ClaimAIModelActivation(ctx, now.Add(time.Second))
			if err != nil || !found {
				t.Fatal(found, err)
			}
			metadata, err := identity.Metadata()
			if err != nil {
				t.Fatal(err)
			}
			activated := now.Add(2 * time.Second)
			commit := face.DeploymentActivationCommit{Identity: identity, Activation: aimodel.FaceActivationCommit{
				OperationID: claimed.Operation.ID, ExpectedRevision: claimed.Operation.Revision, ExpectedAvailabilityRevision: model.AvailabilityRevision, UpdatedAt: activated,
				Generation: aimodel.FaceGeneration{ID: "face_service_new", ModelID: model.ID, PackageID: "buffalo_l", DetectorContentHash: metadata.DetectorContentHash, EmbedderContentHash: metadata.EmbedderContentHash,
					EmbeddingDimension: 512, TransformVersion: aimodel.FaceTransformVersion, ThresholdProfile: metadata.ThresholdProfile, ThresholdProfileHash: metadata.ThresholdProfileHash,
					State: aimodel.GenerationActive, CreatedAt: activated, ActivatedAt: &activated, UpdatedAt: activated}}}
			switch scenario {
			case "wrong-identity":
				commit.Identity.RuntimeSHA256 = strings.Repeat("0", 64)
			case "wrong-component":
				commit.Activation.Generation.DetectorContentHash = strings.Repeat("0", 64)
			case "wrong-threshold":
				commit.Activation.Generation.ThresholdProfileHash = strings.Repeat("0", 64)
			case "stale-operation":
				commit.Activation.ExpectedRevision++
			case "stale-model":
				commit.Activation.ExpectedAvailabilityRevision++
			case "cancelled":
				if _, err := store.db.Exec(`UPDATE ai_model_operations SET state='cancelling' WHERE id=?`, claimed.Operation.ID); err != nil {
					t.Fatal(err)
				}
			case "late-failure":
				if _, err := store.db.Exec(`CREATE TRIGGER test_face_commit_failure BEFORE INSERT ON face_generations BEGIN SELECT RAISE(ABORT,'synthetic insert failure'); END`); err != nil {
					t.Fatal(err)
				}
			}
			var result aimodel.Operation
			if scenario == "legacy-entry" {
				result, err = store.CommitFaceModelActivation(ctx, commit.Activation)
			} else {
				result, err = store.CommitFaceDeploymentActivation(ctx, commit)
			}
			if scenario == "success" {
				if err != nil || result.State != aimodel.OperationSucceeded {
					t.Fatal(result, err)
				}
				if _, err := store.CommitFaceDeploymentActivation(ctx, commit); err == nil {
					t.Fatal("stale replay accepted")
				}
			} else if err == nil {
				t.Fatal("invalid commit accepted")
			}
			var oldState, semanticID, libraryID string
			var count int
			if err := store.db.QueryRow(`SELECT state FROM face_generations WHERE id='face_previous'`).Scan(&oldState); err != nil {
				t.Fatal(err)
			}
			if err := store.db.QueryRow(`SELECT count(*) FROM face_generations WHERE id='face_service_new' AND state='active'`).Scan(&count); err != nil {
				t.Fatal(err)
			}
			if scenario == "success" {
				if oldState != "retired" || count != 1 {
					t.Fatal(oldState, count)
				}
				runtime, err := store.GetFaceDeploymentGenerationRuntime(ctx, "face_service_new")
				if err != nil || runtime.Fingerprint != fingerprint || !runtime.Available || runtime.Dimension != 512 || runtime.State != "active" {
					t.Fatal(runtime, err)
				}
				legacy, err := store.GetFaceDeploymentGenerationRuntime(ctx, "face_previous")
				if err != nil || legacy.Fingerprint != "" || legacy.State != "retired" {
					t.Fatal(legacy, err)
				}
			} else {
				if oldState != "active" || count != 0 {
					t.Fatal("partial transaction", oldState, count)
				}
				op, err := store.GetAIOperation(ctx, claimed.Operation.ID)
				if err != nil {
					t.Fatal(err)
				}
				want := aimodel.OperationRunning
				if scenario == "cancelled" {
					want = aimodel.OperationCancelling
				}
				if op.State != want {
					t.Fatal(op.State)
				}
			}
			if err := store.db.QueryRow(`SELECT active_generation_id FROM ai_model_state WHERE singleton_key=1`).Scan(&semanticID); err != nil {
				t.Fatal(err)
			}
			if err := store.db.QueryRow(`SELECT active_generation_id FROM face_library_settings WHERE library_id=?`, library).Scan(&libraryID); err != nil {
				t.Fatal(err)
			}
			if semanticID != "aig_semantic_untouched" || libraryID != "face_previous" {
				t.Fatal("unrelated pointer changed", semanticID, libraryID)
			}
		})
	}
}
