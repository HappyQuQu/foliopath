package downloadhttp

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
)

var ErrInvalidProxy = errors.New("invalid model download proxy configuration")

func ParseProxyURL(value string) (*url.URL, error) {
	if value == "" {
		return nil, nil
	}
	u, err := url.Parse(value)
	if err != nil || len(value) > 2048 || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" ||
		u.User != nil || u.Opaque != "" || (u.Path != "" && u.Path != "/") || u.RawPath != "" || strings.HasSuffix(u.Host, ":") || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || strings.Contains(value, "#") {
		return nil, ErrInvalidProxy
	}
	if port := u.Port(); port != "" {
		p, err := net.LookupPort("tcp", port)
		if err != nil || p < 1 || p > 65535 {
			return nil, ErrInvalidProxy
		}
	}
	return u, nil
}

// NewWithProxy uses only the explicitly configured trusted proxy. DNS for the
// destination is delegated to it; TLS verification and catalog policy stay local.
func NewWithProxy(value string) (*Source, error) {
	proxy, err := ParseProxyURL(value)
	if err != nil {
		return nil, err
	}
	source := New()
	if proxy == nil {
		return source, nil
	}
	transport := source.transport.(*http.Transport)
	transport.Proxy = http.ProxyURL(proxy)
	port := proxy.Port()
	if port == "" {
		port = "80"
		if proxy.Scheme == "https" {
			port = "443"
		}
	}
	address := net.JoinHostPort(proxy.Hostname(), port)
	dialer := &net.Dialer{Timeout: 15 * time.Second, KeepAlive: 30 * time.Second}
	transport.DialContext = func(ctx context.Context, network, target string) (net.Conn, error) {
		if target != address {
			return nil, aimodel.ErrDownloadUnavailable
		}
		return dialer.DialContext(ctx, network, address)
	}
	return source, nil
}

func validTarget(value string) bool {
	if !aimodel.ValidDownloadURL(value) {
		return false
	}
	u, _ := url.Parse(value)
	host := strings.ToLower(strings.TrimSuffix(u.Hostname(), "."))
	if ip, err := netip.ParseAddr(host); err == nil {
		return publicAddress(ip)
	}
	return strings.Contains(host, ".") && !strings.HasSuffix(host, ".localhost") &&
		!strings.HasSuffix(host, ".local") && !strings.HasSuffix(host, ".localdomain")
}
