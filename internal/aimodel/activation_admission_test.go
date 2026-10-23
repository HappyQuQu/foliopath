package aimodel

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type activationQueueStub struct {
	work         ActivationWork
	found        bool
	created      bool
	mutateCreate func(*ActivationWork)
}

func (queue *activationQueueStub) FindAIModelActivation(context.Context, string) (ActivationWork, bool, error) {
	return queue.work, queue.found, nil
}

func (queue *activationQueueStub) CreateAIModelActivation(_ context.Context, work ActivationWork) (ActivationWork, bool, error) {
	if queue.found {
		return queue.work, false, nil
	}
	queue.work, queue.found, queue.created = work, true, true
	if queue.mutateCreate != nil {
		queue.mutateCreate(&queue.work)
	}
	return queue.work, true, nil
}

func (*activationQueueStub) ClaimAIModelActivation(context.Context, time.Time) (ActivationWork, bool, error) {
	return ActivationWork{}, false, nil
}

func (*activationQueueStub) CommitAIModelActivation(context.Context, ActivationCommit) (Operation, error) {
	return Operation{}, errors.New("not used")
}

type activationWakeStub struct{ calls int }

func (wake *activationWakeStub) Wake() { wake.calls++ }

func TestActivationAdmissionCreatesAndReplaysDurableRequest(t *testing.T) {
	now := time.Date(2026, 8, 27, 20, 0, 0, 0, time.UTC)
	queue := &activationQueueStub{}
	wake := &activationWakeStub{}
	service, err := NewActivationAdmissionService(queue, wake, func() time.Time { return now }, func() (string, error) {
		return "aio_activate123", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	model := activationAdmissionModel(now)
	first, err := service.StartActivation(context.Background(), model, "activate-request-123")
	if err != nil || !first.Created || first.Replayed || first.Operation.Kind != OperationModelActivate ||
		first.Operation.ModelID != model.ID || wake.calls != 1 {
		t.Fatalf("first = %#v, err=%v, wake=%d", first, err, wake.calls)
	}
	second, err := service.StartActivation(context.Background(), model, "activate-request-123")
	if err != nil || second.Created || !second.Replayed || second.Operation.ID != first.Operation.ID || wake.calls != 1 {
		t.Fatalf("replay = %#v, err=%v, wake=%d", second, err, wake.calls)
	}
}

func TestActivationAdmissionRejectsConflictAndUnavailableModel(t *testing.T) {
	now := time.Date(2026, 8, 27, 20, 0, 0, 0, time.UTC)
	queue := &activationQueueStub{}
	service, err := NewActivationAdmissionService(queue, &activationWakeStub{}, func() time.Time { return now }, func() (string, error) {
		return "aio_activate123", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	model := activationAdmissionModel(now)
	if _, err := service.StartActivation(context.Background(), model, "activate-request-123"); err != nil {
		t.Fatal(err)
	}
	changed := model
	changed.AvailabilityRevision++
	if _, _, err := service.ReplayActivation(context.Background(), changed.ID, changed.AvailabilityRevision, "activate-request-123"); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("conflict error = %v", err)
	}
	unavailable := model
	unavailable.State = StateUnavailable
	if _, err := service.StartActivation(context.Background(), unavailable, "another-request-123"); !errors.Is(err, ErrModelUnavailable) {
		t.Fatalf("unavailable error = %v", err)
	}
}

func TestActivationAdmissionValidatesQueueResultBeforeWake(t *testing.T) {
	now := time.Date(2026, 8, 29, 8, 0, 0, 0, time.UTC)
	queue := &activationQueueStub{mutateCreate: func(work *ActivationWork) {
		work.Operation.ModelID = "aim_different123"
	}}
	wake := &activationWakeStub{}
	service, err := NewActivationAdmissionService(queue, wake, func() time.Time { return now }, func() (string, error) {
		return "aio_activate123", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.StartActivation(context.Background(), activationAdmissionModel(now), "activate-request-123"); !errors.Is(err, ErrRepositoryState) {
		t.Fatalf("StartActivation() error = %v", err)
	}
	if wake.calls != 0 {
		t.Fatalf("wake calls = %d, want 0", wake.calls)
	}
}

func activationAdmissionModel(now time.Time) Model {
	return Model{ID: "aim_activation123", Package: VerifiedPackage{
		PackageID: "semantic-package", Purpose: PurposeSemanticImageText, Version: "1.0.0",
		Architecture: "arm64", ContentHash: strings.Repeat("a", 64), LicenseID: "Apache-2.0", PackageSizeByte: 30,
	}, StorageMode: StorageManaged, State: StateAvailable, SourceIdentity: "managed:semantic-package",
		AvailabilityRevision: 3, CreatedAt: now, UpdatedAt: now}
}

var _ ActivationRepository = (*activationQueueStub)(nil)
