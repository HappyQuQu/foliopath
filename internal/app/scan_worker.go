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
	worker    *jobs.WorkerPool[scanner.ScanRun]
	admission *scanner.AdmissionService

	mutex   sync.Mutex
	cancel  context.CancelFunc
	done    chan error
	stopped chan struct{}
}

func newScanWorkerComponent(
	worker *jobs.WorkerPool[scanner.ScanRun],
	admission *scanner.AdmissionService,
) (component, error) {
	if worker == nil || admission == nil {
		return component{}, fmt.Errorf(
			"%w: scan worker and startup admission are required",
			errInvalidComponent,
		)
	}
	service := &scanWorkerComponent{
		worker:    worker,
		admission: admission,
		done:      make(chan error, 1),
		stopped:   make(chan struct{}),
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
	go service.run(ctx, cancel)
	return nil
}

func (service *scanWorkerComponent) run(ctx context.Context, cancel context.CancelFunc) {
	defer close(service.stopped)
	workerDone := make(chan error, 1)
	startupDone := make(chan error, 1)
	go func() {
		workerDone <- service.worker.Run(ctx)
	}()
	go func() {
		_, err := service.admission.RequestStartup(ctx)
		startupDone <- err
	}()

	select {
	case startupErr := <-startupDone:
		if startupErr != nil {
			cancel()
			<-workerDone
			service.done <- fmt.Errorf("admit startup scans: %w", startupErr)
			return
		}
		service.done <- <-workerDone
	case workerErr := <-workerDone:
		cancel()
		<-startupDone
		service.done <- workerErr
	}
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
