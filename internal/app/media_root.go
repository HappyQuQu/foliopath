package app

import (
	"context"
	"errors"
	"io/fs"
	"sync"

	"github.com/HappyQuQu/foliopath/internal/files"
	"github.com/HappyQuQu/foliopath/internal/library"
)

// mediaRootService owns the allowed media root lifecycle and implements the
// library capability's narrow directory-source port. It performs no direct
// filesystem I/O; every operation delegates to internal/files.
type mediaRootService struct {
	path string

	mutex  sync.RWMutex
	root   *files.Root
	source *files.DirectorySource
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

	service.mutex.Lock()
	defer service.mutex.Unlock()
	if service.root != nil {
		_ = root.Close()
		return errors.New("allowed media root is already started")
	}
	service.root = root
	service.source = source
	return nil
}

func (service *mediaRootService) stop(context.Context) error {
	service.mutex.Lock()
	root := service.root
	service.root = nil
	service.source = nil
	service.mutex.Unlock()
	if root == nil {
		return nil
	}
	return root.Close()
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
