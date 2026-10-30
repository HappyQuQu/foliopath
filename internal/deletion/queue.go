package deletion

import (
	"context"
	"errors"
	"time"

	"github.com/HappyQuQu/foliopath/internal/jobs"
)

type Reconciler interface {
	Request(context.Context, int64) (int64, error)
	Status(context.Context, int64) (bool, bool, error)
}

// MaintainedQueue schedules read-only repair after media mutation has stopped.
// It never holds a background processing slot while waiting for a scan.
type MaintainedQueue struct {
	Repository Queue
	Source     MutationSource
	Reconciler Reconciler
}

func (q MaintainedQueue) Handoff(ctx context.Context, c Claim) error {
	return q.Repository.HandoffDeletion(ctx, c)
}

func (q MaintainedQueue) Claim(ctx context.Context, lease time.Duration) (Claim, bool, error) {
	if available, _ := q.Source.MutationAvailability(ctx); !available {
		return Claim{}, false, nil
	}
	return q.Repository.Claim(ctx, lease)
}
func (q MaintainedQueue) RefreshLease(ctx context.Context, c Claim, lease time.Duration) (bool, error) {
	return q.Repository.RefreshLease(ctx, c, lease)
}

func (q MaintainedQueue) RecoverExpired(ctx context.Context) (jobs.RecoverySummary, error) {
	result, err := q.Repository.RecoverExpired(ctx)
	if err != nil {
		return result, err
	}

	repairs, err := q.Repository.ListDeletionRepairs(ctx, MaxActiveOperations)
	if err != nil {
		return result, err
	}
	for _, repair := range repairs {
		identity, err := q.Source.InspectMutationSource(ctx, repair.Root)
		if ctx.Err() != nil {
			return result, ctx.Err()
		}
		if err != nil || !repair.Identity.Matches(identity) {
			if err := q.Repository.CompleteDeletionRepair(ctx, repair, false); err != nil {
				return result, err
			}
			continue
		}
		if repair.ScanID == 0 {
			id, err := q.Reconciler.Request(ctx, repair.LibraryID)
			if errors.Is(err, ErrBusy) {
				continue
			}
			if err != nil {
				return result, err
			}
			if err := q.Repository.AssignDeletionRepair(ctx, repair, id); err != nil {
				return result, err
			}
			continue
		}
		done, success, err := q.Reconciler.Status(ctx, repair.ScanID)
		if err != nil {
			return result, err
		}
		if !done {
			continue
		}
		current, sourceErr := q.Source.InspectMutationSource(ctx, repair.Root)
		if err := q.Repository.CompleteDeletionRepair(ctx, repair, success && sourceErr == nil && repair.Identity.Matches(current)); err != nil {
			return result, err
		}
	}
	return result, nil
}
