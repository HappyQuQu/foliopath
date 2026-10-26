//go:build modelnetwork

package downloadhttp

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"
)

// Explicit opt-in probe for a public pinned tokenizer, never developer media.
// This test does not register, activate, or publish a model.
func TestExplicitProxyPinnedTokenizer(t *testing.T) {
	proxy := os.Getenv("FOLIOPATH_TEST_MODEL_PROXY")
	if proxy == "" {
		t.Skip("set FOLIOPATH_TEST_MODEL_PROXY for the opt-in network probe")
	}
	source, err := NewWithProxy(proxy)
	if err != nil {
		t.Fatal(err)
	}
	defer source.transport.(*http.Transport).CloseIdleConnections()
	transport := source.transport
	source.transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		response, err := transport.RoundTrip(request)
		if response != nil {
			t.Logf("source host=%s status=%d", request.URL.Hostname(), response.StatusCode)
			if location, err := url.Parse(response.Header.Get("Location")); err == nil && location.Host != "" {
				t.Logf("redirect host=%s", location.Hostname())
			}
		}
		if err != nil {
			t.Logf("transport failure type=%T", err)
		}
		return response, err
	})
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	reader, err := source.OpenDownload(ctx, "https://huggingface.co/google/siglip-base-patch16-224/resolve/7fd15f0689c79d79e38b1c2e2e2370a7bf2761ed/spiece.model", []string{"us.aws.cdn.hf.co"}, 798330)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	hash := sha256.New()
	size, err := io.Copy(hash, reader)
	if err != nil || size != 798330 || fmt.Sprintf("%x", hash.Sum(nil)) != "1e5036bed065526c3c212dfbe288752391797c4bb1a284aa18c9a0b23fcaf8ec" {
		t.Fatalf("pinned tokenizer verification failed: size=%d err=%v", size, err)
	}
}
