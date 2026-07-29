package scanner

import (
	"context"
	"testing"
	"time"
)

type reconcileRepositoryStub struct {
	job       ReconcileJob
	libraryID int64
	path      string
	debounce  time.Duration
	maximum   time.Duration
}

func (stub *reconcileRepositoryStub) EnqueueReconcile(
	_ context.Context,
	libraryID int64,
	path string,
	debounce time.Duration,
	maximum time.Duration,
) (ReconcileJob, error) {
	stub.libraryID = libraryID
	stub.path = path
	stub.debounce = debounce
	stub.maximum = maximum
	return stub.job, nil
}

type reconcileWakerStub struct{ count int }

func (stub *reconcileWakerStub) Wake() { stub.count++ }

func TestReconcileAdmissionNormalizesTargetsAndWakesWorker(t *testing.T) {
	repository := &reconcileRepositoryStub{job: ReconcileJob{ID: 1}}
	waker := &reconcileWakerStub{}
	service, err := NewReconcileAdmission(repository, waker)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.MarkDirty(context.Background(), 7, "albums/2026"); err != nil {
		t.Fatal(err)
	}
	if repository.libraryID != 7 || repository.path != "albums/2026" ||
		repository.debounce != ReconcileDebounce ||
		repository.maximum != ReconcileMaximumDebounce ||
		waker.count != 1 {
		t.Fatalf("admission = %#v wakes=%d", repository, waker.count)
	}
	for _, target := range []string{"/absolute", "../escape", "a/../escape", "a//b"} {
		if _, err := service.MarkDirty(context.Background(), 7, target); err == nil {
			t.Errorf("unsafe target %q unexpectedly accepted", target)
		}
	}
	if waker.count != 1 {
		t.Fatalf("invalid targets woke worker %d times", waker.count)
	}
}
