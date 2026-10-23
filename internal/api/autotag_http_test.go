package api

import (
	"context"
	"github.com/HappyQuQu/foliopath/internal/aimodel"
	"github.com/HappyQuQu/foliopath/internal/autotag"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type automaticHTTPStub struct {
	updates, requests int
	err               error
}

func (s *automaticHTTPStub) Get(_ context.Context, id int64) (autotag.Settings, error) {
	return autotag.Settings{LibraryID: id, Revision: 2, Enabled: true, State: "awaiting_model", Coverage: autotag.Coverage{Eligible: 3, Revision: 4}}, s.err
}
func (s *automaticHTTPStub) Update(ctx context.Context, id int64, enabled bool, revision int64) (autotag.Settings, error) {
	s.updates++
	v, err := s.Get(ctx, id)
	v.Enabled = enabled
	v.Revision = revision + 1
	return v, err
}
func (s *automaticHTTPStub) Request(_ context.Context, id int64, mode autotag.Mode, key string) (aimodel.Operation, bool, error) {
	s.requests++
	now := time.Now()
	kind := aimodel.OperationAutomaticTagsMissing
	if mode == autotag.All {
		kind = aimodel.OperationAutomaticTagsRebuild
	}
	return aimodel.Operation{ID: "autotag_test_operation", LibraryID: id, Kind: kind, State: aimodel.OperationQueued, Phase: aimodel.PhaseQueued, Revision: 1, CreatedAt: now, UpdatedAt: now}, false, s.err
}
func TestAutomaticTagHTTPSettingsAndETag(t *testing.T) {
	service := &automaticHTTPStub{}
	mux := http.NewServeMux()
	registerAutomaticTagRoutes(mux, service)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/libraries/lib_1/ai/automatic-tags", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != 200 || w.Header().Get("ETag") != `"autotag-library-1-r2"` || !strings.Contains(w.Body.String(), `"model":null`) || !strings.Contains(w.Body.String(), `"activeOperationId":null`) || !strings.Contains(w.Body.String(), `"state":"awaiting_model"`) {
		t.Fatalf("get %d %s %s", w.Code, w.Header(), w.Body.String())
	}
	r = httptest.NewRequest(http.MethodPut, r.URL.String(), strings.NewReader(`{"enabled":false}`))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("If-Match", `"autotag-library-1-r2"`)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != 200 || w.Header().Get("ETag") != `"autotag-library-1-r3"` || service.updates != 1 {
		t.Fatalf("update %d %s", w.Code, w.Body.String())
	}
	for _, body := range []string{`{}`, `{"enabled":null}`, `{"enabled":true,"model":"invented"}`} {
		r = httptest.NewRequest(http.MethodPut, "/api/v1/libraries/lib_1/ai/automatic-tags", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("If-Match", `"autotag-library-1-r2"`)
		w = httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code != 400 {
			t.Fatalf("invalid body %s: %d", body, w.Code)
		}
	}
	if service.updates != 1 {
		t.Fatal("invalid settings called service")
	}
	service.err = autotag.ErrNotFound
	r = httptest.NewRequest(http.MethodGet, "/api/v1/libraries/lib_999/ai/automatic-tags", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != 404 || !strings.Contains(w.Body.String(), `"code":"library_not_found"`) {
		t.Fatalf("unknown %d %s", w.Code, w.Body.String())
	}
}
func TestAutomaticTagHTTPAllRequiresExplicitConfirmation(t *testing.T) {
	service := &automaticHTTPStub{}
	mux := http.NewServeMux()
	registerAutomaticTagRoutes(mux, service)
	for _, item := range []struct {
		body   string
		status int
	}{
		{`{"mode":"all"}`, 400}, {`{"mode":"all","confirmation":"wrong"}`, 400}, {`{"mode":"missing","confirmation":"reprocess_automatic_tags"}`, 400},
		{`{"mode":"all","confirmation":"reprocess_automatic_tags"}`, 202}, {`{"mode":"missing"}`, 202},
	} {
		r := httptest.NewRequest(http.MethodPost, "/api/v1/libraries/lib_1/ai/automatic-tags/jobs", strings.NewReader(item.body))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Idempotency-Key", "autotag-http-test-1")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code != item.status {
			t.Fatalf("%s: %d %s", item.body, w.Code, w.Body.String())
		}
		if w.Code == 202 && (w.Header().Get("Location") != "/api/v1/ai/operations/autotag_test_operation" || !strings.Contains(w.Body.String(), `"totalItems":null`)) {
			t.Fatalf("operation %s", w.Body.String())
		}
	}
	if service.requests != 2 {
		t.Fatal(service.requests)
	}
}
func TestAutomaticTagHTTPInvalidScopeAndRatePolicies(t *testing.T) {
	service := &automaticHTTPStub{}
	mux := http.NewServeMux()
	registerAutomaticTagRoutes(mux, service)
	for _, path := range []string{"/api/v1/libraries/invalid/ai/automatic-tags", "/api/v1/libraries/lib_1/ai/automatic-tags?model=arbitrary"} {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != 400 {
			t.Fatalf("%s: %d", path, w.Code)
		}
	}
	for _, item := range []struct {
		method, path string
		limit        int
	}{{"GET", "/api/v1/libraries/lib_1/ai/automatic-tags", 60}, {"PUT", "/api/v1/libraries/lib_1/ai/automatic-tags", 20}, {"POST", "/api/v1/libraries/lib_1/ai/automatic-tags/jobs", 10}} {
		policy, ok := requestRatePolicyFor(item.method, item.path)
		if !ok || policy.limit != item.limit {
			t.Fatalf("policy %+v", policy)
		}
	}
}
