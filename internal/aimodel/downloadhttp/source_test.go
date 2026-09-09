package downloadhttp

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
)

func TestDownloadStreamsTLSAndRejectsRedirectsAndSizes(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/redirect":
			http.Redirect(w, r, "https://elsewhere.example/file", http.StatusFound)
		case "/downgrade":
			http.Redirect(w, r, "http://127.0.0.1/file", http.StatusFound)
		case "/missing":
			http.NotFound(w, r)
		default:
			_, _ = io.WriteString(w, "model")
		}
	}))
	defer server.Close()
	// The injected transport maps a public HTTPS fixture name onto the local TLS
	// test server. Production always uses New's verified-address dialer.
	source := &Source{transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		copy := r.Clone(r.Context())
		target := *r.URL
		fixture, _ := http.NewRequest(http.MethodGet, server.URL, nil)
		target.Host = fixture.URL.Host
		copy.URL = &target
		return server.Client().Transport.RoundTrip(copy)
	})}
	reader, err := source.OpenDownload(context.Background(), "https://models.example/model", nil, 5)
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(reader)
	_ = reader.Close()
	if err != nil || string(data) != "model" {
		t.Fatalf("stream = %q %v", data, err)
	}
	reader, err = source.OpenDownload(context.Background(), "https://models.example/redirect", []string{"elsewhere.example"}, 5)
	if err != nil {
		t.Fatal(err)
	}
	data, err = io.ReadAll(reader)
	_ = reader.Close()
	if err != nil || string(data) != "model" {
		t.Fatalf("allowed redirect: %q %v", data, err)
	}
	for _, path := range []string{"/redirect", "/downgrade", "/missing"} {
		if reader, err := source.OpenDownload(context.Background(), "https://models.example"+path, nil, 5); err == nil {
			reader.Close()
			t.Fatalf("accepted %s", path)
		}
	}
	if _, err := source.OpenDownload(context.Background(), "https://models.example/model", nil, 4); !errors.Is(err, aimodel.ErrDownloadUnavailable) {
		t.Fatal("accepted wrong size")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := source.OpenDownload(ctx, "https://models.example/model", nil, 5); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestDownloadPublicAddressBoundary(t *testing.T) {
	for _, address := range []string{"127.0.0.1", "10.0.0.1", "172.16.1.1", "192.168.2.222", "169.254.169.254", "100.100.100.200", "::1", "::ffff:127.0.0.1", "fd00::1", "fe80::1", "64:ff9b::7f00:1", "198.18.0.1", "224.0.0.1", "0.0.0.0"} {
		if publicAddress(netip.MustParseAddr(address)) {
			t.Fatalf("accepted reserved %s", address)
		}
	}
	if !publicAddress(netip.MustParseAddr("1.1.1.1")) {
		t.Fatal("rejected public IP")
	}
	_, err := New().OpenDownload(context.Background(), "https://127.0.0.1/model", nil, 1)
	if !errors.Is(err, aimodel.ErrDownloadUnavailable) || strings.Contains(err.Error(), "127.0.0.1") {
		t.Fatalf("private target: %v", err)
	}
}
