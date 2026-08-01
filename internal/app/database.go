package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/HappyQuQu/foliopath/internal/api"
	"github.com/HappyQuQu/foliopath/internal/auth"
	"github.com/HappyQuQu/foliopath/internal/catalog"
	"github.com/HappyQuQu/foliopath/internal/jobs"
	"github.com/HappyQuQu/foliopath/internal/library"
	"github.com/HappyQuQu/foliopath/internal/media"
	"github.com/HappyQuQu/foliopath/internal/scanner"
	appsettings "github.com/HappyQuQu/foliopath/internal/settings"
	sqlitestore "github.com/HappyQuQu/foliopath/internal/store/sqlite"
	"github.com/HappyQuQu/foliopath/internal/thumbnail"
)

const databaseFilename = "foliopath.db"

type databaseStore interface {
	auth.Repository
	auth.AccountRepository
	catalog.Repository
	library.Repository
	library.LifecycleRepository
	library.RemovalRepository
	scanner.AdmissionRepository
	scanner.Repository
	scanner.QueryRepository
	scanner.ScheduleRepository
	scanner.ReconcileRepository
	scanner.ReconcileExecutionRepository
	scanner.AutomaticDiscoveryStateRepository
	appsettings.Repository
	thumbnail.Repository
	thumbnail.StoryboardRepository
	thumbnail.CacheRepository
	thumbnail.CleanupRepository
	thumbnail.JobCompletionRepository
	thumbnail.ProgressRepository
	thumbnail.DeliveryRepository
	media.ContentRepository
	scanQueueStore
	reconcileQueueStore
	mediaQueueStore
	Close() error
}

func (service *databaseService) EnqueueReconcile(
	ctx context.Context,
	libraryID int64,
	relativeDirectory string,
	debounce time.Duration,
	maximumDebounce time.Duration,
) (scanner.ReconcileJob, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return scanner.ReconcileJob{}, scanner.ErrDatabaseUnavailable
	}
	return service.store.EnqueueReconcile(
		ctx,
		libraryID,
		relativeDirectory,
		debounce,
		maximumDebounce,
	)
}

func (service *databaseService) CommitDirectoryReconcile(
	ctx context.Context,
	job scanner.ReconcileJob,
	entries []scanner.CatalogEntry,
) (scanner.ReconcileCommitResult, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return scanner.ReconcileCommitResult{}, scanner.ErrDatabaseUnavailable
	}
	return service.store.CommitDirectoryReconcile(ctx, job, entries)
}

func (service *databaseService) FailDirectoryReconcile(
	ctx context.Context,
	job scanner.ReconcileJob,
	errorCode string,
) error {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return scanner.ErrDatabaseUnavailable
	}
	return service.store.FailDirectoryReconcile(ctx, job, errorCode)
}

func (service *databaseService) SetAutomaticDiscoveryState(
	ctx context.Context,
	libraryID int64,
	status string,
	errorCode string,
) error {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return scanner.ErrDatabaseUnavailable
	}
	return service.store.SetAutomaticDiscoveryState(ctx, libraryID, status, errorCode)
}

func (service *databaseService) GetContentAsset(
	ctx context.Context,
	assetID int64,
) (media.ContentAsset, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return media.ContentAsset{}, media.ErrContentUnavailable
	}
	return service.store.GetContentAsset(ctx, assetID)
}

func (service *databaseService) ResolveScope(
	ctx context.Context,
	libraryID, selectedDirectoryID int64,
) (catalog.Scope, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return catalog.Scope{}, catalog.ErrRepositoryNotReady
	}
	return service.store.ResolveScope(ctx, libraryID, selectedDirectoryID)
}

func (service *databaseService) ResolveGlobalCatalogRevision(
	ctx context.Context,
) (int64, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return 0, catalog.ErrRepositoryNotReady
	}
	return service.store.ResolveGlobalCatalogRevision(ctx)
}

func (service *databaseService) ResolveCatalogContentRevision(
	ctx context.Context,
) (int64, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return 0, catalog.ErrRepositoryNotReady
	}
	return service.store.ResolveCatalogContentRevision(ctx)
}

func (service *databaseService) ListDirectoryPage(
	ctx context.Context,
	params catalog.DirectoryListParams,
) ([]catalog.Directory, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return nil, catalog.ErrRepositoryNotReady
	}
	return service.store.ListDirectoryPage(ctx, params)
}

func (service *databaseService) ListAssetPage(
	ctx context.Context,
	params catalog.AssetListParams,
) ([]catalog.Asset, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return nil, catalog.ErrRepositoryNotReady
	}
	return service.store.ListAssetPage(ctx, params)
}

func (service *databaseService) CountAssets(
	ctx context.Context,
	query catalog.AssetQuery,
) (catalog.AssetCounts, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return catalog.AssetCounts{}, catalog.ErrRepositoryNotReady
	}
	return service.store.CountAssets(ctx, query)
}

func (service *databaseService) GetAsset(
	ctx context.Context,
	assetID int64,
) (catalog.Asset, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return catalog.Asset{}, catalog.ErrRepositoryNotReady
	}
	return service.store.GetAsset(ctx, assetID)
}

func (service *databaseService) GetDirectoryLineage(
	ctx context.Context,
	directoryID int64,
	maximum int,
) (catalog.DirectoryLineage, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return catalog.DirectoryLineage{}, catalog.ErrRepositoryNotReady
	}
	return service.store.GetDirectoryLineage(ctx, directoryID, maximum)
}

func (service *databaseService) GetAssetForDerivation(
	ctx context.Context,
	assetID int64,
) (thumbnail.Asset, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return thumbnail.Asset{}, thumbnail.ErrRepositoryNotReady
	}
	return service.store.GetAssetForDerivation(ctx, assetID)
}

func (service *databaseService) CommitReady(
	ctx context.Context,
	ready thumbnail.Ready,
) error {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return thumbnail.ErrRepositoryNotReady
	}
	return service.store.CommitReady(ctx, ready)
}

func (service *databaseService) CommitFailure(
	ctx context.Context,
	failure thumbnail.Failure,
) error {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return thumbnail.ErrRepositoryNotReady
	}
	return service.store.CommitFailure(ctx, failure)
}

func (service *databaseService) GetThumbnailDelivery(
	ctx context.Context,
	assetID int64,
	variant thumbnail.Variant,
) (thumbnail.DeliveryState, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return thumbnail.DeliveryState{}, thumbnail.ErrRepositoryNotReady
	}
	return service.store.GetThumbnailDelivery(ctx, assetID, variant)
}

func (service *databaseService) TouchThumbnail(
	ctx context.Context,
	assetID int64,
	variant thumbnail.Variant,
	fingerprint media.SourceFingerprint,
	cachePath string,
) error {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return thumbnail.ErrRepositoryNotReady
	}
	return service.store.TouchThumbnail(
		ctx,
		assetID,
		variant,
		fingerprint,
		cachePath,
	)
}

func (service *databaseService) RequeueMissingThumbnail(
	ctx context.Context,
	state thumbnail.DeliveryState,
) error {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return thumbnail.ErrRepositoryNotReady
	}
	return service.store.RequeueMissingThumbnail(ctx, state)
}

func (service *databaseService) CacheQuota(ctx context.Context) (int64, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return 0, thumbnail.ErrRepositoryNotReady
	}
	return service.store.CacheQuota(ctx)
}

func (service *databaseService) ReadyCacheUsage(ctx context.Context) (int64, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return 0, thumbnail.ErrRepositoryNotReady
	}
	return service.store.ReadyCacheUsage(ctx)
}

func (service *databaseService) ListLRUCacheEntries(
	ctx context.Context,
	limit int,
) ([]thumbnail.CacheEntry, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return nil, thumbnail.ErrRepositoryNotReady
	}
	return service.store.ListLRUCacheEntries(ctx, limit)
}

func (service *databaseService) DeleteReadyCacheEntries(
	ctx context.Context,
	entries []thumbnail.CacheEntry,
) error {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return thumbnail.ErrRepositoryNotReady
	}
	return service.store.DeleteReadyCacheEntries(ctx, entries)
}

func (service *databaseService) ListPendingCacheDeletions(
	ctx context.Context,
	limit int,
) ([]thumbnail.PendingCacheDeletion, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return nil, thumbnail.ErrRepositoryNotReady
	}
	return service.store.ListPendingCacheDeletions(ctx, limit)
}

func (service *databaseService) CompleteCacheDeletion(
	ctx context.Context,
	item thumbnail.PendingCacheDeletion,
) error {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return thumbnail.ErrRepositoryNotReady
	}
	return service.store.CompleteCacheDeletion(ctx, item)
}

func (service *databaseService) FinishMediaJob(
	ctx context.Context,
	job thumbnail.Job,
	result thumbnail.JobResult,
) error {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return thumbnail.ErrRepositoryNotReady
	}
	return service.store.FinishMediaJob(ctx, job, result)
}

func (service *databaseService) GetMediaProcessingProgress(
	ctx context.Context,
	libraryID int64,
) (thumbnail.ProcessingProgress, bool, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return thumbnail.ProcessingProgress{}, false, thumbnail.ErrRepositoryNotReady
	}
	return service.store.GetMediaProcessingProgress(ctx, libraryID)
}

type scanQueueStore interface {
	RecoverExpiredFullScans(context.Context) (jobs.RecoverySummary, error)
	ClaimNextFullScan(context.Context, time.Duration) (scanner.ScanRun, bool, error)
	RefreshFullScanLease(context.Context, int64, time.Duration) (scanner.ScanRun, error)
}

type mediaQueueStore interface {
	ReconcileMediaJobTransform(context.Context, int, int) (int64, error)
	ReconcileStoryboardJobTransform(context.Context, int, int) (int64, error)
	AdmitStoryboardJobs(context.Context, int) (int64, error)
	RecoverExpiredMediaJobs(context.Context) (jobs.RecoverySummary, error)
	ClaimNextMediaJob(
		context.Context,
		time.Duration,
	) (thumbnail.Job, bool, error)
	RefreshMediaJobLease(context.Context, thumbnail.Job, time.Duration) error
}

func (service *databaseService) CommitStoryboardReady(
	ctx context.Context,
	ready thumbnail.StoryboardReady,
) error {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return thumbnail.ErrRepositoryNotReady
	}
	return service.store.CommitStoryboardReady(ctx, ready)
}

func (service *databaseService) CommitStoryboardFailure(
	ctx context.Context,
	failure thumbnail.StoryboardFailure,
) error {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return thumbnail.ErrRepositoryNotReady
	}
	return service.store.CommitStoryboardFailure(ctx, failure)
}

func (service *databaseService) ReconcileMediaJobTransform(
	ctx context.Context,
	transformVersion int,
	limit int,
) (int64, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return 0, thumbnail.ErrRepositoryNotReady
	}
	return service.store.ReconcileMediaJobTransform(ctx, transformVersion, limit)
}

func (service *databaseService) ReconcileStoryboardJobTransform(
	ctx context.Context,
	transformVersion int,
	limit int,
) (int64, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return 0, thumbnail.ErrRepositoryNotReady
	}
	return service.store.ReconcileStoryboardJobTransform(
		ctx,
		transformVersion,
		limit,
	)
}

func (service *databaseService) AdmitStoryboardJobs(
	ctx context.Context,
	limit int,
) (int64, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return 0, thumbnail.ErrRepositoryNotReady
	}
	return service.store.AdmitStoryboardJobs(ctx, limit)
}

func (service *databaseService) RecoverExpiredMediaJobs(
	ctx context.Context,
) (jobs.RecoverySummary, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return jobs.RecoverySummary{}, thumbnail.ErrRepositoryNotReady
	}
	return service.store.RecoverExpiredMediaJobs(ctx)
}

func (service *databaseService) ClaimNextMediaJob(
	ctx context.Context,
	lease time.Duration,
) (thumbnail.Job, bool, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return thumbnail.Job{}, false, thumbnail.ErrRepositoryNotReady
	}
	return service.store.ClaimNextMediaJob(ctx, lease)
}

func (service *databaseService) RefreshMediaJobLease(
	ctx context.Context,
	job thumbnail.Job,
	lease time.Duration,
) error {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return thumbnail.ErrRepositoryNotReady
	}
	return service.store.RefreshMediaJobLease(ctx, job, lease)
}

type databaseOpener func(context.Context, string, sqlitestore.Options) (databaseStore, error)

type databaseService struct {
	dataRoot  string
	readiness *readinessState
	open      databaseOpener

	mutex sync.RWMutex
	store databaseStore
}

func newDatabaseComponent(dataRoot string, readiness *readinessState) (component, *databaseService) {
	service := &databaseService{
		dataRoot:  dataRoot,
		readiness: readiness,
		open: func(ctx context.Context, filename string, options sqlitestore.Options) (databaseStore, error) {
			return sqlitestore.Open(ctx, filename, options)
		},
	}
	return component{
		name:  "database",
		start: service.start,
		stop:  service.stop,
	}, service
}

func (service *databaseService) start(ctx context.Context) error {
	if err := prepareDataRoot(service.dataRoot); err != nil {
		service.readiness.set(api.Readiness{
			ReasonCode: api.ReadinessApplicationData,
		})
		return fmt.Errorf("prepare application data: %w", err)
	}

	store, err := service.open(
		ctx,
		filepath.Join(service.dataRoot, databaseFilename),
		sqlitestore.Options{},
	)
	if err != nil {
		reason := api.ReadinessDatabaseUnavailable
		if errors.Is(err, sqlitestore.ErrMigration) {
			reason = api.ReadinessMigrationFailed
		}
		service.readiness.set(api.Readiness{ReasonCode: reason})
		return fmt.Errorf("open application database: %w", err)
	}

	service.mutex.Lock()
	service.store = store
	service.mutex.Unlock()
	return nil
}

func (service *databaseService) stop(context.Context) error {
	service.mutex.Lock()
	store := service.store
	service.store = nil
	service.mutex.Unlock()
	if store == nil {
		return nil
	}
	return store.Close()
}

func (service *databaseService) AdministratorInitialized(ctx context.Context) (bool, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	store := service.store
	if store == nil {
		return false, auth.ErrRepositoryNotReady
	}
	return store.AdministratorInitialized(ctx)
}

func (service *databaseService) CreateAdministratorWithSession(
	ctx context.Context,
	params auth.CreateAdministratorParams,
	session auth.CreateSessionParams,
) (auth.Administrator, auth.StoredSession, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	store := service.store
	if store == nil {
		return auth.Administrator{}, auth.StoredSession{}, auth.ErrRepositoryNotReady
	}
	return store.CreateAdministratorWithSession(ctx, params, session)
}

func (service *databaseService) FindAdministratorCredential(
	ctx context.Context,
	usernameKey string,
) (auth.AdministratorCredential, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return auth.AdministratorCredential{}, auth.ErrRepositoryNotReady
	}
	return service.store.FindAdministratorCredential(ctx, usernameKey)
}

func (service *databaseService) CreateSession(
	ctx context.Context,
	params auth.CreateSessionParams,
	obsoleteCutoffMS int64,
) (auth.StoredSession, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return auth.StoredSession{}, auth.ErrRepositoryNotReady
	}
	return service.store.CreateSession(ctx, params, obsoleteCutoffMS)
}

func (service *databaseService) FindSession(
	ctx context.Context,
	tokenHash [32]byte,
) (auth.StoredSession, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return auth.StoredSession{}, auth.ErrRepositoryNotReady
	}
	return service.store.FindSession(ctx, tokenHash)
}

func (service *databaseService) TouchSession(
	ctx context.Context,
	params auth.TouchSessionParams,
) (bool, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return false, auth.ErrRepositoryNotReady
	}
	return service.store.TouchSession(ctx, params)
}

func (service *databaseService) RevokeSession(
	ctx context.Context,
	params auth.RevokeSessionParams,
) (bool, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return false, auth.ErrRepositoryNotReady
	}
	return service.store.RevokeSession(ctx, params)
}

func (service *databaseService) GetAccount(
	ctx context.Context,
	userID int64,
) (auth.Account, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return auth.Account{}, auth.ErrRepositoryNotReady
	}
	return service.store.GetAccount(ctx, userID)
}

func (service *databaseService) UpdateAccount(
	ctx context.Context,
	params auth.UpdateAccountParams,
) (auth.Account, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return auth.Account{}, auth.ErrRepositoryNotReady
	}
	return service.store.UpdateAccount(ctx, params)
}

func (service *databaseService) ChangePassword(
	ctx context.Context,
	params auth.ChangePasswordParams,
) (auth.Account, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return auth.Account{}, auth.ErrRepositoryNotReady
	}
	return service.store.ChangePassword(ctx, params)
}

func (service *databaseService) GetCacheCleanup(
	ctx context.Context,
) (thumbnail.Cleanup, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return thumbnail.Cleanup{}, thumbnail.ErrRepositoryNotReady
	}
	return service.store.GetCacheCleanup(ctx)
}

func (service *databaseService) RequestCacheCleanup(
	ctx context.Context,
	keyHash [32]byte,
	requestedAtMS int64,
) (thumbnail.CleanupRequestResult, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return thumbnail.CleanupRequestResult{}, thumbnail.ErrRepositoryNotReady
	}
	return service.store.RequestCacheCleanup(ctx, keyHash, requestedAtMS)
}

func (service *databaseService) ClaimCacheCleanup(
	ctx context.Context,
	usageBytes, startedAtMS int64,
) (thumbnail.Cleanup, bool, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return thumbnail.Cleanup{}, false, thumbnail.ErrRepositoryNotReady
	}
	return service.store.ClaimCacheCleanup(ctx, usageBytes, startedAtMS)
}

func (service *databaseService) UpdateCacheCleanupProgress(
	ctx context.Context,
	progress thumbnail.CleanupProgress,
) error {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return thumbnail.ErrRepositoryNotReady
	}
	return service.store.UpdateCacheCleanupProgress(ctx, progress)
}

func (service *databaseService) FinishCacheCleanup(
	ctx context.Context,
	status thumbnail.CleanupStatus,
	errorCode *string,
	finishedAtMS int64,
) (thumbnail.Cleanup, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return thumbnail.Cleanup{}, thumbnail.ErrRepositoryNotReady
	}
	return service.store.FinishCacheCleanup(ctx, status, errorCode, finishedAtMS)
}

func (service *databaseService) CreateLibrary(
	ctx context.Context,
	params library.CreateParams,
) (library.Library, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return library.Library{}, library.ErrRepositoryNotReady
	}
	return service.store.CreateLibrary(ctx, params)
}

func (service *databaseService) GetLibrary(
	ctx context.Context,
	id int64,
) (library.Library, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return library.Library{}, library.ErrRepositoryNotReady
	}
	return service.store.GetLibrary(ctx, id)
}

func (service *databaseService) ListLibraries(
	ctx context.Context,
) ([]library.Library, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return nil, library.ErrRepositoryNotReady
	}
	return service.store.ListLibraries(ctx)
}

func (service *databaseService) RenameLibrary(
	ctx context.Context,
	id int64,
	name string,
) (library.Library, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return library.Library{}, library.ErrRepositoryNotReady
	}
	return service.store.RenameLibrary(ctx, id, name)
}

func (service *databaseService) FindCreateReplay(
	ctx context.Context,
	keyHash [32]byte,
	requestHash [32]byte,
) (library.CreateResult, bool, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return library.CreateResult{}, false, library.ErrRepositoryNotReady
	}
	return service.store.FindCreateReplay(ctx, keyHash, requestHash)
}

func (service *databaseService) CreateLibraryWithScan(
	ctx context.Context,
	command library.CreateCommand,
) (library.CreateResult, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return library.CreateResult{}, library.ErrRepositoryNotReady
	}
	return service.store.CreateLibraryWithScan(ctx, command)
}

func (service *databaseService) ListLibraryPage(
	ctx context.Context,
	params library.ListParams,
) ([]library.Details, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return nil, library.ErrRepositoryNotReady
	}
	return service.store.ListLibraryPage(ctx, params)
}

func (service *databaseService) GetLibraryDetails(
	ctx context.Context,
	id int64,
) (library.Details, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return library.Details{}, library.ErrRepositoryNotReady
	}
	return service.store.GetLibraryDetails(ctx, id)
}

func (service *databaseService) RenameLibraryIfRevision(
	ctx context.Context,
	command library.RenameCommand,
) (library.Details, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return library.Details{}, library.ErrRepositoryNotReady
	}
	return service.store.RenameLibraryIfRevision(ctx, command)
}

func (service *databaseService) RequestLibraryRemoval(
	ctx context.Context,
	command library.RemoveCommand,
) (library.RemoveResult, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return library.RemoveResult{}, library.ErrRepositoryNotReady
	}
	return service.store.RequestLibraryRemoval(ctx, command)
}

func (service *databaseService) GetLibraryRemoval(
	ctx context.Context,
	id int64,
) (library.Removal, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return library.Removal{}, library.ErrRepositoryNotReady
	}
	return service.store.GetLibraryRemoval(ctx, id)
}

func (service *databaseService) ClaimNextLibraryRemoval(
	ctx context.Context,
) (library.Removal, bool, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return library.Removal{}, false, library.ErrRepositoryNotReady
	}
	return service.store.ClaimNextLibraryRemoval(ctx)
}

func (service *databaseService) LibraryRemovalReady(
	ctx context.Context,
	id int64,
) (bool, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return false, library.ErrRepositoryNotReady
	}
	return service.store.LibraryRemovalReady(ctx, id)
}

func (service *databaseService) CleanupLibraryRemovalBatch(
	ctx context.Context,
	id int64,
	limit int,
) (bool, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return false, library.ErrRepositoryNotReady
	}
	return service.store.CleanupLibraryRemovalBatch(ctx, id, limit)
}

func (service *databaseService) FailLibraryRemoval(
	ctx context.Context,
	id int64,
	code string,
) error {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return library.ErrRepositoryNotReady
	}
	return service.store.FailLibraryRemoval(ctx, id, code)
}

func (service *databaseService) AdmitFullScan(
	ctx context.Context,
	libraryID int64,
	trigger scanner.Trigger,
) (scanner.AdmissionResult, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return scanner.AdmissionResult{}, library.ErrRepositoryNotReady
	}
	return service.store.AdmitFullScan(ctx, libraryID, trigger)
}

func (service *databaseService) ListStartupLibraryIDs(
	ctx context.Context,
	afterID int64,
	limit int,
) ([]int64, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return nil, library.ErrRepositoryNotReady
	}
	return service.store.ListStartupLibraryIDs(ctx, afterID, limit)
}

func (service *databaseService) GetLibraryRoot(
	ctx context.Context,
	libraryID int64,
) (string, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return "", library.ErrRepositoryNotReady
	}
	return service.store.GetLibraryRoot(ctx, libraryID)
}

func (service *databaseService) BeginFullScan(
	ctx context.Context,
	libraryID int64,
	trigger scanner.Trigger,
) (scanner.ScanRun, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return scanner.ScanRun{}, library.ErrRepositoryNotReady
	}
	return service.store.BeginFullScan(ctx, libraryID, trigger)
}

func (service *databaseService) SetFullScanPhase(
	ctx context.Context,
	runID int64,
	phase string,
) error {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return library.ErrRepositoryNotReady
	}
	return service.store.SetFullScanPhase(ctx, runID, phase)
}

func (service *databaseService) UpsertCatalogBatch(
	ctx context.Context,
	runID int64,
	entries []scanner.CatalogEntry,
) error {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return library.ErrRepositoryNotReady
	}
	return service.store.UpsertCatalogBatch(ctx, runID, entries)
}

func (service *databaseService) CompleteFullScan(
	ctx context.Context,
	runID int64,
	skipped scanner.SkipCounts,
) (scanner.ScanRun, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return scanner.ScanRun{}, library.ErrRepositoryNotReady
	}
	return service.store.CompleteFullScan(ctx, runID, skipped)
}

func (service *databaseService) FailFullScan(
	ctx context.Context,
	runID int64,
	skipped scanner.SkipCounts,
	code string,
) (scanner.ScanRun, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return scanner.ScanRun{}, library.ErrRepositoryNotReady
	}
	return service.store.FailFullScan(ctx, runID, skipped, code)
}

func (service *databaseService) CancelFullScan(
	ctx context.Context,
	runID int64,
	skipped scanner.SkipCounts,
) (scanner.ScanRun, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return scanner.ScanRun{}, library.ErrRepositoryNotReady
	}
	return service.store.CancelFullScan(ctx, runID, skipped)
}

func (service *databaseService) OfflineFullScan(
	ctx context.Context,
	runID int64,
	skipped scanner.SkipCounts,
	code string,
) (scanner.ScanRun, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return scanner.ScanRun{}, library.ErrRepositoryNotReady
	}
	return service.store.OfflineFullScan(ctx, runID, skipped, code)
}

func (service *databaseService) InterruptActiveScans(
	ctx context.Context,
) (int64, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return 0, library.ErrRepositoryNotReady
	}
	return service.store.InterruptActiveScans(ctx)
}

func (service *databaseService) GetScanRun(
	ctx context.Context,
	runID int64,
) (scanner.ScanRun, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return scanner.ScanRun{}, library.ErrRepositoryNotReady
	}
	return service.store.GetScanRun(ctx, runID)
}

func (service *databaseService) ListScanRuns(
	ctx context.Context,
	libraryID int64,
	before scanner.QueryPosition,
	limit int,
) ([]scanner.ScanRun, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return nil, library.ErrRepositoryNotReady
	}
	return service.store.ListScanRuns(ctx, libraryID, before, limit)
}

func (service *databaseService) GetScanDetails(
	ctx context.Context,
	scanID int64,
) (scanner.Details, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return scanner.Details{}, library.ErrRepositoryNotReady
	}
	return service.store.GetScanDetails(ctx, scanID)
}

func (service *databaseService) RequestScanCancellation(
	ctx context.Context,
	scanID int64,
) (scanner.ScanRun, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return scanner.ScanRun{}, library.ErrRepositoryNotReady
	}
	return service.store.RequestScanCancellation(ctx, scanID)
}

func (service *databaseService) GetSettings(ctx context.Context) (appsettings.Values, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return appsettings.Values{}, library.ErrRepositoryNotReady
	}
	return service.store.GetSettings(ctx)
}

func (service *databaseService) UpdateSettings(
	ctx context.Context,
	expectedRevision int64,
	values appsettings.Values,
) (appsettings.Values, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return appsettings.Values{}, library.ErrRepositoryNotReady
	}
	return service.store.UpdateSettings(ctx, expectedRevision, values)
}

func (service *databaseService) GetScheduledScanIntervalHours(
	ctx context.Context,
) (*int64, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return nil, library.ErrRepositoryNotReady
	}
	return service.store.GetScheduledScanIntervalHours(ctx)
}

func (service *databaseService) ListDueLibraryIDs(
	ctx context.Context,
	dueBeforeMS int64,
	afterID int64,
	limit int,
) ([]int64, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return nil, library.ErrRepositoryNotReady
	}
	return service.store.ListDueLibraryIDs(ctx, dueBeforeMS, afterID, limit)
}

func (service *databaseService) RecoverExpiredFullScans(
	ctx context.Context,
) (jobs.RecoverySummary, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return jobs.RecoverySummary{}, library.ErrRepositoryNotReady
	}
	return service.store.RecoverExpiredFullScans(ctx)
}

func (service *databaseService) ClaimNextFullScan(
	ctx context.Context,
	lease time.Duration,
) (scanner.ScanRun, bool, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return scanner.ScanRun{}, false, library.ErrRepositoryNotReady
	}
	return service.store.ClaimNextFullScan(ctx, lease)
}

func (service *databaseService) RefreshFullScanLease(
	ctx context.Context,
	runID int64,
	lease time.Duration,
) (scanner.ScanRun, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return scanner.ScanRun{}, library.ErrRepositoryNotReady
	}
	return service.store.RefreshFullScanLease(ctx, runID, lease)
}

func prepareDataRoot(dataRoot string) error {
	if dataRoot == "" {
		return errors.New("data root is empty")
	}
	for _, directory := range []string{
		dataRoot,
		filepath.Join(dataRoot, "cache"),
		filepath.Join(dataRoot, "tmp"),
	} {
		if err := os.MkdirAll(directory, 0o750); err != nil {
			return fmt.Errorf("create data directory: %w", err)
		}
		info, err := os.Stat(directory)
		if err != nil {
			return fmt.Errorf("inspect data directory: %w", err)
		}
		if !info.IsDir() {
			return errors.New("application data path is not a directory")
		}
	}
	return nil
}
