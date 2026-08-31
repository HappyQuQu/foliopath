package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/catalog"
	"github.com/HappyQuQu/foliopath/internal/media"
)

func TestParseDirectoryListQueryAcceptsCurrentDirectoryFilter(t *testing.T) {
	request, err := parseDirectoryListQuery("parentId=dir_7&q=%EF%BC%A1lbum&limit=25")
	if err != nil {
		t.Fatal(err)
	}
	if request.ParentDirectoryID != 7 ||
		request.SearchQuery == nil ||
		*request.SearchQuery != "Ａlbum" ||
		request.Limit != 25 {
		t.Fatalf("directory query = %#v", request)
	}
}

type catalogServiceStub struct {
	contentRevision func(context.Context) (int64, error)
	list            func(context.Context, catalog.DirectoryRequest) (catalog.DirectoryPage, error)
	get             func(context.Context, int64) (catalog.DirectoryDetail, error)
	listAssets      func(context.Context, catalog.AssetRequest) (catalog.AssetPage, error)
	searchAssets    func(context.Context, catalog.GlobalSearchRequest) (catalog.AssetPage, error)
	getAsset        func(context.Context, int64) (catalog.Asset, error)
	getAssetsByIDs  func(context.Context, []int64) ([]catalog.Asset, error)
}

func (stub catalogServiceStub) GetAssetsByIDs(ctx context.Context, assetIDs []int64) ([]catalog.Asset, error) {
	if stub.getAssetsByIDs != nil {
		return stub.getAssetsByIDs(ctx, assetIDs)
	}
	items := make([]catalog.Asset, 0, len(assetIDs))
	for _, assetID := range assetIDs {
		item, err := stub.GetAsset(ctx, assetID)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (stub catalogServiceStub) ContentRevision(ctx context.Context) (int64, error) {
	if stub.contentRevision == nil {
		return 1, nil
	}
	return stub.contentRevision(ctx)
}

func TestCatalogStateSupportsConditionalReads(t *testing.T) {
	service := catalogServiceStub{
		contentRevision: func(context.Context) (int64, error) { return 7, nil },
	}
	mux := http.NewServeMux()
	registerCatalogRoutes(mux, service)

	first := httptest.NewRecorder()
	mux.ServeHTTP(first, httptest.NewRequest(
		http.MethodGet, "/api/v1/catalog/state", nil,
	))
	if first.Code != http.StatusOK ||
		first.Header().Get("ETag") != `"catalog-r7"` ||
		first.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("catalog state response = %d %v %s", first.Code, first.Header(), first.Body)
	}
	var state catalogStateResponse
	if err := json.NewDecoder(first.Body).Decode(&state); err != nil {
		t.Fatal(err)
	}
	if state.ContentRevision != 7 {
		t.Fatalf("content revision = %d, want 7", state.ContentRevision)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/catalog/state", nil)
	request.Header.Set("If-None-Match", `"catalog-r7"`)
	notModified := httptest.NewRecorder()
	mux.ServeHTTP(notModified, request)
	if notModified.Code != http.StatusNotModified || notModified.Body.Len() != 0 {
		t.Fatalf("conditional catalog state = %d %q", notModified.Code, notModified.Body)
	}
}

func TestCatalogStateFailureIsStableAndSafe(t *testing.T) {
	service := catalogServiceStub{
		contentRevision: func(context.Context) (int64, error) {
			return 0, errors.New("sqlite detail must not escape")
		},
	}
	mux := http.NewServeMux()
	registerCatalogRoutes(mux, service)

	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(
		http.MethodGet, "/api/v1/catalog/state", nil,
	))
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("catalog state failure = %d %s", response.Code, response.Body)
	}
	assertSafeErrorResponse(t, response, "internal_error")
	if strings.Contains(response.Body.String(), "sqlite") {
		t.Fatalf("catalog state leaked internal detail: %s", response.Body)
	}
}

func TestCatalogAssetRoutesTranslateBrowseAndDetailContract(t *testing.T) {
	width := int64(100)
	height := int64(80)
	item := catalog.Asset{
		ID: 9, LibraryID: 7, LibraryName: "Family", DirectoryID: 12,
		Name: "photo.jpg", RelativePath: "Album/photo.jpg",
		Kind: catalog.KindImage, MIMEType: "image/jpeg", SizeBytes: 123,
		ModifiedAtNS:      1_700_000_000_123_000_000,
		SourceFingerprint: "v1:123:1700000000123000000",
		Availability:      catalog.SourceAvailable,
		Width:             &width, Height: &height,
		ProbeStatus: media.ProbeReady, PlaybackStatus: media.PlaybackNotApplicable,
		ThumbnailStatus: "ready",
	}
	service := catalogServiceStub{
		listAssets: func(
			_ context.Context,
			request catalog.AssetRequest,
		) (catalog.AssetPage, error) {
			if request.LibraryID != 7 || request.DirectoryID != 12 ||
				!request.Recursive || len(request.Kinds) != 2 ||
				request.Kinds[0] != catalog.KindImage ||
				request.Kinds[1] != catalog.KindVideo ||
				request.Sort != catalog.SortModifiedAt ||
				request.Order != catalog.OrderDesc ||
				request.Cursor != "opaque" || request.Limit != 25 {
				t.Fatalf("asset request = %#v", request)
			}
			return catalog.AssetPage{Items: []catalog.Asset{item}, NextCursor: "next"}, nil
		},
		getAsset: func(_ context.Context, id int64) (catalog.Asset, error) {
			if id != 9 {
				t.Fatalf("asset id = %d", id)
			}
			return item, nil
		},
	}
	mux := http.NewServeMux()
	registerCatalogRoutes(mux, service)

	list := httptest.NewRecorder()
	mux.ServeHTTP(list, httptest.NewRequest(
		http.MethodGet,
		"/api/v1/libraries/lib_7/assets?directoryId=dir_12&recursive=true&kind=image,video&sort=modifiedAt&order=desc&cursor=opaque&limit=25",
		nil,
	))
	if list.Code != http.StatusOK {
		t.Fatalf("list status = %d; body = %s", list.Code, list.Body)
	}
	var page assetPageResponse
	if err := json.NewDecoder(list.Body).Decode(&page); err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].ID != "ast_9" ||
		page.Items[0].Thumbnail.URL == nil ||
		page.NextCursor == nil || *page.NextCursor != "next" {
		t.Fatalf("asset page = %#v", page)
	}
	assertDerivedMediaURL(t, *page.Items[0].Thumbnail.URL, "ast_9", "grid")

	detail := httptest.NewRecorder()
	mux.ServeHTTP(detail, httptest.NewRequest(
		http.MethodGet, "/api/v1/assets/ast_9", nil,
	))
	if detail.Code != http.StatusOK {
		t.Fatalf("detail status = %d; body = %s", detail.Code, detail.Body)
	}
	var got assetResponse
	if err := json.NewDecoder(detail.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.ID != "ast_9" || got.LibraryID != "lib_7" ||
		got.DirectoryID != "dir_12" || got.ModifiedAt == "" ||
		got.SourceAvailability != catalog.SourceAvailable {
		t.Fatalf("asset detail = %#v", got)
	}
}

func TestStoryboardReferenceWireReadyAndNotApplicable(t *testing.T) {
	duration := int64(10_000)
	frameCount, columns, rows := int64(10), int64(5), int64(2)
	cellWidth, cellHeight := int64(320), int64(180)
	ready := storyboardReferenceWire(catalog.Asset{
		ID: 9, LibraryID: 7, RelativePath: "videos/example.mp4",
		SourceFingerprint: "v1:123:1700000000123000000",
		Kind:              catalog.KindVideo, DurationMS: &duration,
		Availability: catalog.SourceAvailable, StoryboardStatus: "ready",
		StoryboardFrameCount: &frameCount, StoryboardColumns: &columns,
		StoryboardRows: &rows, StoryboardCellWidth: &cellWidth,
		StoryboardCellHeight: &cellHeight,
	})
	if ready.Status != "ready" || ready.URL == nil ||
		ready.FrameCount == nil || *ready.FrameCount != 10 ||
		ready.ErrorCode != nil {
		t.Fatalf("ready storyboard = %#v", ready)
	}
	assertDerivedMediaURL(t, *ready.URL, "ast_9", "storyboard")
	notApplicable := storyboardReferenceWire(catalog.Asset{
		ID: 10, Kind: catalog.KindImage, Availability: catalog.SourceAvailable,
	})
	if notApplicable.Status != "not_applicable" ||
		notApplicable.URL != nil ||
		notApplicable.FrameCount != nil {
		t.Fatalf("not-applicable storyboard = %#v", notApplicable)
	}
}

func assertDerivedMediaURL(t *testing.T, raw, assetID, variant string) {
	t.Helper()
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Path != "/api/v1/assets/"+assetID+"/thumbnail" ||
		parsed.Query().Get("variant") != variant ||
		!validThumbnailVersion(parsed.Query().Get("v")) {
		t.Fatalf("derived media URL = %q", raw)
	}
}

func TestDerivedMediaURLChangesWithSourceIdentityAndVariant(t *testing.T) {
	base := catalog.Asset{
		ID: 9, LibraryID: 7, RelativePath: "videos/example.mp4",
		SourceFingerprint: "v1:123:1700000000123000000",
	}
	grid := derivedMediaURL(base, "grid")
	changedPath := base
	changedPath.RelativePath = "videos/other.mp4"
	changedFingerprint := base
	changedFingerprint.SourceFingerprint = "v1:124:1700000000123000001"
	for name, candidate := range map[string]string{
		"path":        derivedMediaURL(changedPath, "grid"),
		"fingerprint": derivedMediaURL(changedFingerprint, "grid"),
		"variant":     derivedMediaURL(base, "storyboard"),
	} {
		if candidate == grid {
			t.Fatalf("%s change retained URL %q", name, grid)
		}
	}
}

func TestCatalogSearchRoutesParseScopesAndUTCDateInterval(t *testing.T) {
	searchCalls := 0
	libraryCalls := 0
	service := catalogServiceStub{
		searchAssets: func(
			_ context.Context,
			request catalog.GlobalSearchRequest,
		) (catalog.AssetPage, error) {
			searchCalls++
			from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).UnixNano()
			before := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC).UnixNano()
			if request.SearchQuery != "上海 PHOTO" ||
				request.ModifiedFromNS == nil || *request.ModifiedFromNS != from ||
				request.ModifiedBeforeNS == nil || *request.ModifiedBeforeNS != before ||
				len(request.Kinds) != 2 || request.Sort != catalog.SortName ||
				request.Order != catalog.OrderAsc || request.Limit != 25 {
				t.Fatalf("global search request = %#v", request)
			}
			return catalog.AssetPage{}, nil
		},
		listAssets: func(
			_ context.Context,
			request catalog.AssetRequest,
		) (catalog.AssetPage, error) {
			libraryCalls++
			switch libraryCalls {
			case 1:
				if !request.DirectorySet || request.DirectoryID != 12 ||
					!request.RecursiveSet || !request.Recursive ||
					request.SearchQuery == nil || *request.SearchQuery != "photo" {
					t.Fatalf("directory search request = %#v", request)
				}
			case 2:
				if request.DirectorySet || !request.RecursiveSet || request.Recursive ||
					request.SearchQuery == nil || *request.SearchQuery != "photo" {
					t.Fatalf("root direct search request = %#v", request)
				}
			default:
				t.Fatalf("unexpected library search request = %#v", request)
			}
			return catalog.AssetPage{}, nil
		},
	}
	mux := http.NewServeMux()
	registerCatalogRoutes(mux, service)

	global := httptest.NewRecorder()
	mux.ServeHTTP(global, httptest.NewRequest(
		http.MethodGet,
		"/api/v1/assets?q=%E4%B8%8A%E6%B5%B7+PHOTO&kind=image,video"+
			"&modifiedFrom=2026-01-01T00%3A00%3A00Z"+
			"&modifiedBefore=2026-02-01T00%3A00%3A00%2B00%3A00"+
			"&sort=name&order=asc&limit=25",
		nil,
	))
	if global.Code != http.StatusOK {
		t.Fatalf("global search status = %d; body = %s", global.Code, global.Body)
	}

	directory := httptest.NewRecorder()
	mux.ServeHTTP(directory, httptest.NewRequest(
		http.MethodGet,
		"/api/v1/libraries/lib_7/assets?directoryId=dir_12&q=photo&recursive=true",
		nil,
	))
	if directory.Code != http.StatusOK {
		t.Fatalf("directory search status = %d; body = %s", directory.Code, directory.Body)
	}

	rootDirect := httptest.NewRecorder()
	mux.ServeHTTP(rootDirect, httptest.NewRequest(
		http.MethodGet,
		"/api/v1/libraries/lib_7/assets?q=photo&recursive=false",
		nil,
	))
	if rootDirect.Code != http.StatusOK {
		t.Fatalf("root direct search status = %d; body = %s", rootDirect.Code, rootDirect.Body)
	}

	for _, target := range []string{
		"/api/v1/assets",
		"/api/v1/assets?q=photo&recursive=false",
		"/api/v1/assets?q=photo&modifiedFrom=2026-01-01T01%3A00%3A00%2B01%3A00",
	} {
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, target, nil))
		if response.Code != http.StatusBadRequest {
			t.Fatalf("%s status = %d; body = %s", target, response.Code, response.Body)
		}
	}
	if searchCalls != 1 || libraryCalls != 2 {
		t.Fatalf("search calls = %d, library calls = %d", searchCalls, libraryCalls)
	}
}

func (stub catalogServiceStub) ListAssets(
	ctx context.Context,
	request catalog.AssetRequest,
) (catalog.AssetPage, error) {
	if stub.listAssets == nil {
		return catalog.AssetPage{}, errors.New("unexpected ListAssets")
	}
	return stub.listAssets(ctx, request)
}

func (stub catalogServiceStub) SearchAssets(
	ctx context.Context,
	request catalog.GlobalSearchRequest,
) (catalog.AssetPage, error) {
	if stub.searchAssets == nil {
		return catalog.AssetPage{}, errors.New("unexpected SearchAssets")
	}
	return stub.searchAssets(ctx, request)
}

func (stub catalogServiceStub) GetAsset(
	ctx context.Context,
	id int64,
) (catalog.Asset, error) {
	if stub.getAsset == nil {
		return catalog.Asset{}, errors.New("unexpected GetAsset")
	}
	return stub.getAsset(ctx, id)
}

func (stub catalogServiceStub) ListDirectories(
	ctx context.Context,
	request catalog.DirectoryRequest,
) (catalog.DirectoryPage, error) {
	if stub.list == nil {
		return catalog.DirectoryPage{}, errors.New("unexpected ListDirectories")
	}
	return stub.list(ctx, request)
}

func (stub catalogServiceStub) GetDirectory(
	ctx context.Context,
	id int64,
) (catalog.DirectoryDetail, error) {
	if stub.get == nil {
		return catalog.DirectoryDetail{}, errors.New("unexpected GetDirectory")
	}
	return stub.get(ctx, id)
}

func TestCatalogDirectoryRoutesTranslateContract(t *testing.T) {
	parentID := int64(11)
	service := catalogServiceStub{
		list: func(
			_ context.Context,
			request catalog.DirectoryRequest,
		) (catalog.DirectoryPage, error) {
			if request.LibraryID != 7 || request.ParentDirectoryID != 11 ||
				request.Cursor != "opaque" || request.Limit != 25 {
				t.Fatalf("directory request = %#v", request)
			}
			return catalog.DirectoryPage{
				Items: []catalog.Directory{{
					ID: 12, LibraryID: 7, ParentID: &parentID,
					Name: "Nested", RelativePath: "Album/Nested",
					DirectAssetCount: 2, RecursiveAssetCount: 3,
					HasChildren: true,
				}},
				NextCursor: "next",
			}, nil
		},
		get: func(_ context.Context, id int64) (catalog.DirectoryDetail, error) {
			if id != 12 {
				t.Fatalf("directory id = %d", id)
			}
			return catalog.DirectoryDetail{
				Directory: catalog.Directory{
					ID: 12, LibraryID: 7, ParentID: &parentID,
					Name: "Nested", RelativePath: "Album/Nested",
					DirectAssetCount: 2, RecursiveAssetCount: 3,
				},
				Breadcrumbs: []catalog.Breadcrumb{
					{ID: 10, Name: "Family", RelativePath: ""},
					{ID: 11, Name: "Album", RelativePath: "Album"},
					{ID: 12, Name: "Nested", RelativePath: "Album/Nested"},
				},
			}, nil
		},
	}
	mux := http.NewServeMux()
	registerCatalogRoutes(mux, service)

	listResponse := httptest.NewRecorder()
	mux.ServeHTTP(listResponse, httptest.NewRequest(
		http.MethodGet,
		"/api/v1/libraries/lib_7/directories?parentId=dir_11&cursor=opaque&limit=25",
		nil,
	))
	if listResponse.Code != http.StatusOK {
		t.Fatalf("list status = %d; body = %s", listResponse.Code, listResponse.Body)
	}
	assertJSONEquals(t, listResponse, map[string]any{
		"items": []any{map[string]any{
			"id": "dir_12", "libraryId": "lib_7", "parentId": "dir_11",
			"name": "Nested", "relativePath": "Album/Nested",
			"directAssetCount": float64(2), "recursiveAssetCount": float64(3),
			"hasChildren": true,
		}},
		"nextCursor": "next",
	})

	detailResponse := httptest.NewRecorder()
	mux.ServeHTTP(detailResponse, httptest.NewRequest(
		http.MethodGet, "/api/v1/directories/dir_12", nil,
	))
	if detailResponse.Code != http.StatusOK {
		t.Fatalf("detail status = %d; body = %s", detailResponse.Code, detailResponse.Body)
	}
	assertJSONEquals(t, detailResponse, map[string]any{
		"id": "dir_12", "libraryId": "lib_7", "parentId": "dir_11",
		"name": "Nested", "relativePath": "Album/Nested",
		"directAssetCount": float64(2), "recursiveAssetCount": float64(3),
		"hasChildren": false,
		"breadcrumbs": []any{
			map[string]any{"id": "dir_10", "name": "Family", "relativePath": ""},
			map[string]any{"id": "dir_11", "name": "Album", "relativePath": "Album"},
			map[string]any{"id": "dir_12", "name": "Nested", "relativePath": "Album/Nested"},
		},
	})
}

func TestCatalogDirectoryListRejectsNonCanonicalQueries(t *testing.T) {
	calls := 0
	service := catalogServiceStub{
		list: func(context.Context, catalog.DirectoryRequest) (catalog.DirectoryPage, error) {
			calls++
			return catalog.DirectoryPage{}, nil
		},
	}
	mux := http.NewServeMux()
	registerCatalogRoutes(mux, service)
	for _, target := range []string{
		"/api/v1/libraries/7/directories",
		"/api/v1/libraries/lib_7/directories?unknown=1",
		"/api/v1/libraries/lib_7/directories?limit=1&limit=2",
		"/api/v1/libraries/lib_7/directories?limit=0",
		"/api/v1/libraries/lib_7/directories?parentId=11",
		"/api/v1/libraries/lib_7/directories?cursor=",
	} {
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, target, nil))
		if response.Code != http.StatusBadRequest && response.Code != http.StatusNotFound {
			t.Fatalf("%s status = %d; body = %s", target, response.Code, response.Body)
		}
	}
	if calls != 0 {
		t.Fatalf("service calls = %d, want 0", calls)
	}
}

func TestCatalogDirectoryErrorsAreStableAndSafe(t *testing.T) {
	service := catalogServiceStub{
		list: func(context.Context, catalog.DirectoryRequest) (catalog.DirectoryPage, error) {
			return catalog.DirectoryPage{}, catalog.ErrDirectoryNotFound
		},
		get: func(context.Context, int64) (catalog.DirectoryDetail, error) {
			return catalog.DirectoryDetail{}, catalog.ErrInvalidTopology
		},
	}
	mux := http.NewServeMux()
	registerCatalogRoutes(mux, service)

	missing := httptest.NewRecorder()
	mux.ServeHTTP(missing, httptest.NewRequest(
		http.MethodGet, "/api/v1/libraries/lib_7/directories", nil,
	))
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing status = %d; body = %s", missing.Code, missing.Body)
	}
	assertSafeErrorResponse(t, missing, "directory_not_found")

	broken := httptest.NewRecorder()
	mux.ServeHTTP(broken, httptest.NewRequest(
		http.MethodGet, "/api/v1/directories/dir_9", nil,
	))
	if broken.Code != http.StatusInternalServerError {
		t.Fatalf("broken status = %d; body = %s", broken.Code, broken.Body)
	}
	assertSafeErrorResponse(t, broken, "internal_error")
}
