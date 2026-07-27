package api

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/auth"
)

const (
	testCookieToken = "test-cookie-token"
	testCSRFToken   = "test-csrf-token-which-is-long-enough"
)

type authenticationStub struct {
	setupStateValue auth.SetupState
	setupStateErr   error
	initialize      func(context.Context, auth.InitializeParams) (auth.EstablishedSession, error)
	login           func(context.Context, auth.LoginParams) (auth.EstablishedSession, error)
	session         auth.CurrentSession
	sessionErr      error
	sessionCalls    int
	logout          func(context.Context, string) error
}

func (stub *authenticationStub) SetupState(context.Context) (auth.SetupState, error) {
	return stub.setupStateValue, stub.setupStateErr
}

func (stub *authenticationStub) Initialize(
	ctx context.Context,
	params auth.InitializeParams,
) (auth.EstablishedSession, error) {
	if stub.initialize == nil {
		return auth.EstablishedSession{}, errors.New("unexpected initialize call")
	}
	return stub.initialize(ctx, params)
}

func (stub *authenticationStub) Login(
	ctx context.Context,
	params auth.LoginParams,
) (auth.EstablishedSession, error) {
	if stub.login == nil {
		return auth.EstablishedSession{}, errors.New("unexpected login call")
	}
	return stub.login(ctx, params)
}

func (stub *authenticationStub) Session(
	context.Context,
	string,
) (auth.CurrentSession, error) {
	stub.sessionCalls++
	return stub.session, stub.sessionErr
}

func (stub *authenticationStub) Logout(ctx context.Context, cookie string) error {
	if stub.logout == nil {
		return errors.New("unexpected logout call")
	}
	return stub.logout(ctx, cookie)
}

func TestAuthenticationStatusIsAnonymousMinimalAndNotCacheable(t *testing.T) {
	service := &authenticationStub{setupStateValue: auth.SetupRequired}
	handler := authenticationTestRoutes(t, service)
	response := requestAuthentication(
		handler,
		http.MethodGet,
		"/api/v1/auth/status",
		"",
		nil,
	)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	assertNoStore(t, response)
	assertJSONEquals(t, response, map[string]any{
		"setupRequired":          true,
		"authenticationRequired": true,
	})
	if service.sessionCalls != 0 {
		t.Fatalf("anonymous status performed %d session lookups", service.sessionCalls)
	}
}

func TestSetupRequiresSameOriginBeforeReadingCredentials(t *testing.T) {
	initializeCalled := false
	service := &authenticationStub{
		initialize: func(
			context.Context,
			auth.InitializeParams,
		) (auth.EstablishedSession, error) {
			initializeCalled = true
			return auth.EstablishedSession{}, nil
		},
	}
	handler := authenticationTestRoutes(t, service)
	for _, origin := range []string{
		"",
		"null",
		"https://foliopath.test",
		"http://attacker.test",
		"http://foliopath.test/path",
		"http://user@foliopath.test",
	} {
		t.Run(origin, func(t *testing.T) {
			response := requestAuthentication(
				handler,
				http.MethodPost,
				"/api/v1/auth/setup",
				`{"username":"admin","displayName":"Admin","password":"secret password"}`,
				map[string]string{"Origin": origin},
			)
			if response.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusForbidden)
			}
			assertSafeErrorResponse(t, response, codeOriginInvalid)
			assertNoStore(t, response)
		})
	}
	if initializeCalled {
		t.Fatal("invalid-origin setup reached the authentication service")
	}
}

func TestSetupCreatesSessionCookieAndContractResponse(t *testing.T) {
	expiresAt := time.Now().UTC().Add(time.Hour).Truncate(time.Millisecond)
	service := &authenticationStub{
		initialize: func(
			_ context.Context,
			params auth.InitializeParams,
		) (auth.EstablishedSession, error) {
			if params.Username != "Admin" ||
				params.DisplayName != "Administrator" ||
				params.Password != "correct horse battery staple" {
				t.Fatalf("initialize params = %#v", params)
			}
			return establishedAuthenticationSession(expiresAt), nil
		},
	}
	handler := authenticationTestRoutes(t, service)
	response := requestAuthentication(
		handler,
		http.MethodPost,
		"/api/v1/auth/setup",
		`{"username":"Admin","displayName":"Administrator","password":"correct horse battery staple"}`,
		map[string]string{
			"Origin":       "http://foliopath.test",
			"Content-Type": "application/json; charset=utf-8",
		},
	)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusCreated, response.Body)
	}
	assertNoStore(t, response)
	cookies := response.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies = %#v, want one session cookie", cookies)
	}
	cookie := cookies[0]
	if cookie.Name != SessionCookieName ||
		cookie.Value != testCookieToken ||
		!cookie.HttpOnly ||
		cookie.Secure ||
		cookie.SameSite != http.SameSiteStrictMode ||
		cookie.Path != "/" {
		t.Fatalf("session cookie = %#v", cookie)
	}

	var payload sessionResponse
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode session response: %v", err)
	}
	if payload.Administrator.ID != "usr_7" ||
		payload.Administrator.Username != "Admin" ||
		payload.Administrator.DisplayName != "Administrator" ||
		payload.CSRFToken != testCSRFToken ||
		payload.ExpiresAt != expiresAt.Format(time.RFC3339Nano) {
		t.Fatalf("session response = %#v", payload)
	}
}

func TestHTTPSLoginSetsSecureCookie(t *testing.T) {
	expiresAt := time.Now().UTC().Add(time.Hour)
	service := &authenticationStub{
		login: func(
			context.Context,
			auth.LoginParams,
		) (auth.EstablishedSession, error) {
			return establishedAuthenticationSession(expiresAt), nil
		},
	}
	handler := authenticationTestRoutes(t, service)
	request := httptest.NewRequest(
		http.MethodPost,
		"https://foliopath.test/api/v1/auth/login",
		strings.NewReader(`{"username":"Admin","password":"password"}`),
	)
	request.Header.Set("Origin", "https://foliopath.test")
	request.Header.Set("Content-Type", "application/json")
	request.TLS = &tls.ConnectionState{}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body)
	}
	cookies := response.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].Secure {
		t.Fatalf("HTTPS session cookies = %#v, want Secure", cookies)
	}
}

func TestAuthenticationRequestRejectsInvalidJSONWithoutCallingService(t *testing.T) {
	service := &authenticationStub{
		login: func(
			context.Context,
			auth.LoginParams,
		) (auth.EstablishedSession, error) {
			t.Fatal("invalid request reached login")
			return auth.EstablishedSession{}, nil
		},
	}
	handler := authenticationTestRoutes(t, service)
	tests := []struct {
		name        string
		contentType string
		body        string
	}{
		{name: "missing content type", body: `{}`},
		{name: "wrong content type", contentType: "text/plain", body: `{}`},
		{name: "unknown field", contentType: "application/json", body: `{"username":"a","password":"b","extra":true}`},
		{name: "multiple values", contentType: "application/json", body: `{} {}`},
		{name: "oversize", contentType: "application/json", body: `{"username":"` + strings.Repeat("a", 5000) + `"}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := requestAuthentication(
				handler,
				http.MethodPost,
				"/api/v1/auth/login",
				test.body,
				map[string]string{
					"Origin":       "http://foliopath.test",
					"Content-Type": test.contentType,
				},
			)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
			}
			assertSafeErrorResponse(t, response, codeInvalidRequest)
		})
	}
}

func TestAuthenticationDomainErrorsHaveStableSafeMapping(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{"invalid username", auth.ErrInvalidUsername, http.StatusUnprocessableEntity, codeValidationFailed},
		{"invalid display name", auth.ErrInvalidDisplayName, http.StatusUnprocessableEntity, codeValidationFailed},
		{"invalid password", auth.ErrInvalidPassword, http.StatusUnprocessableEntity, codeValidationFailed},
		{"setup closed", auth.ErrSetupClosed, http.StatusConflict, codeSetupClosed},
		{"setup in progress", auth.ErrSetupInProgress, http.StatusConflict, codeSetupInProgress},
		{"invalid credentials", auth.ErrInvalidCredentials, http.StatusUnauthorized, codeInvalidCredentials},
		{"repository detail", errors.New("SELECT password at /app/data"), http.StatusInternalServerError, codeInternalError},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mapped := mapAuthenticationError(test.err)
			if mapped.status != test.wantStatus || mapped.code != test.wantCode {
				t.Fatalf("mapped error = %#v, want status %d code %q", mapped, test.wantStatus, test.wantCode)
			}
			for _, forbidden := range []string{"SELECT", "/app/data"} {
				if strings.Contains(mapped.message, forbidden) {
					t.Fatalf("public error leaked %q: %#v", forbidden, mapped)
				}
			}
		})
	}
}

func TestAuthenticationFailuresMaskServiceDetailsInHTTPResponses(t *testing.T) {
	const sensitiveDetail = "SELECT password_hash FROM /app/data/foliopath.db secret-token"
	tests := []struct {
		name    string
		method  string
		path    string
		body    string
		headers map[string]string
		service *authenticationStub
	}{
		{
			name:   "status",
			method: http.MethodGet,
			path:   "/api/v1/auth/status",
			service: &authenticationStub{
				setupStateErr: errors.New(sensitiveDetail),
			},
		},
		{
			name:   "setup",
			method: http.MethodPost,
			path:   "/api/v1/auth/setup",
			body:   `{"username":"admin","displayName":"Admin","password":"do not expose"}`,
			headers: map[string]string{
				"Origin":       "http://foliopath.test",
				"Content-Type": "application/json",
			},
			service: &authenticationStub{
				initialize: func(
					context.Context,
					auth.InitializeParams,
				) (auth.EstablishedSession, error) {
					return auth.EstablishedSession{}, errors.New(sensitiveDetail)
				},
			},
		},
		{
			name:   "login",
			method: http.MethodPost,
			path:   "/api/v1/auth/login",
			body:   `{"username":"admin","password":"do not expose"}`,
			headers: map[string]string{
				"Origin":       "http://foliopath.test",
				"Content-Type": "application/json",
			},
			service: &authenticationStub{
				login: func(
					context.Context,
					auth.LoginParams,
				) (auth.EstablishedSession, error) {
					return auth.EstablishedSession{}, errors.New(sensitiveDetail)
				},
			},
		},
		{
			name:   "session",
			method: http.MethodGet,
			path:   "/api/v1/auth/session",
			headers: map[string]string{
				"Cookie": SessionCookieName + "=" + testCookieToken,
			},
			service: &authenticationStub{sessionErr: errors.New(sensitiveDetail)},
		},
		{
			name:   "logout",
			method: http.MethodPost,
			path:   "/api/v1/auth/logout",
			headers: map[string]string{
				"Cookie":        SessionCookieName + "=" + testCookieToken,
				csrfTokenHeader: testCSRFToken,
			},
			service: func() *authenticationStub {
				service := acceptingAuthentication()
				service.logout = func(context.Context, string) error {
					return errors.New(sensitiveDetail)
				}
				return service
			}(),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := requestAuthentication(
				authenticationTestRoutes(t, test.service),
				test.method,
				test.path,
				test.body,
				test.headers,
			)
			if response.Code != http.StatusInternalServerError {
				t.Fatalf(
					"status = %d, want %d; body = %s",
					response.Code,
					http.StatusInternalServerError,
					response.Body,
				)
			}
			assertSafeErrorResponse(t, response, codeInternalError)
			assertNoStore(t, response)
			for _, forbidden := range []string{
				"SELECT",
				"/app/data",
				"secret-token",
				"do not expose",
			} {
				if strings.Contains(response.Body.String(), forbidden) {
					t.Fatalf("HTTP error leaked %q: %s", forbidden, response.Body)
				}
			}
		})
	}
}

func TestProtectedAPIDefaultsToSessionAndCSRFEnforcement(t *testing.T) {
	service := acceptingAuthentication()
	handler := authenticationTestRoutes(t, service)

	t.Run("missing cookie", func(t *testing.T) {
		response := requestAuthentication(handler, http.MethodGet, "/api/v1/status", "", nil)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
		}
		assertSafeErrorResponse(t, response, codeAuthenticationRequired)
	})

	t.Run("duplicate cookie", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "http://foliopath.test/api/v1/status", nil)
		request.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "first"})
		request.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "second"})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
		}
		assertSafeErrorResponse(t, response, codeAuthenticationRequired)
	})

	t.Run("unknown business route authenticates before not found", func(t *testing.T) {
		response := requestAuthentication(
			handler,
			http.MethodGet,
			"/api/v1/not-implemented",
			"",
			map[string]string{"Cookie": SessionCookieName + "=" + testCookieToken},
		)
		if response.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
		}
	})

	for _, test := range []struct {
		name  string
		token string
	}{
		{name: "missing CSRF"},
		{name: "wrong CSRF", token: "wrong-token"},
	} {
		t.Run(test.name, func(t *testing.T) {
			response := requestAuthentication(
				handler,
				http.MethodPost,
				"/api/v1/not-implemented",
				"",
				map[string]string{
					"Cookie":        SessionCookieName + "=" + testCookieToken,
					csrfTokenHeader: test.token,
				},
			)
			if response.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusForbidden)
			}
			assertSafeErrorResponse(t, response, codeCSRFInvalid)
		})
	}

	t.Run("valid CSRF reaches route", func(t *testing.T) {
		response := requestAuthentication(
			handler,
			http.MethodPost,
			"/api/v1/not-implemented",
			"",
			map[string]string{
				"Cookie":        SessionCookieName + "=" + testCookieToken,
				csrfTokenHeader: testCSRFToken,
			},
		)
		if response.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
		}
	})
}

func TestCSRFRejectionNeverInvokesLogout(t *testing.T) {
	var logoutCalls int
	service := acceptingAuthentication()
	service.logout = func(context.Context, string) error {
		logoutCalls++
		return nil
	}
	handler := authenticationTestRoutes(t, service)

	for _, csrfToken := range []string{"", "wrong-token"} {
		response := requestAuthentication(
			handler,
			http.MethodPost,
			"/api/v1/auth/logout",
			"",
			map[string]string{
				"Cookie":        SessionCookieName + "=" + testCookieToken,
				csrfTokenHeader: csrfToken,
			},
		)
		if response.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want %d", response.Code, http.StatusForbidden)
		}
		assertSafeErrorResponse(t, response, codeCSRFInvalid)
	}
	if logoutCalls != 0 {
		t.Fatalf("CSRF failures invoked logout %d times", logoutCalls)
	}
}

func TestSessionAndLogoutUseAuthenticatedContext(t *testing.T) {
	loggedOutCookie := ""
	service := acceptingAuthentication()
	service.logout = func(_ context.Context, cookie string) error {
		loggedOutCookie = cookie
		return nil
	}
	handler := authenticationTestRoutes(t, service)
	headers := map[string]string{"Cookie": SessionCookieName + "=" + testCookieToken}

	sessionResponseRecorder := requestAuthentication(
		handler,
		http.MethodGet,
		"/api/v1/auth/session",
		"",
		headers,
	)
	if sessionResponseRecorder.Code != http.StatusOK {
		t.Fatalf("session status = %d, want %d", sessionResponseRecorder.Code, http.StatusOK)
	}
	assertNoStore(t, sessionResponseRecorder)

	headers[csrfTokenHeader] = testCSRFToken
	logoutResponse := requestAuthentication(
		handler,
		http.MethodPost,
		"/api/v1/auth/logout",
		"",
		headers,
	)
	if logoutResponse.Code != http.StatusNoContent {
		t.Fatalf("logout status = %d, want %d", logoutResponse.Code, http.StatusNoContent)
	}
	if loggedOutCookie != testCookieToken {
		t.Fatalf("logout cookie = %q, want middleware cookie", loggedOutCookie)
	}
	assertNoStore(t, logoutResponse)
	cookies := logoutResponse.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != SessionCookieName || cookies[0].MaxAge != -1 {
		t.Fatalf("expired cookies = %#v", cookies)
	}
}

func TestSessionFailuresRemainDistinctAndDoNotReachRoute(t *testing.T) {
	for _, test := range []struct {
		name     string
		err      error
		wantCode string
	}{
		{"unknown", auth.ErrAuthenticationRequired, codeAuthenticationRequired},
		{"expired", auth.ErrSessionExpired, codeSessionExpired},
		{"internal", errors.New("database token secret"), codeInternalError},
	} {
		t.Run(test.name, func(t *testing.T) {
			service := &authenticationStub{sessionErr: test.err}
			handler := authenticationTestRoutes(t, service)
			response := requestAuthentication(
				handler,
				http.MethodGet,
				"/api/v1/status",
				"",
				map[string]string{"Cookie": SessionCookieName + "=" + testCookieToken},
			)
			assertSafeErrorResponse(t, response, test.wantCode)
			if strings.Contains(response.Body.String(), "database") ||
				strings.Contains(response.Body.String(), "secret") {
				t.Fatalf("session failure leaked detail: %s", response.Body)
			}
		})
	}
}

func TestSameOriginCanonicalizesDefaultPortsAndRejectsForwardedClaims(t *testing.T) {
	tests := []struct {
		name   string
		target string
		host   string
		origin string
		tls    bool
		want   bool
	}{
		{"http default", "http://foliopath.test/", "foliopath.test", "http://FOLIOPATH.test:80", false, true},
		{"https default", "https://foliopath.test/", "foliopath.test:443", "https://foliopath.test", true, true},
		{"nondefault equal", "http://foliopath.test:8080/", "foliopath.test:8080", "http://foliopath.test:8080", false, true},
		{"different port", "http://foliopath.test:8080/", "foliopath.test:8080", "http://foliopath.test", false, false},
		{"different scheme", "http://foliopath.test/", "foliopath.test", "https://foliopath.test", false, false},
		{"invalid port", "http://foliopath.test/", "foliopath.test", "http://foliopath.test:service", false, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, test.target, nil)
			request.Host = test.host
			request.Header.Set("Origin", test.origin)
			request.Header.Set("Forwarded", "proto=https;host=attacker.test")
			request.Header.Set("X-Forwarded-Proto", "https")
			if test.tls {
				request.TLS = &tls.ConnectionState{}
			}
			if got := requestHasSameOrigin(request); got != test.want {
				t.Fatalf("requestHasSameOrigin() = %t, want %t", got, test.want)
			}
		})
	}
}

func authenticationTestRoutes(
	t *testing.T,
	service AuthenticationService,
) http.Handler {
	t.Helper()
	routes, err := NewRoutes(RouteDependencies{
		Readiness:      func() Readiness { return Readiness{Ready: true} },
		Authentication: service,
		SystemStatus: func(context.Context) (SystemStatus, error) {
			return SystemStatus{
				Version:          "test",
				APIVersion:       "v1",
				RuntimeState:     "ready",
				Initialized:      true,
				ReadOnlyMedia:    true,
				SupportedLocales: []string{"zh-CN", "en"},
				SupportedMedia:   SupportedMedia{},
			}, nil
		},
	})
	if err != nil {
		t.Fatalf("NewRoutes() error = %v", err)
	}
	return NewHandler(routes, discardLogger())
}

func requestAuthentication(
	handler http.Handler,
	method string,
	path string,
	body string,
	headers map[string]string,
) *httptest.ResponseRecorder {
	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}
	request := httptest.NewRequest(method, "http://foliopath.test"+path, reader)
	for name, value := range headers {
		if value != "" {
			request.Header.Set(name, value)
		}
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func establishedAuthenticationSession(expiresAt time.Time) auth.EstablishedSession {
	return auth.EstablishedSession{
		Administrator: auth.Administrator{
			ID:          7,
			Username:    "Admin",
			DisplayName: "Administrator",
		},
		CookieToken: testCookieToken,
		CSRFToken:   testCSRFToken,
		ExpiresAtMS: expiresAt.UnixMilli(),
	}
}

func assertNoStore(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()
	if got := response.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", got)
	}
}
