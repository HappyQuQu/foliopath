package integration_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/files"
)

var (
	errHarnessInvalidAssetID     = errors.New("invalid asset id")
	errHarnessAssetNotFound      = errors.New("asset not found")
	errHarnessContentUnavailable = errors.New("asset content unavailable")
)

// mediaContentUseCase is a test-only stand-in for the future media capability's
// inbound service boundary. The HTTP harness knows only opaque asset IDs and
// this interface; it never resolves or opens filesystem paths itself.
type mediaContentUseCase interface {
	OpenContent(context.Context, string) (*harnessContent, error)
}

type harnessContent struct {
	file        *os.File
	name        string
	contentType string
	modifiedAt  time.Time
	size        int64
}

func (content *harnessContent) Close() error {
	return content.file.Close()
}

type harnessAssetRecord struct {
	relativePath string
	name         string
	contentType  string
}

// filesystemMediaContent is the test capability implementation. Its records
// deliberately include poisoned relative paths so the HTTP-to-files boundary
// can be tested without introducing a production handler or accepting paths
// from HTTP.
type filesystemMediaContent struct {
	root   *files.Root
	assets map[string]harnessAssetRecord
}

func (service *filesystemMediaContent) OpenContent(
	ctx context.Context,
	assetID string,
) (*harnessContent, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !validHarnessAssetID(assetID) {
		return nil, errHarnessInvalidAssetID
	}
	record, ok := service.assets[assetID]
	if !ok {
		return nil, errHarnessAssetNotFound
	}
	file, err := service.root.Open(record.relativePath)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errHarnessContentUnavailable, err)
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("%w: stat failed", errHarnessContentUnavailable)
	}
	return &harnessContent{
		file:        file,
		name:        record.name,
		contentType: record.contentType,
		modifiedAt:  info.ModTime(),
		size:        info.Size(),
	}, nil
}

func validHarnessAssetID(assetID string) bool {
	if assetID == "" || len(assetID) > 64 {
		return false
	}
	for index := range len(assetID) {
		character := assetID[index]
		if (character >= 'a' && character <= 'z') ||
			(character >= 'A' && character <= 'Z') ||
			(character >= '0' && character <= '9') ||
			character == '-' ||
			character == '_' {
			continue
		}
		return false
	}
	return true
}

type harnessContentHandler struct {
	media mediaContentUseCase
}

func newHTTPContentBoundaryHarness(media mediaContentUseCase) http.Handler {
	handler := &harnessContentHandler{media: media}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/assets/{assetId}/content", handler.serveContent)
	mux.HandleFunc("HEAD /api/v1/assets/{assetId}/content", handler.serveContent)
	return mux
}

func (handler *harnessContentHandler) serveContent(
	response http.ResponseWriter,
	request *http.Request,
) {
	response.Header().Set("X-Content-Type-Options", "nosniff")
	if request.URL.RawQuery != "" {
		http.Error(response, "invalid request", http.StatusBadRequest)
		return
	}

	content, err := handler.media.OpenContent(request.Context(), request.PathValue("assetId"))
	if err != nil {
		switch {
		case errors.Is(err, errHarnessInvalidAssetID):
			http.Error(response, "invalid request", http.StatusBadRequest)
		case errors.Is(err, errHarnessAssetNotFound),
			errors.Is(err, errHarnessContentUnavailable):
			http.Error(response, "content unavailable", http.StatusNotFound)
		case errors.Is(err, context.Canceled):
			return
		default:
			http.Error(response, "content unavailable", http.StatusInternalServerError)
		}
		return
	}
	defer content.Close()

	rangeValues := request.Header.Values("Range")
	if len(rangeValues) > 1 || strings.Contains(strings.Join(rangeValues, ","), ",") {
		response.Header().Set("Content-Range", "bytes */"+strconv.FormatInt(content.size, 10))
		http.Error(response, "invalid range", http.StatusRequestedRangeNotSatisfiable)
		return
	}
	response.Header().Set("Content-Type", content.contentType)
	http.ServeContent(response, request, content.name, content.modifiedAt, content.file)
}

type httpBoundaryFixture struct {
	server      *httptest.Server
	client      *http.Client
	content     []byte
	allowedPath string
	outsidePath string
}

func newHTTPBoundaryFixture(t *testing.T) httpBoundaryFixture {
	t.Helper()

	base := t.TempDir()
	allowedPath := filepath.Join(base, "library")
	outsidePath := filepath.Join(base, "outside")
	if err := os.MkdirAll(filepath.Join(allowedPath, "albums"), 0o755); err != nil {
		t.Fatalf("create allowed path: %v", err)
	}
	if err := os.MkdirAll(outsidePath, 0o755); err != nil {
		t.Fatalf("create outside path: %v", err)
	}

	content := []byte("0123456789abcdefghijklmnopqrstuvwxyz")
	mediaPath := filepath.Join(allowedPath, "albums", "photo.jpg")
	if err := os.WriteFile(mediaPath, content, 0o644); err != nil {
		t.Fatalf("write media fixture: %v", err)
	}
	fixedModifiedAt := time.Date(2026, time.July, 23, 8, 9, 10, 0, time.UTC)
	if err := os.Chtimes(mediaPath, fixedModifiedAt, fixedModifiedAt); err != nil {
		t.Fatalf("set media fixture time: %v", err)
	}

	outsideMediaPath := filepath.Join(outsidePath, "secret.jpg")
	if err := os.WriteFile(outsideMediaPath, []byte("outside-secret-must-not-leak"), 0o644); err != nil {
		t.Fatalf("write outside fixture: %v", err)
	}
	if err := os.Symlink(outsideMediaPath, filepath.Join(allowedPath, "escape.jpg")); err != nil {
		t.Fatalf("create escape symlink: %v", err)
	}

	root, err := files.OpenRoot(allowedPath)
	if err != nil {
		t.Fatalf("open allowed media root: %v", err)
	}
	t.Cleanup(func() {
		if err := root.Close(); err != nil {
			t.Errorf("close allowed media root: %v", err)
		}
	})

	media := &filesystemMediaContent{
		root: root,
		assets: map[string]harnessAssetRecord{
			"photo-1": {
				relativePath: "albums/photo.jpg",
				name:         "photo.jpg",
				contentType:  "image/jpeg",
			},
			"poisoned-absolute": {
				relativePath: outsideMediaPath,
				name:         "photo.jpg",
				contentType:  "image/jpeg",
			},
			"poisoned-traversal": {
				relativePath: "../outside/secret.jpg",
				name:         "photo.jpg",
				contentType:  "image/jpeg",
			},
			"poisoned-double-encoding": {
				relativePath: "%252e%252e/outside/secret.jpg",
				name:         "photo.jpg",
				contentType:  "image/jpeg",
			},
			"poisoned-nul": {
				relativePath: "albums/photo.jpg\x00ignored",
				name:         "photo.jpg",
				contentType:  "image/jpeg",
			},
			"poisoned-symlink": {
				relativePath: "escape.jpg",
				name:         "photo.jpg",
				contentType:  "image/jpeg",
			},
		},
	}
	server := httptest.NewServer(newHTTPContentBoundaryHarness(media))
	t.Cleanup(server.Close)

	return httpBoundaryFixture{
		server: server,
		client: &http.Client{
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		content:     content,
		allowedPath: allowedPath,
		outsidePath: outsidePath,
	}
}

func (fixture httpBoundaryFixture) request(
	t *testing.T,
	method string,
	target string,
	headers map[string]string,
) *http.Response {
	t.Helper()
	request, err := http.NewRequest(method, fixture.server.URL+target, nil)
	if err != nil {
		t.Fatalf("create %s %q request: %v", method, target, err)
	}
	for name, value := range headers {
		request.Header.Set(name, value)
	}
	response, err := fixture.client.Do(request)
	if err != nil {
		t.Fatalf("perform %s %q request: %v", method, target, err)
	}
	return response
}

func readResponseBody(t *testing.T, response *http.Response) []byte {
	t.Helper()
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	return body
}

func assertNoBoundaryLeak(
	t *testing.T,
	fixture httpBoundaryFixture,
	response *http.Response,
	body []byte,
) {
	t.Helper()
	exposed := string(body) + fmt.Sprint(response.Header)
	for _, secret := range []string{
		fixture.allowedPath,
		fixture.outsidePath,
		"outside-secret-must-not-leak",
		"/etc/passwd",
		"../outside/secret.jpg",
		"%252e%252e",
		"escape.jpg",
	} {
		if strings.Contains(exposed, secret) {
			t.Fatalf("HTTP response exposed boundary detail %q: status=%d headers=%v body=%q",
				secret, response.StatusCode, response.Header, body)
		}
	}
}

func TestHTTPContentBoundarySupportsSafeReadSemantics(t *testing.T) {
	t.Parallel()
	fixture := newHTTPBoundaryFixture(t)

	t.Run("full GET", func(t *testing.T) {
		response := fixture.request(t, http.MethodGet, "/api/v1/assets/photo-1/content", nil)
		body := readResponseBody(t, response)
		if response.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want 200; body=%q", response.StatusCode, body)
		}
		if string(body) != string(fixture.content) {
			t.Fatalf("body = %q, want full media content", body)
		}
		if got := response.Header.Get("Accept-Ranges"); got != "bytes" {
			t.Errorf("Accept-Ranges = %q, want bytes", got)
		}
		if got := response.Header.Get("Content-Type"); got != "image/jpeg" {
			t.Errorf("Content-Type = %q, want image/jpeg", got)
		}
		if got := response.Header.Get("Content-Length"); got != strconv.Itoa(len(fixture.content)) {
			t.Errorf("Content-Length = %q, want %d", got, len(fixture.content))
		}
		if got := response.Header.Get("X-Content-Type-Options"); got != "nosniff" {
			t.Errorf("X-Content-Type-Options = %q, want nosniff", got)
		}
		if got := response.Header.Get("Last-Modified"); got == "" {
			t.Error("Last-Modified is empty")
		}
	})

	t.Run("single byte range", func(t *testing.T) {
		response := fixture.request(t, http.MethodGet, "/api/v1/assets/photo-1/content", map[string]string{
			"Range": "bytes=5-9",
		})
		body := readResponseBody(t, response)
		if response.StatusCode != http.StatusPartialContent {
			t.Fatalf("status = %d, want 206; body=%q", response.StatusCode, body)
		}
		if string(body) != string(fixture.content[5:10]) {
			t.Fatalf("range body = %q, want %q", body, fixture.content[5:10])
		}
		if got := response.Header.Get("Content-Range"); got != "bytes 5-9/"+strconv.Itoa(len(fixture.content)) {
			t.Errorf("Content-Range = %q", got)
		}
	})

	t.Run("unsatisfiable byte range", func(t *testing.T) {
		response := fixture.request(t, http.MethodGet, "/api/v1/assets/photo-1/content", map[string]string{
			"Range": "bytes=999-1000",
		})
		body := readResponseBody(t, response)
		if response.StatusCode != http.StatusRequestedRangeNotSatisfiable {
			t.Fatalf("status = %d, want 416; body=%q", response.StatusCode, body)
		}
		if got := response.Header.Get("Content-Range"); got != "bytes */"+strconv.Itoa(len(fixture.content)) {
			t.Errorf("Content-Range = %q", got)
		}
		assertNoBoundaryLeak(t, fixture, response, body)
	})

	t.Run("multiple ranges are rejected", func(t *testing.T) {
		response := fixture.request(t, http.MethodGet, "/api/v1/assets/photo-1/content", map[string]string{
			"Range": "bytes=0-1,3-4",
		})
		body := readResponseBody(t, response)
		if response.StatusCode != http.StatusRequestedRangeNotSatisfiable {
			t.Fatalf("status = %d, want 416; body=%q", response.StatusCode, body)
		}
		if got := response.Header.Get("Content-Range"); got != "bytes */"+strconv.Itoa(len(fixture.content)) {
			t.Errorf("Content-Range = %q", got)
		}
		assertNoBoundaryLeak(t, fixture, response, body)
	})

	t.Run("HEAD", func(t *testing.T) {
		response := fixture.request(t, http.MethodHead, "/api/v1/assets/photo-1/content", nil)
		body := readResponseBody(t, response)
		if response.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want 200", response.StatusCode)
		}
		if len(body) != 0 {
			t.Fatalf("HEAD body = %q, want empty", body)
		}
		if got := response.Header.Get("Content-Length"); got != strconv.Itoa(len(fixture.content)) {
			t.Errorf("Content-Length = %q, want %d", got, len(fixture.content))
		}
	})

	t.Run("If-Modified-Since", func(t *testing.T) {
		initial := fixture.request(t, http.MethodGet, "/api/v1/assets/photo-1/content", nil)
		lastModified := initial.Header.Get("Last-Modified")
		_ = readResponseBody(t, initial)
		if lastModified == "" {
			t.Fatal("initial Last-Modified is empty")
		}

		response := fixture.request(t, http.MethodGet, "/api/v1/assets/photo-1/content", map[string]string{
			"If-Modified-Since": lastModified,
		})
		body := readResponseBody(t, response)
		if response.StatusCode != http.StatusNotModified {
			t.Fatalf("status = %d, want 304; body=%q", response.StatusCode, body)
		}
		if len(body) != 0 {
			t.Fatalf("304 body = %q, want empty", body)
		}
	})
}

func TestHTTPContentBoundaryRejectsPathShapedInput(t *testing.T) {
	t.Parallel()
	fixture := newHTTPBoundaryFixture(t)

	for _, testCase := range []struct {
		name   string
		target string
	}{
		{
			name:   "absolute path in escaped asset id",
			target: "/api/v1/assets/%2Fetc%2Fpasswd/content",
		},
		{
			name:   "traversal in escaped asset id",
			target: "/api/v1/assets/%2e%2e%2Foutside%2Fsecret.jpg/content",
		},
		{
			name:   "double encoded traversal",
			target: "/api/v1/assets/%252e%252e%252Foutside%252Fsecret.jpg/content",
		},
		{
			name:   "encoded NUL",
			target: "/api/v1/assets/photo%00.jpg/content",
		},
		{
			name:   "path query is not an API input",
			target: "/api/v1/assets/photo-1/content?path=%2Fetc%2Fpasswd",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			response := fixture.request(t, http.MethodGet, testCase.target, nil)
			body := readResponseBody(t, response)
			if response.StatusCode != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400; body=%q", response.StatusCode, body)
			}
			assertNoBoundaryLeak(t, fixture, response, body)
		})
	}
}

func TestHTTPContentBoundaryRejectsPoisonedCatalogPaths(t *testing.T) {
	t.Parallel()
	fixture := newHTTPBoundaryFixture(t)

	for _, assetID := range []string{
		"poisoned-absolute",
		"poisoned-traversal",
		"poisoned-double-encoding",
		"poisoned-nul",
		"poisoned-symlink",
	} {
		t.Run(assetID, func(t *testing.T) {
			response := fixture.request(
				t,
				http.MethodGet,
				"/api/v1/assets/"+assetID+"/content",
				nil,
			)
			body := readResponseBody(t, response)
			if response.StatusCode != http.StatusNotFound {
				t.Fatalf("status = %d, want 404; body=%q", response.StatusCode, body)
			}
			assertNoBoundaryLeak(t, fixture, response, body)
		})
	}
}
