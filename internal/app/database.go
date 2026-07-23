package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/HappyQuQu/foliopath/internal/api"
	"github.com/HappyQuQu/foliopath/internal/auth"
	sqlitestore "github.com/HappyQuQu/foliopath/internal/store/sqlite"
)

const databaseFilename = "foliopath.db"

type databaseStore interface {
	auth.Repository
	Close() error
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

func (service *databaseService) CreateAdministrator(
	ctx context.Context,
	params auth.CreateAdministratorParams,
) (auth.Administrator, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	store := service.store
	if store == nil {
		return auth.Administrator{}, auth.ErrRepositoryNotReady
	}
	return store.CreateAdministrator(ctx, params)
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
