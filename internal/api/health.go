package api

import (
	"context"
	"errors"
	"net/http"

	appsettings "github.com/HappyQuQu/foliopath/internal/settings"
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
	Readiness        func() Readiness
	Authentication   AuthenticationService
	Account          AccountService
	Cache            CacheService
	SystemStatus     func(context.Context) (SystemStatus, error)
	LibraryPaths     LibraryPathService
	Libraries        LibraryLifecycleService
	ScanAdmission    ScanAdmissionService
	Scans            ScanQueryService
	MediaProgress    MediaProgressService
	MediaDiagnostics MediaDiagnosticsService
	SystemLogs       SystemLogService
	ReleaseInfo      ReleaseInfoService
	Settings         SettingsService
	Catalog          CatalogService
	Curation         CurationService
	Thumbnails       ThumbnailService
	Content          ContentService
	ContentAdmission ContentAdmission
}

type SettingsService interface {
	Get(context.Context) (appsettings.Values, error)
	Update(context.Context, int64, appsettings.Update) (appsettings.Values, error)
}

func NewRoutes(dependencies RouteDependencies) (http.Handler, error) {
	if dependencies.Readiness == nil ||
		dependencies.Authentication == nil ||
		dependencies.SystemStatus == nil ||
		dependencies.LibraryPaths == nil {
		return nil, errInvalidRouteDependencies
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health/live", handleLiveness)
	mux.HandleFunc("GET /health/ready", func(writer http.ResponseWriter, request *http.Request) {
		handleReadiness(writer, request, dependencies.Readiness())
	})
	registerAuthenticationRoutes(mux, dependencies.Authentication)
	if dependencies.Account != nil {
		registerAccountRoutes(mux, dependencies.Account)
	}
	if dependencies.Cache != nil {
		registerCacheRoutes(mux, dependencies.Cache)
	}
	registerLibraryPathRoutes(mux, dependencies.LibraryPaths)
	if dependencies.Libraries != nil {
		registerLibraryRoutes(mux, dependencies.Libraries)
	}
	if dependencies.ScanAdmission != nil {
		registerScanAdmissionRoute(mux, dependencies.ScanAdmission)
	}
	if dependencies.Scans != nil {
		registerScanQueryRoutes(mux, dependencies.Scans)
	}
	if dependencies.MediaProgress != nil {
		registerMediaProgressRoute(mux, dependencies.MediaProgress)
	}
	if dependencies.MediaDiagnostics != nil {
		registerMediaDiagnosticsRoutes(mux, dependencies.MediaDiagnostics)
	}
	if dependencies.SystemLogs != nil {
		registerSystemLogRoute(mux, dependencies.SystemLogs)
	}
	if dependencies.ReleaseInfo != nil {
		registerReleaseInfoRoute(mux, dependencies.ReleaseInfo)
	}
	if dependencies.Settings != nil {
		registerSettingsRoutes(mux, dependencies.Settings)
	}
	if dependencies.Catalog != nil {
		registerCatalogRoutes(mux, dependencies.Catalog)
	}
	if dependencies.Curation != nil {
		registerCurationRoutes(mux, dependencies.Curation)
	}
	if dependencies.Thumbnails != nil {
		registerThumbnailRoute(mux, dependencies.Thumbnails)
	}
	if dependencies.Content != nil {
		registerContentRoutes(mux, dependencies.Content, dependencies.ContentAdmission)
	}
	mux.HandleFunc("GET /api/v1/status", func(writer http.ResponseWriter, request *http.Request) {
		status, err := dependencies.SystemStatus(request.Context())
		if err != nil {
			writeInternalError(writer, request)
			return
		}
		writeJSON(writer, http.StatusOK, status)
	})

	protected := requireAPIAuthentication(
		routeFallback{mux: mux},
		dependencies.Authentication,
	)
	return limitRequests(
		protected,
		newRequestRateLimiter(nil),
	), nil
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
	_, pattern := routes.mux.Handler(request)
	if pattern == "" {
		notFoundHandler().ServeHTTP(writer, request)
		return
	}
	// ServeMux.ServeHTTP performs the match that populates request PathValue
	// entries. Calling the handler returned by ServeMux.Handler directly would
	// discard route variables after authentication middleware has run.
	routes.mux.ServeHTTP(writer, request)
}
