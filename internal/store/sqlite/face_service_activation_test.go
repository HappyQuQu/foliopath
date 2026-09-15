package sqlite

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
)

func TestFaceServiceCannotUseFileActivationCommit(t *testing.T) {
	store, _ := openTestStore(t)
	now := time.Now().UTC()
	model := faceActivationModelFixture(now)
	model.StorageMode = aimodel.StorageService
	model.SourceIdentity = model.Package.ContentHash
	if _, _, err := store.RegisterAIModel(context.Background(), model); err != nil {
		t.Fatal(err)
	}
	work := activationWorkFixture(model, now)
	if _, _, err := store.CreateAIModelActivation(context.Background(), work); err != nil {
		t.Fatal(err)
	}
	claimed, found, err := store.ClaimAIModelActivation(context.Background(), now.Add(time.Second))
	if err != nil || !found {
		t.Fatal(found, err)
	}
	activated := now.Add(2 * time.Second)
	_, err = store.CommitFaceModelActivation(context.Background(), aimodel.FaceActivationCommit{
		OperationID: claimed.Operation.ID, ExpectedRevision: claimed.Operation.Revision, ExpectedAvailabilityRevision: claimed.ExpectedAvailabilityRevision,
		Generation: aimodel.FaceGeneration{ID: "aig_service_rejected", ModelID: model.ID, PackageID: model.Package.PackageID,
			DetectorContentHash: strings.Repeat("a", 64), EmbedderContentHash: strings.Repeat("b", 64), ThresholdProfileHash: strings.Repeat("c", 64),
			EmbeddingDimension: 512, TransformVersion: aimodel.FaceTransformVersion, ThresholdProfile: "test-profile",
			State: aimodel.GenerationActive, CreatedAt: activated, ActivatedAt: &activated, UpdatedAt: activated}, UpdatedAt: activated})
	if !errors.Is(err, aimodel.ErrModelIncompatible) {
		t.Fatal(err)
	}
	var count int
	if err := store.db.QueryRow(`SELECT count(*) FROM face_generations WHERE id='aig_service_rejected'`).Scan(&count); err != nil || count != 0 {
		t.Fatal(count, err)
	}
}
