//go:build linux

package files

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strconv"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/HappyQuQu/foliopath/internal/scanner"
	"golang.org/x/sys/unix"
)

const watchMask = unix.IN_ATTRIB |
	unix.IN_CLOSE_WRITE |
	unix.IN_CREATE |
	unix.IN_DELETE |
	unix.IN_DELETE_SELF |
	unix.IN_MOVE_SELF |
	unix.IN_MOVED_FROM |
	unix.IN_MOVED_TO |
	unix.IN_UNMOUNT |
	unix.IN_ONLYDIR |
	unix.IN_EXCL_UNLINK

type watchRegistration struct {
	libraryID         int64
	relativeDirectory string
}

type linuxLibraryWatcher struct {
	root       *Root
	fd         int
	maxWatches int
	events     chan scanner.WatchEvent

	mu        sync.Mutex
	closed    bool
	byWD      map[int]watchRegistration
	byLibrary map[int64]map[int]struct{}
	overflow  atomic.Bool
}

func NewLibraryWatcher(root *Root, options WatcherOptions) (LibraryWatcher, error) {
	if root == nil {
		return nil, errors.New("files watcher requires a root")
	}
	if options.MaxWatches == 0 {
		options.MaxWatches = scanner.MaxDirectoryWatches
	}
	if options.EventBuffer == 0 {
		options.EventBuffer = scanner.MaxPendingWatchEvents
	}
	if options.MaxWatches < 1 || options.MaxWatches > scanner.MaxDirectoryWatches ||
		options.EventBuffer < 1 || options.EventBuffer > scanner.MaxPendingWatchEvents {
		return nil, errors.New("files watcher options exceed scanner resource limits")
	}
	fd, err := unix.InotifyInit1(unix.IN_CLOEXEC | unix.IN_NONBLOCK)
	if err != nil {
		return nil, mapInotifyError(err)
	}
	return &linuxLibraryWatcher{
		root:       root,
		fd:         fd,
		maxWatches: options.MaxWatches,
		events:     make(chan scanner.WatchEvent, options.EventBuffer),
		byWD:       make(map[int]watchRegistration),
		byLibrary:  make(map[int64]map[int]struct{}),
	}, nil
}

func (watcher *linuxLibraryWatcher) Events() <-chan scanner.WatchEvent {
	return watcher.events
}

func (watcher *linuxLibraryWatcher) WatchLibrary(
	ctx context.Context,
	libraryID int64,
	rootRelative string,
) error {
	if ctx == nil || libraryID <= 0 {
		return fs.ErrInvalid
	}
	normalized, err := Normalize(rootRelative)
	if err != nil || normalized != rootRelative {
		return fs.ErrInvalid
	}
	identity, err := watcher.root.CaptureAt(rootRelative)
	if err != nil {
		return err
	}
	added := make([]int, 0, 64)
	err = watcher.root.walkCaptured(
		ctx,
		rootRelative,
		identity,
		func(relative string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				if isPolicySkip(walkErr) {
					return nil
				}
				return walkErr
			}
			if !entry.IsDir() {
				return nil
			}
			libraryRelative, err := relativeToLibrary(rootRelative, relative)
			if err != nil {
				return err
			}
			if libraryRelative != "" && scanner.IsSystemDirectory(path.Base(libraryRelative)) {
				return fs.SkipDir
			}
			wd, newlyAdded, err := watcher.addDirectory(
				libraryID,
				libraryRelative,
				relative,
			)
			if err != nil {
				return err
			}
			if newlyAdded {
				added = append(added, wd)
			}
			return nil
		},
	)
	if err == nil {
		err = watcher.root.VerifyAt(rootRelative, identity)
	}
	if err != nil {
		watcher.removeWatchDescriptors(added)
		return err
	}
	return nil
}

func (watcher *linuxLibraryWatcher) WatchDirectory(
	ctx context.Context,
	libraryID int64,
	rootRelative string,
	relativeDirectory string,
) error {
	if ctx == nil || libraryID <= 0 {
		return fs.ErrInvalid
	}
	rootNormalized, err := Normalize(rootRelative)
	if err != nil || rootNormalized != rootRelative {
		return fs.ErrInvalid
	}
	directoryNormalized, err := Normalize(relativeDirectory)
	if err != nil || directoryNormalized != relativeDirectory {
		return fs.ErrInvalid
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	allowedRelative := rootRelative
	if relativeDirectory != "" {
		allowedRelative = path.Join(rootRelative, relativeDirectory)
	}
	_, _, err = watcher.addDirectory(libraryID, relativeDirectory, allowedRelative)
	return err
}

func (watcher *linuxLibraryWatcher) addDirectory(
	libraryID int64,
	libraryRelative string,
	allowedRelative string,
) (int, bool, error) {
	directory, err := watcher.root.OpenDir(allowedRelative)
	if err != nil {
		return 0, false, err
	}

	watcher.mu.Lock()
	defer watcher.mu.Unlock()
	if watcher.closed {
		_ = directory.Close()
		return 0, false, ErrWatchClosed
	}
	if len(watcher.byWD) >= watcher.maxWatches {
		_ = directory.Close()
		return 0, false, ErrWatchResourceLimit
	}
	procPath := "/proc/self/fd/" + strconv.FormatUint(uint64(directory.file.Fd()), 10)
	wd, err := unix.InotifyAddWatch(watcher.fd, procPath, watchMask)
	if err != nil {
		_ = directory.Close()
		return 0, false, mapInotifyError(err)
	}
	if err := directory.Close(); err != nil {
		_, _ = unix.InotifyRmWatch(watcher.fd, uint32(wd))
		return 0, false, err
	}
	if existing, ok := watcher.byWD[wd]; ok {
		if existing.libraryID == libraryID &&
			existing.relativeDirectory == libraryRelative {
			return wd, false, nil
		}
		return 0, false, errors.New("inotify watch descriptor collision")
	}
	watcher.byWD[wd] = watchRegistration{
		libraryID:         libraryID,
		relativeDirectory: libraryRelative,
	}
	descriptors := watcher.byLibrary[libraryID]
	if descriptors == nil {
		descriptors = make(map[int]struct{})
		watcher.byLibrary[libraryID] = descriptors
	}
	descriptors[wd] = struct{}{}
	return wd, true, nil
}

func (watcher *linuxLibraryWatcher) UnwatchLibrary(libraryID int64) error {
	if libraryID <= 0 {
		return fs.ErrInvalid
	}
	watcher.mu.Lock()
	descriptors := watcher.byLibrary[libraryID]
	delete(watcher.byLibrary, libraryID)
	var result error
	for wd := range descriptors {
		_, exists := watcher.byWD[wd]
		delete(watcher.byWD, wd)
		if exists {
			if _, err := unix.InotifyRmWatch(watcher.fd, uint32(wd)); err != nil &&
				!errors.Is(err, unix.EINVAL) {
				result = errors.Join(result, err)
			}
		}
	}
	watcher.mu.Unlock()
	return result
}

func (watcher *linuxLibraryWatcher) removeWatchDescriptors(descriptors []int) {
	watcher.mu.Lock()
	defer watcher.mu.Unlock()
	for _, wd := range descriptors {
		registration, exists := watcher.byWD[wd]
		if !exists {
			continue
		}
		delete(watcher.byWD, wd)
		if library := watcher.byLibrary[registration.libraryID]; library != nil {
			delete(library, wd)
			if len(library) == 0 {
				delete(watcher.byLibrary, registration.libraryID)
			}
		}
		_, _ = unix.InotifyRmWatch(watcher.fd, uint32(wd))
	}
}

func (watcher *linuxLibraryWatcher) Run(ctx context.Context) error {
	if ctx == nil {
		return fs.ErrInvalid
	}
	buffer := make([]byte, 64*1024)
	poll := []unix.PollFd{{Fd: int32(watcher.fd), Events: unix.POLLIN}}
	for {
		if err := ctx.Err(); err != nil {
			return nil
		}
		watcher.flushOverflow()
		count, err := unix.Poll(poll, 100)
		if err != nil {
			if errors.Is(err, unix.EINTR) {
				continue
			}
			if watcher.isClosed() {
				return nil
			}
			return fmt.Errorf("poll inotify: %w", err)
		}
		if count == 0 {
			continue
		}
		read, err := unix.Read(watcher.fd, buffer)
		if err != nil {
			if errors.Is(err, unix.EAGAIN) || errors.Is(err, unix.EINTR) {
				continue
			}
			if watcher.isClosed() || errors.Is(err, unix.EBADF) {
				return nil
			}
			return fmt.Errorf("read inotify: %w", err)
		}
		for offset := 0; offset+unix.SizeofInotifyEvent <= read; {
			raw := (*unix.InotifyEvent)(unsafe.Pointer(&buffer[offset]))
			size := unix.SizeofInotifyEvent + int(raw.Len)
			if size < unix.SizeofInotifyEvent || offset+size > read {
				return errors.New("invalid inotify event framing")
			}
			watcher.consume(int(raw.Wd), raw.Mask)
			offset += size
		}
	}
}

func (watcher *linuxLibraryWatcher) consume(wd int, mask uint32) {
	if mask&unix.IN_Q_OVERFLOW != 0 {
		watcher.overflow.Store(true)
		return
	}
	// A regular-file create is not a stable media boundary: writers may keep
	// the file open for an arbitrarily long copy. Wait for IN_CLOSE_WRITE.
	// Directory creates must still dirty the parent so the new subtree can be
	// indexed and watched.
	if mask&unix.IN_CREATE != 0 &&
		mask&unix.IN_ISDIR == 0 &&
		mask&^uint32(unix.IN_CREATE|unix.IN_ISDIR) == 0 {
		return
	}
	watcher.mu.Lock()
	registration, ok := watcher.byWD[wd]
	if ok && mask&unix.IN_IGNORED != 0 {
		delete(watcher.byWD, wd)
		if library := watcher.byLibrary[registration.libraryID]; library != nil {
			delete(library, wd)
			if len(library) == 0 {
				delete(watcher.byLibrary, registration.libraryID)
			}
		}
	}
	watcher.mu.Unlock()
	if !ok {
		return
	}
	kind := scanner.WatchEventDirty
	relativeDirectory := registration.relativeDirectory
	if mask&(unix.IN_DELETE_SELF|unix.IN_MOVE_SELF|unix.IN_UNMOUNT|unix.IN_IGNORED) != 0 {
		kind = scanner.WatchEventInvalidated
		relativeDirectory = path.Dir(relativeDirectory)
		if relativeDirectory == "." {
			relativeDirectory = ""
		}
	}
	watcher.emit(scanner.WatchEvent{
		LibraryID:         registration.libraryID,
		RelativeDirectory: relativeDirectory,
		Kind:              kind,
	})
}

func (watcher *linuxLibraryWatcher) emit(event scanner.WatchEvent) {
	select {
	case watcher.events <- event:
	default:
		watcher.overflow.Store(true)
	}
}

func (watcher *linuxLibraryWatcher) flushOverflow() {
	if !watcher.overflow.Load() {
		return
	}
	select {
	case watcher.events <- scanner.WatchEvent{Kind: scanner.WatchEventOverflow}:
		watcher.overflow.Store(false)
	default:
	}
}

func (watcher *linuxLibraryWatcher) isClosed() bool {
	watcher.mu.Lock()
	defer watcher.mu.Unlock()
	return watcher.closed
}

func (watcher *linuxLibraryWatcher) Close() error {
	watcher.mu.Lock()
	if watcher.closed {
		watcher.mu.Unlock()
		return ErrWatchClosed
	}
	watcher.closed = true
	var result error
	for wd := range watcher.byWD {
		_, _ = unix.InotifyRmWatch(watcher.fd, uint32(wd))
	}
	watcher.byWD = nil
	watcher.byLibrary = nil
	result = errors.Join(result, unix.Close(watcher.fd))
	watcher.mu.Unlock()
	return result
}

func mapInotifyError(err error) error {
	switch {
	case errors.Is(err, unix.ENOSYS), errors.Is(err, unix.ENOTSUP):
		return ErrWatchUnsupported
	case errors.Is(err, unix.ENOSPC), errors.Is(err, unix.EMFILE), errors.Is(err, unix.ENFILE):
		return ErrWatchResourceLimit
	default:
		return err
	}
}
