package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/releaseinfo"
)

type releaseInfoStub struct {
	snapshot releaseinfo.Snapshot
	refresh  bool
}

func (stub *releaseInfoStub) Get(
	_ context.Context,
	refresh bool,
) (releaseinfo.Snapshot, error) {
	stub.refresh = refresh
	return stub.snapshot, nil
}

func TestReleaseInfoHTTPReturnsVersionHistory(t *testing.T) {
	stub := &releaseInfoStub{snapshot: releaseinfo.Snapshot{
		CurrentVersion: "v1.0.0", LatestVersion: "v1.1.0", UpdateAvailable: true,
		CheckedAt: time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC),
		Releases: []releaseinfo.Release{{
			Version: "v1.1.0", Name: "FolioPath 1.1", Summary: "Safer logs",
			Notes:       "### 🐛 Fixes\n\n* Safer logs",
			PublishedAt: time.Date(2026, 8, 2, 0, 0, 0, 0, time.UTC),
			URL:         "https://example.test/v1.1.0",
		}},
	}}
	mux := http.NewServeMux()
	registerReleaseInfoRoute(mux, stub)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet,
		"/api/v1/releases?refresh=true", nil))
	if response.Code != http.StatusOK || !stub.refresh {
		t.Fatalf("status = %d, refresh = %t", response.Code, stub.refresh)
	}
	assertJSONEquals(t, response, map[string]any{
		"currentVersion": "v1.0.0", "latestVersion": "v1.1.0",
		"updateAvailable": true, "checkedAt": "2026-08-03T00:00:00Z",
		"releases": []any{map[string]any{
			"version": "v1.1.0", "name": "FolioPath 1.1", "summary": "Safer logs",
			"notes":       "### 🐛 Fixes\n\n* Safer logs",
			"publishedAt": "2026-08-02T00:00:00Z", "url": "https://example.test/v1.1.0",
		}},
	})
}

func TestReleaseInfoHTTPRejectsUnknownRefreshValue(t *testing.T) {
	mux := http.NewServeMux()
	registerReleaseInfoRoute(mux, &releaseInfoStub{})
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet,
		"/api/v1/releases?refresh=sometimes", nil))
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d", response.Code)
	}
	assertSafeErrorResponse(t, response, "invalid_request")
}
