//go:build linux && watchresource

package files

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

// Run in an isolated privileged Linux environment after temporarily lowering
// fs.inotify.max_user_watches. The runner owns restoring that host setting.
func TestLinuxLibraryWatcherFailsClosedOnKernelWatchENOSPC(t *testing.T) {
	rawLimit, err := os.ReadFile("/proc/sys/fs/inotify/max_user_watches")
	if err != nil {
		t.Fatal(err)
	}
	limit, err := strconv.Atoi(string(bytes.TrimSpace(rawLimit)))
	if err != nil || limit < 1 || limit > 128 {
		t.Fatalf(
			"watch resource fixture requires max_user_watches in [1,128], got %q",
			rawLimit,
		)
	}
	mediaRoot := t.TempDir()
	for index := 0; index <= limit; index++ {
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
	created, err := NewLibraryWatcher(root, WatcherOptions{
		MaxWatches:  128,
		EventBuffer: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	watcher := created.(*linuxLibraryWatcher)
	t.Cleanup(func() { _ = watcher.Close() })
	if err := watcher.WatchLibrary(
		context.Background(),
		1,
		"archive",
	); !errors.Is(err, ErrWatchResourceLimit) {
		t.Fatalf("kernel watch exhaustion error = %v", err)
	}
	if len(watcher.byWD) != 0 || len(watcher.byLibrary) != 0 {
		t.Fatalf(
			"partial registration survived ENOSPC: wd=%d libraries=%d",
			len(watcher.byWD),
			len(watcher.byLibrary),
		)
	}
}
