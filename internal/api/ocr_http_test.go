package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/auth"
	"github.com/HappyQuQu/foliopath/internal/ocr"
)

type ocrHTTPStub struct {
	writes int
	err    error
}

func (s *ocrHTTPStub) Get(_ context.Context, id int64) (ocr.Status, error) {
	return ocr.Status{Settings: ocr.Settings{LibraryID: id, Revision: 2, Epoch: 1}, ExpectedFingerprint: strings.Repeat("a", 64), State: "disabled"}, s.err
}
func (s *ocrHTTPStub) Update(ctx context.Context, id, rev int64, enabled bool) (ocr.Status, error) {
	s.writes++
	v, e := s.Get(ctx, id)
	v.Settings.Enabled = enabled
	v.Settings.Revision = rev + 1
	return v, e
}
func (s *ocrHTTPStub) Request(_ context.Context, id, rev int64, mode ocr.Mode, key string) (ocr.Job, bool, error) {
	s.writes++
	return ocr.Job{ID: "ocrjob_synthetic", LibraryID: id, Epoch: 1, Fingerprint: strings.Repeat("a", 64), State: "queued", Mode: mode}, false, s.err
}
func (s *ocrHTTPStub) Job(_ context.Context, id int64, job string) (ocr.Job, error) {
	return ocr.Job{ID: job, LibraryID: id, Epoch: 1, Fingerprint: strings.Repeat("a", 64), State: "queued", Mode: ocr.Missing}, s.err
}
func (s *ocrHTTPStub) Cancel(ctx context.Context, id int64, job string, rev int64) (ocr.Job, error) {
	s.writes++
	return s.Job(ctx, id, job)
}
func (s *ocrHTTPStub) Clear(ctx context.Context, id, rev int64) (ocr.Status, error) {
	s.writes++
	return s.Get(ctx, id)
}
func ocrHTTPHandler(s *ocrHTTPStub) http.Handler {
	mux := http.NewServeMux()
	registerOCRRoutes(mux, s)
	return mux
}
func ocrHTTPRequest(h http.Handler, method, path, body, etag string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("If-Match", etag)
	r.Header.Set("Idempotency-Key", "test-ocr-key")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}
func TestOCRHTTPContractSettingsJobsAndConfirmations(t *testing.T) {
	s := &ocrHTTPStub{}
	h := ocrHTTPHandler(s)
	path := "/api/v1/libraries/lib_1/ai/ocr"
	w := ocrHTTPRequest(h, "GET", path, "", "")
	if w.Code != 200 || w.Header().Get("ETag") != `"ocr-library-1-r2"` || !strings.Contains(w.Body.String(), `"latestJob":null`) {
		t.Fatal(w.Code, w.Body.String())
	}
	w = ocrHTTPRequest(h, "PUT", path, `{"enabled":true}`, `"ocr-library-1-r2"`)
	if w.Code != 200 || s.writes != 1 {
		t.Fatal(w.Code, w.Body.String())
	}
	w = ocrHTTPRequest(h, "POST", path+"/jobs", `{"mode":"missing"}`, `"ocr-library-1-r2"`)
	if w.Code != 202 || w.Header().Get("Location") != path+"/jobs/ocrjob_synthetic" || w.Header().Get("ETag") != `"ocr-job-ocrjob_synthetic-r1"` {
		t.Fatal(w.Code, w.Header(), w.Body.String())
	}
	w = ocrHTTPRequest(h, "POST", path+"/jobs/ocrjob_synthetic/cancel", `{}`, `"ocr-job-ocrjob_synthetic-r1"`)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	w = ocrHTTPRequest(h, "POST", path+"/clear", `{"confirmation":"clear_ocr_results"}`, `"ocr-library-1-r2"`)
	if w.Code != 202 {
		t.Fatal(w.Code, w.Body.String())
	}
	for _, test := range []struct {
		method, suffix, body, etag string
		code                       int
	}{
		{"PUT", "", `{"enabled":true}`, "", 428}, {"PUT", "", `{"enabled":true}`, `W/"ocr-library-1-r2"`, 412},
		{"PUT", "", `{"enabled":true,"identity":{}}`, `"ocr-library-1-r2"`, 400},
		{"POST", "/jobs", `{"mode":"retry_failed"}`, `"ocr-library-1-r2"`, 400},
		{"POST", "/clear", `{"confirmation":"clear_all"}`, `"ocr-library-1-r2"`, 400},
		{"POST", "/jobs/ocrjob_synthetic/cancel", `null`, `"ocr-job-ocrjob_synthetic-r1"`, 400},
		{"GET", "?libraryId=lib_2", "", "", 400},
	} {
		before := s.writes
		w = ocrHTTPRequest(h, test.method, path+test.suffix, test.body, test.etag)
		if w.Code != test.code || before != s.writes {
			t.Fatal(test, w.Code, w.Body.String())
		}
	}
}
func TestOCRHTTPErrorsNeverLeakSourceOrText(t *testing.T) {
	for _, test := range []struct {
		err    error
		status int
	}{{ocr.ErrNotFound, 404}, {ocr.ErrRevision, 412}, {ocr.ErrConflict, 409}, {ocr.ErrDisabled, 409}, {ocr.ErrOffline, 409}, {errors.New("SELECT plaintext from /private/ocr secret"), 500}} {
		s := &ocrHTTPStub{err: test.err}
		w := ocrHTTPRequest(ocrHTTPHandler(s), "GET", "/api/v1/libraries/lib_1/ai/ocr", "", "")
		if w.Code != test.status || strings.Contains(w.Body.String(), "plaintext") || strings.Contains(w.Body.String(), "/private") {
			t.Fatal(w.Code, w.Body.String())
		}
	}
}
func TestOCRHTTPAuthenticationAndCSRFGuardAllRoutes(t *testing.T) {
	s := &ocrHTTPStub{}
	authentication := &authenticationStub{session: auth.CurrentSession{CSRFToken: "synthetic-csrf"}}
	h := requireAPIAuthentication(ocrHTTPHandler(s), authentication)
	for _, path := range []string{"/api/v1/libraries/lib_1/ai/ocr", "/api/v1/libraries/lib_1/ai/ocr/jobs/ocrjob_synthetic"} {
		w := ocrHTTPRequest(h, "GET", path, "", "")
		if w.Code != 401 {
			t.Fatal(w.Code)
		}
	}
	for _, test := range []struct{ method, suffix, body, etag string }{
		{"PUT", "", `{"enabled":true}`, `"ocr-library-1-r2"`},
		{"POST", "/jobs", `{"mode":"missing"}`, `"ocr-library-1-r2"`},
		{"POST", "/clear", `{"confirmation":"clear_ocr_results"}`, `"ocr-library-1-r2"`},
		{"POST", "/jobs/ocrjob_synthetic/cancel", `{}`, `"ocr-job-ocrjob_synthetic-r1"`},
	} {
		r := httptest.NewRequest(test.method, "/api/v1/libraries/lib_1/ai/ocr"+test.suffix, strings.NewReader(test.body))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("If-Match", test.etag)
		r.Header.Set("Idempotency-Key", "test-ocr-key")
		r.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "synthetic-cookie"})
		before := s.writes
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 403 || s.writes != before {
			t.Fatal(w.Code, w.Body.String())
		}
		r = httptest.NewRequest(test.method, r.URL.String(), strings.NewReader(test.body))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("If-Match", test.etag)
		r.Header.Set("Idempotency-Key", "test-ocr-key")
		r.Header.Set("X-CSRF-Token", "synthetic-csrf")
		r.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "synthetic-cookie"})
		w = httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code >= 300 || s.writes != before+1 {
			t.Fatal(w.Code, w.Body.String())
		}
	}
}
