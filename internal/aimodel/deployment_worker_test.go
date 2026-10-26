package aimodel

import (
	"context"
	"errors"
	"testing"
	"time"
)

type deploymentRuntimeStub struct {
	loaded, committed int
	err               error
}

func (r *deploymentRuntimeStub) LoadAndValidateDeployment(context.Context, Model) (RuntimeMetadata, error) {
	r.loaded++
	return RuntimeMetadata{EmbeddingDimension: 512}, r.err
}
func (r *deploymentRuntimeStub) CommitDeploymentActivation(context.Context, ActivationCommit) (Operation, error) {
	r.committed++
	return Operation{}, r.err
}

func TestDeploymentWorkerUsesSharedAdmissionAndSeparateRuntime(t *testing.T) {
	ctx := context.Background()
	models, err := NewService(&memoryRepository{snapshot: Snapshot{Revision: 1}}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	model, _, err := models.RegisterDeployment(ctx, testPackage())
	if err != nil {
		t.Fatal(err)
	}
	queue := &activationQueueStub{}
	wake := &activationWakeStub{}
	admission, err := NewActivationAdmissionService(queue, wake, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	first, err := admission.StartDeploymentActivation(ctx, model, "deployment-request")
	if err != nil || !first.Created {
		t.Fatalf("admit=%+v %v", first, err)
	}
	second, err := admission.StartDeploymentActivation(ctx, model, "deployment-request")
	if err != nil || !second.Replayed || second.Operation.ID != first.Operation.ID {
		t.Fatalf("replay=%+v %v", second, err)
	}
	worker := &ActivationWorker{models: models, queue: activationRepositoryUnused{}}
	work := ActivationWork{ModelID: model.ID, ExpectedAvailabilityRevision: model.AvailabilityRevision}
	if _, err := worker.load(ctx, work); !errors.Is(err, ErrInferenceRuntimeUnavailable) {
		t.Fatalf("unconfigured=%v", err)
	}
	runtime := &deploymentRuntimeStub{}
	if err := worker.EnableDeploymentRuntime(runtime); err != nil {
		t.Fatal(err)
	}
	if err := worker.EnableDeploymentRuntime(runtime); err == nil {
		t.Fatal("runtime replaced")
	}
	loaded, err := worker.load(ctx, work)
	if err != nil || loaded.semantic.EmbeddingDimension != 512 || runtime.loaded != 1 {
		t.Fatalf("load=%+v %v", loaded, err)
	}
	if err := worker.commit(ctx, work, first.Operation, "aig_service_worker", time.Now().UTC(), loaded); err != nil || runtime.committed != 1 {
		t.Fatalf("commit=%v count=%d", err, runtime.committed)
	}
	runtime.err = ErrModelIncompatible
	if _, err := worker.load(ctx, work); !errors.Is(err, ErrModelIncompatible) {
		t.Fatal(err)
	}
}
