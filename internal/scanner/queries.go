package scanner

import (
	"context"
	"errors"
	"fmt"

	"github.com/HappyQuQu/foliopath/internal/cursor"
)

var (
	ErrInvalidScanCursor   = errors.New("invalid scan cursor")
	ErrScanAlreadyFinished = errors.New("scan already finished")
)

const (
	DefaultScanPageSize = 50
	MaxScanPageSize     = 200
	MaxScanCursorBytes  = 2048
	scanCursorVersion   = 1
)

type Issue struct {
	Code               string
	Count              int64
	SampleRelativePath *string
}

type Details struct {
	Run    ScanRun
	Issues []Issue
}

type Page struct {
	Items      []Details
	NextCursor string
}

type QueryPosition struct {
	CreatedAtMS int64
	ID          int64
}

type QueryRepository interface {
	ListScanRuns(context.Context, int64, QueryPosition, int) ([]ScanRun, error)
	GetScanDetails(context.Context, int64) (Details, error)
	RequestScanCancellation(context.Context, int64) (ScanRun, error)
}

type QueryService struct {
	repository QueryRepository
	codec      *cursor.Codec
}

type scanCursor struct {
	Version     int   `json:"v"`
	LibraryID   int64 `json:"l"`
	CreatedAtMS int64 `json:"t"`
	ID          int64 `json:"i"`
}

func NewQueryService(repository QueryRepository, cursorKey []byte) (*QueryService, error) {
	if repository == nil {
		return nil, errors.New("scan query repository is required")
	}
	codec, err := cursor.New(cursorKey)
	if err != nil {
		return nil, fmt.Errorf("construct scan cursor codec: %w", err)
	}
	return &QueryService{repository: repository, codec: codec}, nil
}

func (service *QueryService) List(
	ctx context.Context,
	libraryID int64,
	cursor string,
	limit int,
) (Page, error) {
	if libraryID <= 0 || limit < 0 || limit > MaxScanPageSize {
		return Page{}, errors.New("invalid scan list request")
	}
	if cursor != "" && (len(cursor) < 8 || len(cursor) > MaxScanCursorBytes) {
		return Page{}, ErrInvalidScanCursor
	}
	if limit == 0 {
		limit = DefaultScanPageSize
	}
	position := QueryPosition{CreatedAtMS: int64(^uint64(0) >> 1), ID: int64(^uint64(0) >> 1)}
	if cursor != "" {
		decoded, err := service.decodeCursor(cursor, libraryID)
		if err != nil {
			return Page{}, err
		}
		position = decoded
	}
	runs, err := service.repository.ListScanRuns(ctx, libraryID, position, limit+1)
	if err != nil {
		return Page{}, err
	}
	more := len(runs) > limit
	if more {
		runs = runs[:limit]
	}
	items := make([]Details, 0, len(runs))
	for _, run := range runs {
		details, err := service.repository.GetScanDetails(ctx, run.ID)
		if err != nil {
			return Page{}, err
		}
		items = append(items, details)
	}
	page := Page{Items: items}
	if more {
		last := runs[len(runs)-1]
		page.NextCursor, err = service.encodeCursor(libraryID, QueryPosition{
			CreatedAtMS: last.CreatedAtMS,
			ID:          last.ID,
		})
		if err != nil {
			return Page{}, err
		}
	}
	return page, nil
}

func (service *QueryService) Get(ctx context.Context, scanID int64) (Details, error) {
	if scanID <= 0 {
		return Details{}, ErrScanRunNotFound
	}
	return service.repository.GetScanDetails(ctx, scanID)
}

func (service *QueryService) Cancel(ctx context.Context, scanID int64) (ScanRun, error) {
	if scanID <= 0 {
		return ScanRun{}, ErrScanRunNotFound
	}
	return service.repository.RequestScanCancellation(ctx, scanID)
}

func (service *QueryService) encodeCursor(libraryID int64, position QueryPosition) (string, error) {
	encoded, err := service.codec.Encode(scanCursor{
		Version: scanCursorVersion, LibraryID: libraryID,
		CreatedAtMS: position.CreatedAtMS, ID: position.ID,
	}, "foliopath:library-scans:v1")
	if err != nil {
		return "", fmt.Errorf("encode scan cursor: %w", err)
	}
	return encoded, nil
}

func (service *QueryService) decodeCursor(value string, libraryID int64) (QueryPosition, error) {
	var cursor scanCursor
	if err := service.codec.Decode(value, "foliopath:library-scans:v1", &cursor); err != nil ||
		cursor.Version != scanCursorVersion ||
		cursor.LibraryID != libraryID ||
		cursor.CreatedAtMS < 0 ||
		cursor.ID <= 0 {
		return QueryPosition{}, ErrInvalidScanCursor
	}
	return QueryPosition{CreatedAtMS: cursor.CreatedAtMS, ID: cursor.ID}, nil
}
