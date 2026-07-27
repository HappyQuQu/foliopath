package app

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/HappyQuQu/foliopath/internal/jobs"
	"github.com/HappyQuQu/foliopath/internal/scanner"
)

type scanWorkerComponent struct {
	worker *jobs.WorkerPool[scanner.ScanRun]

	mutex   sync.Mutex
	cancel  context.CancelFunc
	done    chan error
	stopped chan struct{}
}

func newScanWorkerComponent(worker *jobs.WorkerPool[scanner.ScanRun]) (component, error) {
	if worker == nil {
		return component{}, fmt.Errorf("%w: scan worker is required", errInvalidComponent)
	}
	service := &scanWorkerComponent{
		worker:  worker,
		done:    make(chan error, 1),
		stopped: make(chan struct{}),
	}
	return component{
		name:  "scan-worker",
		start: service.start,
		done:  service.done,
		stop:  service.stop,
	}, nil
}

type scanJobQueue struct {
	database *databaseService
}

func (queue scanJobQueue) RecoverExpired(
	ctx context.Context,
) (jobs.RecoverySummary, error) {
	return queue.database.RecoverExpiredFullScans(ctx)
}

func (queue scanJobQueue) Claim(
	ctx context.Context,
	lease time.Duration,
) (scanner.ScanRun, bool, error) {
	return queue.database.ClaimNextFullScan(ctx, lease)
}

func (queue scanJobQueue) RefreshLease(
	ctx context.Context,
	run scanner.ScanRun,
	lease time.Duration,
) (bool, error) {
	refreshed, err := queue.database.RefreshFullScanLease(ctx, run.ID, lease)
	if err != nil {
		if errors.Is(err, scanner.ErrScanRunNotActive) {
			return false, nil
		}
		return false, err
	}
	return refreshed.CancelRequestedAtMS != nil, nil
}

func (service *scanWorkerComponent) start(context.Context) error {
	service.mutex.Lock()
	defer service.mutex.Unlock()
	if service.cancel != nil {
		return fmt.Errorf("scan worker is already running")
	}
	ctx, cancel := context.WithCancel(context.Background())
	service.cancel = cancel
	go func() {
		err := service.worker.Run(ctx)
		close(service.stopped)
		service.done <- err
	}()
	return nil
}

func (service *scanWorkerComponent) stop(ctx context.Context) error {
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
