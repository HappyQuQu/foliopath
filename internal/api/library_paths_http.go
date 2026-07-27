package api

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"

	"github.com/HappyQuQu/foliopath/internal/library"
)

const (
	codeInvalidCursor            = "invalid_cursor"
	codeLibraryRootUnavailable   = "library_root_unavailable"
	codeLibraryRootSymlink       = "library_root_symlink"
	codeLibraryRootMountBoundary = "library_root_mount_boundary"
)

type LibraryPathService interface {
	ListPaths(context.Context, library.ListPathParams) (library.PathPage, error)
}

type libraryPathLocationResponse struct {
	Name         string `json:"name"`
	RelativePath string `json:"relativePath"`
}

type libraryPathEntryResponse struct {
	Name                   string  `json:"name"`
	RelativePath           string  `json:"relativePath"`
	HasChildren            bool    `json:"hasChildren"`
	Selectable             bool    `json:"selectable"`
	SelectionBlockedReason *string `json:"selectionBlockedReason"`
	ConflictingLibraryID   *string `json:"conflictingLibraryId"`
	ConflictingLibraryName *string `json:"conflictingLibraryName"`
}

type libraryPathPageResponse struct {
	Location    libraryPathLocationResponse   `json:"location"`
	Breadcrumbs []libraryPathLocationResponse `json:"breadcrumbs"`
	Items       []libraryPathEntryResponse    `json:"items"`
	NextCursor  *string                       `json:"nextCursor"`
}

func registerLibraryPathRoutes(mux *http.ServeMux, service LibraryPathService) {
	mux.HandleFunc("GET /api/v1/library-paths", func(
		writer http.ResponseWriter,
		request *http.Request,
	) {
		params, err := parseLibraryPathQuery(request.URL.RawQuery)
		if err != nil {
			writeLibraryPathError(writer, request, err)
			return
		}
		page, err := service.ListPaths(request.Context(), params)
		if err != nil {
			writeLibraryPathError(writer, request, err)
			return
		}
		writeJSON(writer, http.StatusOK, libraryPathPageWire(page))
	})
}

func parseLibraryPathQuery(rawQuery string) (library.ListPathParams, error) {
	query, err := url.ParseQuery(rawQuery)
	if err != nil {
		return library.ListPathParams{}, library.ErrInvalidParent
	}
	for key, values := range query {
		switch key {
		case "parent", "cursor", "limit":
		default:
			return library.ListPathParams{}, library.ErrInvalidParent
		}
		if len(values) != 1 {
			return library.ListPathParams{}, library.ErrInvalidParent
		}
	}
	params := library.ListPathParams{
		Parent: query.Get("parent"),
		Cursor: query.Get("cursor"),
	}
	if cursorValues, exists := query["cursor"]; exists && cursorValues[0] == "" {
		return library.ListPathParams{}, library.ErrInvalidCursor
	}
	if limitValues, exists := query["limit"]; exists {
		limitText := limitValues[0]
		limit, err := strconv.Atoi(limitText)
		if err != nil || limit < 1 || limit > library.MaxPathPageSize {
			return library.ListPathParams{}, library.ErrInvalidParent
		}
		params.Limit = limit
	}
	return params, nil
}

func libraryPathPageWire(page library.PathPage) libraryPathPageResponse {
	breadcrumbs := make([]libraryPathLocationResponse, 0, len(page.Breadcrumbs))
	for _, breadcrumb := range page.Breadcrumbs {
		breadcrumbs = append(breadcrumbs, libraryPathLocationWire(breadcrumb))
	}
	items := make([]libraryPathEntryResponse, 0, len(page.Items))
	for _, item := range page.Items {
		var blockedReason *string
		if item.BlockedReason != "" {
			value := string(item.BlockedReason)
			blockedReason = &value
		}
		items = append(items, libraryPathEntryResponse{
			Name:                   item.Name,
			RelativePath:           item.RelativePath,
			HasChildren:            item.HasChildren,
			Selectable:             item.Selectable,
			SelectionBlockedReason: blockedReason,
			ConflictingLibraryID:   nil,
			ConflictingLibraryName: nil,
		})
	}
	var nextCursor *string
	if page.NextCursor != "" {
		nextCursor = &page.NextCursor
	}
	return libraryPathPageResponse{
		Location:    libraryPathLocationWire(page.Location),
		Breadcrumbs: breadcrumbs,
		Items:       items,
		NextCursor:  nextCursor,
	}
}

func libraryPathLocationWire(location library.PathLocation) libraryPathLocationResponse {
	return libraryPathLocationResponse{
		Name:         location.Name,
		RelativePath: location.RelativePath,
	}
}

func writeLibraryPathError(
	writer http.ResponseWriter,
	request *http.Request,
	err error,
) {
	switch {
	case errors.Is(err, library.ErrInvalidCursor):
		writePublicError(
			writer,
			request,
			http.StatusBadRequest,
			codeInvalidCursor,
			"The pagination cursor is invalid for this request.",
		)
	case errors.Is(err, library.ErrInvalidParent):
		writePublicError(
			writer,
			request,
			http.StatusBadRequest,
			codeInvalidRequest,
			"The requested library path is invalid.",
		)
	case errors.Is(err, library.ErrParentSymlink):
		writePublicError(
			writer,
			request,
			http.StatusConflict,
			codeLibraryRootSymlink,
			"The requested library path contains a symbolic link.",
		)
	case errors.Is(err, library.ErrParentMountBoundary):
		writePublicError(
			writer,
			request,
			http.StatusConflict,
			codeLibraryRootMountBoundary,
			"The requested library path crosses a filesystem boundary.",
		)
	case errors.Is(err, library.ErrParentUnavailable):
		writePublicError(
			writer,
			request,
			http.StatusConflict,
			codeLibraryRootUnavailable,
			"The requested library path is unavailable.",
		)
	default:
		writeInternalError(writer, request)
	}
}
