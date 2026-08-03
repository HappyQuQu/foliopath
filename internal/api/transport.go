package api

import (
	"context"
	"net"
	"net/http"
	"net/netip"
	"strings"
)

const (
	forwardedForHeader   = "X-Forwarded-For"
	forwardedHostHeader  = "X-Forwarded-Host"
	forwardedProtoHeader = "X-Forwarded-Proto"
)

type transportContextKey struct{}

type requestTransport struct {
	authority  string
	clientHost string
	secure     bool
}

// TransportConfig defines the only peers allowed to assert the public HTTPS
// transport and client identity. RequireTrustedProxy opts a listener into a
// proxy-only boundary; direct authenticated LAN HTTP leaves it disabled.
type TransportConfig struct {
	TrustedProxyPrefixes []netip.Prefix
	RequireTrustedProxy  bool
	SystemEvents         SystemEventRecorder
}

func withTrustedTransport(next http.Handler, config TransportConfig) http.Handler {
	next = fallbackHandler(next)
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		directHost, directAddressOK := splitRemoteAddress(request.RemoteAddr)
		directIP, directIPErr := netip.ParseAddr(directHost)
		directIPOK := directIPErr == nil
		trusted := directAddressOK && directIPOK &&
			prefixesContain(config.TrustedProxyPrefixes, directIP.Unmap())

		if !trusted {
			clearForwardingHeaders(request.Header)
			if config.RequireTrustedProxy && !(directIPOK && directIP.IsLoopback()) {
				writePublicError(
					writer,
					request,
					http.StatusBadRequest,
					"trusted_proxy_required",
					"The request did not arrive through a trusted proxy.",
				)
				return
			}
			transport := requestTransport{
				authority:  request.Host,
				clientHost: directHost,
				secure:     request.TLS != nil,
			}
			next.ServeHTTP(writer, request.WithContext(
				context.WithValue(request.Context(), transportContextKey{}, transport),
			))
			return
		}

		transport, ok := trustedProxyTransport(request)
		clearForwardingHeaders(request.Header)
		if !ok {
			writePublicError(
				writer,
				request,
				http.StatusBadRequest,
				"proxy_headers_invalid",
				"The trusted proxy headers are invalid.",
			)
			return
		}
		next.ServeHTTP(writer, request.WithContext(
			context.WithValue(request.Context(), transportContextKey{}, transport),
		))
	})
}

func trustedProxyTransport(request *http.Request) (requestTransport, bool) {
	if len(request.Header.Values("Forwarded")) != 0 {
		return requestTransport{}, false
	}
	proto, protoOK := singleForwardedValue(request.Header, forwardedProtoHeader)
	authority, hostOK := singleForwardedValue(request.Header, forwardedHostHeader)
	client, clientOK := singleForwardedValue(request.Header, forwardedForHeader)
	if !protoOK || !hostOK || !clientOK ||
		proto != "https" ||
		strings.ContainsAny(authority, " \t,") ||
		strings.ContainsAny(client, " \t,") {
		return requestTransport{}, false
	}
	if _, _, ok := canonicalAuthority(authority, "https"); !ok {
		return requestTransport{}, false
	}
	clientIP, err := netip.ParseAddr(client)
	if err != nil {
		return requestTransport{}, false
	}
	return requestTransport{
		authority:  authority,
		clientHost: clientIP.Unmap().String(),
		secure:     true,
	}, true
}

func singleForwardedValue(header http.Header, name string) (string, bool) {
	values := header.Values(name)
	if len(values) != 1 {
		return "", false
	}
	value := strings.TrimSpace(values[0])
	return value, value != "" && !strings.Contains(value, ",")
}

func requestTransportFrom(request *http.Request) requestTransport {
	if request != nil {
		if transport, ok := request.Context().Value(transportContextKey{}).(requestTransport); ok {
			return transport
		}
		directHost, _ := splitRemoteAddress(request.RemoteAddr)
		return requestTransport{
			authority:  request.Host,
			clientHost: directHost,
			secure:     request.TLS != nil,
		}
	}
	return requestTransport{}
}

func splitRemoteAddress(remoteAddress string) (string, bool) {
	host, _, err := net.SplitHostPort(remoteAddress)
	if err != nil || host == "" {
		return "unknown", false
	}
	return host, true
}

func prefixesContain(prefixes []netip.Prefix, address netip.Addr) bool {
	for _, prefix := range prefixes {
		if prefix.Contains(address) {
			return true
		}
	}
	return false
}

func clearForwardingHeaders(header http.Header) {
	header.Del("Forwarded")
	header.Del(forwardedForHeader)
	header.Del(forwardedHostHeader)
	header.Del(forwardedProtoHeader)
}

func fallbackHandler(next http.Handler) http.Handler {
	if next == nil {
		return http.NotFoundHandler()
	}
	return next
}
