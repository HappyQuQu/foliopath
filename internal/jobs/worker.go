package jobs

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

const (
	DefaultWorkerCount       = 2
	DefaultHeartbeatInterval = 15 * time.Second
	DefaultLeaseDuration     = 120 * time.Second
	defaultIdlePollInterval  = time.Second
)

type RecoverySummary struct {
	Requeued    int64
	Interrupted int64
}

// LeaseQueue owns generic durable claim, lease, cancellation, and recovery
// transitions. The durable store remains the queue; implementations must not
// rely on WakeSource notifications for correctness.
type LeaseQueue[T any] interface {
	RecoverExpired(context.Context) (RecoverySummary, error)
	Claim(context.Context, time.Duration) (T, bool, error)
	RefreshLease(context.Context, T, time.Duration) (cancelRequested bool, err error)
}

type Processor[T any] interface {
	Process(context.Context, T) error
}

type WakeSource interface {
	Notifications() <-chan struct{}
}

type WorkerOptions struct {
	Workers           int
	HeartbeatInterval time.Duration
	LeaseDuration     time.Duration
	IdlePollInterval  time.Duration
}

type WorkerPool[T any] struct {
	queue             LeaseQueue[T]
	processor         Processor[T]
	notifications     <-chan struct{}
	workers           int
	heartbeatInterval time.Duration
	leaseDuration     time.Duration
	idlePollInterval  time.Duration
}

func NewWorkerPool[T any](
	queue LeaseQueue[T],
	processor Processor[T],
	wakeSource WakeSource,
	options WorkerOptions,
) (*WorkerPool[T], error) {
	if queue == nil || processor == nil || wakeSource == nil {
		return nil, errors.New("worker dependencies are required")
	}
	if options.Workers == 0 {
		options.Workers = DefaultWorkerCount
	}
	if options.Workers < 1 || options.Workers > DefaultWorkerCount {
		return nil, fmt.Errorf("workers must be between 1 and %d", DefaultWorkerCount)
	}
	if options.HeartbeatInterval == 0 {
		options.HeartbeatInterval = DefaultHeartbeatInterval
	}
	if options.LeaseDuration == 0 {
		options.LeaseDuration = DefaultLeaseDuration
	}
	if options.IdlePollInterval == 0 {
		options.IdlePollInterval = defaultIdlePollInterval
	}
	if options.HeartbeatInterval <= 0 ||
		options.LeaseDuration <= options.HeartbeatInterval ||
		options.IdlePollInterval <= 0 {
		return nil, errors.New("worker timing configuration is invalid")
	}
	notifications := wakeSource.Notifications()
	if notifications == nil {
		return nil, errors.New("worker notifications are required")
	}
	return &WorkerPool[T]{
		queue:             queue,
		processor:         processor,
		notifications:     notifications,
		workers:           options.Workers,
		heartbeatInterval: options.HeartbeatInterval,
		leaseDuration:     options.LeaseDuration,
		idlePollInterval:  options.IdlePollInterval,
	}, nil
}

func (pool *WorkerPool[T]) Run(ctx context.Context) error {
	if ctx == nil {
		return errors.New("worker context is required")
	}
	if _, err := pool.queue.RecoverExpired(ctx); err != nil {
		return fmt.Errorf("recover expired work: %w", err)
	}

	workerContext, cancel := context.WithCancel(ctx)
	defer cancel()
	failures := make(chan error, pool.workers+1)
	var workers sync.WaitGroup
	workers.Add(1)
	go func() {
		defer workers.Done()
		if err := pool.runRecovery(workerContext); err != nil {
			select {
			case failures <- err:
			case <-workerContext.Done():
			}
		}
	}()
	for index := 0; index < pool.workers; index++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			if err := pool.runWorker(workerContext); err != nil {
				select {
				case failures <- err:
				case <-workerContext.Done():
				}
			}
		}()
	}

	select {
	case <-ctx.Done():
		cancel()
		workers.Wait()
		return nil
	case err := <-failures:
		cancel()
		workers.Wait()
		return err
	}
}

func (pool *WorkerPool[T]) runRecovery(ctx context.Context) error {
	ticker := time.NewTicker(pool.idlePollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if _, err := pool.queue.RecoverExpired(ctx); err != nil {
				if ctx.Err() != nil {
					return nil
				}
				return fmt.Errorf("recover expired work: %w", err)
			}
		}
	}
}

func (pool *WorkerPool[T]) runWorker(ctx context.Context) error {
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-pool.notifications:
		case <-timer.C:
		}

		for {
			work, found, err := pool.queue.Claim(ctx, pool.leaseDuration)
			if err != nil {
				if ctx.Err() != nil {
					return nil
				}
				return fmt.Errorf("claim work: %w", err)
			}
			if !found {
				timer.Reset(pool.idlePollInterval)
				break
			}
			if err := pool.execute(ctx, work); err != nil {
				if ctx.Err() != nil {
					return nil
				}
				return err
			}
		}
	}
}

func (pool *WorkerPool[T]) execute(ctx context.Context, work T) error {
	workContext, cancel := context.WithCancel(ctx)
	heartbeatDone := make(chan error, 1)
	go func() {
		heartbeatDone <- pool.heartbeat(workContext, cancel, work)
	}()

	// Process records expected terminal outcomes itself. Its returned error is
	// evidence about this item, not a reason to stop the global worker pool.
	_ = pool.processor.Process(workContext, work)
	cancel()
	if err := <-heartbeatDone; err != nil {
		return fmt.Errorf("refresh work lease: %w", err)
	}
	return nil
}

func (pool *WorkerPool[T]) heartbeat(
	ctx context.Context,
	cancel context.CancelFunc,
	work T,
) error {
	ticker := time.NewTicker(pool.heartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			cancelRequested, err := pool.queue.RefreshLease(ctx, work, pool.leaseDuration)
			if err != nil {
				if ctx.Err() != nil {
					return nil
				}
				cancel()
				return err
			}
			if cancelRequested {
				cancel()
				return nil
			}
		}
	}
}
