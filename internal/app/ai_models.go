package app

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"sync"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
	"github.com/HappyQuQu/foliopath/internal/files"
)

type aiModelSource struct {
	path    string
	mu      sync.RWMutex
	root    *files.Root
	source  *files.ModelSource
	openErr error
}

func newAIModelSource(path string) (*aiModelSource, component) {
	service := &aiModelSource{path: path}
	return service, component{name: "ai-model-source", start: service.start, stop: service.stop}
}

func (service *aiModelSource) start(context.Context) error {
	if service.path == "" {
		service.openErr = aimodel.ErrModelSourceUnavailable
		return nil
	}
	root, err := files.OpenRoot(service.path)
	if err != nil {
		service.openErr = errors.Join(aimodel.ErrModelSourceUnavailable, err)
		return nil
	}
	source, err := files.NewModelSource(root)
	if err != nil {
		_ = root.Close()
		return err
	}
	service.mu.Lock()
	service.root, service.source, service.openErr = root, source, nil
	service.mu.Unlock()
	return nil
}

func (service *aiModelSource) stop(context.Context) error {
	service.mu.Lock()
	root := service.root
	service.root, service.source = nil, nil
	service.mu.Unlock()
	if root != nil {
		return root.Close()
	}
	return nil
}

func (service *aiModelSource) ScanModelPackages(ctx context.Context, packages, filesCount int, bytes int64) ([]aimodel.RawCandidate, bool, error) {
	service.mu.RLock()
	defer service.mu.RUnlock()
	if service.source == nil {
		return nil, false, service.unavailable()
	}
	return service.source.ScanModelPackages(ctx, packages, filesCount, bytes)
}

func (service *aiModelSource) OpenModelPackageFile(ctx context.Context, identity, name string) (io.ReadCloser, int64, error) {
	service.mu.RLock()
	defer service.mu.RUnlock()
	if service.source == nil {
		return nil, 0, service.unavailable()
	}
	return service.source.OpenModelPackageFile(ctx, identity, name)
}

func (service *aiModelSource) ValidateDirectModelSource(ctx context.Context, identity string) error {
	service.mu.RLock()
	defer service.mu.RUnlock()
	if service.source == nil {
		return service.unavailable()
	}
	return service.source.ValidateDirectModelSource(ctx, identity)
}

func (service *aiModelSource) ValidateDirectModelPackage(ctx context.Context, identity string, manifest aimodel.Manifest) error {
	service.mu.RLock()
	defer service.mu.RUnlock()
	if service.source == nil {
		return service.unavailable()
	}
	return service.source.ValidateDirectModelPackage(ctx, identity, manifest)
}

func (service *aiModelSource) OpenDirectRuntimeModelFile(ctx context.Context, identity, name string) (aimodel.RuntimeModelFile, error) {
	service.mu.RLock()
	defer service.mu.RUnlock()
	if service.source == nil {
		return nil, service.unavailable()
	}
	return service.source.OpenDirectRuntimeModelFile(ctx, identity, name)
}

func (service *aiModelSource) unavailable() error {
	if service.openErr != nil {
		return service.openErr
	}
	return errors.Join(aimodel.ErrModelSourceUnavailable, fs.ErrClosed)
}

type aiBackgroundWorker interface {
	Run(context.Context) error
}

type aiAvailabilityRefresher interface {
	Refresh(context.Context) (aimodel.AvailabilitySummary, error)
}

func newAIAvailabilityComponent(refresher aiAvailabilityRefresher) (component, error) {
	if refresher == nil {
		return component{}, errInvalidComponent
	}
	return component{
		name: "ai-model-availability",
		start: func(ctx context.Context) error {
			_, err := refresher.Refresh(ctx)
			return err
		},
		stop: func(context.Context) error { return nil },
	}, nil
}

type aiWorkerComponent struct {
	workers    []aiBackgroundWorker
	operations *aimodel.OperationService
	orphans    interface {
		Reconcile(context.Context, []string, bool) (aimodel.ManagedOrphanSummary, error)
	}
	store interface {
		Reconcile(context.Context) (files.ManagedModelReconcileReport, error)
	}
	mu      sync.Mutex
	cancel  context.CancelFunc
	done    chan error
	stopped chan struct{}
}

func newAIWorkerComponent(workers []aiBackgroundWorker, operations *aimodel.OperationService, orphans interface {
	Reconcile(context.Context, []string, bool) (aimodel.ManagedOrphanSummary, error)
}, store interface {
	Reconcile(context.Context) (files.ManagedModelReconcileReport, error)
}) (component, error) {
	if len(workers) == 0 || operations == nil || orphans == nil || store == nil {
		return component{}, errInvalidComponent
	}
	owned := append([]aiBackgroundWorker(nil), workers...)
	for _, worker := range owned {
		if worker == nil {
			return component{}, errInvalidComponent
		}
	}
	service := &aiWorkerComponent{workers: owned, operations: operations, orphans: orphans, store: store, done: make(chan error, len(owned)), stopped: make(chan struct{})}
	return component{name: "ai-model-workers", start: service.start, done: service.done, stop: service.stop}, nil
}

func (service *aiWorkerComponent) start(context.Context) error {
	service.mu.Lock()
	defer service.mu.Unlock()
	if service.cancel != nil {
		return errors.New("AI model install worker already running")
	}
	ctx, cancel := context.WithCancel(context.Background())
	if _, err := service.operations.RecoverInterrupted(ctx); err != nil {
		cancel()
		return err
	}
	report, err := service.store.Reconcile(ctx)
	if err != nil {
		cancel()
		return err
	}
	if _, err := service.orphans.Reconcile(ctx, report.FinalContentHashes, !report.Truncated); err != nil {
		cancel()
		return err
	}
	service.cancel = cancel
	var wait sync.WaitGroup
	wait.Add(len(service.workers))
	for _, worker := range service.workers {
		go func() {
			defer wait.Done()
			service.done <- worker.Run(ctx)
		}()
	}
	go func() {
		wait.Wait()
		close(service.stopped)
	}()
	return nil
}

func (service *aiWorkerComponent) stop(ctx context.Context) error {
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

var _ aimodel.CandidateSource = (*aiModelSource)(nil)
var _ aimodel.PackageSource = (*aiModelSource)(nil)

type managedAIModelStore struct {
	path  string
	mu    sync.RWMutex
	store *files.ManagedModelStore
}

func newManagedAIModelStore(path string) (*managedAIModelStore, component, error) {
	if path == "" {
		return nil, component{}, errors.New("managed AI model path is required")
	}
	service := &managedAIModelStore{path: path}
	return service, component{name: "managed-ai-model-store", start: service.start, stop: service.stop}, nil
}

func (service *managedAIModelStore) start(context.Context) error {
	store, err := files.NewManagedModelStore(service.path, 0)
	if err != nil {
		return err
	}
	service.mu.Lock()
	service.store = store
	service.mu.Unlock()
	return nil
}

func (service *managedAIModelStore) stop(context.Context) error {
	service.mu.Lock()
	service.store = nil
	service.mu.Unlock()
	return nil
}

func (service *managedAIModelStore) PublishModelPackage(ctx context.Context, model aimodel.VerifiedPackage, manifest aimodel.Manifest, opener aimodel.PackageOpener) (string, error) {
	service.mu.RLock()
	defer service.mu.RUnlock()
	if service.store == nil {
		return "", errAIRepositoryNotReady
	}
	return service.store.PublishModelPackage(ctx, model, manifest, opener)
}

func (service *managedAIModelStore) Reconcile(ctx context.Context) (files.ManagedModelReconcileReport, error) {
	service.mu.RLock()
	defer service.mu.RUnlock()
	if service.store == nil {
		return files.ManagedModelReconcileReport{}, errAIRepositoryNotReady
	}
	return service.store.Reconcile(ctx)
}

func (service *managedAIModelStore) ValidateManagedModelPackage(ctx context.Context, model aimodel.Model, manifest aimodel.Manifest) error {
	service.mu.RLock()
	defer service.mu.RUnlock()
	if service.store == nil {
		return errAIRepositoryNotReady
	}
	return service.store.ValidateManagedModelPackage(ctx, model, manifest)
}

func (service *managedAIModelStore) OpenManagedModelPackageFile(ctx context.Context, model aimodel.Model, name string) (io.ReadCloser, int64, error) {
	service.mu.RLock()
	defer service.mu.RUnlock()
	if service.store == nil {
		return nil, 0, errAIRepositoryNotReady
	}
	return service.store.OpenManagedModelPackageFile(ctx, model, name)
}

func (service *managedAIModelStore) OpenManagedRuntimeModelFile(ctx context.Context, model aimodel.Model, name string) (aimodel.RuntimeModelFile, error) {
	service.mu.RLock()
	defer service.mu.RUnlock()
	if service.store == nil {
		return nil, errAIRepositoryNotReady
	}
	return service.store.OpenManagedRuntimeModelFile(ctx, model, name)
}

var _ aimodel.ManagedPublisher = (*managedAIModelStore)(nil)
var _ aimodel.ManagedActivationSource = (*managedAIModelStore)(nil)
var _ aimodel.DirectActivationSource = (*aiModelSource)(nil)
