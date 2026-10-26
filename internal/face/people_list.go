package face

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"

	cursorcodec "github.com/HappyQuQu/foliopath/internal/cursor"
	"golang.org/x/text/unicode/norm"
)

const MaxPeoplePageSize = 200

var (
	ErrInvalidPeopleCursor = errors.New("invalid people cursor")
	ErrPeopleCursorStale   = errors.New("people cursor stale")
)

type PeoplePosition struct{ Name, ID string }
type PeopleQuery struct {
	Search string
	After  *PeoplePosition
	Limit  int
}
type PeopleSnapshot struct{ Revision int64 }
type PeopleListRepository interface {
	GetPeopleSnapshot(context.Context) (PeopleSnapshot, error)
	ListPeoplePage(context.Context, PeopleQuery) ([]Person, error)
}
type PeopleListRequest struct {
	Search, Cursor string
	Limit          int
}
type PeoplePage struct {
	Items      []Person
	NextCursor string
}
type peopleCursor struct {
	Version    int    `json:"v"`
	SearchHash string `json:"q"`
	Snapshot   int64  `json:"r"`
	Name       string `json:"n"`
	ID         string `json:"i"`
}
type PeopleListService struct {
	repository PeopleListRepository
	cursors    *cursorcodec.Codec
}

func NewPeopleListService(repository PeopleListRepository, key []byte) (*PeopleListService, error) {
	if repository == nil {
		return nil, errors.New("people list repository is required")
	}
	codec, err := cursorcodec.New(key)
	if err != nil {
		return nil, err
	}
	return &PeopleListService{repository: repository, cursors: codec}, nil
}
func (service *PeopleListService) List(ctx context.Context, request PeopleListRequest) (PeoplePage, error) {
	search := norm.NFC.String(strings.TrimSpace(request.Search))
	if request.Limit < 1 || request.Limit > MaxPeoplePageSize || len([]rune(search)) > 100 {
		return PeoplePage{}, ErrInvalidPerson
	}
	snapshot, err := service.repository.GetPeopleSnapshot(ctx)
	if err != nil {
		return PeoplePage{}, err
	}
	if snapshot.Revision < 1 {
		return PeoplePage{}, ErrInvalidPerson
	}
	hash := peopleSearchHash(search)
	var after *PeoplePosition
	if request.Cursor != "" {
		var value peopleCursor
		if len(request.Cursor) > 4096 || service.cursors.Decode(request.Cursor, "foliopath:people-list:v1", &value) != nil || value.Version != 1 || value.SearchHash != hash || value.Name == "" || !validReviewID(value.ID) {
			return PeoplePage{}, ErrInvalidPeopleCursor
		}
		if value.Snapshot != snapshot.Revision {
			return PeoplePage{}, ErrPeopleCursorStale
		}
		after = &PeoplePosition{Name: value.Name, ID: value.ID}
	}
	items, err := service.repository.ListPeoplePage(ctx, PeopleQuery{Search: search, After: after, Limit: request.Limit + 1})
	if err != nil {
		return PeoplePage{}, err
	}
	result := PeoplePage{Items: items}
	if len(result.Items) > request.Limit {
		result.Items = result.Items[:request.Limit]
		last := result.Items[len(result.Items)-1]
		result.NextCursor, err = service.cursors.Encode(peopleCursor{Version: 1, SearchHash: hash, Snapshot: snapshot.Revision, Name: last.Name, ID: last.ID}, "foliopath:people-list:v1")
	}
	return result, err
}
func peopleSearchHash(value string) string {
	sum := sha256.Sum256([]byte("foliopath:people-search:v1\x00" + value))
	return hex.EncodeToString(sum[:])
}
