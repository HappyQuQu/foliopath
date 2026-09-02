package aimodel

import (
	"context"
	"errors"
	"testing"
	"time"
)

type memoryRepository struct {
	snapshot Snapshot
}

func (repository *memoryRepository) ListAIModels(context.Context) (Snapshot, error) {
	return repository.snapshot, nil
}

func (repository *memoryRepository) GetAIModel(_ context.Context, id string) (Model, error) {
	for _, model := range repository.snapshot.Items {
		if model.ID == id {
			return model, nil
		}
	}
	return Model{}, ErrModelNotFound
}

func (repository *memoryRepository) RegisterAIModel(_ context.Context, model Model) (Model, bool, error) {
	for _, existing := range repository.snapshot.Items {
		if existing.Package == model.Package {
			return existing, false, nil
		}
	}
	repository.snapshot.Items = append(repository.snapshot.Items, model)
	repository.snapshot.Revision++
	return model, true, nil
}

func (repository *memoryRepository) SetAIModelAvailability(_ context.Context, id string, revision int64, state State, now time.Time) (Model, error) {
	for index := range repository.snapshot.Items {
		model := &repository.snapshot.Items[index]
		if model.ID != id {
			continue
		}
		if model.AvailabilityRevision != revision {
			return Model{}, ErrPreconditionFailed
		}
		if model.State != state {
			model.State = state
			model.AvailabilityRevision++
			model.UpdatedAt = now
			repository.snapshot.Revision++
		}
		return *model, nil
	}
	return Model{}, ErrModelNotFound
}

func testPackage() VerifiedPackage {
	return VerifiedPackage{
		PackageID:       "semantic-test-v1",
		Purpose:         PurposeSemanticImageText,
		Version:         "1.0.0",
		Architecture:    "arm64",
		ContentHash:     "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		LicenseID:       "Apache-2.0",
		PackageSizeByte: 1024,
	}
}

func TestServiceRegistersIdempotentlyAndTracksAvailabilityRevision(t *testing.T) {
	now := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)
	repository := &memoryRepository{snapshot: Snapshot{Revision: 1}}
	ids := []string{"aim_first", "aim_second"}
	service, err := NewService(repository, func() time.Time { return now }, func() (string, error) {
		value := ids[0]
		ids = ids[1:]
		return value, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	created, wasCreated, err := service.RegisterInstalled(context.Background(), testPackage(), StorageManaged, "managed:sha256:test")
	if err != nil || !wasCreated || created.ID != "aim_first" {
		t.Fatalf("register = %#v, %v, %v", created, wasCreated, err)
	}
	repeated, wasCreated, err := service.RegisterInstalled(context.Background(), testPackage(), StorageManaged, "managed:sha256:test")
	if err != nil || wasCreated || repeated.ID != created.ID {
		t.Fatalf("repeat = %#v, %v, %v", repeated, wasCreated, err)
	}
	now = now.Add(time.Minute)
	unavailable, err := service.SetAvailability(context.Background(), created.ID, 1, false)
	if err != nil || unavailable.State != StateUnavailable || unavailable.AvailabilityRevision != 2 {
		t.Fatalf("unavailable = %#v, %v", unavailable, err)
	}
	if _, err := service.SetAvailability(context.Background(), created.ID, 1, true); !errors.Is(err, ErrPreconditionFailed) {
		t.Fatalf("stale update error = %v", err)
	}
}

func TestServiceRejectsInvalidPackageAndRepositoryState(t *testing.T) {
	repository := &memoryRepository{snapshot: Snapshot{Revision: 0}}
	service, err := NewService(repository, nil, func() (string, error) { return "aim_test", nil })
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.List(context.Background()); !errors.Is(err, ErrRepositoryState) {
		t.Fatalf("list error = %v", err)
	}
	invalid := testPackage()
	invalid.ContentHash = "not-a-hash"
	if _, _, err := service.RegisterInstalled(context.Background(), invalid, StorageManaged, "source"); !errors.Is(err, ErrInvalidModel) {
		t.Fatalf("register invalid error = %v", err)
	}
}

func TestServiceAcceptsOneActiveModelPerPurpose(t *testing.T) {
	now := time.Date(2026, 9, 1, 23, 0, 0, 0, time.UTC)
	semantic := Model{ID: "aim_semantic_active", Package: testPackage(), StorageMode: StorageManaged,
		State: StateAvailable, SourceIdentity: "managed:semantic", AvailabilityRevision: 1,
		CreatedAt: now, UpdatedAt: now, Active: true}
	facePackage := testPackage()
	facePackage.PackageID = "face-package-v3"
	facePackage.Purpose = PurposeFaceDetectionEmbedding
	facePackage.LicenseID = "Apache-2.0 AND MIT"
	face := Model{ID: "aim_face_active", Package: facePackage, StorageMode: StorageManaged,
		State: StateAvailable, SourceIdentity: "managed:face", AvailabilityRevision: 1,
		CreatedAt: now, UpdatedAt: now, Active: true}
	repository := &memoryRepository{snapshot: Snapshot{
		Items: []Model{semantic, face}, ActiveModelID: semantic.ID, ActiveFaceModelID: face.ID, Revision: 3,
	}}
	service, err := NewService(repository, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := service.List(context.Background())
	if err != nil || snapshot.ActiveModelID != semantic.ID || snapshot.ActiveFaceModelID != face.ID {
		t.Fatalf("snapshot=%+v err=%v", snapshot, err)
	}
	repository.snapshot.ActiveFaceModelID = semantic.ID
	if _, err := service.List(context.Background()); !errors.Is(err, ErrRepositoryState) {
		t.Fatalf("cross-purpose active error=%v", err)
	}
}
