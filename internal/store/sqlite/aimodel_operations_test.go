package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
)

func TestAIOperationStateMachinePersistsCASAndCancellation(t *testing.T) {
	store, _ := openTestStore(t)
	now := time.Date(2026, 8, 27, 14, 0, 0, 0, time.UTC)
	service, err := aimodel.NewOperationService(store, func() time.Time { return now }, func() (string, error) {
		return "aio_persisted_operation", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	total := int64(3)
	operation, err := service.Create(context.Background(), aimodel.OperationModelInstall, "", 0, &total)
	if err != nil || operation.State != aimodel.OperationQueued || operation.Revision != 1 {
		t.Fatalf("create = %#v, %v", operation, err)
	}
	now = now.Add(time.Second)
	operation, err = service.Start(context.Background(), operation.ID, operation.Revision, aimodel.PhaseVerifying)
	if err != nil || operation.State != aimodel.OperationRunning || operation.Revision != 2 {
		t.Fatalf("start = %#v, %v", operation, err)
	}
	completed := int64(1)
	now = now.Add(time.Second)
	operation, err = service.Progress(context.Background(), operation.ID, operation.Revision, aimodel.PhaseCopying, completed, &total)
	if err != nil || operation.CompletedItems != 1 || operation.Revision != 3 {
		t.Fatalf("progress = %#v, %v", operation, err)
	}
	if _, err := service.RequestCancel(context.Background(), operation.ID, 2); !errors.Is(err, aimodel.ErrPreconditionFailed) {
		t.Fatalf("stale cancel error = %v", err)
	}
	now = now.Add(time.Second)
	operation, err = service.RequestCancel(context.Background(), operation.ID, operation.Revision)
	if err != nil || operation.State != aimodel.OperationCancelling || operation.Revision != 4 {
		t.Fatalf("request cancel = %#v, %v", operation, err)
	}
	now = now.Add(time.Second)
	operation, err = service.FinishCancelled(context.Background(), operation.ID, operation.Revision)
	if err != nil || operation.State != aimodel.OperationCancelled || operation.FinishedAt == nil || operation.ErrorCode != "cancelled" {
		t.Fatalf("finish cancel = %#v, %v", operation, err)
	}
}

func TestAIOperationLifecycleClampsTimeAcrossClockRollback(t *testing.T) {
	store, _ := openTestStore(t)
	now := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	service, err := aimodel.NewOperationService(store, func() time.Time { return now }, func() (string, error) {
		return "aio_clock_rollback", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	total := int64(2)
	operation, err := service.Create(context.Background(), aimodel.OperationModelInstall, "", 0, &total)
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(-time.Minute)
	operation, err = service.Start(context.Background(), operation.ID, operation.Revision, aimodel.PhaseVerifying)
	if err != nil || operation.UpdatedAt.Before(operation.CreatedAt) {
		t.Fatalf("start=%+v err=%v", operation, err)
	}
	now = now.Add(-time.Minute)
	operation, err = service.RequestCancel(context.Background(), operation.ID, operation.Revision)
	if err != nil || operation.UpdatedAt.Before(operation.CreatedAt) {
		t.Fatalf("cancel=%+v err=%v", operation, err)
	}
	now = now.Add(-time.Minute)
	operation, err = service.FinishCancelled(context.Background(), operation.ID, operation.Revision)
	if err != nil || operation.UpdatedAt.Before(operation.CreatedAt) || operation.FinishedAt == nil || operation.FinishedAt.Before(operation.CreatedAt) {
		t.Fatalf("finish=%+v err=%v", operation, err)
	}
}

func TestAIOperationRecoveryFailsInterruptedWithoutTouchingTerminal(t *testing.T) {
	store, _ := openTestStore(t)
	now := time.Date(2026, 8, 27, 15, 0, 0, 0, time.UTC)
	ids := []string{"aio_interrupted", "aio_terminal"}
	service, err := aimodel.NewOperationService(store, func() time.Time { return now }, func() (string, error) {
		id := ids[0]
		ids = ids[1:]
		return id, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	interrupted, err := service.Create(context.Background(), aimodel.OperationModelInstall, "", 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	interrupted, err = service.Start(context.Background(), interrupted.ID, interrupted.Revision, aimodel.PhaseVerifying)
	if err != nil {
		t.Fatal(err)
	}
	terminal, err := service.Create(context.Background(), aimodel.OperationModelInstall, "", 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	terminal, err = service.Fail(context.Background(), terminal.ID, terminal.Revision, "model_incompatible")
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Minute)
	count, err := service.RecoverInterrupted(context.Background())
	if err != nil || count != 1 {
		t.Fatalf("recover = %d, %v", count, err)
	}
	interrupted, err = service.Get(context.Background(), interrupted.ID)
	if err != nil || interrupted.State != aimodel.OperationFailed || interrupted.ErrorCode != "operation_interrupted" {
		t.Fatalf("interrupted = %#v, %v", interrupted, err)
	}
	reloadedTerminal, err := service.Get(context.Background(), terminal.ID)
	if err != nil || reloadedTerminal.Revision != terminal.Revision || reloadedTerminal.ErrorCode != "model_incompatible" {
		t.Fatalf("terminal = %#v, %v", reloadedTerminal, err)
	}
}

func TestAIOperationRecoveryLeavesRestartSafeSemanticWorkToSemanticQueue(t *testing.T) {
	store, _ := openTestStore(t)
	now := time.Date(2026, 8, 27, 16, 0, 0, 0, time.UTC)
	service, err := aimodel.NewOperationService(store, func() time.Time { return now }, func() (string, error) {
		return "aio_semantic_restart_safe", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	operation, err := service.Create(context.Background(), aimodel.OperationSemanticMissing, "", 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	operation, err = service.Start(context.Background(), operation.ID, operation.Revision, aimodel.PhaseBuilding)
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Minute)
	count, err := service.RecoverInterrupted(context.Background())
	if err != nil || count != 0 {
		t.Fatalf("generic recovery = %d, %v", count, err)
	}
	stored, err := service.Get(context.Background(), operation.ID)
	if err != nil || stored.State != aimodel.OperationRunning || stored.Revision != operation.Revision {
		t.Fatalf("semantic operation = %#v, %v", stored, err)
	}
}
