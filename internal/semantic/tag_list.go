package semantic

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	cursorcodec "github.com/HappyQuQu/foliopath/internal/cursor"
)

const MaxTagSuggestionPageSize = 200

var (
	ErrInvalidTagSuggestionCursor = errors.New("invalid tag suggestion cursor")
	ErrTagSuggestionCursorStale   = errors.New("tag suggestion cursor stale")
)

type TagSuggestionListStatus string

const (
	TagSuggestionPending   TagSuggestionListStatus = "pending"
	TagSuggestionAccepted  TagSuggestionListStatus = "accepted"
	TagSuggestionDismissed TagSuggestionListStatus = "dismissed"
)

type TagSuggestionListSnapshot struct {
	LibraryID          int64
	GenerationID       string
	VocabularyID       string
	VocabularyRevision int64
	CatalogRevision    int64
	SuggestionRevision int64
	ReviewRevision     int64
	Coverage           TagSuggestionCoverage
}

type TagSuggestionCoverage struct {
	Eligible  int64
	Completed int64
	Degraded  int64
	Failed    int64
	Stale     int64
	Revision  int64
}

func (coverage TagSuggestionCoverage) Complete() bool {
	return coverage.Eligible == coverage.Completed+coverage.Degraded+coverage.Failed+coverage.Stale
}

type TagSuggestionListPosition struct {
	Confidence   float32
	ReviewedAtMS int64
	SuggestionID string
}

type TagSuggestionListQuery struct {
	LibraryID int64
	Status    TagSuggestionListStatus
	TagID     int64
	After     *TagSuggestionListPosition
	Limit     int
}

type TagSuggestionView struct {
	ID                 string
	LibraryID          int64
	AssetID            int64
	TagID              int64
	TagName            string
	Confidence         float32
	Status             TagSuggestionListStatus
	GenerationID       string
	VocabularyRevision int64
	Revision           int64
	ReviewedAt         *time.Time
}

type TagSuggestionListRepository interface {
	GetTagSuggestionListSnapshot(context.Context, int64) (TagSuggestionListSnapshot, error)
	ListTagSuggestionViews(context.Context, TagSuggestionListQuery) ([]TagSuggestionView, error)
}

type TagSuggestionListRequest struct {
	LibraryID int64
	Status    TagSuggestionListStatus
	TagID     int64
	Cursor    string
	Limit     int
}

type TagSuggestionPage struct {
	Items          []TagSuggestionView
	NextCursor     string
	Coverage       TagSuggestionCoverage
	ReviewRevision int64
}

type tagSuggestionCursor struct {
	Version      int                     `json:"v"`
	LibraryID    int64                   `json:"l"`
	Status       TagSuggestionListStatus `json:"s"`
	TagID        int64                   `json:"t,omitempty"`
	Snapshot     string                  `json:"p"`
	Confidence   float32                 `json:"c,omitempty"`
	ReviewedAtMS int64                   `json:"r,omitempty"`
	SuggestionID string                  `json:"i"`
}

type TagSuggestionListService struct {
	repository TagSuggestionListRepository
	cursors    *cursorcodec.Codec
}

func NewTagSuggestionListService(repository TagSuggestionListRepository, cursorKey []byte) (*TagSuggestionListService, error) {
	if repository == nil {
		return nil, errors.New("tag suggestion list repository is required")
	}
	codec, err := cursorcodec.New(cursorKey)
	if err != nil {
		return nil, err
	}
	return &TagSuggestionListService{repository: repository, cursors: codec}, nil
}

func (service *TagSuggestionListService) List(ctx context.Context, request TagSuggestionListRequest) (TagSuggestionPage, error) {
	if request.LibraryID < 1 || !validTagSuggestionListStatus(request.Status) || request.TagID < 0 || request.Limit < 1 || request.Limit > MaxTagSuggestionPageSize {
		return TagSuggestionPage{}, ErrInvalidTagSuggestion
	}
	snapshot, err := service.repository.GetTagSuggestionListSnapshot(ctx, request.LibraryID)
	if err != nil {
		return TagSuggestionPage{}, err
	}
	if snapshot.LibraryID != request.LibraryID || len(snapshot.GenerationID) < 8 || len(snapshot.GenerationID) > 128 || len(snapshot.VocabularyID) < 8 || len(snapshot.VocabularyID) > 128 ||
		snapshot.VocabularyRevision < 1 || snapshot.CatalogRevision < 1 || snapshot.SuggestionRevision < 0 || snapshot.ReviewRevision < 1 ||
		snapshot.Coverage.Eligible < 0 || snapshot.Coverage.Completed < 0 || snapshot.Coverage.Degraded < 0 || snapshot.Coverage.Failed < 0 || snapshot.Coverage.Stale < 0 || snapshot.Coverage.Revision < 1 ||
		snapshot.Coverage.Completed+snapshot.Coverage.Degraded+snapshot.Coverage.Failed+snapshot.Coverage.Stale > snapshot.Coverage.Eligible {
		return TagSuggestionPage{}, ErrInvalidTagSuggestion
	}
	fingerprint := tagSuggestionSnapshotFingerprint(snapshot)
	var after *TagSuggestionListPosition
	if request.Cursor != "" {
		var cursor tagSuggestionCursor
		if err := service.cursors.Decode(request.Cursor, "foliopath:tag-suggestion-list:v1", &cursor); err != nil ||
			cursor.Version != 1 || cursor.LibraryID != request.LibraryID || cursor.Status != request.Status || cursor.TagID != request.TagID ||
			cursor.SuggestionID == "" || math.IsNaN(float64(cursor.Confidence)) || math.IsInf(float64(cursor.Confidence), 0) {
			return TagSuggestionPage{}, ErrInvalidTagSuggestionCursor
		}
		if cursor.Snapshot != fingerprint {
			return TagSuggestionPage{}, ErrTagSuggestionCursorStale
		}
		after = &TagSuggestionListPosition{Confidence: cursor.Confidence, ReviewedAtMS: cursor.ReviewedAtMS, SuggestionID: cursor.SuggestionID}
	}
	items, err := service.repository.ListTagSuggestionViews(ctx, TagSuggestionListQuery{LibraryID: request.LibraryID,
		Status: request.Status, TagID: request.TagID, After: after, Limit: request.Limit + 1})
	if err != nil {
		return TagSuggestionPage{}, err
	}
	result := TagSuggestionPage{Items: items, Coverage: snapshot.Coverage, ReviewRevision: snapshot.ReviewRevision}
	if len(result.Items) > request.Limit {
		result.Items = result.Items[:request.Limit]
		last := result.Items[len(result.Items)-1]
		reviewedMS := int64(0)
		if last.ReviewedAt != nil {
			reviewedMS = last.ReviewedAt.UTC().UnixMilli()
		}
		result.NextCursor, err = service.cursors.Encode(tagSuggestionCursor{Version: 1, LibraryID: request.LibraryID,
			Status: request.Status, TagID: request.TagID, Snapshot: fingerprint, Confidence: last.Confidence,
			ReviewedAtMS: reviewedMS, SuggestionID: last.ID}, "foliopath:tag-suggestion-list:v1")
	}
	return result, err
}

func validTagSuggestionListStatus(value TagSuggestionListStatus) bool {
	return value == TagSuggestionPending || value == TagSuggestionAccepted || value == TagSuggestionDismissed
}

func tagSuggestionSnapshotFingerprint(value TagSuggestionListSnapshot) string {
	return digestSemanticValue(fmt.Sprintf("foliopath:tag-list-snapshot:v1\x00%d\x00%s\x00%s\x00%d\x00%d\x00%d\x00%d\x00%d",
		value.LibraryID, value.GenerationID, value.VocabularyID, value.VocabularyRevision, value.CatalogRevision,
		value.SuggestionRevision, value.ReviewRevision, value.Coverage.Revision))
}
