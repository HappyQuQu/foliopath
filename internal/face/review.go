package face

import (
	"context"
	"errors"
	"time"
)

var (
	ErrInvalidReview  = errors.New("invalid face review")
	ErrReviewConflict = errors.New("face review conflict")
)

type ReviewResult struct {
	EventID           string
	Action            string
	AffectedPersonIDs []string
	Revision          int64
	Undoable          bool
	Replayed          bool
}

type AssignFaceCommand struct {
	EventID, RequestHash, AnchorID, FaceID, PersonID string
	ExpectedFaceRevision, ExpectedPersonRevision     int64
	CreatedAt                                        time.Time
}

type AssignClusterCommand struct {
	EventID, RequestHash, ClusterID, PersonID       string
	AnchorIDs                                       []string
	ExpectedClusterRevision, ExpectedPersonRevision int64
	CreatedAt                                       time.Time
}

type CreatePersonFromClusterCommand struct {
	EventID, RequestHash, PersonID, Name, ClusterID string
	AnchorIDs                                       []string
	ExpectedClusterRevision                         int64
	CreatedAt                                       time.Time
}

type ExcludeFaceCommand struct {
	EventID, RequestHash, ExclusionID, FaceID string
	ExpectedFaceRevision                      int64
	CreatedAt                                 time.Time
}

type CannotLinkCommand struct {
	EventID, RequestHash, LeftFaceID, RightFaceID string
	ExpectedLeftRevision, ExpectedRightRevision   int64
	CreatedAt                                     time.Time
}

type MergePeopleCommand struct {
	EventID, RequestHash, SourcePersonID, TargetPersonID string
	ExpectedSourceRevision, ExpectedTargetRevision       int64
	ConflictsAcknowledged                                bool
	CreatedAt                                            time.Time
}

type SplitFaceCommand struct {
	EventID, RequestHash, FaceID, SourcePersonID string
	ExpectedFaceRevision, ExpectedSourceRevision int64
	CreatedAt                                    time.Time
}

type UndoReviewCommand struct {
	EventID, RequestHash, ReviewID string
	ExpectedRevision               int64
	CreatedAt                      time.Time
}

type ReviewRepository interface {
	CreatePersonFromCluster(context.Context, CreatePersonFromClusterCommand) (ReviewResult, error)
	AssignFace(context.Context, AssignFaceCommand) (ReviewResult, error)
	AssignCluster(context.Context, AssignClusterCommand) (ReviewResult, error)
	ExcludeFace(context.Context, ExcludeFaceCommand) (ReviewResult, error)
	CannotLinkFaces(context.Context, CannotLinkCommand) (ReviewResult, error)
	MergePeople(context.Context, MergePeopleCommand) (ReviewResult, error)
	SplitFace(context.Context, SplitFaceCommand) (ReviewResult, error)
	UndoFaceReview(context.Context, UndoReviewCommand) (ReviewResult, error)
}

func validReviewID(value string) bool { return len(value) >= 8 && len(value) <= 128 }
