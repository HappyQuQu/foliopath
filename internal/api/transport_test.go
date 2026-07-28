package api

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/auth"
)

func TestUntrustedPeerCannotInfluenceTransport(t *testing.T) {
	var observed requestTransport
	handler := withTrustedTransport(http.HandlerFunc(func(
		_ http.ResponseWriter,
		request *http.Request,
	) {
		observed = requestTransportFrom(request)
		if request.Header.Get(forwardedProtoHeader) != "" ||
			request.Header.Get(forwardedHostHeader) != "" ||
			request.Header.Get(forwardedForHeader) != "" {
			t.Fatal("untrusted forwarding headers reached the application")
		}
	}), TransportConfig{
		TrustedProxyPrefixes: []netip.Prefix{
			netip.MustParsePrefix("192.0.2.0/24"),
		},
	})
	request := httptest.NewRequest(http.MethodGet, "http://foliopath.local/", nil)
	request.RemoteAddr = "198.51.100.8:4321"
	request.Header.Set(forwardedProtoHeader, "https")
	request.Header.Set(forwardedHostHeader, "photos.example")
	request.Header.Set(forwardedForHeader, "203.0.113.9")

	handler.ServeHTTP(httptest.NewRecorder(), request)

	if observed.secure ||
		observed.authority != "foliopath.local" ||
		observed.clientHost != "198.51.100.8" {
		t.Fatalf("observed transport = %#v", observed)
	}
}

func TestTrustedProxyEstablishesHTTPSAuthorityAndClient(t *testing.T) {
	var observed requestTransport
	handler := withTrustedTransport(http.HandlerFunc(func(
		_ http.ResponseWriter,
		request *http.Request,
	) {
		observed = requestTransportFrom(request)
	}), TransportConfig{
		TrustedProxyPrefixes: []netip.Prefix{
			netip.MustParsePrefix("192.0.2.0/24"),
		},
		RequireTrustedProxy: true,
	})
	request := httptest.NewRequest(http.MethodGet, "http://internal:8080/", nil)
	request.RemoteAddr = "192.0.2.10:4321"
	request.Header.Set(forwardedProtoHeader, "https")
	request.Header.Set(forwardedHostHeader, "photos.example:443")
	request.Header.Set(forwardedForHeader, "203.0.113.9")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	if !observed.secure ||
		observed.authority != "photos.example:443" ||
		observed.clientHost != "203.0.113.9" {
		t.Fatalf("observed transport = %#v", observed)
	}
}

func TestTrustedProxyHeadersFailClosed(t *testing.T) {
	tests := []struct {
		name    string
		headers map[string][]string
	}{
		{
			name: "missing client",
			headers: map[string][]string{
				forwardedProtoHeader: {"https"},
				forwardedHostHeader:  {"photos.example"},
			},
		},
		{
			name: "plain HTTP",
			headers: map[string][]string{
				forwardedProtoHeader: {"http"},
				forwardedHostHeader:  {"photos.example"},
				forwardedForHeader:   {"203.0.113.9"},
			},
		},
		{
			name: "forwarded chain",
			headers: map[string][]string{
				forwardedProtoHeader: {"https"},
				forwardedHostHeader:  {"photos.example"},
				forwardedForHeader:   {"203.0.113.9, 192.0.2.8"},
			},
		},
		{
			name: "ambiguous standard header",
			headers: map[string][]string{
				"Forwarded":          {"for=203.0.113.9;proto=https"},
				forwardedProtoHeader: {"https"},
				forwardedHostHeader:  {"photos.example"},
				forwardedForHeader:   {"203.0.113.9"},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			called := false
			handler := withSecurityHeaders(withTrustedTransport(http.HandlerFunc(func(
				http.ResponseWriter,
				*http.Request,
			) {
				called = true
			}), TransportConfig{
				TrustedProxyPrefixes: []netip.Prefix{
					netip.MustParsePrefix("192.0.2.0/24"),
				},
			}))
			request := httptest.NewRequest(http.MethodGet, "http://internal:8080/", nil)
			request.RemoteAddr = "192.0.2.10:4321"
			for name, values := range test.headers {
				for _, value := range values {
					request.Header.Add(name, value)
				}
			}
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			if called {
				t.Fatal("invalid proxy request reached application")
			}
			if response.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", response.Code)
			}
			assertSafeErrorResponse(t, response, "proxy_headers_invalid")
			if response.Header().Get("Content-Security-Policy") == "" ||
				response.Header().Get("X-Frame-Options") != "DENY" {
				t.Fatal("proxy rejection omitted baseline security headers")
			}
		})
	}
}

func TestRequiredProxyRejectsDirectRemoteButKeepsLoopbackHealthPossible(t *testing.T) {
	handler := withTrustedTransport(http.HandlerFunc(func(
		writer http.ResponseWriter,
		_ *http.Request,
	) {
		writer.WriteHeader(http.StatusNoContent)
	}), TransportConfig{RequireTrustedProxy: true})

	remote := httptest.NewRequest(http.MethodGet, "http://internal:8080/", nil)
	remote.RemoteAddr = "198.51.100.8:4321"
	remoteResponse := httptest.NewRecorder()
	handler.ServeHTTP(remoteResponse, remote)
	if remoteResponse.Code != http.StatusBadRequest {
		t.Fatalf("remote status = %d, want 400", remoteResponse.Code)
	}

	loopback := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8080/health/ready", nil)
	loopback.RemoteAddr = "127.0.0.1:4321"
	loopbackResponse := httptest.NewRecorder()
	handler.ServeHTTP(loopbackResponse, loopback)
	if loopbackResponse.Code != http.StatusNoContent {
		t.Fatalf("loopback status = %d, want 204", loopbackResponse.Code)
	}
}

func TestDirectRemoteHTTPUsesPeerAndIgnoresForwardingHeaders(t *testing.T) {
	var observed requestTransport
	handler := withTrustedTransport(http.HandlerFunc(func(
		writer http.ResponseWriter,
		request *http.Request,
	) {
		observed = requestTransportFrom(request)
		writer.WriteHeader(http.StatusNoContent)
	}), TransportConfig{})
	request := httptest.NewRequest(http.MethodGet, "http://192.168.2.222:8080/", nil)
	request.RemoteAddr = "192.168.2.50:4321"
	request.Header.Set(forwardedProtoHeader, "https")
	request.Header.Set(forwardedHostHeader, "attacker.example")
	request.Header.Set(forwardedForHeader, "203.0.113.9")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", response.Code)
	}
	if observed.secure ||
		observed.authority != "192.168.2.222:8080" ||
		observed.clientHost != "192.168.2.50" {
		t.Fatalf("observed transport = %#v", observed)
	}
}

func TestDirectTLSTransportRemainsSecure(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "https://foliopath.test/", nil)
	request.TLS = &tls.ConnectionState{}
	transport := requestTransportFrom(request)
	if !transport.secure || transport.authority != "foliopath.test" {
		t.Fatalf("transport = %#v", transport)
	}
}

func TestTrustedProxyDrivesOriginSecureCookieRateIdentityAndHeaders(t *testing.T) {
	service := &authenticationStub{
		login: func(
			_ context.Context,
			_ auth.LoginParams,
		) (auth.EstablishedSession, error) {
			return establishedAuthenticationSession(time.Now().Add(time.Hour)), nil
		},
	}
	handler := NewHandlerWithTransport(
		authenticationRoutes(t, service),
		discardLogger(),
		TransportConfig{
			TrustedProxyPrefixes: []netip.Prefix{
				netip.MustParsePrefix("192.0.2.0/24"),
			},
			RequireTrustedProxy: true,
		},
	)

	login := func(client string) *httptest.ResponseRecorder {
		request := httptest.NewRequest(
			http.MethodPost,
			"http://internal:8080/api/v1/auth/login",
			strings.NewReader(`{"username":"admin","password":"correct-password"}`),
		)
		request.RemoteAddr = "192.0.2.10:4321"
		request.Host = "internal:8080"
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Origin", "https://photos.example")
		request.Header.Set(forwardedProtoHeader, "https")
		request.Header.Set(forwardedHostHeader, "photos.example")
		request.Header.Set(forwardedForHeader, client)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		return response
	}

	for requestNumber := 1; requestNumber <= 10; requestNumber++ {
		response := login("203.0.113.9")
		if response.Code != http.StatusOK {
			t.Fatalf("client A request %d status = %d", requestNumber, response.Code)
		}
		if !strings.Contains(response.Header().Get("Set-Cookie"), "Secure") {
			t.Fatal("trusted HTTPS login did not set a Secure cookie")
		}
		if response.Header().Get("Strict-Transport-Security") == "" ||
			response.Header().Get("Content-Security-Policy") == "" ||
			response.Header().Get("X-Frame-Options") != "DENY" {
			t.Fatal("trusted HTTPS response omitted security headers")
		}
	}
	if response := login("203.0.113.9"); response.Code != http.StatusTooManyRequests {
		t.Fatalf("client A overflow status = %d, want 429", response.Code)
	}
	if response := login("203.0.113.10"); response.Code != http.StatusOK {
		t.Fatalf("client B first request status = %d, want 200", response.Code)
	}
}
