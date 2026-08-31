package aimodel

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestAvailabilityRefreshMarksMissingSourceAndExactRecovery(t *testing.T) {
	catalog, manifest, _ := catalogFixture(t)
	now := time.Date(2026, 8, 28, 18, 0, 0, 0, time.UTC)
	model := Model{
		ID: "aim_availability", Package: testPackage(), StorageMode: StorageDirect,
		State: StateAvailable, SourceIdentity: "source:reviewed", AvailabilityRevision: 1,
		CreatedAt: now, UpdatedAt: now,
	}
	model.Package.PackageID = manifest.PackageID
	repository := &memoryRepository{snapshot: Snapshot{Revision: 1, Items: []Model{model}}}
	models, err := NewService(repository, func() time.Time { now = now.Add(time.Second); return now }, nil)
	if err != nil {
		t.Fatal(err)
	}
	source := &activationSourceStub{err: ErrModelSourceUnavailable}
	service, err := NewAvailabilityService(models, catalog, source)
	if err != nil {
		t.Fatal(err)
	}

	summary, err := service.Refresh(context.Background())
	if err != nil || summary.Checked != 1 || summary.Changed != 1 || summary.Unavailable != 1 {
		t.Fatalf("unavailable refresh = %#v, %v", summary, err)
	}
	stored, err := models.Get(context.Background(), model.ID)
	if err != nil || stored.State != StateUnavailable || stored.AvailabilityRevision != 2 {
		t.Fatalf("unavailable model = %#v, %v", stored, err)
	}

	source.err = nil
	summary, err = service.Refresh(context.Background())
	if err != nil || summary.Checked != 1 || summary.Changed != 1 || summary.Unavailable != 0 {
		t.Fatalf("recovery refresh = %#v, %v", summary, err)
	}
	stored, err = models.Get(context.Background(), model.ID)
	if err != nil || stored.State != StateAvailable || stored.AvailabilityRevision != 3 {
		t.Fatalf("recovered model = %#v, %v", stored, err)
	}
}

func TestAvailabilityRefreshFailsClosedForUnreviewedPackageAndHonorsCancellation(t *testing.T) {
	catalog, _, _ := catalogFixture(t)
	now := time.Date(2026, 8, 28, 18, 30, 0, 0, time.UTC)
	model := Model{
		ID: "aim_unreviewed", Package: testPackage(), StorageMode: StorageManaged,
		State: StateAvailable, SourceIdentity: "managed:reviewed", AvailabilityRevision: 1,
		CreatedAt: now, UpdatedAt: now,
	}
	model.Package.PackageID = "removed-from-catalog"
	repository := &memoryRepository{snapshot: Snapshot{Revision: 1, Items: []Model{model}}}
	models, _ := NewService(repository, func() time.Time { now = now.Add(time.Second); return now }, nil)
	service, _ := NewAvailabilityService(models, catalog, &activationSourceStub{})
	summary, err := service.Refresh(context.Background())
	if err != nil || summary.Changed != 1 || summary.Unavailable != 1 {
		t.Fatalf("unreviewed refresh = %#v, %v", summary, err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := service.Refresh(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel error = %v", err)
	}
}

func TestAvailabilityMarkUnavailableIsCASBoundAndIdempotent(t *testing.T) {
	catalog, manifest, _ := catalogFixture(t)
	now := time.Date(2026, 8, 28, 19, 0, 0, 0, time.UTC)
	model := Model{
		ID: "aim_mark_unavailable", Package: testPackage(), StorageMode: StorageDirect,
		State: StateAvailable, SourceIdentity: "source:reviewed", AvailabilityRevision: 1,
		CreatedAt: now, UpdatedAt: now,
	}
	model.Package.PackageID = manifest.PackageID
	repository := &memoryRepository{snapshot: Snapshot{Revision: 1, Items: []Model{model}}}
	models, _ := NewService(repository, func() time.Time { now = now.Add(time.Second); return now }, nil)
	service, _ := NewAvailabilityService(models, catalog, &activationSourceStub{})
	if err := service.MarkUnavailable(context.Background(), model.ID, 1); err != nil {
		t.Fatal(err)
	}
	if err := service.MarkUnavailable(context.Background(), model.ID, 1); err != nil {
		t.Fatalf("stale idempotent mark = %v", err)
	}
	stored, err := models.Get(context.Background(), model.ID)
	if err != nil || stored.State != StateUnavailable || stored.AvailabilityRevision != 2 {
		t.Fatalf("stored = %#v, %v", stored, err)
	}
}
