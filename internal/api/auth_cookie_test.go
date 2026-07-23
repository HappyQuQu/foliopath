package api

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestSessionCookieUsesCanonicalSecurityAttributes(t *testing.T) {
	now := time.Unix(1700000000, 0).UTC()
	expiresAt := now.Add(7 * 24 * time.Hour)

	cookie, err := NewSessionCookie("opaque-session-token", now, expiresAt, true)
	if err != nil {
		t.Fatalf("NewSessionCookie() error = %v", err)
	}
	if cookie.Name != SessionCookieName ||
		cookie.Value != "opaque-session-token" ||
		cookie.Path != "/" ||
		!cookie.Expires.Equal(expiresAt) ||
		cookie.MaxAge != 7*24*60*60 ||
		!cookie.HttpOnly ||
		!cookie.Secure ||
		cookie.SameSite != http.SameSiteStrictMode {
		t.Fatalf("session cookie = %#v", cookie)
	}
	header := cookie.String()
	for _, required := range []string{
		"foliopath_session=opaque-session-token",
		"Path=/",
		"Max-Age=604800",
		"HttpOnly",
		"Secure",
		"SameSite=Strict",
	} {
		if !strings.Contains(header, required) {
			t.Errorf("cookie header %q is missing %q", header, required)
		}
	}
	if strings.Contains(header, "Domain=") {
		t.Fatalf("host-only cookie unexpectedly has Domain: %q", header)
	}
}

func TestSessionCookieSecureFlagFollowsValidatedTransport(t *testing.T) {
	now := time.Unix(1700000000, 0).UTC()
	cookie, err := NewSessionCookie("opaque-session-token", now, now.Add(time.Hour), false)
	if err != nil {
		t.Fatalf("NewSessionCookie() error = %v", err)
	}
	if cookie.Secure || strings.Contains(cookie.String(), "Secure") {
		t.Fatalf("loopback HTTP cookie unexpectedly uses Secure: %q", cookie.String())
	}
}

func TestExpiredSessionCookieClearsOnlyCanonicalCookie(t *testing.T) {
	cookie := ExpiredSessionCookie(true)
	if cookie.Name != SessionCookieName ||
		cookie.Value != "" ||
		cookie.Path != "/" ||
		cookie.MaxAge >= 0 ||
		!cookie.HttpOnly ||
		!cookie.Secure ||
		cookie.SameSite != http.SameSiteStrictMode {
		t.Fatalf("expired cookie = %#v", cookie)
	}
	header := cookie.String()
	for _, required := range []string{
		"foliopath_session=",
		"Path=/",
		"Max-Age=0",
		"HttpOnly",
		"Secure",
		"SameSite=Strict",
	} {
		if !strings.Contains(header, required) {
			t.Errorf("expired cookie header %q is missing %q", header, required)
		}
	}
}

func TestSessionCookieRejectsInvalidIssuance(t *testing.T) {
	now := time.Unix(1700000000, 0).UTC()
	if _, err := NewSessionCookie("", now, now.Add(time.Hour), true); err == nil {
		t.Fatal("empty cookie token unexpectedly accepted")
	}
	if _, err := NewSessionCookie("token", now, now, true); err == nil {
		t.Fatal("non-future cookie expiry unexpectedly accepted")
	}
}
