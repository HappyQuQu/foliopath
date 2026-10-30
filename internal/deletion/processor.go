package deletion

import (
	"context"
	"errors"
	"path"
	"time"
)

type Processor struct {
	Repository Queue
	Source     MutationSource
}

func (p Processor) BackgroundAdmissionCancelled(_ context.Context, c Claim) {
	_ = p.finish(c, "stopped", nil)
}

func (p Processor) Process(ctx context.Context, claim Claim) error {
	mutator, err := p.Source.OpenMutation(ctx)
	if err != nil {
		if ctx.Err() != nil {
			return p.finish(claim, "stopped", err)
		}
		return p.finish(claim, "failed", err)
	}
	state, err := p.process(ctx, mutator, claim)
	_ = mutator.Close()
	return p.finish(claim, state, err)
}

func (p Processor) process(ctx context.Context, m Mutator, c Claim) (string, error) {
	for {
		item, found, err := p.Repository.NextDeletionItem(ctx, c)
		if err != nil {
			return "stopped", err
		}
		if !found {
			return "succeeded", nil
		}
		if err := m.BindLibrary(ctx, item.Root, item.LibraryIdentity); err != nil {
			if err := p.Repository.RecordDeletionFilesystem(ctx, c, item, 0, false, item.Stage == "queued", deletionErrorCode(err)); err != nil {
				return "needs_attention", err
			}
			continue
		}
		if err := p.Repository.BeginDeletionItem(ctx, c, item); err != nil {
			if errors.Is(err, ErrConflict) || errors.Is(err, ErrSourceMissing) {
				if err := p.Repository.RecordDeletionFilesystem(ctx, c, item, 0, false, true, deletionErrorCode(err)); err != nil {
					return "needs_attention", err
				}
				continue
			}
			return "stopped", err
		}
		ready := item.Stage == "filesystem_applied" || item.Stage == "catalog_finalizing"
		if !ready {
			ready, err = p.apply(ctx, m, c, item)
			if err != nil {
				return "stopped", err
			}
		}
		if !ready {
			continue
		}
		for {
			done, err := p.Repository.FinalizeDeletionItem(ctx, c, item)
			if err != nil {
				return "needs_attention", err
			}
			if done {
				break
			}
		}
	}
}

func (p Processor) apply(ctx context.Context, m Mutator, c Claim, item OperationItem) (bool, error) {
	target := path.Join(item.Root, item.RelativePath)
	var source, trash SourceIdentity
	var sourceErr, trashErr error
	if c.Kind != "purge" {
		source, sourceErr = m.Inspect(ctx, target)
	}
	if item.TrashToken != "" {
		trash, trashErr = m.InspectTrash(ctx, c.Instance, item.TrashToken)
	}
	if c.Kind == "trash" || c.Kind == "restore" {
		applied := c.Kind == "trash" && errors.Is(sourceErr, ErrSourceMissing) && trashErr == nil && item.Identity.Matches(trash) || c.Kind == "restore" && errors.Is(trashErr, ErrSourceMissing) && sourceErr == nil && item.Identity.Matches(source)
		if applied {
			return p.record(ctx, c, item, RemovalProgress{Done: true}, nil)
		}
		if c.Kind == "restore" {
			if err := p.Repository.CheckDeletionRestore(ctx, c, item); err != nil {
				return false, p.Repository.RecordDeletionFilesystem(ctx, c, item, 0, false, true, deletionErrorCode(err))
			}
		}
		if c.Kind == "trash" {
			sourceErr = m.Trash(ctx, target, c.Instance, item.TrashToken, item.Identity)
		} else {
			sourceErr = m.Restore(ctx, target, c.Instance, item.TrashToken, item.Identity)
		}
		if sourceErr != nil {
			original, originalErr := m.Inspect(ctx, target)
			recycled, recycledErr := m.InspectTrash(ctx, c.Instance, item.TrashToken)
			unchanged := c.Kind == "trash" && originalErr == nil && item.Identity.Matches(original) && errors.Is(recycledErr, ErrSourceMissing) || c.Kind == "restore" && recycledErr == nil && item.Identity.Matches(recycled)
			return false, p.Repository.RecordDeletionFilesystem(ctx, c, item, 0, false, unchanged, deletionErrorCode(sourceErr))
		}
		return p.record(ctx, c, item, RemovalProgress{Done: true}, nil)
	}
	if c.Kind == "purge" {
		sourceErr = trashErr
	}
	if errors.Is(sourceErr, ErrSourceMissing) {
		return p.record(ctx, c, item, RemovalProgress{Done: true}, nil)
	}
	if sourceErr != nil {
		return p.record(ctx, c, item, RemovalProgress{}, sourceErr)
	}
	for {
		if err := p.Repository.CheckDeletionPermission(ctx, c); err != nil {
			return false, err
		}
		var progress RemovalProgress
		var err error
		if c.Kind == "purge" {
			progress, err = m.PurgeBatch(ctx, c.Instance, item.TrashToken, item.Identity)
		} else {
			progress, err = m.RemoveBatch(ctx, target, item.Identity)
		}
		ready, recordErr := p.record(ctx, c, item, progress, err)
		if recordErr != nil || err != nil || progress.Done {
			return ready, recordErr
		}
	}
}

func (p Processor) record(ctx context.Context, c Claim, item OperationItem, progress RemovalProgress, cause error) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	err := p.Repository.RecordDeletionFilesystem(ctx, c, item, progress.Removed, progress.Done, false, deletionErrorCode(cause))
	return progress.Done && cause == nil, err
}

func deletionErrorCode(cause error) string {
	switch {
	case cause == nil:
		return ""
	case errors.Is(cause, ErrSourceChanged):
		return "source_changed"
	case errors.Is(cause, ErrSourceMissing):
		return "source_missing"
	case errors.Is(cause, ErrConflict):
		return "destination_exists"
	case errors.Is(cause, ErrPermission):
		return "permission_denied"
	case errors.Is(cause, ErrWriteUnavailable):
		return "write_unavailable"
	default:
		return "source_io"
	}
}

func (p Processor) finish(c Claim, state string, cause error) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return errors.Join(cause, p.Repository.FinishDeletionClaim(ctx, c, state))
}
