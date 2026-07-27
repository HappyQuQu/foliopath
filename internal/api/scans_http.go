package api

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"

	"github.com/HappyQuQu/foliopath/internal/scanner"
)

type scanIssueResponse struct {
	Code               string  `json:"code"`
	Message            string  `json:"message"`
	Count              int64   `json:"count"`
	SampleRelativePath *string `json:"sampleRelativePath"`
}

type scanPageResponse struct {
	Items      []scanResponse `json:"items"`
	NextCursor *string        `json:"nextCursor"`
}

var errInvalidScanRequest = errors.New("invalid scan request")

func registerScanQueryRoutes(mux *http.ServeMux, service ScanQueryService) {
	mux.HandleFunc("GET /api/v1/libraries/{libraryId}/scans", func(writer http.ResponseWriter, request *http.Request) {
		libraryID, err := parseResourceID(request.PathValue("libraryId"), "lib_")
		if err != nil {
			writeScanQueryError(writer, request, scanner.ErrLibraryNotFound)
			return
		}
		cursor, limit, err := parseScanListQuery(request.URL.RawQuery)
		if err != nil {
			writeScanQueryError(writer, request, err)
			return
		}
		page, err := service.List(request.Context(), libraryID, cursor, limit)
		if err != nil {
			writeScanQueryError(writer, request, err)
			return
		}
		items := make([]scanResponse, 0, len(page.Items))
		for _, item := range page.Items {
			items = append(items, scanDetailsWire(item))
		}
		var next *string
		if page.NextCursor != "" {
			next = &page.NextCursor
		}
		writeJSON(writer, http.StatusOK, scanPageResponse{Items: items, NextCursor: next})
	})

	mux.HandleFunc("GET /api/v1/scans/{scanId}", func(writer http.ResponseWriter, request *http.Request) {
		id, err := parseResourceID(request.PathValue("scanId"), "scan_")
		if err != nil {
			writeScanQueryError(writer, request, scanner.ErrScanRunNotFound)
			return
		}
		details, err := service.Get(request.Context(), id)
		if err != nil {
			writeScanQueryError(writer, request, err)
			return
		}
		etag := scanETag(details.Run)
		writer.Header().Set("ETag", etag)
		if request.Header.Get("If-None-Match") == etag {
			writer.WriteHeader(http.StatusNotModified)
			return
		}
		writeJSON(writer, http.StatusOK, scanDetailsWire(details))
	})

	mux.HandleFunc("POST /api/v1/scans/{scanId}/cancel", func(writer http.ResponseWriter, request *http.Request) {
		id, err := parseResourceID(request.PathValue("scanId"), "scan_")
		if err != nil {
			writeScanQueryError(writer, request, scanner.ErrScanRunNotFound)
			return
		}
		run, err := service.Cancel(request.Context(), id)
		if err != nil {
			writeScanQueryError(writer, request, err)
			return
		}
		writer.Header().Set("Location", "/api/v1/scans/"+scanID(run.ID))
		writer.Header().Set("ETag", scanETag(run))
		writeJSON(writer, http.StatusAccepted, admissionScanWire(run))
	})
}

func parseScanListQuery(raw string) (string, int, error) {
	query, err := url.ParseQuery(raw)
	if err != nil {
		return "", 0, errInvalidScanRequest
	}
	for key, values := range query {
		if (key != "cursor" && key != "limit") || len(values) != 1 {
			return "", 0, errInvalidScanRequest
		}
	}
	cursor := query.Get("cursor")
	if values, ok := query["cursor"]; ok && values[0] == "" {
		return "", 0, scanner.ErrInvalidScanCursor
	}
	limit := 0
	if value, ok := query["limit"]; ok {
		limit, err = strconv.Atoi(value[0])
		if err != nil || limit < 1 || limit > scanner.MaxScanPageSize {
			return "", 0, errInvalidScanRequest
		}
	}
	return cursor, limit, nil
}

func scanDetailsWire(details scanner.Details) scanResponse {
	response := admissionScanWire(details.Run)
	response.Issues = make([]scanIssueResponse, 0, len(details.Issues))
	for _, issue := range details.Issues {
		response.Issues = append(response.Issues, scanIssueResponse{
			Code: issue.Code, Message: scanIssueMessage(issue.Code),
			Count: issue.Count, SampleRelativePath: issue.SampleRelativePath,
		})
	}
	return response
}

func scanIssueMessage(code string) string {
	switch code {
	case "unreadable_directory":
		return "A directory could not be read."
	case "unsupported_file":
		return "An unsupported file was skipped."
	case "invalid_media":
		return "A media file is invalid."
	case "media_probe_failed":
		return "Media inspection failed."
	case "symlink_skipped":
		return "A symbolic link was skipped."
	case "maintained_directory_skipped":
		return "A maintained system directory was skipped."
	case "source_changed":
		return "A source file changed during processing."
	default:
		return "A media I/O error occurred."
	}
}

func writeScanQueryError(writer http.ResponseWriter, request *http.Request, err error) {
	switch {
	case errors.Is(err, scanner.ErrLibraryNotFound):
		writePublicError(writer, request, http.StatusNotFound, "library_not_found", "The media library was not found.")
	case errors.Is(err, scanner.ErrScanRunNotFound):
		writePublicError(writer, request, http.StatusNotFound, "scan_not_found", "The scan was not found.")
	case errors.Is(err, scanner.ErrInvalidScanCursor):
		writePublicError(writer, request, http.StatusBadRequest, "invalid_cursor", "The pagination cursor is invalid.")
	case errors.Is(err, scanner.ErrScanAlreadyFinished):
		writePublicError(writer, request, http.StatusConflict, "scan_already_finished", "The scan has already finished.")
	default:
		if errors.Is(err, errInvalidScanRequest) {
			writePublicError(writer, request, http.StatusBadRequest, codeInvalidRequest, "The request is invalid.")
			return
		}
		writeInternalError(writer, request)
	}
}
