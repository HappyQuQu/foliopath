package face

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"
)

const MaxGroupReviewFaces = 100

type CoreClusterFaceRepository interface {
	ListCoreClusterFaceIDs(context.Context, string, int64, int) ([]string, error)
}

type ReviewRequest struct {
	Action                                         string
	FaceID, PersonID, ClusterID                    string
	LeftFaceID, RightFaceID                        string
	SourcePersonID, TargetPersonID                 string
	ExpectedFaceRevision, ExpectedPersonRevision   int64
	ExpectedClusterRevision                        int64
	ExpectedLeftRevision, ExpectedRightRevision    int64
	ExpectedSourceRevision, ExpectedTargetRevision int64
	ConflictsAcknowledged                          bool
}

type ReviewService struct {
	repository ReviewRepository
	clusters   CoreClusterFaceRepository
	now        func() time.Time
}

func NewReviewService(repository ReviewRepository, clusters CoreClusterFaceRepository, now func() time.Time) (*ReviewService, error) {
	if repository == nil || clusters == nil {
		return nil, errors.New("face review dependencies are required")
	}
	if now == nil {
		now = time.Now
	}
	return &ReviewService{repository: repository, clusters: clusters, now: now}, nil
}

func (service *ReviewService) Review(ctx context.Context, idempotencyKey string, request ReviewRequest) (ReviewResult, error) {
	eventID, requestHash, err := ReviewIdentity(idempotencyKey, request)
	if err != nil {
		return ReviewResult{}, err
	}
	createdAt := service.now().UTC()
	if createdAt.IsZero() {
		return ReviewResult{}, ErrInvalidReview
	}
	switch request.Action {
	case "assign_face":
		return service.repository.AssignFace(ctx, AssignFaceCommand{
			EventID: eventID, RequestHash: requestHash, AnchorID: derivedReviewResourceID("fanchor", eventID, request.FaceID),
			FaceID: request.FaceID, PersonID: request.PersonID, ExpectedFaceRevision: request.ExpectedFaceRevision,
			ExpectedPersonRevision: request.ExpectedPersonRevision, CreatedAt: createdAt,
		})
	case "assign_cluster":
		faceIDs, err := service.clusters.ListCoreClusterFaceIDs(ctx, request.ClusterID, request.ExpectedClusterRevision, MaxGroupReviewFaces+1)
		if err != nil {
			return ReviewResult{}, err
		}
		if len(faceIDs) < 2 || len(faceIDs) > MaxGroupReviewFaces {
			return ReviewResult{}, ErrReviewConflict
		}
		anchors := make([]string, len(faceIDs))
		for index, faceID := range faceIDs {
			anchors[index] = derivedReviewResourceID("fanchor", eventID, faceID)
		}
		return service.repository.AssignCluster(ctx, AssignClusterCommand{
			EventID: eventID, RequestHash: requestHash, ClusterID: request.ClusterID, PersonID: request.PersonID,
			AnchorIDs: anchors, ExpectedClusterRevision: request.ExpectedClusterRevision,
			ExpectedPersonRevision: request.ExpectedPersonRevision, CreatedAt: createdAt,
		})
	case "exclude_face":
		return service.repository.ExcludeFace(ctx, ExcludeFaceCommand{
			EventID: eventID, RequestHash: requestHash, ExclusionID: derivedReviewResourceID("fexclude", eventID, request.FaceID),
			FaceID: request.FaceID, ExpectedFaceRevision: request.ExpectedFaceRevision, CreatedAt: createdAt,
		})
	case "cannot_link":
		return service.repository.CannotLinkFaces(ctx, CannotLinkCommand{
			EventID: eventID, RequestHash: requestHash, LeftFaceID: request.LeftFaceID, RightFaceID: request.RightFaceID,
			ExpectedLeftRevision: request.ExpectedLeftRevision, ExpectedRightRevision: request.ExpectedRightRevision, CreatedAt: createdAt,
		})
	case "merge_people":
		return service.repository.MergePeople(ctx, MergePeopleCommand{
			EventID: eventID, RequestHash: requestHash, SourcePersonID: request.SourcePersonID, TargetPersonID: request.TargetPersonID,
			ExpectedSourceRevision: request.ExpectedSourceRevision, ExpectedTargetRevision: request.ExpectedTargetRevision,
			ConflictsAcknowledged: request.ConflictsAcknowledged, CreatedAt: createdAt,
		})
	case "split_face":
		return service.repository.SplitFace(ctx, SplitFaceCommand{
			EventID: eventID, RequestHash: requestHash, FaceID: request.FaceID, SourcePersonID: request.SourcePersonID,
			ExpectedFaceRevision: request.ExpectedFaceRevision, ExpectedSourceRevision: request.ExpectedSourceRevision, CreatedAt: createdAt,
		})
	default:
		return ReviewResult{}, ErrInvalidReview
	}
}

func (service *ReviewService) Undo(ctx context.Context, idempotencyKey, reviewID string, expectedRevision int64) (ReviewResult, error) {
	request := struct {
		Action           string `json:"action"`
		ReviewID         string `json:"reviewId"`
		ExpectedRevision int64  `json:"expectedRevision"`
	}{"undo", reviewID, expectedRevision}
	eventID, requestHash, err := ReviewIdentity(idempotencyKey, request)
	if err != nil {
		return ReviewResult{}, err
	}
	return service.repository.UndoFaceReview(ctx, UndoReviewCommand{EventID: eventID, RequestHash: requestHash, ReviewID: reviewID, ExpectedRevision: expectedRevision, CreatedAt: service.now().UTC()})
}

func derivedReviewResourceID(prefix, eventID, targetID string) string {
	digest := sha256.Sum256([]byte("foliopath:face-review-resource:v1\x00" + prefix + "\x00" + eventID + "\x00" + targetID))
	return prefix + "_" + hex.EncodeToString(digest[:20])
}
