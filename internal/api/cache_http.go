package api

import (
	"context"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/HappyQuQu/foliopath/internal/thumbnail"
)

var idempotencyKeyPattern = regexp.MustCompile(`^[A-Za-z0-9._~-]{8,128}$`)

type CacheService interface {
	Summary(context.Context) (thumbnail.CacheSummary, error)
	Cleanup(context.Context) (thumbnail.Cleanup, error)
	StartCleanup(context.Context, string) (thumbnail.CleanupRequestResult, error)
}

type cacheSummaryResponse struct {
	UsageBytes         int64                `json:"usageBytes"`
	QuotaBytes         int64                `json:"quotaBytes"`
	HighWatermarkBytes int64                `json:"highWatermarkBytes"`
	LowWatermarkBytes  int64                `json:"lowWatermarkBytes"`
	AvailableBytes     int64                `json:"availableBytes"`
	SafeFreeSpaceBytes int64                `json:"safeFreeSpaceBytes"`
	Pressure           string               `json:"pressure"`
	Cleanup            cacheCleanupResponse `json:"cleanup"`
}

type cacheCleanupResponse struct {
	Status              thumbnail.CleanupStatus `json:"status"`
	RequestedAt         *string                 `json:"requestedAt"`
	StartedAt           *string                 `json:"startedAt"`
	FinishedAt          *string                 `json:"finishedAt"`
	InitialUsageBytes   int64                   `json:"initialUsageBytes"`
	RemainingUsageBytes int64                   `json:"remainingUsageBytes"`
	ReclaimedBytes      int64                   `json:"reclaimedBytes"`
	DeletedEntries      int64                   `json:"deletedEntries"`
	ErrorCode           *string                 `json:"errorCode"`
}

func registerCacheRoutes(mux *http.ServeMux, service CacheService) {
	mux.HandleFunc("GET /api/v1/cache", func(writer http.ResponseWriter, request *http.Request) {
		summary, err := service.Summary(request.Context())
		if err != nil {
			writeInternalError(writer, request)
			return
		}
		writer.Header().Set("ETag", cacheSummaryETag(summary))
		writeJSON(writer, http.StatusOK, cacheSummaryWire(summary))
	})
	mux.HandleFunc("GET /api/v1/cache/cleanup", func(writer http.ResponseWriter, request *http.Request) {
		cleanup, err := service.Cleanup(request.Context())
		if err != nil {
			writeInternalError(writer, request)
			return
		}
		writer.Header().Set("ETag", cacheCleanupETag(cleanup.Revision))
		writeJSON(writer, http.StatusOK, cacheCleanupWire(cleanup))
	})
	mux.HandleFunc("POST /api/v1/cache/cleanup", func(writer http.ResponseWriter, request *http.Request) {
		key := request.Header.Get("Idempotency-Key")
		if !idempotencyKeyPattern.MatchString(key) {
			writePublicError(
				writer, request, http.StatusBadRequest,
				codeInvalidRequest, "A valid idempotency key is required.",
			)
			return
		}
		result, err := service.StartCleanup(request.Context(), key)
		if err != nil {
			writeInternalError(writer, request)
			return
		}
		status := http.StatusOK
		if result.Created {
			status = http.StatusAccepted
		}
		writer.Header().Set("ETag", cacheCleanupETag(result.Cleanup.Revision))
		writer.Header().Set("Location", "/api/v1/cache/cleanup")
		writer.Header().Set("Idempotency-Replayed", strconv.FormatBool(result.Replayed))
		writeJSON(writer, status, cacheCleanupWire(result.Cleanup))
	})
}

func cacheSummaryWire(summary thumbnail.CacheSummary) cacheSummaryResponse {
	return cacheSummaryResponse{
		UsageBytes: summary.UsageBytes, QuotaBytes: summary.QuotaBytes,
		HighWatermarkBytes: summary.HighWatermarkBytes,
		LowWatermarkBytes:  summary.LowWatermarkBytes,
		AvailableBytes:     summary.AvailableBytes,
		SafeFreeSpaceBytes: summary.SafeFreeSpaceBytes,
		Pressure:           summary.Pressure, Cleanup: cacheCleanupWire(summary.Cleanup),
	}
}

func cacheCleanupWire(cleanup thumbnail.Cleanup) cacheCleanupResponse {
	return cacheCleanupResponse{
		Status:              cleanup.Status,
		RequestedAt:         nullableTimestamp(cleanup.RequestedAtMS),
		StartedAt:           nullableTimestamp(cleanup.StartedAtMS),
		FinishedAt:          nullableTimestamp(cleanup.FinishedAtMS),
		InitialUsageBytes:   cleanup.InitialUsageBytes,
		RemainingUsageBytes: cleanup.RemainingUsageBytes,
		ReclaimedBytes:      cleanup.ReclaimedBytes,
		DeletedEntries:      cleanup.DeletedEntries,
		ErrorCode:           cleanup.ErrorCode,
	}
}

func nullableTimestamp(value *int64) *string {
	if value == nil {
		return nil
	}
	formatted := time.UnixMilli(*value).UTC().Format(time.RFC3339Nano)
	return &formatted
}

func cacheCleanupETag(revision int64) string {
	return `"cache-cleanup-r` + strconv.FormatInt(revision, 10) + `"`
}

func cacheSummaryETag(summary thumbnail.CacheSummary) string {
	return `"cache-r` + strconv.FormatInt(summary.Cleanup.Revision, 10) +
		`-u` + strconv.FormatInt(summary.UsageBytes, 10) +
		`-a` + strconv.FormatInt(summary.AvailableBytes, 10) +
		`-q` + strconv.FormatInt(summary.QuotaBytes, 10) + `"`
}
