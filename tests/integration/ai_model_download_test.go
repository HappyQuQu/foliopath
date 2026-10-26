package integration_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
	"github.com/HappyQuQu/foliopath/internal/files"
	"github.com/HappyQuQu/foliopath/internal/store/sqlite"
)

// Synthetic bytes test the real HTTP -> queue -> streaming installer -> managed
// files -> SQLite workflow. They do not represent or activate an ML model.
func TestAIModelDownloadPersistsProgressAndRejectsCorruption(t *testing.T) {
	for _, mode := range []string{"success", "corrupt", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			ctx := context.Background()
			payload := bytes.Repeat([]byte("synthetic-model"), 16<<10)
			digest := sha256.Sum256(payload)
			manifest := aimodel.Manifest{FormatVersion: 1, PackageID: "synthetic-download", Purpose: aimodel.PurposeSemanticImageText, Version: "test-1", Architecture: "portable-onnx", LicenseID: "MIT"}
			locations := map[string]string{}
			for _, role := range []string{"image_encoder", "text_encoder", "tokenizer"} {
				name := role + ".fixture"
				manifest.Files = append(manifest.Files, aimodel.ManifestFile{Name: name, Role: role, Size: int64(len(payload)), SHA256: hex.EncodeToString(digest[:])})
				locations[name] = "https://models.example/" + name
			}
			definition := aimodel.DownloadDefinition{ID: "download_synthetic", Name: "Synthetic fixture", SourceLabel: "Test server", Manifest: manifest, ContentHash: strings.Repeat("d", 64), RuntimeArchitectures: []string{"amd64", "arm64"}, Files: locations}
			encoded, _ := json.Marshal([]aimodel.DownloadDefinition{definition})
			catalog, err := aimodel.ParseDownloadCatalog(encoded, "amd64")
			if err != nil {
				t.Fatal(err)
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if mode == "corrupt" {
					_, _ = w.Write(bytes.Repeat([]byte("!"), len(payload)))
					return
				}
				if mode == "cancel" {
					_, _ = w.Write(payload[:64<<10])
					w.(http.Flusher).Flush()
					<-r.Context().Done()
					return
				}
				_, _ = w.Write(payload)
			}))
			defer server.Close()
			dbPath := filepath.Join(t.TempDir(), "catalog.db")
			store, err := sqlite.Open(ctx, dbPath, sqlite.Options{})
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = store.Close() }()
			models, _ := aimodel.NewService(store, nil, nil)
			operations, _ := aimodel.NewOperationService(store, nil, nil)
			managed, err := files.NewManagedModelStore(filepath.Join(t.TempDir(), "models"), 0)
			if err != nil {
				t.Fatal(err)
			}
			signal := downloadWake{make(chan struct{}, 1)}
			admission, _ := aimodel.NewInstallAdmissionService(store, signal, nil, nil)
			service, _ := aimodel.NewDownloadService(catalog, models, admission, store, false)
			source := &aimodel.DownloadSource{Catalog: catalog, Remote: downloadFixtureHTTP{server.URL}}
			installer, _ := aimodel.NewInstaller(models, source, managed)
			worker, _ := aimodel.NewInstallWorker(store, installer, operations, signal.channel, 10*time.Millisecond)
			runCtx, stop := context.WithCancel(ctx)
			defer stop()
			done := make(chan error, 1)
			go func() { done <- worker.Run(runCtx) }()
			result, err := service.StartDownload(ctx, definition.ID, "download-test-request")
			if err != nil {
				t.Fatal(err)
			}
			replay, err := service.StartDownload(ctx, definition.ID, "download-test-request")
			if err != nil || replay.Operation.ID != result.Operation.ID || !replay.Replayed {
				t.Fatalf("replay = %+v %v", replay, err)
			}
			if mode == "cancel" {
				if _, err := service.StartDownload(ctx, definition.ID, "another-download-key"); !errors.Is(err, aimodel.ErrDownloadActive) {
					t.Fatalf("duplicate active download: %v", err)
				}
			}
			var terminal aimodel.Operation
			deadline := time.Now().Add(10 * time.Second)
			cancelRequested := false
			for time.Now().Before(deadline) {
				operation, err := operations.Get(ctx, result.Operation.ID)
				if err != nil {
					t.Fatal(err)
				}
				if mode == "cancel" && operation.State == aimodel.OperationRunning && operation.TotalItems != nil && !cancelRequested {
					if _, err := operations.RequestCancel(ctx, operation.ID, operation.Revision); err == nil {
						cancelRequested = true
					}
				}
				if operation.State == aimodel.OperationSucceeded || operation.State == aimodel.OperationFailed || operation.State == aimodel.OperationCancelled {
					terminal = operation
					break
				}
				time.Sleep(10 * time.Millisecond)
			}
			stop()
			if err := <-done; err != nil {
				t.Fatal(err)
			}
			expected := map[string]aimodel.OperationState{"success": aimodel.OperationSucceeded, "corrupt": aimodel.OperationFailed, "cancel": aimodel.OperationCancelled}[mode]
			if terminal.State != expected {
				t.Fatalf("terminal = %+v, want %s", terminal, expected)
			}
			page, err := service.ListDownloads(ctx)
			if len(page.Presets) != 3 {
				t.Fatalf("built-in presets missing alongside configured downloads: %+v", page)
			}
			if err != nil || page.Items[0].Operation == nil || page.Items[0].Operation.State != expected {
				t.Fatalf("history = %+v %v", page, err)
			}
			if mode == "success" {
				if terminal.TotalItems == nil || terminal.CompletedItems != int64(len(payload)*3) || *terminal.TotalItems != terminal.CompletedItems || page.Items[0].InstalledModelID == "" {
					t.Fatalf("byte progress/install = %+v %+v", terminal, page.Items[0])
				}
			} else if page.Items[0].InstalledModelID != "" {
				t.Fatal("failed/cancelled download registered a model")
			}
			if err := store.Close(); err != nil {
				t.Fatal(err)
			}
			store, err = sqlite.Open(ctx, dbPath, sqlite.Options{})
			if err != nil {
				t.Fatal(err)
			}
			persisted, found, err := store.LatestAIModelInstall(ctx, definition.ID)
			if err != nil || !found || persisted.State != expected || persisted.CompletedItems != terminal.CompletedItems {
				t.Fatalf("reopened history = %+v %v", persisted, err)
			}
		})
	}
}

type downloadWake struct{ channel chan struct{} }

func (signal downloadWake) Wake() {
	select {
	case signal.channel <- struct{}{}:
	default:
	}
}

type downloadFixtureHTTP struct{ origin string }

func (source downloadFixtureHTTP) OpenDownload(ctx context.Context, location string, _ []string, _ int64) (io.ReadCloser, error) {
	parsed, _ := url.Parse(location)
	request, _ := http.NewRequestWithContext(ctx, http.MethodGet, source.origin+parsed.Path, nil)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	return response.Body, nil
}
