package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
	"github.com/HappyQuQu/foliopath/internal/face"
)

func TestFaceControlDerivesGenerationAndReturnsCanonicalOperation(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	now := time.Date(2026, 9, 2, 16, 0, 0, 0, time.UTC)
	seedFaceGeneration(t, store, "face_generation_control", 2, "active", now)
	seedFaceReadySettings(t, store, libraryID, "face_generation_control", now)

	settings, err := face.NewSettingsService(store, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	waker := &faceClearWaker{}
	jobs, err := face.NewJobService(store, store, waker, func() time.Time { return now }, faceClearIDs())
	if err != nil {
		t.Fatal(err)
	}
	clears, err := face.NewClearService(store, waker, func() time.Time { return now }, faceClearIDs())
	if err != nil {
		t.Fatal(err)
	}
	operations, err := aimodel.NewOperationService(store, func() time.Time { return now }, nil)
	if err != nil {
		t.Fatal(err)
	}
	control, err := face.NewControlService(settings, jobs, clears, operations)
	if err != nil {
		t.Fatal(err)
	}

	operation, replayed, err := control.RequestFaceJob(context.Background(), libraryID, face.JobMissing, "face-control-job-key-001")
	if err != nil || replayed || operation.Kind != aimodel.OperationFaceMissing || operation.LibraryID != libraryID || operation.State != aimodel.OperationQueued {
		t.Fatalf("operation=%+v replayed=%t err=%v", operation, replayed, err)
	}
	replay, replayed, err := control.RequestFaceJob(context.Background(), libraryID, face.JobMissing, "face-control-job-key-001")
	if err != nil || !replayed || replay.ID != operation.ID {
		t.Fatalf("replay=%+v replayed=%t err=%v", replay, replayed, err)
	}
	cancelled, err := control.CancelFaceOperation(context.Background(), operation.ID, operation.Revision)
	if err != nil || cancelled.ID != operation.ID || cancelled.State != aimodel.OperationCancelled || cancelled.Revision != operation.Revision+1 {
		t.Fatalf("cancelled=%+v err=%v", cancelled, err)
	}
}

func TestFaceControlRejectsDisabledAdmissionAndReturnsClearOperation(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	now := time.Date(2026, 9, 2, 17, 0, 0, 0, time.UTC)
	settings, _ := face.NewSettingsService(store, func() time.Time { return now })
	waker := &faceClearWaker{}
	jobs, _ := face.NewJobService(store, store, waker, func() time.Time { return now }, faceClearIDs())
	clears, _ := face.NewClearService(store, waker, func() time.Time { return now }, faceClearIDs())
	operations, _ := aimodel.NewOperationService(store, func() time.Time { return now }, nil)
	control, err := face.NewControlService(settings, jobs, clears, operations)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := control.RequestFaceJob(context.Background(), libraryID, face.JobMissing, "face-control-disabled-001"); !errors.Is(err, face.ErrFaceDisabled) {
		t.Fatalf("disabled err=%v", err)
	}
	operation, replayed, err := control.RequestManualFaceClear(context.Background(), libraryID, 1, "face-control-clear-001", face.ManualClearCounts{})
	if err != nil || replayed || operation.Kind != aimodel.OperationFaceManualClear || operation.LibraryID != libraryID {
		t.Fatalf("clear=%+v replayed=%t err=%v", operation, replayed, err)
	}
	cancelled, err := control.CancelFaceOperation(context.Background(), operation.ID, operation.Revision)
	if err != nil || cancelled.ID != operation.ID || cancelled.State != aimodel.OperationCancelled || cancelled.Revision != operation.Revision+1 {
		t.Fatalf("clear cancelled=%+v err=%v", cancelled, err)
	}
}
