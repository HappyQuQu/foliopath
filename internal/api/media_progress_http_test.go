package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/thumbnail"
)

type mediaProgressServiceStub struct {
	progress thumbnail.ProcessingProgress
	err      error
}

func (stub mediaProgressServiceStub) Get(
	context.Context,
	int64,
) (thumbnail.ProcessingProgress, error) {
	return stub.progress, stub.err
}

func TestMediaProgressHTTPReturnsSeparateDerivedQueues(t *testing.T) {
	mux := http.NewServeMux()
	registerMediaProgressRoute(mux, mediaProgressServiceStub{
		progress: thumbnail.ProcessingProgress{
			Grid: thumbnail.JobProgress{
				Queued: 2, Running: 1, Succeeded: 6, Failed: 1,
			},
			Storyboard: thumbnail.JobProgress{
				Queued: 1, Succeeded: 2,
			},
			StoryboardPendingEligibility: 1,
		},
	})
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(
		http.MethodGet,
		"/api/v1/libraries/lib_7/media-processing",
		nil,
	))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body)
	}
	assertJSONEquals(t, response, map[string]any{
		"active": true,
		"thumbnails": map[string]any{
			"total": float64(10), "processed": float64(7),
			"queued": float64(2), "running": float64(1),
			"succeeded": float64(6), "failed": float64(1),
		},
		"videoPreviews": map[string]any{
			"total": float64(3), "processed": float64(2),
			"queued": float64(1), "running": float64(0),
			"succeeded": float64(2), "failed": float64(0),
		},
		"videoPreviewsPendingEligibility": float64(1),
	})
}

func TestMediaProgressHTTPMapsMissingLibrary(t *testing.T) {
	for _, target := range []string{
		"/api/v1/libraries/not-a-library/media-processing",
		"/api/v1/libraries/lib_7/media-processing",
	} {
		mux := http.NewServeMux()
		registerMediaProgressRoute(mux, mediaProgressServiceStub{
			err: errors.New("unexpected"),
		})
		if target == "/api/v1/libraries/lib_7/media-processing" {
			mux = http.NewServeMux()
			registerMediaProgressRoute(mux, mediaProgressServiceStub{
				err: thumbnail.ErrProgressLibraryNotFound,
			})
		}
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, target, nil))
		if response.Code != http.StatusNotFound {
			t.Fatalf("%s status = %d", target, response.Code)
		}
		assertSafeErrorResponse(t, response, "library_not_found")
	}
}
