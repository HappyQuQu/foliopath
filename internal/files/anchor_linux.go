//go:build linux

package files

import (
	"errors"
	"io/fs"
	"os"

	"golang.org/x/sys/unix"
)

const linuxPathResolveFlags = unix.RESOLVE_BENEATH |
	unix.RESOLVE_NO_SYMLINKS |
	unix.RESOLVE_NO_XDEV

// anchoredRoot is the Linux filesystem adapter. Every path below the trusted
// media root is opened by openat2, so the kernel—not a preceding userspace
// check—enforces beneath, no-symlink, and no-mount-crossing semantics.
type anchoredRoot struct {
	file     *os.File
	boundary *os.File
	relative string
}

func openAnchoredRoot(name string) (*anchoredRoot, error) {
	fd, err := unix.Open(
		name,
		unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW,
		0,
	)
	if err != nil {
		return nil, linuxRootOpenError(err)
	}
	file := os.NewFile(uintptr(fd), ".")
	root := &anchoredRoot{
		file:     file,
		boundary: file,
	}

	// Fail closed on kernels or seccomp profiles that do not support the
	// required openat2 resolution flags. Linux must never silently fall back to
	// the userspace-only adapter.
	probe, err := root.openFile(".", unix.O_PATH|unix.O_NOFOLLOW)
	if err != nil {
		_ = root.Close()
		if errors.Is(err, ErrKernelBoundaryUnavailable) {
			return nil, err
		}
		return nil, errors.Join(ErrKernelBoundaryUnavailable, err)
	}
	if err := probe.Close(); err != nil {
		_ = root.Close()
		return nil, err
	}
	return root, nil
}

func (root *anchoredRoot) OpenRoot(name string) (*anchoredRoot, error) {
	relative := root.resolve(name)
	file, err := root.openResolved(
		relative,
		unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NONBLOCK|unix.O_NOFOLLOW,
	)
	if err != nil {
		return nil, err
	}
	if relative == "." {
		relative = ""
	}
	return &anchoredRoot{
		file:     file,
		boundary: root.boundary,
		relative: relative,
	}, nil
}

func (root *anchoredRoot) OpenFile(name string, flag int, _ fs.FileMode) (*os.File, error) {
	return root.openFile(name, flag)
}

func (root *anchoredRoot) Open(name string) (*os.File, error) {
	return root.openFile(name, unix.O_RDONLY|unix.O_NONBLOCK|unix.O_NOFOLLOW)
}

func (root *anchoredRoot) Stat(name string) (fs.FileInfo, error) {
	if name == "" || name == "." {
		return root.file.Stat()
	}
	return root.stat(name)
}

func (root *anchoredRoot) Lstat(name string) (fs.FileInfo, error) {
	return root.stat(name)
}

func (root *anchoredRoot) stat(name string) (fs.FileInfo, error) {
	file, err := root.openFile(name, unix.O_PATH|unix.O_NOFOLLOW)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return file.Stat()
}

func (root *anchoredRoot) openFile(name string, flag int) (*os.File, error) {
	return root.openResolved(root.resolve(name), flag)
}

func (root *anchoredRoot) openResolved(relative string, flag int) (*os.File, error) {
	fd, err := unix.Openat2(int(root.boundary.Fd()), relative, &unix.OpenHow{
		Flags: uint64(flag | unix.O_CLOEXEC),
		Resolve: uint64(
			linuxPathResolveFlags,
		),
	})
	if err != nil {
		return nil, linuxOpenat2Error(err)
	}
	return os.NewFile(uintptr(fd), relative), nil
}

func (root *anchoredRoot) resolve(name string) string {
	if name == "" || name == "." {
		if root.relative == "" {
			return "."
		}
		return root.relative
	}
	if root.relative == "" {
		return name
	}
	return root.relative + "/" + name
}

func (root *anchoredRoot) Close() error {
	return root.file.Close()
}

func linuxRootOpenError(err error) error {
	switch {
	case errors.Is(err, unix.ELOOP):
		return ErrSymlink
	case errors.Is(err, unix.ENOTDIR):
		return ErrNotDirectory
	default:
		return err
	}
}

func linuxOpenat2Error(err error) error {
	switch {
	case errors.Is(err, unix.EXDEV):
		return ErrCrossDevice
	case errors.Is(err, unix.ELOOP):
		return ErrSymlink
	case errors.Is(err, unix.ENOTDIR):
		return ErrNotDirectory
	case errors.Is(err, unix.EAGAIN), errors.Is(err, unix.ESTALE):
		return ErrChanged
	case errors.Is(err, unix.ENOSYS), errors.Is(err, unix.E2BIG), errors.Is(err, unix.EINVAL):
		return ErrKernelBoundaryUnavailable
	default:
		return err
	}
}
