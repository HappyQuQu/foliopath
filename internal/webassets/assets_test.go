package webassets

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func TestHandlerServesAssetsAndHistoryFallbackWithoutClaimingAPIRoutes(t *testing.T) {
	site := fstest.MapFS{
		"index.html":             {Data: []byte("<main>FolioPath</main>")},
		"assets/app-123.js":      {Data: []byte("console.log('folio')")},
		"assets/empty-dir/.keep": {Data: nil},
	}
	next := http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusTeapot)
	})
	handler := newHandler(site, next)

	tests := []struct {
		name         string
		method       string
		target       string
		status       int
		cacheControl string
		contentType  string
	}{
		{
			name:         "root",
			method:       http.MethodGet,
			target:       "/",
			status:       http.StatusOK,
			cacheControl: "no-cache",
			contentType:  "text/html; charset=utf-8",
		},
		{
			name:         "history fallback",
			method:       http.MethodGet,
			target:       "/libraries/lib_1/browse/family",
			status:       http.StatusOK,
			cacheControl: "no-cache",
			contentType:  "text/html; charset=utf-8",
		},
		{
			name:         "hashed asset",
			method:       http.MethodGet,
			target:       "/assets/app-123.js",
			status:       http.StatusOK,
			cacheControl: "public, max-age=31536000, immutable",
			contentType:  "text/javascript; charset=utf-8",
		},
		{
			name:   "missing asset",
			method: http.MethodGet,
			target: "/assets/missing.js",
			status: http.StatusTeapot,
		},
		{
			name:   "api",
			method: http.MethodGet,
			target: "/api/v1/status",
			status: http.StatusTeapot,
		},
		{
			name:   "health",
			method: http.MethodGet,
			target: "/health/ready",
			status: http.StatusTeapot,
		},
		{
			name:   "mutation",
			method: http.MethodPost,
			target: "/login",
			status: http.StatusTeapot,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(test.method, test.target, nil)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)

			if response.Code != test.status {
				t.Fatalf("status = %d, want %d", response.Code, test.status)
			}
			if got := response.Header().Get("Cache-Control"); got != test.cacheControl {
				t.Fatalf("Cache-Control = %q, want %q", got, test.cacheControl)
			}
			if got := response.Header().Get("Content-Type"); got != test.contentType {
				t.Fatalf("Content-Type = %q, want %q", got, test.contentType)
			}
			if test.status == http.StatusOK &&
				response.Header().Get("X-Content-Type-Options") != "nosniff" {
				t.Fatal("successful static response omitted nosniff")
			}
		})
	}
}

func TestHandlerFallsThroughWhenProductionBuildIsAbsent(t *testing.T) {
	site := fstest.MapFS{".gitkeep": &fstest.MapFile{}}
	next := http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusServiceUnavailable)
	})
	response := httptest.NewRecorder()

	newHandler(fs.FS(site), next).ServeHTTP(
		response,
		httptest.NewRequest(http.MethodGet, "/", nil),
	)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
}
