package semantic

import (
	"context"
	"errors"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
)

type OperationReader interface {
	Get(context.Context, string) (aimodel.Operation, error)
}

type Service struct {
	settings   *SettingsService
	backfill   *BackfillService
	clear      *ClearService
	tagClear   *TagReviewClearService
	tagJobs    *TagJobService
	vocabulary *TagVocabularyService
	videoJobs  *VideoJobService
	operations OperationReader
}

func (service *Service) EnableVideoJobs(jobs *VideoJobService) error {
	if service == nil || jobs == nil || service.videoJobs != nil {
		return errors.New("video job service is invalid")
	}
	service.videoJobs = jobs
	return nil
}

func (service *Service) RequestVideoJob(ctx context.Context, libraryID int64, mode JobMode, key string) (aimodel.Operation, bool, error) {
	if service.videoJobs == nil {
		return aimodel.Operation{}, false, ErrInvalidVideoJob
	}
	settings, err := service.settings.Get(ctx, libraryID)
	if err != nil {
		return aimodel.Operation{}, false, err
	}
	if !settings.Enabled {
		return aimodel.Operation{}, false, ErrSemanticDisabled
	}
	if settings.ActiveGenerationID == "" {
		return aimodel.Operation{}, false, ErrSemanticGenerationUnavailable
	}
	result, err := service.videoJobs.Request(ctx, libraryID, settings.ActiveGenerationID, mode, key)
	if err != nil {
		return aimodel.Operation{}, false, err
	}
	operation, err := service.operations.Get(ctx, result.Job.OperationID)
	return operation, result.Replayed || result.Coalesced, err
}

func (service *Service) EnableTagJobs(jobs *TagJobService, vocabulary *TagVocabularyService) error {
	if service == nil || jobs == nil || vocabulary == nil || service.tagJobs != nil {
		return errors.New("tag job services are invalid")
	}
	service.tagJobs, service.vocabulary = jobs, vocabulary
	return nil
}

func (service *Service) RequestTagJob(ctx context.Context, libraryID int64, mode JobMode, key string) (aimodel.Operation, bool, error) {
	if service.tagJobs == nil || service.vocabulary == nil {
		return aimodel.Operation{}, false, ErrInvalidTagJob
	}
	settings, err := service.settings.Get(ctx, libraryID)
	if err != nil {
		return aimodel.Operation{}, false, err
	}
	if !settings.Enabled {
		return aimodel.Operation{}, false, ErrSemanticDisabled
	}
	if settings.ActiveGenerationID == "" {
		return aimodel.Operation{}, false, ErrSemanticGenerationUnavailable
	}
	vocabulary, err := service.vocabulary.Get(ctx)
	if err != nil {
		return aimodel.Operation{}, false, err
	}
	result, err := service.tagJobs.Request(ctx, libraryID, settings.ActiveGenerationID, vocabulary.ID, mode, key)
	if err != nil {
		return aimodel.Operation{}, false, err
	}
	operation, err := service.operations.Get(ctx, result.Job.OperationID)
	return operation, result.Replayed || result.Coalesced, err
}

// EnableTagReviewClear composes the independently durable review-clear workflow
// before the service is published to HTTP and operation management.
func (service *Service) EnableTagReviewClear(clear *TagReviewClearService) error {
	if service == nil || clear == nil || service.tagClear != nil {
		return errors.New("tag review clear service is invalid")
	}
	service.tagClear = clear
	return nil
}

func (service *Service) RequestTagReviewClear(ctx context.Context, libraryID, expectedRevision int64, key string) (aimodel.Operation, bool, error) {
	if service.tagClear == nil {
		return aimodel.Operation{}, false, ErrInvalidTagReviewClear
	}
	result, err := service.tagClear.Request(ctx, libraryID, expectedRevision, key)
	if err != nil {
		return aimodel.Operation{}, false, err
	}
	operation, err := service.operations.Get(ctx, result.Job.OperationID)
	return operation, result.Replayed || result.Coalesced, err
}

func NewService(settings *SettingsService, backfill *BackfillService, operations OperationReader, clear ...*ClearService) (*Service, error) {
	if settings == nil || backfill == nil || operations == nil {
		return nil, errors.New("semantic service dependencies are required")
	}
	service := &Service{settings: settings, backfill: backfill, operations: operations}
	if len(clear) > 0 {
		service.clear = clear[0]
	}
	return service, nil
}

func (service *Service) RequestClear(ctx context.Context, libraryID, expectedRevision int64, key string) (aimodel.Operation, bool, error) {
	if service.clear == nil {
		return aimodel.Operation{}, false, ErrInvalidSemanticClear
	}
	result, err := service.clear.Request(ctx, libraryID, expectedRevision, key)
	if err != nil {
		return aimodel.Operation{}, false, err
	}
	operation, err := service.operations.Get(ctx, result.Job.OperationID)
	return operation, result.Replayed || result.Coalesced, err
}

func (service *Service) GetLibrarySettings(ctx context.Context, libraryID int64) (LibrarySettings, error) {
	return service.settings.Get(ctx, libraryID)
}

func (service *Service) UpdateLibrarySettings(ctx context.Context, libraryID int64, enabled bool, revision int64) (LibrarySettings, error) {
	return service.settings.Update(ctx, libraryID, enabled, revision)
}

func (service *Service) RequestBackfill(ctx context.Context, libraryID int64, mode JobMode, key string) (aimodel.Operation, bool, error) {
	settings, err := service.settings.Get(ctx, libraryID)
	if err != nil {
		return aimodel.Operation{}, false, err
	}
	if !settings.Enabled {
		return aimodel.Operation{}, false, ErrSemanticDisabled
	}
	if settings.ActiveGenerationID == "" {
		return aimodel.Operation{}, false, ErrSemanticGenerationUnavailable
	}
	result, err := service.backfill.Request(ctx, libraryID, settings.ActiveGenerationID, mode, key)
	if err != nil {
		return aimodel.Operation{}, false, err
	}
	operation, err := service.operations.Get(ctx, result.Job.OperationID)
	return operation, result.Replayed || result.Coalesced, err
}

func (service *Service) CancelSemanticOperation(ctx context.Context, operationID string, revision int64) (aimodel.Operation, error) {
	operation, err := service.operations.Get(ctx, operationID)
	if err != nil {
		return aimodel.Operation{}, err
	}
	if operation.Kind == aimodel.OperationSemanticClear {
		if service.clear == nil {
			return aimodel.Operation{}, aimodel.ErrOperationNotFound
		}
		job, err := service.clear.CancelOperation(ctx, operationID, revision)
		if errors.Is(err, ErrSemanticClearNotFound) {
			return aimodel.Operation{}, aimodel.ErrOperationNotFound
		}
		if errors.Is(err, ErrSemanticClearConflict) {
			return aimodel.Operation{}, aimodel.ErrPreconditionFailed
		}
		if err != nil {
			return aimodel.Operation{}, err
		}
		return service.operations.Get(ctx, job.OperationID)
	}
	if operation.Kind == aimodel.OperationTagReviewClear {
		if service.tagClear == nil {
			return aimodel.Operation{}, aimodel.ErrOperationNotFound
		}
		job, err := service.tagClear.CancelOperation(ctx, operationID, revision)
		if errors.Is(err, ErrTagReviewClearNotFound) {
			return aimodel.Operation{}, aimodel.ErrOperationNotFound
		}
		if errors.Is(err, ErrTagReviewClearConflict) {
			return aimodel.Operation{}, aimodel.ErrPreconditionFailed
		}
		if err != nil {
			return aimodel.Operation{}, err
		}
		return service.operations.Get(ctx, job.OperationID)
	}
	if operation.Kind == aimodel.OperationTagSuggestionMissing || operation.Kind == aimodel.OperationTagSuggestionRebuild {
		if service.tagJobs == nil {
			return aimodel.Operation{}, aimodel.ErrOperationNotFound
		}
		job, err := service.tagJobs.CancelOperation(ctx, operationID, revision)
		if errors.Is(err, ErrTagJobNotFound) {
			return aimodel.Operation{}, aimodel.ErrOperationNotFound
		}
		if errors.Is(err, ErrTagJobConflict) {
			return aimodel.Operation{}, aimodel.ErrPreconditionFailed
		}
		if err != nil {
			return aimodel.Operation{}, err
		}
		return service.operations.Get(ctx, job.OperationID)
	}
	if operation.Kind == aimodel.OperationVideoSemanticMissing || operation.Kind == aimodel.OperationVideoSemanticRebuild {
		if service.videoJobs == nil {
			return aimodel.Operation{}, aimodel.ErrOperationNotFound
		}
		job, err := service.videoJobs.CancelOperation(ctx, operationID, revision)
		if errors.Is(err, ErrVideoJobNotFound) {
			return aimodel.Operation{}, aimodel.ErrOperationNotFound
		}
		if errors.Is(err, ErrVideoJobConflict) {
			return aimodel.Operation{}, aimodel.ErrPreconditionFailed
		}
		if err != nil {
			return aimodel.Operation{}, err
		}
		return service.operations.Get(ctx, job.OperationID)
	}
	job, err := service.backfill.CancelOperation(ctx, operationID, revision)
	if errors.Is(err, ErrSemanticJobNotFound) {
		return aimodel.Operation{}, aimodel.ErrOperationNotFound
	}
	if errors.Is(err, ErrSemanticJobConflict) {
		return aimodel.Operation{}, aimodel.ErrPreconditionFailed
	}
	if err != nil {
		return aimodel.Operation{}, err
	}
	return service.operations.Get(ctx, job.OperationID)
}
