package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestHealthRoutesMatchContract(t *testing.T) {
	readiness := Readiness{
		Ready:      false,
		ReasonCode: ReadinessDatabaseUnavailable,
	}
	handler := testRoutes(t, RouteDependencies{
		Readiness: func() Readiness {
			return readiness
		},
		AuthorizeStatus: func(*http.Request) bool {
			return false
		},
		SystemStatus: func(context.Context) (SystemStatus, error) {
			t.Fatal("unauthenticated status request reached provider")
			return SystemStatus{}, nil
		},
	})

	t.Run("liveness", func(t *testing.T) {
		response := performRequest(handler, "/health/live")
		if response.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
		}
		assertJSONEquals(t, response, map[string]any{
			"status":     "live",
			"reasonCode": nil,
		})
	})

	t.Run("not ready", func(t *testing.T) {
		response := performRequest(handler, "/health/ready")
		if response.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
		}
		if response.Header().Get("Retry-After") != "1" {
			t.Fatalf("Retry-After = %q, want 1", response.Header().Get("Retry-After"))
		}
		assertJSONEquals(t, response, map[string]any{
			"status":     "not_ready",
			"reasonCode": "database_unavailable",
		})
	})

	t.Run("ready", func(t *testing.T) {
		readiness = Readiness{Ready: true}
		response := performRequest(handler, "/health/ready")
		if response.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
		}
		assertJSONEquals(t, response, map[string]any{
			"status":     "ready",
			"reasonCode": nil,
		})
	})
}

func TestReadinessMasksUnknownReason(t *testing.T) {
	handler := testRoutes(t, RouteDependencies{
		Readiness: func() Readiness {
			return Readiness{ReasonCode: "/app/data/foliopath.db failed"}
		},
		AuthorizeStatus: denyStatus,
		SystemStatus:    unusedStatus,
	})

	response := performRequest(handler, "/health/ready")
	if strings.Contains(response.Body.String(), "/app/data") {
		t.Fatalf("readiness leaked internal reason: %s", response.Body.String())
	}
	assertJSONEquals(t, response, map[string]any{
		"status":     "not_ready",
		"reasonCode": "database_unavailable",
	})
}

func TestSystemStatusRequiresAuthorization(t *testing.T) {
	providerCalled := false
	handler := testRoutes(t, RouteDependencies{
		Readiness: func() Readiness {
			return Readiness{Ready: true}
		},
		AuthorizeStatus: denyStatus,
		SystemStatus: func(context.Context) (SystemStatus, error) {
			providerCalled = true
			return SystemStatus{}, nil
		},
	})

	response := performRequest(handler, "/api/v1/status")
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
	assertSafeErrorResponse(t, response, "authentication_required")
	if providerCalled {
		t.Fatal("unauthorized request reached system status provider")
	}
}

func TestAuthorizedSystemStatusMatchesContract(t *testing.T) {
	want := SystemStatus{
		Version:          "0.1.0",
		APIVersion:       "v1",
		RuntimeState:     "ready",
		Initialized:      true,
		ReadOnlyMedia:    true,
		SupportedLocales: []string{"zh-CN", "en"},
		SupportedMedia: SupportedMedia{
			ImageMIMETypes: []string{
				"image/jpeg",
				"image/png",
				"image/webp",
				"image/gif",
			},
			VideoMIMETypes: []string{
				"video/mp4",
				"video/quicktime",
				"video/x-matroska",
			},
			VideoTranscoding: false,
		},
	}
	handler := testRoutes(t, RouteDependencies{
		Readiness: func() Readiness {
			return Readiness{Ready: true}
		},
		AuthorizeStatus: func(*http.Request) bool {
			return true
		},
		SystemStatus: func(context.Context) (SystemStatus, error) {
			return want, nil
		},
	})

	response := performRequest(handler, "/api/v1/status")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	var got SystemStatus
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("system status = %#v, want %#v", got, want)
	}
}

func TestSystemStatusFailureIsMasked(t *testing.T) {
	handler := testRoutes(t, RouteDependencies{
		Readiness: func() Readiness {
			return Readiness{Ready: true}
		},
		AuthorizeStatus: func(*http.Request) bool {
			return true
		},
		SystemStatus: func(context.Context) (SystemStatus, error) {
			return SystemStatus{}, errors.New("SELECT failed at /app/data/foliopath.db")
		},
	})

	response := performRequest(handler, "/api/v1/status")
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
	assertSafeErrorResponse(t, response, codeInternalError)
	for _, forbidden := range []string{"SELECT", "/app/data"} {
		if strings.Contains(response.Body.String(), forbidden) {
			t.Fatalf("system status response leaked %q: %s", forbidden, response.Body.String())
		}
	}
}

func TestRoutesRejectIncompleteDependencies(t *testing.T) {
	tests := []RouteDependencies{
		{},
		{
			Readiness: func() Readiness { return Readiness{} },
		},
		{
			Readiness:       func() Readiness { return Readiness{} },
			AuthorizeStatus: denyStatus,
		},
	}
	for _, dependencies := range tests {
		if _, err := NewRoutes(dependencies); !errors.Is(err, errInvalidRouteDependencies) {
			t.Fatalf("NewRoutes() error = %v, want errInvalidRouteDependencies", err)
		}
	}
}

func TestUnsupportedMethodUsesSafeFallback(t *testing.T) {
	handler := testRoutes(t, RouteDependencies{
		Readiness:       func() Readiness { return Readiness{Ready: true} },
		AuthorizeStatus: denyStatus,
		SystemStatus:    unusedStatus,
	})
	response := httptest.NewRecorder()
	handler.ServeHTTP(
		response,
		httptest.NewRequest(http.MethodPost, "/health/live", nil),
	)

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want safe not-found fallback", response.Code)
	}
	assertSafeErrorResponse(t, response, codeResourceNotFound)
}

func testRoutes(t *testing.T, dependencies RouteDependencies) http.Handler {
	t.Helper()
	routes, err := NewRoutes(dependencies)
	if err != nil {
		t.Fatalf("NewRoutes() error = %v", err)
	}
	return NewHandler(
		routes,
		slog.New(slog.NewJSONHandler(&bytes.Buffer{}, nil)),
	)
}

func performRequest(handler http.Handler, path string) *httptest.ResponseRecorder {
	response := httptest.NewRecorder()
	handler.ServeHTTP(
		response,
		httptest.NewRequest(http.MethodGet, path, nil),
	)
	return response
}

func assertJSONEquals(t *testing.T, response *httptest.ResponseRecorder, want map[string]any) {
	t.Helper()
	var got map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("response = %#v, want %#v", got, want)
	}
}

func denyStatus(*http.Request) bool {
	return false
}

func unusedStatus(context.Context) (SystemStatus, error) {
	return SystemStatus{}, nil
}
