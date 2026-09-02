package aimodel

import (
	"context"
	"errors"
	"testing"
	"time"
)

type installAdmissionStub struct {
	candidate Candidate
	mode      StorageMode
	key       string
	result    InstallResult
}

type activationAdmissionStub struct {
	model  Model
	key    string
	result ActivationResult
}

func (*activationAdmissionStub) ReplayActivation(context.Context, string, int64, string) (ActivationResult, bool, error) {
	return ActivationResult{}, false, nil
}

func (stub *activationAdmissionStub) StartActivation(_ context.Context, model Model, key string) (ActivationResult, error) {
	stub.model, stub.key = model, key
	return stub.result, nil
}

func (stub *installAdmissionStub) ReplayInstall(context.Context, string, StorageMode, string) (InstallResult, bool, error) {
	return InstallResult{}, false, nil
}

type managementOperationRepository struct{}

func (managementOperationRepository) CreateAIOperation(context.Context, Operation) (Operation, error) {
	return Operation{}, errors.New("not used")
}

func (managementOperationRepository) GetAIOperation(context.Context, string) (Operation, error) {
	return Operation{}, ErrOperationNotFound
}

func (managementOperationRepository) TransitionAIOperation(context.Context, string, OperationTransition) (Operation, error) {
	return Operation{}, errors.New("not used")
}

func (managementOperationRepository) RecoverInterruptedAIOperations(context.Context, time.Time) (int64, error) {
	return 0, nil
}

type semanticManagementOperationRepository struct{ operation Operation }

func (repository semanticManagementOperationRepository) CreateAIOperation(context.Context, Operation) (Operation, error) {
	return Operation{}, errors.New("not used")
}

func (repository semanticManagementOperationRepository) GetAIOperation(context.Context, string) (Operation, error) {
	return repository.operation, nil
}

func (semanticManagementOperationRepository) TransitionAIOperation(context.Context, string, OperationTransition) (Operation, error) {
	return Operation{}, errors.New("not used")
}

func (semanticManagementOperationRepository) RecoverInterruptedAIOperations(context.Context, time.Time) (int64, error) {
	return 0, nil
}

type semanticOperationCancellerStub struct {
	operationID string
	revision    int64
	result      Operation
}

func (stub *semanticOperationCancellerStub) CancelSemanticOperation(_ context.Context, operationID string, revision int64) (Operation, error) {
	stub.operationID, stub.revision = operationID, revision
	return stub.result, nil
}

type faceOperationCancellerStub struct {
	operationID string
	revision    int64
	result      Operation
}

func (stub *faceOperationCancellerStub) CancelFaceOperation(_ context.Context, operationID string, revision int64) (Operation, error) {
	stub.operationID, stub.revision = operationID, revision
	return stub.result, nil
}

func (stub *installAdmissionStub) StartInstall(
	_ context.Context,
	candidate Candidate,
	mode StorageMode,
	key string,
) (InstallResult, error) {
	stub.candidate, stub.mode, stub.key = candidate, mode, key
	return stub.result, nil
}

func TestManagementServiceResolvesOnlyCurrentCandidate(t *testing.T) {
	catalog, manifest, facts := catalogFixture(t)
	encoded, err := jsonMarshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	source := staticCandidateSource{items: []RawCandidate{{
		Manifest: encoded, Files: facts, SourceIdentity: "source:current",
	}}}
	ids := []string{"aic_first123", "aic_second12"}
	scanner, err := NewScanner(source, catalog, "arm64", func() (string, error) {
		id := ids[0]
		ids = ids[1:]
		return id, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	modelRepository := &memoryRepository{snapshot: Snapshot{Revision: 1}}
	models, err := NewService(modelRepository, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	operationRepository := managementOperationRepository{}
	operations, err := NewOperationService(operationRepository, func() time.Time { return time.Unix(1, 0) }, nil)
	if err != nil {
		t.Fatal(err)
	}
	admission := &installAdmissionStub{}
	activation := &activationAdmissionStub{}
	availabilitySource := &activationSourceStub{}
	availability, err := NewAvailabilityService(models, catalog, availabilitySource)
	if err != nil {
		t.Fatal(err)
	}
	management, err := NewManagementService(models, scanner, operations, admission, activation, availability)
	if err != nil {
		t.Fatal(err)
	}
	first, err := management.ScanCandidates(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(2, 0).UTC()
	result := InstallResult{Operation: Operation{
		ID: "aio_install123", Kind: OperationModelInstall, State: OperationQueued, Phase: PhaseQueued,
		Revision: 1, CreatedAt: now, UpdatedAt: now,
	}, Created: true}
	admission.result = result
	got, err := management.StartInstall(context.Background(), first.Candidates[0].ID, StorageManaged, "request-key")
	if err != nil || got.Operation.ID != result.Operation.ID || admission.candidate.SourceIdentity != "source:current" ||
		admission.mode != StorageManaged || admission.key != "request-key" {
		t.Fatalf("install admission = %#v, %v; stub=%#v", got, err, admission)
	}
	if _, err := management.ScanCandidates(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := management.StartInstall(context.Background(), first.Candidates[0].ID, StorageManaged, "request-key-2"); !errors.Is(err, ErrCandidateStale) {
		t.Fatalf("stale candidate error = %v", err)
	}
	model := activationAdmissionModel(now)
	modelRepository.snapshot.Items = []Model{model}
	activation.result = ActivationResult{Operation: Operation{
		ID: "aio_activate123", Kind: OperationModelActivate, State: OperationQueued, Phase: PhaseQueued,
		ModelID: model.ID, Revision: 1, CreatedAt: now, UpdatedAt: now,
	}, Created: true}
	activated, err := management.StartActivation(context.Background(), model.ID, model.AvailabilityRevision, "activate-request")
	if err != nil || activated.Operation.ID != "aio_activate123" || activation.model.ID != model.ID || activation.key != "activate-request" {
		t.Fatalf("activation admission = %#v, %v; stub=%#v", activated, err, activation)
	}
	if _, err := management.StartActivation(context.Background(), model.ID, model.AvailabilityRevision+1, "activate-stale"); !errors.Is(err, ErrPreconditionFailed) {
		t.Fatalf("stale activation error = %v", err)
	}
}

func TestManagementServiceRejectsInvalidAdmissionResults(t *testing.T) {
	now := time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC)
	install := InstallResult{Operation: Operation{
		ID: "aio_install123", Kind: OperationModelInstall, State: OperationQueued, Phase: PhaseQueued,
		Revision: 1, CreatedAt: now, UpdatedAt: now,
	}, Created: true}
	activation := ActivationResult{Operation: Operation{
		ID: "aio_activate123", Kind: OperationModelActivate, State: OperationQueued, Phase: PhaseQueued,
		ModelID: "aim_reviewed123", Revision: 1, CreatedAt: now, UpdatedAt: now,
	}, Created: true}

	if err := validateInstallResult(install); err != nil {
		t.Fatalf("valid install result: %v", err)
	}
	if err := validateActivationResult(activation, "aim_reviewed123"); err != nil {
		t.Fatalf("valid activation result: %v", err)
	}

	installCases := []InstallResult{
		mutateInstallResult(install, func(value *InstallResult) { value.Operation.Kind = OperationModelActivate }),
		mutateInstallResult(install, func(value *InstallResult) {
			value.Operation.State = OperationRunning
			value.Operation.Phase = PhaseCopying
		}),
		mutateInstallResult(install, func(value *InstallResult) { value.Replayed = true }),
		mutateInstallResult(install, func(value *InstallResult) { value.Created = false }),
	}
	for index, result := range installCases {
		if err := validateInstallResult(result); !errors.Is(err, ErrRepositoryState) {
			t.Fatalf("install case %d error = %v", index, err)
		}
	}

	activationCases := []ActivationResult{
		mutateActivationResult(activation, func(value *ActivationResult) { value.Operation.ModelID = "aim_different123" }),
		mutateActivationResult(activation, func(value *ActivationResult) { value.Operation.Kind = OperationModelInstall }),
		mutateActivationResult(activation, func(value *ActivationResult) { value.Replayed = true }),
		mutateActivationResult(activation, func(value *ActivationResult) { value.Created = false }),
	}
	for index, result := range activationCases {
		if err := validateActivationResult(result, "aim_reviewed123"); !errors.Is(err, ErrRepositoryState) {
			t.Fatalf("activation case %d error = %v", index, err)
		}
	}
}

func TestManagementServiceRoutesEveryDerivedSemanticOperationToItsCanonicalOwner(t *testing.T) {
	now := time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)
	for _, kind := range []OperationKind{
		OperationSemanticMissing,
		OperationSemanticRebuild,
		OperationSemanticClear,
		OperationTagSuggestionMissing,
		OperationTagSuggestionRebuild,
		OperationTagReviewClear,
		OperationVideoSemanticMissing,
		OperationVideoSemanticRebuild,
	} {
		t.Run(string(kind), func(t *testing.T) {
			operation := Operation{ID: "aio_semantic_route", Kind: kind, State: OperationQueued, Phase: PhaseQueued,
				Revision: 7, CreatedAt: now, UpdatedAt: now}
			operations, err := NewOperationService(semanticManagementOperationRepository{operation: operation}, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			cancelled := operation
			cancelled.State, cancelled.Phase, cancelled.Revision = OperationCancelling, PhaseBuilding, 8
			canceller := &semanticOperationCancellerStub{result: cancelled}
			management := &ManagementService{operations: operations, semantic: canceller}

			result, err := management.CancelOperation(context.Background(), operation.ID, operation.Revision)
			if err != nil {
				t.Fatal(err)
			}
			if result != cancelled || canceller.operationID != operation.ID || canceller.revision != operation.Revision {
				t.Fatalf("cancel result = %#v; owner = %#v", result, canceller)
			}
		})
	}
}

func TestManagementServiceFailsClosedWhenSemanticOwnerIsUnavailable(t *testing.T) {
	now := time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)
	operation := Operation{ID: "aio_tag_route", Kind: OperationTagSuggestionMissing, State: OperationQueued,
		Phase: PhaseQueued, Revision: 1, CreatedAt: now, UpdatedAt: now}
	operations, err := NewOperationService(semanticManagementOperationRepository{operation: operation}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	management := &ManagementService{operations: operations}
	if _, err := management.CancelOperation(context.Background(), operation.ID, operation.Revision); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("cancel without semantic owner error = %v", err)
	}
}

func TestManagementServiceRoutesEveryFaceOperationToItsCanonicalOwner(t *testing.T) {
	now := time.Date(2026, 9, 2, 18, 0, 0, 0, time.UTC)
	for _, kind := range []OperationKind{
		OperationFaceMissing,
		OperationFaceRebuild,
		OperationFaceDerivedClear,
		OperationFaceManualClear,
	} {
		t.Run(string(kind), func(t *testing.T) {
			operation := Operation{ID: "aio_face_route", Kind: kind, State: OperationQueued, Phase: PhaseQueued,
				Revision: 7, CreatedAt: now, UpdatedAt: now}
			operations, err := NewOperationService(semanticManagementOperationRepository{operation: operation}, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			cancelled := operation
			cancelled.State, cancelled.Phase, cancelled.Revision = OperationCancelling, PhaseBuilding, 8
			canceller := &faceOperationCancellerStub{result: cancelled}
			management := &ManagementService{operations: operations, face: canceller}

			result, err := management.CancelOperation(context.Background(), operation.ID, operation.Revision)
			if err != nil {
				t.Fatal(err)
			}
			if result != cancelled || canceller.operationID != operation.ID || canceller.revision != operation.Revision {
				t.Fatalf("cancel result = %#v; owner = %#v", result, canceller)
			}
		})
	}
}

func TestManagementServiceFailsClosedWhenFaceOwnerIsUnavailable(t *testing.T) {
	now := time.Date(2026, 9, 2, 18, 30, 0, 0, time.UTC)
	operation := Operation{ID: "aio_face_route", Kind: OperationFaceMissing, State: OperationQueued,
		Phase: PhaseQueued, Revision: 1, CreatedAt: now, UpdatedAt: now}
	operations, err := NewOperationService(semanticManagementOperationRepository{operation: operation}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	management := &ManagementService{operations: operations}
	if _, err := management.CancelOperation(context.Background(), operation.ID, operation.Revision); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("cancel without face owner error = %v", err)
	}
}

func mutateInstallResult(value InstallResult, mutate func(*InstallResult)) InstallResult {
	mutate(&value)
	return value
}

func mutateActivationResult(value ActivationResult, mutate func(*ActivationResult)) ActivationResult {
	mutate(&value)
	return value
}

var _ InstallAdmission = (*installAdmissionStub)(nil)
var _ ActivationAdmission = (*activationAdmissionStub)(nil)
