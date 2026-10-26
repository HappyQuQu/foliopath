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
	Coverage               FaceCoverage
	GroupAssignmentAllowed bool
}
type FaceClusterPosition struct{ Role, ID string }
type FaceClusterListQuery struct {
	LibraryID          int64
	GenerationID, Role string
	After              *FaceClusterPosition
	Limit              int
}
type FaceClusterView struct {
	ID              string
	LibraryID       int64
	Role            string
	MemberCount     int64
	PreviewAssetIDs []int64
	Revision        int64
}
type FaceClusterListRepository interface {
	GetFaceClusterSnapshot(context.Context, int64) (FaceClusterSnapshot, error)
	ListFaceClusterViews(context.Context, FaceClusterListQuery) ([]FaceClusterView, error)
}
type FaceClusterListRequest struct {
	LibraryID    int64
	Role, Cursor string
	Limit        int
}
type FaceClusterPage struct {
	Items                  []FaceClusterView
	NextCursor             string
	Coverage               FaceCoverage
	GroupAssignmentAllowed bool
}
type faceClusterCursor struct {
	Version      int    `json:"v"`
	LibraryID    int64  `json:"l"`
	Role         string `json:"k"`
	GenerationID string `json:"g"`
	Snapshot     int64  `json:"r"`
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
	snapshot, err := service.repository.GetFaceClusterSnapshot(ctx, request.LibraryID)
	if err != nil {
		return FaceClusterPage{}, err
	}
	if snapshot.GenerationID == "" || snapshot.Revision < 1 {
		return FaceClusterPage{}, ErrFaceNotReady
	}
	var after *FaceClusterPosition
	if request.Cursor != "" {
		var value faceClusterCursor
		if len(request.Cursor) > 4096 || service.cursors.Decode(request.Cursor, "foliopath:face-clusters:v1", &value) != nil || value.Version != 1 || value.LibraryID != request.LibraryID || value.Role != request.Role || value.GenerationID != snapshot.GenerationID || !validReviewID(value.ID) {
			return FaceClusterPage{}, ErrInvalidFaceClusterCursor
		}
		if value.Snapshot != snapshot.Revision {
			return FaceClusterPage{}, ErrFaceClusterCursorStale
		}
		after = &FaceClusterPosition{Role: value.AfterRole, ID: value.ID}
	}
	items, err := service.repository.ListFaceClusterViews(ctx, FaceClusterListQuery{LibraryID: request.LibraryID, GenerationID: snapshot.GenerationID, Role: request.Role, After: after, Limit: request.Limit + 1})
	if err != nil {
		return FaceClusterPage{}, err
	}
	result := FaceClusterPage{Items: items, Coverage: snapshot.Coverage, GroupAssignmentAllowed: snapshot.GroupAssignmentAllowed}
	if len(result.Items) > request.Limit {
		result.Items = result.Items[:request.Limit]
		last := result.Items[len(result.Items)-1]
		result.NextCursor, err = service.cursors.Encode(faceClusterCursor{Version: 1, LibraryID: request.LibraryID, Role: request.Role, GenerationID: snapshot.GenerationID, Snapshot: snapshot.Revision, AfterRole: last.Role, ID: last.ID}, "foliopath:face-clusters:v1")
	}
	return result, err
}
