//go:build libvips && ocrprobe

package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode"

	"github.com/HappyQuQu/foliopath/internal/curation"
	"github.com/HappyQuQu/foliopath/internal/files"
	"github.com/HappyQuQu/foliopath/internal/inference/remoteml"
	"github.com/HappyQuQu/foliopath/internal/jobs"
	"github.com/HappyQuQu/foliopath/internal/library"
	"github.com/HappyQuQu/foliopath/internal/ocr"
	"github.com/HappyQuQu/foliopath/internal/scanner"
	sqlitestore "github.com/HappyQuQu/foliopath/internal/store/sqlite"
)

type ocrProbeState struct {
	LibraryID       int64
	IDs             map[string]int64
	JobID           string
	ParkedCompleted int64
	Protected       curation.AssetState
	Outcomes        map[string]any
	RepeatRecorded  int64
}

func TestOCRWorkerRealApplication(t *testing.T) {
	if os.Getenv("FOLIOPATH_OCR_TEST_ORIGIN") == "" {
		t.Fatal("explicit isolated origin required")
	}
	phase := os.Getenv("FOLIOPATH_OCR_WORKER_PHASE")
	if phase == "" {
		root := t.TempDir()
		binary, err := os.Executable()
		if err != nil {
			t.Fatal(err)
		}
		// libvips shutdown is process-final. Each application restart runs in a new
		// OS process, sharing only this test-owned Linux tmpfs data directory.
		for _, phase := range []string{"first", "resume", "clear"} {
			command := exec.Command(binary, "-test.run=^TestOCRWorkerRealApplication$", "-test.v", "-test.timeout=4m")
			command.Env = append(os.Environ(), "FOLIOPATH_OCR_WORKER_PHASE="+phase, "FOLIOPATH_OCR_WORKER_ROOT="+root)
			raw, err := command.CombinedOutput()
			t.Logf("phase %s:\n%s", phase, raw)
			if err != nil {
				t.Fatal(err)
			}
		}
		raw, err := os.ReadFile(filepath.Join(root, "state.json"))
		if err != nil {
			t.Fatal(err)
		}
		var state ocrProbeState
		if json.Unmarshal(raw, &state) != nil {
			t.Fatal("state")
		}
		result := map[string]any{"status": "passed", "originalJob": state.JobID, "parkedCompleted": state.ParkedCompleted, "resumedSameJob": true, "outcomes": state.Outcomes, "missingRepeatRecorded": state.RepeatRecorded, "clearDisabled": true, "nonemptyManualPreserved": true, "integrity": true, "separateOSProcesses": 3, "persistedTextMatchesFixture": true, "ocrSearchHTTP": true, "currentSourceFulltext": true, "clearHidesSearch": true}
		raw, _ = json.MarshalIndent(result, "", "  ")
		if err = os.WriteFile("/observations/worker.json", raw, 0600); err != nil {
			t.Fatal(err)
		}
		return
	}
	rootPath := os.Getenv("FOLIOPATH_OCR_WORKER_ROOT")
	if !strings.HasPrefix(rootPath, "/tmp/") {
		t.Fatal("isolated root required")
	}
	dataRoot := filepath.Join(rootPath, "data")
	modelsRoot := filepath.Join(rootPath, "models")
	if err := os.MkdirAll(dataRoot, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(modelsRoot, 0700); err != nil {
		t.Fatal(err)
	}
	statePath := filepath.Join(rootPath, "state.json")
	ctx := context.Background()
	var state ocrProbeState
	openStore := func() *sqlitestore.Store {
		s, err := sqlitestore.Open(ctx, filepath.Join(dataRoot, databaseFilename), sqlitestore.Options{})
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	if phase == "first" {
		s := openStore()
		settings, err := s.GetSettings(ctx)
		if err != nil {
			t.Fatal(err)
		}
		settings.AutomaticDiscoveryEnabled = false
		settings.ScheduledScanIntervalHours = nil
		if _, err = s.UpdateSettings(ctx, settings.Revision, settings); err != nil {
			t.Fatal(err)
		}
		libService, _ := library.NewService(s)
		lib, err := libService.Create(ctx, "OCR synthetic", "")
		if err != nil {
			t.Fatal(err)
		}
		state.LibraryID = lib.ID
		root, err := files.OpenRoot("/library")
		if err != nil {
			t.Fatal(err)
		}
		walker, err := files.NewScanWalker(root)
		if err != nil {
			t.Fatal(err)
		}
		scans, _ := scanner.NewService(s, scanner.Config{})
		if _, err = scans.RunFullScan(ctx, scanner.FullScanRequest{LibraryID: lib.ID, Trigger: scanner.TriggerCreation, Walker: walker}); err != nil {
			t.Fatal(err)
		}
		_ = root.Close()
		state.IDs = map[string]int64{}
		for id := int64(1); id <= 5; id++ {
			asset, e := s.GetContentAsset(ctx, id)
			if e != nil {
				t.Fatal(e)
			}
			state.IDs[asset.RelativePath] = id
		}
		if state.IDs["bilingual.png"] == 0 || state.IDs["empty.png"] == 0 {
			t.Fatal(state.IDs)
		}
		curator, _ := curation.NewService(s, []byte("0123456789abcdef0123456789abcdef"), time.Now)
		tag, _, err := curator.CreateTag(ctx, "人工中文保留")
		if err != nil {
			t.Fatal(err)
		}
		old, err := curator.GetAssetState(ctx, state.IDs["bilingual.png"])
		if err != nil {
			t.Fatal(err)
		}
		state.Protected, err = curator.ReplaceAssetTags(ctx, state.IDs["bilingual.png"], old.Revision, []int64{tag.ID})
		if err != nil {
			t.Fatal(err)
		}
		if err = s.Close(); err != nil {
			t.Fatal(err)
		}
	} else {
		raw, err := os.ReadFile(statePath)
		if err != nil || json.Unmarshal(raw, &state) != nil {
			t.Fatal(err)
		}
	}
	var expected ocr.Identity
	raw, err := os.ReadFile("/contracts/ocr-identity.json")
	if err != nil || json.Unmarshal(raw, &expected) != nil {
		t.Fatal(err)
	}
	var legacy struct {
		Semantic remoteml.ServiceIdentity `json:"semantic"`
	}
	raw, err = os.ReadFile("/contracts/baseline-identities.json")
	if err != nil || json.Unmarshal(raw, &legacy) != nil {
		t.Fatal(err)
	}
	config := deploymentConfigFixture()
	config.Origin = os.Getenv("FOLIOPATH_OCR_TEST_ORIGIN")
	config.Identity = legacy.Semantic
	config.OCR = &ocrDeploymentConfiguration{Identity: expected}
	a, err := composeConfiguration(Input{Version: "ocr-probe"}, configuration{deployment: &config, listenAddress: "127.0.0.1:0", mediaRoot: "/library", dataRoot: dataRoot, modelRoot: modelsRoot})
	if err != nil {
		t.Fatal(err)
	}
	runctx, cancel := context.WithCancelCause(ctx)
	defer cancel(jobs.ErrPlannedShutdown)
	done := make(chan error, 1)
	go func() { done <- a.run(runctx) }()
	address := waitForListenAddress(t, a.http)
	client := &http.Client{Timeout: 10 * time.Second, Transport: &http.Transport{Proxy: nil}}
	defer client.CloseIdleConnections()
	assertRuntimeResponse(t, client, address, "/health/ready", 200, "ready")
	authPath := "/api/v1/auth/login"
	authBody := `{"username":"OCRprobe","password":"synthetic fixture passphrase only"}`
	want := 200
	if phase == "first" {
		authPath = "/api/v1/auth/setup"
		authBody = `{"username":"OCRprobe","displayName":"Synthetic","password":"synthetic fixture passphrase only"}`
		want = 201
	}
	session := runtimeAuthenticationRequest(t, client, address, "POST", authPath, authBody, "", "")
	if session.StatusCode != want {
		t.Fatal(session.StatusCode, session.Body)
	}
	path := fmt.Sprintf("/api/v1/libraries/lib_%d/ai/ocr", state.LibraryID)
	type reply struct {
		status int
		body   []byte
		etag   string
	}
	request := func(method, suffix, body, etag, key string) reply {
		r, _ := http.NewRequest(method, "http://"+address+path+suffix, strings.NewReader(body))
		r.Header.Set("Origin", "http://"+address)
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("If-Match", etag)
		r.Header.Set("Idempotency-Key", key)
		r.Header.Set("X-CSRF-Token", session.CSRFToken)
		r.AddCookie(&http.Cookie{Name: "foliopath_session", Value: session.Cookie})
		response, e := client.Do(r)
		if e != nil {
			t.Fatal(e)
		}
		defer response.Body.Close()
		raw, e := io.ReadAll(response.Body)
		if e != nil {
			t.Fatal(e)
		}
		return reply{response.StatusCode, raw, response.Header.Get("ETag")}
	}
	searchOCR := func(query, mode string, want int) {
		r, _ := http.NewRequest("GET", "http://"+address+"/api/v1/search/assets?q="+url.QueryEscape(query)+"&mode="+mode+fmt.Sprintf("&libraryId=lib_%d", state.LibraryID), nil)
		r.AddCookie(&http.Cookie{Name: "foliopath_session", Value: session.Cookie})
		response, e := client.Do(r)
		if e != nil {
			t.Fatal(e)
		}
		defer response.Body.Close()
		raw, e := io.ReadAll(response.Body)
		if e != nil || response.StatusCode != 200 {
			t.Fatal(response.StatusCode, string(raw), e)
		}
		var page struct {
			Items []struct {
				Sources []string
				OCR     []struct {
					Line int
					Text string
				}
			}
			OCRState    string
			OCRCoverage struct{ Eligible, Ready, Failed int }
		}
		if json.Unmarshal(raw, &page) != nil || len(page.Items) != want {
			t.Fatal("OCR search", mode, query, string(raw))
		}
		for _, hit := range page.Items {
			if len(hit.OCR) == 0 || len(hit.OCR) > 8 || hit.OCR[0].Line < 1 {
				t.Fatal("missing real excerpt", string(raw))
			}
			found := false
			for _, source := range hit.Sources {
				found = found || source == "ocr"
			}
			if !found {
				t.Fatal("missing OCR source")
			}
		}
	}
	status := func() reply {
		r := request("GET", "", "", "", "")
		if r.status != 200 {
			t.Fatal(r.status, string(r.body))
		}
		return r
	}
	admit := func(etag, key string) string {
		deadline := time.Now().Add(20 * time.Second)
		for time.Now().Before(deadline) {
			r := request("POST", "/jobs", `{"mode":"missing"}`, etag, key)
			if r.status == 202 {
				var j struct {
					ID string `json:"id"`
				}
				if json.Unmarshal(r.body, &j) != nil || j.ID == "" {
					t.Fatal(string(r.body))
				}
				return j.ID
			}
			if r.status != 409 {
				t.Fatal(r.status, string(r.body))
			}
			time.Sleep(time.Second)
		}
		t.Fatal("admission timeout")
		return ""
	}
	type jobStatus struct {
		State     string  `json:"state"`
		Recorded  int64   `json:"recorded"`
		ErrorCode *string `json:"errorCode"`
	}
	job := func(id string) jobStatus {
		r := request("GET", "/jobs/"+id, "", "", "")
		if r.status != 200 {
			t.Fatal(r.status, string(r.body))
		}
		var j jobStatus
		if json.Unmarshal(r.body, &j) != nil || j.State == "failed" {
			t.Fatal(string(r.body))
		}
		return j
	}
	if phase == "first" {
		before := status()
		if !strings.Contains(string(before.body), `"enabled":false`) || !strings.Contains(string(before.body), `"latestJob":null`) {
			t.Fatal(string(before.body))
		}
		enabled := request("PUT", "", `{"enabled":true}`, before.etag, "")
		if enabled.status != 200 {
			t.Fatal(enabled.status, string(enabled.body))
		}
		state.JobID = admit(enabled.etag, "real-ocr-first")
		deadline := time.Now().Add(90 * time.Second)
		var progress jobStatus
		for time.Now().Before(deadline) {
			progress = job(state.JobID)
			if progress.Recorded > 0 && progress.Recorded < 5 && progress.State == "running" {
				break
			}
			time.Sleep(300 * time.Millisecond)
		}
		if progress.Recorded < 1 || progress.Recorded >= 5 {
			t.Fatal("missed in-flight handoff", progress)
		}
	} else if phase == "resume" {
		deadline := time.Now().Add(2 * time.Minute)
		var progress jobStatus
		for time.Now().Before(deadline) {
			progress = job(state.JobID)
			if progress.State == "succeeded" {
				break
			}
			time.Sleep(time.Second)
		}
		if progress.State != "succeeded" || progress.Recorded != 5 {
			t.Fatal(progress)
		}
		searchOCR("中文", "ocr", 2)
		searchOCR("订单编号123456", "all", 2)
		searchOCR("LOCAL TEST", "ocr", 2)
		searchOCR("not-in-fixture-xyz", "ocr", 0)
	} else if phase == "clear" {
		current := status()
		id := admit(current.etag, "real-ocr-repeat")
		deadline := time.Now().Add(20 * time.Second)
		var progress jobStatus
		for time.Now().Before(deadline) {
			progress = job(id)
			if progress.State == "succeeded" {
				break
			}
			time.Sleep(200 * time.Millisecond)
		}
		if progress.State != "succeeded" || progress.Recorded != 0 {
			t.Fatal(progress)
		}
		state.RepeatRecorded = progress.Recorded
		cleared := request("POST", "/clear", `{"confirmation":"clear_ocr_results"}`, current.etag, "")
		if cleared.status != 202 {
			t.Fatal(cleared.status, string(cleared.body))
		}
		deadline = time.Now().Add(15 * time.Second)
		clearedOK := false
		for time.Now().Before(deadline) {
			r := status()
			if strings.Contains(string(r.body), `"clearing":false`) && strings.Contains(string(r.body), `"pending":5`) && strings.Contains(string(r.body), `"enabled":false`) {
				clearedOK = true
				break
			}
			time.Sleep(200 * time.Millisecond)
		}
		searchOCR("中文", "ocr", 0)
		if !clearedOK {
			t.Fatal("clear timeout")
		}
	} else {
		t.Fatal("unknown phase")
	}
	cancel(jobs.ErrPlannedShutdown)
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("shutdown timeout")
	}
	// The sole app writer is stopped. This detached verification remains in the
	// same Linux lock domain, never a host SQLite open or live database copy.
	s := openStore()
	defer s.Close()
	if phase == "first" {
		parked, err := s.GetOCRJob(ctx, state.JobID)
		if err != nil || parked.State != "queued" || parked.Completed < 1 || parked.Completed >= 5 {
			t.Fatal(parked, err)
		}
		state.ParkedCompleted = parked.Completed
	}
	if phase == "resume" {
		state.Outcomes = map[string]any{}
		for name, id := range state.IDs {
			v, found, e := s.GetCurrentOCRResult(ctx, state.LibraryID, id)
			if e != nil || !found {
				t.Fatal(name, found, e)
			}
			if name == "corrupt.png" || name == "empty.png" {
				if v.Outcome != "failed" || v.ErrorCode != "invalid_media" {
					t.Fatal(name, v)
				}
			} else if v.Outcome != "ready" {
				t.Fatal(name, v)
			}
			if (name == "bilingual.png" || name == "exif-oriented.jpg") && len(v.Result.Lines) != 4 {
				t.Fatal(name, len(v.Result.Lines))
			}
			if name == "bilingual.png" || name == "exif-oriented.jpg" {
				key := func(value string) string {
					return strings.Map(func(r rune) rune {
						if unicode.IsLetter(r) || unicode.IsDigit(r) {
							return unicode.ToLower(r)
						}
						return -1
					}, value)
				}
				for index, want := range []string{"FolioPath OCR 2026", "中文文字识别测试", "订单编号123456", "LOCAL TEST ONLY"} {
					if key(v.Result.Lines[index].Text) != key(want) {
						t.Fatal("persisted OCR text mismatch", name, index)
					}
				}
			}
			if name == "blank.png" && len(v.Result.Lines) != 0 {
				t.Fatal(v)
			}
			state.Outcomes[name] = map[string]any{"outcome": v.Outcome, "error": v.ErrorCode, "lines": len(v.Result.Lines), "fingerprint": v.Fingerprint}
		}
	}
	curator, _ := curation.NewService(s, []byte("0123456789abcdef0123456789abcdef"), time.Now)
	after, e := curator.GetAssetState(ctx, state.IDs["bilingual.png"])
	if e != nil || len(after.Tags) != 1 || after.Tags[0].ID != state.Protected.Tags[0].ID || after.Revision != state.Protected.Revision {
		t.Fatal(after, e)
	}
	if err = sqlitestore.CheckIntegrity(ctx, filepath.Join(dataRoot, databaseFilename)); err != nil {
		t.Fatal(err)
	}
	raw, _ = json.Marshal(state)
	if err = os.WriteFile(statePath, raw, 0600); err != nil {
		t.Fatal(err)
	}
}
