//go:build !linux

package files

func NewLibraryWatcher(*Root, WatcherOptions) (LibraryWatcher, error) {
	return nil, ErrWatchUnsupported
}
