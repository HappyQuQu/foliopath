package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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
func (stub *curationServiceStub) ReplaceAssetTags(_ context.Context, _ int64, revision int64, ids []int64) (curation.AssetState, error) {
	stub.replacedRev = revision
	stub.replacedIDs = append([]int64(nil), ids...)
	return stub.state, stub.stateErr
}
func (stub *curationServiceStub) ListTags(context.Context, curation.TagListRequest) (curation.TagPage, error) {
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
