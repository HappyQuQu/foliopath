package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/catalog"
	"github.com/HappyQuQu/foliopath/internal/search"
)

type fusedHTTPStub struct {
	request   search.Request
	result    search.Result
	err       error
	verifyErr error
}

func (s *fusedHTTPStub) Search(_ context.Context, r search.Request) (search.Result, error) {
	s.request = r
	return s.result, s.err
}
func TestFusedHTTPActualSourcesScopeAndDegradedWire(t *testing.T) {
	score := float32(.8)
	stub := &fusedHTTPStub{result: search.Result{SemanticState: "ready", Hits: []search.Hit{{Evidence: search.Evidence("", "car.jpg", "folder/car.jpg"), LibraryID: 7, AssetID: 9, Sources: []search.Source{search.Filename, search.Visual}, SemanticScore: &score}}}}
	cat := catalogServiceStub{getAsset: func(_ context.Context, id int64) (catalog.Asset, error) {
		return catalog.Asset{ID: id, LibraryID: 7, Name: "car.jpg", RelativePath: "folder/car.jpg", Kind: catalog.KindImage}, nil
	}}
	mux := http.NewServeMux()
	admission := newSemanticSearchAdmission()
	registerFusedSearchRoute(mux, stub, cat, admission)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest("GET", "/api/v1/search/assets?q=车&mode=all&libraryId=lib_7&directoryId=dir_5&recursive=true&kind=image,animated&limit=20&modifiedFrom=2026-01-01T00:00:00Z", nil))
	if response.Code != 200 {
		t.Fatal(response.Code, response.Body.String())
	}
	r := stub.request
	if r.LibraryID != 7 || r.DirectoryID != 5 || !r.Recursive || r.Limit != 20 || r.ModifiedFromNS == nil || len(r.Kinds) != 2 {
		t.Fatalf("request=%+v", r)
	}
	var body fusedPageResponse
	if e := json.Unmarshal(response.Body.Bytes(), &body); e != nil {
		t.Fatal(e)
	}
	if len(body.Items) != 1 || len(body.Items[0].Sources) != 2 || body.Items[0].SemanticScore == nil {
		t.Fatal(response.Body.String())
	}
	stub.result = search.Result{SemanticState: "unavailable", Hits: []search.Hit{{Evidence: search.Evidence("", "car.jpg", "folder/car.jpg"), LibraryID: 7, AssetID: 9, Sources: []search.Source{search.Filename}}}}
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest("GET", "/api/v1/search/assets?q=car", nil))
	if response.Code != 200 || !strings.Contains(response.Body.String(), `"semanticScore":null`) || !strings.Contains(response.Body.String(), `"semanticState":"unavailable"`) {
		t.Fatal(response.Body.String())
	}

	// Same ID with a changed source must not receive stale match badges.
	stub.result.Hits[0].Evidence = search.Evidence("v1:old", "car.jpg", "folder/car.jpg")
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest("GET", "/api/v1/search/assets?q=car", nil))
	if response.Code != 409 {
		t.Fatal("source changed during hydration", response.Code, response.Body.String())
	}
	admission.acquire()
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest("GET", "/api/v1/search/assets?q=car", nil))
	admission.release()
	if response.Code != 429 {
		t.Fatal("unbounded admission", response.Code)
	}
}
func TestFusedHTTPInvalidAndStaleFailClosed(t *testing.T) {
	for _, raw := range []string{"q=x&q=y", "q=", "q=x&mode=", "q=x&unknown=y", "q=x&libraryId=/etc/passwd", "q=x&limit=201", "q=x&recursive=1", "q=x&modifiedFrom=tomorrow"} {
		if _, e := parseFusedSearchQuery(raw); e == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
	for _, tc := range []struct {
		err    error
		status int
		code   string
	}{{search.ErrCursor, 400, "invalid_cursor"}, {search.ErrStale, 409, "search_cursor_stale"}, {search.ErrUnavailable, 503, "model_unavailable"}, {errors.New("private SQL/path"), 500, "internal_error"}} {
		mux := http.NewServeMux()
		registerFusedSearchRoute(mux, &fusedHTTPStub{err: tc.err}, catalogServiceStub{}, newSemanticSearchAdmission())
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/search/assets?q=x", nil))
		if w.Code != tc.status || !strings.Contains(w.Body.String(), tc.code) || strings.Contains(w.Body.String(), "private SQL") {
			t.Fatal(w.Code, w.Body.String())
		}
	}
}

func (s *fusedHTTPStub) Verify(context.Context, search.Request, search.Result) error {
	return s.verifyErr
}

func TestFusedHTTPOCRExcerptAndPostHydrationFence(t *testing.T) {
	stub := &fusedHTTPStub{result: search.Result{OCRState: "available", SemanticState: "not_requested", Hits: []search.Hit{{LibraryID: 7, AssetID: 9, Evidence: search.Evidence("", "car.jpg", "folder/car.jpg"), Sources: []search.Source{search.OCR}, OCR: []search.OCRExcerpt{{Line: 2, Text: "中文 <script>alert(1)</script>"}}}}}}
	cat := catalogServiceStub{getAsset: func(_ context.Context, id int64) (catalog.Asset, error) {
		return catalog.Asset{ID: id, LibraryID: 7, Name: "car.jpg", RelativePath: "folder/car.jpg"}, nil
	}}
	mux := http.NewServeMux()
	registerFusedSearchRoute(mux, stub, cat, newSemanticSearchAdmission())
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest("GET", "/api/v1/search/assets?q=test&mode=ocr", nil))
	if response.Code != 200 || strings.Contains(response.Body.String(), "<script>") {
		t.Fatal(response.Code, response.Body.String())
	}
	var page fusedPageResponse
	if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil || len(page.Items[0].OCR) != 1 || page.Items[0].OCR[0].Line != 2 {
		t.Fatal(page, err)
	}
	stub.verifyErr = search.ErrStale
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest("GET", "/api/v1/search/assets?q=test&mode=ocr", nil))
	if response.Code != 409 || strings.Contains(response.Body.String(), "script") {
		t.Fatal(response.Code, response.Body.String())
	}
}
