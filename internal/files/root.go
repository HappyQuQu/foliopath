package files

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"unicode/utf8"
)

// Root anchors all access to one read-only media directory.
//
// Root combines platform-anchored access with no-symlink and root-identity
// checks. Linux additionally enforces no-mount-crossing in the kernel. Root is
// safe for concurrent use.
type Root struct {
	mu       sync.RWMutex
	anchor   *anchoredRoot
	path     string
	identity Identity
	closed   bool
}

// OpenRoot opens and captures an absolute, trusted media boundary such as the
// container's fixed /library mount. name must come from deployment
// configuration, never from a user-selected library path. User-selected
// library roots are resolved below this boundary with CaptureAt and VerifyAt.
//
// Missing or unreadable roots return an error matching ErrOffline. The final
// root component may not itself be a symbolic link. Linux also fails closed
// with ErrKernelBoundaryUnavailable when its required openat2 policy cannot be
// enforced.
func OpenRoot(name string) (*Root, error) {
	if name == "" || !utf8.ValidString(name) || strings.IndexByte(name, 0) >= 0 || !filepath.IsAbs(name) {
		return nil, opError("open-root", "", ErrInvalidRoot)
	}
	name = filepath.Clean(name)

	info, err := inspectRootPath(name)
	if err != nil {
		return nil, openRootError(err)
	}
	anchor, err := openAnchoredRoot(name)
	if err != nil {
		return nil, openRootError(err)
	}
	anchorInfo, err := anchor.Stat(".")
	if err != nil {
		_ = anchor.Close()
		return nil, opError("open-root", "", offlineError(err))
	}
	if !sameFileInfo(info, anchorInfo) {
		_ = anchor.Close()
		return nil, opError("open-root", "", changedRootError())
	}

	root := &Root{
		anchor:   anchor,
		path:     name,
		identity: identityFromInfo(anchorInfo),
	}
	if _, err := root.captureLocked(); err != nil {
		_ = anchor.Close()
		return nil, opError("open-root", "", err)
	}
	return root, nil
}

func openRootError(err error) error {
	switch {
	case errors.Is(err, ErrSymlink):
		return opError("open-root", "", errors.Join(ErrInvalidRoot, ErrSymlink))
	case errors.Is(err, ErrNotDirectory):
		return opError("open-root", "", errors.Join(ErrInvalidRoot, ErrNotDirectory))
	case errors.Is(err, ErrRootChanged):
		return opError("open-root", "", changedRootError())
	case errors.Is(err, ErrKernelBoundaryUnavailable):
		return opError("open-root", "", ErrKernelBoundaryUnavailable)
	default:
		return opError("open-root", "", offlineError(err))
	}
}

// Close releases the root anchor. Files already returned by Open remain the
// caller's responsibility.
func (root *Root) Close() error {
	root.mu.Lock()
	defer root.mu.Unlock()
	if root.closed {
		return fs.ErrClosed
	}
	root.closed = true
	return root.anchor.Close()
}

// Capture returns the root's current device/inode identity after verifying the
// configured path is still readable and still points at the anchored root.
func (root *Root) Capture() (Identity, error) {
	root.mu.RLock()
	defer root.mu.RUnlock()
	if root.closed {
		return Identity{}, opError("capture", "", fs.ErrClosed)
	}
	identity, err := root.captureLocked()
	if err != nil {
		return Identity{}, opError("capture", "", err)
	}
	return identity, nil
}

// Verify checks that the configured path still names the captured identity.
// A changed or replaced root matches both ErrOffline and ErrRootChanged.
func (root *Root) Verify(expected Identity) error {
	root.mu.RLock()
	defer root.mu.RUnlock()
	if root.closed {
		return opError("verify", "", fs.ErrClosed)
	}
	if !expected.valid() {
		return opError("verify", "", ErrInvalidIdentity)
	}
	current, err := root.captureLocked()
	if err != nil {
		return opError("verify", "", err)
	}
	if !expected.Equal(current) {
		return opError("verify", "", changedRootError())
	}
	return nil
}

// ReadOnly reports whether the anchored filesystem itself is mounted read-only.
// Linux uses the already-open root descriptor. Non-Linux adapters fail closed
// because their development-only path boundary is not release evidence for a
// direct model source.
func (root *Root) ReadOnly() (bool, error) {
	root.mu.RLock()
	defer root.mu.RUnlock()
	if root.closed {
		return false, opError("read-only", "", fs.ErrClosed)
	}
	if _, err := root.captureLocked(); err != nil {
		return false, opError("read-only", "", err)
	}
	readOnly, err := anchoredRootReadOnly(root.anchor)
	if err != nil {
		return false, opError("read-only", "", err)
	}
	if _, err := root.captureLocked(); err != nil {
		return false, opError("read-only", "", err)
	}
	return readOnly, nil
}

// CaptureAt captures the identity of a user-selected library root relative to
// the trusted media boundary. Every platform rejects symlinks and
// cross-filesystem entries; Linux additionally rejects every nested mount
// atomically through openat2. The empty path identifies the boundary itself.
func (root *Root) CaptureAt(relative string) (Identity, error) {
	normalized, err := Normalize(relative)
	if err != nil {
		return Identity{}, opError("capture-at", "", err)
	}

	root.mu.RLock()
	defer root.mu.RUnlock()
	if root.closed {
		return Identity{}, opError("capture-at", normalized, fs.ErrClosed)
	}
	identity, err := root.captureAtLocked(normalized)
	if err != nil {
		return Identity{}, opError("capture-at", normalized, root.captureAtErrorLocked(err, false))
	}
	return identity, nil
}

// VerifyAt checks that a user-selected library root still identifies the same
// directory captured earlier. A missing, replaced, newly symlinked, or newly
// cross-device library root matches ErrOffline and ErrRootChanged, preventing
// an empty replacement directory from being treated as a successful scan.
func (root *Root) VerifyAt(relative string, expected Identity) error {
	normalized, err := Normalize(relative)
	if err != nil {
		return opError("verify-at", "", err)
	}
	if !expected.valid() {
		return opError("verify-at", normalized, ErrInvalidIdentity)
	}

	root.mu.RLock()
	defer root.mu.RUnlock()
	if root.closed {
		return opError("verify-at", normalized, fs.ErrClosed)
	}
	current, err := root.captureAtLocked(normalized)
	if err != nil {
		return opError("verify-at", normalized, root.captureAtErrorLocked(err, true))
	}
	if !expected.Equal(current) {
		return opError("verify-at", normalized, changedRootError())
	}
	return nil
}

func (root *Root) captureAtLocked(relative string) (Identity, error) {
	if _, err := root.captureLocked(); err != nil {
		return Identity{}, err
	}
	directory, info, err := root.openDirRootLocked(relative)
	if err != nil {
		return Identity{}, err
	}
	if err := directory.Close(); err != nil {
		return Identity{}, err
	}
	if _, err := root.captureLocked(); err != nil {
		return Identity{}, err
	}
	return identityFromInfo(info), nil
}

func (root *Root) captureAtErrorLocked(cause error, verifying bool) error {
	if _, err := root.captureLocked(); err != nil {
		return err
	}
	if verifying && (errors.Is(cause, fs.ErrNotExist) ||
		errors.Is(cause, fs.ErrPermission) ||
		errors.Is(cause, ErrSymlink) ||
		errors.Is(cause, ErrNotDirectory) ||
		errors.Is(cause, ErrCrossDevice) ||
		errors.Is(cause, ErrChanged)) {
		return changedRootError()
	}
	if errors.Is(cause, fs.ErrNotExist) || errors.Is(cause, fs.ErrPermission) {
		return offlineError(cause)
	}
	if errors.Is(cause, ErrSymlink) || errors.Is(cause, ErrNotDirectory) || errors.Is(cause, ErrCrossDevice) {
		return cause
	}
	if errors.Is(cause, ErrChanged) {
		return changedRootError()
	}
	return offlineError(cause)
}

func (root *Root) captureLocked() (Identity, error) {
	currentInfo, err := inspectRootPath(root.path)
	if err != nil {
		if errors.Is(err, ErrSymlink) || errors.Is(err, ErrNotDirectory) || errors.Is(err, ErrRootChanged) {
			return Identity{}, changedRootError()
		}
		return Identity{}, offlineError(err)
	}
	anchorInfo, err := root.anchor.Stat(".")
	if err != nil {
		return Identity{}, offlineError(err)
	}
	current := identityFromInfo(currentInfo)
	if !root.identity.matches(anchorInfo) || !root.identity.Equal(current) {
		return Identity{}, changedRootError()
	}
	return current, nil
}

func inspectRootPath(name string) (fs.FileInfo, error) {
	lstat, err := os.Lstat(name)
	if err != nil {
		return nil, err
	}
	if lstat.Mode()&fs.ModeSymlink != 0 {
		return nil, ErrSymlink
	}
	if !lstat.IsDir() {
		return nil, ErrNotDirectory
	}

	directory, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer directory.Close()
	opened, err := directory.Stat()
	if err != nil {
		return nil, err
	}
	if !sameFileInfo(lstat, opened) {
		return nil, ErrRootChanged
	}
	_, err = directory.ReadDir(1)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	return opened, nil
}

// Open opens a regular file for read-only access. No path component may be a
// symlink, and the file must be on the same filesystem as the root.
func (root *Root) Open(relative string) (*os.File, error) {
	normalized, err := Normalize(relative)
	if err != nil {
		return nil, opError("open", "", err)
	}
	if normalized == "" {
		return nil, opError("open", normalized, ErrNotRegular)
	}

	root.mu.RLock()
	defer root.mu.RUnlock()
	if root.closed {
		return nil, opError("open", normalized, fs.ErrClosed)
	}
	if _, err := root.captureLocked(); err != nil {
		return nil, opError("open", normalized, err)
	}

	parentPath, base := splitParent(normalized)
	parent, _, err := root.openDirRootLocked(parentPath)
	if err != nil {
		return nil, root.operationErrorLocked("open", normalized, err)
	}
	defer parent.Close()

	before, err := parent.Lstat(base)
	if err != nil {
		return nil, root.operationErrorLocked("open", normalized, err)
	}
	if before.Mode()&fs.ModeSymlink != 0 {
		return nil, opError("open", normalized, ErrSymlink)
	}
	if !before.Mode().IsRegular() {
		return nil, opError("open", normalized, ErrNotRegular)
	}
	if !root.identity.sameFilesystem(before) {
		return nil, opError("open", normalized, ErrCrossDevice)
	}

	file, err := parent.OpenFile(base, os.O_RDONLY|platformReadOnlyFlags(), 0)
	if err != nil {
		return nil, root.operationErrorLocked("open", normalized, err)
	}
	opened, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, root.operationErrorLocked("open", normalized, err)
	}
	after, err := parent.Lstat(base)
	if err != nil {
		_ = file.Close()
		return nil, root.operationErrorLocked("open", normalized, err)
	}
	if after.Mode()&fs.ModeSymlink != 0 || !opened.Mode().IsRegular() || !sameFileInfo(before, opened) || !sameFileInfo(opened, after) {
		_ = file.Close()
		return nil, opError("open", normalized, ErrChanged)
	}
	if !root.identity.sameFilesystem(opened) {
		_ = file.Close()
		return nil, opError("open", normalized, ErrCrossDevice)
	}
	if _, err := root.captureLocked(); err != nil {
		_ = file.Close()
		return nil, opError("open", normalized, err)
	}
	return file, nil
}

// Dir is a streaming, no-symlink directory reader returned by OpenDir.
type Dir struct {
	mu       sync.Mutex
	owner    *Root
	root     *anchoredRoot
	file     *os.File
	path     string
	info     fs.FileInfo
	identity Identity
	closed   bool
}

// OpenDir opens a directory without following symlinks or crossing a
// filesystem boundary. The empty relative path opens the root itself.
func (root *Root) OpenDir(relative string) (*Dir, error) {
	normalized, err := Normalize(relative)
	if err != nil {
		return nil, opError("open-dir", "", err)
	}

	root.mu.RLock()
	defer root.mu.RUnlock()
	if root.closed {
		return nil, opError("open-dir", normalized, fs.ErrClosed)
	}
	if _, err := root.captureLocked(); err != nil {
		return nil, opError("open-dir", normalized, err)
	}
	directory, info, err := root.openDirRootLocked(normalized)
	if err != nil {
		return nil, root.operationErrorLocked("open-dir", normalized, err)
	}
	file, err := directory.Open(".")
	if err != nil {
		_ = directory.Close()
		return nil, root.operationErrorLocked("open-dir", normalized, err)
	}
	opened, err := file.Stat()
	if err != nil || !opened.IsDir() || !sameFileInfo(info, opened) {
		_ = file.Close()
		_ = directory.Close()
		if err == nil {
			err = ErrChanged
		}
		return nil, root.operationErrorLocked("open-dir", normalized, err)
	}
	if _, err := root.captureLocked(); err != nil {
		_ = file.Close()
		_ = directory.Close()
		return nil, opError("open-dir", normalized, err)
	}
	return &Dir{
		owner:    root,
		root:     directory,
		file:     file,
		path:     normalized,
		info:     opened,
		identity: identityFromInfo(opened),
	}, nil
}

// Read returns up to n entries without following them. It has the same n and
// io.EOF behavior as os.File.ReadDir.
func (directory *Dir) Read(n int) ([]fs.DirEntry, error) {
	directory.mu.Lock()
	defer directory.mu.Unlock()
	if directory.closed {
		return nil, opError("read-dir", directory.path, fs.ErrClosed)
	}

	directory.owner.mu.RLock()
	defer directory.owner.mu.RUnlock()
	if directory.owner.closed {
		return nil, opError("read-dir", directory.path, fs.ErrClosed)
	}
	if err := directory.verifyLocked(); err != nil {
		return nil, opError("read-dir", directory.path, err)
	}
	entries, err := directory.file.ReadDir(n)
	if verifyErr := directory.verifyLocked(); verifyErr != nil {
		return nil, opError("read-dir", directory.path, verifyErr)
	}
	if err != nil && !errors.Is(err, io.EOF) {
		return entries, directory.owner.operationErrorLocked("read-dir", directory.path, err)
	}
	return entries, err
}

func (directory *Dir) verifyLocked() error {
	if _, err := directory.owner.captureLocked(); err != nil {
		return err
	}
	currentRoot, currentInfo, err := directory.owner.openDirRootLocked(directory.path)
	if err != nil {
		return err
	}
	_ = currentRoot.Close()
	if !directory.identity.matches(currentInfo) {
		return ErrChanged
	}
	anchored, err := directory.root.Stat(".")
	if err != nil {
		return err
	}
	if !directory.identity.matches(anchored) {
		return ErrChanged
	}
	return nil
}

func (directory *Dir) Close() error {
	directory.mu.Lock()
	defer directory.mu.Unlock()
	if directory.closed {
		return fs.ErrClosed
	}
	directory.closed = true
	return errors.Join(directory.file.Close(), directory.root.Close())
}

// ReadDir reads and name-sorts one directory. Walk should be used for large,
// recursive trees because it reads directories in bounded batches.
func (root *Root) ReadDir(relative string) ([]fs.DirEntry, error) {
	directory, err := root.OpenDir(relative)
	if err != nil {
		return nil, err
	}
	defer directory.Close()
	entries, err := directory.Read(-1)
	if err != nil {
		return nil, err
	}
	slices.SortFunc(entries, func(left, right fs.DirEntry) int {
		return strings.Compare(left.Name(), right.Name())
	})
	return entries, nil
}

func (root *Root) openDirRootLocked(relative string) (*anchoredRoot, fs.FileInfo, error) {
	current, err := root.anchor.OpenRoot(".")
	if err != nil {
		return nil, nil, err
	}
	currentInfo, err := current.Stat(".")
	if err != nil {
		_ = current.Close()
		return nil, nil, err
	}
	if !root.identity.matches(currentInfo) {
		_ = current.Close()
		return nil, nil, ErrChanged
	}

	if relative == "" {
		return current, currentInfo, nil
	}
	for _, component := range strings.Split(relative, "/") {
		before, err := current.Lstat(component)
		if err != nil {
			_ = current.Close()
			return nil, nil, err
		}
		if before.Mode()&fs.ModeSymlink != 0 {
			_ = current.Close()
			return nil, nil, ErrSymlink
		}
		if !before.IsDir() {
			_ = current.Close()
			return nil, nil, ErrNotDirectory
		}
		if !root.identity.sameFilesystem(before) {
			_ = current.Close()
			return nil, nil, ErrCrossDevice
		}

		next, err := current.OpenRoot(component)
		if err != nil {
			_ = current.Close()
			return nil, nil, err
		}
		opened, err := next.Stat(".")
		if err != nil {
			_ = next.Close()
			_ = current.Close()
			return nil, nil, err
		}
		after, err := current.Lstat(component)
		if err != nil {
			_ = next.Close()
			_ = current.Close()
			return nil, nil, err
		}
		if after.Mode()&fs.ModeSymlink != 0 || !sameFileInfo(before, opened) || !sameFileInfo(opened, after) {
			_ = next.Close()
			_ = current.Close()
			return nil, nil, ErrChanged
		}
		_ = current.Close()
		current = next
		currentInfo = opened
	}
	return current, currentInfo, nil
}

func (root *Root) operationErrorLocked(op, relative string, cause error) error {
	if _, err := root.captureLocked(); err != nil {
		return opError(op, relative, err)
	}
	return opError(op, relative, cause)
}

func splitParent(relative string) (parent, base string) {
	parent = path.Dir(relative)
	if parent == "." {
		parent = ""
	}
	return parent, path.Base(relative)
}
