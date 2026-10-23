package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/jobs"
)

func TestAIWorkerStopPreservesPlannedCauseAndHandoffFailure(t *testing.T) {
	parent, stop := context.WithCancelCause(context.Background())
	worker, cancel := context.WithCancelCause(context.Background())
	service := &aiWorkerComponent{parent: parent, cancel: cancel, stopped: make(chan struct{})}
	failure := errors.New("injected handoff failure")
	observed := make(chan error, 1)
	go func() {
		<-worker.Done()
		observed <- context.Cause(worker)
		service.mu.Lock()
		service.runErr = jobs.WorkerFailure{Stage: "handoff", Err: failure}
		service.mu.Unlock()
		close(service.stopped)
	}()
	stop(jobs.ErrPlannedShutdown)
	ctx, finish := context.WithTimeout(context.Background(), time.Second)
	defer finish()
	if err := service.stop(ctx); !errors.Is(err, failure) {
		t.Fatal("handoff failure swallowed", err)
	}
	if cause := <-observed; !errors.Is(cause, jobs.ErrPlannedShutdown) {
		t.Fatal("signal cause lost", cause)
	}
	// Stopping an unstarted component is harmless.
	if err := (&aiWorkerComponent{}).stop(ctx); err != nil {
		t.Fatal(err)
	}
}
