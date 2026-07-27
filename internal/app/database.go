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
	"github.com/HappyQuQu/foliopath/internal/jobs"
	"github.com/HappyQuQu/foliopath/internal/library"
	"github.com/HappyQuQu/foliopath/internal/scanner"
	appsettings "github.com/HappyQuQu/foliopath/internal/settings"
	sqlitestore "github.com/HappyQuQu/foliopath/internal/store/sqlite"
)

const databaseFilename = "foliopath.db"

type databaseStore interface {
	auth.Repository
	library.Repository
	library.LifecycleRepository
	library.RemovalRepository
	scanner.AdmissionRepository
	scanner.Repository
	scanner.QueryRepository
	scanner.ScheduleRepository
	appsettings.Repository
	scanQueueStore
	Close() error
}

type scanQueueStore interface {
	RecoverExpiredFullScans(context.Context) (jobs.RecoverySummary, error)
	ClaimNextFullScan(context.Context, time.Duration) (scanner.ScanRun, bool, error)
	RefreshFullScanLease(context.Context, int64, time.Duration) (scanner.ScanRun, error)
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
