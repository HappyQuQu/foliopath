package app

import (
	"context"
	"errors"
	"time"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
)

var errAIRepositoryNotReady = errors.New("AI model repository is not ready")

func (service *databaseService) ListAIModels(ctx context.Context) (aimodel.Snapshot, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return aimodel.Snapshot{}, errAIRepositoryNotReady
	}
	return service.store.ListAIModels(ctx)
}
func (service *databaseService) GetAIModel(ctx context.Context, id string) (aimodel.Model, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return aimodel.Model{}, errAIRepositoryNotReady
	}
	return service.store.GetAIModel(ctx, id)
}
func (service *databaseService) RegisterAIModel(ctx context.Context, model aimodel.Model) (aimodel.Model, bool, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return aimodel.Model{}, false, errAIRepositoryNotReady
	}
	return service.store.RegisterAIModel(ctx, model)
}
func (service *databaseService) SetAIModelAvailability(ctx context.Context, id string, revision int64, state aimodel.State, now time.Time) (aimodel.Model, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return aimodel.Model{}, errAIRepositoryNotReady
	}
	return service.store.SetAIModelAvailability(ctx, id, revision, state, now)
}
func (service *databaseService) CreateAIOperation(ctx context.Context, op aimodel.Operation) (aimodel.Operation, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return aimodel.Operation{}, errAIRepositoryNotReady
	}
	return service.store.CreateAIOperation(ctx, op)
}
func (service *databaseService) GetAIOperation(ctx context.Context, id string) (aimodel.Operation, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return aimodel.Operation{}, errAIRepositoryNotReady
	}
	return service.store.GetAIOperation(ctx, id)
}
func (service *databaseService) TransitionAIOperation(ctx context.Context, id string, transition aimodel.OperationTransition) (aimodel.Operation, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return aimodel.Operation{}, errAIRepositoryNotReady
	}
	return service.store.TransitionAIOperation(ctx, id, transition)
}
func (service *databaseService) RecoverInterruptedAIOperations(ctx context.Context, now time.Time) (int64, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return 0, errAIRepositoryNotReady
	}
	return service.store.RecoverInterruptedAIOperations(ctx, now)
}
func (service *databaseService) FindAIModelInstall(ctx context.Context, key string) (aimodel.InstallWork, bool, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return aimodel.InstallWork{}, false, errAIRepositoryNotReady
	}
	return service.store.FindAIModelInstall(ctx, key)
}
func (service *databaseService) CreateAIModelInstall(ctx context.Context, work aimodel.InstallWork) (aimodel.InstallWork, bool, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return aimodel.InstallWork{}, false, errAIRepositoryNotReady
	}
	return service.store.CreateAIModelInstall(ctx, work)
}
func (service *databaseService) ClaimAIModelInstall(ctx context.Context, now time.Time) (aimodel.InstallWork, bool, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return aimodel.InstallWork{}, false, errAIRepositoryNotReady
	}
	return service.store.ClaimAIModelInstall(ctx, now)
}

func (service *databaseService) FindAIModelActivation(ctx context.Context, key string) (aimodel.ActivationWork, bool, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return aimodel.ActivationWork{}, false, errAIRepositoryNotReady
	}
	return service.store.FindAIModelActivation(ctx, key)
}
func (service *databaseService) CreateAIModelActivation(ctx context.Context, work aimodel.ActivationWork) (aimodel.ActivationWork, bool, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return aimodel.ActivationWork{}, false, errAIRepositoryNotReady
	}
	return service.store.CreateAIModelActivation(ctx, work)
}
func (service *databaseService) ClaimAIModelActivation(ctx context.Context, now time.Time) (aimodel.ActivationWork, bool, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return aimodel.ActivationWork{}, false, errAIRepositoryNotReady
	}
	return service.store.ClaimAIModelActivation(ctx, now)
}
func (service *databaseService) CommitAIModelActivation(ctx context.Context, commit aimodel.ActivationCommit) (aimodel.Operation, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return aimodel.Operation{}, errAIRepositoryNotReady
	}
	return service.store.CommitAIModelActivation(ctx, commit)
}

var _ aimodel.Repository = (*databaseService)(nil)
var _ aimodel.OperationRepository = (*databaseService)(nil)
var _ aimodel.InstallQueue = (*databaseService)(nil)
var _ aimodel.ActivationRepository = (*databaseService)(nil)
