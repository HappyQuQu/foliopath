package app

import (
	"context"
	"errors"
	"github.com/HappyQuQu/foliopath/internal/deletion"
	"github.com/HappyQuQu/foliopath/internal/jobs"
	"time"
)

var errDeletionRepositoryNotReady = errors.New("deletion repository not ready")

func (service *databaseService) GetDeletionSettings(ctx context.Context) (deletion.Settings, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return deletion.Settings{}, errDeletionRepositoryNotReady
	}
	return service.store.GetDeletionSettings(ctx)
}

func (service *databaseService) UpdateDeletionSettings(ctx context.Context, revision int64, update deletion.SettingsUpdate) (deletion.Settings, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return deletion.Settings{}, errDeletionRepositoryNotReady
	}
	return service.store.UpdateDeletionSettings(ctx, revision, update)
}

func (service *databaseService) PreviewDeletionRetention(ctx context.Context, days int64) (int64, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return 0, errDeletionRepositoryNotReady
	}
	return service.store.PreviewDeletionRetention(ctx, days)
}

func (service *databaseService) ResolveDeletionTargets(ctx context.Context, targets []deletion.Target) ([]deletion.Selection, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return nil, errDeletionRepositoryNotReady
	}
	return service.store.ResolveDeletionTargets(ctx, targets)
}

func (service *databaseService) SaveDeletionPreflight(ctx context.Context, value deletion.Preflight) (deletion.Preflight, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return deletion.Preflight{}, errDeletionRepositoryNotReady
	}
	return service.store.SaveDeletionPreflight(ctx, value)
}

func (service *databaseService) GetDeletionPreflight(ctx context.Context, id string) (deletion.Preflight, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return deletion.Preflight{}, errDeletionRepositoryNotReady
	}
	return service.store.GetDeletionPreflight(ctx, id)
}

func (service *databaseService) FindDeletionReplay(ctx context.Context, key, request [32]byte) (deletion.Operation, bool, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return deletion.Operation{}, false, errDeletionRepositoryNotReady
	}
	return service.store.FindDeletionReplay(ctx, key, request)
}

func (service *databaseService) EnqueueDeletion(ctx context.Context, command deletion.EnqueueCommand) (deletion.Operation, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return deletion.Operation{}, errDeletionRepositoryNotReady
	}
	return service.store.EnqueueDeletion(ctx, command)
}

func (service *databaseService) GetDeletionOperation(ctx context.Context, id int64) (deletion.Operation, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return deletion.Operation{}, errDeletionRepositoryNotReady
	}
	return service.store.GetDeletionOperation(ctx, id)
}

func (service *databaseService) ListDeletionOperationItems(ctx context.Context, id, before int64, limit int) ([]deletion.OperationItem, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return nil, errDeletionRepositoryNotReady
	}
	return service.store.ListDeletionOperationItems(ctx, id, before, limit)
}

func (service *databaseService) ListDeletionOperations(ctx context.Context, libraryID, before int64, limit int) ([]deletion.Operation, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return nil, errDeletionRepositoryNotReady
	}
	return service.store.ListDeletionOperations(ctx, libraryID, before, limit)
}

func (service *databaseService) ChangeDeletionOperation(ctx context.Context, command deletion.ChangeCommand) (deletion.Operation, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return deletion.Operation{}, errDeletionRepositoryNotReady
	}
	return service.store.ChangeDeletionOperation(ctx, command)
}

func (service *databaseService) ListTrashEntries(ctx context.Context, libraryID, before int64, limit int) ([]deletion.TrashEntry, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return nil, errDeletionRepositoryNotReady
	}
	return service.store.ListTrashEntries(ctx, libraryID, before, limit)
}

func (service *databaseService) EnqueueTrashOperation(ctx context.Context, command deletion.TrashCommand) (deletion.Operation, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return deletion.Operation{}, errDeletionRepositoryNotReady
	}
	return service.store.EnqueueTrashOperation(ctx, command)
}

func (service *databaseService) RecoverExpired(ctx context.Context) (jobs.RecoverySummary, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return jobs.RecoverySummary{}, errDeletionRepositoryNotReady
	}
	return service.store.RecoverExpired(ctx)
}

func (service *databaseService) Claim(ctx context.Context, lease time.Duration) (deletion.Claim, bool, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return deletion.Claim{}, false, errDeletionRepositoryNotReady
	}
	return service.store.Claim(ctx, lease)
}

func (service *databaseService) RefreshLease(ctx context.Context, claim deletion.Claim, lease time.Duration) (bool, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return false, errDeletionRepositoryNotReady
	}
	return service.store.RefreshLease(ctx, claim, lease)
}

func (service *databaseService) NextDeletionItem(ctx context.Context, claim deletion.Claim) (deletion.OperationItem, bool, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return deletion.OperationItem{}, false, errDeletionRepositoryNotReady
	}
	return service.store.NextDeletionItem(ctx, claim)
}

func (service *databaseService) CheckDeletionPermission(ctx context.Context, claim deletion.Claim) error {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return errDeletionRepositoryNotReady
	}
	return service.store.CheckDeletionPermission(ctx, claim)
}

func (service *databaseService) BeginDeletionItem(ctx context.Context, claim deletion.Claim, item deletion.OperationItem) error {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return errDeletionRepositoryNotReady
	}
	return service.store.BeginDeletionItem(ctx, claim, item)
}

func (service *databaseService) RecordDeletionFilesystem(ctx context.Context, claim deletion.Claim, item deletion.OperationItem, removed int64, done, unchanged bool, code string) error {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return errDeletionRepositoryNotReady
	}
	return service.store.RecordDeletionFilesystem(ctx, claim, item, removed, done, unchanged, code)
}

func (service *databaseService) FinalizeDeletionItem(ctx context.Context, claim deletion.Claim, item deletion.OperationItem) (bool, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return false, errDeletionRepositoryNotReady
	}
	return service.store.FinalizeDeletionItem(ctx, claim, item)
}

func (service *databaseService) FinishDeletionClaim(ctx context.Context, claim deletion.Claim, state string) error {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return errDeletionRepositoryNotReady
	}
	return service.store.FinishDeletionClaim(ctx, claim, state)
}

func (service *databaseService) ListDeletionRepairs(ctx context.Context, limit int) ([]deletion.RepairRequest, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return nil, errDeletionRepositoryNotReady
	}
	return service.store.ListDeletionRepairs(ctx, limit)
}

func (service *databaseService) AssignDeletionRepair(ctx context.Context, request deletion.RepairRequest, id int64) error {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return errDeletionRepositoryNotReady
	}
	return service.store.AssignDeletionRepair(ctx, request, id)
}

func (service *databaseService) CompleteDeletionRepair(ctx context.Context, request deletion.RepairRequest, success bool) error {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return errDeletionRepositoryNotReady
	}
	return service.store.CompleteDeletionRepair(ctx, request, success)
}

func (service *databaseService) CheckDeletionRestore(ctx context.Context, claim deletion.Claim, item deletion.OperationItem) error {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return errDeletionRepositoryNotReady
	}
	return service.store.CheckDeletionRestore(ctx, claim, item)
}

func (service *databaseService) HandoffDeletion(ctx context.Context, claim deletion.Claim) error {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return errDeletionRepositoryNotReady
	}
	return service.store.HandoffDeletion(ctx, claim)
}
