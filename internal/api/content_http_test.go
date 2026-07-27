package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"net/http"
	"net/http/httptest"

	"github.com/HappyQuQu/foliopath/internal/media"
)

type contentServiceFunc func(context.Context, int64) (media.Content, error)

func (function contentServiceFunc) Open(
	ctx context.Context,
	assetID int64,
) (media.Content, error) {
	return function(ctx, assetID)
}

type contentServiceStub struct {
	path    string
	content media.Content
	err     error
}

func (stub contentServiceStub) Open(context.Context, int64) (media.Content, error) {
	if stub.err != nil {
		return media.Content{}, stub.err
	}
	file, err := os.Open(stub.path)
	if err != nil {
		return media.Content{}, err
	}
	content := stub.content
	content.File = file
	return content, nil
}

func TestContentHandlerServesFullHeadRangesAndValidators(t *testing.T) {
	sourcePath := filepath.Join(t.TempDir(), "照片.jpg")
	body := []byte("0123456789")
	if err := os.WriteFile(sourcePath, body, 0o600); err != nil {
		t.Fatal(err)
	}
	modified := time.Date(2026, 7, 27, 12, 0, 0, 123, time.UTC)
	handler := &contentHandler{
		service: contentServiceStub{path: sourcePath, content: media.Content{
			Name:       "family/照片.jpg",
			MIMEType:   "image/jpeg",
			SizeBytes:  int64(len(body)),
			ModifiedAt: modified,
			ETag:       `"current"`,
		}},
		slots: make(chan struct{}, 1),
	}
	tests := []struct {
		name         string
		method       string
		headers      map[string]string
		wantStatus   int
		wantBody     string
		contentRange string
	}{
		{name: "full", method: http.MethodGet, wantStatus: http.StatusOK, wantBody: string(body)},
		{name: "head ignores range", method: http.MethodHead, headers: map[string]string{"Range": "bytes=2-4"}, wantStatus: http.StatusOK},
		{name: "closed range", method: http.MethodGet, headers: map[string]string{"Range": "bytes=2-4"}, wantStatus: http.StatusPartialContent, wantBody: "234", contentRange: "bytes 2-4/10"},
		{name: "open range", method: http.MethodGet, headers: map[string]string{"Range": "bytes=7-"}, wantStatus: http.StatusPartialContent, wantBody: "789", contentRange: "bytes 7-9/10"},
		{name: "suffix range", method: http.MethodGet, headers: map[string]string{"Range": "bytes=-3"}, wantStatus: http.StatusPartialContent, wantBody: "789", contentRange: "bytes 7-9/10"},
		{name: "etag not modified", method: http.MethodGet, headers: map[string]string{"If-None-Match": `W/"current"`}, wantStatus: http.StatusNotModified},
		{name: "date not modified", method: http.MethodGet, headers: map[string]string{"If-Modified-Since": modified.Format(http.TimeFormat)}, wantStatus: http.StatusNotModified},
		{name: "if range mismatch is full", method: http.MethodGet, headers: map[string]string{"Range": "bytes=2-4", "If-Range": `"old"`}, wantStatus: http.StatusOK, wantBody: string(body)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(test.method, "/api/v1/assets/ast_7/content", nil)
			request.SetPathValue("assetId", "ast_7")
			for key, value := range test.headers {
				request.Header.Set(key, value)
			}
			response := httptest.NewRecorder()
			handler.handle(response, request)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", response.Code, test.wantStatus, response.Body.String())
			}
			if response.Body.String() != test.wantBody {
				t.Fatalf("body = %q, want %q", response.Body.String(), test.wantBody)
			}
			if response.Header().Get("Content-Range") != test.contentRange {
				t.Fatalf("Content-Range = %q, want %q", response.Header().Get("Content-Range"), test.contentRange)
			}
			if response.Header().Get("ETag") != `"current"` {
				t.Fatalf("ETag = %q", response.Header().Get("ETag"))
			}
			if !strings.Contains(response.Header().Get("Content-Disposition"), "inline") ||
				strings.Contains(response.Header().Get("Content-Disposition"), "family/") {
				t.Fatalf("Content-Disposition = %q", response.Header().Get("Content-Disposition"))
			}
		})
	}
}

func TestContentHandlerRejectsInvalidRangesAndQueries(t *testing.T) {
	sourcePath := filepath.Join(t.TempDir(), "video.mp4")
	if err := os.WriteFile(sourcePath, []byte("0123456789"), 0o600); err != nil {
		t.Fatal(err)
	}
	handler := &contentHandler{
		service: contentServiceStub{path: sourcePath, content: media.Content{
			Name: "video.mp4", MIMEType: "video/mp4", SizeBytes: 10,
			ModifiedAt: time.Now(), ETag: `"video"`,
		}},
		slots: make(chan struct{}, 1),
	}
	tests := []struct {
		name       string
		target     string
		rangeValue string
		wantStatus int
	}{
		{name: "query", target: "/api/v1/assets/ast_1/content?path=/etc/passwd", wantStatus: http.StatusBadRequest},
		{name: "multiple", target: "/api/v1/assets/ast_1/content", rangeValue: "bytes=0-1,3-4", wantStatus: http.StatusRequestedRangeNotSatisfiable},
		{name: "malformed", target: "/api/v1/assets/ast_1/content", rangeValue: "bytes=abc", wantStatus: http.StatusRequestedRangeNotSatisfiable},
		{name: "unsatisfiable", target: "/api/v1/assets/ast_1/content", rangeValue: "bytes=10-", wantStatus: http.StatusRequestedRangeNotSatisfiable},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, test.target, nil)
			request.SetPathValue("assetId", "ast_1")
			if test.rangeValue != "" {
				request.Header.Set("Range", test.rangeValue)
			}
			response := httptest.NewRecorder()
			handler.handle(response, request)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, test.wantStatus)
			}
			if test.wantStatus == http.StatusRequestedRangeNotSatisfiable {
				if response.Header().Get("Content-Range") != "bytes */10" {
					t.Fatalf("Content-Range = %q", response.Header().Get("Content-Range"))
				}
				var payload errorResponse
				if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
					t.Fatal(err)
				}
				if payload.Error.Code != "range_not_satisfiable" {
					t.Fatalf("code = %q", payload.Error.Code)
				}
			}
		})
	}
}

func TestContentHandlerBoundsConcurrencyAndReleasesCancelledRequests(t *testing.T) {
	slots := make(chan struct{}, 1)
	slots <- struct{}{}
	called := false
	handler := &contentHandler{
		service: contentServiceFunc(func(context.Context, int64) (media.Content, error) {
			called = true
			return media.Content{}, nil
		}),
		slots: slots,
	}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/assets/ast_1/content", nil)
	request.SetPathValue("assetId", "ast_1")
	response := httptest.NewRecorder()
	handler.handle(response, request)
	if response.Code != http.StatusTooManyRequests ||
		response.Header().Get("Retry-After") != "1" || called {
		t.Fatalf("saturated response = %d, headers=%v, called=%t",
			response.Code, response.Header(), called)
	}

	<-slots
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cancelledRequest := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/assets/ast_1/content",
		nil,
	).WithContext(ctx)
	cancelledRequest.SetPathValue("assetId", "ast_1")
	cancelledResponse := httptest.NewRecorder()
	handler.service = contentServiceFunc(func(ctx context.Context, _ int64) (media.Content, error) {
		return media.Content{}, ctx.Err()
	})
	handler.handle(cancelledResponse, cancelledRequest)
	if len(slots) != 0 || cancelledResponse.Body.Len() != 0 ||
		cancelledResponse.Header().Get("Content-Type") != "" {
		t.Fatalf("cancelled response wrote output or leaked slot: body=%q headers=%v slots=%d",
			cancelledResponse.Body.String(), cancelledResponse.Header(), len(slots))
	}
}

func TestCopyContentStopsCooperativelyAfterCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	reader := &cancelAfterFirstRead{
		reader: strings.NewReader("abcdef"),
		cancel: cancel,
	}
	var output bytes.Buffer
	err := copyContent(ctx, &output, reader, 6)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("copyContent() error = %v, want context.Canceled", err)
	}
	if output.String() != "ab" {
		t.Fatalf("copied bytes = %q, want first bounded chunk", output.String())
	}
}

type cancelAfterFirstRead struct {
	reader *strings.Reader
	cancel context.CancelFunc
	read   bool
}

func (reader *cancelAfterFirstRead) Read(buffer []byte) (int, error) {
	if len(buffer) > 2 {
		buffer = buffer[:2]
	}
	count, err := reader.reader.Read(buffer)
	if !reader.read {
		reader.read = true
		reader.cancel()
	}
	return count, err
}

func TestContentHandlerEmptySourceAndFailureMapping(t *testing.T) {
	emptyPath := filepath.Join(t.TempDir(), "empty.jpg")
	if err := os.WriteFile(emptyPath, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	handler := &contentHandler{
		service: contentServiceStub{path: emptyPath, content: media.Content{
			Name: "empty.jpg", MIMEType: "image/jpeg",
			ModifiedAt: time.Now(), ETag: `"empty"`,
		}},
		slots: make(chan struct{}, 1),
	}
	fullRequest := httptest.NewRequest(http.MethodGet, "/api/v1/assets/ast_1/content", nil)
	fullRequest.SetPathValue("assetId", "ast_1")
	fullResponse := httptest.NewRecorder()
	handler.handle(fullResponse, fullRequest)
	if fullResponse.Code != http.StatusOK ||
		fullResponse.Header().Get("Content-Length") != "0" ||
		fullResponse.Body.Len() != 0 {
		t.Fatalf("empty response = %d headers=%v body=%q",
			fullResponse.Code, fullResponse.Header(), fullResponse.Body.String())
	}
	rangeRequest := httptest.NewRequest(http.MethodGet, "/api/v1/assets/ast_1/content", nil)
	rangeRequest.SetPathValue("assetId", "ast_1")
	rangeRequest.Header.Set("Range", "bytes=0-")
	rangeResponse := httptest.NewRecorder()
	handler.handle(rangeResponse, rangeRequest)
	if rangeResponse.Code != http.StatusRequestedRangeNotSatisfiable ||
		rangeResponse.Header().Get("Content-Range") != "bytes */0" {
		t.Fatalf("empty range response = %d headers=%v",
			rangeResponse.Code, rangeResponse.Header())
	}

	tests := []struct {
		err        error
		wantStatus int
		wantCode   string
	}{
		{media.ErrContentAssetNotFound, http.StatusNotFound, "asset_not_found"},
		{media.ErrContentSourceOffline, http.StatusConflict, "source_offline"},
		{media.ErrContentSourceChanged, http.StatusConflict, "source_missing"},
		{media.ErrContentUnavailable, http.StatusConflict, "source_unreadable"},
		{errors.New("SELECT secret FROM /app/data/foliopath.db"), http.StatusInternalServerError, "internal_error"},
	}
	for _, test := range tests {
		t.Run(test.wantCode, func(t *testing.T) {
			handler.service = contentServiceStub{err: test.err}
			request := httptest.NewRequest(http.MethodGet, "/api/v1/assets/ast_1/content", nil)
			request.SetPathValue("assetId", "ast_1")
			response := httptest.NewRecorder()
			handler.handle(response, request)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, test.wantStatus)
			}
			var payload errorResponse
			if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
				t.Fatal(err)
			}
			if payload.Error.Code != test.wantCode ||
				strings.Contains(response.Body.String(), "/app/data") ||
				strings.Contains(response.Body.String(), "SELECT") {
				t.Fatalf("error response = %s", response.Body.String())
			}
		})
	}
}

func TestContentHandlerRejectsOversizeAndDuplicateConditionalHeaders(t *testing.T) {
	handler := &contentHandler{
		service: contentServiceFunc(func(context.Context, int64) (media.Content, error) {
			t.Fatal("invalid headers reached content service")
			return media.Content{}, nil
		}),
		slots: make(chan struct{}, 1),
	}
	tests := []struct {
		name   string
		header string
		values []string
	}{
		{name: "oversize etag", header: "If-None-Match", values: []string{strings.Repeat("a", maxETagHeaderBytes+1)}},
		{name: "duplicate date", header: "If-Modified-Since", values: []string{"one", "two"}},
		{name: "oversize if range", header: "If-Range", values: []string{strings.Repeat("a", maxETagHeaderBytes+1)}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/api/v1/assets/ast_1/content", nil)
			request.SetPathValue("assetId", "ast_1")
			for _, value := range test.values {
				request.Header.Add(test.header, value)
			}
			response := httptest.NewRecorder()
			handler.handle(response, request)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", response.Code)
			}
		})
	}
}
