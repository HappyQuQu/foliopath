//go:build linux

package files

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/scanner"
	"golang.org/x/sys/unix"
)

func TestLinuxLibraryWatcherReportsDirtyDirectories(t *testing.T) {
	mediaRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(mediaRoot, "archive", "albums"), 0o755); err != nil {
		t.Fatal(err)
	}
	root, err := OpenRoot(mediaRoot)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = root.Close() })
	created, err := NewLibraryWatcher(root, WatcherOptions{})
	if err != nil {
		t.Fatal(err)
	}
	watcher := created.(*linuxLibraryWatcher)
	t.Cleanup(func() { _ = watcher.Close() })
	if err := watcher.WatchLibrary(context.Background(), 7, "archive"); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	runDone := make(chan error, 1)
	go func() { runDone <- watcher.Run(ctx) }()

	if err := os.WriteFile(
		filepath.Join(mediaRoot, "archive", "albums", "new.jpg"),
		[]byte("fixture"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	event := awaitWatchEvent(t, watcher.Events())
	if event.LibraryID != 7 ||
		event.RelativeDirectory != "albums" ||
		event.Kind != scanner.WatchEventDirty {
		t.Fatalf("watch event = %#v", event)
	}

	cancel()
	select {
	case err := <-runDone:
		if err != nil {
			t.Fatalf("watcher run: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("watcher did not stop after cancellation")
	}
}

func TestLinuxLibraryWatcherWaitsForRegularFileClose(t *testing.T) {
	mediaRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(mediaRoot, "archive", "albums"), 0o755); err != nil {
		t.Fatal(err)
	}
	root, err := OpenRoot(mediaRoot)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = root.Close() })
	watcher, err := NewLibraryWatcher(root, WatcherOptions{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = watcher.Close() })
	if err := watcher.WatchLibrary(context.Background(), 8, "archive"); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go func() { _ = watcher.Run(ctx) }()

	file, err := os.Create(filepath.Join(mediaRoot, "archive", "albums", "slow.jpg"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte("partial")); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	select {
	case event := <-watcher.Events():
		_ = file.Close()
		t.Fatalf("event before close = %#v", event)
	case <-time.After(150 * time.Millisecond):
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	event := awaitWatchEvent(t, watcher.Events())
	if event.LibraryID != 8 || event.RelativeDirectory != "albums" {
		t.Fatalf("close-write event = %#v", event)
	}
}

func TestLinuxLibraryWatcherTurnsOutputBackpressureIntoOverflow(t *testing.T) {
	root, err := OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = root.Close() })
	created, err := NewLibraryWatcher(root, WatcherOptions{
		MaxWatches:  1,
		EventBuffer: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	watcher := created.(*linuxLibraryWatcher)
	t.Cleanup(func() { _ = watcher.Close() })

	watcher.emit(scanner.WatchEvent{LibraryID: 1, Kind: scanner.WatchEventDirty})
	watcher.emit(scanner.WatchEvent{LibraryID: 1, Kind: scanner.WatchEventDirty})
	first := <-watcher.Events()
	if first.Kind != scanner.WatchEventDirty {
		t.Fatalf("first event = %#v", first)
	}
	watcher.flushOverflow()
	overflow := <-watcher.Events()
	if overflow.Kind != scanner.WatchEventOverflow {
		t.Fatalf("overflow event = %#v", overflow)
	}
}

func TestLinuxLibraryWatcherKeepsHundredThousandEventBurstBounded(t *testing.T) {
	root, err := OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = root.Close() })
	created, err := NewLibraryWatcher(root, WatcherOptions{
		MaxWatches:  1,
		EventBuffer: scanner.MaxPendingWatchEvents,
	})
	if err != nil {
		t.Fatal(err)
	}
	watcher := created.(*linuxLibraryWatcher)
	t.Cleanup(func() { _ = watcher.Close() })

	for index := 0; index < 100_000; index++ {
		watcher.emit(scanner.WatchEvent{
			LibraryID: 1,
			Kind:      scanner.WatchEventDirty,
		})
	}
	if got := len(watcher.events); got != scanner.MaxPendingWatchEvents {
		t.Fatalf("buffered burst events = %d, want %d", got, scanner.MaxPendingWatchEvents)
	}
	if !watcher.overflow.Load() {
		t.Fatal("burst did not record overflow")
	}
	<-watcher.events
	watcher.flushOverflow()
	if got := len(watcher.events); got != scanner.MaxPendingWatchEvents {
		t.Fatalf("buffer after overflow publication = %d", got)
	}
	foundOverflow := false
	for len(watcher.events) > 0 {
		if event := <-watcher.events; event.Kind == scanner.WatchEventOverflow {
			foundOverflow = true
		}
	}
	if !foundOverflow {
		t.Fatal("bounded burst omitted overflow signal")
	}
}

func TestLinuxLibraryWatcherRollsBackPartialRegistrationAtLimit(t *testing.T) {
	mediaRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(mediaRoot, "archive", "child"), 0o755); err != nil {
		t.Fatal(err)
	}
	root, err := OpenRoot(mediaRoot)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = root.Close() })
	created, err := NewLibraryWatcher(root, WatcherOptions{
		MaxWatches:  1,
		EventBuffer: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	watcher := created.(*linuxLibraryWatcher)
	t.Cleanup(func() { _ = watcher.Close() })
	if err := watcher.WatchLibrary(context.Background(), 1, "archive"); !errors.Is(
		err,
		ErrWatchResourceLimit,
	) {
		t.Fatalf("watch over limit error = %v", err)
	}
	if len(watcher.byWD) != 0 || len(watcher.byLibrary) != 0 {
		t.Fatalf(
			"partial watch registration retained: wd=%d libraries=%d",
			len(watcher.byWD),
			len(watcher.byLibrary),
		)
	}
}

func TestLinuxLibraryWatcherSkipsSymlinksWithoutDegrading(t *testing.T) {
	mediaRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(mediaRoot, "archive", "album"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(
		"album",
		filepath.Join(mediaRoot, "archive", "latest"),
	); err != nil {
		t.Fatal(err)
	}
	root, err := OpenRoot(mediaRoot)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = root.Close() })
	watcher, err := NewLibraryWatcher(root, WatcherOptions{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = watcher.Close() })
	if err := watcher.WatchLibrary(context.Background(), 3, "archive"); err != nil {
		t.Fatalf("watch library containing a skipped symlink: %v", err)
	}
}

func TestLinuxLibraryWatcherInvalidatesMovedLibraryRoot(t *testing.T) {
	mediaRoot := t.TempDir()
	if err := os.Mkdir(filepath.Join(mediaRoot, "archive"), 0o755); err != nil {
		t.Fatal(err)
	}
	root, err := OpenRoot(mediaRoot)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = root.Close() })
	watcher, err := NewLibraryWatcher(root, WatcherOptions{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = watcher.Close() })
	if err := watcher.WatchLibrary(context.Background(), 11, "archive"); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go func() { _ = watcher.Run(ctx) }()

	if err := os.Rename(
		filepath.Join(mediaRoot, "archive"),
		filepath.Join(mediaRoot, "archive-moved"),
	); err != nil {
		t.Fatal(err)
	}
	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()
	for {
		select {
		case event := <-watcher.Events():
			if event.Kind == scanner.WatchEventInvalidated {
				if event.LibraryID != 11 || event.RelativeDirectory != "" {
					t.Fatalf("root invalidation event = %#v", event)
				}
				return
			}
		case <-timer.C:
			t.Fatal("timed out waiting for root invalidation")
		}
	}
}

func TestLinuxLibraryWatcherMapsKernelResourceLimits(t *testing.T) {
	for _, input := range []error{unix.ENOSPC, unix.EMFILE, unix.ENFILE} {
		if err := mapInotifyError(input); !errors.Is(err, ErrWatchResourceLimit) {
			t.Errorf("mapInotifyError(%v) = %v", input, err)
		}
	}
}

func TestLinuxLibraryWatcherDoesNotRetainOneFDPerDirectory(t *testing.T) {
	mediaRoot := t.TempDir()
	for index := 0; index < 512; index++ {
		if err := os.MkdirAll(
			filepath.Join(mediaRoot, "archive", "directory-"+strconv.Itoa(index)),
			0o755,
		); err != nil {
			t.Fatal(err)
		}
	}
	root, err := OpenRoot(mediaRoot)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = root.Close() })
	created, err := NewLibraryWatcher(root, WatcherOptions{})
	if err != nil {
		t.Fatal(err)
	}
	watcher := created.(*linuxLibraryWatcher)
	t.Cleanup(func() { _ = watcher.Close() })
	before, err := os.ReadDir("/proc/self/fd")
	if err != nil {
		t.Fatal(err)
	}
	if err := watcher.WatchLibrary(context.Background(), 13, "archive"); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadDir("/proc/self/fd")
	if err != nil {
		t.Fatal(err)
	}
	if len(watcher.byWD) != 513 {
		t.Fatalf("registered watches = %d, want 513", len(watcher.byWD))
	}
	if delta := len(after) - len(before); delta > 4 {
		t.Fatalf("watch registration retained %d additional file descriptors", delta)
	}
}

func awaitWatchEvent(
	t *testing.T,
	events <-chan scanner.WatchEvent,
) scanner.WatchEvent {
	t.Helper()
	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()
	for {
		select {
		case event := <-events:
			if event.Kind == scanner.WatchEventDirty {
				return event
			}
		case <-timer.C:
			t.Fatal("timed out waiting for watch event")
		}
	}
}
