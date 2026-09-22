package face

import "context"

// GroupAssignmentAllowed stays fail-closed until an accepted quality-evidence
// contract supplies independently verified precision. A profile hash is not proof.
func GroupAssignmentAllowed() bool { return false }

// UncalibratedGroupReview is the production group-review boundary. Individual
// confirmed face assignments remain available through ReviewService.
type UncalibratedGroupReview struct{}

func (UncalibratedGroupReview) ListCoreClusterFaceIDs(context.Context, string, int64, int) ([]string, error) {
	return nil, ErrReviewConflict
}
