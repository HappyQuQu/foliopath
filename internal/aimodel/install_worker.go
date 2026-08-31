package aimodel

import (
	"context"
	"errors"
	"io/fs"
	"time"
)

type InstallExecutor interface {
	Install(context.Context, Candidate, StorageMode) (Model, bool, error)
}

type InstallWorker struct {
	queue         InstallQueue
	installer     InstallExecutor
	operations    *OperationService
	notifications <-chan struct{}
	pollInterval  time.Duration
}

func NewInstallWorker(
	queue InstallQueue,
	installer InstallExecutor,
	operations *OperationService,
	notifications <-chan struct{},
	pollInterval time.Duration,
) (*InstallWorker, error) {
	if queue == nil || installer == nil || operations == nil || notifications == nil {
		return nil, errors.New("AI model install worker dependencies are required")
	}
	if pollInterval == 0 {
		pollInterval = time.Second
	}
	if pollInterval < 10*time.Millisecond || pollInterval > time.Minute {
		return nil, errors.New("AI model install worker poll interval is invalid")
	}
	return &InstallWorker{queue: queue, installer: installer, operations: operations, notifications: notifications, pollInterval: pollInterval}, nil
}

func (worker *InstallWorker) Run(ctx context.Context) error {
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-worker.notifications:
		case <-timer.C:
		}
		for {
			work, found, err := worker.queue.ClaimAIModelInstall(ctx, time.Now().UTC())
			if err != nil {
				if ctx.Err() != nil {
					return nil
				}
				return err
			}
			if !found {
				timer.Reset(worker.pollInterval)
				break
			}
			if err := validateClaimedInstallWork(work); err != nil {
				return err
			}
			worker.process(ctx, work)
		}
	}
}

func validateClaimedInstallWork(work InstallWork) error {
	if len(work.IdempotencyKey) < 8 || len(work.IdempotencyKey) > 128 ||
		work.RequestHash != InstallRequestHash(work.CandidateID, work.StorageMode) ||
		validateInstallWork(work, work.CandidateID, work.StorageMode, work.IdempotencyKey) != nil ||
		work.Operation.State != OperationRunning || work.Operation.Phase != PhaseVerifying ||
		work.Operation.Revision != 2 || work.Operation.CompletedItems != 0 ||
		work.Operation.TotalItems != nil || work.Operation.LibraryID != 0 {
		return ErrRepositoryState
	}
	return nil
}

func (worker *InstallWorker) process(parent context.Context, work InstallWork) {
	ctx, cancel := context.WithCancel(parent)
	done := make(chan struct{})
	go worker.watchCancellation(ctx, cancel, work.Operation.ID, done)
	_, _, installErr := worker.installer.Install(ctx, work.Candidate, work.StorageMode)
	cancel()
	<-done
	current, err := worker.operations.Get(context.WithoutCancel(parent), work.Operation.ID)
	if err != nil {
		return
	}
	if current.State == OperationCancelling {
		settleWorkerFailure(context.WithoutCancel(parent), worker.operations, current.ID, "internal_error")
		return
	}
	if parent.Err() != nil {
		return
	}
	if installErr != nil {
		settleWorkerFailure(parent, worker.operations, current.ID, installErrorCode(installErr))
		return
	}
	operationID := current.ID
	current, err = worker.operations.Progress(parent, operationID, current.Revision, PhaseFinalizing, 0, nil)
	if err != nil {
		worker.settleFinalizationFailure(parent, operationID)
		return
	}
	if _, err := worker.operations.Succeed(parent, current.ID, current.Revision); err != nil {
		worker.settleFinalizationFailure(parent, operationID)
	}
}

func (worker *InstallWorker) settleFinalizationFailure(parent context.Context, operationID string) {
	if parent.Err() != nil {
		return
	}
	settleWorkerFailure(context.WithoutCancel(parent), worker.operations, operationID, "internal_error")
}

func (worker *InstallWorker) watchCancellation(ctx context.Context, cancel context.CancelFunc, operationID string, done chan<- struct{}) {
	defer close(done)
	ticker := time.NewTicker(worker.pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			operation, err := worker.operations.Get(ctx, operationID)
			if err == nil && operation.State == OperationCancelling {
				cancel()
				return
			}
		}
	}
}

func installErrorCode(err error) string {
	switch {
	case errors.Is(err, context.Canceled):
		return "cancelled"
	case errors.Is(err, ErrModelIncompatible), errors.Is(err, ErrInvalidModel):
		return "model_incompatible"
	case errors.Is(err, ErrInsufficientSpace):
		return "insufficient_space"
	case errors.Is(err, fs.ErrPermission):
		return "model_source_unsafe"
	case errors.Is(err, fs.ErrNotExist):
		return "model_source_unavailable"
	default:
		return "internal_error"
	}
}

var _ InstallExecutor = (*Installer)(nil)
