package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"
)

var errCatalogAddress = errors.New("catalog origin resolved to a forbidden address")

const defaultCatalogAddressAttemptTimeout = 5 * time.Second

var catalogForbiddenPrefixes = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("10.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("127.0.0.0/8"),
	netip.MustParsePrefix("169.254.0.0/16"),
	netip.MustParsePrefix("172.16.0.0/12"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("192.168.0.0/16"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("224.0.0.0/4"),
	netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("100::/64"),
	netip.MustParsePrefix("2001:db8::/32"),
	netip.MustParsePrefix("fc00::/7"),
	netip.MustParsePrefix("fe80::/10"),
	netip.MustParsePrefix("ff00::/8"),
}

type catalogResolver interface {
	LookupIPAddr(context.Context, string) ([]net.IPAddr, error)
}

type catalogDialContext func(context.Context, string, string) (net.Conn, error)

// catalogHTTPClient marks a client whose origin was resolved once and whose
// transport dials only those validated addresses. Tests may wrap httptest's
// loopback client only for artifacts carrying the private test exception.
type catalogHTTPClient struct {
	client *http.Client
}

func newCatalogHTTPClient(
	ctx context.Context,
	artifact trustedArtifact,
	resolver catalogResolver,
	dial catalogDialContext,
) (*catalogHTTPClient, error) {
	return newCatalogHTTPClientWithRoots(ctx, artifact, resolver, dial, nil)
}

func newCatalogHTTPClientWithRoots(
	ctx context.Context,
	artifact trustedArtifact,
	resolver catalogResolver,
	dial catalogDialContext,
	rootCAs *x509.CertPool,
) (*catalogHTTPClient, error) {
	if resolver == nil || dial == nil {
		return nil, errors.New("catalog resolver and dialer are required")
	}
	if err := validateArtifactOrigin(artifact.URL, artifact.Origin, artifact.allowLoopbackForTest); err != nil {
		return nil, err
	}
	parsed, err := url.Parse(artifact.URL)
	if err != nil {
		return nil, err
	}
	addresses, err := resolveCatalogAddresses(ctx, resolver, parsed.Hostname(), artifact.allowLoopbackForTest)
	if err != nil {
		return nil, err
	}
	pinned := &pinnedCatalogDialer{
		hostname:       parsed.Hostname(),
		addresses:      addresses,
		dial:           dial,
		attemptTimeout: defaultCatalogAddressAttemptTimeout,
	}
	transport := &http.Transport{
		Proxy:                 nil,
		DialContext:           pinned.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          2,
		MaxIdleConnsPerHost:   1,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   15 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
			ServerName: parsed.Hostname(),
			RootCAs:    rootCAs,
		},
	}
	return &catalogHTTPClient{client: &http.Client{Transport: transport}}, nil
}

func testCatalogHTTPClient(client *http.Client) *catalogHTTPClient {
	return &catalogHTTPClient{client: client}
}

func resolveCatalogAddresses(ctx context.Context, resolver catalogResolver, hostname string, allowLoopbackForTest bool) ([]net.IPAddr, error) {
	addresses, err := resolver.LookupIPAddr(ctx, hostname)
	if err != nil {
		return nil, fmt.Errorf("resolve catalog origin: %w", err)
	}
	if len(addresses) == 0 {
		return nil, errCatalogAddress
	}
	seen := make(map[string]struct{}, len(addresses))
	validated := make([]net.IPAddr, 0, len(addresses))
	for _, address := range addresses {
		if !catalogAddressAllowed(address.IP, allowLoopbackForTest) {
			return nil, fmt.Errorf("%w: %s", errCatalogAddress, address.IP)
		}
		key := address.IP.String()
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		validated = append(validated, address)
	}
	return validated, nil
}

func catalogAddressAllowed(address net.IP, allowLoopbackForTest bool) bool {
	parsed, ok := netip.AddrFromSlice(address)
	if !ok {
		return false
	}
	parsed = parsed.Unmap()
	if parsed.IsLoopback() {
		return allowLoopbackForTest
	}
	if !parsed.IsGlobalUnicast() || parsed.IsPrivate() || parsed.IsLinkLocalUnicast() || parsed.IsLinkLocalMulticast() {
		return false
	}
	for _, prefix := range catalogForbiddenPrefixes {
		if prefix.Contains(parsed) {
			return false
		}
	}
	return true
}

type pinnedCatalogDialer struct {
	hostname       string
	addresses      []net.IPAddr
	dial           catalogDialContext
	attemptTimeout time.Duration
}

func (dialer *pinnedCatalogDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	hostname, port, err := net.SplitHostPort(address)
	if err != nil || !strings.EqualFold(hostname, dialer.hostname) {
		return nil, errDownloadOrigin
	}
	var lastErr error
	for _, pinned := range dialer.addresses {
		if network == "tcp4" && pinned.IP.To4() == nil || network == "tcp6" && pinned.IP.To4() != nil {
			continue
		}
		attemptContext := ctx
		cancel := func() {}
		if dialer.attemptTimeout > 0 {
			attemptContext, cancel = context.WithTimeout(ctx, dialer.attemptTimeout)
		}
		connection, err := dialer.dial(
			attemptContext, network, net.JoinHostPort(pinned.IP.String(), port),
		)
		cancel()
		if err == nil {
			return connection, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = errors.New("no catalog address matched the requested network")
	}
	return nil, lastErr
}
