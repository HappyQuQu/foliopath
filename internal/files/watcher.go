package files

import (
	"context"
	"errors"

	"github.com/HappyQuQu/foliopath/internal/scanner"
)

var (
	ErrWatchUnsupported   = errors.New("filesystem watching is unsupported")
	ErrWatchResourceLimit = errors.New("filesystem watch resource limit reached")
	ErrWatchClosed        = errors.New("filesystem watcher is closed")
)

type WatcherOptions struct {
	MaxWatches  int
	EventBuffer int
}

type LibraryWatcher interface {
	WatchLibrary(context.Context, int64, string) error
	WatchDirectory(context.Context, int64, string, string) error
	UnwatchLibrary(int64) error
	Events() <-chan scanner.WatchEvent
	Run(context.Context) error
	Close() error
}
