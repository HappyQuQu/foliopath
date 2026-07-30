package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/thumbnail"
)

type cacheServiceStub struct {
	summary thumbnail.CacheSummary
	result  thumbnail.CleanupRequestResult
	key     string
}

func (stub *cacheServiceStub) Summary(context.Context) (thumbnail.CacheSummary, error) {
	return stub.summary, nil
}

func (stub *cacheServiceStub) Cleanup(context.Context) (thumbnail.Cleanup, error) {
	return stub.summary.Cleanup, nil
}

func (stub *cacheServiceStub) StartCleanup(
	_ context.Context,
	key string,
) (thumbnail.CleanupRequestResult, error) {
	stub.key = key
	return stub.result, nil
}

func TestCacheRoutesReturnSafeSummaryAndAcceptCleanup(t *testing.T) {
	requestedAt := int64(1_700_000_000_000)
	service := &cacheServiceStub{
		summary: thumbnail.CacheSummary{
			UsageBytes: 100, QuotaBytes: 1000,
			HighWatermarkBytes: 900, LowWatermarkBytes: 750,
			AvailableBytes: 2000, SafeFreeSpaceBytes: 500,
			Pressure: "normal",
			Cleanup: thumbnail.Cleanup{
				Revision: 2, Status: thumbnail.CleanupQueued,
				RequestedAtMS: &requestedAt,
			},
		},
		result: thumbnail.CleanupRequestResult{
			Created: true,
			Cleanup: thumbnail.Cleanup{
				Revision: 3, Status: thumbnail.CleanupQueued,
				RequestedAtMS: &requestedAt,
			},
		},
	}
	mux := http.NewServeMux()
	registerCacheRoutes(mux, service)

	summary := httptest.NewRecorder()
	mux.ServeHTTP(summary, httptest.NewRequest(http.MethodGet, "/api/v1/cache", nil))
	if summary.Code != http.StatusOK ||
		summary.Header().Get("ETag") != `"cache-r2-u100-a2000-q1000"` {
		t.Fatalf("cache summary = %d %q %s", summary.Code, summary.Header().Get("ETag"), summary.Body.String())
	}
	if summary.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("cache summary Cache-Control = %q", summary.Header().Get("Cache-Control"))
	}

	startRequest := httptest.NewRequest(http.MethodPost, "/api/v1/cache/cleanup", nil)
	startRequest.Header.Set("Idempotency-Key", "cleanup-key-1")
	start := httptest.NewRecorder()
	mux.ServeHTTP(start, startRequest)
	if start.Code != http.StatusAccepted ||
		start.Header().Get("Location") != "/api/v1/cache/cleanup" ||
		start.Header().Get("Idempotency-Replayed") != "false" ||
		service.key != "cleanup-key-1" {
		t.Fatalf("start cleanup = %d %#v %s", start.Code, start.Header(), start.Body.String())
	}
}

func TestCacheCleanupRejectsInvalidIdempotencyKey(t *testing.T) {
	service := &cacheServiceStub{}
	mux := http.NewServeMux()
	registerCacheRoutes(mux, service)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/cache/cleanup", nil)
	request.Header.Set("Idempotency-Key", "short")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest || service.key != "" {
		t.Fatalf("invalid idempotency response = %d %s", response.Code, response.Body.String())
	}
}
