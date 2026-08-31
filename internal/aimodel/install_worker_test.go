package aimodel

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type installExecutorStub struct {
	err   error
	calls int
}

func (stub *installExecutorStub) Install(context.Context, Candidate, StorageMode) (Model, bool, error) {
	stub.calls++
	return Model{}, stub.err == nil, stub.err
}

func TestInstallErrorCodePreservesKernelSpaceExhaustionMapping(t *testing.T) {
	err := errors.Join(ErrInsufficientSpace, errors.New("kernel space exhausted"))
	if code := installErrorCode(err); code != "insufficient_space" {
		t.Fatalf("install error code = %q", code)
	}
}

type installClaimQueue struct {
	InstallQueue
	work InstallWork
}

func (queue installClaimQueue) ClaimAIModelInstall(context.Context, time.Time) (InstallWork, bool, error) {
	return queue.work, true, nil
}

func TestInstallWorkerRejectsCorruptClaimBeforeInstaller(t *testing.T) {
	now := time.Date(2026, 8, 29, 11, 0, 0, 0, time.UTC)
	operation := Operation{ID: "aio_install_claim", Kind: OperationModelInstall, State: OperationRunning,
		Phase: PhaseVerifying, Revision: 2, CreatedAt: now, UpdatedAt: now}
	executor := &installExecutorStub{}
	operations, err := NewOperationService(&installFinalizationRepository{operation: operation}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	worker, err := NewInstallWorker(installClaimQueue{work: InstallWork{
		IdempotencyKey: "install-request-123", RequestHash: "corrupt", CandidateID: "aic_claim",
		StorageMode: StorageManaged, Operation: operation,
	}}, executor, operations, make(chan struct{}), 10*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if err := worker.Run(context.Background()); !errors.Is(err, ErrRepositoryState) {
		t.Fatalf("Run() error = %v", err)
	}
	if executor.calls != 0 {
		t.Fatalf("installer calls = %d, want 0", executor.calls)
	}
}

type installQueueUnused struct{}

func (installQueueUnused) FindAIModelInstall(context.Context, string) (InstallWork, bool, error) {
	return InstallWork{}, false, nil
}
func (installQueueUnused) CreateAIModelInstall(context.Context, InstallWork) (InstallWork, bool, error) {
	return InstallWork{}, false, nil
}
func (installQueueUnused) ClaimAIModelInstall(context.Context, time.Time) (InstallWork, bool, error) {
	return InstallWork{}, false, nil
}

type installFinalizationRepository struct {
	mu             sync.Mutex
	operation      Operation
	failTransition int
	cancelOnFail   bool
}

func (*installFinalizationRepository) CreateAIOperation(context.Context, Operation) (Operation, error) {
	return Operation{}, errors.New("not used")
}
func (repository *installFinalizationRepository) GetAIOperation(context.Context, string) (Operation, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	return repository.operation, nil
}
func (repository *installFinalizationRepository) TransitionAIOperation(_ context.Context, _ string, transition OperationTransition) (Operation, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	if repository.operation.Revision != transition.ExpectedRevision {
		return Operation{}, ErrPreconditionFailed
	}
	if repository.failTransition > 0 {
		repository.failTransition--
		if repository.cancelOnFail {
			repository.operation.State = OperationCancelling
			repository.operation.Revision++
		}
		return Operation{}, ErrPreconditionFailed
	}
	repository.operation.State = transition.State
	repository.operation.Phase = transition.Phase
	repository.operation.CompletedItems = transition.CompletedItems
	repository.operation.TotalItems = cloneInt64(transition.TotalItems)
	repository.operation.ErrorCode = transition.ErrorCode
	repository.operation.FinishedAt = transition.FinishedAt
	repository.operation.UpdatedAt = transition.UpdatedAt
	repository.operation.Revision++
	return repository.operation, nil
}
func (*installFinalizationRepository) RecoverInterruptedAIOperations(context.Context, time.Time) (int64, error) {
	return 0, nil
}

func TestInstallWorkerConvergesConcurrentCancellationDuringFinalization(t *testing.T) {
	repository, operations, worker, work := installFinalizationFixture(t, true)
	worker.process(context.Background(), work)
	result, err := operations.Get(context.Background(), work.Operation.ID)
	if err != nil {
		t.Fatal(err)
	}
	if result.State != OperationCancelled || result.ErrorCode != "cancelled" || result.FinishedAt == nil {
		t.Fatalf("operation = %#v", result)
	}
	if repository.failTransition != 0 {
		t.Fatalf("unused injected transition failures = %d", repository.failTransition)
	}
}

func TestInstallWorkerFailsInsteadOfLeavingRunningAfterFinalizationError(t *testing.T) {
	_, operations, worker, work := installFinalizationFixture(t, false)
	worker.process(context.Background(), work)
	result, err := operations.Get(context.Background(), work.Operation.ID)
	if err != nil {
		t.Fatal(err)
	}
	if result.State != OperationFailed || result.ErrorCode != "internal_error" || result.FinishedAt == nil {
		t.Fatalf("operation = %#v", result)
	}
}

func TestWorkerFailureConvergesConcurrentCancellation(t *testing.T) {
	repository, operations, _, work := installFinalizationFixture(t, true)
	settleWorkerFailure(context.Background(), operations, work.Operation.ID, "model_incompatible")
	result, err := operations.Get(context.Background(), work.Operation.ID)
	if err != nil {
		t.Fatal(err)
	}
	if result.State != OperationCancelled || result.ErrorCode != "cancelled" || result.FinishedAt == nil {
		t.Fatalf("operation = %#v", result)
	}
	if repository.failTransition != 0 {
		t.Fatalf("unused injected transition failures = %d", repository.failTransition)
	}
}

func installFinalizationFixture(t *testing.T, cancelOnFail bool) (*installFinalizationRepository, *OperationService, *InstallWorker, InstallWork) {
	t.Helper()
	now := time.Date(2026, 8, 28, 22, 0, 0, 0, time.UTC)
	operation := Operation{ID: "aio_install_finalize", Kind: OperationModelInstall, State: OperationRunning,
		Phase: PhaseVerifying, Revision: 2, CreatedAt: now, UpdatedAt: now}
	repository := &installFinalizationRepository{operation: operation, failTransition: 1, cancelOnFail: cancelOnFail}
	operations, err := NewOperationService(repository, func() time.Time { return now.Add(time.Second) }, nil)
	if err != nil {
		t.Fatal(err)
	}
	worker, err := NewInstallWorker(installQueueUnused{}, &installExecutorStub{}, operations, make(chan struct{}), 10*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	return repository, operations, worker, InstallWork{Operation: operation, StorageMode: StorageManaged}
}
