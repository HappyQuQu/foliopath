package app

import (
	"context"
	"fmt"
)

type nativeImageRuntime interface {
	Start() error
	Shutdown()
}

func newImageRuntimeComponent(runtime nativeImageRuntime) (component, error) {
	if runtime == nil {
		return component{}, fmt.Errorf(
			"%w: image runtime is required",
			errInvalidComponent,
		)
	}
	return component{
		name: "image-runtime",
		start: func(context.Context) error {
			return runtime.Start()
		},
		stop: func(context.Context) error {
			runtime.Shutdown()
			return nil
		},
	}, nil
}
