package downloadhttp

import (
	"context"
	"crypto/x509"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
)

func TestProxyConfiguration(t *testing.T) {
	for _, value := range []string{"", "http://192.168.2.231:7890", "https://proxy.example/", "http://[::1]:7890"} {
		if _, err := NewWithProxy(value); err != nil {
			t.Fatalf("valid proxy rejected: %v", err)
		}
	}
	for _, value := range []string{"socks5://proxy.example:1080", "http://user:secret@proxy.example", "http://proxy.example/file", "http://proxy.example?", "http://proxy.example#", "http://proxy.example:0", "http://proxy.example:65536", "http://", "secret"} {
		if _, err := NewWithProxy(value); !errors.Is(err, ErrInvalidProxy) || strings.Contains(err.Error(), "secret") {
			t.Fatalf("bad proxy validation: %v", err)
		}
	}
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:9")
	t.Setenv("NO_PROXY", "*")
	if New().transport.(*http.Transport).Proxy != nil {
		t.Fatal("inherited ambient proxy")
	}
}

func TestProxyConnectTLSAndBoundary(t *testing.T) {
	upstream := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirect" {
			http.Redirect(w, r, "https://127.0.0.1/model", 302)
			return
		}
		io.WriteString(w, "model")
	}))
	defer upstream.Close()
	var connections atomic.Int64
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		connections.Add(1)
		if r.Method != "CONNECT" || r.Host != "example.com:443" {
			t.Error("unexpected CONNECT target", r.Method, r.Host)
			http.Error(w, "blocked", 403)
			return
		}
		target, err := net.DialTimeout("tcp", upstream.Listener.Addr().String(), time.Second)
		if err != nil {
			http.Error(w, "unavailable", 502)
			return
		}
		client, buffered, err := w.(http.Hijacker).Hijack()
		if err != nil {
			target.Close()
			return
		}
		defer client.Close()
		defer target.Close()
		io.WriteString(client, "HTTP/1.1 200 Connection established\r\n\r\n")
		go func() { io.Copy(target, buffered); target.Close() }()
		io.Copy(client, target)
	}))
	defer proxy.Close()
	source, err := NewWithProxy(proxy.URL)
	if err != nil {
		t.Fatal(err)
	}
	transport := source.transport.(*http.Transport)
	defer transport.CloseIdleConnections()
	// Trust only the synthetic origin certificate; production keeps system roots.
	pool := x509.NewCertPool()
	pool.AddCert(upstream.Certificate())
	transport.TLSClientConfig.RootCAs = pool
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	reader, err := source.OpenDownload(ctx, "https://example.com/model", nil, 5)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(reader)
	reader.Close()
	if err != nil || string(body) != "model" || connections.Load() != 1 {
		t.Fatalf("proxied stream: %q %v", body, err)
	}
	for _, target := range []string{"https://127.0.0.1/model", "https://169.254.169.254/model", "https://host.local/model", "https://localhost/model"} {
		if _, err := source.OpenDownload(ctx, target, nil, 5); !errors.Is(err, aimodel.ErrDownloadUnavailable) {
			t.Fatal("private target accepted", err)
		}
	}
	if _, err := source.OpenDownload(ctx, "https://example.com/redirect", []string{"127.0.0.1"}, 5); !errors.Is(err, aimodel.ErrDownloadUnavailable) {
		t.Fatal("private redirect accepted", err)
	}
	if _, err := transport.DialContext(ctx, "tcp", "elsewhere.example:443"); !errors.Is(err, aimodel.ErrDownloadUnavailable) {
		t.Fatal("direct fallback allowed", err)
	}
	untrusted, _ := NewWithProxy(proxy.URL)
	defer untrusted.transport.(*http.Transport).CloseIdleConnections()
	if _, err := untrusted.OpenDownload(ctx, "https://example.com/model", nil, 5); !errors.Is(err, aimodel.ErrDownloadUnavailable) {
		t.Fatal("untrusted TLS accepted", err)
	}
	cancelled, stop := context.WithCancel(context.Background())
	stop()
	if _, err := source.OpenDownload(cancelled, "https://example.com/model", nil, 5); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
