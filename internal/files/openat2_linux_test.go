//go:build linux

package files

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"testing"

	"golang.org/x/sys/unix"
)

func TestLinuxOpenat2RejectsExistingMountBoundary(t *testing.T) {
	if _, err := os.Stat("/proc"); err != nil {
		t.Skipf("/proc is unavailable: %v", err)
	}
	root, err := OpenRoot("/")
	if err != nil {
		t.Fatalf("OpenRoot(/): %v", err)
	}
	defer root.Close()

	file, err := root.Open("proc/version")
	if file != nil {
		_ = file.Close()
	}
	if !errors.Is(err, ErrCrossDevice) {
		t.Fatalf("Open(proc/version) error = %v, want ErrCrossDevice", err)
	}

	directory, err := root.OpenDir("proc")
	if directory != nil {
		_ = directory.Close()
	}
	if !errors.Is(err, ErrCrossDevice) {
		t.Fatalf("OpenDir(proc) error = %v, want ErrCrossDevice", err)
	}

	if _, err := root.CaptureAt("proc"); !errors.Is(err, ErrCrossDevice) {
		t.Fatalf("CaptureAt(proc) error = %v, want ErrCrossDevice", err)
	}

	if err := root.Walk(context.Background(), "proc", func(string, fs.DirEntry, error) error {
		t.Fatal("Walk callback ran inside /proc")
		return nil
	}); !errors.Is(err, ErrCrossDevice) {
		t.Fatalf("Walk(proc) error = %v, want ErrCrossDevice", err)
	}
}

func TestLinuxOpenat2ErrorsFailClosed(t *testing.T) {
	for _, testCase := range []struct {
		name string
		err  error
		want error
	}{
		{name: "mount crossing", err: unix.EXDEV, want: ErrCrossDevice},
		{name: "symlink", err: unix.ELOOP, want: ErrSymlink},
		{name: "rename race", err: unix.EAGAIN, want: ErrChanged},
		{name: "stale handle", err: unix.ESTALE, want: ErrChanged},
		{name: "missing syscall", err: unix.ENOSYS, want: ErrKernelBoundaryUnavailable},
		{name: "unsupported flags", err: unix.EINVAL, want: ErrKernelBoundaryUnavailable},
		{name: "unsupported struct", err: unix.E2BIG, want: ErrKernelBoundaryUnavailable},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if err := linuxOpenat2Error(testCase.err); !errors.Is(err, testCase.want) {
				t.Fatalf("linuxOpenat2Error(%v) = %v, want %v", testCase.err, err, testCase.want)
			}
		})
	}
}
