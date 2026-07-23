package app

import (
	"context"
	"fmt"

	"github.com/HappyQuQu/foliopath/internal/api"
	"github.com/HappyQuQu/foliopath/internal/auth"
	"github.com/HappyQuQu/foliopath/internal/media"
)

type setupStatus interface {
	SetupState(context.Context) (auth.SetupState, error)
}

func systemStatusProvider(
	version string,
	readiness *readinessState,
	setup setupStatus,
) func(context.Context) (api.SystemStatus, error) {
	return func(ctx context.Context) (api.SystemStatus, error) {
		setupState, err := setup.SetupState(ctx)
		if err != nil {
			return api.SystemStatus{}, fmt.Errorf("read system setup state: %w", err)
		}
		runtimeState := "degraded"
		if readiness.snapshot().Ready {
			runtimeState = "ready"
		}
		return api.SystemStatus{
			Version:          normalizedApplicationVersion(version),
			APIVersion:       "v1",
			RuntimeState:     runtimeState,
			Initialized:      setupState == auth.SetupComplete,
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
