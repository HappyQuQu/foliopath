package app

import (
	"context"
	"errors"
	"sync"

	"github.com/HappyQuQu/foliopath/internal/deletion"
	"github.com/HappyQuQu/foliopath/internal/files"
	"github.com/HappyQuQu/foliopath/internal/resourcecontrol"
)

type mediaMutationSource struct {
	media      *mediaRootService
	resources  *resourcecontrol.Controller
	configured bool
	probeMutex sync.Mutex
	probed     bool
	supported  bool
}

func newMediaMutationSource(media *mediaRootService, resources *resourcecontrol.Controller, configured bool) (*mediaMutationSource, error) {
	return &mediaMutationSource{media: media, resources: resources, configured: configured}, nil
}

func (source *mediaMutationSource) MutationAvailability(ctx context.Context) (bool, string) {
	if !source.configured {
		return false, "not_configured"
	}
	source.media.mutex.RLock()
	defer source.media.mutex.RUnlock()
	root := source.media.root
	if root == nil {
		return false, "storage_unavailable"
	}
	readOnly, err := root.ReadOnly()
	if err != nil {
		if errors.Is(err, files.ErrKernelBoundaryUnavailable) {
			return false, "unsupported"
		}
		return false, "storage_unavailable"
	}
	if readOnly {
		return false, "read_only"
	}
	source.probeMutex.Lock()
	defer source.probeMutex.Unlock()
	if !source.probed {
		release, ok := source.resources.TryAcquireMediaMutation()
		if !ok {
			return false, "storage_unavailable"
		}
		mutator, err := files.NewMediaMutator(ctx, root)
		if err == nil {
			_ = mutator.Close()
		}
		release()
		source.probed, source.supported = true, err == nil
	}
	if !source.supported {
		return false, "unsupported"
	}
	return true, ""
}

func (source *mediaMutationSource) OpenMutation(ctx context.Context) (deletion.Mutator, error) {
	available, _ := source.MutationAvailability(ctx)
	if !available {
		return nil, deletion.ErrWriteUnavailable
	}
	release, err := source.resources.AcquireMediaMutation(ctx)
	if err != nil {
		return nil, err
	}
	source.media.mutex.RLock()
	if source.media.root == nil {
		source.media.mutex.RUnlock()
		release()
		return nil, deletion.ErrSourceIO
	}
	mutator, err := files.NewMediaMutator(ctx, source.media.root)
	source.media.mutex.RUnlock()
	if err != nil {
		release()
		return nil, err
	}
	return &leasedMediaMutator{Mutator: mutator, release: release}, nil
}

func (source *mediaMutationSource) InspectMutationSource(ctx context.Context, relative string) (deletion.SourceIdentity, error) {
	if err := ctx.Err(); err != nil {
		return deletion.SourceIdentity{}, err
	}
	source.media.mutex.RLock()
	defer source.media.mutex.RUnlock()
	if source.media.root == nil {
		return deletion.SourceIdentity{}, deletion.ErrSourceIO
	}
	return source.media.root.InspectMutationSource(relative)
}

type leasedMediaMutator struct {
	deletion.Mutator
	release func()
	once    sync.Once
}

func (mutator *leasedMediaMutator) Close() error {
	var err error
	mutator.once.Do(func() { err = mutator.Mutator.Close(); mutator.release() })
	return err
}
