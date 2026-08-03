package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/systemlog"
)

type systemLogStub struct {
	events []systemlog.Event
	query  systemlog.Query
}

func (stub *systemLogStub) List(
	_ context.Context,
	query systemlog.Query,
) ([]systemlog.Event, error) {
	stub.query = query
	return stub.events, nil
}

func TestSystemLogHTTPListsSanitizedEvents(t *testing.T) {
	stub := &systemLogStub{events: []systemlog.Event{{
		ID: 7, OccurredAtMS: 1_000, Level: systemlog.LevelError,
		Module: "http", EventCode: "http.request_failed",
		RequestID: "req_test", Method: "POST",
		RoutePattern: "POST /api/v1/settings", StatusCode: 500,
		DurationMS: 12,
	}}}
	mux := http.NewServeMux()
	registerSystemLogRoute(mux, stub)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(
		http.MethodGet,
		"/api/v1/system-logs?level=error&module=http&limit=10",
		nil,
	))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body)
	}
	if stub.query.Level != systemlog.LevelError || stub.query.Module != "http" ||
		stub.query.Limit != 10 {
		t.Fatalf("query = %#v", stub.query)
	}
	assertJSONEquals(t, response, map[string]any{
		"items": []any{map[string]any{
			"id": "sevt_7", "level": "error", "module": "http",
			"eventCode":  "http.request_failed",
			"occurredAt": "1970-01-01T00:00:01Z",
			"requestId":  "req_test", "method": "POST",
			"routePattern": "POST /api/v1/settings",
			"statusCode":   float64(500), "durationMs": float64(12),
		}},
		"nextCursor": nil,
	})
}

func TestSystemLogHTTPRejectsInvalidLevel(t *testing.T) {
	mux := http.NewServeMux()
	registerSystemLogRoute(mux, &systemLogStub{})
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(
		http.MethodGet,
		"/api/v1/system-logs?level=debug",
		nil,
	))
	assertSafeErrorResponse(t, response, "invalid_request")
}
