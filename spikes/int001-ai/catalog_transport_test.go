package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"errors"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type staticCatalogResolver struct {
	addresses []net.IPAddr
	calls     int
	err       error
}

func (resolver *staticCatalogResolver) LookupIPAddr(context.Context, string) ([]net.IPAddr, error) {
	resolver.calls++
	if resolver.err != nil {
		return nil, resolver.err
	}
	return append([]net.IPAddr(nil), resolver.addresses...), nil
}

func TestCatalogNetworkPolicyRejectsUnsafeDNSAnswers(t *testing.T) {
	for _, test := range []struct {
		name      string
		addresses []net.IPAddr
	}{
		{name: "loopback", addresses: []net.IPAddr{{IP: net.ParseIP("127.0.0.1")}}},
		{name: "private", addresses: []net.IPAddr{{IP: net.ParseIP("10.2.3.4")}}},
		{name: "link-local", addresses: []net.IPAddr{{IP: net.ParseIP("169.254.1.2")}}},
		{name: "carrier-grade-nat", addresses: []net.IPAddr{{IP: net.ParseIP("100.64.1.2")}}},
		{name: "benchmark", addresses: []net.IPAddr{{IP: net.ParseIP("198.18.1.2")}}},
		{name: "documentation-v6", addresses: []net.IPAddr{{IP: net.ParseIP("2001:db8::1")}}},
		{name: "mapped-private-v4", addresses: []net.IPAddr{{IP: net.ParseIP("::ffff:192.168.1.9")}}},
		{name: "mixed-public-private", addresses: []net.IPAddr{{IP: net.ParseIP("93.184.216.34")}, {IP: net.ParseIP("192.168.1.9")}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			resolver := &staticCatalogResolver{addresses: test.addresses}
			_, err := resolveCatalogAddresses(context.Background(), resolver, "models.example.test", false)
			if !errors.Is(err, errCatalogAddress) {
				t.Fatalf("expected forbidden DNS answer, got %v", err)
			}
		})
	}
}

func TestCatalogResolutionFailuresFailClosedWithoutImplicitRetry(t *testing.T) {
	resolutionFailure := errors.New("resolver unavailable")
	for _, resolver := range []*staticCatalogResolver{
		{err: resolutionFailure},
		{},
	} {
		_, err := newCatalogHTTPClient(
			context.Background(),
			trustedArtifact{
				URL:    "https://models.example.test/model.onnx",
				Origin: "https://models.example.test",
			},
			resolver,
			func(context.Context, string, string) (net.Conn, error) {
				t.Fatal("dial attempted after failed resolution")
				return nil, nil
			},
		)
		if err == nil {
			t.Fatal("failed or empty resolution unexpectedly created a client")
		}
		if resolver.calls != 1 {
			t.Fatalf("resolver calls = %d, want one fail-closed attempt", resolver.calls)
		}
	}
}

func TestCatalogClientPinsOneResolutionAndDisablesProxy(t *testing.T) {
	resolver := &staticCatalogResolver{addresses: []net.IPAddr{{IP: net.ParseIP("93.184.216.34")}}}
	var dialed string
	dialFailure := errors.New("stop after observing pinned dial")
	client, err := newCatalogHTTPClient(context.Background(), trustedArtifact{
		URL: "https://models.example.test/model.onnx", Origin: "https://models.example.test",
	}, resolver, func(_ context.Context, _, address string) (net.Conn, error) {
		dialed = address
		return nil, dialFailure
	})
	if err != nil {
		t.Fatal(err)
	}
	transport, ok := client.client.Transport.(*http.Transport)
	if !ok || transport.Proxy != nil {
		t.Fatal("catalog transport must ignore environment proxy configuration")
	}
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:9")
	_, err = client.client.Get("https://models.example.test/model.onnx")
	if err == nil || !strings.Contains(err.Error(), dialFailure.Error()) {
		t.Fatalf("expected injected dial failure, got %v", err)
	}
	if resolver.calls != 1 {
		t.Fatalf("catalog hostname resolved %d times, want exactly one", resolver.calls)
	}
	if dialed != "93.184.216.34:443" {
		t.Fatalf("transport did not dial the pinned address: %q", dialed)
	}
}

func TestPinnedCatalogDialerBoundsEachAddressAttempt(t *testing.T) {
	secondAttempted := false
	dialer := &pinnedCatalogDialer{
		hostname: "models.example.test",
		addresses: []net.IPAddr{
			{IP: net.ParseIP("93.184.216.34")},
			{IP: net.ParseIP("93.184.216.35")},
		},
		attemptTimeout: 10 * time.Millisecond,
		dial: func(ctx context.Context, _, address string) (net.Conn, error) {
			if strings.HasPrefix(address, "93.184.216.34:") {
				<-ctx.Done()
				return nil, ctx.Err()
			}
			secondAttempted = true
			client, server := net.Pipe()
			_ = server.Close()
			return client, nil
		},
	}
	started := time.Now()
	connection, err := dialer.DialContext(
		context.Background(), "tcp", "models.example.test:443",
	)
	if err != nil {
		t.Fatal(err)
	}
	_ = connection.Close()
	if !secondAttempted {
		t.Fatal("second validated address was not attempted after bounded timeout")
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("per-address fallback took %s, want below one second in test", elapsed)
	}
}

func TestCatalogPinnedClientFeedsVerifiedDownloader(t *testing.T) {
	payload := []byte("pinned-loopback-test-artifact")
	digest := sha256.Sum256(payload)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("ETag", `"pinned-test"`)
		_, _ = writer.Write(payload)
	}))
	defer server.Close()
	host, _, err := net.SplitHostPort(strings.TrimPrefix(server.URL, "http://"))
	if err != nil {
		t.Fatal(err)
	}
	artifact := trustedArtifact{
		URL: server.URL + "/model.onnx", Origin: server.URL, ETag: `"pinned-test"`,
		Size: int64(len(payload)), SHA256: hex.EncodeToString(digest[:]), allowLoopbackForTest: true,
	}
	resolver := &staticCatalogResolver{addresses: []net.IPAddr{{IP: net.ParseIP(host)}}}
	dialer := &net.Dialer{}
	client, err := newCatalogHTTPClient(context.Background(), artifact, resolver, dialer.DialContext)
	if err != nil {
		t.Fatal(err)
	}
	parent := t.TempDir()
	if err := fetchTrustedArtifact(context.Background(), client, artifact, parent, "model.partial", "model.verified", int64(len(payload))); err != nil {
		t.Fatal(err)
	}
	actual, err := os.ReadFile(filepath.Join(parent, "model.verified"))
	if err != nil || string(actual) != string(payload) {
		t.Fatalf("verified download mismatch: %v", err)
	}
}

func TestCatalogPinnedClientUsesTLSAndFallsBackWithinValidatedAddressSet(t *testing.T) {
	payload := []byte("tls-pinned-model-artifact")
	digest := sha256.Sum256(payload)
	server, certificatePool, origin := newCatalogTLSServer(t, http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("ETag", `"tls-pinned-test"`)
		_, _ = writer.Write(payload)
	}))
	defer server.Close()
	artifact := trustedArtifact{
		URL: origin + "/model.onnx", Origin: origin, ETag: `"tls-pinned-test"`,
		Size: int64(len(payload)), SHA256: hex.EncodeToString(digest[:]), allowLoopbackForTest: true,
	}
	resolver := &staticCatalogResolver{addresses: []net.IPAddr{
		{IP: net.ParseIP("127.0.0.2")},
		{IP: net.ParseIP("127.0.0.1")},
	}}
	dialer := &net.Dialer{}
	firstAddressFailed := false
	dial := func(ctx context.Context, network, address string) (net.Conn, error) {
		if strings.HasPrefix(address, "127.0.0.2:") {
			firstAddressFailed = true
			return nil, errors.New("simulated unavailable validated CDN address")
		}
		return dialer.DialContext(ctx, network, address)
	}
	client, err := newCatalogHTTPClientWithRoots(
		context.Background(), artifact, resolver, dial, certificatePool,
	)
	if err != nil {
		t.Fatal(err)
	}
	parent := t.TempDir()
	if err := fetchTrustedArtifact(
		context.Background(), client, artifact, parent,
		"model.partial", "model.verified", int64(len(payload)),
	); err != nil {
		t.Fatal(err)
	}
	if resolver.calls != 1 {
		t.Fatalf("TLS catalog hostname resolved %d times, want one pinned lookup", resolver.calls)
	}
	if !firstAddressFailed {
		t.Fatal("TLS catalog client did not try the first validated address before fallback")
	}
	actual, err := os.ReadFile(filepath.Join(parent, "model.verified"))
	if err != nil || string(actual) != string(payload) {
		t.Fatalf("TLS verified download mismatch: %v", err)
	}
	transport := client.client.Transport.(*http.Transport)
	if transport.TLSClientConfig.MinVersion != tls.VersionTLS12 {
		t.Fatalf("TLS minimum = %d, want TLS 1.2", transport.TLSClientConfig.MinVersion)
	}
}

func TestCatalogPinnedClientRejectsUntrustedTLSCertificate(t *testing.T) {
	server, _, origin := newCatalogTLSServer(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer server.Close()
	artifact := trustedArtifact{
		URL: origin + "/model.onnx", Origin: origin, allowLoopbackForTest: true,
	}
	resolver := &staticCatalogResolver{addresses: []net.IPAddr{{IP: net.ParseIP("127.0.0.1")}}}
	dialer := &net.Dialer{}
	client, err := newCatalogHTTPClient(
		context.Background(), artifact, resolver, dialer.DialContext,
	)
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.client.Get(artifact.URL)
	if response != nil {
		_ = response.Body.Close()
	}
	if err == nil {
		t.Fatal("untrusted TLS certificate unexpectedly accepted")
	}
}

func newCatalogTLSServer(
	t *testing.T,
	handler http.Handler,
) (*httptest.Server, *x509.CertPool, string) {
	t.Helper()
	now := time.Now()
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "FolioPath INT-001 test root"},
		NotBefore:             now.Add(-time.Minute),
		NotAfter:              now.Add(time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}
	caCertificate, err := x509.ParseCertificate(caDER)
	if err != nil {
		t.Fatal(err)
	}
	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	leafTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "models.example.test"},
		DNSNames:     []string{"models.example.test"},
		NotBefore:    now.Add(-time.Minute),
		NotAfter:     now.Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	leafDER, err := x509.CreateCertificate(rand.Reader, leafTemplate, caCertificate, &leafKey.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewUnstartedServer(handler)
	server.TLS = &tls.Config{
		MinVersion: tls.VersionTLS12,
		Certificates: []tls.Certificate{{
			Certificate: [][]byte{leafDER, caDER},
			PrivateKey:  leafKey,
		}},
	}
	server.StartTLS()
	_, port, err := net.SplitHostPort(strings.TrimPrefix(server.URL, "https://"))
	if err != nil {
		server.Close()
		t.Fatal(err)
	}
	certificatePool := x509.NewCertPool()
	certificatePool.AddCert(caCertificate)
	return server, certificatePool, "https://models.example.test:" + port
}
