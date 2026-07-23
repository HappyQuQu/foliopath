package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequestIDReplacesClientValueAndReachesContext(t *testing.T) {
	var contextRequestID string
	handler := withRequestID(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		contextRequestID, _ = RequestIDFromContext(request.Context())
		writer.WriteHeader(http.StatusNoContent)
	}), func() (string, error) {
		return "req_server_generated", nil
	})

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set(RequestIDHeader, "req_client_supplied")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if got := response.Header().Get(RequestIDHeader); got != "req_server_generated" {
		t.Fatalf("%s = %q, want server-generated value", RequestIDHeader, got)
	}
	if contextRequestID != "req_server_generated" {
		t.Fatalf("context request ID = %q, want server-generated value", contextRequestID)
	}
}

func TestRequestIDFallsBackWhenGenerationFailsOrIsInvalid(t *testing.T) {
	tests := []struct {
		name     string
		generate requestIDGenerator
	}{
		{
			name: "generation failure",
			generate: func() (string, error) {
				return "", errors.New("entropy unavailable")
			},
		},
		{
			name: "invalid generated value",
			generate: func() (string, error) {
				return "client value with spaces", nil
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := withRequestID(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.WriteHeader(http.StatusNoContent)
			}), test.generate)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))

			requestID := response.Header().Get(RequestIDHeader)
			if !validRequestID(requestID) || !strings.HasPrefix(requestID, "req_fallback_") {
				t.Fatalf("fallback request ID = %q, want valid fallback", requestID)
			}
		})
	}
}

func TestHandlerReturnsContractShapedNotFound(t *testing.T) {
	response := httptest.NewRecorder()
	NewHandler(nil, discardLogger()).ServeHTTP(
		response,
		httptest.NewRequest(http.MethodGet, "/not-a-route", nil),
	)

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
	assertSafeErrorResponse(t, response, codeResourceNotFound)
}

func TestRequestLogUsesSafeStructuredMetadata(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	handler := NewHandler(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
	}), logger)
	request := httptest.NewRequest(
		http.MethodGet,
		"/private/photo.jpg?token=secret-query",
		nil,
	)
	request.Header.Set("Cookie", "session=secret-cookie")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	logged := logs.String()
	for _, expected := range []string{
		`"msg":"http.request_completed"`,
		`"method":"GET"`,
		`"status":204`,
		`"request_id":"req_`,
	} {
		if !strings.Contains(logged, expected) {
			t.Fatalf("structured request log missing %q: %s", expected, logged)
		}
	}
	for _, forbidden := range []string{
		"private/photo.jpg",
		"secret-query",
		"secret-cookie",
	} {
		if strings.Contains(logged, forbidden) {
			t.Fatalf("structured request log leaked %q: %s", forbidden, logged)
		}
	}
}

func TestRecoveryMasksPanicAndLogsOnlyCorrelationData(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	handler := withRequestID(
		recoverPanics(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			panic("database /app/data/foliopath.db SELECT token=secret")
		}), logger),
		func() (string, error) {
			return "req_recovery_test", nil
		},
	)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
	assertSafeErrorResponse(t, response, codeInternalError)

	combined := response.Body.String() + logs.String()
	for _, forbidden := range []string{"/app/data", "SELECT", "token=secret"} {
		if strings.Contains(combined, forbidden) {
			t.Fatalf("response or log leaked %q: %s", forbidden, combined)
		}
	}
	if !strings.Contains(logs.String(), `"request_id":"req_recovery_test"`) {
		t.Fatalf("structured log missing request ID: %s", logs.String())
	}
}

func TestRecoveryDoesNotAppendErrorAfterResponseWasCommitted(t *testing.T) {
	handler := withRequestID(
		recoverPanics(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			writer.WriteHeader(http.StatusAccepted)
			_, _ = writer.Write([]byte("partial"))
			panic("after commit")
		}), discardLogger()),
		func() (string, error) {
			return "req_committed_test", nil
		},
	)
	response := httptest.NewRecorder()
	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("committed panic was not returned to net/http")
			}
		}()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	}()

	if response.Code != http.StatusAccepted || response.Body.String() != "partial" {
		t.Fatalf("committed response = (%d, %q), want (202, partial)", response.Code, response.Body.String())
	}
}

func assertSafeErrorResponse(t *testing.T, response *httptest.ResponseRecorder, wantCode string) {
	t.Helper()

	if got := response.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}
	requestID := response.Header().Get(RequestIDHeader)
	if !validRequestID(requestID) {
		t.Fatalf("%s = %q, want valid server ID", RequestIDHeader, requestID)
	}

	var payload map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	errorValue, ok := payload["error"].(map[string]any)
	if !ok {
		t.Fatalf("response error = %#v, want object", payload["error"])
	}
	if len(errorValue) != 3 {
		t.Fatalf("error fields = %#v, want exactly code/message/requestId", errorValue)
	}
	if errorValue["code"] != wantCode || errorValue["requestId"] != requestID {
		t.Fatalf("error = %#v, want code %q and request ID %q", errorValue, wantCode, requestID)
	}
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
}
