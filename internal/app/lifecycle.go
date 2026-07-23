package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"
)

type component struct {
	name  string
	start func(context.Context) error
	done  <-chan error
	stop  func(context.Context) error
}

type application struct {
	configuration   configuration
	logger          *slog.Logger
	components      []component
	shutdownTimeout time.Duration
}

func newApplication(components []component, shutdownTimeout time.Duration) (*application, error) {
	if shutdownTimeout <= 0 {
		return nil, fmt.Errorf("%w: shutdown timeout must be positive", errInvalidComponent)
	}

	seen := make(map[string]struct{}, len(components))
	owned := make([]component, len(components))
	copy(owned, components)

	for index, candidate := range owned {
		if candidate.name == "" {
			return nil, fmt.Errorf("%w: component %d has no name", errInvalidComponent, index)
		}
		if candidate.start == nil || candidate.stop == nil {
			return nil, fmt.Errorf("%w: component %q has an incomplete lifecycle", errInvalidComponent, candidate.name)
		}
		if _, exists := seen[candidate.name]; exists {
			return nil, fmt.Errorf("%w: duplicate component %q", errInvalidComponent, candidate.name)
		}
		seen[candidate.name] = struct{}{}
	}

	return &application{
		logger:          slog.New(slog.NewTextHandler(io.Discard, nil)),
		components:      owned,
		shutdownTimeout: shutdownTimeout,
	}, nil
}

func (application *application) run(ctx context.Context) error {
	application.logger.InfoContext(
		ctx,
		"application.starting",
		slog.Int("component_count", len(application.components)),
	)
	started := make([]component, 0, len(application.components))
	for _, candidate := range application.components {
		if ctx.Err() != nil {
			return application.shutdown(started)
		}
		if err := candidate.start(ctx); err != nil {
			application.logger.ErrorContext(
				ctx,
				"application.component_start_failed",
				slog.String("component", candidate.name),
			)
			startErr := fmt.Errorf("start component %q: %w", candidate.name, err)
			return errors.Join(startErr, application.shutdown(started))
		}
		started = append(started, candidate)
	}

	runErr := waitForStop(ctx, started)
	if runErr != nil {
		application.logger.ErrorContext(
			ctx,
			"application.component_stopped",
		)
	}
	shutdownErr := application.shutdown(started)
	if shutdownErr != nil {
		application.logger.Error(
			"application.shutdown_failed",
		)
	}
	application.logger.Info("application.stopped")
	return errors.Join(runErr, shutdownErr)
}

func waitForStop(ctx context.Context, components []component) error {
	monitorContext, cancel := context.WithCancel(ctx)
	defer cancel()

	failures := make(chan error, len(components))
	for _, candidate := range components {
		if candidate.done == nil {
			continue
		}
		go monitor(candidate, monitorContext, failures)
	}

	select {
	case <-ctx.Done():
		return nil
	case err := <-failures:
		return err
	}
}

func monitor(candidate component, ctx context.Context, failures chan<- error) {
	select {
	case err, open := <-candidate.done:
		if !open || err == nil {
			err = errComponentStopped
		}
		select {
		case failures <- fmt.Errorf("component %q stopped: %w", candidate.name, err):
		case <-ctx.Done():
		}
	case <-ctx.Done():
	}
}

func (application *application) shutdown(components []component) error {
	if len(components) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), application.shutdownTimeout)
	defer cancel()

	var shutdownErr error
	for index := len(components) - 1; index >= 0; index-- {
		candidate := components[index]
		if err := stopComponent(ctx, candidate); err != nil {
			shutdownErr = errors.Join(
				shutdownErr,
				fmt.Errorf("stop component %q: %w", candidate.name, err),
			)
		}
		if ctx.Err() != nil {
			break
		}
	}
	return shutdownErr
}

func stopComponent(ctx context.Context, candidate component) error {
	stopped := make(chan error, 1)
	go func() {
		stopped <- candidate.stop(ctx)
	}()

	select {
	case err := <-stopped:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
