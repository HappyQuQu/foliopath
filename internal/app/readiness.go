package app

import (
	"context"
	"sync"

	"github.com/HappyQuQu/foliopath/internal/api"
)

type readinessState struct {
	mutex     sync.RWMutex
	readiness api.Readiness
}

func newReadinessState() *readinessState {
	return &readinessState{
		readiness: api.Readiness{
			Ready:      false,
			ReasonCode: api.ReadinessDatabaseUnavailable,
		},
	}
}

func (state *readinessState) snapshot() api.Readiness {
	state.mutex.RLock()
	defer state.mutex.RUnlock()
	return state.readiness
}

func (state *readinessState) set(readiness api.Readiness) {
	state.mutex.Lock()
	defer state.mutex.Unlock()
	state.readiness = readiness
}

func readinessLifecycle(state *readinessState) component {
	return component{
		name: "readiness",
		start: func(context.Context) error {
			state.set(api.Readiness{Ready: true})
			return nil
		},
		stop: func(context.Context) error {
			state.set(api.Readiness{
				Ready:      false,
				ReasonCode: api.ReadinessShuttingDown,
			})
			return nil
		},
	}
}
