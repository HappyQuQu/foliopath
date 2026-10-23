package jobs

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

type handoffQueueStub struct {
	*workerQueueStub
	called chan struct{}
	reaped <-chan struct{}
	err    error
}

func (q *handoffQueueStub) Handoff(ctx context.Context, _ int) error {
	if ctx.Err() != nil {
		return errors.New("handoff inherited cancelled context")
	}
	select {
	case <-q.reaped:
	default:
		return errors.New("processor not reaped")
	}
	close(q.called)
	return q.err
}

type handoffProcessorStub struct {
	started chan struct{}
	reaped  chan struct{}
}

func (p *handoffProcessorStub) Process(ctx context.Context, _ int) error {
	close(p.started)
	<-ctx.Done()
	close(p.reaped)
	return ctx.Err()
}

func TestWorkerPlannedHandoffAfterReapAndReportsFailure(t *testing.T) {
	for _, planned := range []bool{false, true} {
		for _, fail := range []bool{false, true} {
			t.Run(fmt.Sprintf("planned=%v/failure=%v", planned, fail), func(t *testing.T) {
				p := &handoffProcessorStub{started: make(chan struct{}), reaped: make(chan struct{})}
				q := &handoffQueueStub{workerQueueStub: &workerQueueStub{work: []int{7}}, called: make(chan struct{}), reaped: p.reaped}
				injected := errors.New("handoff write failed")
				if fail {
					q.err = injected
				}
				pool, err := NewWorkerPool[int](q, p, NewSignal(), WorkerOptions{Workers: 1})
				if err != nil {
					t.Fatal(err)
				}
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				done := make(chan error, 1)
				go func() { done <- pool.Run(ctx) }()
				select {
				case <-p.started:
				case <-time.After(time.Second):
					t.Fatal("not started")
				}
				if planned {
					cancel(ErrPlannedShutdown)
				} else {
					cancel(errors.New("component failure"))
				}
				select {
				case err = <-done:
				case <-time.After(time.Second):
					t.Fatal("not stopped")
				}
				if errors.Is(err, injected) != (planned && fail) {
					t.Fatalf("failure propagation planned=%v fail=%v err=%v", planned, fail, err)
				}
				select {
				case <-q.called:
					if !planned {
						t.Fatal("crash got planned handoff")
					}
				default:
					if planned {
						t.Fatal("missing handoff")
					}
				}
			})
		}
	}
}
