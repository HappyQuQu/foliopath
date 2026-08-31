package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
	"github.com/HappyQuQu/foliopath/internal/catalog"
	"github.com/HappyQuQu/foliopath/internal/semantic"
)

type semanticHTTPStub struct {
	settings  semantic.LibrarySettings
	operation aimodel.Operation
	updated   bool
	requested bool
	cleared   bool
	video     bool
}

type semanticSearchHTTPStub struct {
	request semantic.SearchRequest
	result  semantic.SearchResult
	err     error
	search  func(context.Context, semantic.SearchRequest) (semantic.SearchResult, error)
}

type videoSemanticSearchHTTPStub struct {
	request semantic.SearchRequest
	result  semantic.VideoSearchResult
	err     error
	search  func(context.Context, semantic.SearchRequest) (semantic.VideoSearchResult, error)
}

func (stub *videoSemanticSearchHTTPStub) Search(ctx context.Context, request semantic.SearchRequest) (semantic.VideoSearchResult, error) {
	stub.request = request
	if stub.search != nil {
		return stub.search(ctx, request)
	}
	return stub.result, stub.err
}

func (stub *semanticSearchHTTPStub) Search(ctx context.Context, request semantic.SearchRequest) (semantic.SearchResult, error) {
	stub.request = request
	if stub.search != nil {
		return stub.search(ctx, request)
	}
	return stub.result, stub.err
}

func (stub *semanticHTTPStub) GetLibrarySettings(context.Context, int64) (semantic.LibrarySettings, error) {
	return stub.settings, nil
}
func (stub *semanticHTTPStub) UpdateLibrarySettings(_ context.Context, _ int64, enabled bool, revision int64) (semantic.LibrarySettings, error) {
	stub.updated = enabled && revision == stub.settings.Revision
	stub.settings.Enabled, stub.settings.Revision = enabled, revision+1
	return stub.settings, nil
}
func (stub *semanticHTTPStub) RequestBackfill(_ context.Context, _ int64, mode semantic.JobMode, key string) (aimodel.Operation, bool, error) {
	stub.requested = mode == semantic.JobMissing && key == "semantic-request-001"
	return stub.operation, false, nil
}
func (stub *semanticHTTPStub) RequestClear(_ context.Context, libraryID, revision int64, key string) (aimodel.Operation, bool, error) {
	stub.cleared = libraryID == 7 && revision == stub.settings.Revision && key == "semantic-clear-001"
	return stub.operation, false, nil
}
func (stub *semanticHTTPStub) RequestVideoJob(_ context.Context, libraryID int64, mode semantic.JobMode, key string) (aimodel.Operation, bool, error) {
	stub.video = libraryID == 7 && mode == semantic.JobMissing && key == "video-semantic-001"
	return stub.operation, false, nil
}

func TestSemanticHTTPSettingsAndBackfillWireContract(t *testing.T) {
	now := time.Date(2026, 8, 28, 9, 0, 0, 0, time.UTC)
	total := int64(3)
	stub := &semanticHTTPStub{settings: semantic.LibrarySettings{LibraryID: 7, Enabled: false, State: semantic.LibraryDisabled, Revision: 2,
		ActiveGenerationID: "aig_generation123", Coverage: semantic.Coverage{Eligible: 3, Completed: 2, Failed: 1, Revision: 4}},
		operation: aimodel.Operation{ID: "aio_semantic123", Kind: aimodel.OperationSemanticMissing, State: aimodel.OperationQueued,
			Phase: aimodel.PhaseQueued, TotalItems: &total, Revision: 1, CreatedAt: now, UpdatedAt: now}}
	mux := http.NewServeMux()
	registerSemanticRoutes(mux, stub)
	registerVideoSemanticJobRoute(mux, stub)

	get := httptest.NewRecorder()
	mux.ServeHTTP(get, httptest.NewRequest(http.MethodGet, "/api/v1/libraries/lib_7/ai/semantic", nil))
	if get.Code != http.StatusOK || get.Header().Get("ETag") != `"semantic-library-7-r2"` || !strings.Contains(get.Body.String(), `"libraryId":"lib_7"`) || !strings.Contains(get.Body.String(), `"complete":false`) {
		t.Fatalf("GET = %d %#v %s", get.Code, get.Header(), get.Body.String())
	}

	putRequest := httptest.NewRequest(http.MethodPut, "/api/v1/libraries/lib_7/ai/semantic", strings.NewReader(`{"enabled":true}`))
	putRequest.Header.Set("Content-Type", "application/json")
	putRequest.Header.Set("If-Match", `"semantic-library-7-r2"`)
	put := httptest.NewRecorder()
	mux.ServeHTTP(put, putRequest)
	if put.Code != http.StatusOK || !stub.updated || put.Header().Get("ETag") != `"semantic-library-7-r3"` {
		t.Fatalf("PUT = %d updated %v %#v %s", put.Code, stub.updated, put.Header(), put.Body.String())
	}

	jobRequest := httptest.NewRequest(http.MethodPost, "/api/v1/libraries/lib_7/ai/semantic/jobs", strings.NewReader(`{"mode":"missing"}`))
	jobRequest.Header.Set("Content-Type", "application/json; charset=utf-8")
	jobRequest.Header.Set("Idempotency-Key", "semantic-request-001")
	job := httptest.NewRecorder()
	mux.ServeHTTP(job, jobRequest)
	if job.Code != http.StatusAccepted || !stub.requested || job.Header().Get("Location") != "/api/v1/ai/operations/aio_semantic123" || job.Header().Get("ETag") != `"aio_semantic123-r1"` {
		t.Fatalf("job = %d requested %v %#v %s", job.Code, stub.requested, job.Header(), job.Body.String())
	}

	clearRequest := httptest.NewRequest(http.MethodPost, "/api/v1/libraries/lib_7/ai/semantic/clear", strings.NewReader(`{"confirmation":"clear_semantic_data"}`))
	clearRequest.Header.Set("Content-Type", "application/json")
	clearRequest.Header.Set("Idempotency-Key", "semantic-clear-001")
	clearRequest.Header.Set("If-Match", `"semantic-library-7-r3"`)
	clearResponse := httptest.NewRecorder()
	mux.ServeHTTP(clearResponse, clearRequest)
	if clearResponse.Code != http.StatusAccepted || !stub.cleared {
		t.Fatalf("clear = %d cleared %v %#v %s", clearResponse.Code, stub.cleared, clearResponse.Header(), clearResponse.Body.String())
	}
	videoRequest := httptest.NewRequest(http.MethodPost, "/api/v1/libraries/lib_7/ai/video-semantic/jobs", strings.NewReader(`{"mode":"missing"}`))
	videoRequest.Header.Set("Content-Type", "application/json")
	videoRequest.Header.Set("Idempotency-Key", "video-semantic-001")
	videoResponse := httptest.NewRecorder()
	mux.ServeHTTP(videoResponse, videoRequest)
	if videoResponse.Code != http.StatusAccepted || !stub.video {
		t.Fatalf("video=%d requested=%v body=%s", videoResponse.Code, stub.video, videoResponse.Body.String())
	}
}

func TestSemanticHTTPRejectsMalformedWriteContracts(t *testing.T) {
	stub := &semanticHTTPStub{settings: semantic.LibrarySettings{LibraryID: 7, Revision: 1}}
	mux := http.NewServeMux()
	registerSemanticRoutes(mux, stub)
	for _, request := range []*http.Request{
		httptest.NewRequest(http.MethodPut, "/api/v1/libraries/lib_7/ai/semantic", strings.NewReader(`{"enabled":true}`)),
		httptest.NewRequest(http.MethodPost, "/api/v1/libraries/lib_7/ai/semantic/jobs", strings.NewReader(`{"mode":"video"}`)),
	} {
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, request)
		if response.Code != http.StatusPreconditionRequired && response.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, body %s", response.Code, response.Body.String())
		}
	}
}

func TestSemanticJobRejectsOversizedTrailingWhitespace(t *testing.T) {
	stub := &semanticHTTPStub{settings: semantic.LibrarySettings{LibraryID: 7, Revision: 1}}
	mux := http.NewServeMux()
	registerSemanticRoutes(mux, stub)
	body := `{"mode":"missing"}` + strings.Repeat(" ", maxAIModelJSONBytes)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/libraries/lib_7/ai/semantic/jobs", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "semantic-request-001")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest || stub.requested {
		t.Fatalf("response=%d body=%s requested=%t", response.Code, response.Body.String(), stub.requested)
	}
	assertSafeErrorResponse(t, response, codeInvalidRequest)
}

func TestSemanticSearchHTTPParsesScopeAndReturnsAssetPage(t *testing.T) {
	search := &semanticSearchHTTPStub{result: semantic.SearchResult{
		Matches:    []semantic.VectorMatch{{LibraryID: 7, AssetID: 9, Score: 0.75}},
		NextCursor: "opaque-cursor",
		Coverage:   semantic.Coverage{Eligible: 3, Completed: 2, Failed: 1, Revision: 4},
		Excluded:   []semantic.SearchExclusion{{LibraryID: 8, Reason: "offline", SettingsRevision: 2}},
	}}
	catalogStub := catalogServiceStub{getAsset: func(_ context.Context, assetID int64) (catalog.Asset, error) {
		return catalog.Asset{ID: assetID, LibraryID: 7, LibraryName: "Main", DirectoryID: 5,
			Name: "portrait.jpg", RelativePath: "set/portrait.jpg", Kind: catalog.KindImage,
			MIMEType: "image/jpeg", SizeBytes: 10, Availability: catalog.SourceAvailable}, nil
	}}
	mux := http.NewServeMux()
	registerSemanticSearchRoute(mux, search, catalogStub)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet,
		"/api/v1/semantic/assets?q=Red+Dress&libraryId=lib_7&directoryId=dir_5&recursive=true&limit=20", nil))
	if response.Code != http.StatusOK || search.request.Query != "Red Dress" || search.request.LibraryID != 7 ||
		search.request.DirectoryID != 5 || !search.request.Recursive || search.request.Limit != 20 {
		t.Fatalf("response=%d request=%#v body=%s", response.Code, search.request, response.Body.String())
	}
	var body semanticAssetPageResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.Items) != 1 || body.Items[0].ID != "ast_9" || body.NextCursor == nil || *body.NextCursor != "opaque-cursor" ||
		len(body.Coverage.ExcludedLibraries) != 1 || body.Coverage.ExcludedLibraries[0].LibraryID != "lib_8" || body.Coverage.Complete {
		t.Fatalf("body = %#v", body)
	}
}

func TestVideoSemanticSearchHTTPReturnsBestFrameEvidence(t *testing.T) {
	search := &videoSemanticSearchHTTPStub{result: semantic.VideoSearchResult{
		Matches:    []semantic.VideoVectorMatch{{LibraryID: 7, AssetID: 9, Score: .75, PlanSize: 10, Ordinal: 3, TimestampMS: 4200}},
		NextCursor: "opaque-video-cursor",
		Coverage:   semantic.VideoCoverage{Eligible: 3, Ready: 1, Degraded: 1, Failed: 1, Revision: 4},
	}}
	catalogStub := catalogServiceStub{getAsset: func(_ context.Context, assetID int64) (catalog.Asset, error) {
		return catalog.Asset{ID: assetID, LibraryID: 7, LibraryName: "Main", DirectoryID: 5,
			Name: "clip.mp4", RelativePath: "set/clip.mp4", Kind: catalog.KindVideo,
			MIMEType: "video/mp4", SizeBytes: 10, Availability: catalog.SourceAvailable}, nil
	}}
	mux := http.NewServeMux()
	registerVideoSemanticSearchRoute(mux, search, catalogStub)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet,
		"/api/v1/semantic/videos?q=Red+Dress&libraryId=lib_7&limit=20", nil))
	if response.Code != http.StatusOK || search.request.Query != "Red Dress" || search.request.LibraryID != 7 {
		t.Fatalf("response=%d request=%#v body=%s", response.Code, search.request, response.Body.String())
	}
	var body semanticVideoPageResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.Items) != 1 || body.Items[0].Asset.ID != "ast_9" || body.Items[0].MatchedFrame.Ordinal != 3 ||
		body.Items[0].MatchedFrame.TimestampMS != 4200 || body.Items[0].MatchedFrame.PlanSize != 10 ||
		body.NextCursor == nil || body.Coverage.Complete || body.Coverage.Degraded != 1 {
		t.Fatalf("body = %#v", body)
	}
}

func TestSemanticSearchHTTPRejectsCatalogProjectionDrift(t *testing.T) {
	for _, test := range []struct {
		name         string
		assets       []catalog.Asset
		wantCode     string
		wantHTTPCode int
	}{
		{name: "missing asset", assets: nil, wantCode: "semantic_cursor_stale", wantHTTPCode: http.StatusConflict},
		{name: "reordered asset", assets: []catalog.Asset{{ID: 10, LibraryID: 7, Availability: catalog.SourceAvailable}, {ID: 9, LibraryID: 7, Availability: catalog.SourceAvailable}}, wantCode: "semantic_cursor_stale", wantHTTPCode: http.StatusConflict},
		{name: "reused id in another library", assets: []catalog.Asset{{ID: 9, LibraryID: 8, Availability: catalog.SourceAvailable}, {ID: 10, LibraryID: 7, Availability: catalog.SourceAvailable}}, wantCode: "semantic_cursor_stale", wantHTTPCode: http.StatusConflict},
		{name: "library became offline", assets: []catalog.Asset{{ID: 9, LibraryID: 7, Availability: catalog.SourceOffline}, {ID: 10, LibraryID: 7, Availability: catalog.SourceAvailable}}, wantCode: "semantic_not_ready", wantHTTPCode: http.StatusConflict},
	} {
		t.Run(test.name, func(t *testing.T) {
			search := &semanticSearchHTTPStub{result: semantic.SearchResult{Matches: []semantic.VectorMatch{
				{LibraryID: 7, AssetID: 9, Score: .9},
				{LibraryID: 7, AssetID: 10, Score: .8},
			}}}
			catalogStub := catalogServiceStub{getAssetsByIDs: func(context.Context, []int64) ([]catalog.Asset, error) {
				return test.assets, nil
			}}
			mux := http.NewServeMux()
			registerSemanticSearchRoute(mux, search, catalogStub)
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/semantic/assets?q=portrait&libraryId=lib_7", nil))
			if response.Code != test.wantHTTPCode {
				t.Fatalf("response=%d body=%s, want %d", response.Code, response.Body.String(), test.wantHTTPCode)
			}
			assertSafeErrorResponse(t, response, test.wantCode)
		})
	}
}

func TestSemanticSearchHTTPRejectsInvalidAndMapsStableErrors(t *testing.T) {
	catalogStub := catalogServiceStub{}
	for _, test := range []struct {
		name   string
		url    string
		err    error
		status int
		code   string
	}{
		{name: "directory without library", url: "/api/v1/semantic/assets?q=x&directoryId=dir_5", status: 400, code: "invalid_request"},
		{name: "unknown parameter", url: "/api/v1/semantic/assets?q=x&score=1", status: 400, code: "invalid_request"},
		{name: "tokenizer control literal", url: "/api/v1/semantic/assets?q=portrait", err: semantic.ErrInvalidQuery, status: 400, code: "invalid_request"},
		{name: "invalid cursor", url: "/api/v1/semantic/assets?q=x", err: semantic.ErrInvalidSemanticCursor, status: 400, code: "invalid_cursor"},
		{name: "stale cursor", url: "/api/v1/semantic/assets?q=x", err: semantic.ErrSemanticCursorStale, status: 409, code: "semantic_cursor_stale"},
		{name: "model unavailable", url: "/api/v1/semantic/assets?q=x", err: semantic.ErrSemanticGenerationUnavailable, status: 503, code: "model_unavailable"},
		{name: "invalid internal snapshot", url: "/api/v1/semantic/assets?q=x", err: semantic.ErrInvalidSemanticSnapshot, status: 500, code: "internal_error"},
	} {
		t.Run(test.name, func(t *testing.T) {
			search := &semanticSearchHTTPStub{err: test.err}
			mux := http.NewServeMux()
			registerSemanticSearchRoute(mux, search, catalogStub)
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, test.url, nil))
			if response.Code != test.status || !strings.Contains(response.Body.String(), `"code":"`+test.code+`"`) {
				t.Fatalf("response=%d body=%s", response.Code, response.Body.String())
			}
		})
	}
	search := &semanticSearchHTTPStub{result: semantic.SearchResult{Matches: []semantic.VectorMatch{{AssetID: 99}}}}
	catalogStub.getAsset = func(context.Context, int64) (catalog.Asset, error) { return catalog.Asset{}, catalog.ErrAssetNotFound }
	mux := http.NewServeMux()
	registerSemanticSearchRoute(mux, search, catalogStub)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/semantic/assets?q=x", nil))
	if response.Code != http.StatusConflict || !strings.Contains(response.Body.String(), `"code":"semantic_cursor_stale"`) {
		t.Fatalf("asset race response=%d body=%s", response.Code, response.Body.String())
	}
}

func TestSemanticSearchHTTPRejectsOversizedRawQueryBeforeParsing(t *testing.T) {
	search := &semanticSearchHTTPStub{}
	mux := http.NewServeMux()
	registerSemanticSearchRoute(mux, search, catalogServiceStub{})
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet,
		"/api/v1/semantic/assets?q="+strings.Repeat("a", maxSemanticSearchRawQueryBytes), nil))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("response=%d body=%s", response.Code, response.Body.String())
	}
	assertSafeErrorResponse(t, response, codeInvalidRequest)
	if search.request.Query != "" {
		t.Fatalf("oversized raw query reached semantic service")
	}
}

func TestSemanticSearchHTTPFailsFastWhenInteractiveSlotIsBusy(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	search := &semanticSearchHTTPStub{search: func(context.Context, semantic.SearchRequest) (semantic.SearchResult, error) {
		close(entered)
		<-release
		return semantic.SearchResult{}, nil
	}}
	mux := http.NewServeMux()
	registerSemanticSearchRoute(mux, search, catalogServiceStub{})
	firstDone := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/semantic/assets?q=portrait", nil))
		firstDone <- response
	}()
	<-entered
	busy := httptest.NewRecorder()
	mux.ServeHTTP(busy, httptest.NewRequest(http.MethodGet, "/api/v1/semantic/assets?q=landscape", nil))
	if busy.Code != http.StatusTooManyRequests || busy.Header().Get("Retry-After") != "1" ||
		!strings.Contains(busy.Body.String(), `"code":"semantic_busy"`) {
		t.Fatalf("busy response=%d headers=%v body=%s", busy.Code, busy.Header(), busy.Body.String())
	}
	close(release)
	if first := <-firstDone; first.Code != http.StatusOK {
		t.Fatalf("first response=%d body=%s", first.Code, first.Body.String())
	}
}

func TestImageAndVideoSemanticSearchShareOneInteractiveSlot(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	image := &semanticSearchHTTPStub{search: func(context.Context, semantic.SearchRequest) (semantic.SearchResult, error) {
		close(entered)
		<-release
		return semantic.SearchResult{}, nil
	}}
	video := &videoSemanticSearchHTTPStub{}
	admission := newSemanticSearchAdmission()
	mux := http.NewServeMux()
	registerSemanticSearchRouteWithAdmission(mux, image, catalogServiceStub{}, admission)
	registerVideoSemanticSearchRouteWithAdmission(mux, video, catalogServiceStub{}, admission)
	firstDone := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/semantic/assets?q=portrait", nil))
		firstDone <- response
	}()
	<-entered
	busy := httptest.NewRecorder()
	mux.ServeHTTP(busy, httptest.NewRequest(http.MethodGet, "/api/v1/semantic/videos?q=clip", nil))
	if busy.Code != http.StatusTooManyRequests || busy.Header().Get("Retry-After") != "1" || video.request.Query != "" {
		t.Fatalf("busy response=%d headers=%v body=%s video request=%#v", busy.Code, busy.Header(), busy.Body.String(), video.request)
	}
	close(release)
	if first := <-firstDone; first.Code != http.StatusOK {
		t.Fatalf("first response=%d body=%s", first.Code, first.Body.String())
	}
}

func TestSemanticSearchRouteRequiresAuthenticationBeforeService(t *testing.T) {
	called := 0
	search := &semanticSearchHTTPStub{search: func(context.Context, semantic.SearchRequest) (semantic.SearchResult, error) {
		called++
		return semantic.SearchResult{}, nil
	}}
	base := RouteDependencies{
		Readiness:      func() Readiness { return Readiness{Ready: true} },
		SystemStatus:   unusedStatus,
		SemanticSearch: search,
		Catalog:        catalogServiceStub{},
	}
	base.Authentication = rejectingAuthentication()
	unauthorized := performRequest(testRoutes(t, base), "/api/v1/semantic/assets?q=private+portrait")
	if unauthorized.Code != http.StatusUnauthorized || called != 0 || strings.Contains(unauthorized.Body.String(), "private") {
		t.Fatalf("unauthorized=%d called=%d body=%s", unauthorized.Code, called, unauthorized.Body.String())
	}
	base.Authentication = acceptingAuthentication()
	authorized := performRequest(testRoutes(t, base), "/api/v1/semantic/assets?q=private+portrait")
	if authorized.Code != http.StatusOK || called != 1 || strings.Contains(authorized.Body.String(), "private") {
		t.Fatalf("authorized=%d called=%d body=%s", authorized.Code, called, authorized.Body.String())
	}
}

func TestSemanticSearchMasksQueryFromErrorsAndRequestLogs(t *testing.T) {
	const privateQuery = "private portrait secret"
	search := &semanticSearchHTTPStub{err: errors.New("encoder failed for " + privateQuery)}
	routes, err := NewRoutes(RouteDependencies{
		Readiness:      func() Readiness { return Readiness{Ready: true} },
		Authentication: acceptingAuthentication(),
		LibraryPaths:   &libraryPathServiceStub{},
		SystemStatus:   unusedStatus,
		SemanticSearch: search,
		Catalog:        catalogServiceStub{},
	})
	if err != nil {
		t.Fatal(err)
	}
	var logs bytes.Buffer
	handler := NewHandler(routes, slog.New(slog.NewJSONHandler(&logs, nil)))
	request := httptest.NewRequest(http.MethodGet, "/api/v1/semantic/assets?q=private+portrait+secret", nil)
	request.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "test-cookie"})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	for _, output := range []string{response.Body.String(), logs.String()} {
		if strings.Contains(output, privateQuery) || strings.Contains(output, "private+portrait") || strings.Contains(output, "encoder failed") {
			t.Fatalf("semantic query leaked: %s", output)
		}
	}
}
