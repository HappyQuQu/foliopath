package api

import (
	"context"
	"github.com/HappyQuQu/foliopath/internal/aimodel"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type proxyHTTPStub struct {
	calls int
	err   error
}

func (*proxyHTTPStub) Get(context.Context) (aimodel.ProxySnapshot, error) {
	return aimodel.ProxySnapshot{ProxySettings: aimodel.ProxySettings{Mode: "inherit", Revision: 1}, DeploymentConfigured: true, EffectiveMode: "proxy"}, nil
}
func (s *proxyHTTPStub) Save(_ context.Context, v aimodel.ProxySettings) (aimodel.ProxySnapshot, error) {
	s.calls++
	return aimodel.ProxySnapshot{ProxySettings: v}, s.err
}
func (s *proxyHTTPStub) Test(context.Context, aimodel.ProxySettings) (bool, error) {
	s.calls++
	return false, s.err
}
func TestModelProxyAuthenticationAndCSRF(t *testing.T) {
	s := &proxyHTTPStub{}
	mux := http.NewServeMux()
	registerModelProxyRoutes(mux, s)
	handler := requireAPIAuthentication(mux, acceptingAuthentication())
	for _, method := range []string{"GET", "PUT", "POST"} {
		path := "/api/v1/ai/model-proxy"
		if method == "POST" {
			path += "/test"
		}
		w := httptest.NewRecorder()
		r := httptest.NewRequest(method, path, nil)
		handler.ServeHTTP(w, r)
		if w.Code != 401 {
			t.Fatalf("anonymous %s: %d", method, w.Code)
		}
		if method == "GET" {
			continue
		}
		r = httptest.NewRequest(method, path, strings.NewReader(`{"mode":"direct","proxyUrl":"","revision":1}`))
		r.AddCookie(&http.Cookie{Name: SessionCookieName, Value: testCookieToken})
		w = httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != 403 || s.calls != 0 {
			t.Fatalf("missing csrf accepted %s: %d", method, w.Code)
		}
	}
}
func TestModelProxyHTTPContract(t *testing.T) {
	s := &proxyHTTPStub{}
	mux := http.NewServeMux()
	registerModelProxyRoutes(mux, s)
	for _, method := range []string{"PUT", "POST"} {
		path := "/api/v1/ai/model-proxy"
		if method == "POST" {
			path += "/test"
		}
		for _, body := range []string{`{}`, `{"mode":"direct","revision":1}`, `{"mode":"direct","proxyUrl":null,"revision":1}`, `{"mode":"direct","proxyUrl":"","revision":0}`, `{"mode":"direct","proxyUrl":"","revision":1,"target":"https://arbitrary.example"}`} {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(method, path, strings.NewReader(body))
			r.Header.Set("Content-Type", "application/json")
			mux.ServeHTTP(w, r)
			if w.Code != 400 || s.calls != 0 {
				t.Fatalf("invalid body accepted: %d %s", w.Code, w.Body.String())
			}
		}
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/ai/model-proxy", nil))
	if w.Code != 200 || w.Header().Get("Cache-Control") != "no-store" || !strings.Contains(w.Body.String(), `"deploymentConfigured":true`) {
		t.Fatal(w.Body.String())
	}
	for _, tt := range []struct {
		method, path string
		err          error
		status       int
	}{
		{"PUT", "/api/v1/ai/model-proxy", aimodel.ErrPreconditionFailed, 412},
		{"POST", "/api/v1/ai/model-proxy/test", aimodel.ErrDownloadActive, 409},
		{"POST", "/api/v1/ai/model-proxy/test", nil, 200},
	} {
		s.err = tt.err
		w = httptest.NewRecorder()
		r := httptest.NewRequest(tt.method, tt.path, strings.NewReader(`{"mode":"inherit","proxyUrl":"","revision":1}`))
		r.Header.Set("Content-Type", "application/json")
		mux.ServeHTTP(w, r)
		if w.Code != tt.status {
			t.Fatalf("status %d: %s", w.Code, w.Body.String())
		}
	}
}
