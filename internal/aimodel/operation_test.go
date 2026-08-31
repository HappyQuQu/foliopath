package aimodel

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestValidateOperationRejectsInconsistentPersistedState(t *testing.T) {
	now := time.Date(2026, 8, 28, 10, 0, 0, 0, time.UTC)
	finished := now.Add(time.Second)
	validQueued := Operation{
		ID: "aio_operation123", Kind: OperationModelInstall, State: OperationQueued, Phase: PhaseQueued,
		Revision: 1, CreatedAt: now, UpdatedAt: now,
	}
	validFailed := validQueued
	validFailed.State, validFailed.Phase = OperationFailed, PhaseCompleted
	validFailed.ErrorCode, validFailed.FinishedAt, validFailed.UpdatedAt = "internal_error", &finished, finished

	tests := []struct {
		name      string
		operation Operation
	}{
		{name: "queued with completed phase", operation: mutateOperation(validQueued, func(value *Operation) { value.Phase = PhaseCompleted })},
		{name: "running with queued phase", operation: mutateOperation(validQueued, func(value *Operation) { value.State = OperationRunning })},
		{name: "active with error code", operation: mutateOperation(validQueued, func(value *Operation) {
			value.State, value.Phase, value.ErrorCode = OperationRunning, PhaseCopying, "internal_error"
		})},
		{name: "succeeded with error code", operation: mutateOperation(validFailed, func(value *Operation) {
			value.State = OperationSucceeded
		})},
		{name: "failed without error code", operation: mutateOperation(validFailed, func(value *Operation) { value.ErrorCode = "" })},
		{name: "cancelled without error code", operation: mutateOperation(validFailed, func(value *Operation) {
			value.State, value.ErrorCode = OperationCancelled, ""
		})},
		{name: "oversized error code", operation: mutateOperation(validFailed, func(value *Operation) {
			value.ErrorCode = string(make([]byte, 129))
		})},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := validateOperation(test.operation); !errors.Is(err, ErrRepositoryState) {
				t.Fatalf("validateOperation() error = %v", err)
			}
		})
	}
	if err := validateOperation(validQueued); err != nil {
		t.Fatalf("valid queued operation: %v", err)
	}
	if err := validateOperation(validFailed); err != nil {
		t.Fatalf("valid failed operation: %v", err)
	}
}

func TestOperationServiceValidatesRepositoryWriteResults(t *testing.T) {
	now := time.Date(2026, 8, 28, 11, 0, 0, 0, time.UTC)
	t.Run("create", func(t *testing.T) {
		repository := &operationWriteResultRepository{corruptCreate: true}
		service, err := NewOperationService(repository, func() time.Time { return now }, func() (string, error) {
			return "aio_operation123", nil
		})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := service.Create(context.Background(), OperationModelInstall, "", 0, nil); !errors.Is(err, ErrRepositoryState) {
			t.Fatalf("Create() error = %v", err)
		}
	})

	t.Run("create identity", func(t *testing.T) {
		repository := &operationWriteResultRepository{replaceCreateID: true}
		service, err := NewOperationService(repository, func() time.Time { return now }, func() (string, error) {
			return "aio_operation123", nil
		})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := service.Create(context.Background(), OperationModelInstall, "", 0, nil); !errors.Is(err, ErrRepositoryState) {
			t.Fatalf("Create() error = %v", err)
		}
	})

	t.Run("get identity", func(t *testing.T) {
		repository := &operationWriteResultRepository{operation: Operation{
			ID: "aio_different123", Kind: OperationModelInstall, State: OperationQueued, Phase: PhaseQueued,
			Revision: 1, CreatedAt: now, UpdatedAt: now,
		}}
		service, err := NewOperationService(repository, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := service.Get(context.Background(), "aio_operation123"); !errors.Is(err, ErrRepositoryState) {
			t.Fatalf("Get() error = %v", err)
		}
	})

	t.Run("transition", func(t *testing.T) {
		repository := &operationWriteResultRepository{operation: Operation{
			ID: "aio_operation123", Kind: OperationModelInstall, State: OperationQueued, Phase: PhaseQueued,
			Revision: 1, CreatedAt: now, UpdatedAt: now,
		}, corruptTransition: true}
		service, err := NewOperationService(repository, func() time.Time { return now.Add(time.Second) }, nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := service.Start(context.Background(), "aio_operation123", 1, PhaseCopying); !errors.Is(err, ErrRepositoryState) {
			t.Fatalf("Start() error = %v", err)
		}
	})

	t.Run("transition identity", func(t *testing.T) {
		repository := &operationWriteResultRepository{operation: Operation{
			ID: "aio_operation123", Kind: OperationModelInstall, State: OperationQueued, Phase: PhaseQueued,
			Revision: 1, CreatedAt: now, UpdatedAt: now,
		}, replaceTransitionID: true}
		service, err := NewOperationService(repository, func() time.Time { return now.Add(time.Second) }, nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := service.Start(context.Background(), "aio_operation123", 1, PhaseCopying); !errors.Is(err, ErrRepositoryState) {
			t.Fatalf("Start() error = %v", err)
		}
	})
}

type operationWriteResultRepository struct {
	operation           Operation
	corruptCreate       bool
	corruptTransition   bool
	replaceCreateID     bool
	replaceTransitionID bool
}

func (repository *operationWriteResultRepository) CreateAIOperation(_ context.Context, operation Operation) (Operation, error) {
	repository.operation = operation
	if repository.corruptCreate {
		repository.operation.Phase = PhaseCompleted
	}
	if repository.replaceCreateID {
		repository.operation.ID = "aio_different123"
	}
	return repository.operation, nil
}

func (repository *operationWriteResultRepository) GetAIOperation(context.Context, string) (Operation, error) {
	return repository.operation, nil
}

func (repository *operationWriteResultRepository) TransitionAIOperation(
	_ context.Context,
	_ string,
	transition OperationTransition,
) (Operation, error) {
	repository.operation.State = transition.State
	repository.operation.Phase = transition.Phase
	repository.operation.CompletedItems = transition.CompletedItems
	repository.operation.TotalItems = transition.TotalItems
	repository.operation.ErrorCode = transition.ErrorCode
	repository.operation.FinishedAt = transition.FinishedAt
	repository.operation.UpdatedAt = transition.UpdatedAt
	repository.operation.Revision++
	if repository.corruptTransition {
		repository.operation.Phase = PhaseQueued
	}
	if repository.replaceTransitionID {
		repository.operation.ID = "aio_different123"
	}
	return repository.operation, nil
}

func (*operationWriteResultRepository) RecoverInterruptedAIOperations(context.Context, time.Time) (int64, error) {
	return 0, nil
}

func mutateOperation(value Operation, mutate func(*Operation)) Operation {
	mutate(&value)
	return value
}
