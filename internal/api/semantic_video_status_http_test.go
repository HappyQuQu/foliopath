package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/semantic"
)

type videoStatusStub struct {
	status semantic.VideoStatus
	err    error
}

func (stub videoStatusStub) Get(context.Context, int64) (semantic.VideoStatus, error) {
	return stub.status, stub.err
}

func TestVideoStatusHTTP(t *testing.T) {
	for _, recorded := range []bool{false, true} {
		status := semantic.VideoStatus{LibraryID: 1, Offline: true}
		if recorded {
			status.GenerationID = "aig_test"
			status.Coverage = &semantic.VideoCoverage{Eligible: 3, Ready: 1, Degraded: 1, Revision: 2}
		}
		mux := http.NewServeMux()
		registerVideoSemanticStatusRoute(mux, videoStatusStub{status: status})
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, httptest.NewRequest("GET", "/api/v1/libraries/lib_1/ai/video-semantic", nil))
		var result videoSemanticStatusResponse
		if response.Code != 200 || json.Unmarshal(response.Body.Bytes(), &result) != nil {
			t.Fatal(response.Body.String())
		}
		if !result.Offline || (result.Coverage != nil) != recorded {
			t.Fatalf("status: %+v", result)
		}
		if recorded && (result.Coverage.Complete || result.Coverage.Degraded != 1) {
			t.Fatal(result.Coverage)
		}
	}
	for _, test := range []struct {
		id   string
		err  error
		code int
	}{
		{"bad", nil, 400}, {"lib_1", semantic.ErrSemanticLibraryNotFound, 404}, {"lib_1", context.Canceled, 500},
	} {
		mux := http.NewServeMux()
		registerVideoSemanticStatusRoute(mux, videoStatusStub{err: test.err})
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, httptest.NewRequest("GET", "/api/v1/libraries/"+test.id+"/ai/video-semantic", nil))
		if response.Code != test.code {
			t.Fatalf("%s: %d", test.id, response.Code)
		}
	}
}

func TestVideoStatusRequiresAuthentication(t *testing.T) {
	handler := testRoutes(t, RouteDependencies{Readiness: func() Readiness { return Readiness{Ready: true} }, SystemStatus: func(context.Context) (SystemStatus, error) { return SystemStatus{}, nil }, Authentication: rejectingAuthentication(), VideoSemanticStatus: videoStatusStub{}})
	response := performRequest(handler, "/api/v1/libraries/lib_1/ai/video-semantic")
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", response.Code)
	}
	assertSafeErrorResponse(t, response, "authentication_required")
}
