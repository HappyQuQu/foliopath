package aimodel

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type faceDeploymentStub struct {
	metadata  DeploymentFaceMetadata
	committed *FaceActivationCommit
	err       error
}

func (r *faceDeploymentStub) LoadAndValidateFaceDeployment(context.Context, Model) (DeploymentFaceMetadata, error) {
	return r.metadata, r.err
}
func (r *faceDeploymentStub) CommitFaceDeploymentActivation(_ context.Context, c FaceActivationCommit) (Operation, error) {
	r.committed = &c
	return Operation{}, r.err
}

func TestFaceDeploymentUsesDedicatedRuntimeAndCommit(t *testing.T) {
	ctx := context.Background()
	models, _ := NewService(&memoryRepository{snapshot: Snapshot{Revision: 1}}, nil, nil)
	pkg := testPackage()
	pkg.Purpose = PurposeFaceDetectionEmbedding
	if _, _, err := models.RegisterDeployment(ctx, pkg); !errors.Is(err, ErrInvalidModel) {
		t.Fatal("semantic registration accepted face")
	}
	model, created, err := models.RegisterFaceDeployment(ctx, pkg)
	if err != nil || !created || ValidateModel(model) != nil {
		t.Fatal(model, err)
	}
	if _, _, err := models.RegisterFaceDeployment(ctx, testPackage()); !errors.Is(err, ErrInvalidModel) {
		t.Fatal("face registration accepted semantic")
	}
	queue := &activationQueueStub{}
	admission, _ := NewActivationAdmissionService(queue, &activationWakeStub{}, nil, nil)
	if _, err := admission.StartActivation(ctx, model, "face-service-request"); !errors.Is(err, ErrModelIncompatible) {
		t.Fatal(err)
	}
	first, err := admission.StartDeploymentActivation(ctx, model, "face-service-request")
	if err != nil || !first.Created {
		t.Fatal(first, err)
	}
	replay, err := admission.StartDeploymentActivation(ctx, model, "face-service-request")
	if err != nil || !replay.Replayed || replay.Operation.ID != first.Operation.ID {
		t.Fatal(replay, err)
	}
	worker := &ActivationWorker{models: models, queue: activationRepositoryUnused{}}
	semantic := &deploymentRuntimeStub{}
	if err := worker.EnableDeploymentRuntime(semantic); err != nil {
		t.Fatal(err)
	}
	work := ActivationWork{ModelID: model.ID, ExpectedAvailabilityRevision: model.AvailabilityRevision}
	if _, err := worker.load(ctx, work); !errors.Is(err, ErrInferenceRuntimeUnavailable) {
		t.Fatal(err)
	}
	if semantic.loaded != 0 {
		t.Fatal("face reached CLIP runtime")
	}
	runtime := &faceDeploymentStub{metadata: DeploymentFaceMetadata{
		FaceRuntimeMetadata: FaceRuntimeMetadata{EmbeddingDimension: 512, ThresholdProfile: "buffalo-test"},
		DetectorContentHash: strings.Repeat("a", 64), EmbedderContentHash: strings.Repeat("b", 64), ThresholdProfileHash: strings.Repeat("c", 64)}}
	if err := worker.EnableFaceDeploymentRuntime(runtime); err != nil {
		t.Fatal(err)
	}
	if err := worker.EnableFaceDeploymentRuntime(runtime); err == nil {
		t.Fatal("runtime replaced")
	}
	loaded, err := worker.load(ctx, work)
	if err != nil {
		t.Fatal(err)
	}
	if err := worker.commit(ctx, work, first.Operation, "aig_face_service", time.Now().UTC(), loaded); err != nil {
		t.Fatal(err)
	}
	if runtime.committed == nil || runtime.committed.Generation.DetectorContentHash != runtime.metadata.DetectorContentHash || semantic.committed != 0 {
		t.Fatal("wrong commit")
	}
	runtime.committed = nil
	loaded.faceService.ThresholdProfileHash = ""
	if err := worker.commit(ctx, work, first.Operation, "aig_face_service", time.Now().UTC(), loaded); !errors.Is(err, ErrModelIncompatible) || runtime.committed != nil {
		t.Fatal("invalid generation committed", err)
	}
	runtime.err = context.Canceled
	if _, err := worker.load(ctx, work); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	work.ExpectedAvailabilityRevision++
	if _, err := worker.load(ctx, work); !errors.Is(err, ErrModelUnavailable) {
		t.Fatal(err)
	}
}
