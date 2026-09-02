// Package catalog owns indexed directory and asset browse semantics.
package catalog

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	cursorcodec "github.com/HappyQuQu/foliopath/internal/cursor"
	"github.com/HappyQuQu/foliopath/internal/media"
	"github.com/HappyQuQu/foliopath/internal/pathpolicy"
	"golang.org/x/text/cases"
	"golang.org/x/text/collate"
	"golang.org/x/text/language"
	"golang.org/x/text/unicode/norm"
)

const (
	DefaultPageSize = 50
	MaxPageSize     = 200
	MaxCursorBytes  = 2048
	MaxBreadcrumbs  = 2049

	queryVersion     = 1
	assetCursorV2    = 2
	directoryOrderV1 = 1
	assetOrderV2     = 2
	searchProfileV1  = 1
)

var (
	ErrLibraryNotFound    = errors.New("catalog library not found")
	ErrDirectoryNotFound  = errors.New("catalog directory not found")
	ErrAssetNotFound      = errors.New("catalog asset not found")
	ErrInvalidQuery       = errors.New("invalid catalog query")
	ErrInvalidCursor      = errors.New("invalid catalog cursor")
	ErrInvalidTopology    = errors.New("invalid catalog directory topology")
	ErrRepositoryNotReady = errors.New("catalog repository is not ready")
)

type AssetScopeKind string

const (
	ScopeDirectory AssetScopeKind = "directory"
	ScopeLibrary   AssetScopeKind = "library"
	ScopeGlobal    AssetScopeKind = "global"
)

type SourceAvailability string

const (
	SourceAvailable SourceAvailability = "available"
	SourceOffline   SourceAvailability = "offline"
)

type SortField string

const (
	SortDefault    SortField = ""
	SortName       SortField = "name"
	SortModifiedAt SortField = "modifiedAt"
	SortSize       SortField = "size"
)

type SortOrder string

const (
	OrderDefault SortOrder = ""
	OrderAsc     SortOrder = "asc"
	OrderDesc    SortOrder = "desc"
)

type AssetKind string

const (
	KindImage    AssetKind = "image"
	KindAnimated AssetKind = "animated"
	KindVideo    AssetKind = "video"
)

func (kind AssetKind) valid() bool {
	switch kind {
	case KindImage, KindAnimated, KindVideo:
		return true
	default:
		return false
	}
}

type Scope struct {
	LibraryID            int64
	RootDirectoryID      int64
	DirectoryID          int64
	CanonicalDirectoryID int64
	Generation           int64
	Availability         SourceAvailability
}

// NormalizeRootScope maps the indexed root ID and omitted-root form to the
// same canonical scope ID. Non-root directories retain their opaque ID.
func NormalizeRootScope(rootDirectoryID, selectedDirectoryID int64) (int64, error) {
	if rootDirectoryID == 0 && selectedDirectoryID == 0 {
		return 0, nil
	}
	if rootDirectoryID <= 0 || selectedDirectoryID <= 0 {
		return 0, ErrDirectoryNotFound
	}
	if selectedDirectoryID == rootDirectoryID {
		return 0, nil
	}
	return selectedDirectoryID, nil
}

type Directory struct {
	ID                  int64
	LibraryID           int64
	ParentID            *int64
	RelativePath        string
	Name                string
	NaturalNameKey      []byte
	DirectAssetCount    int64
	RecursiveAssetCount int64
	HasChildren         bool
}

type DirectoryLineage struct {
	LibraryName string
	Items       []Directory
}

type Breadcrumb struct {
	ID           int64
	Name         string
	RelativePath string
}

type DirectoryDetail struct {
	Directory
	Breadcrumbs []Breadcrumb
}

type Asset struct {
	ID                   int64
	LibraryID            int64
	LibraryName          string
	DirectoryID          int64
	RelativePath         string
	Name                 string
	NaturalNameKey       []byte
	Kind                 AssetKind
	MediaFormat          string
	MIMEType             string
	SizeBytes            int64
	ModifiedAtNS         int64
	SourceFingerprint    string
	Availability         SourceAvailability
	Width                *int64
	Height               *int64
	DurationMS           *int64
	ProbeStatus          media.ProbeStatus
	ProbeErrorCode       *media.ProcessingErrorCode
	PlaybackStatus       media.PlaybackStatus
	ThumbnailStatus      string
	ThumbnailErrorCode   *media.ProcessingErrorCode
	StoryboardStatus     string
	StoryboardErrorCode  *media.ProcessingErrorCode
	StoryboardFrameCount *int64
	StoryboardColumns    *int64
	StoryboardRows       *int64
	StoryboardCellWidth  *int64
	StoryboardCellHeight *int64
	Favorite             bool
}

type DirectoryPosition struct {
	NaturalNameKey []byte
	Name           string
	ID             int64
}

type AssetPosition struct {
	DirectoryPath  string
	NaturalNameKey []byte
	Name           string
	LibraryID      int64
	RelativePath   string
	ModifiedAtNS   int64
	SizeBytes      int64
	ID             int64
}

type DirectoryListParams struct {
	Scope       Scope
	SearchTerms []string
	After       *DirectoryPosition
	Limit       int
}

type AssetQuery struct {
	Scope            Scope
	ScopeKind        AssetScopeKind
	CatalogRevision  int64
	Recursive        bool
	SearchTerms      []string
	Kinds            []AssetKind
	ModifiedFromNS   *int64
	ModifiedBeforeNS *int64
	Sort             SortField
	Order            SortOrder
}

type AssetListParams struct {
	Query AssetQuery
	After *AssetPosition
	Limit int
}

type Repository interface {
	ResolveScope(context.Context, int64, int64) (Scope, error)
	ResolveGlobalCatalogRevision(context.Context) (int64, error)
	ResolveCatalogContentRevision(context.Context) (int64, error)
	ListDirectoryPage(context.Context, DirectoryListParams) ([]Directory, error)
	ListAssetPage(context.Context, AssetListParams) ([]Asset, error)
	CountAssets(context.Context, AssetQuery) (AssetCounts, error)
	GetAsset(context.Context, int64) (Asset, error)
	GetAssetsByIDs(context.Context, []int64) ([]Asset, error)
	GetDirectoryLineage(context.Context, int64, int) (DirectoryLineage, error)
}

func (service *Service) GetDirectory(
	ctx context.Context,
	directoryID int64,
) (DirectoryDetail, error) {
	if err := ctx.Err(); err != nil {
		return DirectoryDetail{}, err
	}
	if directoryID <= 0 {
		return DirectoryDetail{}, ErrDirectoryNotFound
	}
	lineage, err := service.repository.GetDirectoryLineage(ctx, directoryID, MaxBreadcrumbs)
	if err != nil {
		return DirectoryDetail{}, err
	}
	if lineage.LibraryName == "" || len(lineage.Items) == 0 ||
		len(lineage.Items) > MaxBreadcrumbs {
		return DirectoryDetail{}, ErrInvalidTopology
	}
	seen := make(map[int64]struct{}, len(lineage.Items))
	breadcrumbs := make([]Breadcrumb, 0, len(lineage.Items))
	for index, item := range lineage.Items {
		if item.ID <= 0 || item.LibraryID <= 0 || item.DirectAssetCount < 0 ||
			item.RecursiveAssetCount < 0 {
			return DirectoryDetail{}, ErrInvalidTopology
		}
		if _, duplicate := seen[item.ID]; duplicate {
			return DirectoryDetail{}, ErrInvalidTopology
		}
		seen[item.ID] = struct{}{}
		isRoot := index == 0
		if isRoot {
			if item.ParentID != nil || item.RelativePath != "" {
				return DirectoryDetail{}, ErrInvalidTopology
			}
			item.Name = lineage.LibraryName
			lineage.Items[index].Name = item.Name
		} else {
			parent := lineage.Items[index-1]
			if item.ParentID == nil || *item.ParentID != parent.ID ||
				item.LibraryID != parent.LibraryID || item.Name == "" {
				return DirectoryDetail{}, ErrInvalidTopology
			}
			normalized, normalizeErr := pathpolicy.Normalize(item.RelativePath)
			if normalizeErr != nil || normalized != item.RelativePath ||
				path.Base(item.RelativePath) != item.Name ||
				path.Dir(item.RelativePath) != rootParentPath(parent.RelativePath) {
				return DirectoryDetail{}, ErrInvalidTopology
			}
		}
		breadcrumbs = append(breadcrumbs, Breadcrumb{
			ID: item.ID, Name: item.Name, RelativePath: item.RelativePath,
		})
	}
	current := lineage.Items[len(lineage.Items)-1]
	if current.ID != directoryID {
		return DirectoryDetail{}, ErrInvalidTopology
	}
	return DirectoryDetail{Directory: current, Breadcrumbs: breadcrumbs}, nil
}

func rootParentPath(relative string) string {
	if relative == "" {
		return "."
	}
	return relative
}

type DirectoryRequest struct {
	LibraryID         int64
	ParentDirectoryID int64
	SearchQuery       *string
	Cursor            string
	Limit             int
}

type AssetRequest struct {
	LibraryID        int64
	DirectoryID      int64
	DirectorySet     bool
	Recursive        bool
	RecursiveSet     bool
	SearchQuery      *string
	Kinds            []AssetKind
	ModifiedFromNS   *int64
	ModifiedBeforeNS *int64
	Sort             SortField
	Order            SortOrder
	Cursor           string
	Limit            int
}

type GlobalSearchRequest struct {
	SearchQuery      string
	Kinds            []AssetKind
	ModifiedFromNS   *int64
	ModifiedBeforeNS *int64
	Sort             SortField
	Order            SortOrder
	Cursor           string
	Limit            int
}

type DirectoryPage struct {
	Items      []Directory
	NextCursor string
}

type AssetPage struct {
	Items      []Asset
	NextCursor string
	Counts     AssetCounts
}

type AssetCounts struct {
	All    int64
	Images int64
	Videos int64
}

type Service struct {
	repository Repository
	codec      *cursorcodec.Codec
}

func NewService(repository Repository, cursorKey []byte) (*Service, error) {
	if repository == nil {
		return nil, errors.New("catalog repository is required")
	}
	codec, err := cursorcodec.New(cursorKey)
	if err != nil {
		return nil, fmt.Errorf("construct catalog cursor codec: %w", err)
	}
	return &Service{repository: repository, codec: codec}, nil
}

func (service *Service) ContentRevision(ctx context.Context) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	revision, err := service.repository.ResolveCatalogContentRevision(ctx)
	if err != nil {
		return 0, err
	}
	if revision < 1 {
		return 0, errors.New("catalog repository returned an invalid content revision")
	}
	return revision, nil
}

func (service *Service) ListDirectories(
	ctx context.Context,
	request DirectoryRequest,
) (DirectoryPage, error) {
	if err := ctx.Err(); err != nil {
		return DirectoryPage{}, err
	}
	limit, err := normalizeLimit(request.Limit)
	if err != nil {
		return DirectoryPage{}, err
	}
	scope, err := service.resolveScope(ctx, request.LibraryID, request.ParentDirectoryID)
	if err != nil {
		return DirectoryPage{}, err
	}
	var searchTerms []string
	if request.SearchQuery != nil {
		searchTerms, err = NormalizeSearchTerms(*request.SearchQuery)
		if err != nil {
			return DirectoryPage{}, err
		}
	}
	fingerprint := directoryFingerprint(scope, searchTerms)
	var after *DirectoryPosition
	if request.Cursor != "" {
		position, decodeErr := service.decodeDirectoryCursor(request.Cursor, scope.Generation, fingerprint)
		if decodeErr != nil {
			return DirectoryPage{}, decodeErr
		}
		after = &position
	}
	items, err := service.repository.ListDirectoryPage(ctx, DirectoryListParams{
		Scope:       scope,
		SearchTerms: searchTerms,
		After:       after,
		Limit:       limit + 1,
	})
	if err != nil {
		return DirectoryPage{}, err
	}
	if len(items) <= limit {
		return DirectoryPage{Items: items}, nil
	}
	items = items[:limit]
	last := items[len(items)-1]
	next, err := service.encodeDirectoryCursor(scope.Generation, fingerprint, DirectoryPosition{
		NaturalNameKey: last.NaturalNameKey,
		Name:           last.Name,
		ID:             last.ID,
	})
	if err != nil {
		return DirectoryPage{}, err
	}
	return DirectoryPage{Items: items, NextCursor: next}, nil
}

func (service *Service) ListAssets(ctx context.Context, request AssetRequest) (AssetPage, error) {
	if err := ctx.Err(); err != nil {
		return AssetPage{}, err
	}
	limit, err := normalizeLimit(request.Limit)
	if err != nil {
		return AssetPage{}, err
	}
	query, searchRequested, err := normalizeAssetQuery(request)
	if err != nil {
		return AssetPage{}, err
	}
	scope, err := service.resolveScope(ctx, request.LibraryID, request.DirectoryID)
	if err != nil {
		return AssetPage{}, err
	}
	query.Scope = scope
	if searchRequested && !request.DirectorySet && !request.RecursiveSet {
		query.ScopeKind = ScopeLibrary
	} else {
		query.ScopeKind = ScopeDirectory
	}
	return service.listAssetQuery(ctx, query, request.Cursor, limit)
}

func (service *Service) SearchAssets(
	ctx context.Context,
	request GlobalSearchRequest,
) (AssetPage, error) {
	if err := ctx.Err(); err != nil {
		return AssetPage{}, err
	}
	limit, err := normalizeLimit(request.Limit)
	if err != nil {
		return AssetPage{}, err
	}
	query, err := normalizeGlobalSearchQuery(request)
	if err != nil {
		return AssetPage{}, err
	}
	revision, err := service.repository.ResolveGlobalCatalogRevision(ctx)
	if err != nil {
		return AssetPage{}, err
	}
	if revision < 1 {
		return AssetPage{}, errors.New("catalog repository returned an invalid global revision")
	}
	query.CatalogRevision = revision
	return service.listAssetQuery(ctx, query, request.Cursor, limit)
}

func (service *Service) listAssetQuery(
	ctx context.Context,
	query AssetQuery,
	cursor string,
	limit int,
) (AssetPage, error) {
	revision := query.Scope.Generation
	if query.ScopeKind == ScopeGlobal {
		revision = query.CatalogRevision
	}
	fingerprint := assetFingerprint(query)
	var after *AssetPosition
	var counts AssetCounts
	var err error
	if cursor != "" {
		position, cursorCounts, decodeErr := service.decodeAssetCursor(
			cursor, revision, fingerprint, query,
		)
		if decodeErr != nil {
			return AssetPage{}, decodeErr
		}
		after = &position
		counts = cursorCounts
	} else {
		countQuery := query
		countQuery.Kinds = nil
		counts, err = service.repository.CountAssets(ctx, countQuery)
		if err != nil {
			return AssetPage{}, err
		}
	}
	if !validAssetCounts(counts) {
		return AssetPage{}, errors.New("catalog repository returned invalid asset counts")
	}
	items, err := service.repository.ListAssetPage(ctx, AssetListParams{
		Query: query,
		After: after,
		Limit: limit + 1,
	})
	if err != nil {
		return AssetPage{}, err
	}
	for _, item := range items {
		if err := validateAsset(item); err != nil {
			return AssetPage{}, err
		}
	}
	if len(items) <= limit {
		return AssetPage{Items: items, Counts: counts}, nil
	}
	items = items[:limit]
	last := items[len(items)-1]
	next, err := service.encodeAssetCursor(revision, fingerprint, query, counts, AssetPosition{
		DirectoryPath:  assetDirectoryPath(last.RelativePath),
		NaturalNameKey: last.NaturalNameKey,
		Name:           last.Name,
		LibraryID:      last.LibraryID,
		RelativePath:   last.RelativePath,
		ModifiedAtNS:   last.ModifiedAtNS,
		SizeBytes:      last.SizeBytes,
		ID:             last.ID,
	})
	if err != nil {
		return AssetPage{}, err
	}
	return AssetPage{Items: items, NextCursor: next, Counts: counts}, nil
}

func (service *Service) GetAsset(ctx context.Context, assetID int64) (Asset, error) {
	if err := ctx.Err(); err != nil {
		return Asset{}, err
	}
	if assetID <= 0 {
		return Asset{}, ErrAssetNotFound
	}
	item, err := service.repository.GetAsset(ctx, assetID)
	if err != nil {
		return Asset{}, err
	}
	if err := validateAsset(item); err != nil {
		return Asset{}, err
	}
	return item, nil
}

func (service *Service) GetAssetsByIDs(ctx context.Context, assetIDs []int64) ([]Asset, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(assetIDs) < 1 || len(assetIDs) > MaxPageSize {
		return nil, ErrInvalidQuery
	}
	seen := make(map[int64]struct{}, len(assetIDs))
	for _, assetID := range assetIDs {
		if assetID < 1 {
			return nil, ErrInvalidQuery
		}
		if _, exists := seen[assetID]; exists {
			return nil, ErrInvalidQuery
		}
		seen[assetID] = struct{}{}
	}
	items, err := service.repository.GetAssetsByIDs(ctx, assetIDs)
	if err != nil {
		return nil, err
	}
	if len(items) != len(assetIDs) {
		return nil, ErrAssetNotFound
	}
	byID := make(map[int64]Asset, len(items))
	for _, item := range items {
		if err := validateAsset(item); err != nil {
			return nil, err
		}
		if _, exists := byID[item.ID]; exists {
			return nil, ErrInvalidQuery
		}
		byID[item.ID] = item
	}
	ordered := make([]Asset, 0, len(assetIDs))
	for _, assetID := range assetIDs {
		item, found := byID[assetID]
		if !found {
			return nil, ErrAssetNotFound
		}
		ordered = append(ordered, item)
	}
	return ordered, nil
}

func validateAsset(item Asset) error {
	if item.ID <= 0 || item.LibraryID <= 0 || item.LibraryName == "" ||
		item.DirectoryID <= 0 || item.RelativePath == "" || item.Name == "" ||
		item.SizeBytes < 0 || item.SourceFingerprint == "" ||
		!item.Kind.valid() {
		return errors.New("catalog repository returned an invalid asset")
	}
	switch item.ProbeStatus {
	case media.ProbePending:
		if item.ProbeErrorCode != nil {
			return errors.New("catalog repository returned invalid pending probe state")
		}
	case media.ProbeReady:
		if item.Width == nil || *item.Width < 1 || item.Height == nil || *item.Height < 1 ||
			item.ProbeErrorCode != nil {
			return errors.New("catalog repository returned invalid ready probe state")
		}
	case media.ProbeFailed, media.ProbeUnsupported:
		if item.ProbeErrorCode == nil {
			return errors.New("catalog repository returned invalid failed probe state")
		}
	default:
		return errors.New("catalog repository returned unknown probe state")
	}
	switch item.ThumbnailStatus {
	case "pending":
		if item.ThumbnailErrorCode != nil {
			return errors.New("catalog repository returned invalid pending thumbnail state")
		}
	case "ready":
		if item.ProbeStatus != media.ProbeReady || item.ThumbnailErrorCode != nil {
			return errors.New("catalog repository returned invalid ready thumbnail state")
		}
	case "failed":
		if item.ThumbnailErrorCode == nil {
			return errors.New("catalog repository returned invalid failed thumbnail state")
		}
	default:
		return errors.New("catalog repository returned unknown thumbnail state")
	}
	return nil
}

func (service *Service) resolveScope(
	ctx context.Context,
	libraryID, selectedDirectoryID int64,
) (Scope, error) {
	if libraryID <= 0 || selectedDirectoryID < 0 {
		return Scope{}, ErrInvalidQuery
	}
	scope, err := service.repository.ResolveScope(ctx, libraryID, selectedDirectoryID)
	if err != nil {
		return Scope{}, err
	}
	if scope.LibraryID != libraryID || scope.RootDirectoryID < 0 ||
		scope.DirectoryID < 0 || scope.Generation < 0 ||
		(scope.RootDirectoryID == 0) != (scope.DirectoryID == 0) {
		return Scope{}, errors.New("catalog repository returned an invalid scope")
	}
	canonical, err := NormalizeRootScope(scope.RootDirectoryID, scope.DirectoryID)
	if err != nil {
		return Scope{}, errors.New("catalog repository returned an invalid root scope")
	}
	scope.CanonicalDirectoryID = canonical
	return scope, nil
}

func normalizeLimit(limit int) (int, error) {
	if limit == 0 {
		return DefaultPageSize, nil
	}
	if limit < 1 || limit > MaxPageSize {
		return 0, ErrInvalidQuery
	}
	return limit, nil
}

func normalizeAssetQuery(request AssetRequest) (AssetQuery, bool, error) {
	searchRequested := request.SearchQuery != nil
	var searchTerms []string
	if searchRequested {
		var err error
		searchTerms, err = NormalizeSearchTerms(*request.SearchQuery)
		if err != nil {
			return AssetQuery{}, false, ErrInvalidQuery
		}
	}

	kinds, err := normalizeKinds(request.Kinds)
	if err != nil {
		return AssetQuery{}, false, err
	}
	if err := validateModifiedBounds(request.ModifiedFromNS, request.ModifiedBeforeNS); err != nil {
		return AssetQuery{}, false, err
	}

	sortField := request.Sort
	if sortField == SortDefault {
		if request.Recursive || searchRequested {
			sortField = SortModifiedAt
		} else {
			sortField = SortName
		}
	}
	if sortField != SortName && sortField != SortModifiedAt && sortField != SortSize {
		return AssetQuery{}, false, ErrInvalidQuery
	}
	order := request.Order
	if order == OrderDefault {
		if sortField == SortName {
			order = OrderAsc
		} else {
			order = OrderDesc
		}
	}
	if order != OrderAsc && order != OrderDesc {
		return AssetQuery{}, false, ErrInvalidQuery
	}
	return AssetQuery{
		Recursive: request.Recursive, SearchTerms: searchTerms, Kinds: kinds,
		ModifiedFromNS: request.ModifiedFromNS, ModifiedBeforeNS: request.ModifiedBeforeNS,
		Sort: sortField, Order: order,
	}, searchRequested, nil
}

func normalizeGlobalSearchQuery(request GlobalSearchRequest) (AssetQuery, error) {
	terms, err := NormalizeSearchTerms(request.SearchQuery)
	if err != nil {
		return AssetQuery{}, ErrInvalidQuery
	}
	kinds, err := normalizeKinds(request.Kinds)
	if err != nil {
		return AssetQuery{}, err
	}
	if err := validateModifiedBounds(request.ModifiedFromNS, request.ModifiedBeforeNS); err != nil {
		return AssetQuery{}, err
	}
	sortField := request.Sort
	if sortField == SortDefault {
		sortField = SortModifiedAt
	}
	if sortField != SortName && sortField != SortModifiedAt && sortField != SortSize {
		return AssetQuery{}, ErrInvalidQuery
	}
	order := request.Order
	if order == OrderDefault {
		if sortField == SortName {
			order = OrderAsc
		} else {
			order = OrderDesc
		}
	}
	if order != OrderAsc && order != OrderDesc {
		return AssetQuery{}, ErrInvalidQuery
	}
	return AssetQuery{
		ScopeKind: ScopeGlobal, SearchTerms: terms, Kinds: kinds,
		ModifiedFromNS: request.ModifiedFromNS, ModifiedBeforeNS: request.ModifiedBeforeNS,
		Sort: sortField, Order: order,
	}, nil
}

func normalizeKinds(values []AssetKind) ([]AssetKind, error) {
	kinds := append([]AssetKind(nil), values...)
	slices.Sort(kinds)
	kinds = slices.Compact(kinds)
	if len(kinds) > 3 {
		return nil, ErrInvalidQuery
	}
	for _, kind := range kinds {
		if !kind.valid() {
			return nil, ErrInvalidQuery
		}
	}
	return kinds, nil
}

func validateModifiedBounds(from, before *int64) error {
	if from != nil && before != nil && *from >= *before {
		return ErrInvalidQuery
	}
	return nil
}

// NormalizeSearchTerms implements the public search profile v1.
func NormalizeSearchTerms(value string) ([]string, error) {
	if !utf8.ValidString(value) || strings.ContainsRune(value, '\x00') ||
		utf8.RuneCountInString(value) > 256 {
		return nil, ErrInvalidQuery
	}
	trimmed := strings.TrimFunc(value, unicode.IsSpace)
	if trimmed == "" {
		return nil, ErrInvalidQuery
	}
	folded := cases.Fold().String(norm.NFKC.String(trimmed))
	terms := strings.FieldsFunc(folded, unicode.IsSpace)
	if len(terms) == 0 {
		return nil, ErrInvalidQuery
	}
	slices.Sort(terms)
	terms = slices.Compact(terms)
	return terms, nil
}

// SearchTextKey is the canonical persisted search representation.
func SearchTextKey(value string) string {
	return cases.Fold().String(norm.NFKC.String(value))
}

type queryFingerprint struct {
	Version          int            `json:"v"`
	OrderVersion     int            `json:"o"`
	SearchProfile    int            `json:"p,omitempty"`
	ScopeKind        AssetScopeKind `json:"c,omitempty"`
	LibraryID        int64          `json:"l"`
	DirectoryID      int64          `json:"d"`
	Generation       int64          `json:"g"`
	CatalogRevision  int64          `json:"x,omitempty"`
	Recursive        bool           `json:"r,omitempty"`
	Terms            []string       `json:"q,omitempty"`
	Kinds            []AssetKind    `json:"k,omitempty"`
	ModifiedFromNS   *int64         `json:"b,omitempty"`
	ModifiedBeforeNS *int64         `json:"e,omitempty"`
	Sort             SortField      `json:"s,omitempty"`
	Order            SortOrder      `json:"a,omitempty"`
}

func directoryFingerprint(scope Scope, terms []string) [sha256.Size]byte {
	return hashFingerprint(queryFingerprint{
		Version: queryVersion, OrderVersion: directoryOrderV1,
		SearchProfile: searchProfileV1, Terms: terms,
		LibraryID: scope.LibraryID, DirectoryID: scope.CanonicalDirectoryID,
		Generation: scope.Generation,
	})
}

func assetFingerprint(query AssetQuery) [sha256.Size]byte {
	return hashFingerprint(queryFingerprint{
		Version: queryVersion, OrderVersion: assetOrderV2,
		SearchProfile: searchProfileV1, ScopeKind: query.ScopeKind,
		LibraryID: query.Scope.LibraryID, DirectoryID: query.Scope.CanonicalDirectoryID,
		Generation: query.Scope.Generation, CatalogRevision: query.CatalogRevision,
		Recursive: query.Recursive, Terms: query.SearchTerms, Kinds: query.Kinds,
		ModifiedFromNS: query.ModifiedFromNS, ModifiedBeforeNS: query.ModifiedBeforeNS,
		Sort: query.Sort, Order: query.Order,
	})
}

func hashFingerprint(value queryFingerprint) [sha256.Size]byte {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic("catalog query fingerprint is not serializable")
	}
	return sha256.Sum256(encoded)
}

type directoryCursor struct {
	Version     int    `json:"v"`
	Generation  int64  `json:"g"`
	Fingerprint []byte `json:"f"`
	Key         []byte `json:"k"`
	Name        string `json:"n"`
	ID          int64  `json:"i"`
}

func (service *Service) encodeDirectoryCursor(
	generation int64,
	fingerprint [sha256.Size]byte,
	position DirectoryPosition,
) (string, error) {
	if generation < 0 || len(position.NaturalNameKey) == 0 ||
		position.Name == "" || position.ID <= 0 {
		return "", errors.New("catalog repository returned an invalid directory position")
	}
	value, err := service.codec.Encode(directoryCursor{
		Version: queryVersion, Generation: generation, Fingerprint: fingerprint[:],
		Key: position.NaturalNameKey, Name: position.Name, ID: position.ID,
	}, "foliopath:catalog-directories:v1")
	if err != nil {
		return "", fmt.Errorf("encode directory cursor: %w", err)
	}
	return value, nil
}

func (service *Service) decodeDirectoryCursor(
	value string,
	generation int64,
	fingerprint [sha256.Size]byte,
) (DirectoryPosition, error) {
	if len(value) < 8 || len(value) > MaxCursorBytes {
		return DirectoryPosition{}, ErrInvalidCursor
	}
	var decoded directoryCursor
	if err := service.codec.Decode(value, "foliopath:catalog-directories:v1", &decoded); err != nil ||
		decoded.Version != queryVersion || decoded.Generation != generation ||
		!bytes.Equal(decoded.Fingerprint, fingerprint[:]) || len(decoded.Key) == 0 ||
		decoded.Name == "" || decoded.ID <= 0 {
		return DirectoryPosition{}, ErrInvalidCursor
	}
	return DirectoryPosition{NaturalNameKey: decoded.Key, Name: decoded.Name, ID: decoded.ID}, nil
}

type assetCursor struct {
	Version       int       `json:"v"`
	Generation    int64     `json:"g"`
	Fingerprint   []byte    `json:"f"`
	Sort          SortField `json:"s"`
	DirectoryPath string    `json:"d,omitempty"`
	Key           []byte    `json:"k,omitempty"`
	Name          string    `json:"n,omitempty"`
	LibraryID     int64     `json:"l,omitempty"`
	RelativePath  string    `json:"p,omitempty"`
	ModifiedAtNS  int64     `json:"m,omitempty"`
	SizeBytes     int64     `json:"z,omitempty"`
	ID            int64     `json:"i"`
	All           int64     `json:"a"`
	Images        int64     `json:"x"`
	Videos        int64     `json:"y"`
}

func (service *Service) encodeAssetCursor(
	revision int64,
	fingerprint [sha256.Size]byte,
	query AssetQuery,
	counts AssetCounts,
	position AssetPosition,
) (string, error) {
	if revision < 0 || position.ID <= 0 ||
		!validAssetCounts(counts) ||
		(query.ScopeKind == ScopeGlobal && query.Sort == SortName && position.LibraryID <= 0) ||
		(query.Sort == SortName &&
			(len(position.NaturalNameKey) == 0 || position.Name == "" ||
				position.RelativePath == "" ||
				position.DirectoryPath != assetDirectoryPath(position.RelativePath))) {
		return "", errors.New("catalog repository returned an invalid asset position")
	}
	value, err := service.codec.Encode(assetCursor{
		Version: assetCursorV2, Generation: revision, Fingerprint: fingerprint[:],
		Sort: query.Sort, DirectoryPath: position.DirectoryPath,
		Key: position.NaturalNameKey, Name: position.Name,
		LibraryID:    position.LibraryID,
		RelativePath: position.RelativePath, ModifiedAtNS: position.ModifiedAtNS,
		SizeBytes: position.SizeBytes,
		ID:        position.ID,
		All:       counts.All,
		Images:    counts.Images,
		Videos:    counts.Videos,
	}, assetCursorAudience(query))
	if err != nil {
		return "", fmt.Errorf("encode asset cursor: %w", err)
	}
	return value, nil
}

func (service *Service) decodeAssetCursor(
	value string,
	revision int64,
	fingerprint [sha256.Size]byte,
	query AssetQuery,
) (AssetPosition, AssetCounts, error) {
	if len(value) < 8 || len(value) > MaxCursorBytes {
		return AssetPosition{}, AssetCounts{}, ErrInvalidCursor
	}
	var decoded assetCursor
	if err := service.codec.Decode(value, assetCursorAudience(query), &decoded); err != nil ||
		decoded.Version != assetCursorV2 || decoded.Generation != revision ||
		decoded.Sort != query.Sort || !bytes.Equal(decoded.Fingerprint, fingerprint[:]) ||
		decoded.ID <= 0 || !validAssetCounts(AssetCounts{
		All: decoded.All, Images: decoded.Images, Videos: decoded.Videos,
	}) {
		return AssetPosition{}, AssetCounts{}, ErrInvalidCursor
	}
	if query.Sort == SortName &&
		(len(decoded.Key) == 0 || decoded.Name == "" || decoded.RelativePath == "" ||
			decoded.DirectoryPath != assetDirectoryPath(decoded.RelativePath)) {
		return AssetPosition{}, AssetCounts{}, ErrInvalidCursor
	}
	if query.ScopeKind == ScopeGlobal && query.Sort == SortName && decoded.LibraryID <= 0 {
		return AssetPosition{}, AssetCounts{}, ErrInvalidCursor
	}
	return AssetPosition{
		DirectoryPath: decoded.DirectoryPath, NaturalNameKey: decoded.Key,
		Name: decoded.Name, RelativePath: decoded.RelativePath,
		LibraryID: decoded.LibraryID, ModifiedAtNS: decoded.ModifiedAtNS,
		SizeBytes: decoded.SizeBytes, ID: decoded.ID,
	}, AssetCounts{All: decoded.All, Images: decoded.Images, Videos: decoded.Videos}, nil
}

func validAssetCounts(counts AssetCounts) bool {
	return counts.All >= 0 && counts.Images >= 0 && counts.Videos >= 0 &&
		counts.Images+counts.Videos == counts.All
}

func assetDirectoryPath(relativePath string) string {
	directoryPath := path.Dir(relativePath)
	if directoryPath == "." {
		return ""
	}
	return directoryPath
}

func assetCursorAudience(query AssetQuery) string {
	if query.ScopeKind == ScopeGlobal {
		return "foliopath:catalog-search-all:v1"
	}
	return "foliopath:catalog-assets:v1"
}

// NaturalNameKey is the canonical locale-neutral, numeric-aware catalog key.
func NaturalNameKey(name string) []byte {
	collator := collate.New(language.Und, collate.Loose, collate.Numeric)
	buffer := &collate.Buffer{}
	key := append([]byte(nil), collator.KeyFromString(buffer, name)...)
	if len(key) == 0 {
		return []byte{0}
	}
	return key
}
