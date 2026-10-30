package files

import (
	"context"
	"errors"
	"io/fs"
	"sync"

	"github.com/HappyQuQu/foliopath/internal/deletion"
)

func (root *Root) InspectMutationSource(relative string) (deletion.SourceIdentity, error) {
	normalized, err := Normalize(relative)
	if err != nil {
		return deletion.SourceIdentity{}, deletion.ErrInvalid
	}
	root.mu.RLock()
	defer root.mu.RUnlock()
	if root.closed {
		return deletion.SourceIdentity{}, deletion.ErrWriteUnavailable
	}
	if _, err := root.captureLocked(); err != nil {
		return deletion.SourceIdentity{}, deletion.ErrSourceIO
	}
	info, err := root.anchor.Lstat(normalized)
	if err != nil {
		return deletion.SourceIdentity{}, mutationSourceError(err)
	}
	if !info.IsDir() && !info.Mode().IsRegular() {
		return deletion.SourceIdentity{}, deletion.ErrSourceChanged
	}
	if _, err := root.captureLocked(); err != nil {
		return deletion.SourceIdentity{}, deletion.ErrSourceIO
	}
	device, inode, ok := platformIdentity(info)
	if !ok {
		return deletion.SourceIdentity{}, deletion.ErrWriteUnavailable
	}
	return deletion.SourceIdentity{Device: device, Inode: inode, Size: info.Size(), ModifiedNS: info.ModTime().UnixNano(), Directory: info.IsDir()}, nil
}

func mutationSourceError(err error) error {
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return deletion.ErrSourceMissing
	case errors.Is(err, fs.ErrPermission):
		return deletion.ErrPermission
	case errors.Is(err, ErrChanged), errors.Is(err, ErrSymlink):
		return deletion.ErrSourceChanged
	case errors.Is(err, ErrKernelBoundaryUnavailable), errors.Is(err, ErrCrossDevice):
		return deletion.ErrWriteUnavailable
	default:
		return deletion.ErrSourceIO
	}
}

type mutationRequest struct {
	Kind     string
	Path     string
	Instance string
	Entry    string
	Expected deletion.SourceIdentity
}

type mutationReply struct {
	Identity deletion.SourceIdentity
	Removed  int64
	Done     bool
}

type MediaMutator struct {
	mu     sync.Mutex
	state  *mutationState
	closed bool
}

func NewMediaMutator(ctx context.Context, root *Root) (*MediaMutator, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	state, err := newMutationState(root)
	if err != nil {
		return nil, err
	}
	return &MediaMutator{state: state}, nil
}

func (client *MediaMutator) Inspect(ctx context.Context, relative string) (deletion.SourceIdentity, error) {
	reply, err := client.call(ctx, mutationRequest{Kind: "inspect", Path: relative})
	return reply.Identity, err
}

func (client *MediaMutator) BindLibrary(ctx context.Context, relative string, expected deletion.SourceIdentity) error {
	_, err := client.call(ctx, mutationRequest{Kind: "bind", Path: relative, Expected: expected})
	return err
}

func (client *MediaMutator) InspectTrash(ctx context.Context, instance, entry string) (deletion.SourceIdentity, error) {
	reply, err := client.call(ctx, mutationRequest{Kind: "inspect_trash", Instance: instance, Entry: entry})
	return reply.Identity, err
}

func (client *MediaMutator) PrepareTrash(ctx context.Context, instance string) error {
	_, err := client.call(ctx, mutationRequest{Kind: "prepare", Instance: instance})
	return err
}

func (client *MediaMutator) Trash(ctx context.Context, relative, instance, entry string, expected deletion.SourceIdentity) error {
	_, err := client.call(ctx, mutationRequest{Kind: "trash", Path: relative, Instance: instance, Entry: entry, Expected: expected})
	return err
}

func (client *MediaMutator) Restore(ctx context.Context, relative, instance, entry string, expected deletion.SourceIdentity) error {
	_, err := client.call(ctx, mutationRequest{Kind: "restore", Path: relative, Instance: instance, Entry: entry, Expected: expected})
	return err
}

func (client *MediaMutator) RemoveBatch(ctx context.Context, relative string, expected deletion.SourceIdentity) (deletion.RemovalProgress, error) {
	reply, err := client.call(ctx, mutationRequest{Kind: "remove", Path: relative, Expected: expected})
	return deletion.RemovalProgress{Removed: reply.Removed, Done: reply.Done}, err
}

func (client *MediaMutator) PurgeBatch(ctx context.Context, instance, entry string, expected deletion.SourceIdentity) (deletion.RemovalProgress, error) {
	reply, err := client.call(ctx, mutationRequest{Kind: "purge", Instance: instance, Entry: entry, Expected: expected})
	return deletion.RemovalProgress{Removed: reply.Removed, Done: reply.Done}, err
}

func (client *MediaMutator) Close() error {
	client.mu.Lock()
	defer client.mu.Unlock()
	if client.closed {
		return nil
	}
	client.closed = true
	return client.state.close()
}

func (client *MediaMutator) call(ctx context.Context, request mutationRequest) (mutationReply, error) {
	client.mu.Lock()
	defer client.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return mutationReply{}, err
	}
	if client.closed {
		return mutationReply{}, deletion.ErrWriteUnavailable
	}
	if err := validateMutationRequest(request); err != nil {
		return mutationReply{}, err
	}
	reply, err := client.state.execute(ctx, request)
	if err != nil {
		client.state.closeRemoval()
		if ctx.Err() != nil {
			return reply, ctx.Err()
		}
		return reply, mutationOperationError(err)
	}
	return reply, nil
}
