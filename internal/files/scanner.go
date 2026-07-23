package files

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"sync"

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
		return scanner.RootIdentity{}, scannerError(err)
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
				return err
			}
			return scannerError(walkErr)
		}

		info, err := entry.Info()
		if err != nil {
			return scannerError(err)
		}
		decision, err := visit(scanner.WalkEntry{
			RelativePath: libraryRelative,
			IsDirectory:  info.IsDir(),
			SizeBytes:    info.Size(),
			MTimeNS:      info.ModTime().UnixNano(),
		})
		if err != nil {
			return err
		}
		if decision == scanner.WalkSkipDirectory {
			return fs.SkipDir
		}
		return nil
	})
	return scannerError(err)
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
		return scannerError(err)
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
		errors.Is(err, ErrCrossDevice) ||
		errors.Is(err, ErrSpecialFile)
}

func scannerError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrRootChanged) {
		return scanner.ErrRootIdentityChanged
	}
	if errors.Is(err, ErrOffline) {
		return scanner.ErrLibraryOffline
	}
	return err
}
