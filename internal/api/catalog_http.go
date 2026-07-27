package api

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"

	"github.com/HappyQuQu/foliopath/internal/catalog"
)

type CatalogService interface {
	ListDirectories(context.Context, catalog.DirectoryRequest) (catalog.DirectoryPage, error)
	GetDirectory(context.Context, int64) (catalog.DirectoryDetail, error)
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

func writeCatalogError(writer http.ResponseWriter, request *http.Request, err error) {
	switch {
	case errors.Is(err, catalog.ErrLibraryNotFound):
		writePublicError(writer, request, http.StatusNotFound, "library_not_found", "The media library was not found.")
	case errors.Is(err, catalog.ErrDirectoryNotFound):
		writePublicError(writer, request, http.StatusNotFound, "directory_not_found", "The directory was not found.")
	case errors.Is(err, catalog.ErrInvalidCursor):
		writePublicError(writer, request, http.StatusBadRequest, "invalid_cursor", "The pagination cursor is invalid.")
	case errors.Is(err, catalog.ErrInvalidQuery):
		writePublicError(writer, request, http.StatusBadRequest, codeInvalidRequest, "The request is invalid.")
	default:
		writeInternalError(writer, request)
	}
}
