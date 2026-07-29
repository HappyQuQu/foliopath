package app

import (
	"bytes"
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

	"github.com/HappyQuQu/foliopath/internal/media"
	sqlitestore "github.com/HappyQuQu/foliopath/internal/store/sqlite"
	"github.com/HappyQuQu/foliopath/internal/thumbnail"
	"github.com/HappyQuQu/foliopath/internal/thumbnail/cachefs"
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

func TestComposedStoryboardAuthenticationDeliveryAndStateMapping(t *testing.T) {
	mediaRoot := t.TempDir()
	dataRoot := t.TempDir()
	original := []byte("synthetic immutable storyboard source")
	writeRuntimeFixture(t, mediaRoot, "family/clip.mp4", string(original))
	libraryID := seedRuntimeLibrary(t, dataRoot, mediaRoot)
	assetID, storyboardBytes := seedReadyRuntimeStoryboard(
		t,
		dataRoot,
		libraryID,
	)

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
		t,
		client,
		address,
		http.MethodPost,
		"/api/v1/auth/setup",
		`{"username":"Administrator","displayName":"Administrator","password":"correct horse battery staple"}`,
		"",
		"",
	)
	if setup.StatusCode != http.StatusCreated {
		t.Fatalf("setup response = %#v", setup)
	}
	target := "/api/v1/assets/ast_" +
		strconv.FormatInt(assetID, 10) +
		"/thumbnail?variant=storyboard"

	unauthorized := requestRuntimeContent(
		t, client, address, http.MethodGet, target, "", nil,
	)
	if unauthorized.status != http.StatusUnauthorized ||
		unauthorized.code != "authentication_required" ||
		bytes.Contains(unauthorized.body, storyboardBytes) {
		t.Fatalf("unauthorized storyboard response = %#v", unauthorized)
	}

	ready := requestRuntimeContent(
		t, client, address, http.MethodGet, target, setup.Cookie, nil,
	)
	if ready.status != http.StatusOK ||
		!bytes.Equal(ready.body, storyboardBytes) ||
		ready.headers.Get("Content-Type") != "image/webp" ||
		ready.headers.Get("Cache-Control") !=
			"private, max-age=31536000, immutable" ||
		ready.headers.Get("X-Content-Type-Options") != "nosniff" ||
		ready.headers.Get("ETag") == "" {
		t.Fatalf("ready storyboard response = %#v", ready)
	}
	conditional := requestRuntimeContent(
		t,
		client,
		address,
		http.MethodGet,
		target,
		setup.Cookie,
		map[string]string{"If-None-Match": ready.headers.Get("ETag")},
	)
	if conditional.status != http.StatusNotModified ||
		len(conditional.body) != 0 ||
		conditional.headers.Get("ETag") != ready.headers.Get("ETag") {
		t.Fatalf("conditional storyboard response = %#v", conditional)
	}
	detail := runtimeAuthenticationRequest(
		t,
		client,
		address,
		http.MethodGet,
		"/api/v1/assets/ast_"+strconv.FormatInt(assetID, 10),
		"",
		setup.Cookie,
		"",
	)
	if detail.StatusCode != http.StatusOK ||
		!strings.Contains(detail.Body, `"storyboard":{"status":"ready"`) ||
		!strings.Contains(detail.Body, `"frameCount":10`) ||
		!strings.Contains(detail.Body, `"columns":5`) ||
		!strings.Contains(detail.Body, `"rows":2`) {
		t.Fatalf("storyboard asset detail = %#v", detail)
	}

	setRuntimeLibraryStatus(t, dataRoot, libraryID, "offline")
	offline := requestRuntimeContent(
		t, client, address, http.MethodGet, target, setup.Cookie, nil,
	)
	if offline.status != http.StatusConflict ||
		offline.code != "source_offline" ||
		strings.Contains(string(offline.body), mediaRoot) {
		t.Fatalf("offline storyboard response = %#v", offline)
	}
	setRuntimeLibraryStatus(t, dataRoot, libraryID, "ready")

	setRuntimeStoryboardState(t, dataRoot, assetID, "pending", "")
	pending := requestRuntimeContent(
		t, client, address, http.MethodGet, target, setup.Cookie, nil,
	)
	if pending.status != http.StatusAccepted ||
		!bytes.Contains(pending.body, []byte(`"variant":"storyboard"`)) {
		t.Fatalf("pending storyboard response = %#v", pending)
	}

	setRuntimeStoryboardState(
		t,
		dataRoot,
		assetID,
		"failed",
		string(media.ErrorInvalidMedia),
	)
	failed := requestRuntimeContent(
		t, client, address, http.MethodGet, target, setup.Cookie, nil,
	)
	if failed.status != http.StatusUnprocessableEntity ||
		failed.code != string(media.ErrorInvalidMedia) ||
		strings.Contains(string(failed.body), mediaRoot) {
		t.Fatalf("failed storyboard response = %#v", failed)
	}

	missing := requestRuntimeContent(
		t,
		client,
		address,
		http.MethodGet,
		"/api/v1/assets/ast_999999/thumbnail?variant=storyboard",
		setup.Cookie,
		nil,
	)
	if missing.status != http.StatusNotFound ||
		missing.code != "asset_not_found" {
		t.Fatalf("missing storyboard response = %#v", missing)
	}
	after, err := os.ReadFile(filepath.Join(mediaRoot, "family", "clip.mp4"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, original) {
		t.Fatal("storyboard HTTP delivery modified original source")
	}
}

func seedReadyRuntimeStoryboard(
	t *testing.T,
	dataRoot string,
	libraryID int64,
) (int64, []byte) {
	t.Helper()
	ctx := context.Background()
	inspector, err := sql.Open(
		"sqlite",
		filepath.Join(dataRoot, databaseFilename),
	)
	if err != nil {
		t.Fatal(err)
	}
	var assetID int64
	if err := inspector.QueryRow(`
        SELECT id FROM assets
        WHERE library_id = ? AND relative_path = 'clip.mp4'`,
		libraryID,
	).Scan(&assetID); err != nil {
		_ = inspector.Close()
		t.Fatal(err)
	}
	if err := inspector.Close(); err != nil {
		t.Fatal(err)
	}
	store, err := sqlitestore.Open(
		ctx,
		filepath.Join(dataRoot, databaseFilename),
		sqlitestore.Options{},
	)
	if err != nil {
		t.Fatal(err)
	}
	asset, err := store.GetAssetForDerivation(ctx, assetID)
	if err != nil {
		_ = store.Close()
		t.Fatal(err)
	}
	publisher, err := cachefs.New(filepath.Join(dataRoot, "cache"))
	if err != nil {
		_ = store.Close()
		t.Fatal(err)
	}
	gridBytes := []byte("grid-webp")
	gridDerivation, err := thumbnail.GridDerivation(
		libraryID,
		assetID,
		asset.SourceFingerprint,
	)
	if err != nil {
		_ = store.Close()
		t.Fatal(err)
	}
	gridPublished, err := publisher.Publish(ctx, gridDerivation, gridBytes)
	if err != nil {
		_ = store.Close()
		t.Fatal(err)
	}
	durationMS := int64(10_000)
	if err := store.CommitReady(ctx, thumbnail.Ready{
		AssetID:           assetID,
		SourceFingerprint: asset.SourceFingerprint,
		Result: media.ProcessingResult{
			Metadata: media.Metadata{
				Width:          1920,
				Height:         1080,
				DurationMS:     &durationMS,
				PlaybackStatus: media.PlaybackPlayable,
			},
			Thumbnail: media.Thumbnail{
				Bytes: gridBytes,
				Width: 320, Height: 180,
			},
		},
		CacheRelativePath: gridPublished.CacheRelativePath,
		ByteSize:          gridPublished.ByteSize,
		CreatedAtMS:       1,
	}); err != nil {
		_ = store.Close()
		t.Fatal(err)
	}
	storyboardBytes := []byte("RIFFxxxxWEBP")
	storyboardDerivation, err := thumbnail.StoryboardDerivation(
		libraryID,
		assetID,
		asset.SourceFingerprint,
	)
	if err != nil {
		_ = store.Close()
		t.Fatal(err)
	}
	storyboardPublished, err := publisher.Publish(
		ctx,
		storyboardDerivation,
		storyboardBytes,
	)
	if err != nil {
		_ = store.Close()
		t.Fatal(err)
	}
	storyboardResult := media.StoryboardResult{
		Bytes:      storyboardBytes,
		FrameCount: 10,
		Columns:    5,
		Rows:       2,
		CellWidth:  320,
		CellHeight: 180,
	}
	if err := store.CommitStoryboardReady(ctx, thumbnail.StoryboardReady{
		AssetID:           assetID,
		SourceFingerprint: asset.SourceFingerprint,
		Result:            storyboardResult,
		CacheRelativePath: storyboardPublished.CacheRelativePath,
		ByteSize:          storyboardPublished.ByteSize,
		CreatedAtMS:       1,
	}); err != nil {
		_ = store.Close()
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	db, err := sql.Open("sqlite", filepath.Join(dataRoot, databaseFilename))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`
        UPDATE media_jobs
        SET status = 'succeeded', finished_at_ms = 1
        WHERE asset_id = ? AND variant = 'grid'`,
		assetID,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
        INSERT INTO media_jobs(
            library_id, asset_id, variant, priority, transform_version,
            source_fingerprint, status, available_at_ms, attempt_count,
            created_at_ms, finished_at_ms
        ) VALUES (?, ?, 'storyboard', 100, ?, ?, 'succeeded', 0, 0, 0, 1)`,
		libraryID,
		assetID,
		thumbnail.StoryboardTransformVersion,
		asset.SourceFingerprint.String(),
	); err != nil {
		t.Fatal(err)
	}
	return assetID, storyboardBytes
}

func setRuntimeStoryboardState(
	t *testing.T,
	dataRoot string,
	assetID int64,
	status string,
	errorCode string,
) {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(dataRoot, databaseFilename))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var code any
	if errorCode != "" {
		code = errorCode
	}
	if _, err := db.Exec(`
        UPDATE thumbnails
        SET status = ?, error_code = ?, cache_rel_path = NULL,
            width = NULL, height = NULL, byte_size = NULL,
            created_at_ms = NULL, last_accessed_at_ms = NULL,
            frame_count = NULL, sprite_columns = NULL, sprite_rows = NULL,
            cell_width = NULL, cell_height = NULL
        WHERE asset_id = ? AND variant = 'storyboard'`,
		status,
		code,
		assetID,
	); err != nil {
		t.Fatal(err)
	}
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
