package api

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"net/http"
	"net/http/httptest"

	"github.com/HappyQuQu/foliopath/internal/media"
)

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
