// Package downloadhttp streams fixed, deployment-approved model files over HTTPS.
package downloadhttp

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/http"
	"net/netip"
	"slices"
	"time"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
)

type Source struct{ transport http.RoundTripper }

func New() *Source {
	dialer := &net.Dialer{Timeout: 15 * time.Second, KeepAlive: 30 * time.Second}
	transport := &http.Transport{
		TLSClientConfig:     &tls.Config{MinVersion: tls.VersionTLS12},
		TLSHandshakeTimeout: 15 * time.Second, ResponseHeaderTimeout: 30 * time.Second,
		IdleConnTimeout: 30 * time.Second, MaxIdleConns: 2, MaxConnsPerHost: 2,
		DisableCompression: true,
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(address)
			if err != nil {
				return nil, aimodel.ErrDownloadUnavailable
			}
			addresses, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
			if err != nil || len(addresses) == 0 {
				return nil, aimodel.ErrDownloadUnavailable
			}
			for _, ip := range addresses {
				if !publicAddress(ip) {
					return nil, aimodel.ErrDownloadUnavailable
				}
			}
			for _, ip := range addresses {
				conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
				if err == nil {
					return conn, nil
				}
			}
			return nil, aimodel.ErrDownloadUnavailable
		},
	}
	return &Source{transport: transport}
}

func publicAddress(ip netip.Addr) bool {
	ip = ip.Unmap()
	if !ip.IsValid() || !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
		return false
	}
	for _, prefix := range blockedPrefixes {
		if prefix.Contains(ip) {
			return false
		}
	}
	return true
}

var blockedPrefixes = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"), netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("192.0.0.0/24"), netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("198.18.0.0/15"), netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"), netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("64:ff9b::/96"), netip.MustParsePrefix("64:ff9b:1::/48"),
	netip.MustParsePrefix("100::/64"), netip.MustParsePrefix("2001::/23"),
	netip.MustParsePrefix("2001:db8::/32"), netip.MustParsePrefix("2002::/16"),
}

func (source *Source) OpenDownload(parent context.Context, location string, redirectHosts []string, size int64) (io.ReadCloser, error) {
	if !aimodel.ValidDownloadURL(location) || size <= 0 || size > aimodel.MaxPackageBytes {
		return nil, aimodel.ErrDownloadUnavailable
	}
	ctx, cancel := context.WithTimeout(parent, 30*time.Minute)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, location, nil)
	if err != nil {
		cancel()
		return nil, aimodel.ErrDownloadUnavailable
	}
	request.Header.Set("Accept-Encoding", "identity")
	request.Header.Set("User-Agent", "FolioPath-model-downloader")
	host := request.URL.Hostname()
	client := &http.Client{Transport: source.transport, CheckRedirect: func(next *http.Request, via []*http.Request) error {
		if len(via) >= 5 || !aimodel.ValidDownloadURL(next.URL.String()) ||
			(next.URL.Hostname() != host && !slices.Contains(redirectHosts, next.URL.Hostname())) {
			return aimodel.ErrDownloadUnavailable
		}
		return nil
	}}
	response, err := client.Do(request)
	if err != nil {
		cancel()
		if parent.Err() != nil {
			return nil, parent.Err()
		}
		return nil, aimodel.ErrDownloadUnavailable
	}
	if response.StatusCode != http.StatusOK || (response.ContentLength >= 0 && response.ContentLength != size) ||
		(response.Header.Get("Content-Encoding") != "" && response.Header.Get("Content-Encoding") != "identity") {
		_ = response.Body.Close()
		cancel()
		return nil, aimodel.ErrDownloadUnavailable
	}
	return &downloadBody{body: response.Body, reader: io.LimitReader(response.Body, size+1), cancel: cancel}, nil
}

type downloadBody struct {
	body   io.ReadCloser
	reader io.Reader
	cancel context.CancelFunc
}

func (body *downloadBody) Read(buffer []byte) (int, error) {
	n, err := body.reader.Read(buffer)
	if err != nil && !errors.Is(err, io.EOF) {
		return n, aimodel.ErrDownloadUnavailable
	}
	return n, err
}
func (body *downloadBody) Close() error { body.cancel(); return body.body.Close() }
