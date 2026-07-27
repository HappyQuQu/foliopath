package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/HappyQuQu/foliopath/internal/library"
)

type libraryCacheCleaner struct {
	dataRoot string
}

func (cleaner libraryCacheCleaner) RemoveLibraryCache(
	ctx context.Context,
	libraryID int64,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	target := filepath.Join(
		cleaner.dataRoot,
		"cache",
		"libraries",
		"lib_"+strconv.FormatInt(libraryID, 10),
	)
	if err := os.RemoveAll(target); err != nil {
		return fmt.Errorf("remove derived library cache: %w", err)
	}
	return nil
}

type removalWorkerComponent struct {
	worker *library.RemovalWorker

	mutex  sync.Mutex
	cancel context.CancelFunc
	done   chan error
}

func newRemovalWorkerComponent(
	worker *library.RemovalWorker,
) (component, error) {
	if worker == nil {
		return component{}, fmt.Errorf("%w: removal worker is required", errInvalidComponent)
	}
	service := &removalWorkerComponent{
		worker: worker,
		done:   make(chan error, 1),
	}
	return component{
		name:  "library-removal-worker",
		start: service.start,
		done:  service.done,
		stop:  service.stop,
	}, nil
}

func (service *removalWorkerComponent) start(context.Context) error {
	service.mutex.Lock()
	defer service.mutex.Unlock()
	if service.cancel != nil {
		return fmt.Errorf("library removal worker is already running")
	}
	ctx, cancel := context.WithCancel(context.Background())
	service.cancel = cancel
	go func() {
		service.done <- service.worker.Run(ctx)
	}()
	return nil
}

func (service *removalWorkerComponent) stop(context.Context) error {
	service.mutex.Lock()
	cancel := service.cancel
	service.cancel = nil
	service.mutex.Unlock()
	if cancel != nil {
		cancel()
	}
	return nil
}
