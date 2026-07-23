// Package files provides the only filesystem boundary for read-only media.
package files

import (
	"errors"
	"fmt"
	"io/fs"
	"strconv"
)

var (
	// ErrInvalidRoot means the configured root is not an acceptable directory.
	ErrInvalidRoot = errors.New("invalid media root")
	// ErrInvalidPath means a caller supplied a non-canonical or unsafe relative path.
	ErrInvalidPath = errors.New("invalid relative path")
	// ErrInvalidIdentity means a zero or otherwise unusable root identity was supplied.
	ErrInvalidIdentity = errors.New("invalid root identity")
	// ErrOffline means the configured root is missing or unreadable.
	ErrOffline = errors.New("media root offline")
	// ErrRootChanged means the path now names a different directory than it did when captured.
	ErrRootChanged = errors.New("media root identity changed")
	// ErrSymlink means an operation would follow a symbolic link.
	ErrSymlink = errors.New("symbolic link not allowed")
	// ErrCrossDevice means an operation would cross a filesystem or mount boundary.
	ErrCrossDevice = errors.New("filesystem boundary crossed")
	// ErrSpecialFile means an entry is neither a regular file nor a directory.
	ErrSpecialFile = errors.New("special filesystem entry not allowed")
	// ErrNotRegular means a regular file was required.
	ErrNotRegular = errors.New("not a regular file")
	// ErrNotDirectory means a directory was required.
	ErrNotDirectory = errors.New("not a directory")
	// ErrChanged means an entry changed while it was being validated or opened.
	ErrChanged = errors.New("filesystem entry changed during access")
	// ErrKernelBoundaryUnavailable means Linux cannot enforce the required
	// openat2 path-resolution policy. Linux fails closed instead of falling back.
	ErrKernelBoundaryUnavailable = errors.New("kernel path boundary unavailable")
)

// Error describes a failed operation. Path is always relative to the media root;
// it never contains the configured host or container root path.
type Error struct {
	Op   string
	Path string
	Err  error
}

func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	location := ""
	if e.Path != "" {
		location = " " + strconv.Quote(e.Path)
	}
	return fmt.Sprintf("files: %s%s: %s", e.Op, location, safeErrorText(e.Err))
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func opError(op, relative string, err error) error {
	if err == nil {
		return nil
	}
	return &Error{Op: op, Path: relative, Err: err}
}

func safeErrorText(err error) string {
	for _, policy := range []error{
		ErrInvalidRoot,
		ErrInvalidPath,
		ErrInvalidIdentity,
		ErrOffline,
		ErrRootChanged,
		ErrSymlink,
		ErrCrossDevice,
		ErrSpecialFile,
		ErrNotRegular,
		ErrNotDirectory,
		ErrChanged,
		ErrKernelBoundaryUnavailable,
	} {
		if errors.Is(err, policy) {
			// Policy errors are generated locally and contain no absolute paths.
			return err.Error()
		}
	}

	for _, portable := range []error{
		fs.ErrPermission,
		fs.ErrNotExist,
		fs.ErrExist,
		fs.ErrClosed,
		fs.ErrInvalid,
	} {
		if errors.Is(err, portable) {
			return portable.Error()
		}
	}
	return "filesystem operation failed"
}

func offlineError(cause error) error {
	if errors.Is(cause, fs.ErrPermission) {
		return fmt.Errorf("%w: %w", ErrOffline, fs.ErrPermission)
	}
	if errors.Is(cause, fs.ErrNotExist) {
		return fmt.Errorf("%w: %w", ErrOffline, fs.ErrNotExist)
	}
	return ErrOffline
}

func changedRootError() error {
	return fmt.Errorf("%w: %w", ErrOffline, ErrRootChanged)
}
