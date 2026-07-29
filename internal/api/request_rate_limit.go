package api

import (
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const (
	requestRateWindow = time.Minute
	maxRequestBuckets = 4096
)

// requestRatePolicy keeps endpoint limits in one transport-level owner.
type requestRatePolicy struct {
	operation string
	limit     int
}

type requestRateBucket struct {
	start time.Time
	used  int
}

type requestRateLimiter struct {
	mu      sync.Mutex
	now     func() time.Time
	buckets map[string]requestRateBucket
}

func newRequestRateLimiter(now func() time.Time) *requestRateLimiter {
	if now == nil {
		now = time.Now
	}
	return &requestRateLimiter{
		now:     now,
		buckets: make(map[string]requestRateBucket),
	}
}

func limitRequests(
	next http.Handler,
	limiter *requestRateLimiter,
) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		policy, ok := requestRatePolicyFor(request.Method, request.URL.Path)
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
				"Too many requests were received.",
			)
			return
		}
		next.ServeHTTP(writer, request)
	})
}

func requestRatePolicyFor(method, path string) (requestRatePolicy, bool) {
	switch {
	case method == http.MethodPost && path == "/api/v1/auth/setup":
		return requestRatePolicy{operation: "setup", limit: 10}, true
	case method == http.MethodPost && path == "/api/v1/auth/login":
		return requestRatePolicy{operation: "login", limit: 10}, true
	case method == http.MethodGet && path == "/api/v1/auth/status":
		return requestRatePolicy{operation: "status", limit: 120}, true
	case method == http.MethodGet && path == "/api/v1/auth/session":
		return requestRatePolicy{operation: "session", limit: 120}, true
	case method == http.MethodPost && path == "/api/v1/auth/logout":
		return requestRatePolicy{operation: "logout", limit: 60}, true
	case method == http.MethodGet && path == "/api/v1/catalog/state":
		return requestRatePolicy{operation: "catalog_state", limit: 120}, true
	default:
		return requestRatePolicy{}, false
	}
}

func (limiter *requestRateLimiter) allow(key string, limit int) (string, bool) {
	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	now := limiter.now().UTC()
	bucket, exists := limiter.buckets[key]
	if exists && now.Sub(bucket.start) >= requestRateWindow {
		delete(limiter.buckets, key)
		exists = false
	}
	if !exists {
		if len(limiter.buckets) >= maxRequestBuckets {
			limiter.deleteExpired(now)
		}
		if len(limiter.buckets) >= maxRequestBuckets {
			return "1", false
		}
		bucket = requestRateBucket{start: now}
	}
	if bucket.used >= limit {
		remaining := requestRateWindow - now.Sub(bucket.start)
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

func (limiter *requestRateLimiter) deleteExpired(now time.Time) {
	for key, bucket := range limiter.buckets {
		if now.Sub(bucket.start) >= requestRateWindow {
			delete(limiter.buckets, key)
		}
	}
}
