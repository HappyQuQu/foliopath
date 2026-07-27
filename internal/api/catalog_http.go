package api

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/HappyQuQu/foliopath/internal/catalog"
	"github.com/HappyQuQu/foliopath/internal/media"
)

type CatalogService interface {
	ListDirectories(context.Context, catalog.DirectoryRequest) (catalog.DirectoryPage, error)
	GetDirectory(context.Context, int64) (catalog.DirectoryDetail, error)
	ListAssets(context.Context, catalog.AssetRequest) (catalog.AssetPage, error)
	SearchAssets(context.Context, catalog.GlobalSearchRequest) (catalog.AssetPage, error)
	GetAsset(context.Context, int64) (catalog.Asset, error)
}

type directoryResponse struct {
	ID                  string  `json:"id"`
	LibraryID           string  `json:"libraryId"`
	ParentID            *string `json:"parentId"`
	Name                string  `json:"name"`
	RelativePath        string  `json:"relativePath"`
	DirectAssetCount    int64   `json:"directAssetCount"`
	RecursiveAssetCount int64   `json:"recursiveAssetCount"`
	HasChildren         bool    `json:"hasChildren"`
}

type breadcrumbResponse struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	RelativePath string `json:"relativePath"`
}

type directoryDetailResponse struct {
	directoryResponse
	Breadcrumbs []breadcrumbResponse `json:"breadcrumbs"`
}

type directoryPageResponse struct {
	Items      []directoryResponse `json:"items"`
	NextCursor *string             `json:"nextCursor"`
}

type thumbnailReferenceResponse struct {
	Status    string  `json:"status"`
	URL       *string `json:"url"`
	ErrorCode *string `json:"errorCode"`
}

type assetResponse struct {
	ID                 string                     `json:"id"`
	LibraryID          string                     `json:"libraryId"`
	LibraryName        string                     `json:"libraryName"`
	DirectoryID        string                     `json:"directoryId"`
	Name               string                     `json:"name"`
	RelativePath       string                     `json:"relativePath"`
	Kind               catalog.AssetKind          `json:"kind"`
	MIMEType           string                     `json:"mimeType"`
	SizeBytes          int64                      `json:"sizeBytes"`
	Width              *int64                     `json:"width"`
	Height             *int64                     `json:"height"`
	DurationMS         *int64                     `json:"durationMs"`
	ModifiedAt         string                     `json:"modifiedAt"`
	ProbeStatus        media.ProbeStatus          `json:"probeStatus"`
	PlaybackStatus     media.PlaybackStatus       `json:"playbackStatus"`
	SourceAvailability catalog.SourceAvailability `json:"sourceAvailability"`
	Thumbnail          thumbnailReferenceResponse `json:"thumbnail"`
}

type assetPageResponse struct {
	Items      []assetResponse `json:"items"`
	NextCursor *string         `json:"nextCursor"`
}

func registerCatalogRoutes(mux *http.ServeMux, service CatalogService) {
	mux.HandleFunc("GET /api/v1/libraries/{libraryId}/directories", func(
		writer http.ResponseWriter,
		request *http.Request,
	) {
		libraryID, err := parseResourceID(request.PathValue("libraryId"), "lib_")
		if err != nil {
			writeCatalogError(writer, request, catalog.ErrLibraryNotFound)
			return
		}
		query, err := parseDirectoryListQuery(request.URL.RawQuery)
		if err != nil {
			writeCatalogError(writer, request, err)
			return
		}
		query.LibraryID = libraryID
		page, err := service.ListDirectories(request.Context(), query)
		if err != nil {
			writeCatalogError(writer, request, err)
			return
		}
		items := make([]directoryResponse, 0, len(page.Items))
		for _, item := range page.Items {
			items = append(items, directoryWire(item))
		}
		var next *string
		if page.NextCursor != "" {
			next = &page.NextCursor
		}
		writeJSON(writer, http.StatusOK, directoryPageResponse{
			Items: items, NextCursor: next,
		})
	})

	mux.HandleFunc("GET /api/v1/directories/{directoryId}", func(
		writer http.ResponseWriter,
		request *http.Request,
	) {
		directoryID, err := parseResourceID(request.PathValue("directoryId"), "dir_")
		if err != nil {
			writeCatalogError(writer, request, catalog.ErrDirectoryNotFound)
			return
		}
		detail, err := service.GetDirectory(request.Context(), directoryID)
		if err != nil {
			writeCatalogError(writer, request, err)
			return
		}
		response := directoryDetailResponse{
			directoryResponse: directoryWire(detail.Directory),
			Breadcrumbs:       make([]breadcrumbResponse, 0, len(detail.Breadcrumbs)),
		}
		for _, item := range detail.Breadcrumbs {
			response.Breadcrumbs = append(response.Breadcrumbs, breadcrumbResponse{
				ID: directoryIDString(item.ID), Name: item.Name,
				RelativePath: item.RelativePath,
			})
		}
		writeJSON(writer, http.StatusOK, response)
	})

	mux.HandleFunc("GET /api/v1/libraries/{libraryId}/assets", func(
		writer http.ResponseWriter,
		request *http.Request,
	) {
		libraryID, err := parseResourceID(request.PathValue("libraryId"), "lib_")
		if err != nil {
			writeCatalogError(writer, request, catalog.ErrLibraryNotFound)
			return
		}
		query, err := parseAssetListQuery(request.URL.RawQuery)
		if err != nil {
			writeCatalogError(writer, request, err)
			return
		}
		query.LibraryID = libraryID
		page, err := service.ListAssets(request.Context(), query)
		if err != nil {
			writeCatalogError(writer, request, err)
			return
		}
		writeAssetPage(writer, page)
	})

	mux.HandleFunc("GET /api/v1/assets", func(
		writer http.ResponseWriter,
		request *http.Request,
	) {
		query, err := parseGlobalSearchQuery(request.URL.RawQuery)
		if err != nil {
			writeCatalogError(writer, request, err)
			return
		}
		page, err := service.SearchAssets(request.Context(), query)
		if err != nil {
			writeCatalogError(writer, request, err)
			return
		}
		writeAssetPage(writer, page)
	})

	mux.HandleFunc("GET /api/v1/assets/{assetId}", func(
		writer http.ResponseWriter,
		request *http.Request,
	) {
		assetID, err := parseResourceID(request.PathValue("assetId"), "ast_")
		if err != nil {
			writeCatalogError(writer, request, catalog.ErrAssetNotFound)
			return
		}
		item, err := service.GetAsset(request.Context(), assetID)
		if err != nil {
			writeCatalogError(writer, request, err)
			return
		}
		writeJSON(writer, http.StatusOK, assetWire(item))
	})
}

func writeAssetPage(writer http.ResponseWriter, page catalog.AssetPage) {
	items := make([]assetResponse, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, assetWire(item))
	}
	var next *string
	if page.NextCursor != "" {
		next = &page.NextCursor
	}
	writeJSON(writer, http.StatusOK, assetPageResponse{Items: items, NextCursor: next})
}

func parseAssetListQuery(raw string) (catalog.AssetRequest, error) {
	values, err := url.ParseQuery(raw)
	if err != nil {
		return catalog.AssetRequest{}, catalog.ErrInvalidQuery
	}
	allowed := map[string]bool{
		"directoryId": true, "recursive": true, "q": true, "kind": true,
		"modifiedFrom": true, "modifiedBefore": true,
		"sort": true, "order": true, "cursor": true, "limit": true,
	}
	for key, entries := range values {
		if !allowed[key] || len(entries) != 1 {
			return catalog.AssetRequest{}, catalog.ErrInvalidQuery
		}
	}
	var request catalog.AssetRequest
	if value, ok := values["directoryId"]; ok {
		request.DirectoryID, err = parseResourceID(value[0], "dir_")
		if err != nil {
			return catalog.AssetRequest{}, catalog.ErrInvalidQuery
		}
		request.DirectorySet = true
	}
	if value, ok := values["recursive"]; ok {
		if value[0] != "true" && value[0] != "false" {
			return catalog.AssetRequest{}, catalog.ErrInvalidQuery
		}
		request.Recursive = value[0] == "true"
		request.RecursiveSet = true
	}
	if value, ok := values["q"]; ok {
		request.SearchQuery = &value[0]
	}
	if value, ok := values["kind"]; ok {
		if value[0] == "" {
			return catalog.AssetRequest{}, catalog.ErrInvalidQuery
		}
		for _, item := range strings.Split(value[0], ",") {
			request.Kinds = append(request.Kinds, catalog.AssetKind(item))
		}
	}
	if value, ok := values["modifiedFrom"]; ok {
		request.ModifiedFromNS, err = parseUTCInstant(value[0])
		if err != nil {
			return catalog.AssetRequest{}, err
		}
	}
	if value, ok := values["modifiedBefore"]; ok {
		request.ModifiedBeforeNS, err = parseUTCInstant(value[0])
		if err != nil {
			return catalog.AssetRequest{}, err
		}
	}
	if value, ok := values["sort"]; ok {
		request.Sort = catalog.SortField(value[0])
	}
	if value, ok := values["order"]; ok {
		request.Order = catalog.SortOrder(value[0])
	}
	if value, ok := values["cursor"]; ok {
		if value[0] == "" {
			return catalog.AssetRequest{}, catalog.ErrInvalidCursor
		}
		request.Cursor = value[0]
	}
	if value, ok := values["limit"]; ok {
		request.Limit, err = strconv.Atoi(value[0])
		if err != nil || request.Limit < 1 || request.Limit > catalog.MaxPageSize {
			return catalog.AssetRequest{}, catalog.ErrInvalidQuery
		}
	}
	if request.SearchQuery != nil && !request.DirectorySet && request.RecursiveSet {
		return catalog.AssetRequest{}, catalog.ErrInvalidQuery
	}
	return request, nil
}

func parseGlobalSearchQuery(raw string) (catalog.GlobalSearchRequest, error) {
	parsed, err := parseAssetListQuery(raw)
	if err != nil {
		return catalog.GlobalSearchRequest{}, err
	}
	if parsed.SearchQuery == nil || parsed.DirectorySet || parsed.RecursiveSet {
		return catalog.GlobalSearchRequest{}, catalog.ErrInvalidQuery
	}
	return catalog.GlobalSearchRequest{
		SearchQuery: *parsed.SearchQuery, Kinds: parsed.Kinds,
		ModifiedFromNS: parsed.ModifiedFromNS, ModifiedBeforeNS: parsed.ModifiedBeforeNS,
		Sort: parsed.Sort, Order: parsed.Order, Cursor: parsed.Cursor, Limit: parsed.Limit,
	}, nil
}

func parseUTCInstant(value string) (*int64, error) {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, catalog.ErrInvalidQuery
	}
	_, offset := parsed.Zone()
	if offset != 0 {
		return nil, catalog.ErrInvalidQuery
	}
	nanoseconds := parsed.UnixNano()
	if !time.Unix(0, nanoseconds).UTC().Equal(parsed.UTC()) {
		return nil, catalog.ErrInvalidQuery
	}
	return &nanoseconds, nil
}

func parseDirectoryListQuery(raw string) (catalog.DirectoryRequest, error) {
	values, err := url.ParseQuery(raw)
	if err != nil {
		return catalog.DirectoryRequest{}, catalog.ErrInvalidQuery
	}
	for key, entries := range values {
		if (key != "parentId" && key != "cursor" && key != "limit") ||
			len(entries) != 1 {
			return catalog.DirectoryRequest{}, catalog.ErrInvalidQuery
		}
	}
	var request catalog.DirectoryRequest
	if value, ok := values["parentId"]; ok {
		request.ParentDirectoryID, err = parseResourceID(value[0], "dir_")
		if err != nil {
			return catalog.DirectoryRequest{}, catalog.ErrInvalidQuery
		}
	}
	if value, ok := values["cursor"]; ok {
		if value[0] == "" {
			return catalog.DirectoryRequest{}, catalog.ErrInvalidCursor
		}
		request.Cursor = value[0]
	}
	if value, ok := values["limit"]; ok {
		request.Limit, err = strconv.Atoi(value[0])
		if err != nil || request.Limit < 1 || request.Limit > catalog.MaxPageSize {
			return catalog.DirectoryRequest{}, catalog.ErrInvalidQuery
		}
	}
	return request, nil
}

func directoryWire(item catalog.Directory) directoryResponse {
	var parentID *string
	if item.ParentID != nil {
		value := directoryIDString(*item.ParentID)
		parentID = &value
	}
	return directoryResponse{
		ID: directoryIDString(item.ID), LibraryID: libraryID(item.LibraryID),
		ParentID: parentID, Name: item.Name, RelativePath: item.RelativePath,
		DirectAssetCount:    item.DirectAssetCount,
		RecursiveAssetCount: item.RecursiveAssetCount,
		HasChildren:         item.HasChildren,
	}
}

func directoryIDString(id int64) string {
	return "dir_" + strconv.FormatInt(id, 10)
}

func assetIDString(id int64) string {
	return "ast_" + strconv.FormatInt(id, 10)
}

func assetWire(item catalog.Asset) assetResponse {
	return assetResponse{
		ID: assetIDString(item.ID), LibraryID: libraryID(item.LibraryID),
		LibraryName: item.LibraryName, DirectoryID: directoryIDString(item.DirectoryID),
		Name: item.Name, RelativePath: item.RelativePath, Kind: item.Kind,
		MIMEType: item.MIMEType, SizeBytes: item.SizeBytes,
		Width: item.Width, Height: item.Height, DurationMS: item.DurationMS,
		ModifiedAt:  time.Unix(0, item.ModifiedAtNS).UTC().Format(time.RFC3339Nano),
		ProbeStatus: item.ProbeStatus, PlaybackStatus: item.PlaybackStatus,
		SourceAvailability: item.Availability,
		Thumbnail:          thumbnailReferenceWire(item),
	}
}

func thumbnailReferenceWire(item catalog.Asset) thumbnailReferenceResponse {
	if item.Availability == catalog.SourceOffline {
		code := "source_offline"
		return thumbnailReferenceResponse{
			Status: "unavailable", ErrorCode: &code,
		}
	}
	switch item.ThumbnailStatus {
	case "ready":
		value := "/api/v1/assets/" + assetIDString(item.ID) + "/thumbnail?variant=grid"
		return thumbnailReferenceResponse{Status: "ready", URL: &value}
	case "failed":
		var code *string
		if item.ThumbnailErrorCode != nil {
			value := publicThumbnailErrorCode(*item.ThumbnailErrorCode)
			code = &value
		}
		return thumbnailReferenceResponse{Status: "failed", ErrorCode: code}
	default:
		return thumbnailReferenceResponse{Status: "pending"}
	}
}

func publicThumbnailErrorCode(code media.ProcessingErrorCode) string {
	if code == media.ErrorProcessingFailed {
		return "thumbnail_failed"
	}
	return string(code)
}

func writeCatalogError(writer http.ResponseWriter, request *http.Request, err error) {
	switch {
	case errors.Is(err, catalog.ErrLibraryNotFound):
		writePublicError(writer, request, http.StatusNotFound, "library_not_found", "The media library was not found.")
	case errors.Is(err, catalog.ErrDirectoryNotFound):
		writePublicError(writer, request, http.StatusNotFound, "directory_not_found", "The directory was not found.")
	case errors.Is(err, catalog.ErrAssetNotFound):
		writePublicError(writer, request, http.StatusNotFound, "asset_not_found", "The media item was not found.")
	case errors.Is(err, catalog.ErrInvalidCursor):
		writePublicError(writer, request, http.StatusBadRequest, "invalid_cursor", "The pagination cursor is invalid.")
	case errors.Is(err, catalog.ErrInvalidQuery):
		writePublicError(writer, request, http.StatusBadRequest, codeInvalidRequest, "The request is invalid.")
	default:
		writeInternalError(writer, request)
	}
}
