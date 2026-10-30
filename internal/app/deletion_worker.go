package app

import (
	"context"
	"errors"
	"sync"

	"github.com/HappyQuQu/foliopath/internal/deletion"
	"github.com/HappyQuQu/foliopath/internal/jobs"
	"github.com/HappyQuQu/foliopath/internal/scanner"
)

type deletionReconciler struct {
	admission *scanner.AdmissionService
	queries   *scanner.QueryService
}

func (r deletionReconciler) Request(ctx context.Context, libraryID int64) (int64, error) {
	result, err := r.admission.RequestManual(ctx, libraryID)
	if errors.Is(err, scanner.ErrAdmissionCapacity) || errors.Is(err, scanner.ErrAdmissionConflict) {
		return 0, deletion.ErrBusy
	}
	return result.Run.ID, err
}
func (r deletionReconciler) Status(ctx context.Context, id int64) (bool, bool, error) {
	details, err := r.queries.Get(ctx, id)
	if err != nil {
		return false, false, err
	}
	switch details.Run.Status {
	case scanner.RunStatusQueued, scanner.RunStatusRunning:
		return false, false, nil
	case scanner.RunStatusSucceeded:
		return true, true, nil
	default:
		return true, false, nil
	}
}

type deletionWorkerComponent struct {
	worker  *jobs.WorkerPool[deletion.Claim]
	mutex   sync.Mutex
	parent  context.Context
	cancel  context.CancelCauseFunc
	done    chan error
	stopped chan struct{}
	runErr  error
}

func newDeletionWorkerComponent(worker *jobs.WorkerPool[deletion.Claim]) component {
	service := &deletionWorkerComponent{worker: worker, done: make(chan error, 1), stopped: make(chan struct{})}
	return component{name: "media-deletion-worker", start: service.start, stop: service.stop, done: service.done}
}
func (service *deletionWorkerComponent) start(parent context.Context) error {
	service.mutex.Lock()
	defer service.mutex.Unlock()
	if service.cancel != nil {
		return errors.New("media deletion worker already running")
	}
	ctx, cancel := context.WithCancelCause(context.WithoutCancel(parent))
	service.cancel = cancel
	service.parent = parent
	go func() {
		err := service.worker.Run(ctx)
		service.mutex.Lock()
		service.runErr = err
		service.mutex.Unlock()
		service.done <- err
		close(service.stopped)
	}()
	return nil
}
func (service *deletionWorkerComponent) stop(ctx context.Context) error {
	service.mutex.Lock()
	cancel := service.cancel
	service.cancel = nil
	var cause error
	if service.parent != nil {
		cause = context.Cause(service.parent)
	}
	service.mutex.Unlock()
	if cancel == nil {
		return nil
	}
	cancel(cause)
	select {
	case <-service.stopped:
		service.mutex.Lock()
		defer service.mutex.Unlock()
		return service.runErr
	case <-ctx.Done():
		return ctx.Err()
	}
}
