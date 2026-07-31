package app

import (
	"context"
	"fmt"

	"github.com/HappyQuQu/foliopath/internal/resourcecontrol"
	appsettings "github.com/HappyQuQu/foliopath/internal/settings"
)

func newResourceProfileComponent(
	settingsService *appsettings.Service,
	controller *resourcecontrol.Controller,
) (component, error) {
	if settingsService == nil || controller == nil {
		return component{}, fmt.Errorf(
			"%w: resource profile dependencies are required",
			errInvalidComponent,
		)
	}
	return component{
		name: "resource-profile",
		start: func(ctx context.Context) error {
			values, err := settingsService.Get(ctx)
			if err != nil {
				return fmt.Errorf("load resource profile: %w", err)
			}
			if err := controller.ApplyResourceProfile(values.ResourceProfile); err != nil {
				return fmt.Errorf("apply resource profile: %w", err)
			}
			return nil
		},
		stop: func(context.Context) error { return nil },
	}, nil
}
