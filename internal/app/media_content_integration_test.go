package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

type runtimeContentResponse struct {
	status  int
	headers http.Header
	body    []byte
	code    string
}

func TestComposedMediaContentAuthenticationRangeAndSourceFailures(t *testing.T) {
	mediaRoot := t.TempDir()
	dataRoot := t.TempDir()
	content := []byte("not-a-decoded-jpeg-but-the-original-bytes")
	mediaPath := filepath.Join(mediaRoot, "family", "photo.jpg")
	if err := os.MkdirAll(filepath.Dir(mediaPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mediaPath, content, 0o640); err != nil {
		t.Fatal(err)
	}
	indexedTime := time.Date(2026, time.July, 27, 12, 34, 56, 0, time.UTC)
	if err := os.Chtimes(mediaPath, indexedTime, indexedTime); err != nil {
		t.Fatal(err)
	}
	libraryID := seedRuntimeLibrary(t, dataRoot, mediaRoot)

	application, err := composeConfiguration(
		Input{Version: "integration"},
		configuration{
			listenAddress: "127.0.0.1:0",
			mediaRoot:     mediaRoot,
			dataRoot:      dataRoot,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() { result <- application.run(ctx) }()
	defer func() {
		cancel()
		select {
		case runErr := <-result:
			if runErr != nil {
				t.Errorf("application.run() error = %v", runErr)
			}
		case <-time.After(runtimeIntegrationTimeout):
			t.Error("application did not stop")
		}
	}()

	address := waitForListenAddress(t, application.http)
	client := &http.Client{Timeout: runtimeIntegrationTimeout}
	setup := runtimeAuthenticationRequest(
		t, client, address, http.MethodPost, "/api/v1/auth/setup",
		`{"username":"Administrator","displayName":"Administrator","password":"correct horse battery staple"}`,
		"", "",
	)
	if setup.StatusCode != http.StatusCreated {
		t.Fatalf("setup response = %#v", setup)
	}
	assetID := waitForRuntimeContentAsset(t, dataRoot, libraryID)
	target := "/api/v1/assets/ast_" + strconv.FormatInt(assetID, 10) + "/content"

	unauthorized := requestRuntimeContent(t, client, address, http.MethodGet, target, "", nil)
	if unauthorized.status != http.StatusUnauthorized ||
		unauthorized.code != "authentication_required" ||
		strings.Contains(string(unauthorized.body), string(content)) {
		t.Fatalf("unauthorized content response = %#v", unauthorized)
	}

	full := requestRuntimeContent(t, client, address, http.MethodGet, target, setup.Cookie, nil)
	if full.status != http.StatusOK || string(full.body) != string(content) {
		t.Fatalf("full content response = %#v", full)
	}
	if full.headers.Get("Content-Type") != "image/jpeg" ||
		full.headers.Get("Content-Length") != strconv.Itoa(len(content)) ||
		full.headers.Get("Accept-Ranges") != "bytes" ||
		full.headers.Get("X-Content-Type-Options") != "nosniff" ||
		full.headers.Get("ETag") == "" ||
		full.headers.Get("Last-Modified") != indexedTime.Format(http.TimeFormat) ||
		strings.Contains(full.headers.Get("Content-Disposition"), "family/") {
		t.Fatalf("full content headers = %#v", full.headers)
	}

	head := requestRuntimeContent(t, client, address, http.MethodHead, target, setup.Cookie, map[string]string{
		"Range": "bytes=2-5",
	})
	if head.status != http.StatusOK || len(head.body) != 0 ||
		head.headers.Get("Content-Length") != strconv.Itoa(len(content)) {
		t.Fatalf("HEAD response = %#v", head)
	}
	partial := requestRuntimeContent(t, client, address, http.MethodGet, target, setup.Cookie, map[string]string{
		"Range": "bytes=2-5",
	})
	if partial.status != http.StatusPartialContent ||
		string(partial.body) != string(content[2:6]) ||
		partial.headers.Get("Content-Range") != "bytes 2-5/"+strconv.Itoa(len(content)) {
		t.Fatalf("partial response = %#v", partial)
	}
	notModified := requestRuntimeContent(t, client, address, http.MethodGet, target, setup.Cookie, map[string]string{
		"If-None-Match": full.headers.Get("ETag"),
	})
	if notModified.status != http.StatusNotModified || len(notModified.body) != 0 {
		t.Fatalf("conditional response = %#v", notModified)
	}
	ifRangeFallback := requestRuntimeContent(t, client, address, http.MethodGet, target, setup.Cookie, map[string]string{
		"Range":    "bytes=2-5",
		"If-Range": `"stale"`,
	})
	if ifRangeFallback.status != http.StatusOK ||
		string(ifRangeFallback.body) != string(content) {
		t.Fatalf("If-Range fallback = %#v", ifRangeFallback)
	}
	invalidRange := requestRuntimeContent(t, client, address, http.MethodGet, target, setup.Cookie, map[string]string{
		"Range": "bytes=0-1,3-4",
	})
	if invalidRange.status != http.StatusRequestedRangeNotSatisfiable ||
		invalidRange.code != "range_not_satisfiable" ||
		invalidRange.headers.Get("Content-Range") != "bytes */"+strconv.Itoa(len(content)) {
		t.Fatalf("invalid range response = %#v", invalidRange)
	}

	poisonRuntimeAssetPath(t, dataRoot, assetID, "../outside/secret.jpg")
	poisoned := requestRuntimeContent(t, client, address, http.MethodGet, target, setup.Cookie, nil)
	assertSafeContentFailure(t, poisoned, "source_unreadable", mediaRoot, "../outside/secret.jpg")
	poisonRuntimeAssetPath(t, dataRoot, assetID, "photo.jpg")

	changed := []byte("changed-source-with-a-new-fingerprint")
	if err := os.WriteFile(mediaPath, changed, 0o640); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(mediaPath, indexedTime.Add(time.Hour), indexedTime.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	sourceChanged := requestRuntimeContent(t, client, address, http.MethodGet, target, setup.Cookie, nil)
	assertSafeContentFailure(t, sourceChanged, "source_missing", mediaRoot, string(changed))

	if err := os.Remove(mediaPath); err != nil {
		t.Fatal(err)
	}
	missing := requestRuntimeContent(t, client, address, http.MethodGet, target, setup.Cookie, nil)
	assertSafeContentFailure(t, missing, "source_unreadable", mediaRoot, string(content))

	setRuntimeLibraryStatus(t, dataRoot, libraryID, "offline")
	offline := requestRuntimeContent(t, client, address, http.MethodGet, target, setup.Cookie, nil)
	assertSafeContentFailure(t, offline, "source_offline", mediaRoot, string(content))
}

func waitForRuntimeContentAsset(t *testing.T, dataRoot string, libraryID int64) int64 {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(dataRoot, databaseFilename))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	deadline := time.Now().Add(runtimeIntegrationTimeout)
	for time.Now().Before(deadline) {
		var generation, assetID int64
		var status string
		err := db.QueryRow(`
            SELECT current_generation, status
            FROM libraries WHERE id = ?`, libraryID,
		).Scan(&generation, &status)
		if err == nil && generation >= 2 && status == "ready" {
			err = db.QueryRow(`
                SELECT id FROM assets
                WHERE library_id = ? AND relative_path = 'photo.jpg'`,
				libraryID,
			).Scan(&assetID)
			if err == nil {
				return assetID
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("startup reconciliation did not publish the content asset")
	return 0
}

func requestRuntimeContent(
	t *testing.T,
	client *http.Client,
	address, method, target, cookie string,
	headers map[string]string,
) runtimeContentResponse {
	t.Helper()
	request, err := http.NewRequest(method, "http://"+address+target, nil)
	if err != nil {
		t.Fatal(err)
	}
	if cookie != "" {
		request.AddCookie(&http.Cookie{Name: "foliopath_session", Value: cookie})
	}
	for name, value := range headers {
		request.Header.Set(name, value)
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	_ = json.Unmarshal(body, &document)
	return runtimeContentResponse{
		status: response.StatusCode, headers: response.Header.Clone(),
		body: body, code: document.Error.Code,
	}
}

func poisonRuntimeAssetPath(t *testing.T, dataRoot string, assetID int64, relativePath string) {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(dataRoot, databaseFilename))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(
		`UPDATE assets SET relative_path = ?, name = ? WHERE id = ?`,
		relativePath, filepath.Base(relativePath), assetID,
	); err != nil {
		t.Fatal(err)
	}
}

func setRuntimeLibraryStatus(t *testing.T, dataRoot string, libraryID int64, status string) {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(dataRoot, databaseFilename))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`UPDATE libraries SET status = ? WHERE id = ?`, status, libraryID); err != nil {
		t.Fatal(err)
	}
}

func assertSafeContentFailure(
	t *testing.T,
	response runtimeContentResponse,
	wantCode string,
	forbidden ...string,
) {
	t.Helper()
	if response.status != http.StatusConflict || response.code != wantCode {
		t.Fatalf("content failure = %#v, want 409/%s", response, wantCode)
	}
	exposed := string(response.body) + response.headers.Get("Content-Disposition")
	for _, value := range forbidden {
		if value != "" && strings.Contains(exposed, value) {
			t.Fatalf("content failure exposed %q: %s", value, exposed)
		}
	}
}
