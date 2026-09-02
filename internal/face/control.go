package face

import (
	"context"
	"errors"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
)

var ErrFaceDisabled = errors.New("face analysis disabled")

type operationReader interface {
	Get(context.Context, string) (aimodel.Operation, error)
}

type ControlService struct {
	settings   *SettingsService
	jobs       *JobService
	clears     *ClearService
	operations operationReader
}

func NewControlService(settings *SettingsService, jobs *JobService, clears *ClearService, operations operationReader) (*ControlService, error) {
	if settings == nil || jobs == nil || clears == nil || operations == nil {
		return nil, errors.New("face control dependencies are required")
	}
	return &ControlService{settings: settings, jobs: jobs, clears: clears, operations: operations}, nil
}

func (service *ControlService) Get(ctx context.Context, libraryID int64) (LibrarySettings, error) {
	return service.settings.Get(ctx, libraryID)
}

func (service *ControlService) Update(ctx context.Context, libraryID int64, enabled bool, revision int64) (LibrarySettings, error) {
	return service.settings.Update(ctx, libraryID, enabled, revision)
}

func (service *ControlService) RequestFaceJob(ctx context.Context, libraryID int64, mode JobMode, key string) (aimodel.Operation, bool, error) {
	settings, err := service.settings.Get(ctx, libraryID)
	if err != nil {
		return aimodel.Operation{}, false, err
	}
	if !settings.Enabled {
		return aimodel.Operation{}, false, ErrFaceDisabled
	}
	if settings.ActiveGenerationID == "" {
		return aimodel.Operation{}, false, ErrFaceModelUnavailable
	}
	result, err := service.jobs.Request(ctx, libraryID, settings.ActiveGenerationID, mode, key)
	if err != nil {
		return aimodel.Operation{}, false, err
	}
	operation, err := service.readOperation(ctx, result.Job.OperationID, result.Job.OperationKind(), libraryID)
	return operation, !result.Created, err
}

func (service *ControlService) RequestDerivedFaceClear(ctx context.Context, libraryID, revision int64, key string) (aimodel.Operation, bool, error) {
	result, err := service.clears.RequestDerived(ctx, libraryID, revision, key)
	if err != nil {
		return aimodel.Operation{}, false, err
	}
	operation, err := service.readOperation(ctx, result.Job.OperationID, aimodel.OperationFaceDerivedClear, libraryID)
	return operation, result.Replayed, err
}

func (service *ControlService) RequestManualFaceClear(ctx context.Context, libraryID, revision int64, key string, counts ManualClearCounts) (aimodel.Operation, bool, error) {
	result, err := service.clears.RequestManual(ctx, libraryID, revision, key, counts)
	if err != nil {
		return aimodel.Operation{}, false, err
	}
	operation, err := service.readOperation(ctx, result.Job.OperationID, aimodel.OperationFaceManualClear, libraryID)
	return operation, result.Replayed, err
}

func (service *ControlService) CancelFaceOperation(ctx context.Context, operationID string, revision int64) (aimodel.Operation, error) {
	operation, err := service.operations.Get(ctx, operationID)
	if err != nil {
		return aimodel.Operation{}, err
	}
	switch operation.Kind {
	case aimodel.OperationFaceMissing, aimodel.OperationFaceRebuild:
		job, err := service.jobs.Cancel(ctx, operationID, revision)
		if errors.Is(err, ErrFaceJobNotFound) {
			return aimodel.Operation{}, aimodel.ErrOperationNotFound
		}
		if errors.Is(err, ErrFaceJobConflict) {
			return aimodel.Operation{}, aimodel.ErrPreconditionFailed
		}
		if err != nil {
			return aimodel.Operation{}, err
		}
		return service.readOperation(ctx, job.OperationID, operation.Kind, operation.LibraryID)
	case aimodel.OperationFaceDerivedClear, aimodel.OperationFaceManualClear:
		job, err := service.clears.Cancel(ctx, operationID, revision)
		if errors.Is(err, ErrFaceClearConflict) {
			return aimodel.Operation{}, aimodel.ErrPreconditionFailed
		}
		if err != nil {
			return aimodel.Operation{}, err
		}
		return service.readOperation(ctx, job.OperationID, operation.Kind, operation.LibraryID)
	default:
		return aimodel.Operation{}, aimodel.ErrInvalidTransition
	}
}

func (service *ControlService) readOperation(ctx context.Context, id string, kind aimodel.OperationKind, libraryID int64) (aimodel.Operation, error) {
	operation, err := service.operations.Get(ctx, id)
	if err != nil {
		return aimodel.Operation{}, err
	}
	if operation.ID != id || operation.Kind != kind || operation.LibraryID != libraryID {
		return aimodel.Operation{}, aimodel.ErrRepositoryState
	}
	return operation, nil
}
