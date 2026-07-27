package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/catalog"
)

type catalogServiceStub struct {
	list func(context.Context, catalog.DirectoryRequest) (catalog.DirectoryPage, error)
	get  func(context.Context, int64) (catalog.DirectoryDetail, error)
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
