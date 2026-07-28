//go:build libvips

package imagevips

import (
	"errors"
	"sync"

	"github.com/davidbyttow/govips/v2/vips"
)

type Runtime struct {
	mutex   sync.Mutex
	started bool
}

func NewRuntime() *Runtime {
	return &Runtime{}
}

func (runtime *Runtime) Start() error {
	runtime.mutex.Lock()
	defer runtime.mutex.Unlock()
	if runtime.started {
		return errors.New("libvips runtime is already started")
	}
	// libvips emits an informational line for every load and transformation.
	// Media libraries make that an unbounded operational log stream, so keep
	// routine processing silent while retaining warnings and errors.
	vips.LoggingSettings(nil, vips.LogLevelWarning)
	if err := vips.Startup(&vips.Config{
		ConcurrencyLevel: NativeConcurrency,
		MaxCacheFiles:    NativeCacheFiles,
		MaxCacheMem:      NativeCacheMemory,
		MaxCacheSize:     NativeCacheEntries,
	}); err != nil {
		return err
	}
	runtime.started = true
	return nil
}

func (runtime *Runtime) Shutdown() {
	runtime.mutex.Lock()
	defer runtime.mutex.Unlock()
	if !runtime.started {
		return
	}
	vips.Shutdown()
	runtime.started = false
}
