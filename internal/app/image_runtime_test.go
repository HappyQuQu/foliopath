package app

import (
	"context"
	"errors"
	"testing"
)

type imageRuntimeStub struct {
	startErr      error
	started       int
	shutdownCalls int
}

func (runtime *imageRuntimeStub) Start() error {
	runtime.started++
	return runtime.startErr
}

func (runtime *imageRuntimeStub) Shutdown() {
	runtime.shutdownCalls++
}

func TestImageRuntimeComponentOwnsNativeLifecycle(t *testing.T) {
	runtime := &imageRuntimeStub{}
	component, err := newImageRuntimeComponent(runtime)
	if err != nil {
		t.Fatal(err)
	}
	if err := component.start(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := component.stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	if runtime.started != 1 || runtime.shutdownCalls != 1 {
		t.Fatalf(
			"native lifecycle = start %d shutdown %d",
			runtime.started, runtime.shutdownCalls,
		)
	}
}

func TestImageRuntimeComponentFailsApplicationStartupClosed(t *testing.T) {
	startErr := errors.New("native runtime failed")
	runtime := &imageRuntimeStub{startErr: startErr}
	component, err := newImageRuntimeComponent(runtime)
	if err != nil {
		t.Fatal(err)
	}
	if err := component.start(context.Background()); !errors.Is(err, startErr) {
		t.Fatalf("start error = %v", err)
	}
	if runtime.shutdownCalls != 0 {
		t.Fatal("runtime that failed to start was shut down")
	}
}
