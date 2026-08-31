package api

import (
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	requestRateWindow               = time.Minute
	maxRequestBuckets               = 4096
	semanticSearchRequestsPerMinute = 30
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
	case method == http.MethodGet && path == "/api/v1/account":
		return requestRatePolicy{operation: "account_read", limit: 120}, true
	case method == http.MethodPatch && path == "/api/v1/account":
		return requestRatePolicy{operation: "account_update", limit: 30}, true
	case method == http.MethodPost && path == "/api/v1/account/password":
		return requestRatePolicy{operation: "account_password", limit: 10}, true
	case method == http.MethodGet && path == "/api/v1/cache":
		return requestRatePolicy{operation: "cache_read", limit: 120}, true
	case method == http.MethodGet && path == "/api/v1/cache/cleanup":
		return requestRatePolicy{operation: "cache_cleanup_read", limit: 120}, true
	case method == http.MethodPost && path == "/api/v1/cache/cleanup":
		return requestRatePolicy{operation: "cache_cleanup_start", limit: 10}, true
	case method == http.MethodGet && path == "/api/v1/catalog/state":
		return requestRatePolicy{operation: "catalog_state", limit: 120}, true
	case method == http.MethodGet && (path == "/api/v1/semantic/assets" || path == "/api/v1/semantic/videos"):
		return requestRatePolicy{operation: "semantic_search", limit: semanticSearchRequestsPerMinute}, true
	case method == http.MethodGet && path == "/api/v1/ai/tag-vocabulary":
		return requestRatePolicy{operation: "ai_tag_vocabulary_read", limit: 120}, true
	case method == http.MethodPut && path == "/api/v1/ai/tag-vocabulary":
		return requestRatePolicy{operation: "ai_tag_vocabulary_write", limit: 10}, true
	case method == http.MethodPost && path == "/api/v1/ai/tag-suggestion-reviews":
		return requestRatePolicy{operation: "ai_tag_reviews", limit: 30}, true
	case method == http.MethodGet && libraryAIPath(path, "/ai/tag-suggestions"):
		return requestRatePolicy{operation: "ai_tag_suggestions_read", limit: 60}, true
	case method == http.MethodPost && libraryAIPath(path, "/ai/tag-suggestions/jobs"):
		return requestRatePolicy{operation: "ai_tag_suggestions_job", limit: 10}, true
	case method == http.MethodPost && libraryAIPath(path, "/ai/tag-suggestion-reviews/clear"):
		return requestRatePolicy{operation: "ai_tag_reviews_clear", limit: 10}, true
	case method == http.MethodPost && libraryAIPath(path, "/ai/video-semantic/jobs"):
		return requestRatePolicy{operation: "ai_video_semantic_job", limit: 10}, true
	default:
		return requestRatePolicy{}, false
	}
}

func libraryAIPath(path, suffix string) bool {
	prefix := "/api/v1/libraries/"
	if !strings.HasPrefix(path, prefix) || !strings.HasSuffix(path, suffix) {
		return false
	}
	id := strings.TrimSuffix(strings.TrimPrefix(path, prefix), suffix)
	if !strings.HasPrefix(id, "lib_") || len(id) == len("lib_") {
		return false
	}
	for _, current := range id[len("lib_"):] {
		if current < '0' || current > '9' {
			return false
		}
	}
	return true
}

func (limiter *requestRateLimiter) allow(key string, limit int) (string, bool) {
	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	now := limiter.now().UTC()
	bucket, exists := limiter.buckets[key]
	if exists && now.Before(bucket.start) {
		bucket.start = now
		limiter.buckets[key] = bucket
	}
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
		if now.Before(bucket.start) {
			bucket.start = now
			limiter.buckets[key] = bucket
			continue
		}
		if now.Sub(bucket.start) >= requestRateWindow {
			delete(limiter.buckets, key)
		}
	}
}
