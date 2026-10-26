package aimodel

import (
	"context"
	"errors"
	"testing"
)

func TestDeploymentModelIsNotAFileInstallation(t *testing.T) {
	repo := &memoryRepository{snapshot: Snapshot{Revision: 1}}
	models, err := NewService(repo, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	model, created, err := models.RegisterDeployment(context.Background(), testPackage())
	if err != nil || !created || model.StorageMode != StorageService || model.SourceIdentity != model.Package.ContentHash {
		t.Fatalf("model=%+v created=%v err=%v", model, created, err)
	}
	if _, _, err := models.RegisterInstalled(context.Background(), testPackage(), StorageService, model.SourceIdentity); !errors.Is(err, ErrInvalidModel) {
		t.Fatal("file install accepted service")
	}
	catalog, _, _ := catalogFixture(t)
	refresh, err := NewAvailabilityService(models, catalog, &activationSourceStub{err: ErrModelSourceUnavailable})
	if err != nil {
		t.Fatal(err)
	}
	summary, err := refresh.Refresh(context.Background())
	if err != nil || summary.Checked != 0 || summary.Changed != 0 {
		t.Fatalf("file refresh touched deployment: %+v %v", summary, err)
	}
	queue := &activationQueueStub{}
	activation, err := NewActivationAdmissionService(queue, &activationWakeStub{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := activation.StartActivation(context.Background(), model, "deployment-activation"); !errors.Is(err, ErrModelIncompatible) || queue.found {
		t.Fatalf("legacy activation admitted service: %v", err)
	}
	model.SourceIdentity = "/models/wrong"
	if ValidateModel(model) == nil {
		t.Fatal("service accepted path identity")
	}
}
