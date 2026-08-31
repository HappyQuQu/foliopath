package aimodel

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

type activationSourceStub struct {
	err    error
	opened []string
}

func (stub *activationSourceStub) ValidateActivationSource(context.Context, Model, Manifest) error {
	return stub.err
}
func (stub *activationSourceStub) OpenActivationModelFile(_ context.Context, _ Model, name string) (RuntimeModelFile, error) {
	stub.opened = append(stub.opened, name)
	return &runtimeModelFileStub{ReadCloser: io.NopCloser(strings.NewReader("x")), path: "/proc/self/fd/7", size: 1}, nil
}

type inferenceRuntimeStub struct {
	metadata RuntimeMetadata
	called   bool
}

func (stub *inferenceRuntimeStub) LoadAndValidate(ctx context.Context, model Model, manifest Manifest, opener RuntimeFileOpener) (RuntimeMetadata, error) {
	stub.called = true
	if model.ID == "" || len(manifest.Files) == 0 {
		return RuntimeMetadata{}, ErrModelIncompatible
	}
	reader, err := opener(ctx, manifest.Files[0].Name)
	if err != nil {
		return RuntimeMetadata{}, err
	}
	_ = reader.Close()
	return stub.metadata, nil
}

type runtimeModelFileStub struct {
	io.ReadCloser
	path string
	size int64
}

func (file *runtimeModelFileStub) RuntimePath() string { return file.path }
func (file *runtimeModelFileStub) Size() int64         { return file.size }

func TestActivationWorkerLoadsOnlyAvailableReviewedModelThroughSourcePort(t *testing.T) {
	catalog, manifest, _ := catalogFixture(t)
	now := time.Date(2026, 8, 27, 20, 0, 0, 0, time.UTC)
	model := Model{ID: "aim_activation_worker", Package: testPackage(), StorageMode: StorageManaged,
		State: StateAvailable, SourceIdentity: "managed:test", AvailabilityRevision: 4, CreatedAt: now, UpdatedAt: now}
	model.Package.PackageID = manifest.PackageID
	repository := &memoryRepository{snapshot: Snapshot{Revision: 1, Items: []Model{model}}}
	models, err := NewService(repository, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	operations, err := NewOperationService(managementOperationRepository{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	source := &activationSourceStub{}
	runtime := &inferenceRuntimeStub{metadata: RuntimeMetadata{EmbeddingDimension: 768}}
	availability, err := NewAvailabilityService(models, catalog, source)
	if err != nil {
		t.Fatal(err)
	}
	worker, err := NewActivationWorker(activationRepositoryUnused{}, models, operations, catalog, source, runtime, availability, make(chan struct{}), time.Second, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	metadata, err := worker.load(context.Background(), ActivationWork{ModelID: model.ID, ExpectedAvailabilityRevision: 4})
	if err != nil || metadata.EmbeddingDimension != 768 || !runtime.called || len(source.opened) != 1 {
		t.Fatalf("load = %#v, %v; runtime=%v opened=%v", metadata, err, runtime.called, source.opened)
	}
	if _, err := worker.load(context.Background(), ActivationWork{ModelID: model.ID, ExpectedAvailabilityRevision: 3}); !errors.Is(err, ErrModelUnavailable) {
		t.Fatalf("stale availability error = %v", err)
	}
	source.err = ErrModelSourceUnavailable
	if _, err := worker.load(context.Background(), ActivationWork{ModelID: model.ID, ExpectedAvailabilityRevision: 4}); !errors.Is(err, ErrModelSourceUnavailable) {
		t.Fatalf("source error = %v", err)
	}
	stored, err := models.Get(context.Background(), model.ID)
	if err != nil || stored.State != StateUnavailable || stored.AvailabilityRevision != 5 {
		t.Fatalf("source failure availability = %#v, %v", stored, err)
	}
}

type activationClaimQueue struct {
	ActivationRepository
	work ActivationWork
}

func (queue activationClaimQueue) ClaimAIModelActivation(context.Context, time.Time) (ActivationWork, bool, error) {
	return queue.work, true, nil
}

func TestActivationWorkerRejectsCorruptClaimBeforeRuntime(t *testing.T) {
	catalog, manifest, _ := catalogFixture(t)
	now := time.Date(2026, 8, 29, 11, 0, 0, 0, time.UTC)
	model := Model{ID: "aim_activation_claim", Package: testPackage(), StorageMode: StorageManaged,
		State: StateAvailable, SourceIdentity: "managed:test", AvailabilityRevision: 4, CreatedAt: now, UpdatedAt: now}
	model.Package.PackageID = manifest.PackageID
	models, err := NewService(&memoryRepository{snapshot: Snapshot{Revision: 1, Items: []Model{model}}}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	operation := Operation{ID: "aio_activation_claim", Kind: OperationModelActivate, State: OperationRunning,
		Phase: PhaseLoading, ModelID: model.ID, Revision: 2, CreatedAt: now, UpdatedAt: now}
	operations, err := NewOperationService(&activationOperationStateRepository{operation: operation}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	source := &activationSourceStub{}
	availability, err := NewAvailabilityService(models, catalog, source)
	if err != nil {
		t.Fatal(err)
	}
	runtime := &inferenceRuntimeStub{metadata: RuntimeMetadata{EmbeddingDimension: 768}}
	worker, err := NewActivationWorker(activationClaimQueue{work: ActivationWork{
		IdempotencyKey: "activate-request-123", RequestHash: "corrupt", ModelID: model.ID,
		ExpectedAvailabilityRevision: model.AvailabilityRevision, Operation: operation,
	}}, models, operations, catalog, source, runtime, availability, make(chan struct{}), 10*time.Millisecond, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := worker.Run(context.Background()); !errors.Is(err, ErrRepositoryState) {
		t.Fatalf("Run() error = %v", err)
	}
	if runtime.called || len(source.opened) != 0 {
		t.Fatalf("runtime called = %v, opened = %v", runtime.called, source.opened)
	}
}

func TestActivationWorkerFailsOperationWhenFinalAvailabilityCASIsStale(t *testing.T) {
	catalog, manifest, _ := catalogFixture(t)
	now := time.Date(2026, 8, 27, 20, 30, 0, 0, time.UTC)
	model := Model{ID: "aim_activation_worker_stale", Package: testPackage(), StorageMode: StorageManaged,
		State: StateAvailable, SourceIdentity: "managed:test", AvailabilityRevision: 4, CreatedAt: now, UpdatedAt: now}
	model.Package.PackageID = manifest.PackageID
	models, err := NewService(&memoryRepository{snapshot: Snapshot{Revision: 1, Items: []Model{model}}}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	operation := Operation{ID: "aio_activation_stale", Kind: OperationModelActivate, State: OperationRunning,
		Phase: PhaseLoading, ModelID: model.ID, Revision: 1, CreatedAt: now, UpdatedAt: now}
	operationRepository := &activationOperationStateRepository{operation: operation}
	operations, err := NewOperationService(operationRepository, func() time.Time { return now.Add(time.Second) }, nil)
	if err != nil {
		t.Fatal(err)
	}
	source := &activationSourceStub{}
	availability, err := NewAvailabilityService(models, catalog, source)
	if err != nil {
		t.Fatal(err)
	}
	queue := &activationCommitFailureRepository{err: ErrPreconditionFailed}
	worker, err := NewActivationWorker(queue, models, operations, catalog, source,
		&inferenceRuntimeStub{metadata: RuntimeMetadata{EmbeddingDimension: 768}}, availability,
		make(chan struct{}), 10*time.Millisecond, func() time.Time { return now.Add(time.Second) },
		func() (string, error) { return "aig_stale_worker", nil })
	if err != nil {
		t.Fatal(err)
	}
	worker.process(context.Background(), ActivationWork{ModelID: model.ID, ExpectedAvailabilityRevision: 4, Operation: operation})
	result, err := operations.Get(context.Background(), operation.ID)
	if err != nil {
		t.Fatal(err)
	}
	if queue.commit.ExpectedAvailabilityRevision != 4 || result.State != OperationFailed || result.ErrorCode != "model_unavailable" {
		t.Fatalf("commit=%#v operation=%#v", queue.commit, result)
	}
}

type activationCommitFailureRepository struct {
	commit ActivationCommit
	err    error
}

func (*activationCommitFailureRepository) FindAIModelActivation(context.Context, string) (ActivationWork, bool, error) {
	return ActivationWork{}, false, nil
}
func (*activationCommitFailureRepository) CreateAIModelActivation(context.Context, ActivationWork) (ActivationWork, bool, error) {
	return ActivationWork{}, false, nil
}
func (*activationCommitFailureRepository) ClaimAIModelActivation(context.Context, time.Time) (ActivationWork, bool, error) {
	return ActivationWork{}, false, nil
}
func (repository *activationCommitFailureRepository) CommitAIModelActivation(_ context.Context, commit ActivationCommit) (Operation, error) {
	repository.commit = commit
	return Operation{}, repository.err
}

type activationOperationStateRepository struct {
	mu        sync.Mutex
	operation Operation
}

func (*activationOperationStateRepository) CreateAIOperation(context.Context, Operation) (Operation, error) {
	return Operation{}, errors.New("not used")
}
func (repository *activationOperationStateRepository) GetAIOperation(context.Context, string) (Operation, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	return repository.operation, nil
}
func (repository *activationOperationStateRepository) TransitionAIOperation(_ context.Context, _ string, transition OperationTransition) (Operation, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	if repository.operation.Revision != transition.ExpectedRevision {
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
func (*activationOperationStateRepository) RecoverInterruptedAIOperations(context.Context, time.Time) (int64, error) {
	return 0, nil
}

type activationRepositoryUnused struct{}

func (activationRepositoryUnused) FindAIModelActivation(context.Context, string) (ActivationWork, bool, error) {
	return ActivationWork{}, false, nil
}
func (activationRepositoryUnused) CreateAIModelActivation(context.Context, ActivationWork) (ActivationWork, bool, error) {
	return ActivationWork{}, false, nil
}
func (activationRepositoryUnused) ClaimAIModelActivation(context.Context, time.Time) (ActivationWork, bool, error) {
	return ActivationWork{}, false, nil
}
func (activationRepositoryUnused) CommitAIModelActivation(context.Context, ActivationCommit) (Operation, error) {
	return Operation{}, nil
}

var _ ActivationPackageSource = (*activationSourceStub)(nil)
var _ InferenceRuntime = (*inferenceRuntimeStub)(nil)
