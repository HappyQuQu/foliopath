package api

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/media"
	"github.com/HappyQuQu/foliopath/internal/thumbnail"
)

type thumbnailServiceStub struct {
	delivery thumbnail.Delivery
	err      error
	seen     *thumbnail.Variant
}

func (stub thumbnailServiceStub) Get(
	_ context.Context,
	_ int64,
	variant thumbnail.Variant,
) (thumbnail.Delivery, error) {
	if stub.seen != nil {
		*stub.seen = variant
	}
	return stub.delivery, stub.err
}

type readSeekCloser struct{ *bytes.Reader }

func (readSeekCloser) Close() error { return nil }

func TestThumbnailHTTPReadyAndConditionalResponses(t *testing.T) {
	service := thumbnailServiceStub{delivery: thumbnail.Delivery{
		Status:       thumbnail.DeliveryReady,
		Content:      readSeekCloser{bytes.NewReader([]byte("webp"))},
		ContentBytes: 4, ETag: `"thumb-test"`,
	}}
	mux := http.NewServeMux()
	registerThumbnailRoute(mux, service)

	ready := httptest.NewRecorder()
	mux.ServeHTTP(ready, httptest.NewRequest(
		http.MethodGet, "/api/v1/assets/ast_7/thumbnail?variant=grid", nil,
	))
	body, _ := io.ReadAll(ready.Result().Body)
	if ready.Code != http.StatusOK || string(body) != "webp" ||
		ready.Header().Get("Content-Type") != "image/webp" ||
		ready.Header().Get("ETag") != `"thumb-test"` ||
		ready.Header().Get("Cache-Control") != "private, max-age=31536000, immutable" ||
		ready.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("ready response = %d, %q, %#v", ready.Code, body, ready.Header())
	}

	service.delivery.Content = readSeekCloser{bytes.NewReader([]byte("webp"))}
	notModified := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodGet, "/api/v1/assets/ast_7/thumbnail", nil,
	)
	request.Header.Set("If-None-Match", `W/"other", "thumb-test"`)
	mux = http.NewServeMux()
	registerThumbnailRoute(mux, service)
	mux.ServeHTTP(notModified, request)
	if notModified.Code != http.StatusNotModified || notModified.Body.Len() != 0 ||
		notModified.Header().Get("ETag") != `"thumb-test"` {
		t.Fatalf("conditional response = %d, %q", notModified.Code, notModified.Body)
	}
}

func TestThumbnailHTTPMapsPendingFailedOfflineAndInvalidQuery(t *testing.T) {
	tests := []struct {
		name       string
		delivery   thumbnail.Delivery
		target     string
		wantStatus int
		wantCode   string
	}{
		{
			name: "pending",
			delivery: thumbnail.Delivery{
				Status: thumbnail.DeliveryRunning, RetryAfterMS: 1000,
			},
			target:     "/api/v1/assets/ast_7/thumbnail?v=0123456789abcdef0123456789abcdef",
			wantStatus: http.StatusAccepted,
		},
		{
			name: "failed",
			delivery: thumbnail.Delivery{
				Status: thumbnail.DeliveryFailed, ErrorCode: media.ErrorInvalidMedia,
			},
			target:     "/api/v1/assets/ast_7/thumbnail?variant=grid",
			wantStatus: http.StatusUnprocessableEntity, wantCode: "invalid_media",
		},
		{
			name: "offline",
			delivery: thumbnail.Delivery{
				Status:    thumbnail.DeliveryOffline,
				ErrorCode: media.ProcessingErrorCode("source_offline"),
			},
			target:     "/api/v1/assets/ast_7/thumbnail",
			wantStatus: http.StatusConflict, wantCode: "source_offline",
		},
		{
			name:       "invalid variant",
			delivery:   thumbnail.Delivery{Status: thumbnail.DeliveryQueued},
			target:     "/api/v1/assets/ast_7/thumbnail?variant=viewer",
			wantStatus: http.StatusBadRequest, wantCode: codeInvalidRequest,
		},
		{
			name:       "invalid cache version",
			delivery:   thumbnail.Delivery{Status: thumbnail.DeliveryQueued},
			target:     "/api/v1/assets/ast_7/thumbnail?v=old",
			wantStatus: http.StatusBadRequest, wantCode: codeInvalidRequest,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mux := http.NewServeMux()
			registerThumbnailRoute(mux, thumbnailServiceStub{delivery: test.delivery})
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, test.target, nil))
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d; body = %s", response.Code, response.Body)
			}
			if test.wantCode != "" {
				assertSafeErrorResponse(t, response, test.wantCode)
			}
		})
	}
}

func TestThumbnailHTTPPassesStoryboardVariantToService(t *testing.T) {
	var seen thumbnail.Variant
	mux := http.NewServeMux()
	registerThumbnailRoute(mux, thumbnailServiceStub{
		delivery: thumbnail.Delivery{
			Status: thumbnail.DeliveryQueued, RetryAfterMS: 1000,
		},
		seen: &seen,
	})
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(
		http.MethodGet,
		"/api/v1/assets/ast_7/thumbnail?variant=storyboard",
		nil,
	))
	if response.Code != http.StatusAccepted ||
		seen != thumbnail.VariantStoryboard ||
		!bytes.Contains(response.Body.Bytes(), []byte(`"variant":"storyboard"`)) {
		t.Fatalf(
			"storyboard response = %d %q, variant %q",
			response.Code,
			response.Body,
			seen,
		)
	}
}
