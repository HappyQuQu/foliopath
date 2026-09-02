package face

import (
	"context"
	"time"
)

type ReconciliationResult struct{ Bound, NeedsReview int64 }

type ReconciliationRepository interface {
	ReconcileFaceAnchors(context.Context, string, int64, time.Time) (ReconciliationResult, error)
}
