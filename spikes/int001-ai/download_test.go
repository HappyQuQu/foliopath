package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestFetchTrustedArtifactFailureMatrix(t *testing.T) {
	payload := []byte(strings.Repeat("verified-model-artifact-", 64))
	digestBytes := sha256.Sum256(payload)
	digest := hex.EncodeToString(digestBytes[:])
	const etag = `"catalog-generation-7"`

	serveArtifact := func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("ETag", etag)
		if request.Header.Get("Range") == "bytes=137-" && request.Header.Get("If-Range") == etag {
			writer.Header().Set("Content-Range", contentRange(137, int64(len(payload)-1), int64(len(payload))))
			writer.WriteHeader(http.StatusPartialContent)
			_, _ = writer.Write(payload[137:])
			return
		}
		_, _ = writer.Write(payload)
	}
	server := httptest.NewServer(http.HandlerFunc(serveArtifact))
	defer server.Close()
	artifact := trustedArtifact{
		URL: server.URL + "/model.onnx", Origin: server.URL, ETag: etag,
		Size: int64(len(payload)), SHA256: digest, allowLoopbackForTest: true,
	}

	t.Run("loopback source needs explicit test-only exception", func(t *testing.T) {
		blocked := artifact
		blocked.allowLoopbackForTest = false
		err := fetchTrustedArtifact(context.Background(), testCatalogHTTPClient(server.Client()), blocked, t.TempDir(), "model.partial", "model.verified", int64(len(payload)))
		if !errors.Is(err, errDownloadOrigin) {
			t.Fatalf("expected loopback origin rejection, got %v", err)
		}
	})

	t.Run("verified initial download publishes", func(t *testing.T) {
		parent := t.TempDir()
		if err := fetchTrustedArtifact(context.Background(), testCatalogHTTPClient(server.Client()), artifact, parent, "model.partial", "model.verified", int64(len(payload))); err != nil {
			t.Fatal(err)
		}
		actual, err := os.ReadFile(filepath.Join(parent, "model.verified"))
		if err != nil || string(actual) != string(payload) {
			t.Fatalf("published bytes mismatch: %v", err)
		}
	})

	t.Run("stable validator resumes partial download", func(t *testing.T) {
		parent := t.TempDir()
		if err := os.WriteFile(filepath.Join(parent, "model.partial"), payload[:137], 0o600); err != nil {
			t.Fatal(err)
		}
		if err := fetchTrustedArtifact(context.Background(), testCatalogHTTPClient(server.Client()), artifact, parent, "model.partial", "model.verified", int64(len(payload))); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("changed validator rejects resume and removes stale partial", func(t *testing.T) {
		changed := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			writer.Header().Set("ETag", `"changed"`)
			_, _ = writer.Write(payload)
		}))
		defer changed.Close()
		parent := t.TempDir()
		partial := filepath.Join(parent, "model.partial")
		if err := os.WriteFile(partial, payload[:137], 0o600); err != nil {
			t.Fatal(err)
		}
		changedArtifact := artifact
		changedArtifact.URL = changed.URL + "/model.onnx"
		changedArtifact.Origin = changed.URL
		err := fetchTrustedArtifact(context.Background(), testCatalogHTTPClient(changed.Client()), changedArtifact, parent, "model.partial", "model.verified", int64(len(payload)))
		if !errors.Is(err, errDownloadResume) {
			t.Fatalf("expected resume rejection, got %v", err)
		}
		if _, err := os.Stat(partial); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("stale partial remains after changed object: %v", err)
		}
	})

	t.Run("redirect outside reviewed origin is rejected", func(t *testing.T) {
		other := httptest.NewServer(http.HandlerFunc(serveArtifact))
		defer other.Close()
		redirect := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			http.Redirect(writer, request, other.URL+"/private", http.StatusFound)
		}))
		defer redirect.Close()
		redirected := artifact
		redirected.URL = redirect.URL + "/model.onnx"
		redirected.Origin = redirect.URL
		err := fetchTrustedArtifact(context.Background(), testCatalogHTTPClient(redirect.Client()), redirected, t.TempDir(), "model.partial", "model.verified", int64(len(payload)))
		if !errors.Is(err, errDownloadOrigin) {
			t.Fatalf("expected origin rejection, got %v", err)
		}
	})

	t.Run("redirect inside reviewed origin is accepted", func(t *testing.T) {
		var sameOrigin *httptest.Server
		sameOrigin = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			if request.URL.Path == "/start" {
				http.Redirect(writer, request, sameOrigin.URL+"/model.onnx", http.StatusFound)
				return
			}
			serveArtifact(writer, request)
		}))
		defer sameOrigin.Close()
		redirected := artifact
		redirected.URL = sameOrigin.URL + "/start"
		redirected.Origin = sameOrigin.URL
		if err := fetchTrustedArtifact(context.Background(), testCatalogHTTPClient(sameOrigin.Client()), redirected, t.TempDir(), "model.partial", "model.verified", int64(len(payload))); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("quota rejects before network", func(t *testing.T) {
		err := fetchTrustedArtifact(context.Background(), testCatalogHTTPClient(server.Client()), artifact, t.TempDir(), "model.partial", "model.verified", int64(len(payload)-1))
		if !errors.Is(err, errDownloadQuota) {
			t.Fatalf("expected quota rejection, got %v", err)
		}
	})

	t.Run("wrong digest cannot replace active generation", func(t *testing.T) {
		wrong := artifact
		wrong.SHA256 = strings.Repeat("0", 64)
		assertFailedFetchPreservesActive(t, testCatalogHTTPClient(server.Client()), context.Background(), wrong, int64(len(payload)), errDownloadIntegrity)
	})

	t.Run("existing verified generation is never replaced", func(t *testing.T) {
		parent := t.TempDir()
		verified := filepath.Join(parent, "model.verified")
		if err := os.WriteFile(verified, []byte("existing"), 0o600); err != nil {
			t.Fatal(err)
		}
		err := fetchTrustedArtifact(context.Background(), testCatalogHTTPClient(server.Client()), artifact, parent, "model.partial", "model.verified", int64(len(payload)))
		if !errors.Is(err, os.ErrExist) {
			t.Fatalf("expected no-replace rejection, got %v", err)
		}
		actual, readErr := os.ReadFile(verified)
		if readErr != nil || string(actual) != "existing" {
			t.Fatalf("verified generation changed: %q, %v", actual, readErr)
		}
	})

	t.Run("cancellation cannot replace active generation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		assertFailedFetchPreservesActive(t, testCatalogHTTPClient(server.Client()), ctx, artifact, int64(len(payload)), context.Canceled)
	})

	t.Run("midstream cancellation resumes with stable validator", func(t *testing.T) {
		largePayload := []byte(strings.Repeat("bounded-model-chunk-", 4096))
		largeDigestBytes := sha256.Sum256(largePayload)
		largeDigest := hex.EncodeToString(largeDigestBytes[:])
		const cut = 8192
		prefixSent := make(chan struct{})
		interrupted := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			writer.Header().Set("ETag", etag)
			rangeHeader := request.Header.Get("Range")
			if rangeHeader == "" {
				writer.WriteHeader(http.StatusOK)
				_, _ = writer.Write(largePayload[:cut])
				if flusher, ok := writer.(http.Flusher); ok {
					flusher.Flush()
				}
				close(prefixSent)
				<-request.Context().Done()
				return
			}
			var start int64
			if _, err := fmt.Sscanf(rangeHeader, "bytes=%d-", &start); err != nil || start <= 0 || start >= int64(len(largePayload)) {
				http.Error(writer, "bad range", http.StatusRequestedRangeNotSatisfiable)
				return
			}
			writer.Header().Set("Content-Range", contentRange(start, int64(len(largePayload)-1), int64(len(largePayload))))
			writer.WriteHeader(http.StatusPartialContent)
			_, _ = writer.Write(largePayload[start:])
		}))
		defer interrupted.Close()
		resumable := trustedArtifact{
			URL: interrupted.URL + "/model.onnx", Origin: interrupted.URL, ETag: etag,
			Size: int64(len(largePayload)), SHA256: largeDigest, allowLoopbackForTest: true,
		}
		parent := t.TempDir()
		ctx, cancel := context.WithCancel(context.Background())
		result := make(chan error, 1)
		go func() {
			result <- fetchTrustedArtifact(ctx, testCatalogHTTPClient(interrupted.Client()), resumable, parent, "model.partial", "model.verified", int64(len(largePayload)))
		}()
		<-prefixSent
		partial := filepath.Join(parent, "model.partial")
		deadline := time.Now().Add(time.Second)
		for {
			info, err := os.Stat(partial)
			if err == nil && info.Size() == cut {
				break
			}
			if time.Now().After(deadline) {
				t.Fatalf("partial prefix was not persisted before cancellation: %v", err)
			}
			time.Sleep(time.Millisecond)
		}
		cancel()
		if err := <-result; !errors.Is(err, context.Canceled) {
			t.Fatalf("expected midstream cancellation, got %v", err)
		}
		if err := fetchTrustedArtifact(context.Background(), testCatalogHTTPClient(interrupted.Client()), resumable, parent, "model.partial", "model.verified", int64(len(largePayload))); err != nil {
			t.Fatalf("resume after cancellation: %v", err)
		}
		actual, err := os.ReadFile(filepath.Join(parent, "model.verified"))
		if err != nil || string(actual) != string(largePayload) {
			t.Fatalf("resumed artifact mismatch: %v", err)
		}
	})
}

func TestFetchTrustedArtifactActualENOSPC(t *testing.T) {
	root := os.Getenv("INT001_ENOSPC_DIR")
	if root == "" {
		t.Skip("set INT001_ENOSPC_DIR to a deliberately size-limited filesystem")
	}
	parent, err := os.MkdirTemp(root, "download-enospc-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(parent)
	payload := []byte(strings.Repeat("model-block-", 128*1024))
	digestBytes := sha256.Sum256(payload)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("ETag", `"enospc-generation"`)
		_, _ = writer.Write(payload)
	}))
	defer server.Close()
	artifact := trustedArtifact{
		URL: server.URL + "/model.onnx", Origin: server.URL,
		ETag: `"enospc-generation"`, Size: int64(len(payload)),
		SHA256: hex.EncodeToString(digestBytes[:]), allowLoopbackForTest: true,
	}
	active := filepath.Join(parent, "active-generation")
	if err := os.WriteFile(active, []byte("current"), 0o600); err != nil {
		t.Fatal(err)
	}
	err = fetchTrustedArtifact(context.Background(), testCatalogHTTPClient(server.Client()), artifact, parent, "model.partial", "candidate-generation", int64(len(payload)))
	if !errors.Is(err, syscall.ENOSPC) {
		t.Fatalf("expected kernel ENOSPC, got %v", err)
	}
	actual, readErr := os.ReadFile(active)
	if readErr != nil || string(actual) != "current" {
		t.Fatalf("active generation changed after ENOSPC: %q, %v", actual, readErr)
	}
	if _, statErr := os.Stat(filepath.Join(parent, "candidate-generation")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("ENOSPC candidate became visible: %v", statErr)
	}
}

func TestFetchTrustedArtifactStrongKillRecovery(t *testing.T) {
	if os.Getenv("INT001_STRONG_KILL_HELPER") == "1" {
		runStrongKillHelper(t)
		return
	}
	payload := strongKillPayload()
	digestBytes := sha256.Sum256(payload)
	const etag = `"strong-kill-generation"`
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("ETag", etag)
		rangeHeader := request.Header.Get("Range")
		if rangeHeader != "" {
			var start int64
			if _, err := fmt.Sscanf(rangeHeader, "bytes=%d-", &start); err != nil || start <= 0 || start >= int64(len(payload)) {
				http.Error(writer, "bad range", http.StatusRequestedRangeNotSatisfiable)
				return
			}
			writer.Header().Set("Content-Range", contentRange(start, int64(len(payload)-1), int64(len(payload))))
			writer.WriteHeader(http.StatusPartialContent)
			_, _ = writer.Write(payload[start:])
			return
		}
		writer.Header().Set("Content-Length", strconv.Itoa(len(payload)))
		writer.WriteHeader(http.StatusOK)
		for offset := 0; offset < len(payload); offset += 4096 {
			end := min(offset+4096, len(payload))
			if _, err := writer.Write(payload[offset:end]); err != nil {
				return
			}
			if flusher, ok := writer.(http.Flusher); ok {
				flusher.Flush()
			}
			time.Sleep(time.Millisecond)
		}
	}))
	defer server.Close()
	parent := t.TempDir()
	active := filepath.Join(parent, "active-generation")
	if err := os.WriteFile(active, []byte("current"), 0o600); err != nil {
		t.Fatal(err)
	}
	helperEnvironment := append(os.Environ(),
		"INT001_STRONG_KILL_HELPER=1",
		"INT001_STRONG_KILL_URL="+server.URL+"/model.onnx",
		"INT001_STRONG_KILL_ORIGIN="+server.URL,
		"INT001_STRONG_KILL_ETAG="+etag,
		"INT001_STRONG_KILL_SIZE="+strconv.Itoa(len(payload)),
		"INT001_STRONG_KILL_SHA256="+hex.EncodeToString(digestBytes[:]),
		"INT001_STRONG_KILL_PARENT="+parent,
	)
	first := exec.Command(os.Args[0], "-test.run=^TestFetchTrustedArtifactStrongKillRecovery$")
	first.Env = helperEnvironment
	if err := first.Start(); err != nil {
		t.Fatal(err)
	}
	partial := filepath.Join(parent, "model.partial")
	deadline := time.Now().Add(5 * time.Second)
	for {
		info, err := os.Stat(partial)
		if err == nil && info.Size() >= 32*1024 && info.Size() < int64(len(payload)) {
			break
		}
		if time.Now().After(deadline) {
			_ = first.Process.Kill()
			_, _ = first.Process.Wait()
			t.Fatalf("helper did not persist a bounded partial before deadline: %v", err)
		}
		time.Sleep(time.Millisecond)
	}
	if err := first.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	if err := first.Wait(); err == nil {
		t.Fatal("strong-killed helper unexpectedly exited successfully")
	}
	partialInfo, err := os.Stat(partial)
	if err != nil || partialInfo.Size() <= 0 || partialInfo.Size() >= int64(len(payload)) {
		t.Fatalf("strong kill did not preserve a resumable partial: %v", err)
	}
	second := exec.Command(os.Args[0], "-test.run=^TestFetchTrustedArtifactStrongKillRecovery$")
	second.Env = helperEnvironment
	if output, err := second.CombinedOutput(); err != nil {
		t.Fatalf("restarted helper failed: %v\n%s", err, output)
	}
	verified, err := os.ReadFile(filepath.Join(parent, "model.verified"))
	if err != nil || string(verified) != string(payload) {
		t.Fatalf("restarted helper published wrong bytes: %v", err)
	}
	current, err := os.ReadFile(active)
	if err != nil || string(current) != "current" {
		t.Fatalf("active generation changed across strong kill: %q, %v", current, err)
	}
}

func runStrongKillHelper(t *testing.T) {
	t.Helper()
	size, err := strconv.ParseInt(os.Getenv("INT001_STRONG_KILL_SIZE"), 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	artifact := trustedArtifact{
		URL: os.Getenv("INT001_STRONG_KILL_URL"), Origin: os.Getenv("INT001_STRONG_KILL_ORIGIN"),
		ETag: os.Getenv("INT001_STRONG_KILL_ETAG"), Size: size,
		SHA256: os.Getenv("INT001_STRONG_KILL_SHA256"), allowLoopbackForTest: true,
	}
	if err := fetchTrustedArtifact(context.Background(), testCatalogHTTPClient(http.DefaultClient), artifact, os.Getenv("INT001_STRONG_KILL_PARENT"), "model.partial", "model.verified", size); err != nil {
		t.Fatal(err)
	}
}

func strongKillPayload() []byte {
	return []byte(strings.Repeat("restart-safe-model-block-", 128*1024))
}

func assertFailedFetchPreservesActive(t *testing.T, client *catalogHTTPClient, ctx context.Context, artifact trustedArtifact, quota int64, expected error) {
	t.Helper()
	parent := t.TempDir()
	active := filepath.Join(parent, "active-generation")
	if err := os.WriteFile(active, []byte("current"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := fetchTrustedArtifact(ctx, client, artifact, parent, "model.partial", "candidate-generation", quota)
	if !errors.Is(err, expected) {
		t.Fatalf("expected %v, got %v", expected, err)
	}
	actual, readErr := os.ReadFile(active)
	if readErr != nil || string(actual) != "current" {
		t.Fatalf("active generation changed: %q, %v", actual, readErr)
	}
	if _, statErr := os.Stat(filepath.Join(parent, "candidate-generation")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("failed candidate became visible: %v", statErr)
	}
}
