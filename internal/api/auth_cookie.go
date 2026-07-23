package api

import (
	"errors"
	"math"
	"net/http"
	"time"
)

const SessionCookieName = "foliopath_session"

func NewSessionCookie(
	token string,
	now time.Time,
	expiresAt time.Time,
	secure bool,
) (http.Cookie, error) {
	remaining := expiresAt.Sub(now)
	if token == "" {
		return http.Cookie{}, errors.New("session cookie token is empty")
	}
	if remaining <= 0 {
		return http.Cookie{}, errors.New("session cookie expiry is not in the future")
	}
	maxAge := int(math.Ceil(remaining.Seconds()))
	return http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt.UTC(),
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
	}, nil
}

func ExpiredSessionCookie(secure bool) http.Cookie {
	return http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(1, 0).UTC(),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
	}
}
