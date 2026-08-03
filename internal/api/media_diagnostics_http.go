package api

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/HappyQuQu/foliopath/internal/thumbnail"
)

type MediaDiagnosticsService interface {
	ListFailures(context.Context, thumbnail.FailureQuery) ([]thumbnail.MediaFailure, error)
	LatestFailureRevision(context.Context, thumbnail.FailureQuery) (thumbnail.FailureRevision, bool, error)
	GetFailure(context.Context, int64) (thumbnail.MediaFailure, error)
	ProcessMedia(context.Context, int64, thumbnail.RequeueMode, int) (thumbnail.RetrySummary, error)
}

type mediaFailureResponse struct {
	ID            string                       `json:"id"`
	LibraryID     string                       `json:"libraryId"`
	LibraryName   string                       `json:"libraryName"`
	AssetID       string                       `json:"assetId"`
	RelativePath  string                       `json:"relativePath"`
	Variant       string                       `json:"variant"`
	ErrorCode     string                       `json:"errorCode"`
	Attempts      int                          `json:"attempts"`
	FinishedAt    string                       `json:"finishedAt"`
	LatestAttempt *mediaFailureAttemptResponse `json:"latestAttempt,omitempty"`
}

type mediaFailureAttemptResponse struct {
	AttemptNumber int     `json:"attemptNumber"`
	Outcome       string  `json:"outcome"`
	Stage         *string `json:"stage"`
	ReasonCode    *string `json:"reasonCode"`
	Tool          *string `json:"tool"`
	ExitCode      *int    `json:"exitCode"`
	DurationMS    int64   `json:"durationMs"`
	FinishedAt    string  `json:"finishedAt"`
}

type mediaFailureDetailResponse struct {
	Failure        mediaFailureResponse          `json:"failure"`
	AttemptHistory []mediaFailureAttemptResponse `json:"attemptHistory"`
}

type mediaFailurePageResponse struct {
	Items      []mediaFailureResponse `json:"items"`
	NextCursor *string                `json:"nextCursor"`
	Revision   *string                `json:"revision"`
}

type retryMediaFailuresResponse struct {
	Requeued          int64 `json:"requeued"`
	RemainingEligible int64 `json:"remainingEligible"`
	PermanentFailures int64 `json:"permanentFailures"`
}

func registerMediaDiagnosticsRoutes(mux *http.ServeMux, service MediaDiagnosticsService) {
	mux.HandleFunc("GET /api/v1/diagnostics/media-failures", func(writer http.ResponseWriter, request *http.Request) {
		query, err := parseMediaFailureQuery(request.URL.RawQuery)
		if err != nil {
			writePublicError(writer, request, http.StatusUnprocessableEntity,
				"invalid_request", "The diagnostics query is invalid.")
			return
		}
		failures, err := service.ListFailures(request.Context(), query)
		if err != nil {
			writeMediaDiagnosticsError(writer, request, err)
			return
		}
		revision, found, err := service.LatestFailureRevision(request.Context(), query)
		if err != nil {
			writeMediaDiagnosticsError(writer, request, err)
			return
		}
		items := make([]mediaFailureResponse, 0, len(failures))
		for _, failure := range failures {
			items = append(items, mediaFailureWire(failure))
		}
		var next *string
		if len(failures) == query.Limit {
			value := mediaJobID(failures[len(failures)-1].JobID)
			next = &value
		}
		var revisionWire *string
		if found {
			value := mediaFailureRevision(revision)
			revisionWire = &value
		}
		writeJSON(writer, http.StatusOK, mediaFailurePageResponse{
			Items: items, NextCursor: next, Revision: revisionWire,
		})
	})

	mux.HandleFunc("GET /api/v1/diagnostics/media-failures/{jobId}", func(writer http.ResponseWriter, request *http.Request) {
		jobID, err := parseResourceID(request.PathValue("jobId"), "mjob_")
		if err != nil {
			writePublicError(writer, request, http.StatusNotFound,
				"media_failure_not_found", "The media failure was not found.")
			return
		}
		failure, err := service.GetFailure(request.Context(), jobID)
		if err != nil {
			writeMediaDiagnosticsError(writer, request, err)
			return
		}
		attempts := make([]mediaFailureAttemptResponse, 0, len(failure.AttemptHistory))
		for _, attempt := range failure.AttemptHistory {
			attempts = append(attempts, mediaFailureAttemptWire(attempt))
		}
		writeJSON(writer, http.StatusOK, mediaFailureDetailResponse{
			Failure: mediaFailureWire(failure), AttemptHistory: attempts,
		})
	})

	mux.HandleFunc("POST /api/v1/libraries/{libraryId}/media-processing/repair", func(writer http.ResponseWriter, request *http.Request) {
		libraryID, err := parseResourceID(request.PathValue("libraryId"), "lib_")
		if err != nil {
			writePublicError(writer, request, http.StatusNotFound,
				"library_not_found", "The media library was not found.")
			return
		}
		mode, err := parseMediaProcessingMode(request.URL.Query().Get("mode"))
		if err != nil {
			writePublicError(writer, request, http.StatusUnprocessableEntity,
				"invalid_request", "The media processing mode is invalid.")
			return
		}
		summary, err := service.ProcessMedia(request.Context(), libraryID, mode, 256)
		if err != nil {
			writeMediaDiagnosticsError(writer, request, err)
			return
		}
		writeJSON(writer, http.StatusAccepted, retryMediaFailuresResponse{
			Requeued: summary.Requeued, RemainingEligible: summary.RemainingEligible,
			PermanentFailures: summary.PermanentFailures,
		})
	})
}

func parseMediaProcessingMode(value string) (thumbnail.RequeueMode, error) {
	if value == "" || value == string(thumbnail.RequeueMissing) {
		return thumbnail.RequeueMissing, nil
	}
	if value == string(thumbnail.RequeueAll) {
		return thumbnail.RequeueAll, nil
	}
	return "", thumbnail.ErrInvalidDiagnosticsRequest
}

func parseMediaFailureQuery(raw string) (thumbnail.FailureQuery, error) {
	values, err := url.ParseQuery(raw)
	if err != nil {
		return thumbnail.FailureQuery{}, thumbnail.ErrInvalidDiagnosticsRequest
	}
	for key, entries := range values {
		if (key != "libraryId" && key != "variant" && key != "errorCode" &&
			key != "cursor" && key != "limit") || len(entries) != 1 {
			return thumbnail.FailureQuery{}, thumbnail.ErrInvalidDiagnosticsRequest
		}
	}
	query := thumbnail.FailureQuery{Limit: 50}
	if value := values.Get("libraryId"); value != "" {
		query.LibraryID, err = parseResourceID(value, "lib_")
		if err != nil {
			return thumbnail.FailureQuery{}, thumbnail.ErrInvalidDiagnosticsRequest
		}
	}
	if value := values.Get("variant"); value != "" {
		query.Variant = thumbnail.Variant(value)
	}
	if value := values.Get("errorCode"); value != "" {
		query.ErrorCode = thumbnail.JobErrorCode(value)
	}
	if value := values.Get("cursor"); value != "" {
		query.BeforeID, err = parseResourceID(value, "mjob_")
		if err != nil {
			return thumbnail.FailureQuery{}, thumbnail.ErrInvalidDiagnosticsRequest
		}
	}
	if value := values.Get("limit"); value != "" {
		query.Limit, err = strconv.Atoi(value)
		if err != nil {
			return thumbnail.FailureQuery{}, thumbnail.ErrInvalidDiagnosticsRequest
		}
	}
	return query, nil
}

func mediaFailureWire(failure thumbnail.MediaFailure) mediaFailureResponse {
	response := mediaFailureResponse{
		ID: mediaJobID(failure.JobID), LibraryID: libraryID(failure.LibraryID),
		LibraryName: failure.LibraryName, AssetID: assetID(failure.AssetID),
		RelativePath: failure.RelativePath, Variant: string(failure.Variant),
		ErrorCode: string(failure.ErrorCode), Attempts: failure.AttemptCount,
		FinishedAt: time.UnixMilli(failure.FinishedAtMS).UTC().Format(time.RFC3339Nano),
	}
	if failure.LatestAttempt != nil {
		attempt := mediaFailureAttemptWire(*failure.LatestAttempt)
		response.LatestAttempt = &attempt
	}
	return response
}

func mediaFailureAttemptWire(attempt thumbnail.MediaFailureAttempt) mediaFailureAttemptResponse {
	response := mediaFailureAttemptResponse{
		AttemptNumber: attempt.AttemptNumber, Outcome: string(attempt.Outcome),
		ExitCode: attempt.ExitCode, DurationMS: attempt.DurationMS,
		FinishedAt: time.UnixMilli(attempt.FinishedAtMS).UTC().Format(time.RFC3339Nano),
	}
	if attempt.Stage != "" {
		value := string(attempt.Stage)
		response.Stage = &value
	}
	if attempt.Reason != "" {
		value := string(attempt.Reason)
		response.ReasonCode = &value
	}
	if attempt.Tool != "" {
		value := attempt.Tool
		response.Tool = &value
	}
	return response
}

func writeMediaDiagnosticsError(writer http.ResponseWriter, request *http.Request, err error) {
	if errors.Is(err, thumbnail.ErrDiagnosticsLibraryNotFound) {
		writePublicError(writer, request, http.StatusNotFound,
			"library_not_found", "The media library was not found.")
		return
	}
	if errors.Is(err, thumbnail.ErrDiagnosticsFailureNotFound) {
		writePublicError(writer, request, http.StatusNotFound,
			"media_failure_not_found", "The media failure was not found.")
		return
	}
	if errors.Is(err, thumbnail.ErrInvalidDiagnosticsRequest) {
		writePublicError(writer, request, http.StatusUnprocessableEntity,
			"invalid_request", "The diagnostics request is invalid.")
		return
	}
	writeInternalError(writer, request)
}

func mediaJobID(id int64) string { return "mjob_" + strconv.FormatInt(id, 10) }

func mediaFailureRevision(revision thumbnail.FailureRevision) string {
	return "mfailrev_" + strconv.FormatInt(revision.FinishedAtMS, 10) + "_" +
		strconv.FormatInt(revision.JobID, 10)
}
