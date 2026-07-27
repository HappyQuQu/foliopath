package app

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/files"
	"github.com/HappyQuQu/foliopath/internal/library"
	"github.com/HappyQuQu/foliopath/internal/scanner"
	sqlitestore "github.com/HappyQuQu/foliopath/internal/store/sqlite"
)

const runtimeIntegrationTimeout = 10 * time.Second

func seedRuntimeLibrary(
	t *testing.T,
	dataRoot string,
	mediaRoot string,
) int64 {
	t.Helper()
	store, err := sqlitestore.Open(
		context.Background(),
		filepath.Join(dataRoot, databaseFilename),
		sqlitestore.Options{},
	)
	if err != nil {
		t.Fatal(err)
	}
	root, err := files.OpenRoot(mediaRoot)
	if err != nil {
		_ = store.Close()
		t.Fatal(err)
	}
	walker, err := files.NewScanWalker(root)
	if err != nil {
		_ = root.Close()
		_ = store.Close()
		t.Fatal(err)
	}
	libraries, err := library.NewService(store)
	if err != nil {
		_ = root.Close()
		_ = store.Close()
		t.Fatal(err)
	}
	record, err := libraries.Create(context.Background(), "Family", "family")
	if err != nil {
		_ = root.Close()
		_ = store.Close()
		t.Fatal(err)
	}
	scans, err := scanner.NewService(store, scanner.Config{})
	if err != nil {
		_ = root.Close()
		_ = store.Close()
		t.Fatal(err)
	}
	if _, err := scans.RunFullScan(context.Background(), scanner.FullScanRequest{
		LibraryID: record.ID,
		Trigger:   scanner.TriggerCreation,
		Walker:    walker,
	}); err != nil {
		_ = root.Close()
		_ = store.Close()
		t.Fatal(err)
	}
	if err := root.Close(); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	return record.ID
}

func runUntilRuntimeScanConverges(
	t *testing.T,
	dataRoot string,
	mediaRoot string,
	libraryID int64,
	wantTrigger scanner.Trigger,
) {
	t.Helper()
	application, err := composeConfiguration(
		Input{Version: "integration"},
		configuration{
			listenAddress: "127.0.0.1:0",
			mediaRoot:     mediaRoot,
			dataRoot:      dataRoot,
		},
	)
	if err != nil {
		t.Fatalf("composeConfiguration() error = %v", err)
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
			t.Error("application did not stop after scan recovery")
		}
	}()

	inspector, err := sql.Open("sqlite", filepath.Join(dataRoot, databaseFilename))
	if err != nil {
		t.Fatal(err)
	}
	defer inspector.Close()
	deadline := time.Now().Add(runtimeIntegrationTimeout)
	for time.Now().Before(deadline) {
		var generation int64
		var status string
		var trigger string
		var newCount, oldCount int
		err := inspector.QueryRow(`
			SELECT current_generation, status
			FROM libraries WHERE id = ?`,
			libraryID,
		).Scan(&generation, &status)
		if err == nil && generation == 2 && status == "ready" {
			err = inspector.QueryRow(`
				SELECT trigger_kind
				FROM scan_runs
				WHERE library_id = ? AND generation = 2`,
				libraryID,
			).Scan(&trigger)
			if err == nil {
				err = inspector.QueryRow(`
					SELECT
						COUNT(*) FILTER (WHERE relative_path = 'new.jpg'),
						COUNT(*) FILTER (WHERE relative_path = 'old.jpg')
					FROM assets
					WHERE library_id = ?`,
					libraryID,
				).Scan(&newCount, &oldCount)
			}
			if err == nil && trigger == string(wantTrigger) &&
				newCount == 1 && oldCount == 0 {
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("runtime scan did not converge with trigger %q", wantTrigger)
}

func TestComposedApplicationReconcilesLibrariesAndLateExpiredLeases(t *testing.T) {
	t.Run("startup admission", func(t *testing.T) {
		mediaRoot := t.TempDir()
		dataRoot := t.TempDir()
		writeRuntimeFixture(t, mediaRoot, "family/old.jpg", "old")
		libraryID := seedRuntimeLibrary(t, dataRoot, mediaRoot)
		if err := os.Remove(filepath.Join(mediaRoot, "family", "old.jpg")); err != nil {
			t.Fatal(err)
		}
		writeRuntimeFixture(t, mediaRoot, "family/new.jpg", "new")

		runUntilRuntimeScanConverges(
			t,
			dataRoot,
			mediaRoot,
			libraryID,
			scanner.TriggerStartup,
		)
	})

	t.Run("late expired lease", func(t *testing.T) {
		mediaRoot := t.TempDir()
		dataRoot := t.TempDir()
		writeRuntimeFixture(t, mediaRoot, "family/old.jpg", "old")
		libraryID := seedRuntimeLibrary(t, dataRoot, mediaRoot)
		if err := os.Remove(filepath.Join(mediaRoot, "family", "old.jpg")); err != nil {
			t.Fatal(err)
		}
		writeRuntimeFixture(t, mediaRoot, "family/new.jpg", "new")

		store, err := sqlitestore.Open(
			context.Background(),
			filepath.Join(dataRoot, databaseFilename),
			sqlitestore.Options{},
		)
		if err != nil {
			t.Fatal(err)
		}
		admitted, err := store.AdmitFullScan(
			context.Background(),
			libraryID,
			scanner.TriggerManual,
		)
		if err != nil {
			_ = store.Close()
			t.Fatal(err)
		}
		claimed, found, err := store.ClaimNextFullScan(
			context.Background(),
			2*time.Minute,
		)
		if err != nil || !found || claimed.ID != admitted.Run.ID {
			_ = store.Close()
			t.Fatalf("claimed scan = %#v, found %t, err %v", claimed, found, err)
		}
		if err := store.Close(); err != nil {
			t.Fatal(err)
		}
		inspector, err := sql.Open("sqlite", filepath.Join(dataRoot, databaseFilename))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := inspector.Exec(`
			UPDATE scan_runs
			SET lease_expires_at_ms = ?
			WHERE id = ?`,
			time.Now().Add(250*time.Millisecond).UnixMilli(),
			claimed.ID,
		); err != nil {
			_ = inspector.Close()
			t.Fatal(err)
		}
		if err := inspector.Close(); err != nil {
			t.Fatal(err)
		}

		runUntilRuntimeScanConverges(
			t,
			dataRoot,
			mediaRoot,
			libraryID,
			scanner.TriggerManual,
		)
	})
}

func writeRuntimeFixture(t *testing.T, root string, relative string, contents string) {
	t.Helper()
	filename := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filename, []byte(contents), 0o640); err != nil {
		t.Fatal(err)
	}
}

func TestComposedCreationScanIndexesEmptyDirectoriesAndCounts(t *testing.T) {
	mediaRoot := t.TempDir()
	dataRoot := t.TempDir()
	for relative, contents := range map[string]string{
		"family/root.jpg":                 "root",
		"family/album/direct.png":         "direct",
		"family/album/nested/deep.MOV":    "deep",
		"family/album/nested/ignored.txt": "ignored",
	} {
		filename := filepath.Join(mediaRoot, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filename, []byte(contents), 0o640); err != nil {
			t.Fatal(err)
		}
	}
	for _, relative := range []string{
		"family/empty-root",
		"family/album/empty-deep",
	} {
		if err := os.MkdirAll(
			filepath.Join(mediaRoot, filepath.FromSlash(relative)),
			0o755,
		); err != nil {
			t.Fatal(err)
		}
	}

	application, err := composeConfiguration(
		Input{Version: "integration"},
		configuration{
			listenAddress: "127.0.0.1:0",
			mediaRoot:     mediaRoot,
			dataRoot:      dataRoot,
		},
	)
	if err != nil {
		t.Fatalf("composeConfiguration() error = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() { result <- application.run(ctx) }()
	stopped := false
	defer func() {
		if stopped {
			return
		}
		cancel()
		select {
		case runErr := <-result:
			if runErr != nil {
				t.Errorf("application.run() cleanup error = %v", runErr)
			}
		case <-time.After(runtimeIntegrationTimeout):
			t.Error("application did not stop during test cleanup")
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
		cancel()
		t.Fatalf("setup response = %#v", setup)
	}
	created := runtimeLibraryCreateRequest(
		t,
		client,
		address,
		"Family",
		"family",
		"s2-103-create-family",
		setup.Cookie,
		setup.CSRFToken,
	)
	if created.StatusCode != http.StatusCreated {
		cancel()
		t.Fatalf("create response = %#v", created)
	}

	deadline := time.Now().Add(runtimeIntegrationTimeout)
	var current runtimeAuthenticationResponse
	for time.Now().Before(deadline) {
		current = runtimeAuthenticationRequest(
			t,
			client,
			address,
			http.MethodGet,
			created.Location,
			"",
			setup.Cookie,
			"",
		)
		if current.StatusCode == http.StatusOK &&
			strings.Contains(current.Body, `"status":"ready"`) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if current.StatusCode != http.StatusOK ||
		!strings.Contains(current.Body, `"status":"ready"`) ||
		!strings.Contains(current.Body, `"assetCount":3`) ||
		!strings.Contains(current.Body, `"directoryCount":5`) {
		cancel()
		t.Fatalf("creation scan result = %#v", current)
	}

	cancel()
	select {
	case runErr := <-result:
		if runErr != nil {
			t.Fatalf("application.run() error = %v", runErr)
		}
	case <-time.After(runtimeIntegrationTimeout):
		t.Fatal("application did not stop after creation scan")
	}
	stopped = true

	inspector, err := sql.Open(
		"sqlite",
		filepath.Join(dataRoot, databaseFilename),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer inspector.Close()
	expected := map[string][2]int64{
		"":                 {1, 3},
		"album":            {1, 2},
		"album/nested":     {1, 1},
		"album/empty-deep": {0, 0},
		"empty-root":       {0, 0},
	}
	rows, err := inspector.Query(`
		SELECT relative_path, direct_asset_count, recursive_asset_count
		FROM directories
		WHERE library_id = 1`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	seen := make(map[string][2]int64)
	for rows.Next() {
		var relative string
		var direct, recursive int64
		if err := rows.Scan(&relative, &direct, &recursive); err != nil {
			t.Fatal(err)
		}
		seen[relative] = [2]int64{direct, recursive}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(seen, expected) {
		t.Fatalf("directory counts = %#v, want %#v", seen, expected)
	}

	var discoveredAssets, processedAssets, skippedTotal, skippedDirectories, skippedFiles int64
	if err := inspector.QueryRow(`
		SELECT discovered_assets, processed_assets,
		       skipped_count, skipped_directories, skipped_files
		FROM scan_runs WHERE id = 1`,
	).Scan(
		&discoveredAssets,
		&processedAssets,
		&skippedTotal,
		&skippedDirectories,
		&skippedFiles,
	); err != nil {
		t.Fatal(err)
	}
	if discoveredAssets != 3 || processedAssets != 3 {
		t.Fatalf(
			"asset counters = discovered %d, processed %d; want 3 and 3",
			discoveredAssets,
			processedAssets,
		)
	}
	if skippedTotal != 1 || skippedDirectories != 0 || skippedFiles != 1 {
		t.Fatalf(
			"skipped counters = total %d, directories %d, files %d; want 1, 0, 1",
			skippedTotal,
			skippedDirectories,
			skippedFiles,
		)
	}
	var fingerprintedAssets int64
	if err := inspector.QueryRow(`
		SELECT COUNT(*)
		FROM assets
		WHERE source_fingerprint =
		      'v1:' || size_bytes || ':' || mtime_ns`,
	).Scan(&fingerprintedAssets); err != nil {
		t.Fatal(err)
	}
	if fingerprintedAssets != 3 {
		t.Fatalf("canonically fingerprinted assets = %d, want 3", fingerprintedAssets)
	}
}

func TestComposedApplicationStartsServesStopsAndRestarts(t *testing.T) {
	dataRoot := t.TempDir()
	mediaRoot := t.TempDir()
	var sessionCookie string

	for runNumber := 1; runNumber <= 2; runNumber++ {
		runComposedApplication(t, dataRoot, mediaRoot, runNumber, &sessionCookie)
	}

	for _, relativePath := range []string{
		databaseFilename,
		"cache",
		"tmp",
	} {
		if _, err := os.Stat(filepath.Join(dataRoot, relativePath)); err != nil {
			t.Fatalf("application data %q: %v", relativePath, err)
		}
	}
}

func TestComposedAuthenticationHandlesConcurrentSetupAndSessionRequests(t *testing.T) {
	application, err := composeConfiguration(
		Input{Version: "integration"},
		configuration{
			listenAddress: "127.0.0.1:0",
			mediaRoot:     t.TempDir(),
			dataRoot:      t.TempDir(),
		},
	)
	if err != nil {
		t.Fatalf("composeConfiguration() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		result <- application.run(ctx)
	}()
	defer func() {
		cancel()
		select {
		case runErr := <-result:
			if runErr != nil {
				t.Errorf("application.run() error = %v", runErr)
			}
		case <-time.After(runtimeIntegrationTimeout):
			t.Error("application did not stop after cancellation")
		}
	}()

	address := waitForListenAddress(t, application.http)
	client := &http.Client{Timeout: runtimeIntegrationTimeout}
	const concurrentRequests = 8
	type setupResult struct {
		status    int
		errorCode string
		body      string
		cookie    string
		err       error
	}
	results := make([]setupResult, concurrentRequests)
	start := make(chan struct{})
	var group sync.WaitGroup
	for index := range results {
		group.Add(1)
		go func() {
			defer group.Done()
			<-start
			request, requestErr := http.NewRequestWithContext(
				context.Background(),
				http.MethodPost,
				"http://"+address+"/api/v1/auth/setup",
				strings.NewReader(
					`{"username":"Administrator","displayName":"Administrator",`+
						`"password":"correct horse battery staple"}`,
				),
			)
			if requestErr != nil {
				results[index].err = requestErr
				return
			}
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Origin", "http://"+address)
			response, requestErr := client.Do(request)
			if requestErr != nil {
				results[index].err = requestErr
				return
			}
			body, readErr := io.ReadAll(response.Body)
			closeErr := response.Body.Close()
			if readErr != nil || closeErr != nil {
				results[index].err = errors.Join(readErr, closeErr)
				return
			}
			results[index].status = response.StatusCode
			results[index].body = string(body)
			var document struct {
				Error struct {
					Code string `json:"code"`
				} `json:"error"`
			}
			if err := json.Unmarshal(body, &document); err != nil {
				results[index].err = err
				return
			}
			results[index].errorCode = document.Error.Code
			for _, cookie := range response.Cookies() {
				if cookie.Name == "foliopath_session" {
					results[index].cookie = cookie.Value
				}
			}
		}()
	}
	close(start)
	group.Wait()

	successes := 0
	sessionCookie := ""
	for index, setup := range results {
		if setup.err != nil {
			t.Fatalf("concurrent setup %d error = %v", index, setup.err)
		}
		switch setup.status {
		case http.StatusCreated:
			successes++
			sessionCookie = setup.cookie
			if setup.cookie == "" {
				t.Fatalf("successful concurrent setup %d returned no cookie", index)
			}
		case http.StatusConflict:
			if setup.errorCode != "setup_in_progress" && setup.errorCode != "setup_closed" {
				t.Fatalf("concurrent setup %d error = %#v", index, setup)
			}
			for _, forbidden := range []string{
				"correct horse battery staple",
				"/app/data",
				"password_hash",
			} {
				if strings.Contains(setup.body, forbidden) {
					t.Fatalf("concurrent setup %d leaked %q: %s", index, forbidden, setup.body)
				}
			}
		default:
			t.Fatalf("concurrent setup %d = %#v", index, setup)
		}
	}
	if successes != 1 {
		t.Fatalf("successful concurrent setups = %d, want 1", successes)
	}

	repeated := runtimeAuthenticationRequest(
		t,
		client,
		address,
		http.MethodPost,
		"/api/v1/auth/setup",
		`{"username":"Other","displayName":"Other","password":"another valid password"}`,
		"",
		"",
	)
	if repeated.StatusCode != http.StatusConflict ||
		repeated.ErrorCode != "setup_closed" ||
		repeated.Cookie != "" {
		t.Fatalf("repeated setup response = %#v", repeated)
	}

	wrongPassword := runtimeAuthenticationRequest(
		t,
		client,
		address,
		http.MethodPost,
		"/api/v1/auth/login",
		`{"username":"Administrator","password":"wrong password"}`,
		"",
		"",
	)
	unknownAdministrator := runtimeAuthenticationRequest(
		t,
		client,
		address,
		http.MethodPost,
		"/api/v1/auth/login",
		`{"username":"Unknown","password":"wrong password"}`,
		"",
		"",
	)
	for name, response := range map[string]runtimeAuthenticationResponse{
		"wrong password":        wrongPassword,
		"unknown administrator": unknownAdministrator,
	} {
		if response.StatusCode != http.StatusUnauthorized ||
			response.ErrorCode != "invalid_credentials" ||
			response.Cookie != "" {
			t.Fatalf("%s response = %#v", name, response)
		}
		for _, forbidden := range []string{"wrong password", "Administrator", "Unknown"} {
			if strings.Contains(response.Body, forbidden) {
				t.Fatalf("%s response leaked %q: %s", name, forbidden, response.Body)
			}
		}
	}
	if wrongPassword.ErrorMessage != unknownAdministrator.ErrorMessage {
		t.Fatalf(
			"credential failure messages differ: %q != %q",
			wrongPassword.ErrorMessage,
			unknownAdministrator.ErrorMessage,
		)
	}

	type sessionResult struct {
		status int
		err    error
	}
	sessionResults := make([]sessionResult, 32)
	start = make(chan struct{})
	group = sync.WaitGroup{}
	for index := range sessionResults {
		group.Add(1)
		go func() {
			defer group.Done()
			<-start
			request, requestErr := http.NewRequestWithContext(
				context.Background(),
				http.MethodGet,
				"http://"+address+"/api/v1/auth/session",
				nil,
			)
			if requestErr != nil {
				sessionResults[index].err = requestErr
				return
			}
			request.AddCookie(&http.Cookie{
				Name:  "foliopath_session",
				Value: sessionCookie,
			})
			response, requestErr := client.Do(request)
			if requestErr != nil {
				sessionResults[index].err = requestErr
				return
			}
			_, readErr := io.Copy(io.Discard, response.Body)
			closeErr := response.Body.Close()
			sessionResults[index] = sessionResult{
				status: response.StatusCode,
				err:    errors.Join(readErr, closeErr),
			}
		}()
	}
	close(start)
	group.Wait()
	for index, session := range sessionResults {
		if session.err != nil || session.status != http.StatusOK {
			t.Fatalf("concurrent session %d = %#v", index, session)
		}
	}
}

func TestComposedLibraryCreationFailsClosedAcrossUnsafeRoots(t *testing.T) {
	mediaRoot := t.TempDir()
	dataRoot := t.TempDir()
	for _, directory := range []string{"family", "family/2026", "other"} {
		if err := os.MkdirAll(filepath.Join(mediaRoot, directory), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	ordinaryFile := filepath.Join(mediaRoot, "ordinary-file")
	if err := os.WriteFile(ordinaryFile, []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}
	externalRoot := t.TempDir()
	if err := os.Symlink(externalRoot, filepath.Join(mediaRoot, "external-link")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	application, err := composeConfiguration(
		Input{Version: "integration"},
		configuration{
			listenAddress: "127.0.0.1:0",
			mediaRoot:     mediaRoot,
			dataRoot:      dataRoot,
		},
	)
	if err != nil {
		t.Fatalf("composeConfiguration() error = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		result <- application.run(ctx)
	}()
	defer func() {
		cancel()
		select {
		case runErr := <-result:
			if runErr != nil {
				t.Errorf("application.run() error = %v", runErr)
			}
		case <-time.After(runtimeIntegrationTimeout):
			t.Error("application did not stop after cancellation")
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
	if setup.StatusCode != http.StatusCreated ||
		setup.Cookie == "" ||
		setup.CSRFToken == "" {
		t.Fatalf("setup response = %#v", setup)
	}

	tests := []struct {
		name       string
		root       string
		wantStatus int
		wantCode   string
	}{
		{name: "parent traversal", root: "../outside", wantStatus: http.StatusUnprocessableEntity, wantCode: "validation_failed"},
		{name: "dot component", root: "family/../other", wantStatus: http.StatusUnprocessableEntity, wantCode: "validation_failed"},
		{name: "encoded traversal", root: "%2e%2e", wantStatus: http.StatusUnprocessableEntity, wantCode: "validation_failed"},
		{name: "double encoded traversal", root: "%252e%252e", wantStatus: http.StatusUnprocessableEntity, wantCode: "validation_failed"},
		{name: "encoded separator", root: "family%2f2026", wantStatus: http.StatusUnprocessableEntity, wantCode: "validation_failed"},
		{name: "absolute", root: "/private", wantStatus: http.StatusUnprocessableEntity, wantCode: "validation_failed"},
		{name: "backslash", root: `family\2026`, wantStatus: http.StatusUnprocessableEntity, wantCode: "validation_failed"},
		{name: "NUL", root: "family\x00hidden", wantStatus: http.StatusUnprocessableEntity, wantCode: "validation_failed"},
		{name: "missing", root: "missing-private-name", wantStatus: http.StatusConflict, wantCode: "library_root_unavailable"},
		{name: "ordinary file", root: "ordinary-file", wantStatus: http.StatusConflict, wantCode: "library_root_unavailable"},
		{name: "symlink", root: "external-link", wantStatus: http.StatusConflict, wantCode: "library_root_symlink"},
	}
	for index, test := range tests {
		response := runtimeLibraryCreateRequest(
			t,
			client,
			address,
			"Unsafe "+test.name,
			test.root,
			"s2-005-unsafe-"+strconv.Itoa(index),
			setup.Cookie,
			setup.CSRFToken,
		)
		if response.StatusCode != test.wantStatus || response.ErrorCode != test.wantCode {
			t.Fatalf("%s response = %#v", test.name, response)
		}
		for _, leaked := range []string{mediaRoot, externalRoot, ordinaryFile, "permission denied"} {
			if strings.Contains(response.Body, leaked) {
				t.Fatalf("%s response leaked %q: %s", test.name, leaked, response.Body)
			}
		}
	}

	created := runtimeLibraryCreateRequest(
		t,
		client,
		address,
		"Family",
		"family",
		"s2-005-valid-family",
		setup.Cookie,
		setup.CSRFToken,
	)
	if created.StatusCode != http.StatusCreated {
		t.Fatalf("valid create response = %#v", created)
	}
	if !strings.Contains(created.Body, `"id":"lib_1"`) {
		t.Fatalf("unsafe-root attempts persisted state before valid create: %s", created.Body)
	}
	for index, root := range []string{"family", "family/2026", ""} {
		response := runtimeLibraryCreateRequest(
			t,
			client,
			address,
			"Overlap "+strconv.Itoa(index),
			root,
			"s2-005-overlap-"+strconv.Itoa(index),
			setup.Cookie,
			setup.CSRFToken,
		)
		if response.StatusCode != http.StatusConflict ||
			response.ErrorCode != "library_path_overlap" {
			t.Fatalf("overlap root %q response = %#v", root, response)
		}
	}
}

func TestComposedLibraryRemovalPreservesOriginalMediaByteForByte(t *testing.T) {
	mediaRoot := t.TempDir()
	dataRoot := t.TempDir()
	fixtures := map[string][]byte{
		"family/album/photo.jpg": {
			0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10, 0x4a, 0x46,
			0x49, 0x46, 0x00, 0x01, 0x00, 0xff, 0xd9,
		},
		"family/album/empty.png": {},
		"other/unrelated.mov":    {0x00, 0x00, 0x00, 0x14, 0x66, 0x74, 0x79, 0x70},
	}
	for relative, content := range fixtures {
		filename := filepath.Join(mediaRoot, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filename, content, 0o640); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(mediaRoot, "family", "empty"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(
		"album",
		filepath.Join(mediaRoot, "family", "latest"),
	); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	before := snapshotMediaTree(t, mediaRoot)

	application, err := composeConfiguration(
		Input{Version: "integration"},
		configuration{
			listenAddress: "127.0.0.1:0",
			mediaRoot:     mediaRoot,
			dataRoot:      dataRoot,
		},
	)
	if err != nil {
		t.Fatalf("composeConfiguration() error = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		result <- application.run(ctx)
	}()
	defer func() {
		cancel()
		select {
		case runErr := <-result:
			if runErr != nil {
				t.Errorf("application.run() error = %v", runErr)
			}
		case <-time.After(runtimeIntegrationTimeout):
			t.Error("application did not stop after cancellation")
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
	if setup.StatusCode != http.StatusCreated ||
		setup.Cookie == "" ||
		setup.CSRFToken == "" {
		t.Fatalf("setup response = %#v", setup)
	}
	created := runtimeLibraryCreateRequest(
		t,
		client,
		address,
		"Family",
		"family",
		"s2-006-create-family",
		setup.Cookie,
		setup.CSRFToken,
	)
	if created.StatusCode != http.StatusCreated ||
		created.ETag == "" ||
		created.Location != "/api/v1/libraries/lib_1" ||
		!strings.Contains(created.Body, `"id":"lib_1"`) {
		t.Fatalf("create response = %#v", created)
	}
	deadline := time.Now().Add(runtimeIntegrationTimeout)
	var currentLibrary runtimeAuthenticationResponse
	for time.Now().Before(deadline) {
		currentLibrary = runtimeAuthenticationRequest(
			t,
			client,
			address,
			http.MethodGet,
			created.Location,
			"",
			setup.Cookie,
			"",
		)
		if currentLibrary.StatusCode == http.StatusOK &&
			strings.Contains(currentLibrary.Body, `"status":"ready"`) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if currentLibrary.StatusCode != http.StatusOK ||
		currentLibrary.ETag == "" ||
		!strings.Contains(currentLibrary.Body, `"status":"ready"`) {
		t.Fatalf("creation scan did not complete before removal: %#v", currentLibrary)
	}

	cachePath := filepath.Join(
		dataRoot,
		"cache",
		"libraries",
		"lib_1",
		"thumbnails",
		"derived.webp",
	)
	if err := os.MkdirAll(filepath.Dir(cachePath), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cachePath, []byte("derived-cache"), 0o600); err != nil {
		t.Fatal(err)
	}

	removal := runtimeLibraryRemoveRequest(
		t,
		client,
		address,
		created.Location,
		currentLibrary.ETag,
		"s2-006-remove-family",
		setup.Cookie,
		setup.CSRFToken,
	)
	if removal.StatusCode != http.StatusAccepted ||
		removal.Location != "/api/v1/library-removals/rmv_1" {
		t.Fatalf("remove response = %#v", removal)
	}

	deadline = time.Now().Add(runtimeIntegrationTimeout)
	var terminal runtimeAuthenticationResponse
	for time.Now().Before(deadline) {
		terminal = runtimeAuthenticationRequest(
			t,
			client,
			address,
			http.MethodGet,
			removal.Location,
			"",
			setup.Cookie,
			"",
		)
		if terminal.StatusCode == http.StatusOK &&
			strings.Contains(terminal.Body, `"status":"succeeded"`) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if terminal.StatusCode != http.StatusOK ||
		!strings.Contains(terminal.Body, `"status":"succeeded"`) {
		t.Fatalf("terminal removal response = %#v", terminal)
	}
	if _, err := os.Stat(cachePath); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("derived cache still exists or failed unexpectedly: %v", err)
	}
	removedLibrary := runtimeAuthenticationRequest(
		t,
		client,
		address,
		http.MethodGet,
		created.Location,
		"",
		setup.Cookie,
		"",
	)
	if removedLibrary.StatusCode != http.StatusNotFound ||
		removedLibrary.ErrorCode != "library_not_found" {
		t.Fatalf("removed library response = %#v", removedLibrary)
	}
	assertMediaTreeUnchanged(t, before, snapshotMediaTree(t, mediaRoot))
}

type mediaTreeSnapshotEntry struct {
	mode    fs.FileMode
	link    string
	content []byte
}

func snapshotMediaTree(t *testing.T, root string) map[string]mediaTreeSnapshotEntry {
	t.Helper()
	snapshot := make(map[string]mediaTreeSnapshotEntry)
	err := filepath.WalkDir(root, func(filename string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(root, filename)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		info, err := os.Lstat(filename)
		if err != nil {
			return err
		}
		item := mediaTreeSnapshotEntry{mode: info.Mode()}
		switch {
		case info.Mode().IsRegular():
			item.content, err = os.ReadFile(filename)
		case info.Mode()&fs.ModeSymlink != 0:
			item.link, err = os.Readlink(filename)
		}
		if err != nil {
			return err
		}
		snapshot[relative] = item
		return nil
	})
	if err != nil {
		t.Fatalf("snapshot media tree: %v", err)
	}
	return snapshot
}

func assertMediaTreeUnchanged(
	t *testing.T,
	before, after map[string]mediaTreeSnapshotEntry,
) {
	t.Helper()
	if len(after) != len(before) {
		t.Fatalf("media tree entry count changed: before %d, after %d", len(before), len(after))
	}
	for relative, want := range before {
		got, exists := after[relative]
		if !exists {
			t.Fatalf("media entry %q was removed or renamed", relative)
		}
		if got.mode != want.mode || got.link != want.link ||
			!bytes.Equal(got.content, want.content) {
			t.Fatalf("media entry %q changed: before %#v, after %#v", relative, want, got)
		}
	}
}

func runComposedApplication(
	t *testing.T,
	dataRoot string,
	mediaRoot string,
	runNumber int,
	sessionCookie *string,
) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(mediaRoot, "albums", "2026"), 0o755); err != nil {
		t.Fatalf("prepare media root: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(mediaRoot, "ordinary-file.jpg"),
		[]byte("synthetic"),
		0o644,
	); err != nil {
		t.Fatalf("prepare ordinary media fixture: %v", err)
	}

	application, err := composeConfiguration(
		Input{Version: "integration"},
		configuration{
			listenAddress: "127.0.0.1:0",
			mediaRoot:     mediaRoot,
			dataRoot:      dataRoot,
		},
	)
	if err != nil {
		t.Fatalf("run %d composeConfiguration() error = %v", runNumber, err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		result <- application.run(ctx)
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case <-result:
		default:
		}
	})

	address := waitForListenAddress(t, application.http)
	client := &http.Client{Timeout: runtimeIntegrationTimeout}
	assertRuntimeResponse(t, client, address, "/health/ready", http.StatusOK, "ready")
	assertRuntimeResponse(t, client, address, "/health/live", http.StatusOK, "live")
	assertRuntimeResponse(
		t,
		client,
		address,
		"/api/v1/status",
		http.StatusUnauthorized,
		"authentication_required",
	)
	assertAdministratorSessionHTTP(t, application, client, address, runNumber, sessionCookie)
	assertLibraryPathHTTP(t, client, address, *sessionCookie)

	cancel()
	select {
	case runErr := <-result:
		if runErr != nil {
			t.Fatalf("run %d application.run() error = %v", runNumber, runErr)
		}
	case <-time.After(runtimeIntegrationTimeout):
		t.Fatalf("run %d application did not stop after cancellation", runNumber)
	}

	connection, err := net.DialTimeout("tcp", address, 100*time.Millisecond)
	if err == nil {
		connection.Close()
		t.Fatalf("run %d HTTP listener %q remained open after shutdown", runNumber, address)
	}
}

func assertLibraryPathHTTP(
	t *testing.T,
	client *http.Client,
	address string,
	sessionCookie string,
) {
	t.Helper()
	request, err := http.NewRequest(
		http.MethodGet,
		"http://"+address+"/api/v1/library-paths?limit=1",
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	request.AddCookie(&http.Cookie{
		Name:  "foliopath_session",
		Value: sessionCookie,
	})
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("library path request: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("library path status = %d; body = %s", response.StatusCode, body)
	}
	var page struct {
		Items []struct {
			Name         string `json:"name"`
			RelativePath string `json:"relativePath"`
			Selectable   bool   `json:"selectable"`
		} `json:"items"`
	}
	if err := json.NewDecoder(response.Body).Decode(&page); err != nil {
		t.Fatalf("decode library path page: %v", err)
	}
	if len(page.Items) != 1 ||
		page.Items[0].Name != "albums" ||
		page.Items[0].RelativePath != "albums" ||
		!page.Items[0].Selectable {
		t.Fatalf("library path page = %#v", page)
	}
}

func assertAdministratorSessionHTTP(
	t *testing.T,
	application *application,
	client *http.Client,
	address string,
	runNumber int,
	sessionCookie *string,
) {
	t.Helper()
	if application.authentication == nil {
		t.Fatal("composed application has no authentication service")
	}

	status := runtimeAuthenticationRequest(
		t,
		client,
		address,
		http.MethodGet,
		"/api/v1/auth/status",
		"",
		"",
		"",
	)
	if runNumber == 1 {
		if status.StatusCode != http.StatusOK || !status.SetupRequired {
			t.Fatalf("initial authentication status = %#v", status)
		}
		established := runtimeAuthenticationRequest(
			t,
			client,
			address,
			http.MethodPost,
			"/api/v1/auth/setup",
			`{"username":"Administrator","displayName":"Administrator","password":"correct horse battery staple"}`,
			"",
			"",
		)
		if established.StatusCode != http.StatusCreated ||
			established.Cookie == "" ||
			established.CSRFToken == "" {
			t.Fatalf("setup response = %#v", established)
		}
		*sessionCookie = established.Cookie
		status = runtimeAuthenticationRequest(
			t,
			client,
			address,
			http.MethodGet,
			"/api/v1/auth/status",
			"",
			"",
			"",
		)
	}
	if status.StatusCode != http.StatusOK || status.SetupRequired {
		t.Fatalf("run %d authentication status = %#v", runNumber, status)
	}

	current := runtimeAuthenticationRequest(
		t,
		client,
		address,
		http.MethodGet,
		"/api/v1/auth/session",
		"",
		*sessionCookie,
		"",
	)
	if current.StatusCode != http.StatusOK || current.CSRFToken == "" {
		t.Fatalf("run %d restore HTTP session = %#v", runNumber, current)
	}

	authorizedStatus := runtimeAuthenticationRequest(
		t,
		client,
		address,
		http.MethodGet,
		"/api/v1/status",
		"",
		*sessionCookie,
		"",
	)
	if authorizedStatus.StatusCode != http.StatusOK {
		t.Fatalf("run %d authorized status = %#v", runNumber, authorizedStatus)
	}

	if runNumber == 2 {
		loggedIn := runtimeAuthenticationRequest(
			t,
			client,
			address,
			http.MethodPost,
			"/api/v1/auth/login",
			`{"username":"administrator","password":"correct horse battery staple"}`,
			"",
			"",
		)
		if loggedIn.StatusCode != http.StatusOK ||
			loggedIn.Cookie == "" ||
			loggedIn.CSRFToken == "" {
			t.Fatalf("login administrator after restart = %#v", loggedIn)
		}
		loggedOut := runtimeAuthenticationRequest(
			t,
			client,
			address,
			http.MethodPost,
			"/api/v1/auth/logout",
			"",
			loggedIn.Cookie,
			loggedIn.CSRFToken,
		)
		if loggedOut.StatusCode != http.StatusNoContent || !loggedOut.CookieExpired {
			t.Fatalf("logout response = %#v", loggedOut)
		}
		revoked := runtimeAuthenticationRequest(
			t,
			client,
			address,
			http.MethodGet,
			"/api/v1/auth/session",
			"",
			loggedIn.Cookie,
			"",
		)
		if revoked.StatusCode != http.StatusUnauthorized ||
			revoked.ErrorCode != "session_expired" {
			t.Fatalf("revoked session response = %#v", revoked)
		}
	}
}

type runtimeAuthenticationResponse struct {
	StatusCode    int
	SetupRequired bool
	CSRFToken     string
	Cookie        string
	CookieExpired bool
	ErrorCode     string
	ErrorMessage  string
	ETag          string
	Location      string
	Body          string
}

func runtimeLibraryCreateRequest(
	t *testing.T,
	client *http.Client,
	address string,
	name string,
	root string,
	idempotencyKey string,
	cookie string,
	csrfToken string,
) runtimeAuthenticationResponse {
	t.Helper()
	body, err := json.Marshal(map[string]string{
		"name":     name,
		"rootPath": root,
	})
	if err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"http://"+address+"/api/v1/libraries",
		strings.NewReader(string(body)),
	)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", "http://"+address)
	request.Header.Set("X-CSRF-Token", csrfToken)
	request.Header.Set("Idempotency-Key", idempotencyKey)
	request.AddCookie(&http.Cookie{Name: "foliopath_session", Value: cookie})

	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	document := struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}{}
	if err := json.Unmarshal(responseBody, &document); err != nil {
		t.Fatalf("decode create response: %v; body = %s", err, responseBody)
	}
	return runtimeAuthenticationResponse{
		StatusCode:   response.StatusCode,
		ErrorCode:    document.Error.Code,
		ErrorMessage: document.Error.Message,
		ETag:         response.Header.Get("ETag"),
		Location:     response.Header.Get("Location"),
		Body:         string(responseBody),
	}
}

func runtimeLibraryRemoveRequest(
	t *testing.T,
	client *http.Client,
	address string,
	path string,
	etag string,
	idempotencyKey string,
	cookie string,
	csrfToken string,
) runtimeAuthenticationResponse {
	t.Helper()
	request, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodDelete,
		"http://"+address+path,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Origin", "http://"+address)
	request.Header.Set("X-CSRF-Token", csrfToken)
	request.Header.Set("If-Match", etag)
	request.Header.Set("Idempotency-Key", idempotencyKey)
	request.AddCookie(&http.Cookie{Name: "foliopath_session", Value: cookie})

	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	document := struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}{}
	if err := json.Unmarshal(responseBody, &document); err != nil {
		t.Fatalf("decode removal response: %v; body = %s", err, responseBody)
	}
	return runtimeAuthenticationResponse{
		StatusCode:   response.StatusCode,
		ErrorCode:    document.Error.Code,
		ErrorMessage: document.Error.Message,
		ETag:         response.Header.Get("ETag"),
		Location:     response.Header.Get("Location"),
		Body:         string(responseBody),
	}
}

func runtimeAuthenticationRequest(
	t *testing.T,
	client *http.Client,
	address string,
	method string,
	path string,
	body string,
	cookie string,
	csrfToken string,
) runtimeAuthenticationResponse {
	t.Helper()
	request, err := http.NewRequestWithContext(
		context.Background(),
		method,
		"http://"+address+path,
		strings.NewReader(body),
	)
	if err != nil {
		t.Fatalf("create %s %s request: %v", method, path, err)
	}
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Origin", "http://"+address)
	}
	if cookie != "" {
		request.AddCookie(&http.Cookie{Name: "foliopath_session", Value: cookie})
	}
	if csrfToken != "" {
		request.Header.Set("X-CSRF-Token", csrfToken)
	}

	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer response.Body.Close()
	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read %s %s response: %v", method, path, err)
	}
	document := struct {
		SetupRequired bool   `json:"setupRequired"`
		CSRFToken     string `json:"csrfToken"`
		Error         struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}{}
	if response.StatusCode != http.StatusNoContent {
		if err := json.Unmarshal(bodyBytes, &document); err != nil {
			t.Fatalf("decode %s %s response: %v", method, path, err)
		}
	}
	result := runtimeAuthenticationResponse{
		StatusCode:    response.StatusCode,
		SetupRequired: document.SetupRequired,
		CSRFToken:     document.CSRFToken,
		ErrorCode:     document.Error.Code,
		ErrorMessage:  document.Error.Message,
		ETag:          response.Header.Get("ETag"),
		Location:      response.Header.Get("Location"),
		Body:          string(bodyBytes),
	}
	for _, responseCookie := range response.Cookies() {
		if responseCookie.Name != "foliopath_session" {
			continue
		}
		result.Cookie = responseCookie.Value
		result.CookieExpired = responseCookie.MaxAge < 0
	}
	return result
}

func waitForListenAddress(t *testing.T, service *httpService) string {
	t.Helper()
	if service == nil {
		t.Fatal("composed application has no HTTP service")
	}

	deadline := time.Now().Add(runtimeIntegrationTimeout)
	for time.Now().Before(deadline) {
		if address := service.listenAddress(); address != "" {
			return address
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("HTTP service did not start")
	return ""
}

func assertRuntimeResponse(
	t *testing.T,
	client *http.Client,
	address string,
	path string,
	wantStatus int,
	wantValue string,
) {
	t.Helper()

	deadline := time.Now().Add(runtimeIntegrationTimeout)
	var lastErr error
	for time.Now().Before(deadline) {
		request, err := http.NewRequestWithContext(
			context.Background(),
			http.MethodGet,
			"http://"+address+path,
			nil,
		)
		if err != nil {
			t.Fatalf("create request: %v", err)
		}
		response, err := client.Do(request)
		if err != nil {
			lastErr = err
			time.Sleep(10 * time.Millisecond)
			continue
		}
		body, readErr := io.ReadAll(response.Body)
		closeErr := response.Body.Close()
		if readErr != nil || closeErr != nil {
			lastErr = errors.Join(readErr, closeErr)
			time.Sleep(10 * time.Millisecond)
			continue
		}
		if response.StatusCode != wantStatus {
			t.Fatalf(
				"GET %s status = %d, want %d; body = %s",
				path,
				response.StatusCode,
				wantStatus,
				body,
			)
		}

		var document struct {
			Status string `json:"status"`
			Error  struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		if err := json.Unmarshal(body, &document); err != nil {
			t.Fatalf("GET %s response is not JSON: %v", path, err)
		}
		if document.Status != wantValue && document.Error.Code != wantValue {
			t.Fatalf("GET %s response = %s, want value %q", path, body, wantValue)
		}
		if response.Header.Get("X-Request-ID") == "" {
			t.Fatalf("GET %s response has no X-Request-ID", path)
		}
		return
	}
	t.Fatalf("GET %s did not succeed: %v", path, lastErr)
}
