package face

import (
	"context"
	"errors"

	cursorcodec "github.com/HappyQuQu/foliopath/internal/cursor"
)

const MaxFaceClusterPageSize = 200

var (
	ErrInvalidFaceClusterCursor = errors.New("invalid face cluster cursor")
	ErrFaceClusterCursorStale   = errors.New("face cluster cursor stale")
	ErrFaceNotReady             = errors.New("face analysis not ready")
)

type FaceCoverage struct{ Eligible, Completed, Failed, Stale, Revision int64 }

func (c FaceCoverage) Complete() bool { return c.Eligible == c.Completed+c.Failed+c.Stale }

type FaceClusterSnapshot struct {
	LibraryID              int64
	GenerationID           string
	Revision               int64
	Digest                 string
	Coverage               FaceCoverage
	GroupAssignmentAllowed bool
}
type FaceClusterPosition struct{ Role, ID string }
type FaceClusterListQuery struct {
	LibraryID    int64
	Role, Review string
	After        *FaceClusterPosition
	Limit        int
}
type FaceClusterView struct {
	ID                 string
	LibraryID          int64
	Role               string
	MemberCount        int64
	PendingMemberCount int64
	PreviewAssetIDs    []int64
	Revision           int64
}

// ReadFaceClusterPage returns metadata, counts and bounded items from one read snapshot.
type FaceClusterListRepository interface {
	ReadFaceClusterPage(context.Context, FaceClusterListQuery) (FaceClusterReadPage, error)
}
type FaceClusterCounts struct{ TotalGroups, PendingGroups, TotalFaces, PendingFaces int64 }
type FaceClusterReadPage struct {
	Snapshot FaceClusterSnapshot
	Counts   FaceClusterCounts
	Items    []FaceClusterView
}
type FaceClusterListRequest struct {
	LibraryID            int64
	Role, Review, Cursor string
	Limit                int
}
type FaceClusterPage struct {
	Items                  []FaceClusterView
	Counts                 FaceClusterCounts
	NextCursor             string
	Coverage               FaceCoverage
	GroupAssignmentAllowed bool
}
type faceClusterCursor struct {
	Version      int    `json:"v"`
	LibraryID    int64  `json:"l"`
	Role         string `json:"k"`
	GenerationID string `json:"g"`
	Snapshot     string `json:"r"`
	Review       string `json:"f"`
	AfterRole    string `json:"a"`
	ID           string `json:"i"`
}
type FaceClusterListService struct {
	repository FaceClusterListRepository
	cursors    *cursorcodec.Codec
}

func NewFaceClusterListService(repository FaceClusterListRepository, key []byte) (*FaceClusterListService, error) {
	if repository == nil {
		return nil, errors.New("face cluster list repository is required")
	}
	codec, err := cursorcodec.New(key)
	if err != nil {
		return nil, err
	}
	return &FaceClusterListService{repository: repository, cursors: codec}, nil
}
func (service *FaceClusterListService) List(ctx context.Context, request FaceClusterListRequest) (FaceClusterPage, error) {
	if request.LibraryID < 1 || (request.Role != "" && request.Role != "core" && request.Role != "edge") || request.Limit < 1 || request.Limit > MaxFaceClusterPageSize {
		return FaceClusterPage{}, ErrInvalidClusterRecord
	}
	if request.Review == "" {
		request.Review = "all"
	}
	if request.Review != "all" && request.Review != "pending" {
		return FaceClusterPage{}, ErrInvalidClusterRecord
	}
	var after *FaceClusterPosition
	var value faceClusterCursor
	if request.Cursor != "" {
		if len(request.Cursor) > 4096 || service.cursors.Decode(request.Cursor, "foliopath:face-clusters:v2", &value) != nil || value.Version != 2 || value.LibraryID != request.LibraryID || value.Role != request.Role || value.Review != request.Review || !validReviewID(value.ID) || (value.AfterRole != "core" && value.AfterRole != "edge") {
			return FaceClusterPage{}, ErrInvalidFaceClusterCursor
		}
		after = &FaceClusterPosition{Role: value.AfterRole, ID: value.ID}
	}
	page, err := service.repository.ReadFaceClusterPage(ctx, FaceClusterListQuery{LibraryID: request.LibraryID, Role: request.Role, Review: request.Review, After: after, Limit: request.Limit + 1})
	if err != nil {
		return FaceClusterPage{}, err
	}
	snapshot := page.Snapshot
	if snapshot.GenerationID == "" || snapshot.Revision < 1 || snapshot.Digest == "" {
		return FaceClusterPage{}, ErrFaceNotReady
	}
	if request.Cursor != "" && (value.GenerationID != snapshot.GenerationID || value.Snapshot != snapshot.Digest) {
		return FaceClusterPage{}, ErrFaceClusterCursorStale
	}
	result := FaceClusterPage{Items: page.Items, Counts: page.Counts, Coverage: snapshot.Coverage, GroupAssignmentAllowed: snapshot.GroupAssignmentAllowed}
	if len(result.Items) > request.Limit {
		result.Items = result.Items[:request.Limit]
		last := result.Items[len(result.Items)-1]
		result.NextCursor, err = service.cursors.Encode(faceClusterCursor{Version: 2, LibraryID: request.LibraryID, Role: request.Role, Review: request.Review, GenerationID: snapshot.GenerationID, Snapshot: snapshot.Digest, AfterRole: last.Role, ID: last.ID}, "foliopath:face-clusters:v2")
	}
	return result, err
}
