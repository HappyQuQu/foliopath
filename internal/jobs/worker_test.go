package jobs

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

type workerQueueStub struct {
	mutex         sync.Mutex
	work          []int
	recoverCalled bool
	recoverCalls  int
	recoverErrAt  int
	refreshCancel bool
	refreshes     int
}

func (stub *workerQueueStub) RecoverExpired(context.Context) (RecoverySummary, error) {
	stub.mutex.Lock()
	defer stub.mutex.Unlock()
	stub.recoverCalled = true
	stub.recoverCalls++
	if stub.recoverErrAt > 0 && stub.recoverCalls >= stub.recoverErrAt {
		return RecoverySummary{}, errors.New("injected recovery failure")
	}
	return RecoverySummary{}, nil
}

func (stub *workerQueueStub) Claim(
	context.Context,
	time.Duration,
) (int, bool, error) {
	stub.mutex.Lock()
	defer stub.mutex.Unlock()
	if !stub.recoverCalled {
		return 0, false, errors.New("claim happened before recovery")
	}
	if len(stub.work) == 0 {
		return 0, false, nil
	}
	work := stub.work[0]
	stub.work = stub.work[1:]
	return work, true, nil
}

func (stub *workerQueueStub) RefreshLease(
	context.Context,
	int,
	time.Duration,
) (bool, error) {
	stub.mutex.Lock()
	defer stub.mutex.Unlock()
	stub.refreshes++
	return stub.refreshCancel, nil
}

type blockingProcessor struct {
	mutex       sync.Mutex
	active      int
	maxActive   int
	completed   int
	started     chan struct{}
	release     <-chan struct{}
	allComplete chan struct{}
	total       int
}

func (processor *blockingProcessor) Process(ctx context.Context, _ int) error {
	processor.mutex.Lock()
	processor.active++
	if processor.active > processor.maxActive {
		processor.maxActive = processor.active
	}
	processor.mutex.Unlock()
	processor.started <- struct{}{}

	select {
	case <-processor.release:
	case <-ctx.Done():
		return ctx.Err()
	}

	processor.mutex.Lock()
	processor.active--
	processor.completed++
	if processor.completed == processor.total {
		close(processor.allComplete)
	}
	processor.mutex.Unlock()
	return nil
}

type testWakeSource struct {
	notifications chan struct{}
}

func (source testWakeSource) Notifications() <-chan struct{} {
	return source.notifications
}

func TestWorkerPoolRecoversThenProcessesWithGlobalBound(t *testing.T) {
	queue := &workerQueueStub{work: []int{1, 2, 3}}
	release := make(chan struct{})
	processor := &blockingProcessor{
		started:     make(chan struct{}, 3),
		release:     release,
		allComplete: make(chan struct{}),
		total:       3,
	}
	pool, err := NewWorkerPool(
		queue,
		processor,
		testWakeSource{notifications: make(chan struct{}, 1)},
		WorkerOptions{
			Workers:           2,
			HeartbeatInterval: 20 * time.Millisecond,
			LeaseDuration:     100 * time.Millisecond,
			IdlePollInterval:  5 * time.Millisecond,
		},
	)
	if err != nil {
		t.Fatalf("NewWorkerPool() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() { result <- pool.Run(ctx) }()

	for index := 0; index < 2; index++ {
		select {
		case <-processor.started:
		case <-time.After(time.Second):
			t.Fatal("two workers did not start")
		}
	}
	select {
	case <-processor.started:
		t.Fatal("third item started before worker capacity was released")
	case <-time.After(25 * time.Millisecond):
	}
	close(release)
	select {
	case <-processor.allComplete:
	case <-time.After(time.Second):
		t.Fatal("queued work did not complete")
	}

	cancel()
	if err := <-result; err != nil {
		t.Fatalf("WorkerPool.Run() error = %v", err)
	}
	processor.mutex.Lock()
	defer processor.mutex.Unlock()
	if processor.maxActive != 2 {
		t.Fatalf("maximum concurrent work = %d, want 2", processor.maxActive)
	}
}

type cancellationProcessor struct {
	cancelled chan struct{}
}

func (processor cancellationProcessor) Process(ctx context.Context, _ int) error {
	<-ctx.Done()
	close(processor.cancelled)
	return ctx.Err()
}

func TestWorkerPoolHeartbeatPropagatesDurableCancellation(t *testing.T) {
	queue := &workerQueueStub{work: []int{9}, refreshCancel: true}
	processor := cancellationProcessor{cancelled: make(chan struct{})}
	pool, err := NewWorkerPool(
		queue,
		processor,
		testWakeSource{notifications: make(chan struct{}, 1)},
		WorkerOptions{
			Workers:           1,
			HeartbeatInterval: 5 * time.Millisecond,
			LeaseDuration:     50 * time.Millisecond,
			IdlePollInterval:  5 * time.Millisecond,
		},
	)
	if err != nil {
		t.Fatalf("NewWorkerPool() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() { result <- pool.Run(ctx) }()
	select {
	case <-processor.cancelled:
	case <-time.After(time.Second):
		t.Fatal("durable cancellation did not reach processor")
	}
	cancel()
	if err := <-result; err != nil {
		t.Fatalf("WorkerPool.Run() error = %v", err)
	}
	queue.mutex.Lock()
	defer queue.mutex.Unlock()
	if queue.refreshes == 0 {
		t.Fatal("work lease was never refreshed")
	}
}

func TestWorkerPoolContinuesRecoveringExpiredWork(t *testing.T) {
	queue := &workerQueueStub{}
	pool, err := NewWorkerPool(
		queue,
		&blockingProcessor{},
		testWakeSource{notifications: make(chan struct{}, 1)},
		WorkerOptions{
			Workers:           1,
			HeartbeatInterval: 20 * time.Millisecond,
			LeaseDuration:     100 * time.Millisecond,
			IdlePollInterval:  5 * time.Millisecond,
		},
	)
	if err != nil {
		t.Fatalf("NewWorkerPool() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() { result <- pool.Run(ctx) }()
	deadline := time.After(time.Second)
	for {
		queue.mutex.Lock()
		recoverCalls := queue.recoverCalls
		queue.mutex.Unlock()
		if recoverCalls >= 2 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("periodic expired-work recovery did not run")
		case <-time.After(time.Millisecond):
		}
	}
	cancel()
	if err := <-result; err != nil {
		t.Fatalf("WorkerPool.Run() error = %v", err)
	}
}

func TestWorkerPoolStopsWhenPeriodicRecoveryFails(t *testing.T) {
	queue := &workerQueueStub{recoverErrAt: 2}
	pool, err := NewWorkerPool(
		queue,
		&blockingProcessor{},
		testWakeSource{notifications: make(chan struct{}, 1)},
		WorkerOptions{
			Workers:           1,
			HeartbeatInterval: 20 * time.Millisecond,
			LeaseDuration:     100 * time.Millisecond,
			IdlePollInterval:  time.Millisecond,
		},
	)
	if err != nil {
		t.Fatalf("NewWorkerPool() error = %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := pool.Run(ctx); err == nil ||
		!strings.Contains(err.Error(), "recover expired work") {
		t.Fatalf("WorkerPool.Run() error = %v", err)
	}
}

func TestNewWorkerPoolRejectsInvalidCapacityAndTiming(t *testing.T) {
	queue := &workerQueueStub{}
	processor := cancellationProcessor{cancelled: make(chan struct{})}
	wake := testWakeSource{notifications: make(chan struct{})}
	for name, options := range map[string]WorkerOptions{
		"too many workers": {Workers: DefaultWorkerCount + 1},
		"lease too short": {
			Workers:           1,
			HeartbeatInterval: time.Second,
			LeaseDuration:     time.Second,
			IdlePollInterval:  time.Second,
		},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := NewWorkerPool(queue, processor, wake, options); err == nil {
				t.Fatal("NewWorkerPool() succeeded with invalid options")
			}
		})
	}
}
