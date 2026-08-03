package app

import (
	"context"

	"github.com/HappyQuQu/foliopath/internal/systemlog"
)

type systemEventService interface {
	Record(context.Context, systemlog.Event) error
}

func newSystemEventComponent(service systemEventService) component {
	return component{
		name: "system-events",
		start: func(ctx context.Context) error {
			return service.Record(ctx, systemlog.Event{
				Level: systemlog.LevelInfo, Module: "application",
				EventCode: "application.started",
			})
		},
		stop: func(ctx context.Context) error {
			return service.Record(ctx, systemlog.Event{
				Level: systemlog.LevelInfo, Module: "application",
				EventCode: "application.stopped",
			})
		},
	}
}
