package app

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestRunReturnsAfterRootCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := run(ctx, Input{}); err != nil {
		t.Fatalf("run() error = %v, want nil", err)
	}
}

func TestApplicationStartsInOrderAndStopsInReverse(t *testing.T) {
	var (
		mutex  sync.Mutex
		events []string
	)
	started := make(chan struct{}, 2)

	record := func(event string) {
		mutex.Lock()
		defer mutex.Unlock()
		events = append(events, event)
	}
	makeComponent := func(name string) component {
		return component{
			name: name,
			start: func(context.Context) error {
				record("start " + name)
				started <- struct{}{}
				return nil
			},
			stop: func(context.Context) error {
				record("stop " + name)
				return nil
			},
		}
	}

	application, err := newApplication(
		[]component{makeComponent("database"), makeComponent("http")},
		time.Second,
	)
	if err != nil {
		t.Fatalf("newApplication() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		result <- application.run(ctx)
	}()

	<-started
	<-started
	cancel()

	if err := <-result; err != nil {
		t.Fatalf("application.run() error = %v, want nil", err)
	}

	mutex.Lock()
	defer mutex.Unlock()
	want := []string{"start database", "start http", "stop http", "stop database"}
	if strings.Join(events, ",") != strings.Join(want, ",") {
		t.Fatalf("lifecycle events = %q, want %q", events, want)
	}
}

func TestApplicationRollsBackStartedComponentsAfterStartupFailure(t *testing.T) {
	var events []string
	startFailure := errors.New("listen failed")
	application, err := newApplication([]component{
		{
			name: "database",
			start: func(context.Context) error {
				events = append(events, "start database")
				return nil
			},
			stop: func(context.Context) error {
				events = append(events, "stop database")
				return nil
			},
		},
		{
			name: "http",
			start: func(context.Context) error {
				events = append(events, "start http")
				return startFailure
			},
			stop: func(context.Context) error {
				t.Fatal("component that failed to start must not be stopped")
				return nil
			},
		},
	}, time.Second)
	if err != nil {
		t.Fatalf("newApplication() error = %v", err)
	}

	runErr := application.run(context.Background())
	if !errors.Is(runErr, startFailure) {
		t.Fatalf("application.run() error = %v, want startup failure", runErr)
	}
	want := []string{"start database", "start http", "stop database"}
	if strings.Join(events, ",") != strings.Join(want, ",") {
		t.Fatalf("lifecycle events = %q, want %q", events, want)
	}
}

func TestApplicationPropagatesRuntimeFailureAndStopsAllComponents(t *testing.T) {
	runtimeFailure := errors.New("serve failed")
	httpDone := make(chan error, 1)
	var events []string

	makeComponent := func(name string, done <-chan error) component {
		return component{
			name: name,
			start: func(context.Context) error {
				events = append(events, "start "+name)
				return nil
			},
			done: done,
			stop: func(context.Context) error {
				events = append(events, "stop "+name)
				return nil
			},
		}
	}
	application, err := newApplication([]component{
		makeComponent("database", nil),
		makeComponent("http", httpDone),
	}, time.Second)
	if err != nil {
		t.Fatalf("newApplication() error = %v", err)
	}

	httpDone <- runtimeFailure
	runErr := application.run(context.Background())
	if !errors.Is(runErr, runtimeFailure) {
		t.Fatalf("application.run() error = %v, want runtime failure", runErr)
	}
	if !strings.Contains(runErr.Error(), `component "http" stopped`) {
		t.Fatalf("application.run() error = %q, want component name", runErr)
	}
	want := []string{"start database", "start http", "stop http", "stop database"}
	if strings.Join(events, ",") != strings.Join(want, ",") {
		t.Fatalf("lifecycle events = %q, want %q", events, want)
	}
}

func TestApplicationJoinsShutdownErrors(t *testing.T) {
	stopFailure := errors.New("close failed")
	started := make(chan struct{})
	application, err := newApplication([]component{{
		name: "database",
		start: func(context.Context) error {
			close(started)
			return nil
		},
		stop: func(context.Context) error {
			return stopFailure
		},
	}}, time.Second)
	if err != nil {
		t.Fatalf("newApplication() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		result <- application.run(ctx)
	}()
	<-started
	cancel()

	if runErr := <-result; !errors.Is(runErr, stopFailure) {
		t.Fatalf("application.run() error = %v, want shutdown failure", runErr)
	}
}

func TestApplicationBoundsShutdown(t *testing.T) {
	started := make(chan struct{})
	application, err := newApplication([]component{{
		name: "stuck",
		start: func(context.Context) error {
			close(started)
			return nil
		},
		stop: func(shutdownContext context.Context) error {
			<-shutdownContext.Done()
			return shutdownContext.Err()
		},
	}}, 20*time.Millisecond)
	if err != nil {
		t.Fatalf("newApplication() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		result <- application.run(ctx)
	}()
	<-started
	cancel()

	select {
	case runErr := <-result:
		if !errors.Is(runErr, context.DeadlineExceeded) {
			t.Fatalf("application.run() error = %v, want deadline exceeded", runErr)
		}
	case <-time.After(time.Second):
		t.Fatal("application.run() did not respect shutdown timeout")
	}
}

func TestNewApplicationRejectsInvalidLifecycle(t *testing.T) {
	tests := []struct {
		name       string
		components []component
		timeout    time.Duration
	}{
		{name: "non-positive timeout", timeout: 0},
		{name: "missing name", components: []component{{start: noOp, stop: noOp}}, timeout: time.Second},
		{name: "missing start", components: []component{{name: "http", stop: noOp}}, timeout: time.Second},
		{name: "missing stop", components: []component{{name: "http", start: noOp}}, timeout: time.Second},
		{
			name: "duplicate name",
			components: []component{
				{name: "http", start: noOp, stop: noOp},
				{name: "http", start: noOp, stop: noOp},
			},
			timeout: time.Second,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := newApplication(test.components, test.timeout); !errors.Is(err, errInvalidComponent) {
				t.Fatalf("newApplication() error = %v, want errInvalidComponent", err)
			}
		})
	}
}

func noOp(context.Context) error {
	return nil
}
