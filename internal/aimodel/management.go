package aimodel

import (
	"context"
	"errors"
)

type InstallResult struct {
	Operation Operation
	Created   bool
	Replayed  bool
}

type InstallAdmission interface {
	ReplayInstall(context.Context, string, StorageMode, string) (InstallResult, bool, error)
	StartInstall(context.Context, Candidate, StorageMode, string) (InstallResult, error)
}

// ManagementService is the single application-facing owner for model-package
// queries and mutations. HTTP never coordinates scanner, installer, and
// operation state directly.
type ManagementService struct {
	models       *Service
	scanner      *Scanner
	operations   *OperationService
	install      InstallAdmission
	activation   ActivationAdmission
	availability *AvailabilityService
	semantic     SemanticOperationCanceller
}

type SemanticOperationCanceller interface {
	CancelSemanticOperation(context.Context, string, int64) (Operation, error)
}

func NewManagementService(
	models *Service,
	scanner *Scanner,
	operations *OperationService,
	install InstallAdmission,
	activation ActivationAdmission,
	availability *AvailabilityService,
	semanticCanceller ...SemanticOperationCanceller,
) (*ManagementService, error) {
	if models == nil || scanner == nil || operations == nil || install == nil || activation == nil || availability == nil {
		return nil, errors.New("AI model management dependencies are required")
	}
	var semantic SemanticOperationCanceller
	if len(semanticCanceller) > 1 {
		return nil, errors.New("only one semantic operation canceller is allowed")
	}
	if len(semanticCanceller) == 1 {
		semantic = semanticCanceller[0]
	}
	return &ManagementService{
		models: models, scanner: scanner, operations: operations, install: install, activation: activation,
		availability: availability, semantic: semantic,
	}, nil
}

func (service *ManagementService) List(ctx context.Context) (Snapshot, error) {
	return service.models.List(ctx)
}

func (service *ManagementService) ScanCandidates(ctx context.Context) (CandidateScan, error) {
	if _, err := service.availability.Refresh(ctx); err != nil {
		return CandidateScan{}, err
	}
	return service.scanner.Scan(ctx)
}

func (service *ManagementService) StartInstall(
	ctx context.Context,
	candidateID string,
	storageMode StorageMode,
	idempotencyKey string,
) (InstallResult, error) {
	if storageMode != StorageManaged && storageMode != StorageDirect {
		return InstallResult{}, ErrInvalidModel
	}
	if replay, found, err := service.install.ReplayInstall(ctx, candidateID, storageMode, idempotencyKey); err != nil {
		return InstallResult{}, err
	} else if found {
		if err := validateInstallResult(replay); err != nil {
			return InstallResult{}, err
		}
		return replay, nil
	}
	candidate, err := service.scanner.ResolveCurrent(candidateID)
	if err != nil {
		return InstallResult{}, err
	}
	result, err := service.install.StartInstall(ctx, candidate, storageMode, idempotencyKey)
	if err != nil {
		return InstallResult{}, err
	}
	if err := validateInstallResult(result); err != nil {
		return InstallResult{}, err
	}
	return result, nil
}

func (service *ManagementService) StartActivation(
	ctx context.Context,
	modelID string,
	availabilityRevision int64,
	idempotencyKey string,
) (ActivationResult, error) {
	if modelID == "" || availabilityRevision < 1 {
		return ActivationResult{}, ErrInvalidModel
	}
	if replay, found, err := service.activation.ReplayActivation(ctx, modelID, availabilityRevision, idempotencyKey); err != nil {
		return ActivationResult{}, err
	} else if found {
		if err := validateActivationResult(replay, modelID); err != nil {
			return ActivationResult{}, err
		}
		return replay, nil
	}
	model, err := service.models.Get(ctx, modelID)
	if err != nil {
		return ActivationResult{}, err
	}
	if model.AvailabilityRevision != availabilityRevision {
		return ActivationResult{}, ErrPreconditionFailed
	}
	result, err := service.activation.StartActivation(ctx, model, idempotencyKey)
	if err != nil {
		return ActivationResult{}, err
	}
	if err := validateActivationResult(result, modelID); err != nil {
		return ActivationResult{}, err
	}
	return result, nil
}

func validateInstallResult(result InstallResult) error {
	if validateOperation(result.Operation) != nil || result.Operation.Kind != OperationModelInstall ||
		result.Operation.ModelID != "" || result.Created == result.Replayed {
		return ErrRepositoryState
	}
	if result.Created && (result.Operation.State != OperationQueued || result.Operation.Phase != PhaseQueued ||
		result.Operation.Revision != 1) {
		return ErrRepositoryState
	}
	return nil
}

func validateActivationResult(result ActivationResult, modelID string) error {
	if validateOperation(result.Operation) != nil || result.Operation.Kind != OperationModelActivate ||
		result.Operation.ModelID != modelID || result.Created == result.Replayed {
		return ErrRepositoryState
	}
	if result.Created && (result.Operation.State != OperationQueued || result.Operation.Phase != PhaseQueued ||
		result.Operation.Revision != 1) {
		return ErrRepositoryState
	}
	return nil
}

func (service *ManagementService) GetOperation(ctx context.Context, operationID string) (Operation, error) {
	return service.operations.Get(ctx, operationID)
}

func (service *ManagementService) CancelOperation(
	ctx context.Context,
	operationID string,
	revision int64,
) (Operation, error) {
	operation, err := service.operations.Get(ctx, operationID)
	if err != nil {
		return Operation{}, err
	}
	if operation.Kind == OperationSemanticMissing || operation.Kind == OperationSemanticRebuild || operation.Kind == OperationSemanticClear ||
		operation.Kind == OperationTagReviewClear || operation.Kind == OperationTagSuggestionMissing || operation.Kind == OperationTagSuggestionRebuild ||
		operation.Kind == OperationVideoSemanticMissing || operation.Kind == OperationVideoSemanticRebuild {
		if service.semantic == nil {
			return Operation{}, ErrInvalidTransition
		}
		return service.semantic.CancelSemanticOperation(ctx, operationID, revision)
	}
	return service.operations.RequestCancel(ctx, operationID, revision)
}
