package app

import (
	"context"
	"fmt"

	"github.com/HappyQuQu/foliopath/internal/resourcecontrol"
	appsettings "github.com/HappyQuQu/foliopath/internal/settings"
)

func newResourceLimitsComponent(
	settingsService *appsettings.Service,
	controller *resourcecontrol.Controller,
) (component, error) {
	if settingsService == nil || controller == nil {
		return component{}, fmt.Errorf(
			"%w: resource limit dependencies are required",
			errInvalidComponent,
		)
	}
	return component{
		name: "resource-limits",
		start: func(ctx context.Context) error {
			values, err := settingsService.Get(ctx)
			if err != nil {
				return fmt.Errorf("load resource limits: %w", err)
			}
			if err := controller.ApplyLimits(resourcecontrol.Limits{
				Background: int(values.BackgroundConcurrency),
				Content:    int(values.ContentReadConcurrency),
			}); err != nil {
				return fmt.Errorf("apply resource limits: %w", err)
			}
			return nil
		},
		stop: func(context.Context) error { return nil },
	}, nil
}
