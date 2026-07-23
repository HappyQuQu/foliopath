package api

import (
	"context"
	"errors"
	"net/http"
)

const (
	ReadinessApplicationData     = "application_data_unavailable"
	ReadinessMigrationFailed     = "migration_failed"
	ReadinessDatabaseUnavailable = "database_unavailable"
	ReadinessShuttingDown        = "shutting_down"
)

var errInvalidRouteDependencies = errors.New("invalid API route dependencies")

type Readiness struct {
	Ready      bool
	ReasonCode string
}

type SystemStatus struct {
	Version          string         `json:"version"`
	APIVersion       string         `json:"apiVersion"`
	RuntimeState     string         `json:"runtimeState"`
	Initialized      bool           `json:"initialized"`
	ReadOnlyMedia    bool           `json:"readOnlyMedia"`
	SupportedLocales []string       `json:"supportedLocales"`
	SupportedMedia   SupportedMedia `json:"supportedMedia"`
}

type SupportedMedia struct {
	ImageMIMETypes   []string `json:"imageMimeTypes"`
	VideoMIMETypes   []string `json:"videoMimeTypes"`
	VideoTranscoding bool     `json:"videoTranscoding"`
}

type RouteDependencies struct {
	Readiness       func() Readiness
	AuthorizeStatus func(*http.Request) bool
	SystemStatus    func(context.Context) (SystemStatus, error)
}

func NewRoutes(dependencies RouteDependencies) (http.Handler, error) {
	if dependencies.Readiness == nil ||
		dependencies.AuthorizeStatus == nil ||
		dependencies.SystemStatus == nil {
		return nil, errInvalidRouteDependencies
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health/live", handleLiveness)
	mux.HandleFunc("GET /health/ready", func(writer http.ResponseWriter, request *http.Request) {
		handleReadiness(writer, request, dependencies.Readiness())
	})
	mux.HandleFunc("GET /api/v1/status", func(writer http.ResponseWriter, request *http.Request) {
		if !dependencies.AuthorizeStatus(request) {
			writePublicError(
				writer,
				request,
				http.StatusUnauthorized,
				"authentication_required",
				"An authenticated administrator session is required.",
			)
			return
		}

		status, err := dependencies.SystemStatus(request.Context())
		if err != nil {
			writeInternalError(writer, request)
			return
		}
		writeJSON(writer, http.StatusOK, status)
	})

	return routeFallback{mux: mux}, nil
}

func handleLiveness(writer http.ResponseWriter, _ *http.Request) {
	writeJSON(writer, http.StatusOK, map[string]any{
		"status":     "live",
		"reasonCode": nil,
	})
}

func handleReadiness(writer http.ResponseWriter, _ *http.Request, readiness Readiness) {
	if readiness.Ready {
		writeJSON(writer, http.StatusOK, map[string]any{
			"status":     "ready",
			"reasonCode": nil,
		})
		return
	}

	writer.Header().Set("Retry-After", "1")
	writeJSON(writer, http.StatusServiceUnavailable, map[string]any{
		"status":     "not_ready",
		"reasonCode": safeReadinessReason(readiness.ReasonCode),
	})
}

func safeReadinessReason(reason string) string {
	switch reason {
	case ReadinessApplicationData,
		ReadinessMigrationFailed,
		ReadinessDatabaseUnavailable,
		ReadinessShuttingDown:
		return reason
	default:
		return ReadinessDatabaseUnavailable
	}
}

type routeFallback struct {
	mux *http.ServeMux
}

func (routes routeFallback) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	handler, pattern := routes.mux.Handler(request)
	if pattern == "" {
		notFoundHandler().ServeHTTP(writer, request)
		return
	}
	handler.ServeHTTP(writer, request)
}
