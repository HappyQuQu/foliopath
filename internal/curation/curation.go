// Package curation owns application-only favorites and manual-tag semantics.
package curation

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/HappyQuQu/foliopath/internal/catalog"
	cursorcodec "github.com/HappyQuQu/foliopath/internal/cursor"
	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"
)

const (
	DefaultPageSize = 50
	MaxPageSize     = 200
	MaxTagsPerAsset = 20
	MaxTagRunes     = 32
	queryVersion    = 1
)

var (
	ErrAssetNotFound      = errors.New("curation asset not found")
	ErrLibraryNotFound    = errors.New("curation library not found")
	ErrTagNotFound        = errors.New("curation tag not found")
	ErrTagNameConflict    = errors.New("curation tag name conflict")
	ErrInvalidRequest     = errors.New("invalid curation request")
	ErrInvalidCursor      = errors.New("invalid curation cursor")
	ErrPreconditionFailed = errors.New("curation precondition failed")
)

type Tag struct {
	ID             int64
	Name           string
	NormalizedName string
	AssetCount     int64
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type AssetState struct {
	AssetID     int64
	Favorite    bool
	FavoritedAt *time.Time
	Tags        []Tag
	Revision    int64
}

type CuratedAsset struct {
	Asset catalog.Asset
	State AssetState
}

type SortField string

const (
	SortFavoriteAt SortField = "favoritedAt"
	SortModifiedAt SortField = "modifiedAt"
	SortName       SortField = "name"
	SortSize       SortField = "size"
)

type SortOrder string

const (
	OrderAsc  SortOrder = "asc"
	OrderDesc SortOrder = "desc"
)

type TagPosition struct {
	NormalizedName string
	ID             int64
}

type AssetPosition struct {
	FavoriteAtMS   int64
	ModifiedAtNS   int64
	SizeBytes      int64
	LibraryID      int64
	DirectoryPath  string
	NaturalNameKey []byte
	Name           string
	RelativePath   string
	ID             int64
}

type TagListParams struct {
	SearchKey string
	After     *TagPosition
	Limit     int
}

type AssetQuery struct {
	FavoriteOnly bool
	TagID        int64
	LibraryID    int64
	Kinds        []catalog.AssetKind
	Sort         SortField
	Order        SortOrder
	Revision     int64
}

type AssetListParams struct {
	Query AssetQuery
	After *AssetPosition
	Limit int
}

type Repository interface {
	Revision(context.Context) (int64, error)
	GetAssetState(context.Context, int64) (AssetState, error)
	SetFavorite(context.Context, int64, bool, time.Time) (bool, error)
	CreateTag(context.Context, string, string, time.Time) (Tag, bool, error)
	RenameTag(context.Context, int64, string, string, time.Time) (Tag, error)
	DeleteTag(context.Context, int64) error
	ReplaceAssetTags(context.Context, int64, int64, []int64, time.Time) error
	AddAssetTag(context.Context, int64, int64, int64, time.Time) error
	ListTagPage(context.Context, TagListParams) ([]Tag, error)
	ListCuratedAssetPage(context.Context, AssetListParams) ([]CuratedAsset, error)
	CountCuratedAssets(context.Context, AssetQuery) (catalog.AssetCounts, error)
	ResolveLibrary(context.Context, int64) error
	ResolveTag(context.Context, int64) error
}

type Service struct {
	repository Repository
	codec      *cursorcodec.Codec
	now        func() time.Time
}

func NewService(repository Repository, cursorKey []byte, now func() time.Time) (*Service, error) {
	if repository == nil {
		return nil, errors.New("curation repository is required")
	}
	codec, err := cursorcodec.New(cursorKey)
	if err != nil {
		return nil, fmt.Errorf("construct curation cursor codec: %w", err)
	}
	if now == nil {
		now = time.Now
	}
	return &Service{repository: repository, codec: codec, now: now}, nil
}

func NormalizeTagName(input string) (string, string, error) {
	for _, current := range input {
		if unicode.IsControl(current) {
			return "", "", ErrInvalidRequest
		}
	}
	display := strings.Join(strings.Fields(norm.NFC.String(input)), " ")
	if display == "" || !utf8.ValidString(display) || utf8.RuneCountInString(display) > MaxTagRunes {
		return "", "", ErrInvalidRequest
	}
	key := cases.Fold().String(display)
	if key == "" {
		return "", "", ErrInvalidRequest
	}
	return display, key, nil
}

func (service *Service) GetAssetState(ctx context.Context, assetID int64) (AssetState, error) {
	if assetID <= 0 {
		return AssetState{}, ErrAssetNotFound
	}
	state, err := service.repository.GetAssetState(ctx, assetID)
	if err != nil {
		return AssetState{}, err
	}
	return validateState(state, assetID)
}

func (service *Service) SetFavorite(ctx context.Context, assetID int64, favorite bool) (AssetState, error) {
	if assetID <= 0 {
		return AssetState{}, ErrAssetNotFound
	}
	if _, err := service.repository.SetFavorite(ctx, assetID, favorite, service.now().UTC()); err != nil {
		return AssetState{}, err
	}
	return service.GetAssetState(ctx, assetID)
}

func (service *Service) CreateTag(ctx context.Context, input string) (Tag, bool, error) {
	name, key, err := NormalizeTagName(input)
	if err != nil {
		return Tag{}, false, err
	}
	tag, created, err := service.repository.CreateTag(ctx, name, key, service.now().UTC())
	if err != nil {
		return Tag{}, false, err
	}
	validated, err := validateTag(tag)
	return validated, created, err
}

func (service *Service) RenameTag(ctx context.Context, tagID int64, input string) (Tag, error) {
	if tagID <= 0 {
		return Tag{}, ErrTagNotFound
	}
	name, key, err := NormalizeTagName(input)
	if err != nil {
		return Tag{}, err
	}
	tag, err := service.repository.RenameTag(ctx, tagID, name, key, service.now().UTC())
	if err != nil {
		return Tag{}, err
	}
	validated, err := validateTag(tag)
	return validated, err
}

func (service *Service) DeleteTag(ctx context.Context, tagID int64) error {
	if tagID <= 0 {
		return ErrTagNotFound
	}
	return service.repository.DeleteTag(ctx, tagID)
}

func (service *Service) ReplaceAssetTags(ctx context.Context, assetID, revision int64, tagIDs []int64) (AssetState, error) {
	if assetID <= 0 {
		return AssetState{}, ErrAssetNotFound
	}
	if revision <= 0 || len(tagIDs) > MaxTagsPerAsset {
		return AssetState{}, ErrInvalidRequest
	}
	ids := append([]int64(nil), tagIDs...)
	slices.Sort(ids)
	ids = slices.Compact(ids)
	if len(ids) != len(tagIDs) {
		return AssetState{}, ErrInvalidRequest
	}
	for _, id := range ids {
		if id <= 0 {
			return AssetState{}, ErrInvalidRequest
		}
	}
	if err := service.repository.ReplaceAssetTags(ctx, assetID, revision, ids, service.now().UTC()); err != nil {
		return AssetState{}, err
	}
	return service.GetAssetState(ctx, assetID)
}

// AddAssetTag is the curation-owned narrow port used by controlled AI
// suggestion acceptance. It never replaces or removes existing manual tags.
func (service *Service) AddAssetTag(ctx context.Context, assetID, revision, tagID int64) (AssetState, error) {
	if assetID < 1 || revision < 1 || tagID < 1 {
		return AssetState{}, ErrInvalidRequest
	}
	if err := service.repository.AddAssetTag(ctx, assetID, revision, tagID, service.now().UTC()); err != nil {
		return AssetState{}, err
	}
	return service.GetAssetState(ctx, assetID)
}

type TagListRequest struct {
	Search string
	Cursor string
	Limit  int
}

type TagPage struct {
	Items      []Tag
	NextCursor string
}

func (service *Service) ListTags(ctx context.Context, request TagListRequest) (TagPage, error) {
	limit, err := normalizeLimit(request.Limit)
	if err != nil {
		return TagPage{}, err
	}
	searchKey := ""
	if request.Search != "" {
		_, searchKey, err = NormalizeTagName(request.Search)
		if err != nil {
			return TagPage{}, err
		}
	}
	revision, err := service.repository.Revision(ctx)
	if err != nil {
		return TagPage{}, err
	}
	fingerprint := fingerprint(struct {
		Search string `json:"q"`
	}{searchKey})
	var after *TagPosition
	if request.Cursor != "" {
		position, decodeErr := service.decodeTagCursor(request.Cursor, revision, fingerprint)
		if decodeErr != nil {
			return TagPage{}, decodeErr
		}
		after = &position
	}
	items, err := service.repository.ListTagPage(ctx, TagListParams{SearchKey: searchKey, After: after, Limit: limit + 1})
	if err != nil {
		return TagPage{}, err
	}
	for index := range items {
		validated, validateErr := validateTag(items[index])
		if validateErr != nil {
			return TagPage{}, validateErr
		}
		items[index] = validated
	}
	if len(items) <= limit {
		return TagPage{Items: items}, nil
	}
	items = items[:limit]
	last := items[len(items)-1]
	next, err := service.encodeTagCursor(revision, fingerprint, TagPosition{NormalizedName: last.NormalizedName, ID: last.ID})
	if err != nil {
		return TagPage{}, err
	}
	return TagPage{Items: items, NextCursor: next}, nil
}

type AssetListRequest struct {
	FavoriteOnly bool
	TagID        int64
	LibraryID    int64
	Kinds        []catalog.AssetKind
	Sort         SortField
	Order        SortOrder
	Cursor       string
	Limit        int
}

type CuratedAssetPage struct {
	Items      []CuratedAsset
	NextCursor string
	Counts     catalog.AssetCounts
}

func (service *Service) ListAssets(ctx context.Context, request AssetListRequest) (CuratedAssetPage, error) {
	limit, err := normalizeLimit(request.Limit)
	if err != nil || request.FavoriteOnly == (request.TagID > 0) {
		return CuratedAssetPage{}, ErrInvalidRequest
	}
	if request.LibraryID < 0 || request.TagID < 0 {
		return CuratedAssetPage{}, ErrInvalidRequest
	}
	if request.LibraryID > 0 {
		if err := service.repository.ResolveLibrary(ctx, request.LibraryID); err != nil {
			return CuratedAssetPage{}, err
		}
	}
	if request.TagID > 0 {
		if err := service.repository.ResolveTag(ctx, request.TagID); err != nil {
			return CuratedAssetPage{}, err
		}
	}
	query, err := normalizeAssetQuery(request)
	if err != nil {
		return CuratedAssetPage{}, err
	}
	revision, err := service.repository.Revision(ctx)
	if err != nil {
		return CuratedAssetPage{}, err
	}
	query.Revision = revision
	fingerprintValue := fingerprint(query)
	var after *AssetPosition
	if request.Cursor != "" {
		position, decodeErr := service.decodeAssetCursor(request.Cursor, revision, fingerprintValue)
		if decodeErr != nil {
			return CuratedAssetPage{}, decodeErr
		}
		after = &position
	}
	countQuery := query
	countQuery.Kinds = nil
	counts, err := service.repository.CountCuratedAssets(ctx, countQuery)
	if err != nil {
		return CuratedAssetPage{}, err
	}
	items, err := service.repository.ListCuratedAssetPage(ctx, AssetListParams{Query: query, After: after, Limit: limit + 1})
	if err != nil {
		return CuratedAssetPage{}, err
	}
	for _, item := range items {
		if item.Asset.ID <= 0 {
			return CuratedAssetPage{}, errors.New("curation repository returned invalid asset")
		}
		if _, err := validateState(item.State, item.Asset.ID); err != nil {
			return CuratedAssetPage{}, err
		}
	}
	if len(items) <= limit {
		return CuratedAssetPage{Items: items, Counts: counts}, nil
	}
	items = items[:limit]
	last := items[len(items)-1]
	position := assetPosition(last)
	next, err := service.encodeAssetCursor(revision, fingerprintValue, position)
	if err != nil {
		return CuratedAssetPage{}, err
	}
	return CuratedAssetPage{Items: items, NextCursor: next, Counts: counts}, nil
}

func normalizeAssetQuery(request AssetListRequest) (AssetQuery, error) {
	kinds := append([]catalog.AssetKind(nil), request.Kinds...)
	slices.Sort(kinds)
	kinds = slices.Compact(kinds)
	for _, kind := range kinds {
		if kind != catalog.KindImage && kind != catalog.KindAnimated && kind != catalog.KindVideo {
			return AssetQuery{}, ErrInvalidRequest
		}
	}
	sortField := request.Sort
	if sortField == "" {
		if request.FavoriteOnly {
			sortField = SortFavoriteAt
		} else {
			sortField = SortModifiedAt
		}
	}
	if sortField == SortFavoriteAt && !request.FavoriteOnly {
		return AssetQuery{}, ErrInvalidRequest
	}
	if sortField != SortFavoriteAt && sortField != SortModifiedAt && sortField != SortName && sortField != SortSize {
		return AssetQuery{}, ErrInvalidRequest
	}
	order := request.Order
	if order == "" {
		if sortField == SortName {
			order = OrderAsc
		} else {
			order = OrderDesc
		}
	}
	if order != OrderAsc && order != OrderDesc {
		return AssetQuery{}, ErrInvalidRequest
	}
	return AssetQuery{FavoriteOnly: request.FavoriteOnly, TagID: request.TagID, LibraryID: request.LibraryID, Kinds: kinds, Sort: sortField, Order: order}, nil
}

func normalizeLimit(limit int) (int, error) {
	if limit == 0 {
		return DefaultPageSize, nil
	}
	if limit < 1 || limit > MaxPageSize {
		return 0, ErrInvalidRequest
	}
	return limit, nil
}

func validateTag(tag Tag) (Tag, error) {
	name, key, err := NormalizeTagName(tag.Name)
	if err != nil || tag.ID <= 0 || tag.AssetCount < 0 || tag.CreatedAt.IsZero() || tag.UpdatedAt.Before(tag.CreatedAt) || key != tag.NormalizedName {
		return Tag{}, errors.New("curation repository returned invalid tag")
	}
	tag.Name = name
	return tag, nil
}

func validateState(state AssetState, assetID int64) (AssetState, error) {
	if state.AssetID != assetID || state.Revision < 1 || len(state.Tags) > MaxTagsPerAsset || state.Favorite != (state.FavoritedAt != nil) {
		return AssetState{}, errors.New("curation repository returned invalid asset state")
	}
	for index := range state.Tags {
		validated, err := validateTag(state.Tags[index])
		if err != nil {
			return AssetState{}, err
		}
		state.Tags[index] = validated
	}
	return state, nil
}

type tagCursor struct {
	Version     int         `json:"v"`
	Revision    int64       `json:"r"`
	Fingerprint string      `json:"f"`
	Position    TagPosition `json:"p"`
}

type assetCursor struct {
	Version     int           `json:"v"`
	Revision    int64         `json:"r"`
	Fingerprint string        `json:"f"`
	Position    AssetPosition `json:"p"`
}

func (service *Service) encodeTagCursor(revision int64, fingerprint string, position TagPosition) (string, error) {
	return service.codec.Encode(tagCursor{queryVersion, revision, fingerprint, position}, "curation:tags:v1")
}

func (service *Service) decodeTagCursor(encoded string, revision int64, fingerprint string) (TagPosition, error) {
	var payload tagCursor
	if err := service.codec.Decode(encoded, "curation:tags:v1", &payload); err != nil || payload.Version != queryVersion || payload.Revision != revision || payload.Fingerprint != fingerprint || payload.Position.ID <= 0 || payload.Position.NormalizedName == "" {
		return TagPosition{}, ErrInvalidCursor
	}
	return payload.Position, nil
}

func (service *Service) encodeAssetCursor(revision int64, fingerprint string, position AssetPosition) (string, error) {
	return service.codec.Encode(assetCursor{queryVersion, revision, fingerprint, position}, "curation:assets:v1")
}

func (service *Service) decodeAssetCursor(encoded string, revision int64, fingerprint string) (AssetPosition, error) {
	var payload assetCursor
	if err := service.codec.Decode(encoded, "curation:assets:v1", &payload); err != nil || payload.Version != queryVersion || payload.Revision != revision || payload.Fingerprint != fingerprint || payload.Position.ID <= 0 {
		return AssetPosition{}, ErrInvalidCursor
	}
	return payload.Position, nil
}

func fingerprint(value any) string {
	encoded, _ := json.Marshal(value)
	digest := sha256.Sum256(encoded)
	return fmt.Sprintf("%x", digest[:16])
}

func assetPosition(item CuratedAsset) AssetPosition {
	position := AssetPosition{
		ModifiedAtNS:   item.Asset.ModifiedAtNS,
		SizeBytes:      item.Asset.SizeBytes,
		LibraryID:      item.Asset.LibraryID,
		DirectoryPath:  assetDirectoryPath(item.Asset.RelativePath),
		NaturalNameKey: item.Asset.NaturalNameKey,
		Name:           item.Asset.Name,
		RelativePath:   item.Asset.RelativePath,
		ID:             item.Asset.ID,
	}
	if item.State.FavoritedAt != nil {
		position.FavoriteAtMS = item.State.FavoritedAt.UnixMilli()
	}
	return position
}

func assetDirectoryPath(relativePath string) string {
	index := strings.LastIndexByte(relativePath, '/')
	if index < 0 {
		return ""
	}
	return relativePath[:index]
}
