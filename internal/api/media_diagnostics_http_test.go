package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/media"
	"github.com/HappyQuQu/foliopath/internal/thumbnail"
)

type mediaDiagnosticsStub struct {
	failures []thumbnail.MediaFailure
	retry    thumbnail.RetrySummary
	query    thumbnail.FailureQuery
	err      error
	mode     thumbnail.RequeueMode
}

func (stub *mediaDiagnosticsStub) LatestFailureRevision(
	_ context.Context,
	_ thumbnail.FailureQuery,
) (thumbnail.FailureRevision, bool, error) {
	if len(stub.failures) == 0 {
		return thumbnail.FailureRevision{}, false, nil
	}
	return thumbnail.FailureRevision{
		FinishedAtMS: stub.failures[0].FinishedAtMS,
		JobID:        stub.failures[0].JobID,
	}, true, nil
}

func TestMediaDiagnosticsHTTPReturnsSafeAttemptHistory(t *testing.T) {
	exitCode := 124
	attempt := thumbnail.MediaFailureAttempt{
		AttemptNumber: 3, Outcome: thumbnail.JobPermanent,
		Stage: media.StageFrameExtract, Reason: media.ReasonTimedOut,
		Tool: "ffmpeg", ExitCode: &exitCode, DurationMS: 45_000, FinishedAtMS: 1_000,
	}
	stub := &mediaDiagnosticsStub{failures: []thumbnail.MediaFailure{{
		JobID: 9, LibraryID: 2, AssetID: 7, LibraryName: "Family",
		RelativePath: "videos/broken.mp4", Variant: thumbnail.VariantStoryboard,
		ErrorCode: thumbnail.JobErrorTimeout, AttemptCount: 3, FinishedAtMS: 1_000,
		LatestAttempt: &attempt, AttemptHistory: []thumbnail.MediaFailureAttempt{attempt},
	}}}
	mux := http.NewServeMux()
	registerMediaDiagnosticsRoutes(mux, stub)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet,
		"/api/v1/diagnostics/media-failures/mjob_9", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body)
	}
	assertJSONEquals(t, response, map[string]any{
		"failure": map[string]any{
			"id": "mjob_9", "libraryId": "lib_2", "libraryName": "Family",
			"assetId": "ast_7", "relativePath": "videos/broken.mp4",
			"variant": "storyboard", "errorCode": "media_processing_timeout",
			"attempts": float64(3), "finishedAt": "1970-01-01T00:00:01Z",
			"latestAttempt": map[string]any{
				"attemptNumber": float64(3), "outcome": "permanent_failure",
				"stage": "frame_extract", "reasonCode": "time_limit_exceeded",
				"tool": "ffmpeg", "exitCode": float64(124),
				"durationMs": float64(45000), "finishedAt": "1970-01-01T00:00:01Z",
			},
		},
		"attemptHistory": []any{map[string]any{
			"attemptNumber": float64(3), "outcome": "permanent_failure",
			"stage": "frame_extract", "reasonCode": "time_limit_exceeded",
			"tool": "ffmpeg", "exitCode": float64(124),
			"durationMs": float64(45000), "finishedAt": "1970-01-01T00:00:01Z",
		}},
	})
}

func (stub *mediaDiagnosticsStub) ListFailures(
	_ context.Context,
	query thumbnail.FailureQuery,
) ([]thumbnail.MediaFailure, error) {
	stub.query = query
	return stub.failures, nil
}

func (stub *mediaDiagnosticsStub) GetFailure(
	_ context.Context,
	_ int64,
) (thumbnail.MediaFailure, error) {
	if stub.err != nil {
		return thumbnail.MediaFailure{}, stub.err
	}
	if len(stub.failures) == 0 {
		return thumbnail.MediaFailure{}, thumbnail.ErrDiagnosticsFailureNotFound
	}
	return stub.failures[0], nil
}

func (stub *mediaDiagnosticsStub) ProcessMedia(
	_ context.Context,
	_ int64,
	mode thumbnail.RequeueMode,
	_ int,
) (thumbnail.RetrySummary, error) {
	stub.mode = mode
	return stub.retry, stub.err
}

func TestMediaDiagnosticsHTTPRejectsMissingLibrary(t *testing.T) {
	stub := &mediaDiagnosticsStub{err: thumbnail.ErrDiagnosticsLibraryNotFound}
	mux := http.NewServeMux()
	registerMediaDiagnosticsRoutes(mux, stub)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost,
		"/api/v1/libraries/lib_999/media-processing/repair", nil))
	assertSafeErrorResponse(t, response, "library_not_found")
}

func TestMediaDiagnosticsHTTPListsSafeFailureDetails(t *testing.T) {
	stub := &mediaDiagnosticsStub{failures: []thumbnail.MediaFailure{{
		JobID: 9, LibraryID: 2, AssetID: 7, LibraryName: "Family",
		RelativePath: "videos/broken.mp4", Variant: thumbnail.VariantStoryboard,
		ErrorCode: thumbnail.JobErrorTimeout, AttemptCount: 3, FinishedAtMS: 1_000,
	}}}
	mux := http.NewServeMux()
	registerMediaDiagnosticsRoutes(mux, stub)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet,
		"/api/v1/diagnostics/media-failures?libraryId=lib_2&variant=storyboard&limit=10", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body)
	}
	if stub.query.LibraryID != 2 || stub.query.Limit != 10 ||
		stub.query.Variant != thumbnail.VariantStoryboard {
		t.Fatalf("query = %#v", stub.query)
	}
	assertJSONEquals(t, response, map[string]any{
		"items": []any{map[string]any{
			"id": "mjob_9", "libraryId": "lib_2", "libraryName": "Family",
			"assetId": "ast_7", "relativePath": "videos/broken.mp4",
			"variant": "storyboard", "errorCode": "media_processing_timeout",
			"attempts": float64(3), "finishedAt": "1970-01-01T00:00:01Z",
		}},
		"nextCursor": nil,
		"revision":   "mfailrev_1000_9",
	})
}

func TestMediaDiagnosticsHTTPRequeuesRequestedMediaMode(t *testing.T) {
	stub := &mediaDiagnosticsStub{retry: thumbnail.RetrySummary{
		Requeued: 34,
	}}
	mux := http.NewServeMux()
	registerMediaDiagnosticsRoutes(mux, stub)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost,
		"/api/v1/libraries/lib_2/media-processing/repair?mode=all", nil))
	if response.Code != http.StatusAccepted {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body)
	}
	assertJSONEquals(t, response, map[string]any{
		"requeued": float64(34), "remainingEligible": float64(0),
		"permanentFailures": float64(0),
	})
	if stub.mode != thumbnail.RequeueAll {
		t.Fatalf("mode = %q", stub.mode)
	}
}
