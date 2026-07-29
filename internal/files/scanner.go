package files

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"strings"
	"sync"

	"github.com/HappyQuQu/foliopath/internal/pathpolicy"
	"github.com/HappyQuQu/foliopath/internal/scanner"
)

// ScanWalker adapts the read-only filesystem boundary to scanner.Walker. One
// instance may be shared by scans for multiple non-overlapping libraries.
type ScanWalker struct {
	root *Root

	mu       sync.Mutex
	captured map[string]Identity
}

func NewScanWalker(root *Root) (*ScanWalker, error) {
	if root == nil {
		return nil, errors.New("files scan walker requires a root")
	}
	return &ScanWalker{
		root:     root,
		captured: make(map[string]Identity),
	}, nil
}

func (walker *ScanWalker) CaptureRoot(
	ctx context.Context,
	relativeRoot string,
) (scanner.RootIdentity, error) {
	if err := ctx.Err(); err != nil {
		return scanner.RootIdentity{}, err
	}
	identity, err := walker.root.CaptureAt(relativeRoot)
	if err != nil {
		return scanner.RootIdentity{}, scannerRootError(err)
	}
	device, inode, ok := identity.Key()
	if !ok || inode == 0 {
		return scanner.RootIdentity{}, scanner.ErrInvalidRootIdentity
	}

	walker.mu.Lock()
	walker.captured[relativeRoot] = identity
	walker.mu.Unlock()
	return scanner.RootIdentity{Device: device, Inode: inode}, nil
}

func (walker *ScanWalker) Walk(
	ctx context.Context,
	relativeRoot string,
	visit func(scanner.WalkEntry) (scanner.WalkDecision, error),
) error {
	walker.mu.Lock()
	identity, ok := walker.captured[relativeRoot]
	walker.mu.Unlock()
	if !ok {
		return scanner.ErrInvalidRootIdentity
	}

	var visitErr error
	err := walker.root.walkCaptured(ctx, relativeRoot, identity, func(relative string, entry fs.DirEntry, walkErr error) error {
		libraryRelative, err := relativeToLibrary(relativeRoot, relative)
		if err != nil {
			return err
		}
		if walkErr != nil {
			if isPolicySkip(walkErr) {
				_, err := visit(scanner.WalkEntry{
					RelativePath: libraryRelative,
					IsDirectory:  entry != nil && entry.IsDir(),
					Skipped:      true,
				})
				if err != nil {
					visitErr = err
				}
				return err
			}
			return scannerWalkError(walkErr)
		}

		info, err := entry.Info()
		if err != nil {
			return scannerWalkError(err)
		}
		decision, err := visit(scanner.WalkEntry{
			RelativePath: libraryRelative,
			IsDirectory:  info.IsDir(),
			SizeBytes:    info.Size(),
			MTimeNS:      info.ModTime().UnixNano(),
		})
		if err != nil {
			visitErr = err
			return err
		}
		if decision == scanner.WalkSkipDirectory {
			return fs.SkipDir
		}
		return nil
	})
	if visitErr != nil {
		return visitErr
	}
	return scannerWalkError(err)
}

func (walker *ScanWalker) ReadDirectory(
	ctx context.Context,
	relativeRoot string,
	relativeDirectory string,
	visit func(scanner.WalkEntry) error,
) error {
	if ctx == nil || visit == nil {
		return scanner.ErrInvalidEntry
	}
	walker.mu.Lock()
	_, captured := walker.captured[relativeRoot]
	walker.mu.Unlock()
	if !captured {
		return scanner.ErrInvalidRootIdentity
	}
	normalizedDirectory, err := pathpolicy.Normalize(relativeDirectory)
	if err != nil || normalizedDirectory != relativeDirectory {
		return scanner.ErrInvalidEntry
	}
	allowedRelative := relativeRoot
	if relativeDirectory != "" {
		allowedRelative = path.Join(relativeRoot, relativeDirectory)
	}
	directory, err := walker.root.OpenDir(allowedRelative)
	if err != nil {
		return scannerWalkError(err)
	}
	defer directory.Close()

	seen := 0
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		entries, readErr := directory.Read(directoryEnumerationBatchSize)
		for _, raw := range entries {
			if err := ctx.Err(); err != nil {
				return err
			}
			seen++
			if seen > scanner.MaxReconcileEntries {
				return scanner.ErrBatchTooLarge
			}
			libraryRelative := raw.Name()
			if relativeDirectory != "" {
				libraryRelative = path.Join(relativeDirectory, raw.Name())
			}
			info, statErr := directory.root.Lstat(raw.Name())
			if statErr != nil {
				return scannerWalkError(statErr)
			}
			skipped := info.Mode()&fs.ModeSymlink != 0 ||
				!walker.root.identity.sameFilesystem(info) ||
				(!info.IsDir() && !info.Mode().IsRegular())
			if err := visit(scanner.WalkEntry{
				RelativePath: libraryRelative,
				IsDirectory:  info.IsDir(),
				SizeBytes:    info.Size(),
				MTimeNS:      info.ModTime().UnixNano(),
				Skipped:      skipped,
			}); err != nil {
				return err
			}
		}
		if errors.Is(readErr, io.EOF) {
			return nil
		}
		if readErr != nil {
			return scannerWalkError(readErr)
		}
	}
}

func (walker *ScanWalker) VerifyRoot(
	ctx context.Context,
	relativeRoot string,
	expected scanner.RootIdentity,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	walker.mu.Lock()
	identity, ok := walker.captured[relativeRoot]
	delete(walker.captured, relativeRoot)
	walker.mu.Unlock()
	if !ok {
		return scanner.ErrInvalidRootIdentity
	}
	device, inode, ok := identity.Key()
	if !ok || device != expected.Device || inode != expected.Inode {
		return scanner.ErrRootIdentityChanged
	}
	if err := walker.root.VerifyAt(relativeRoot, identity); err != nil {
		return scannerRootError(err)
	}
	return nil
}

func relativeToLibrary(root, walked string) (string, error) {
	if root == "" {
		return walked, nil
	}
	if walked == root {
		return "", nil
	}
	prefix := root + "/"
	if !strings.HasPrefix(walked, prefix) {
		return "", fmt.Errorf("%w: filesystem walker returned a path outside its library root", scanner.ErrInvalidEntry)
	}
	return strings.TrimPrefix(walked, prefix), nil
}

func isPolicySkip(err error) bool {
	return errors.Is(err, ErrInvalidPath) ||
		errors.Is(err, ErrSymlink) ||
		errors.Is(err, ErrSpecialFile)
}

func scannerRootError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrRootChanged) {
		return scanner.ErrRootIdentityChanged
	}
	if errors.Is(err, ErrOffline) {
		return scanner.ErrLibraryOffline
	}
	switch {
	case errors.Is(err, ErrSymlink):
		return scanner.ErrLibraryRootSymlink
	case errors.Is(err, ErrCrossDevice):
		return scanner.ErrLibraryMountBoundary
	default:
		return scanner.ErrScanIO
	}
}

func scannerWalkError(err error) error {
	if err == nil {
		return nil
	}
	for _, known := range []error{
		context.Canceled,
		context.DeadlineExceeded,
		scanner.ErrInvalidEntry,
		scanner.ErrRootIdentityChanged,
		scanner.ErrLibraryMountBoundary,
		scanner.ErrPartialTreeUnreadable,
		scanner.ErrScanIO,
	} {
		if errors.Is(err, known) {
			return known
		}
	}
	switch {
	case errors.Is(err, ErrRootChanged):
		return scanner.ErrRootIdentityChanged
	case errors.Is(err, ErrCrossDevice):
		return scanner.ErrLibraryMountBoundary
	case errors.Is(err, fs.ErrPermission), errors.Is(err, ErrOffline):
		return scanner.ErrPartialTreeUnreadable
	default:
		return scanner.ErrScanIO
	}
}
