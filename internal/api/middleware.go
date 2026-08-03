package api

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/HappyQuQu/foliopath/internal/systemlog"
)

type SystemEventRecorder interface {
	Record(context.Context, systemlog.Event) error
}

// NewHandler applies the transport invariants shared by every route. A nil
// next handler becomes the contract-shaped not-found fallback.
func NewHandler(next http.Handler, logger *slog.Logger) http.Handler {
	return NewHandlerWithTransport(next, logger, TransportConfig{})
}

// NewHandlerWithTransport applies the shared transport invariants with an
// explicit trusted-proxy policy.
func NewHandlerWithTransport(
	next http.Handler,
	logger *slog.Logger,
	transport TransportConfig,
) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	if next == nil {
		next = notFoundHandler()
	}

	return withRequestID(
		withSecurityHeaders(
			logRequests(
				recoverPanics(
					withTrustedTransport(
						withHSTS(next),
						transport,
					),
					logger,
				),
				logger,
				transport.SystemEvents,
			),
		),
		nil,
	)
}

func logRequests(
	next http.Handler,
	logger *slog.Logger,
	recorder SystemEventRecorder,
) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		tracker := &responseTracker{ResponseWriter: writer}
		started := time.Now()
		defer func() {
			requestID, _ := RequestIDFromContext(request.Context())
			status := tracker.status
			if status == 0 {
				status = http.StatusOK
			}
			duration := time.Since(started).Milliseconds()
			logger.InfoContext(
				request.Context(),
				"http.request_completed",
				slog.String("request_id", requestID),
				slog.String("method", request.Method),
				slog.Int("status", status),
				slog.Int64("duration_ms", duration),
			)
			if recorder != nil {
				level, eventCode, record := systemEventForRequest(
					request.Method,
					status,
				)
				if record {
					_ = recorder.Record(request.Context(), systemlog.Event{
						Level: level, Module: "http", EventCode: eventCode,
						RequestID: requestID, Method: request.Method,
						RoutePattern: request.Pattern, StatusCode: status,
						DurationMS: duration,
					})
				}
			}
		}()
		next.ServeHTTP(tracker, request)
	})
}

func systemEventForRequest(
	method string,
	status int,
) (systemlog.Level, string, bool) {
	if status >= 500 {
		return systemlog.LevelError, "http.request_failed", true
	}
	if status >= 400 {
		return systemlog.LevelWarning, "http.request_rejected", true
	}
	if method != http.MethodGet && method != http.MethodHead &&
		method != http.MethodOptions {
		return systemlog.LevelInfo, "http.admin_operation", true
	}
	return "", "", false
}

func recoverPanics(next http.Handler, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		tracker := &responseTracker{ResponseWriter: writer}
		defer func() {
			recovered := recover()
			if recovered == nil {
				return
			}

			requestID, _ := RequestIDFromContext(request.Context())
			logger.ErrorContext(
				request.Context(),
				"http.panic_recovered",
				slog.String("request_id", requestID),
			)
			if !tracker.committed {
				writeInternalError(tracker, request)
				return
			}
			panic(recovered)
		}()

		next.ServeHTTP(tracker, request)
	})
}

type responseTracker struct {
	http.ResponseWriter
	committed bool
	status    int
}

func (tracker *responseTracker) WriteHeader(status int) {
	if tracker.committed {
		return
	}
	tracker.committed = true
	tracker.status = status
	tracker.ResponseWriter.WriteHeader(status)
}

func (tracker *responseTracker) Write(body []byte) (int, error) {
	if !tracker.committed {
		tracker.WriteHeader(http.StatusOK)
	}
	return tracker.ResponseWriter.Write(body)
}

func (tracker *responseTracker) Unwrap() http.ResponseWriter {
	return tracker.ResponseWriter
}
