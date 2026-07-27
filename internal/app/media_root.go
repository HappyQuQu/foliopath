package app

import (
	"context"
	"errors"
	"io/fs"
	"path"
	"sync"

	"github.com/HappyQuQu/foliopath/internal/files"
	"github.com/HappyQuQu/foliopath/internal/library"
	"github.com/HappyQuQu/foliopath/internal/pathpolicy"
	"github.com/HappyQuQu/foliopath/internal/scanner"
	"github.com/HappyQuQu/foliopath/internal/thumbnail"
)

// mediaRootService owns the allowed media root lifecycle and implements the
// library capability's narrow directory-source port. It performs no direct
// filesystem I/O; every operation delegates to internal/files.
type mediaRootService struct {
	path string

	mutex  sync.RWMutex
	root   *files.Root
	source *files.DirectorySource
	walker *files.ScanWalker
}

func (service *mediaRootService) OpenAsset(
	ctx context.Context,
	libraryRoot, relativePath string,
) (thumbnail.SourceFile, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	normalizedRoot, err := pathpolicy.Normalize(libraryRoot)
	if err != nil || normalizedRoot != libraryRoot {
		return nil, thumbnail.ErrSourceUnavailable
	}
	normalizedAsset, err := pathpolicy.Normalize(relativePath)
	if err != nil || normalizedAsset == "" || normalizedAsset != relativePath {
		return nil, thumbnail.ErrSourceUnavailable
	}
	target := normalizedAsset
	if normalizedRoot != "" {
		target = path.Join(normalizedRoot, normalizedAsset)
	}
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.root == nil {
		return nil, thumbnail.ErrSourceUnavailable
	}
	file, err := service.root.Open(target)
	if err != nil {
		return nil, thumbnail.ErrSourceUnavailable
	}
	return file, nil
}

func newMediaRootService(path string) (*mediaRootService, component, error) {
	if path == "" {
		return nil, component{}, errors.New("allowed media root path is required")
	}
	service := &mediaRootService{path: path}
	return service, component{
		name:  "media-root",
		start: service.start,
		stop:  service.stop,
	}, nil
}

func (service *mediaRootService) start(context.Context) error {
	root, err := files.OpenRoot(service.path)
	if err != nil {
		return err
	}
	source, err := files.NewDirectorySource(root)
	if err != nil {
		_ = root.Close()
		return err
	}
	walker, err := files.NewScanWalker(root)
	if err != nil {
		_ = root.Close()
		return err
	}

	service.mutex.Lock()
	defer service.mutex.Unlock()
	if service.root != nil {
		_ = root.Close()
		return errors.New("allowed media root is already started")
	}
	service.root = root
	service.source = source
	service.walker = walker
	return nil
}

func (service *mediaRootService) stop(context.Context) error {
	service.mutex.Lock()
	root := service.root
	service.root = nil
	service.source = nil
	service.walker = nil
	service.mutex.Unlock()
	if root == nil {
		return nil
	}
	return root.Close()
}

func (service *mediaRootService) CaptureRoot(
	ctx context.Context,
	relative string,
) (scanner.RootIdentity, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.walker == nil {
		return scanner.RootIdentity{}, scanner.ErrLibraryOffline
	}
	return service.walker.CaptureRoot(ctx, relative)
}

func (service *mediaRootService) Walk(
	ctx context.Context,
	relative string,
	visit func(scanner.WalkEntry) (scanner.WalkDecision, error),
) error {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.walker == nil {
		return scanner.ErrLibraryOffline
	}
	return service.walker.Walk(ctx, relative, visit)
}

func (service *mediaRootService) VerifyRoot(
	ctx context.Context,
	relative string,
	identity scanner.RootIdentity,
) error {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.walker == nil {
		return scanner.ErrLibraryOffline
	}
	return service.walker.VerifyRoot(ctx, relative, identity)
}

func (service *mediaRootService) EnumerateDirectories(
	ctx context.Context,
	parent string,
	visit func(library.DirectoryCandidate) error,
) error {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.source == nil {
		return library.ErrParentUnavailable
	}
	return service.source.EnumerateDirectories(ctx, parent, visit)
}

func (service *mediaRootService) ValidateLibraryRoot(
	ctx context.Context,
	relative string,
) error {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.source == nil {
		return library.ErrRootUnavailable
	}
	err := service.source.ValidateLibraryRoot(ctx, relative)
	switch {
	case err == nil:
		return nil
	case errors.Is(err, files.ErrInvalidPath):
		return library.ErrRootOutsideAllowed
	case errors.Is(err, files.ErrSymlink):
		return library.ErrRootSymlink
	case errors.Is(err, files.ErrCrossDevice),
		errors.Is(err, files.ErrKernelBoundaryUnavailable):
		return library.ErrRootMountBoundary
	case errors.Is(err, files.ErrOffline),
		errors.Is(err, files.ErrRootChanged),
		errors.Is(err, files.ErrNotDirectory),
		errors.Is(err, fs.ErrNotExist),
		errors.Is(err, fs.ErrPermission):
		return library.ErrRootUnavailable
	default:
		return err
	}
}
