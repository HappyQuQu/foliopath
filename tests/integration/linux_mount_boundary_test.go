//go:build linux && fsboundary

package integration_test

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/files"
	"github.com/HappyQuQu/foliopath/internal/library"
)

// These acceptance probes require CAP_SYS_ADMIN in an isolated Linux
// container or mount namespace. They are intentionally outside the default
// test set: run them explicitly with `go test -tags fsboundary`.

func TestFS01RejectsCrossDeviceBindMount(t *testing.T) {
	base := t.TempDir()
	allowedPath := filepath.Join(base, "library")
	mountTarget := filepath.Join(allowedPath, "mounted")
	sourcePath := filepath.Join("/dev/shm", "foliopath-fs01-"+filepath.Base(base))
	prepareMountFixture(t, allowedPath, mountTarget, sourcePath)

	assertDifferentDevice(t, allowedPath, sourcePath)
	assertBindMountRejected(t, allowedPath, sourcePath, mountTarget)
}

func TestFS01RejectsSameDeviceBindMount(t *testing.T) {
	base := t.TempDir()
	allowedPath := filepath.Join(base, "library")
	mountTarget := filepath.Join(allowedPath, "mounted")
	sourcePath := filepath.Join(base, "outside")
	prepareMountFixture(t, allowedPath, mountTarget, sourcePath)

	assertSameDevice(t, allowedPath, sourcePath)
	assertBindMountRejected(t, allowedPath, sourcePath, mountTarget)
}

func TestFS01RejectsSelfBindMount(t *testing.T) {
	base := t.TempDir()
	allowedPath := filepath.Join(base, "library")
	mountTarget := filepath.Join(allowedPath, "mounted")
	if err := os.MkdirAll(mountTarget, 0o755); err != nil {
		t.Fatalf("create self-bind target: %v", err)
	}

	root, err := files.OpenRoot(allowedPath)
	if err != nil {
		t.Fatalf("open allowed media root: %v", err)
	}
	defer root.Close()
	captured, err := root.CaptureAt("mounted")
	if err != nil {
		t.Fatalf("capture self-bind target: %v", err)
	}

	// Mounting the directory onto itself preserves both device and inode. An
	// identity-only implementation would accept VerifyAt; RESOLVE_NO_XDEV must
	// reject the mount transition itself.
	mountBind(t, mountTarget, mountTarget)
	if _, err := root.CaptureAt("mounted"); !errors.Is(err, files.ErrCrossDevice) {
		t.Fatalf("Root.CaptureAt(self-bind) error = %v, want ErrCrossDevice", err)
	}
	verifyErr := root.VerifyAt("mounted", captured)
	if !errors.Is(verifyErr, files.ErrOffline) || !errors.Is(verifyErr, files.ErrRootChanged) {
		t.Fatalf("Root.VerifyAt(self-bind) error = %v, want ErrOffline and ErrRootChanged", verifyErr)
	}
}

func prepareMountFixture(t *testing.T, allowedPath, mountTarget, sourcePath string) {
	t.Helper()
	for _, directory := range []string{allowedPath, mountTarget, sourcePath} {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Fatalf("create mount fixture %q: %v", directory, err)
		}
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(sourcePath); err != nil {
			t.Errorf("remove mount source %q: %v", sourcePath, err)
		}
	})
	if err := os.WriteFile(filepath.Join(sourcePath, "secret.jpg"), []byte("mounted-secret"), 0o644); err != nil {
		t.Fatalf("write mounted fixture: %v", err)
	}
}

func mountBind(t *testing.T, sourcePath, mountTarget string) {
	t.Helper()
	if os.Geteuid() != 0 {
		t.Fatal("bind-mount acceptance probe requires root plus CAP_SYS_ADMIN")
	}
	if _, err := exec.LookPath("mount"); err != nil {
		t.Fatalf("mount executable unavailable: %v", err)
	}
	command := exec.Command("mount", "--bind", sourcePath, mountTarget)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("bind mount unavailable (run in an isolated container with CAP_SYS_ADMIN): %v: %s",
			err, output)
	}
	t.Cleanup(func() {
		command := exec.Command("umount", mountTarget)
		if output, err := command.CombinedOutput(); err != nil {
			t.Errorf("unmount %q: %v: %s", mountTarget, err, output)
		}
	})
}

func assertBindMountRejected(t *testing.T, allowedPath, sourcePath, mountTarget string) {
	t.Helper()
	root, err := files.OpenRoot(allowedPath)
	if err != nil {
		t.Fatalf("open allowed media root: %v", err)
	}
	defer root.Close()

	captured, err := root.CaptureAt("mounted")
	if err != nil {
		t.Fatalf("capture mount target before bind: %v", err)
	}
	mountBind(t, sourcePath, mountTarget)

	file, err := root.Open("mounted/secret.jpg")
	if file != nil {
		_ = file.Close()
	}
	if err == nil {
		t.Fatal("Root.Open succeeded; mounted content crossed the media boundary")
	}
	if !errors.Is(err, files.ErrCrossDevice) {
		t.Fatalf("Root.Open error = %v, want ErrCrossDevice", err)
	}

	directory, err := root.OpenDir("mounted")
	if directory != nil {
		_ = directory.Close()
	}
	if !errors.Is(err, files.ErrCrossDevice) {
		t.Fatalf("Root.OpenDir error = %v, want ErrCrossDevice", err)
	}

	if _, err := root.CaptureAt("mounted"); !errors.Is(err, files.ErrCrossDevice) {
		t.Fatalf("Root.CaptureAt error = %v, want ErrCrossDevice", err)
	}

	verifyErr := root.VerifyAt("mounted", captured)
	if !errors.Is(verifyErr, files.ErrOffline) || !errors.Is(verifyErr, files.ErrRootChanged) {
		t.Fatalf("Root.VerifyAt error = %v, want ErrOffline and ErrRootChanged", verifyErr)
	}

	if err := root.Walk(context.Background(), "mounted", func(string, fs.DirEntry, error) error {
		t.Fatal("Walk callback ran inside a bind mount")
		return nil
	}); !errors.Is(err, files.ErrCrossDevice) {
		t.Fatalf("Root.Walk(mounted) error = %v, want ErrCrossDevice", err)
	}

	directorySource, err := files.NewDirectorySource(root)
	if err != nil {
		t.Fatalf("NewDirectorySource: %v", err)
	}
	if err := directorySource.EnumerateDirectories(
		context.Background(),
		"mounted",
		func(library.DirectoryCandidate) error {
			t.Fatal("directory enumeration entered a bind mount")
			return nil
		},
	); !errors.Is(err, library.ErrParentMountBoundary) {
		t.Fatalf(
			"EnumerateDirectories(mounted) error = %v, want ErrParentMountBoundary",
			err,
		)
	}
	var mountedCandidate library.DirectoryCandidate
	if err := directorySource.EnumerateDirectories(
		context.Background(),
		"",
		func(candidate library.DirectoryCandidate) error {
			if candidate.Name == "mounted" {
				mountedCandidate = candidate
			}
			return nil
		},
	); err != nil {
		t.Fatalf("EnumerateDirectories(root): %v", err)
	}
	if mountedCandidate.BlockedReason != library.SelectionBlockedMountBoundary {
		t.Fatalf("mounted directory candidate = %#v, want mount boundary", mountedCandidate)
	}

	var reportedMountError error
	sawMountedContent := false
	if err := root.Walk(context.Background(), "", func(
		relative string,
		_ fs.DirEntry,
		walkErr error,
	) error {
		switch relative {
		case "mounted":
			reportedMountError = walkErr
		case "mounted/secret.jpg":
			sawMountedContent = true
		}
		return nil
	}); err != nil {
		t.Fatalf("Root.Walk(root) error = %v", err)
	}
	if !errors.Is(reportedMountError, files.ErrCrossDevice) {
		t.Fatalf("Root.Walk(root) mount error = %v, want ErrCrossDevice", reportedMountError)
	}
	if sawMountedContent {
		t.Fatal("Root.Walk(root) entered mounted content")
	}
}

func assertSameDevice(t *testing.T, left, right string) {
	t.Helper()
	leftDevice := deviceNumber(t, left)
	rightDevice := deviceNumber(t, right)
	if leftDevice != rightDevice {
		t.Fatalf("fixture is not same-device: %d != %d", leftDevice, rightDevice)
	}
}

func assertDifferentDevice(t *testing.T, left, right string) {
	t.Helper()
	leftDevice := deviceNumber(t, left)
	rightDevice := deviceNumber(t, right)
	if leftDevice == rightDevice {
		t.Fatalf("fixture is not cross-device: both report %d", leftDevice)
	}
}

func deviceNumber(t *testing.T, name string) uint64 {
	t.Helper()
	info, err := os.Stat(name)
	if err != nil {
		t.Fatalf("stat %q: %v", name, err)
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		t.Fatalf("stat %q has no syscall.Stat_t", name)
	}
	return uint64(stat.Dev)
}
