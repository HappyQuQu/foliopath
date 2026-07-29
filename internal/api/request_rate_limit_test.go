package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/auth"
)

func TestRequestRateLimiterEnforcesAndResetsFixedWindow(t *testing.T) {
	now := time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC)
	limiter := newRequestRateLimiter(func() time.Time { return now })
	for requestNumber := 1; requestNumber <= 10; requestNumber++ {
		if retry, allowed := limiter.allow("peer\x00login", 10); !allowed || retry != "" {
			t.Fatalf("request %d = (%q, %t), want allowed", requestNumber, retry, allowed)
		}
	}
	if retry, allowed := limiter.allow("peer\x00login", 10); allowed || retry != "60" {
		t.Fatalf("limited request = (%q, %t), want (60, false)", retry, allowed)
	}
	now = now.Add(time.Minute)
	if retry, allowed := limiter.allow("peer\x00login", 10); !allowed || retry != "" {
		t.Fatalf("reset request = (%q, %t), want allowed", retry, allowed)
	}
}

func TestRequestRateLimiterIsConcurrencySafe(t *testing.T) {
	limiter := newRequestRateLimiter(func() time.Time {
		return time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC)
	})
	var allowed atomic.Int64
	var group sync.WaitGroup
	for range 100 {
		group.Add(1)
		go func() {
			defer group.Done()
			if _, ok := limiter.allow("peer\x00setup", 10); ok {
				allowed.Add(1)
			}
		}()
	}
	group.Wait()
	if got := allowed.Load(); got != 10 {
		t.Fatalf("allowed requests = %d, want 10", got)
	}
}

func TestLoginRateLimitStopsBeforePasswordVerification(t *testing.T) {
	var loginCalls atomic.Int64
	service := &authenticationStub{
		login: func(
			context.Context,
			auth.LoginParams,
		) (auth.EstablishedSession, error) {
			loginCalls.Add(1)
			return auth.EstablishedSession{}, auth.ErrInvalidCredentials
		},
	}
	handler := authenticationTestRoutes(t, service)
	headers := map[string]string{
		"Origin":            "http://foliopath.test",
		"Content-Type":      "application/json",
		"X-Forwarded-For":   "203.0.113.5",
		"X-Forwarded-Proto": "https",
	}
	for requestNumber := 1; requestNumber <= 11; requestNumber++ {
		response := requestAuthentication(
			handler,
			http.MethodPost,
			"/api/v1/auth/login",
			`{"username":"admin","password":"wrong"}`,
			headers,
		)
		if requestNumber <= 10 {
			if response.Code != http.StatusUnauthorized {
				t.Fatalf("request %d status = %d, want 401", requestNumber, response.Code)
			}
			continue
		}
		if response.Code != http.StatusTooManyRequests {
			t.Fatalf("request %d status = %d, want 429", requestNumber, response.Code)
		}
		if response.Header().Get("Retry-After") == "" {
			t.Fatal("rate-limited response has no Retry-After")
		}
		assertSafeErrorResponse(t, response, "rate_limited")
	}
	if got := loginCalls.Load(); got != 10 {
		t.Fatalf("login calls = %d, want 10", got)
	}
}

func TestCatalogStateReadRateLimitStopsBeforeService(t *testing.T) {
	now := time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC)
	var calls atomic.Int64
	mux := http.NewServeMux()
	registerCatalogRoutes(mux, catalogServiceStub{
		contentRevision: func(context.Context) (int64, error) {
			calls.Add(1)
			return 9, nil
		},
	})
	handler := limitRequests(
		mux,
		newRequestRateLimiter(func() time.Time { return now }),
	)
	for requestNumber := 1; requestNumber <= 121; requestNumber++ {
		request := httptest.NewRequest(
			http.MethodGet,
			"/api/v1/catalog/state",
			nil,
		)
		request.RemoteAddr = "192.0.2.44:4321"
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if requestNumber <= 120 {
			if response.Code != http.StatusOK {
				t.Fatalf(
					"catalog state request %d status = %d",
					requestNumber,
					response.Code,
				)
			}
			continue
		}
		if response.Code != http.StatusTooManyRequests ||
			response.Header().Get("Retry-After") != "60" {
			t.Fatalf(
				"limited catalog state = %d headers=%v body=%s",
				response.Code,
				response.Header(),
				response.Body,
			)
		}
		assertSafeErrorResponse(t, response, "rate_limited")
	}
	if calls.Load() != 120 {
		t.Fatalf("catalog state service calls = %d, want 120", calls.Load())
	}
}

func TestRemoteAddressParsingRequiresHostAndPort(t *testing.T) {
	if got, ok := splitRemoteAddress("192.0.2.10:4321"); !ok || got != "192.0.2.10" {
		t.Fatalf("splitRemoteAddress() = (%q, %t)", got, ok)
	}
	if got, ok := splitRemoteAddress("spoofed-client"); ok || got != "unknown" {
		t.Fatalf("malformed splitRemoteAddress() = (%q, %t)", got, ok)
	}
}
