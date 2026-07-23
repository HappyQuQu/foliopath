// Package app is the application's sole composition and lifecycle boundary.
package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/HappyQuQu/foliopath/internal/api"
	"github.com/HappyQuQu/foliopath/internal/auth"
)

const defaultShutdownTimeout = 10 * time.Second

var (
	errInvalidComponent = errors.New("invalid application component")
	errComponentStopped = errors.New("application component stopped unexpectedly")
)

// Input contains the process values that the minimal command entry hands to the
// application. Configuration parsing and validation remain owned by this
// package, not cmd/foliopath.
type Input struct {
	Args    []string
	Environ []string
	Version string
}

// Run owns the process root context. All concrete dependency construction and
// lifecycle wiring stays below this boundary.
func Run(input Input) error {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	return run(ctx, input)
}

func run(ctx context.Context, input Input) error {
	if ctx == nil {
		return errors.New("application context is nil")
	}

	application, err := compose(input)
	if err != nil {
		return fmt.Errorf("compose application: %w", err)
	}

	return application.run(ctx)
}

// compose is the only function that may know the complete concrete dependency
// graph. Later Stage 1 tasks add storage and workers here without moving their
// construction into cmd or capability packages.
func compose(input Input) (*application, error) {
	configuration, err := loadConfiguration(input)
	if err != nil {
		return nil, err
	}

	return composeConfiguration(input, configuration)
}

// composeConfiguration keeps the production dependency graph reusable by
// package-level integration tests without exposing configurable filesystem
// roots at the process boundary.
func composeConfiguration(input Input, configuration configuration) (*application, error) {
	logger := newJSONLogger(os.Stdout)
	readiness := newReadinessState()
	databaseComponent, database := newDatabaseComponent(configuration.dataRoot, readiness)
	authentication, err := auth.NewService(
		database,
		auth.NewArgon2idPasswordManager(nil),
		auth.ServiceOptions{},
	)
	if err != nil {
		return nil, fmt.Errorf("construct authentication service: %w", err)
	}
	routes, err := api.NewRoutes(api.RouteDependencies{
		Readiness:      readiness.snapshot,
		Authentication: authentication,
		SystemStatus:   systemStatusProvider(input.Version, readiness, authentication),
	})
	if err != nil {
		return nil, err
	}
	httpComponent, httpService := newHTTPComponent(
		configuration.listenAddress,
		api.NewHandler(routes, logger),
		logger,
	)
	application, err := newApplication(
		[]component{
			databaseComponent,
			httpComponent,
			readinessLifecycle(readiness),
		},
		defaultShutdownTimeout,
	)
	if err != nil {
		return nil, err
	}
	application.configuration = configuration
	application.logger = logger
	application.http = httpService
	application.authentication = authentication
	return application, nil
}
