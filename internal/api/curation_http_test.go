package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/catalog"
	"github.com/HappyQuQu/foliopath/internal/curation"
)

type curationServiceStub struct {
	state          curation.AssetState
	stateErr       error
	createdTag     curation.Tag
	tagCreated     bool
	tagErr         error
	replacedIDs    []int64
	replacedRev    int64
	assetList      curation.CuratedAssetPage
	assetListQuery curation.AssetListRequest
	tagListQuery   curation.TagListRequest
	tagListCalls   int
}

func (stub *curationServiceStub) GetAssetState(context.Context, int64) (curation.AssetState, error) {
	return stub.state, stub.stateErr
}
func (stub *curationServiceStub) SetFavorite(context.Context, int64, bool) (curation.AssetState, error) {
	return stub.state, stub.stateErr
}
func (stub *curationServiceStub) CreateTag(context.Context, string) (curation.Tag, bool, error) {
	return stub.createdTag, stub.tagCreated, stub.tagErr
}
func (stub *curationServiceStub) RenameTag(context.Context, int64, string) (curation.Tag, error) {
	return stub.createdTag, stub.tagErr
}
func (stub *curationServiceStub) DeleteTag(context.Context, int64) error { return stub.tagErr }
func (stub *curationServiceStub) AddAssetTag(_ context.Context, _ int64, revision, tagID int64) (curation.AssetState, error) {
	return stub.ReplaceAssetTags(context.Background(), 0, revision, []int64{tagID})
}
func (stub *curationServiceStub) ReplaceAssetTags(_ context.Context, _ int64, revision int64, ids []int64) (curation.AssetState, error) {
	stub.replacedRev = revision
	stub.replacedIDs = append([]int64(nil), ids...)
	return stub.state, stub.stateErr
}
func (stub *curationServiceStub) ListTags(_ context.Context, query curation.TagListRequest) (curation.TagPage, error) {
	stub.tagListQuery, stub.tagListCalls = query, stub.tagListCalls+1
	return curation.TagPage{}, stub.tagErr
}
func (stub *curationServiceStub) ListAssets(_ context.Context, query curation.AssetListRequest) (curation.CuratedAssetPage, error) {
	stub.assetListQuery = query
	return stub.assetList, stub.stateErr
}

func TestCurationFavoriteResponseUsesOpaqueIDAndETag(t *testing.T) {
	mux := http.NewServeMux()
	favoriteTime := time.Date(2026, 8, 10, 10, 0, 0, 0, time.UTC)
	service := &curationServiceStub{state: curation.AssetState{AssetID: 7, Favorite: true, FavoritedAt: &favoriteTime, Revision: 4}}
	registerCurationRoutes(mux, service)
	request := httptest.NewRequest(http.MethodPut, "/api/v1/assets/ast_7/favorite", strings.NewReader(`{"favorite":true}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusOK || response.Header().Get("ETag") != `"curation-r4"` {
		t.Fatalf("status/etag = %d/%q; body=%s", response.Code, response.Header().Get("ETag"), response.Body.String())
	}
	for _, expected := range []string{`"assetId":"ast_7"`, `"favorite":true`, `"revision":4`} {
		if !strings.Contains(response.Body.String(), expected) {
			t.Fatalf("response body %q missing %q", response.Body.String(), expected)
		}
	}
}

func TestCurationReplaceTagsRequiresCurrentValidator(t *testing.T) {
	mux := http.NewServeMux()
	service := &curationServiceStub{state: curation.AssetState{AssetID: 7, Revision: 6}}
	registerCurationRoutes(mux, service)

	missing := httptest.NewRequest(http.MethodPut, "/api/v1/assets/ast_7/tags", strings.NewReader(`{"tagIds":[]}`))
	missing.Header.Set("Content-Type", "application/json")
	missingResponse := httptest.NewRecorder()
	mux.ServeHTTP(missingResponse, missing)
	if missingResponse.Code != http.StatusPreconditionRequired || !strings.Contains(missingResponse.Body.String(), `"precondition_required"`) {
		t.Fatalf("missing precondition = %d %s", missingResponse.Code, missingResponse.Body.String())
	}

	request := httptest.NewRequest(http.MethodPut, "/api/v1/assets/ast_7/tags", strings.NewReader(`{"tagIds":["tag_2","tag_5"]}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("If-Match", `"curation-r5"`)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusOK || service.replacedRev != 5 || len(service.replacedIDs) != 2 || service.replacedIDs[0] != 2 || service.replacedIDs[1] != 5 {
		t.Fatalf("replace response/query = %d rev=%d ids=%v body=%s", response.Code, service.replacedRev, service.replacedIDs, response.Body.String())
	}
}

func TestExplicitManualTagUsesCASAndAssociationSource(t *testing.T) {
	mux := http.NewServeMux()
	service := &curationServiceStub{state: curation.AssetState{AssetID: 7, Revision: 6,
		Tags: []curation.Tag{{ID: 2, Name: "flower", Source: "manual"}, {ID: 3, Name: "sky", Source: "ai"}}}}
	registerCurationRoutes(mux, service)
	for _, test := range []struct {
		etag, body string
		want       int
	}{
		{"", "", 428}, {`"curation-r5"`, `{}`, 400}, {`"curation-r5"`, "", 200},
	} {
		req := httptest.NewRequest(http.MethodPut, "/api/v1/assets/ast_7/tags/tag_2", strings.NewReader(test.body))
		req.Header.Set("If-Match", test.etag)
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, req)
		if response.Code != test.want {
			t.Fatal(response.Code, response.Body.String())
		}
		if test.want == 200 {
			if service.replacedRev != 5 || !slices.Equal(service.replacedIDs, []int64{2}) {
				t.Fatal("wrong explicit manual action")
			}
			for _, expected := range []string{`"source":"manual"`, `"source":"ai"`} {
				if !strings.Contains(response.Body.String(), expected) {
					t.Fatal(response.Body.String())
				}
			}
		}
	}
}

func TestCurationRoutesMapConflictsAndParseFavoriteScope(t *testing.T) {
	mux := http.NewServeMux()
	service := &curationServiceStub{tagErr: curation.ErrTagNameConflict}
	registerCurationRoutes(mux, service)
	request := httptest.NewRequest(http.MethodPatch, "/api/v1/tags/tag_3", strings.NewReader(`{"name":"Trip"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusConflict || !strings.Contains(response.Body.String(), `"tag_name_conflict"`) {
		t.Fatalf("conflict response = %d %s", response.Code, response.Body.String())
	}

	service.tagErr = nil
	listRequest := httptest.NewRequest(http.MethodGet, "/api/v1/favorites?libraryId=lib_4&kind=image,video&sort=size&order=asc&limit=25", nil)
	listResponse := httptest.NewRecorder()
	mux.ServeHTTP(listResponse, listRequest)
	if listResponse.Code != http.StatusOK || service.assetListQuery.LibraryID != 4 || !service.assetListQuery.FavoriteOnly || service.assetListQuery.Sort != curation.SortSize || service.assetListQuery.Order != curation.OrderAsc || len(service.assetListQuery.Kinds) != 2 || service.assetListQuery.Limit != 25 {
		t.Fatalf("favorite list response/query = %d %#v body=%s", listResponse.Code, service.assetListQuery, listResponse.Body.String())
	}

	writeResponse := httptest.NewRecorder()
	writeCurationError(writeResponse, httptest.NewRequest(http.MethodGet, "/", nil), errors.New("database /app/data/secret SELECT"))
	if writeResponse.Code != http.StatusInternalServerError || strings.Contains(writeResponse.Body.String(), "/app/data") || strings.Contains(writeResponse.Body.String(), "SELECT") {
		t.Fatalf("internal error leaked detail: %s", writeResponse.Body.String())
	}
}

func TestCurationTagListParsesOptionalScopeAndMapsUnknownLibrary(t *testing.T) {
	mux := http.NewServeMux()
	service := &curationServiceStub{}
	registerCurationRoutes(mux, service)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/tags?libraryId=lib_4&kind=image,animated&q=Trip&limit=25", nil))
	if response.Code != http.StatusOK || service.tagListQuery.LibraryID != 4 || service.tagListQuery.Search != "Trip" || service.tagListQuery.Limit != 25 ||
		!slices.Equal(service.tagListQuery.Kinds, []catalog.AssetKind{catalog.KindImage, catalog.KindAnimated}) {
		t.Fatalf("tag list response/query = %d %#v %s", response.Code, service.tagListQuery, response.Body.String())
	}
	service.tagErr = curation.ErrLibraryNotFound
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/tags?libraryId=lib_999", nil))
	if response.Code != http.StatusNotFound || !strings.Contains(response.Body.String(), `"library_not_found"`) || service.tagListCalls != 2 || service.tagListQuery.LibraryID != 999 {
		t.Fatalf("unknown library fell back: calls=%d query=%#v status/body=%d/%s", service.tagListCalls, service.tagListQuery, response.Code, response.Body.String())
	}
	service.tagErr = nil
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/tags", nil))
	if response.Code != http.StatusOK || service.tagListQuery.LibraryID != 0 || len(service.tagListQuery.Kinds) != 0 {
		t.Fatalf("global compatibility = %d %#v", response.Code, service.tagListQuery)
	}
}

func TestCurationTagListRejectsMalformedScopeBeforeService(t *testing.T) {
	for _, query := range []string{
		"libraryId=", "libraryId=lib_0", "libraryId=ast_1", "libraryId=lib_1&libraryId=lib_2",
		"kind=", "kind=image&kind=video", "q=", "limit=201", "scope=all", "kind=%zz",
	} {
		t.Run(query, func(t *testing.T) {
			mux := http.NewServeMux()
			service := &curationServiceStub{}
			registerCurationRoutes(mux, service)
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/tags?"+query, nil))
			if response.Code != http.StatusBadRequest || service.tagListCalls != 0 {
				t.Fatalf("invalid query reached service: status=%d calls=%d body=%s", response.Code, service.tagListCalls, response.Body.String())
			}
		})
	}
}
