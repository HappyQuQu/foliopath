package app

import (
	"context"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/api"
)

func TestReadinessStartsUnavailableAndTransitionsToShutdown(t *testing.T) {
	state := newReadinessState()
	if got := state.snapshot(); got.Ready || got.ReasonCode != api.ReadinessDatabaseUnavailable {
		t.Fatalf("initial readiness = %#v, want database unavailable", got)
	}

	component := readinessLifecycle(state)
	if err := component.start(context.Background()); err != nil {
		t.Fatalf("start readiness lifecycle: %v", err)
	}
	if got := state.snapshot(); !got.Ready || got.ReasonCode != "" {
		t.Fatalf("ready state = %#v, want ready", got)
	}

	if err := component.stop(context.Background()); err != nil {
		t.Fatalf("stop readiness lifecycle: %v", err)
	}
	if got := state.snapshot(); got.Ready || got.ReasonCode != api.ReadinessShuttingDown {
		t.Fatalf("shutdown readiness = %#v, want shutting down", got)
	}
}
