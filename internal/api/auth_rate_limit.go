package api

import (
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const (
	authenticationRateWindow = time.Minute
	maxAuthenticationBuckets = 4096
)

type authenticationRatePolicy struct {
	operation string
	limit     int
}

type authenticationRateBucket struct {
	start time.Time
	used  int
}

type authenticationRateLimiter struct {
	mu      sync.Mutex
	now     func() time.Time
	buckets map[string]authenticationRateBucket
}

func newAuthenticationRateLimiter(now func() time.Time) *authenticationRateLimiter {
	if now == nil {
		now = time.Now
	}
	return &authenticationRateLimiter{
		now:     now,
		buckets: make(map[string]authenticationRateBucket),
	}
}

func limitAuthenticationRequests(
	next http.Handler,
	limiter *authenticationRateLimiter,
) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		policy, ok := authenticationRatePolicyFor(request.Method, request.URL.Path)
		if !ok {
			next.ServeHTTP(writer, request)
			return
		}
		retryAfter, allowed := limiter.allow(
			requestTransportFrom(request).clientHost+"\x00"+policy.operation,
			policy.limit,
		)
		if !allowed {
			writer.Header().Set("Retry-After", retryAfter)
			writePublicError(
				writer,
				request,
				http.StatusTooManyRequests,
				"rate_limited",
				"Too many authentication requests were received.",
			)
			return
		}
		next.ServeHTTP(writer, request)
	})
}

func authenticationRatePolicyFor(method, path string) (authenticationRatePolicy, bool) {
	switch {
	case method == http.MethodPost && path == "/api/v1/auth/setup":
		return authenticationRatePolicy{operation: "setup", limit: 10}, true
	case method == http.MethodPost && path == "/api/v1/auth/login":
		return authenticationRatePolicy{operation: "login", limit: 10}, true
	case method == http.MethodGet && path == "/api/v1/auth/status":
		return authenticationRatePolicy{operation: "status", limit: 120}, true
	case method == http.MethodGet && path == "/api/v1/auth/session":
		return authenticationRatePolicy{operation: "session", limit: 120}, true
	case method == http.MethodPost && path == "/api/v1/auth/logout":
		return authenticationRatePolicy{operation: "logout", limit: 60}, true
	default:
		return authenticationRatePolicy{}, false
	}
}

func (limiter *authenticationRateLimiter) allow(key string, limit int) (string, bool) {
	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	now := limiter.now().UTC()
	bucket, exists := limiter.buckets[key]
	if exists && now.Sub(bucket.start) >= authenticationRateWindow {
		delete(limiter.buckets, key)
		exists = false
	}
	if !exists {
		if len(limiter.buckets) >= maxAuthenticationBuckets {
			limiter.deleteExpired(now)
		}
		if len(limiter.buckets) >= maxAuthenticationBuckets {
			return "1", false
		}
		bucket = authenticationRateBucket{start: now}
	}
	if bucket.used >= limit {
		remaining := authenticationRateWindow - now.Sub(bucket.start)
		seconds := int(math.Ceil(remaining.Seconds()))
		if seconds < 1 {
			seconds = 1
		}
		return strconv.Itoa(seconds), false
	}
	bucket.used++
	limiter.buckets[key] = bucket
	return "", true
}

func (limiter *authenticationRateLimiter) deleteExpired(now time.Time) {
	for key, bucket := range limiter.buckets {
		if now.Sub(bucket.start) >= authenticationRateWindow {
			delete(limiter.buckets, key)
		}
	}
}
