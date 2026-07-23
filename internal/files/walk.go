package files

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"path"
)

const walkBatchSize = 256

type walkFrame struct {
	path    string
	entry   fs.DirEntry
	dir     *Dir
	pending []fs.DirEntry
	index   int
}

// Walk traverses a directory tree in bounded batches and calls walkFn in
// pre-order with root-relative paths. The empty start path represents the root.
//
// Every child is Lstat'd. Symlinks, non-regular/non-directory nodes, entries on
// another filesystem, and names rejected by Normalize are reported to walkFn
// with a non-nil error and are never opened. Returning nil records/ignores that
// skipped entry and continues. A directory callback may return fs.SkipDir, and
// any callback may return fs.SkipAll. The caller can skip maintained system
// directories by returning fs.SkipDir for those directory names.
func (root *Root) Walk(ctx context.Context, start string, walkFn fs.WalkDirFunc) error {
	return root.walk(ctx, start, Identity{}, false, walkFn)
}

// walkCaptured is used by ScanWalker to bind traversal to the exact library
// root captured before the scan. This prevents an A -> B -> A path replacement
// from scanning B and then passing a final identity check after A is restored.
func (root *Root) walkCaptured(
	ctx context.Context,
	start string,
	expected Identity,
	walkFn fs.WalkDirFunc,
) error {
	if !expected.valid() {
		return opError("walk", start, ErrInvalidIdentity)
	}
	return root.walk(ctx, start, expected, true, walkFn)
}

func (root *Root) walk(
	ctx context.Context,
	start string,
	expected Identity,
	requireExpected bool,
	walkFn fs.WalkDirFunc,
) error {
	if ctx == nil {
		return opError("walk", "", fs.ErrInvalid)
	}
	if walkFn == nil {
		return opError("walk", "", fs.ErrInvalid)
	}
	normalized, err := Normalize(start)
	if err != nil {
		return opError("walk", "", err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	directory, err := root.OpenDir(normalized)
	if err != nil {
		return err
	}
	if requireExpected && !expected.Equal(directory.identity) {
		_ = directory.Close()
		return opError("walk", normalized, changedRootError())
	}
	rootEntry := fs.FileInfoToDirEntry(directory.info)
	callbackErr := walkFn(normalized, rootEntry, nil)
	switch {
	case callbackErr == nil:
	case errors.Is(callbackErr, fs.SkipDir), errors.Is(callbackErr, fs.SkipAll):
		_ = directory.Close()
		return nil
	default:
		_ = directory.Close()
		return callbackErr
	}

	stack := []*walkFrame{{path: normalized, entry: rootEntry, dir: directory}}
	defer func() {
		closeWalkFrames(stack)
	}()

	for len(stack) > 0 {
		if err := ctx.Err(); err != nil {
			return err
		}
		frame := stack[len(stack)-1]
		if frame.index >= len(frame.pending) {
			entries, readErr := frame.dir.Read(walkBatchSize)
			frame.pending = entries
			frame.index = 0
			if readErr != nil && !errors.Is(readErr, io.EOF) {
				if errors.Is(readErr, ErrOffline) {
					return readErr
				}
				callbackErr := walkFn(frame.path, frame.entry, readErr)
				if callbackErr != nil && !errors.Is(callbackErr, fs.SkipDir) && !errors.Is(callbackErr, fs.SkipAll) {
					return callbackErr
				}
				popWalkFrame(&stack)
				if errors.Is(callbackErr, fs.SkipAll) {
					return nil
				}
				continue
			}
			if len(entries) == 0 {
				popWalkFrame(&stack)
				continue
			}
		}

		rawEntry := frame.pending[frame.index]
		frame.index++
		if err := ctx.Err(); err != nil {
			return err
		}

		info, lstatErr := frame.dir.root.Lstat(rawEntry.Name())
		childPath := joinRelative(frame.path, rawEntry.Name())
		if lstatErr != nil {
			if stop, skipParent, err := callWalk(walkFn, childPath, rawEntry, lstatErr, false); stop {
				return err
			} else if skipParent {
				popWalkFrame(&stack)
			}
			continue
		}
		entry := fs.FileInfoToDirEntry(info)

		if _, normalizeErr := Normalize(childPath); normalizeErr != nil {
			if stop, skipParent, err := callWalk(walkFn, childPath, entry, normalizeErr, info.IsDir()); stop {
				return err
			} else if skipParent {
				popWalkFrame(&stack)
			}
			continue
		}
		if info.Mode()&fs.ModeSymlink != 0 {
			if stop, skipParent, err := callWalk(walkFn, childPath, entry, ErrSymlink, false); stop {
				return err
			} else if skipParent {
				popWalkFrame(&stack)
			}
			continue
		}
		if !root.identity.sameFilesystem(info) {
			if stop, skipParent, err := callWalk(walkFn, childPath, entry, ErrCrossDevice, info.IsDir()); stop {
				return err
			} else if skipParent {
				popWalkFrame(&stack)
			}
			continue
		}
		if !info.IsDir() && !info.Mode().IsRegular() {
			if stop, skipParent, err := callWalk(walkFn, childPath, entry, ErrSpecialFile, false); stop {
				return err
			} else if skipParent {
				popWalkFrame(&stack)
			}
			continue
		}

		callbackErr = walkFn(childPath, entry, nil)
		if errors.Is(callbackErr, fs.SkipAll) {
			return nil
		}
		if callbackErr != nil && !errors.Is(callbackErr, fs.SkipDir) {
			return callbackErr
		}
		if !info.IsDir() {
			if errors.Is(callbackErr, fs.SkipDir) {
				popWalkFrame(&stack)
			}
			continue
		}
		if errors.Is(callbackErr, fs.SkipDir) {
			continue
		}

		childDir, openErr := root.openChildDir(frame.dir, childPath, rawEntry.Name(), info)
		if openErr != nil {
			if errors.Is(openErr, ErrOffline) {
				return openErr
			}
			if stop, skipParent, err := callWalk(walkFn, childPath, entry, openErr, true); stop {
				return err
			} else if skipParent {
				popWalkFrame(&stack)
			}
			continue
		}
		stack = append(stack, &walkFrame{path: childPath, entry: entry, dir: childDir})
	}
	return nil
}

func (root *Root) openChildDir(parent *Dir, relative, name string, before fs.FileInfo) (*Dir, error) {
	parent.owner.mu.RLock()
	defer parent.owner.mu.RUnlock()
	if parent.owner.closed {
		return nil, opError("walk", relative, fs.ErrClosed)
	}
	if _, err := parent.owner.captureLocked(); err != nil {
		return nil, opError("walk", relative, err)
	}

	next, err := parent.root.OpenRoot(name)
	if err != nil {
		return nil, parent.owner.operationErrorLocked("walk", relative, err)
	}
	opened, err := next.Stat(".")
	if err != nil {
		_ = next.Close()
		return nil, parent.owner.operationErrorLocked("walk", relative, err)
	}
	after, err := parent.root.Lstat(name)
	if err != nil {
		_ = next.Close()
		return nil, parent.owner.operationErrorLocked("walk", relative, err)
	}
	if after.Mode()&fs.ModeSymlink != 0 || !opened.IsDir() || !sameFileInfo(before, opened) || !sameFileInfo(opened, after) {
		_ = next.Close()
		return nil, opError("walk", relative, ErrChanged)
	}
	if !root.identity.sameFilesystem(opened) {
		_ = next.Close()
		return nil, opError("walk", relative, ErrCrossDevice)
	}
	file, err := next.Open(".")
	if err != nil {
		_ = next.Close()
		return nil, parent.owner.operationErrorLocked("walk", relative, err)
	}
	return &Dir{
		owner:    root,
		root:     next,
		file:     file,
		path:     relative,
		info:     opened,
		identity: identityFromInfo(opened),
	}, nil
}

func callWalk(walkFn fs.WalkDirFunc, relative string, entry fs.DirEntry, reported error, isDir bool) (stop, skipParent bool, err error) {
	callbackErr := walkFn(relative, entry, opError("walk", relative, reported))
	switch {
	case callbackErr == nil:
		return false, false, nil
	case errors.Is(callbackErr, fs.SkipAll):
		return true, false, nil
	case errors.Is(callbackErr, fs.SkipDir):
		return false, !isDir, nil
	default:
		return true, false, callbackErr
	}
}

func joinRelative(parent, name string) string {
	if parent == "" {
		return name
	}
	return path.Join(parent, name)
}

func popWalkFrame(stack *[]*walkFrame) {
	last := len(*stack) - 1
	_ = (*stack)[last].dir.Close()
	*stack = (*stack)[:last]
}

func closeWalkFrames(stack []*walkFrame) {
	for index := len(stack) - 1; index >= 0; index-- {
		_ = stack[index].dir.Close()
	}
}
