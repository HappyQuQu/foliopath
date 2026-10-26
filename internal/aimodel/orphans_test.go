package aimodel

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type managedOrphanSourceStub struct {
	invalid map[string]bool
	calls   int
}

func (stub *managedOrphanSourceStub) ValidateManagedModelPackage(_ context.Context, model Model, manifest Manifest) error {
	stub.calls++
	if manifest.PackageID != model.Package.PackageID || stub.invalid[model.Package.ContentHash] {
		return ErrModelIncompatible
	}
	return nil
}

func (*managedOrphanSourceStub) OpenManagedRuntimeModelFile(context.Context, Model, string) (RuntimeModelFile, error) {
	return nil, errors.New("unused")
}

func TestManagedOrphanServiceRegistersOnlyExactReviewedFinals(t *testing.T) {
	catalog, _, _ := catalogFixture(t)
	repository := &memoryRepository{snapshot: Snapshot{Revision: 1}}
	ids := []string{"aim_reconciled_orphan", "aim_replayed_orphan"}
	models, err := NewService(repository, func() time.Time {
		return time.Date(2026, 8, 28, 23, 0, 0, 0, time.UTC)
	}, func() (string, error) {
		id := ids[0]
		ids = ids[1:]
		return id, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	source := &managedOrphanSourceStub{invalid: map[string]bool{}}
	service, err := NewManagedOrphanService(models, catalog, source, "arm64")
	if err != nil {
		t.Fatal(err)
	}
	reviewedHash := strings.Repeat("d", 64)
	unknownHash := strings.Repeat("e", 64)

	incomplete, err := service.Reconcile(context.Background(), []string{reviewedHash, unknownHash}, false)
	if err != nil || !incomplete.Truncated || incomplete.Registered != 0 || source.calls != 0 || len(repository.snapshot.Items) != 0 {
		t.Fatalf("incomplete report = %#v calls=%d models=%d err=%v", incomplete, source.calls, len(repository.snapshot.Items), err)
	}
	complete, err := service.Reconcile(context.Background(), []string{reviewedHash, unknownHash}, true)
	if err != nil || complete.Scanned != 2 || complete.Reviewed != 1 || complete.Registered != 1 ||
		complete.Unrecognized != 1 || complete.Invalid != 0 || len(repository.snapshot.Items) != 1 {
		t.Fatalf("complete report = %#v models=%#v err=%v", complete, repository.snapshot.Items, err)
	}
	registered := repository.snapshot.Items[0]
	if registered.Active || registered.State != StateAvailable || registered.StorageMode != StorageManaged ||
		registered.SourceIdentity != "managed:"+reviewedHash {
		t.Fatalf("registered orphan = %#v", registered)
	}
	replayed, err := service.Reconcile(context.Background(), []string{reviewedHash}, true)
	if err != nil || replayed.Registered != 0 || len(repository.snapshot.Items) != 1 {
		t.Fatalf("replayed report = %#v models=%d err=%v", replayed, len(repository.snapshot.Items), err)
	}
}

func TestManagedOrphanServiceRejectsMalformedReportAndLeavesInvalidFinalUnregistered(t *testing.T) {
	catalog, _, _ := catalogFixture(t)
	repository := &memoryRepository{snapshot: Snapshot{Revision: 1}}
	models, err := NewService(repository, nil, func() (string, error) { return "aim_invalid_orphan", nil })
	if err != nil {
		t.Fatal(err)
	}
	reviewedHash := strings.Repeat("d", 64)
	source := &managedOrphanSourceStub{invalid: map[string]bool{reviewedHash: true}}
	service, err := NewManagedOrphanService(models, catalog, source, "arm64")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Reconcile(context.Background(), []string{reviewedHash, reviewedHash}, true); !errors.Is(err, ErrRepositoryState) {
		t.Fatalf("duplicate report error = %v", err)
	}
	result, err := service.Reconcile(context.Background(), []string{reviewedHash}, true)
	if err != nil || result.Invalid != 1 || result.Registered != 0 || len(repository.snapshot.Items) != 0 {
		t.Fatalf("invalid final = %#v models=%d err=%v", result, len(repository.snapshot.Items), err)
	}
}
