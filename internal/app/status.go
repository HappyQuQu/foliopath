package app

import (
	"context"
	"net/http"

	"github.com/HappyQuQu/foliopath/internal/api"
	"github.com/HappyQuQu/foliopath/internal/media"
)

// denySystemStatus is replaced by the session service wiring in S1-103. Until
// then, the implemented status handler remains fail-closed in production.
func denySystemStatus(*http.Request) bool {
	return false
}

func systemStatusProvider(
	version string,
	readiness *readinessState,
) func(context.Context) (api.SystemStatus, error) {
	return func(context.Context) (api.SystemStatus, error) {
		runtimeState := "degraded"
		if readiness.snapshot().Ready {
			runtimeState = "ready"
		}
		return api.SystemStatus{
			Version:          normalizedApplicationVersion(version),
			APIVersion:       "v1",
			RuntimeState:     runtimeState,
			Initialized:      false,
			ReadOnlyMedia:    true,
			SupportedLocales: []string{"zh-CN", "en"},
			SupportedMedia: api.SupportedMedia{
				ImageMIMETypes:   media.ImageMIMETypes(),
				VideoMIMETypes:   media.VideoMIMETypes(),
				VideoTranscoding: false,
			},
		}, nil
	}
}

func normalizedApplicationVersion(version string) string {
	if version == "" {
		return "dev"
	}
	return version
}
