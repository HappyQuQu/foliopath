package api

import (
	"context"
	"crypto/subtle"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/HappyQuQu/foliopath/internal/auth"
)

type authenticatedRequestContextKey struct{}

type authenticatedRequest struct {
	current     auth.CurrentSession
	cookieToken string
}

func requireAPIAuthentication(
	next http.Handler,
	service AuthenticationService,
) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if (request.URL.Path != "/api/v1" &&
			!strings.HasPrefix(request.URL.Path, "/api/v1/")) ||
			anonymousAuthenticationOperation(request.Method, request.URL.Path) {
			next.ServeHTTP(writer, request)
			return
		}

		cookies := request.CookiesNamed(SessionCookieName)
		if len(cookies) != 1 || cookies[0].Value == "" {
			writeAuthenticationError(writer, request, authenticationRequiredError())
			return
		}
		current, err := service.Session(request.Context(), cookies[0].Value)
		if err != nil {
			writeAuthenticationError(writer, request, mapAuthenticationError(err))
			return
		}
		if stateChangingMethod(request.Method) &&
			!constantTimeTokenEqual(request.Header.Get(csrfTokenHeader), current.CSRFToken) {
			writeAuthenticationError(writer, request, authHTTPError{
				status:  http.StatusForbidden,
				code:    codeCSRFInvalid,
				message: "The CSRF token is invalid.",
			})
			return
		}

		ctx := context.WithValue(request.Context(), authenticatedRequestContextKey{}, authenticatedRequest{
			current:     current,
			cookieToken: cookies[0].Value,
		})
		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}

func anonymousAuthenticationOperation(method, path string) bool {
	switch {
	case method == http.MethodGet && path == "/api/v1/auth/status":
		return true
	case method == http.MethodPost && path == "/api/v1/auth/setup":
		return true
	case method == http.MethodPost && path == "/api/v1/auth/login":
		return true
	default:
		return false
	}
}

func stateChangingMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return false
	default:
		return true
	}
}

func constantTimeTokenEqual(provided, expected string) bool {
	if provided == "" || len(provided) != len(expected) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1
}

func authenticatedRequestFromContext(ctx context.Context) (authenticatedRequest, bool) {
	session, ok := ctx.Value(authenticatedRequestContextKey{}).(authenticatedRequest)
	return session, ok && session.cookieToken != ""
}

func requestHasSameOrigin(request *http.Request) bool {
	origins := request.Header.Values("Origin")
	if len(origins) != 1 {
		return false
	}
	origin, err := url.Parse(origins[0])
	if err != nil ||
		origin.Scheme == "" ||
		origin.Host == "" ||
		origin.User != nil ||
		origin.Opaque != "" ||
		origin.Path != "" ||
		origin.RawQuery != "" ||
		origin.Fragment != "" {
		return false
	}

	transport := requestTransportFrom(request)
	requestScheme := "http"
	if transport.secure {
		requestScheme = "https"
	}
	if !strings.EqualFold(origin.Scheme, requestScheme) {
		return false
	}
	originHost, originPort, ok := canonicalAuthority(origin.Host, requestScheme)
	if !ok {
		return false
	}
	requestHost, requestPort, ok := canonicalAuthority(transport.authority, requestScheme)
	return ok &&
		strings.EqualFold(originHost, requestHost) &&
		originPort == requestPort
}

func canonicalAuthority(authority, scheme string) (string, string, bool) {
	parsed, err := url.Parse("//" + authority)
	if err != nil || parsed.Host != authority || parsed.User != nil {
		return "", "", false
	}
	host := parsed.Hostname()
	if host == "" {
		return "", "", false
	}
	port := parsed.Port()
	if port == "" {
		switch scheme {
		case "http":
			port = "80"
		case "https":
			port = "443"
		default:
			return "", "", false
		}
	} else {
		number, err := strconv.Atoi(port)
		if err != nil || number < 1 || number > 65535 {
			return "", "", false
		}
	}
	return host, port, true
}
