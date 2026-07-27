package files

import (
	"context"
	"errors"
	"io"
	"io/fs"

	"github.com/HappyQuQu/foliopath/internal/library"
)

const directoryEnumerationBatchSize = 256

// DirectorySource adapts the anchored media root to library.DirectorySource.
// It reports only direct directory candidates and never follows directory
// symlinks or crosses a descendant mount.
type DirectorySource struct {
	root *Root
}

func NewDirectorySource(root *Root) (*DirectorySource, error) {
	if root == nil {
		return nil, errors.New("files directory source requires a root")
	}
	return &DirectorySource{root: root}, nil
}

func (source *DirectorySource) EnumerateDirectories(
	ctx context.Context,
	parent string,
	visit func(library.DirectoryCandidate) error,
) error {
	if ctx == nil || visit == nil {
		return fs.ErrInvalid
	}
	directory, err := source.root.OpenDir(parent)
	if err != nil {
		return mapParentEnumerationError(err)
	}
	defer directory.Close()

	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		entries, readErr := directory.Read(directoryEnumerationBatchSize)
		for _, entry := range entries {
			if err := ctx.Err(); err != nil {
				return err
			}
			candidate, include, inspectErr := source.inspectDirectory(directory, parent, entry)
			if inspectErr != nil {
				return inspectErr
			}
			if include {
				if err := visit(candidate); err != nil {
					return err
				}
			}
		}
		if errors.Is(readErr, io.EOF) {
			return nil
		}
		if readErr != nil {
			return mapParentEnumerationError(readErr)
		}
	}
}

func (source *DirectorySource) inspectDirectory(
	parent *Dir,
	parentPath string,
	raw fs.DirEntry,
) (library.DirectoryCandidate, bool, error) {
	name := raw.Name()
	info, err := parent.root.Lstat(name)
	if err != nil {
		if raw.IsDir() || raw.Type()&fs.ModeSymlink != 0 {
			return blockedDirectoryCandidate(name, err), true, nil
		}
		return library.DirectoryCandidate{}, false, nil
	}
	if info.Mode()&fs.ModeSymlink != 0 {
		return library.DirectoryCandidate{
			Name:          name,
			BlockedReason: library.SelectionBlockedSymlink,
		}, true, nil
	}
	if !info.IsDir() {
		return library.DirectoryCandidate{}, false, nil
	}
	if !source.root.identity.sameFilesystem(info) {
		return library.DirectoryCandidate{
			Name:          name,
			BlockedReason: library.SelectionBlockedMountBoundary,
		}, true, nil
	}

	relative := joinRelative(parentPath, name)
	child, err := source.root.openChildDir(parent, relative, name, info)
	if err != nil {
		if errors.Is(err, ErrOffline) {
			return library.DirectoryCandidate{}, false, mapParentEnumerationError(err)
		}
		return blockedDirectoryCandidate(name, err), true, nil
	}
	defer child.Close()

	hasChildren, err := directoryHasChildren(child)
	if err != nil {
		return blockedDirectoryCandidate(name, err), true, nil
	}
	return library.DirectoryCandidate{
		Name:        name,
		HasChildren: hasChildren,
	}, true, nil
}

func directoryHasChildren(directory *Dir) (bool, error) {
	entries, err := directory.Read(directoryEnumerationBatchSize)
	if err != nil && !errors.Is(err, io.EOF) {
		return false, err
	}
	for _, entry := range entries {
		if entry.IsDir() || entry.Type()&fs.ModeSymlink != 0 {
			return true, nil
		}
	}
	return false, nil
}

func blockedDirectoryCandidate(name string, err error) library.DirectoryCandidate {
	reason := library.SelectionBlockedUnavailable
	switch {
	case errors.Is(err, ErrSymlink):
		reason = library.SelectionBlockedSymlink
	case errors.Is(err, ErrCrossDevice):
		reason = library.SelectionBlockedMountBoundary
	case errors.Is(err, fs.ErrPermission):
		reason = library.SelectionBlockedUnreadable
	}
	return library.DirectoryCandidate{Name: name, BlockedReason: reason}
}

func mapParentEnumerationError(err error) error {
	switch {
	case errors.Is(err, ErrInvalidPath):
		return library.ErrInvalidParent
	case errors.Is(err, ErrSymlink):
		return library.ErrParentSymlink
	case errors.Is(err, ErrCrossDevice), errors.Is(err, ErrKernelBoundaryUnavailable):
		return library.ErrParentMountBoundary
	case errors.Is(err, ErrOffline),
		errors.Is(err, ErrRootChanged),
		errors.Is(err, ErrNotDirectory),
		errors.Is(err, fs.ErrNotExist),
		errors.Is(err, fs.ErrPermission):
		return library.ErrParentUnavailable
	default:
		return err
	}
}
