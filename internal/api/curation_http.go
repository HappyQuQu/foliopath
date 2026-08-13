package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/HappyQuQu/foliopath/internal/catalog"
	"github.com/HappyQuQu/foliopath/internal/curation"
)

const maxCurationRequestBodyBytes = 16 << 10

type CurationService interface {
	GetAssetState(context.Context, int64) (curation.AssetState, error)
	SetFavorite(context.Context, int64, bool) (curation.AssetState, error)
	CreateTag(context.Context, string) (curation.Tag, bool, error)
	RenameTag(context.Context, int64, string) (curation.Tag, error)
	DeleteTag(context.Context, int64) error
	ReplaceAssetTags(context.Context, int64, int64, []int64) (curation.AssetState, error)
	ListTags(context.Context, curation.TagListRequest) (curation.TagPage, error)
	ListAssets(context.Context, curation.AssetListRequest) (curation.CuratedAssetPage, error)
}

type tagResponse struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	AssetCount int64  `json:"assetCount"`
	CreatedAt  string `json:"createdAt"`
	UpdatedAt  string `json:"updatedAt"`
}

type tagPageResponse struct {
	Items      []tagResponse `json:"items"`
	NextCursor *string       `json:"nextCursor"`
}

type assetCurationResponse struct {
	AssetID     string        `json:"assetId"`
	Favorite    bool          `json:"favorite"`
	FavoritedAt *string       `json:"favoritedAt"`
	Tags        []tagResponse `json:"tags"`
	Revision    int64         `json:"revision"`
}

type curatedAssetResponse struct {
	Asset    assetResponse         `json:"asset"`
	Curation assetCurationResponse `json:"curation"`
}

type curatedAssetPageResponse struct {
	Items      []curatedAssetResponse `json:"items"`
	NextCursor *string                `json:"nextCursor"`
	Counts     assetCountsResponse    `json:"counts"`
}

type favoriteUpdateRequest struct {
	Favorite bool `json:"favorite"`
}

type tagNameRequest struct {
	Name string `json:"name"`
}

type replaceAssetTagsRequest struct {
	TagIDs []string `json:"tagIds"`
}

func registerCurationRoutes(mux *http.ServeMux, service CurationService) {
	mux.HandleFunc("GET /api/v1/favorites", func(writer http.ResponseWriter, request *http.Request) {
		query, err := parseCuratedAssetQuery(request.URL.RawQuery, true, 0)
		if err != nil {
			writeCurationError(writer, request, err)
			return
		}
		page, err := service.ListAssets(request.Context(), query)
		if err != nil {
			writeCurationError(writer, request, err)
			return
		}
		writeJSON(writer, http.StatusOK, curatedAssetPageWire(page))
	})

	mux.HandleFunc("PUT /api/v1/assets/{assetId}/favorite", func(writer http.ResponseWriter, request *http.Request) {
		assetID, err := parseResourceID(request.PathValue("assetId"), "ast_")
		if err != nil {
			writeCurationError(writer, request, curation.ErrAssetNotFound)
			return
		}
		var input favoriteUpdateRequest
		if err := decodeCurationJSON(writer, request, &input); err != nil {
			writeCurationError(writer, request, err)
			return
		}
		state, err := service.SetFavorite(request.Context(), assetID, input.Favorite)
		if err != nil {
			writeCurationError(writer, request, err)
			return
		}
		writeAssetCuration(writer, http.StatusOK, state)
	})

	mux.HandleFunc("GET /api/v1/assets/{assetId}/curation", func(writer http.ResponseWriter, request *http.Request) {
		assetID, err := parseResourceID(request.PathValue("assetId"), "ast_")
		if err != nil {
			writeCurationError(writer, request, curation.ErrAssetNotFound)
			return
		}
		state, err := service.GetAssetState(request.Context(), assetID)
		if err != nil {
			writeCurationError(writer, request, err)
			return
		}
		writeAssetCuration(writer, http.StatusOK, state)
	})

	mux.HandleFunc("PUT /api/v1/assets/{assetId}/tags", func(writer http.ResponseWriter, request *http.Request) {
		assetID, err := parseResourceID(request.PathValue("assetId"), "ast_")
		if err != nil {
			writeCurationError(writer, request, curation.ErrAssetNotFound)
			return
		}
		revision, err := parseCurationIfMatch(request.Header.Get("If-Match"))
		if err != nil {
			writeCurationError(writer, request, err)
			return
		}
		var input replaceAssetTagsRequest
		if err := decodeCurationJSON(writer, request, &input); err != nil {
			writeCurationError(writer, request, err)
			return
		}
		ids := make([]int64, 0, len(input.TagIDs))
		for _, value := range input.TagIDs {
			id, parseErr := parseResourceID(value, "tag_")
			if parseErr != nil {
				writeCurationError(writer, request, curation.ErrInvalidRequest)
				return
			}
			ids = append(ids, id)
		}
		state, err := service.ReplaceAssetTags(request.Context(), assetID, revision, ids)
		if err != nil {
			writeCurationError(writer, request, err)
			return
		}
		writeAssetCuration(writer, http.StatusOK, state)
	})

	mux.HandleFunc("GET /api/v1/tags", func(writer http.ResponseWriter, request *http.Request) {
		query, err := parseTagListQuery(request.URL.RawQuery)
		if err != nil {
			writeCurationError(writer, request, err)
			return
		}
		page, err := service.ListTags(request.Context(), query)
		if err != nil {
			writeCurationError(writer, request, err)
			return
		}
		items := make([]tagResponse, 0, len(page.Items))
		for _, item := range page.Items {
			items = append(items, tagWire(item))
		}
		var next *string
		if page.NextCursor != "" {
			next = &page.NextCursor
		}
		writeJSON(writer, http.StatusOK, tagPageResponse{Items: items, NextCursor: next})
	})

	mux.HandleFunc("POST /api/v1/tags", func(writer http.ResponseWriter, request *http.Request) {
		var input tagNameRequest
		if err := decodeCurationJSON(writer, request, &input); err != nil {
			writeCurationError(writer, request, err)
			return
		}
		tag, created, err := service.CreateTag(request.Context(), input.Name)
		if err != nil {
			writeCurationError(writer, request, err)
			return
		}
		status := http.StatusOK
		if created {
			status = http.StatusCreated
		}
		writeJSON(writer, status, tagWire(tag))
	})

	mux.HandleFunc("PATCH /api/v1/tags/{tagId}", func(writer http.ResponseWriter, request *http.Request) {
		tagID, err := parseResourceID(request.PathValue("tagId"), "tag_")
		if err != nil {
			writeCurationError(writer, request, curation.ErrTagNotFound)
			return
		}
		var input tagNameRequest
		if err := decodeCurationJSON(writer, request, &input); err != nil {
			writeCurationError(writer, request, err)
			return
		}
		tag, err := service.RenameTag(request.Context(), tagID, input.Name)
		if err != nil {
			writeCurationError(writer, request, err)
			return
		}
		writeJSON(writer, http.StatusOK, tagWire(tag))
	})

	mux.HandleFunc("DELETE /api/v1/tags/{tagId}", func(writer http.ResponseWriter, request *http.Request) {
		tagID, err := parseResourceID(request.PathValue("tagId"), "tag_")
		if err != nil {
			writeCurationError(writer, request, curation.ErrTagNotFound)
			return
		}
		if err := service.DeleteTag(request.Context(), tagID); err != nil {
			writeCurationError(writer, request, err)
			return
		}
		writer.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("GET /api/v1/tags/{tagId}/assets", func(writer http.ResponseWriter, request *http.Request) {
		tagID, err := parseResourceID(request.PathValue("tagId"), "tag_")
		if err != nil {
			writeCurationError(writer, request, curation.ErrTagNotFound)
			return
		}
		query, err := parseCuratedAssetQuery(request.URL.RawQuery, false, tagID)
		if err != nil {
			writeCurationError(writer, request, err)
			return
		}
		page, err := service.ListAssets(request.Context(), query)
		if err != nil {
			writeCurationError(writer, request, err)
			return
		}
		writeJSON(writer, http.StatusOK, curatedAssetPageWire(page))
	})
}

func decodeCurationJSON(writer http.ResponseWriter, request *http.Request, target any) error {
	if request.Header.Get("Content-Type") != "application/json" {
		return curation.ErrInvalidRequest
	}
	request.Body = http.MaxBytesReader(writer, request.Body, maxCurationRequestBodyBytes)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return curation.ErrInvalidRequest
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return curation.ErrInvalidRequest
	}
	return nil
}

func parseTagListQuery(raw string) (curation.TagListRequest, error) {
	values, err := url.ParseQuery(raw)
	if err != nil {
		return curation.TagListRequest{}, curation.ErrInvalidRequest
	}
	for key, entries := range values {
		if (key != "q" && key != "cursor" && key != "limit") || len(entries) != 1 {
			return curation.TagListRequest{}, curation.ErrInvalidRequest
		}
	}
	request := curation.TagListRequest{}
	if value, ok := values["q"]; ok {
		if value[0] == "" {
			return curation.TagListRequest{}, curation.ErrInvalidRequest
		}
		request.Search = value[0]
	}
	if value, ok := values["cursor"]; ok {
		if value[0] == "" {
			return curation.TagListRequest{}, curation.ErrInvalidCursor
		}
		request.Cursor = value[0]
	}
	if value, ok := values["limit"]; ok {
		request.Limit, err = strconv.Atoi(value[0])
		if err != nil || request.Limit < 1 || request.Limit > curation.MaxPageSize {
			return curation.TagListRequest{}, curation.ErrInvalidRequest
		}
	}
	return request, nil
}

func parseCuratedAssetQuery(raw string, favorites bool, tagID int64) (curation.AssetListRequest, error) {
	values, err := url.ParseQuery(raw)
	if err != nil {
		return curation.AssetListRequest{}, curation.ErrInvalidRequest
	}
	for key, entries := range values {
		if (key != "libraryId" && key != "kind" && key != "sort" && key != "order" && key != "cursor" && key != "limit") || len(entries) != 1 {
			return curation.AssetListRequest{}, curation.ErrInvalidRequest
		}
	}
	request := curation.AssetListRequest{FavoriteOnly: favorites, TagID: tagID}
	if value, ok := values["libraryId"]; ok {
		request.LibraryID, err = parseResourceID(value[0], "lib_")
		if err != nil {
			return curation.AssetListRequest{}, curation.ErrInvalidRequest
		}
	}
	if value, ok := values["kind"]; ok {
		if value[0] == "" {
			return curation.AssetListRequest{}, curation.ErrInvalidRequest
		}
		for _, kind := range strings.Split(value[0], ",") {
			request.Kinds = append(request.Kinds, catalog.AssetKind(kind))
		}
	}
	if value, ok := values["sort"]; ok {
		request.Sort = curation.SortField(value[0])
	}
	if value, ok := values["order"]; ok {
		request.Order = curation.SortOrder(value[0])
	}
	if value, ok := values["cursor"]; ok {
		if value[0] == "" {
			return curation.AssetListRequest{}, curation.ErrInvalidCursor
		}
		request.Cursor = value[0]
	}
	if value, ok := values["limit"]; ok {
		request.Limit, err = strconv.Atoi(value[0])
		if err != nil || request.Limit < 1 || request.Limit > curation.MaxPageSize {
			return curation.AssetListRequest{}, curation.ErrInvalidRequest
		}
	}
	return request, nil
}

func parseCurationIfMatch(value string) (int64, error) {
	if value == "" {
		return 0, errPreconditionRequired
	}
	if !strings.HasPrefix(value, `"curation-r`) || !strings.HasSuffix(value, `"`) {
		return 0, curation.ErrPreconditionFailed
	}
	revision, err := strconv.ParseInt(strings.TrimSuffix(strings.TrimPrefix(value, `"curation-r`), `"`), 10, 64)
	if err != nil || revision <= 0 {
		return 0, curation.ErrPreconditionFailed
	}
	return revision, nil
}

func writeAssetCuration(writer http.ResponseWriter, status int, state curation.AssetState) {
	writer.Header().Set("ETag", `"curation-r`+strconv.FormatInt(state.Revision, 10)+`"`)
	writeJSON(writer, status, assetCurationWire(state))
}

func assetCurationWire(state curation.AssetState) assetCurationResponse {
	var favoritedAt *string
	if state.FavoritedAt != nil {
		value := state.FavoritedAt.UTC().Format(time.RFC3339Nano)
		favoritedAt = &value
	}
	tags := make([]tagResponse, 0, len(state.Tags))
	for _, item := range state.Tags {
		tags = append(tags, tagWire(item))
	}
	return assetCurationResponse{AssetID: assetIDString(state.AssetID), Favorite: state.Favorite, FavoritedAt: favoritedAt, Tags: tags, Revision: state.Revision}
}

func tagWire(tag curation.Tag) tagResponse {
	return tagResponse{ID: "tag_" + strconv.FormatInt(tag.ID, 10), Name: tag.Name, AssetCount: tag.AssetCount, CreatedAt: tag.CreatedAt.UTC().Format(time.RFC3339Nano), UpdatedAt: tag.UpdatedAt.UTC().Format(time.RFC3339Nano)}
}

func curatedAssetPageWire(page curation.CuratedAssetPage) curatedAssetPageResponse {
	items := make([]curatedAssetResponse, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, curatedAssetResponse{Asset: assetWire(item.Asset), Curation: assetCurationWire(item.State)})
	}
	var next *string
	if page.NextCursor != "" {
		next = &page.NextCursor
	}
	return curatedAssetPageResponse{Items: items, NextCursor: next, Counts: assetCountsResponse{All: page.Counts.All, Images: page.Counts.Images, Videos: page.Counts.Videos}}
}

func writeCurationError(writer http.ResponseWriter, request *http.Request, err error) {
	switch {
	case errors.Is(err, errPreconditionRequired):
		writePublicError(writer, request, http.StatusPreconditionRequired, "precondition_required", "A current resource validator is required.")
	case errors.Is(err, curation.ErrPreconditionFailed):
		writePublicError(writer, request, http.StatusPreconditionFailed, "precondition_failed", "The curation state has changed.")
	case errors.Is(err, curation.ErrAssetNotFound):
		writePublicError(writer, request, http.StatusNotFound, "asset_not_found", "The media item was not found.")
	case errors.Is(err, curation.ErrLibraryNotFound):
		writePublicError(writer, request, http.StatusNotFound, "library_not_found", "The media library was not found.")
	case errors.Is(err, curation.ErrTagNotFound):
		writePublicError(writer, request, http.StatusNotFound, "tag_not_found", "The tag was not found.")
	case errors.Is(err, curation.ErrTagNameConflict):
		writePublicError(writer, request, http.StatusConflict, "tag_name_conflict", "A tag with that name already exists.")
	case errors.Is(err, curation.ErrInvalidCursor):
		writePublicError(writer, request, http.StatusBadRequest, "invalid_cursor", "The pagination cursor is invalid.")
	case errors.Is(err, curation.ErrInvalidRequest):
		writePublicError(writer, request, http.StatusBadRequest, codeInvalidRequest, "The request is invalid.")
	default:
		writeInternalError(writer, request)
	}
}
