package app

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/HappyQuQu/foliopath/internal/files"
	"github.com/HappyQuQu/foliopath/internal/jobs"
	"github.com/HappyQuQu/foliopath/internal/library"
	"github.com/HappyQuQu/foliopath/internal/scanner"
)

type reconcileQueueStore interface {
	RecoverExpiredReconciles(context.Context) (jobs.RecoverySummary, error)
	ClaimNextReconcile(context.Context, time.Duration) (scanner.ReconcileJob, bool, error)
	RefreshReconcileLease(
		context.Context,
		scanner.ReconcileJob,
		time.Duration,
	) (scanner.ReconcileJob, error)
}

func (service *databaseService) RecoverExpiredReconciles(
	ctx context.Context,
) (jobs.RecoverySummary, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return jobs.RecoverySummary{}, scanner.ErrDatabaseUnavailable
	}
	return service.store.RecoverExpiredReconciles(ctx)
}

func (service *databaseService) ClaimNextReconcile(
	ctx context.Context,
	lease time.Duration,
) (scanner.ReconcileJob, bool, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return scanner.ReconcileJob{}, false, scanner.ErrDatabaseUnavailable
	}
	return service.store.ClaimNextReconcile(ctx, lease)
}

func (service *databaseService) RefreshReconcileLease(
	ctx context.Context,
	job scanner.ReconcileJob,
	lease time.Duration,
) (scanner.ReconcileJob, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return scanner.ReconcileJob{}, scanner.ErrDatabaseUnavailable
	}
	return service.store.RefreshReconcileLease(ctx, job, lease)
}

type reconcileJobQueue struct {
	database *databaseService
}

func (queue reconcileJobQueue) RecoverExpired(
	ctx context.Context,
) (jobs.RecoverySummary, error) {
	return queue.database.RecoverExpiredReconciles(ctx)
}

func (queue reconcileJobQueue) Claim(
	ctx context.Context,
	lease time.Duration,
) (scanner.ReconcileJob, bool, error) {
	return queue.database.ClaimNextReconcile(ctx, lease)
}

func (queue reconcileJobQueue) RefreshLease(
	ctx context.Context,
	job scanner.ReconcileJob,
	lease time.Duration,
) (bool, error) {
	_, err := queue.database.RefreshReconcileLease(ctx, job, lease)
	if errors.Is(err, scanner.ErrReconcileNotActive) {
		return false, nil
	}
	return false, err
}

type automaticDiscoveryCoordinator struct {
	database      *databaseService
	mediaRoot     *mediaRootService
	admission     *scanner.ReconcileAdmission
	scanAdmission *scanner.AdmissionService
	newWatcher    func(files.WatcherOptions) (files.LibraryWatcher, error)
	configWake    <-chan struct{}
	recoveryWake  <-chan struct{}

	mu      sync.Mutex
	watcher files.LibraryWatcher
	active  map[int64]string
	blocked map[int64]int64
}

func newAutomaticDiscoveryCoordinator(
	database *databaseService,
	mediaRoot *mediaRootService,
	admission *scanner.ReconcileAdmission,
	scanAdmission *scanner.AdmissionService,
	configWake jobs.WakeSource,
	recoveryWake jobs.WakeSource,
) (*automaticDiscoveryCoordinator, error) {
	if database == nil || mediaRoot == nil || admission == nil ||
		scanAdmission == nil || configWake == nil || recoveryWake == nil {
		return nil, errors.New("automatic discovery coordinator dependencies are required")
	}
	configNotifications := configWake.Notifications()
	recoveryNotifications := recoveryWake.Notifications()
	if configNotifications == nil || recoveryNotifications == nil {
		return nil, errors.New("automatic discovery notifications are required")
	}
	return &automaticDiscoveryCoordinator{
		database:      database,
		mediaRoot:     mediaRoot,
		admission:     admission,
		scanAdmission: scanAdmission,
		newWatcher: func(options files.WatcherOptions) (files.LibraryWatcher, error) {
			return mediaRoot.newLibraryWatcher(options)
		},
		configWake:   configNotifications,
		recoveryWake: recoveryNotifications,
		active:       make(map[int64]string),
		blocked:      make(map[int64]int64),
	}, nil
}

func (coordinator *automaticDiscoveryCoordinator) Run(ctx context.Context) error {
	watcher, err := coordinator.newWatcher(files.WatcherOptions{})
	if err != nil && !errors.Is(err, files.ErrWatchUnsupported) {
		return err
	}
	coordinator.mu.Lock()
	coordinator.watcher = watcher
	coordinator.mu.Unlock()
	if watcher != nil {
		defer watcher.Close()
	}
	if err := coordinator.reconfigure(ctx); err != nil {
		return err
	}

	var watcherDone <-chan error
	if watcher != nil {
		done := make(chan error, 1)
		go func() { done <- watcher.Run(ctx) }()
		watcherDone = done
	}
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-watcherDone:
			if err == nil && ctx.Err() != nil {
				return nil
			}
			return err
		case event := <-coordinator.events():
			if err := coordinator.handleEvent(ctx, event); err != nil {
				return err
			}
		case <-coordinator.configWake:
			if err := coordinator.reconfigure(ctx); err != nil {
				return err
			}
		case <-coordinator.recoveryWake:
			if err := coordinator.reconfigure(ctx); err != nil {
				return err
			}
		case <-ticker.C:
			if err := coordinator.reconfigure(ctx); err != nil {
				return err
			}
		}
	}
}

func (coordinator *automaticDiscoveryCoordinator) events() <-chan scanner.WatchEvent {
	coordinator.mu.Lock()
	defer coordinator.mu.Unlock()
	if coordinator.watcher == nil {
		return nil
	}
	return coordinator.watcher.Events()
}

func (coordinator *automaticDiscoveryCoordinator) reconfigure(ctx context.Context) error {
	settings, err := coordinator.database.GetSettings(ctx)
	if err != nil {
		return err
	}
	libraries, err := coordinator.database.ListLibraries(ctx)
	if err != nil {
		return err
	}
	current := make(map[int64]library.Library, len(libraries))
	for _, item := range libraries {
		current[item.ID] = item
	}

	coordinator.mu.Lock()
	watcher := coordinator.watcher
	for id := range coordinator.active {
		if _, exists := current[id]; !exists || !settings.AutomaticDiscoveryEnabled {
			if watcher != nil {
				_ = watcher.UnwatchLibrary(id)
			}
			delete(coordinator.active, id)
		}
	}
	for id := range coordinator.blocked {
		if _, exists := current[id]; !exists || !settings.AutomaticDiscoveryEnabled {
			delete(coordinator.blocked, id)
		}
	}
	coordinator.mu.Unlock()

	for _, item := range libraries {
		if !settings.AutomaticDiscoveryEnabled {
			if err := coordinator.database.SetAutomaticDiscoveryState(
				ctx,
				item.ID,
				string(library.AutomaticDiscoveryDisabled),
				"",
			); err != nil && !errors.Is(err, library.ErrNotFound) {
				return err
			}
			continue
		}
		if watcher == nil {
			if err := coordinator.database.SetAutomaticDiscoveryState(
				ctx,
				item.ID,
				string(library.AutomaticDiscoveryUnsupported),
				"watch_unavailable",
			); err != nil && !errors.Is(err, library.ErrNotFound) {
				return err
			}
			continue
		}
		// A successful full generation is the baseline that gives targeted
		// reconciliation a reliable directory tree and cleanup boundary.
		if item.CurrentGeneration < 1 {
			continue
		}
		coordinator.mu.Lock()
		_, active := coordinator.active[item.ID]
		blockedGeneration, blocked := coordinator.blocked[item.ID]
		if blocked && item.CurrentGeneration > blockedGeneration {
			delete(coordinator.blocked, item.ID)
			blocked = false
		}
		coordinator.mu.Unlock()
		if active || blocked {
			continue
		}
		if err := watcher.WatchLibrary(ctx, item.ID, item.RootRelativePath); err != nil {
			code := "watch_unavailable"
			if errors.Is(err, files.ErrWatchResourceLimit) {
				code = "watch_resource_limit"
			} else if errors.Is(err, files.ErrOffline) {
				code = "source_unavailable"
			}
			if degradeErr := coordinator.degradeAndFallback(
				ctx,
				item.ID,
				code,
			); degradeErr != nil {
				return errors.Join(err, degradeErr)
			}
			continue
		}
		coordinator.mu.Lock()
		coordinator.active[item.ID] = item.RootRelativePath
		coordinator.mu.Unlock()
		if err := coordinator.database.SetAutomaticDiscoveryState(
			ctx,
			item.ID,
			string(library.AutomaticDiscoveryActive),
			"",
		); err != nil && !errors.Is(err, library.ErrNotFound) {
			return err
		}
	}
	return nil
}

func (coordinator *automaticDiscoveryCoordinator) handleEvent(
	ctx context.Context,
	event scanner.WatchEvent,
) error {
	if event.Kind == scanner.WatchEventOverflow {
		coordinator.mu.Lock()
		ids := make([]int64, 0, len(coordinator.active))
		for id := range coordinator.active {
			ids = append(ids, id)
		}
		coordinator.mu.Unlock()
		for _, id := range ids {
			if err := coordinator.degradeAndFallback(
				ctx,
				id,
				"watch_overflow",
			); err != nil {
				return err
			}
		}
		return nil
	}
	if event.LibraryID <= 0 {
		return errors.New("watcher returned an invalid library ID")
	}
	if event.Kind == scanner.WatchEventInvalidated {
		return coordinator.degradeAndFallback(
			ctx,
			event.LibraryID,
			"source_unavailable",
		)
	}
	if event.Kind != scanner.WatchEventDirty {
		return errors.New("watcher returned an invalid event kind")
	}
	_, err := coordinator.admission.MarkDirty(
		ctx,
		event.LibraryID,
		event.RelativeDirectory,
	)
	if errors.Is(err, library.ErrNotFound) {
		return nil
	}
	if errors.Is(err, scanner.ErrReconcileCapacity) {
		return coordinator.degradeAndFallback(
			ctx,
			event.LibraryID,
			"watch_overflow",
		)
	}
	return err
}

func (coordinator *automaticDiscoveryCoordinator) DirectoryDiscovered(
	ctx context.Context,
	libraryID int64,
	rootRelative string,
	relativeDirectory string,
) error {
	coordinator.mu.Lock()
	watcher := coordinator.watcher
	coordinator.mu.Unlock()
	if watcher == nil {
		return files.ErrWatchUnsupported
	}
	if err := watcher.WatchDirectory(
		ctx,
		libraryID,
		rootRelative,
		relativeDirectory,
	); err != nil {
		code := "watch_unavailable"
		if errors.Is(err, files.ErrWatchResourceLimit) {
			code = "watch_resource_limit"
		}
		return coordinator.degradeAndFallback(ctx, libraryID, code)
	}
	_, err := coordinator.admission.MarkDirty(ctx, libraryID, relativeDirectory)
	if errors.Is(err, scanner.ErrReconcileCapacity) {
		return coordinator.degradeAndFallback(
			ctx,
			libraryID,
			"watch_overflow",
		)
	}
	return err
}

func (coordinator *automaticDiscoveryCoordinator) ReconcileFailed(
	ctx context.Context,
	job scanner.ReconcileJob,
	code string,
) error {
	libraryCode := "internal_error"
	if code == "source_unavailable" || code == "source_changed" ||
		code == "source_unreadable" {
		libraryCode = "source_unavailable"
	}
	return coordinator.degradeAndFallback(ctx, job.LibraryID, libraryCode)
}

func (coordinator *automaticDiscoveryCoordinator) degradeAndFallback(
	ctx context.Context,
	libraryID int64,
	errorCode string,
) error {
	item, err := coordinator.database.GetLibrary(ctx, libraryID)
	if errors.Is(err, library.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	coordinator.mu.Lock()
	watcher := coordinator.watcher
	if watcher != nil {
		_ = watcher.UnwatchLibrary(libraryID)
	}
	delete(coordinator.active, libraryID)
	coordinator.blocked[libraryID] = item.CurrentGeneration
	coordinator.mu.Unlock()
	if err := coordinator.database.SetAutomaticDiscoveryState(
		ctx,
		libraryID,
		string(library.AutomaticDiscoveryDegraded),
		errorCode,
	); err != nil && !errors.Is(err, library.ErrNotFound) {
		return err
	}
	_, err = coordinator.scanAdmission.RequestAutomaticFallback(ctx, libraryID)
	if errors.Is(err, scanner.ErrAdmissionConflict) ||
		errors.Is(err, scanner.ErrAdmissionCapacity) ||
		errors.Is(err, scanner.ErrLibraryNotFound) {
		return nil
	}
	return err
}

type automaticDiscoveryComponent struct {
	worker      *jobs.WorkerPool[scanner.ReconcileJob]
	coordinator *automaticDiscoveryCoordinator

	mu      sync.Mutex
	cancel  context.CancelFunc
	done    chan error
	stopped chan struct{}
}

func newAutomaticDiscoveryComponent(
	worker *jobs.WorkerPool[scanner.ReconcileJob],
	coordinator *automaticDiscoveryCoordinator,
) (component, error) {
	if worker == nil || coordinator == nil {
		return component{}, fmt.Errorf(
			"%w: automatic discovery dependencies are required",
			errInvalidComponent,
		)
	}
	service := &automaticDiscoveryComponent{
		worker:      worker,
		coordinator: coordinator,
		done:        make(chan error, 1),
		stopped:     make(chan struct{}),
	}
	return component{
		name:  "automatic-discovery",
		start: service.start,
		done:  service.done,
		stop:  service.stop,
	}, nil
}

func (service *automaticDiscoveryComponent) start(context.Context) error {
	service.mu.Lock()
	defer service.mu.Unlock()
	if service.cancel != nil {
		return errors.New("automatic discovery is already running")
	}
	ctx, cancel := context.WithCancel(context.Background())
	service.cancel = cancel
	go service.run(ctx, cancel)
	return nil
}

func (service *automaticDiscoveryComponent) run(
	ctx context.Context,
	cancel context.CancelFunc,
) {
	defer close(service.stopped)
	workerDone := make(chan error, 1)
	coordinatorDone := make(chan error, 1)
	go func() { workerDone <- service.worker.Run(ctx) }()
	go func() { coordinatorDone <- service.coordinator.Run(ctx) }()
	select {
	case err := <-workerDone:
		cancel()
		<-coordinatorDone
		service.done <- err
	case err := <-coordinatorDone:
		cancel()
		<-workerDone
		service.done <- err
	}
}

func (service *automaticDiscoveryComponent) stop(ctx context.Context) error {
	service.mu.Lock()
	cancel := service.cancel
	service.cancel = nil
	service.mu.Unlock()
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

type multiWaker []interface{ Wake() }

func (wakers multiWaker) Wake() {
	for _, waker := range wakers {
		waker.Wake()
	}
}
