package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/HappyQuQu/foliopath/internal/library"
	"github.com/HappyQuQu/foliopath/internal/scanner"
)

const maxLibraryRequestBodyBytes = 16 << 10

type LibraryLifecycleService interface {
	Create(context.Context, string, string, string) (library.CreateResult, error)
	List(context.Context, string, int) (library.Page, error)
	Get(context.Context, int64) (library.Details, error)
	Rename(context.Context, int64, int64, string) (library.Details, error)
	Remove(context.Context, int64, int64, string) (library.RemoveResult, error)
	GetRemoval(context.Context, int64) (library.Removal, error)
}

type ScanAdmissionService interface {
	RequestManual(context.Context, int64) (scanner.AdmissionResult, error)
}

type ScanQueryService interface {
	List(context.Context, int64, string, int) (scanner.Page, error)
	Get(context.Context, int64) (scanner.Details, error)
	Cancel(context.Context, int64) (scanner.ScanRun, error)
}

type createLibraryRequest struct {
	Name     string `json:"name"`
	RootPath string `json:"rootPath"`
}

type renameLibraryRequest struct {
	Name string `json:"name"`
}

type libraryResponse struct {
	ID                   string  `json:"id"`
	Name                 string  `json:"name"`
	RootPath             string  `json:"rootPath"`
	DisplayPath          string  `json:"displayPath"`
	Status               string  `json:"status"`
	LastSuccessfulScanAt *string `json:"lastSuccessfulScanAt"`
	LatestScanID         *string `json:"latestScanId"`
	AssetCount           int64   `json:"assetCount"`
	DirectoryCount       int64   `json:"directoryCount"`
	CreatedAt            string  `json:"createdAt"`
	UpdatedAt            string  `json:"updatedAt"`
}

type scanResponse struct {
	ID                    string              `json:"id"`
	LibraryID             string              `json:"libraryId"`
	Trigger               string              `json:"trigger"`
	Status                string              `json:"status"`
	Phase                 string              `json:"phase"`
	Generation            int64               `json:"generation"`
	DiscoveredDirectories int64               `json:"discoveredDirectories"`
	DiscoveredAssets      int64               `json:"discoveredAssets"`
	ProcessedAssets       int64               `json:"processedAssets"`
	SkippedDirectories    int64               `json:"skippedDirectories"`
	SkippedFiles          int64               `json:"skippedFiles"`
	ErrorCount            int64               `json:"errorCount"`
	Issues                []scanIssueResponse `json:"issues"`
	IssuesTruncated       bool                `json:"issuesTruncated"`
	ErrorCode             *string             `json:"errorCode"`
	ProgressRatio         *float64            `json:"progressRatio"`
	CreatedAt             string              `json:"createdAt"`
	StartedAt             *string             `json:"startedAt"`
	FinishedAt            *string             `json:"finishedAt"`
	CancelRequestedAt     *string             `json:"cancelRequestedAt"`
	CanCancel             bool                `json:"canCancel"`
}

type createLibraryResponse struct {
	Library libraryResponse `json:"library"`
	Scan    scanResponse    `json:"scan"`
}

type libraryPageResponse struct {
	Items      []libraryResponse `json:"items"`
	NextCursor *string           `json:"nextCursor"`
}

type removalResponse struct {
	ID          string  `json:"id"`
	LibraryID   string  `json:"libraryId"`
	LibraryName string  `json:"libraryName"`
	Status      string  `json:"status"`
	CreatedAt   string  `json:"createdAt"`
	StartedAt   *string `json:"startedAt"`
	FinishedAt  *string `json:"finishedAt"`
	ErrorCode   *string `json:"errorCode"`
}

func registerLibraryRoutes(mux *http.ServeMux, service LibraryLifecycleService) {
	mux.HandleFunc("GET /api/v1/libraries", func(writer http.ResponseWriter, request *http.Request) {
		cursor, limit, err := parseLibraryListQuery(request.URL.RawQuery)
		if err != nil {
			writeLibraryError(writer, request, err)
			return
		}
		page, err := service.List(request.Context(), cursor, limit)
		if err != nil {
			writeLibraryError(writer, request, err)
			return
		}
		items := make([]libraryResponse, 0, len(page.Items))
		for _, item := range page.Items {
			items = append(items, libraryWire(item))
		}
		var next *string
		if page.NextCursor != "" {
			next = &page.NextCursor
		}
		writeJSON(writer, http.StatusOK, libraryPageResponse{Items: items, NextCursor: next})
	})

	mux.HandleFunc("POST /api/v1/libraries", func(writer http.ResponseWriter, request *http.Request) {
		var payload createLibraryRequest
		if err := decodeLibraryJSON(writer, request, &payload); err != nil {
			writeLibraryError(writer, request, err)
			return
		}
		result, err := service.Create(
			request.Context(),
			payload.Name,
			payload.RootPath,
			request.Header.Get("Idempotency-Key"),
		)
		if err != nil {
			writeLibraryError(writer, request, err)
			return
		}
		writer.Header().Set("Location", "/api/v1/libraries/"+libraryID(result.Library.ID))
		writer.Header().Set("ETag", libraryETag(result.Library))
		writer.Header().Set("Idempotency-Replayed", strconv.FormatBool(result.Replayed))
		writeJSON(writer, http.StatusCreated, createLibraryResponse{
			Library: libraryWire(result.Library),
			Scan:    scanWire(result.Scan),
		})
	})

	mux.HandleFunc("GET /api/v1/libraries/{libraryId}", func(writer http.ResponseWriter, request *http.Request) {
		id, err := parseResourceID(request.PathValue("libraryId"), "lib_")
		if err != nil {
			writeLibraryError(writer, request, library.ErrNotFound)
			return
		}
		item, err := service.Get(request.Context(), id)
		if err != nil {
			writeLibraryError(writer, request, err)
			return
		}
		writer.Header().Set("ETag", libraryETag(item))
		writeJSON(writer, http.StatusOK, libraryWire(item))
	})

	mux.HandleFunc("PATCH /api/v1/libraries/{libraryId}", func(writer http.ResponseWriter, request *http.Request) {
		id, err := parseResourceID(request.PathValue("libraryId"), "lib_")
		if err != nil {
			writeLibraryError(writer, request, library.ErrNotFound)
			return
		}
		revision, err := parseLibraryIfMatch(request.Header.Get("If-Match"), id)
		if err != nil {
			writeLibraryError(writer, request, err)
			return
		}
		var payload renameLibraryRequest
		if err := decodeLibraryJSON(writer, request, &payload); err != nil {
			writeLibraryError(writer, request, err)
			return
		}
		item, err := service.Rename(request.Context(), id, revision, payload.Name)
		if err != nil {
			writeLibraryError(writer, request, err)
			return
		}
		writer.Header().Set("ETag", libraryETag(item))
		writeJSON(writer, http.StatusOK, libraryWire(item))
	})

	mux.HandleFunc("DELETE /api/v1/libraries/{libraryId}", func(writer http.ResponseWriter, request *http.Request) {
		id, err := parseResourceID(request.PathValue("libraryId"), "lib_")
		if err != nil {
			writeLibraryError(writer, request, library.ErrNotFound)
			return
		}
		revision, err := parseLibraryIfMatch(request.Header.Get("If-Match"), id)
		if err != nil {
			writeLibraryError(writer, request, err)
			return
		}
		result, err := service.Remove(
			request.Context(),
			id,
			revision,
			request.Header.Get("Idempotency-Key"),
		)
		if err != nil {
			writeLibraryError(writer, request, err)
			return
		}
		writer.Header().Set("Location", "/api/v1/library-removals/"+removalID(result.Removal.ID))
		writer.Header().Set("Idempotency-Replayed", strconv.FormatBool(result.Replayed))
		writeJSON(writer, http.StatusAccepted, removalWire(result.Removal))
	})

	mux.HandleFunc("GET /api/v1/library-removals/{removalId}", func(writer http.ResponseWriter, request *http.Request) {
		id, err := parseResourceID(request.PathValue("removalId"), "rmv_")
		if err != nil {
			writeLibraryError(writer, request, library.ErrRemovalNotFound)
			return
		}
		removal, err := service.GetRemoval(request.Context(), id)
		if err != nil {
			writeLibraryError(writer, request, err)
			return
		}
		etag := removalETag(removal)
		writer.Header().Set("ETag", etag)
		if request.Header.Get("If-None-Match") == etag {
			writer.WriteHeader(http.StatusNotModified)
			return
		}
		writeJSON(writer, http.StatusOK, removalWire(removal))
	})
}

func registerScanAdmissionRoute(mux *http.ServeMux, service ScanAdmissionService) {
	mux.HandleFunc("POST /api/v1/libraries/{libraryId}/scans", func(
		writer http.ResponseWriter,
		request *http.Request,
	) {
		id, err := parseResourceID(request.PathValue("libraryId"), "lib_")
		if err != nil {
			writeScanAdmissionError(writer, request, scanner.ErrLibraryNotFound)
			return
		}
		result, err := service.RequestManual(request.Context(), id)
		if err != nil {
			writeScanAdmissionError(writer, request, err)
			return
		}
		writer.Header().Set("Location", "/api/v1/scans/"+scanID(result.Run.ID))
		writer.Header().Set("ETag", scanETag(result.Run))
		status := http.StatusAccepted
		if result.Coalesced {
			status = http.StatusOK
		}
		writeJSON(writer, status, admissionScanWire(result.Run))
	})
}

func parseLibraryListQuery(raw string) (string, int, error) {
	query, err := url.ParseQuery(raw)
	if err != nil {
		return "", 0, errors.New("invalid library request")
	}
	for key, values := range query {
		if (key != "cursor" && key != "limit") || len(values) != 1 {
			return "", 0, errors.New("invalid library request")
		}
	}
	cursor := query.Get("cursor")
	if values, ok := query["cursor"]; ok && values[0] == "" {
		return "", 0, library.ErrInvalidLibraryCursor
	}
	limit := 0
	if text, ok := query["limit"]; ok {
		limit, err = strconv.Atoi(text[0])
		if err != nil || limit < 1 || limit > library.MaxLibraryPageSize {
			return "", 0, errors.New("invalid library request")
		}
	}
	return cursor, limit, nil
}

func decodeLibraryJSON(writer http.ResponseWriter, request *http.Request, target any) error {
	contentType, _, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if err != nil || contentType != "application/json" {
		return errors.New("invalid library request")
	}
	request.Body = http.MaxBytesReader(writer, request.Body, maxLibraryRequestBodyBytes)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return errors.New("invalid library request")
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("invalid library request")
	}
	return nil
}

func parseResourceID(value, prefix string) (int64, error) {
	if !strings.HasPrefix(value, prefix) {
		return 0, library.ErrInvalidID
	}
	id, err := strconv.ParseInt(strings.TrimPrefix(value, prefix), 10, 64)
	if err != nil || id <= 0 || value != prefix+strconv.FormatInt(id, 10) {
		return 0, library.ErrInvalidID
	}
	return id, nil
}

func parseLibraryIfMatch(value string, id int64) (int64, error) {
	if value == "" {
		return 0, errPreconditionRequired
	}
	prefix := `"` + libraryID(id) + "-r"
	if !strings.HasPrefix(value, prefix) || !strings.HasSuffix(value, `"`) {
		return 0, library.ErrPreconditionFailed
	}
	revision, err := strconv.ParseInt(strings.TrimSuffix(strings.TrimPrefix(value, prefix), `"`), 10, 64)
	if err != nil || revision <= 0 {
		return 0, library.ErrPreconditionFailed
	}
	return revision, nil
}

var errPreconditionRequired = errors.New("precondition required")

func libraryID(id int64) string { return "lib_" + strconv.FormatInt(id, 10) }
func scanID(id int64) string    { return "scan_" + strconv.FormatInt(id, 10) }
func removalID(id int64) string { return "rmv_" + strconv.FormatInt(id, 10) }

func libraryETag(item library.Details) string {
	return fmt.Sprintf(`"%s-r%d"`, libraryID(item.ID), item.Revision)
}

func removalETag(item library.Removal) string {
	return fmt.Sprintf(`"%s-r%d"`, removalID(item.ID), item.Revision)
}

func scanETag(item scanner.ScanRun) string {
	return fmt.Sprintf(`"%s-r%d"`, scanID(item.ID), item.Revision)
}

func libraryWire(item library.Details) libraryResponse {
	var lastSuccess *string
	if item.LastSuccessfulScanAtMS != nil {
		value := timestamp(*item.LastSuccessfulScanAtMS)
		lastSuccess = &value
	}
	var latestScan *string
	if item.LatestScanID != nil {
		value := scanID(*item.LatestScanID)
		latestScan = &value
	}
	displayPath := "/library"
	if item.RootRelativePath != "" {
		displayPath += "/" + item.RootRelativePath
	}
	return libraryResponse{
		ID:                   libraryID(item.ID),
		Name:                 item.Name,
		RootPath:             item.RootRelativePath,
		DisplayPath:          displayPath,
		Status:               string(item.Status),
		LastSuccessfulScanAt: lastSuccess,
		LatestScanID:         latestScan,
		AssetCount:           item.AssetCount,
		DirectoryCount:       item.DirectoryCount,
		CreatedAt:            timestamp(item.CreatedAtMS),
		UpdatedAt:            timestamp(item.UpdatedAtMS),
	}
}

func scanWire(item library.Scan) scanResponse {
	var errorCode *string
	if item.ErrorCode != "" {
		errorCode = &item.ErrorCode
	}
	return scanResponse{
		ID:                    scanID(item.ID),
		LibraryID:             libraryID(item.LibraryID),
		Trigger:               item.Trigger,
		Status:                item.Status,
		Phase:                 item.Phase,
		Generation:            item.Generation,
		DiscoveredDirectories: item.DiscoveredDirectories,
		DiscoveredAssets:      item.DiscoveredAssets,
		ProcessedAssets:       item.ProcessedAssets,
		SkippedDirectories:    item.SkippedDirectories,
		SkippedFiles:          item.SkippedFiles,
		ErrorCount:            item.ErrorCount,
		Issues:                []scanIssueResponse{},
		IssuesTruncated:       item.IssuesTruncated,
		ErrorCode:             errorCode,
		CreatedAt:             timestamp(item.CreatedAtMS),
		StartedAt:             optionalTimestamp(item.StartedAtMS),
		FinishedAt:            optionalTimestamp(item.FinishedAtMS),
		CanCancel:             item.Status == "queued" || item.Status == "running",
	}
}

func admissionScanWire(item scanner.ScanRun) scanResponse {
	var errorCode *string
	if item.ErrorCode != "" {
		errorCode = &item.ErrorCode
	}
	return scanResponse{
		ID:                    scanID(item.ID),
		LibraryID:             libraryID(item.LibraryID),
		Trigger:               string(item.Trigger),
		Status:                string(item.Status),
		Phase:                 item.Phase,
		Generation:            item.Generation,
		DiscoveredDirectories: item.DiscoveredDirectories,
		DiscoveredAssets:      item.DiscoveredAssets,
		ProcessedAssets:       item.ProcessedAssets,
		SkippedDirectories:    item.SkippedDirectories,
		SkippedFiles:          item.SkippedFiles,
		ErrorCount:            item.ErrorCount,
		Issues:                []scanIssueResponse{},
		IssuesTruncated:       item.IssuesTruncated,
		ErrorCode:             errorCode,
		CreatedAt:             timestamp(item.CreatedAtMS),
		StartedAt:             optionalTimestamp(item.StartedAtMS),
		FinishedAt:            optionalTimestamp(item.FinishedAtMS),
		CancelRequestedAt:     optionalTimestamp(item.CancelRequestedAtMS),
		CanCancel:             item.Status == scanner.RunStatusQueued || item.Status == scanner.RunStatusRunning,
	}
}

func removalWire(item library.Removal) removalResponse {
	var errorCode *string
	if item.ErrorCode != "" {
		errorCode = &item.ErrorCode
	}
	return removalResponse{
		ID:          removalID(item.ID),
		LibraryID:   libraryID(item.LibraryID),
		LibraryName: item.LibraryName,
		Status:      string(item.Status),
		CreatedAt:   timestamp(item.CreatedAtMS),
		StartedAt:   optionalTimestamp(item.StartedAtMS),
		FinishedAt:  optionalTimestamp(item.FinishedAtMS),
		ErrorCode:   errorCode,
	}
}

func timestamp(milliseconds int64) string {
	return time.UnixMilli(milliseconds).UTC().Format(time.RFC3339Nano)
}

func optionalTimestamp(milliseconds *int64) *string {
	if milliseconds == nil {
		return nil
	}
	value := timestamp(*milliseconds)
	return &value
}

func writeLibraryError(writer http.ResponseWriter, request *http.Request, err error) {
	status, code, message := http.StatusInternalServerError, codeInternalError, "An unexpected error occurred."
	switch {
	case errors.Is(err, errPreconditionRequired):
		status, code, message = http.StatusPreconditionRequired, "precondition_required", "A current resource validator is required."
	case errors.Is(err, library.ErrPreconditionFailed):
		status, code, message = http.StatusPreconditionFailed, "precondition_failed", "The resource has changed."
	case errors.Is(err, library.ErrNotFound):
		status, code, message = http.StatusNotFound, "library_not_found", "The media library was not found."
	case errors.Is(err, library.ErrRemovalNotFound):
		status, code, message = http.StatusNotFound, "removal_not_found", "The library removal was not found."
	case errors.Is(err, library.ErrInvalidLibraryCursor):
		status, code, message = http.StatusBadRequest, "invalid_cursor", "The pagination cursor is invalid."
	case errors.Is(err, library.ErrInvalidName), errors.Is(err, library.ErrInvalidRoot):
		status, code, message = http.StatusUnprocessableEntity, "validation_failed", "One or more media-library fields are invalid."
	case errors.Is(err, library.ErrInvalidIdempotencyKey):
		status, code, message = http.StatusBadRequest, codeInvalidRequest, "The idempotency key is invalid."
	case errors.Is(err, library.ErrNameExists):
		status, code, message = http.StatusConflict, "library_name_conflict", "The media-library name is already in use."
	case errors.Is(err, library.ErrRootOverlap):
		status, code, message = http.StatusConflict, "library_path_overlap", "The media-library path overlaps another library."
	case errors.Is(err, library.ErrRootUnavailable):
		status, code, message = http.StatusConflict, codeLibraryRootUnavailable, "The selected media-library root is unavailable."
	case errors.Is(err, library.ErrRootSymlink):
		status, code, message = http.StatusConflict, codeLibraryRootSymlink, "The selected media-library root contains a symbolic link."
	case errors.Is(err, library.ErrRootMountBoundary):
		status, code, message = http.StatusConflict, codeLibraryRootMountBoundary, "The selected media-library root crosses a filesystem boundary."
	case errors.Is(err, library.ErrRootOutsideAllowed):
		status, code, message = http.StatusConflict, "library_root_outside_allowed", "The selected media-library root is outside the allowed root."
	case errors.Is(err, library.ErrIdempotencyConflict), errors.Is(err, library.ErrRemovalActive):
		status, code, message = http.StatusConflict, "idempotency_conflict", "The requested operation conflicts with an existing operation."
	case errors.Is(err, library.ErrScanCapacity):
		status, code, message = http.StatusTooManyRequests, "rate_limited", "The scan queue is at capacity."
	default:
		if strings.Contains(err.Error(), "invalid library request") {
			status, code, message = http.StatusBadRequest, codeInvalidRequest, "The request is invalid."
		}
	}
	writePublicError(writer, request, status, code, message)
}

func writeScanAdmissionError(
	writer http.ResponseWriter,
	request *http.Request,
	err error,
) {
	switch {
	case errors.Is(err, scanner.ErrLibraryNotFound):
		writePublicError(
			writer, request, http.StatusNotFound,
			"library_not_found", "The media library was not found.",
		)
	case errors.Is(err, scanner.ErrAdmissionConflict):
		writePublicError(
			writer, request, http.StatusConflict,
			"idempotency_conflict", "The media library is being removed.",
		)
	case errors.Is(err, scanner.ErrAdmissionCapacity):
		writePublicError(
			writer, request, http.StatusTooManyRequests,
			"rate_limited", "The scan queue is at capacity.",
		)
	default:
		writeInternalError(writer, request)
	}
}
