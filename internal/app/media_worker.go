package app

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/HappyQuQu/foliopath/internal/jobs"
	"github.com/HappyQuQu/foliopath/internal/scanner"
	sqlitestore "github.com/HappyQuQu/foliopath/internal/store/sqlite"
	"github.com/HappyQuQu/foliopath/internal/thumbnail"
)

type mediaWakeScanRepository struct {
	scanner.Repository
	waker interface{ Wake() }
}

func (repository mediaWakeScanRepository) UpsertCatalogBatch(
	ctx context.Context,
	runID int64,
	entries []scanner.CatalogEntry,
) error {
	if err := repository.Repository.UpsertCatalogBatch(ctx, runID, entries); err != nil {
		return err
	}
	repository.waker.Wake()
	return nil
}

type mediaJobQueue struct {
	database *databaseService
}

func (queue mediaJobQueue) RecoverExpired(
	ctx context.Context,
) (jobs.RecoverySummary, error) {
	if _, err := queue.database.ReconcileMediaJobTransform(
		ctx, thumbnail.GridTransformVersion, 256,
	); err != nil {
		return jobs.RecoverySummary{}, err
	}
	if _, err := queue.database.ReconcileStoryboardJobTransform(
		ctx, thumbnail.StoryboardTransformVersion, 128,
	); err != nil {
		return jobs.RecoverySummary{}, err
	}
	if _, err := queue.database.AdmitStoryboardJobs(
		ctx, sqlitestore.MaxStoryboardAdmissionBatch,
	); err != nil {
		return jobs.RecoverySummary{}, err
	}
	return queue.database.RecoverExpiredMediaJobs(ctx)
}

func (queue mediaJobQueue) Claim(
	ctx context.Context,
	lease time.Duration,
) (thumbnail.Job, bool, error) {
	return queue.database.ClaimNextMediaJob(ctx, lease)
}

func (queue mediaJobQueue) RefreshLease(
	ctx context.Context,
	job thumbnail.Job,
	lease time.Duration,
) (bool, error) {
	err := queue.database.RefreshMediaJobLease(ctx, job, lease)
	if errors.Is(err, thumbnail.ErrJobNotActive) {
		return false, nil
	}
	return false, err
}

type mediaWorkerComponent struct {
	worker       *jobs.WorkerPool[thumbnail.Job]
	cacheManager *thumbnail.CacheManager
	cacheWake    <-chan struct{}

	mutex   sync.Mutex
	cancel  context.CancelFunc
	done    chan error
	stopped chan struct{}
}

func newMediaWorkerComponent(
	worker *jobs.WorkerPool[thumbnail.Job],
	cacheManager *thumbnail.CacheManager,
	cacheWake jobs.WakeSource,
) (component, error) {
	if worker == nil || cacheManager == nil || cacheWake == nil {
		return component{}, fmt.Errorf(
			"%w: media worker dependencies are required",
			errInvalidComponent,
		)
	}
	notifications := cacheWake.Notifications()
	if notifications == nil {
		return component{}, fmt.Errorf(
			"%w: cache notifications are required",
			errInvalidComponent,
		)
	}
	service := &mediaWorkerComponent{
		worker:       worker,
		cacheManager: cacheManager,
		cacheWake:    notifications,
		done:         make(chan error, 1),
		stopped:      make(chan struct{}),
	}
	return component{
		name:  "media-worker",
		start: service.start,
		done:  service.done,
		stop:  service.stop,
	}, nil
}

func (service *mediaWorkerComponent) start(context.Context) error {
	service.mutex.Lock()
	defer service.mutex.Unlock()
	if service.cancel != nil {
		return errors.New("media worker is already running")
	}
	ctx, cancel := context.WithCancel(context.Background())
	service.cancel = cancel
	go service.run(ctx, cancel)
	return nil
}

func (service *mediaWorkerComponent) run(
	ctx context.Context,
	cancel context.CancelFunc,
) {
	defer close(service.stopped)
	workerDone := make(chan error, 1)
	cacheDone := make(chan error, 1)
	go func() { workerDone <- service.worker.Run(ctx) }()
	go func() { cacheDone <- service.cacheManager.Run(ctx, service.cacheWake) }()

	select {
	case err := <-workerDone:
		cancel()
		<-cacheDone
		service.done <- err
	case err := <-cacheDone:
		cancel()
		<-workerDone
		service.done <- err
	}
}

func (service *mediaWorkerComponent) stop(ctx context.Context) error {
	service.mutex.Lock()
	cancel := service.cancel
	service.cancel = nil
	service.mutex.Unlock()
	if cancel == nil {
		return nil
	}
	cancel()
	select {
	case <-service.stopped:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
