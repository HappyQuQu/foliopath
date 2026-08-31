package aimodel

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type installQueueStub struct {
	work         InstallWork
	found        bool
	mutateCreate func(*InstallWork)
}

func (queue *installQueueStub) FindAIModelInstall(context.Context, string) (InstallWork, bool, error) {
	return queue.work, queue.found, nil
}

func (queue *installQueueStub) CreateAIModelInstall(_ context.Context, work InstallWork) (InstallWork, bool, error) {
	if queue.found {
		return queue.work, false, nil
	}
	queue.work, queue.found = work, true
	if queue.mutateCreate != nil {
		queue.mutateCreate(&queue.work)
	}
	return queue.work, true, nil
}

func (*installQueueStub) ClaimAIModelInstall(context.Context, time.Time) (InstallWork, bool, error) {
	return InstallWork{}, false, nil
}

func TestInstallAdmissionValidatesQueueResultBeforeWake(t *testing.T) {
	now := time.Date(2026, 8, 29, 8, 0, 0, 0, time.UTC)
	catalog, manifest, _ := catalogFixture(t)
	verified, _, found := catalog.PackageByContentHash(strings.Repeat("d", 64), "arm64")
	if !found {
		t.Fatal("catalog package missing")
	}
	candidate := Candidate{
		ID: "aic_candidate123", Package: verified, Manifest: manifest,
		Compatibility: "compatible", SourceIdentity: "source:reviewed",
	}
	queue := &installQueueStub{mutateCreate: func(work *InstallWork) {
		work.Candidate.SourceIdentity = "source:different"
	}}
	wake := &activationWakeStub{}
	service, err := NewInstallAdmissionService(queue, wake, func() time.Time { return now }, func() (string, error) {
		return "aio_install123", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.StartInstall(context.Background(), candidate, StorageManaged, "install-request-123"); !errors.Is(err, ErrRepositoryState) {
		t.Fatalf("StartInstall() error = %v", err)
	}
	if wake.calls != 0 {
		t.Fatalf("wake calls = %d, want 0", wake.calls)
	}
}

var _ InstallQueue = (*installQueueStub)(nil)
