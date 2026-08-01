// Package app is the application's sole composition and lifecycle boundary.
package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/HappyQuQu/foliopath/internal/api"
	"github.com/HappyQuQu/foliopath/internal/auth"
	"github.com/HappyQuQu/foliopath/internal/catalog"
	"github.com/HappyQuQu/foliopath/internal/jobs"
	"github.com/HappyQuQu/foliopath/internal/library"
	"github.com/HappyQuQu/foliopath/internal/media"
	"github.com/HappyQuQu/foliopath/internal/media/imagevips"
	"github.com/HappyQuQu/foliopath/internal/media/videoffmpeg"
	"github.com/HappyQuQu/foliopath/internal/resourcecontrol"
	"github.com/HappyQuQu/foliopath/internal/scanner"
	appsettings "github.com/HappyQuQu/foliopath/internal/settings"
	"github.com/HappyQuQu/foliopath/internal/thumbnail"
	"github.com/HappyQuQu/foliopath/internal/thumbnail/cachefs"
	"github.com/HappyQuQu/foliopath/internal/webassets"
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
	directorySource, mediaRootComponent, err := newMediaRootService(configuration.mediaRoot)
	if err != nil {
		return nil, fmt.Errorf("construct allowed media root: %w", err)
	}
	libraryPaths, err := library.NewPathService(
		directorySource,
		database,
		library.PathServiceOptions{},
	)
	if err != nil {
		return nil, fmt.Errorf("construct library path service: %w", err)
	}
	removalWorker, err := library.NewRemovalWorker(
		database,
		libraryCacheCleaner{dataRoot: configuration.dataRoot},
		0,
	)
	if err != nil {
		return nil, fmt.Errorf("construct library removal worker: %w", err)
	}
	removalComponent, err := newRemovalWorkerComponent(removalWorker)
	if err != nil {
		return nil, err
	}
	scanSignal := jobs.NewSignal()
	scheduleSignal := jobs.NewSignal()
	discoveryConfigSignal := jobs.NewSignal()
	discoveryRecoverySignal := jobs.NewSignal()
	reconcileSignal := jobs.NewSignal()
	mediaSignal := jobs.NewSignal()
	cacheSignal := jobs.NewSignal()
	resourceController, err := resourcecontrol.NewController(resourcecontrol.ProfileEco)
	if err != nil {
		return nil, fmt.Errorf("construct resource controller: %w", err)
	}
	scanAdmission, err := scanner.NewAdmissionService(database, scanSignal)
	if err != nil {
		return nil, fmt.Errorf("construct scan admission service: %w", err)
	}
	scanQueries, err := scanner.NewQueryService(database, nil)
	if err != nil {
		return nil, fmt.Errorf("construct scan query service: %w", err)
	}
	settingsService, err := appsettings.NewService(
		database,
		scheduleSignal,
		discoveryConfigSignal,
		cacheSignal,
		resourceController,
		appsettings.FieldValidators{
			Schedule:   scanner.ValidateScheduledScanInterval,
			CacheQuota: thumbnail.ValidateCacheQuota,
			Language:   appsettings.ValidateLanguage,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("construct settings service: %w", err)
	}
	resourceProfileComponent, err := newResourceProfileComponent(
		settingsService,
		resourceController,
	)
	if err != nil {
		return nil, err
	}
	catalogService, err := catalog.NewService(database, nil)
	if err != nil {
		return nil, fmt.Errorf("construct catalog service: %w", err)
	}
	contentService, err := media.NewContentService(database, directorySource)
	if err != nil {
		return nil, fmt.Errorf("construct media content service: %w", err)
	}
	scanService, err := scanner.NewService(
		mediaWakeScanRepository{
			Repository:     database,
			waker:          mediaSignal,
			discoveryWaker: discoveryRecoverySignal,
		},
		scanner.Config{},
	)
	if err != nil {
		return nil, fmt.Errorf("construct scan service: %w", err)
	}
	scanProcessor, err := scanner.NewClaimedProcessor(scanService, directorySource)
	if err != nil {
		return nil, fmt.Errorf("construct scan processor: %w", err)
	}
	limitedScanProcessor, err := resourcecontrol.LimitBackground(resourceController, scanProcessor)
	if err != nil {
		return nil, fmt.Errorf("limit scan processor: %w", err)
	}
	scanWorker, err := jobs.NewWorkerPool(
		scanJobQueue{database: database},
		limitedScanProcessor,
		scanSignal,
		jobs.WorkerOptions{},
	)
	if err != nil {
		return nil, fmt.Errorf("construct scan worker: %w", err)
	}
	scanScheduler, err := scanner.NewScheduler(
		database, scanSignal, scheduleSignal, scanner.SchedulerOptions{},
	)
	if err != nil {
		return nil, fmt.Errorf("construct scan scheduler: %w", err)
	}
	scanComponent, err := newScanWorkerComponent(scanWorker, scanAdmission, scanScheduler)
	if err != nil {
		return nil, err
	}
	reconcileAdmission, err := scanner.NewReconcileAdmission(
		database,
		reconcileSignal,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"construct automatic discovery admission service: %w",
			err,
		)
	}
	discoveryCoordinator, err := newAutomaticDiscoveryCoordinator(
		database,
		directorySource,
		reconcileAdmission,
		scanAdmission,
		discoveryConfigSignal,
		discoveryRecoverySignal,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"construct automatic discovery coordinator: %w",
			err,
		)
	}
	reconcileProcessor, err := scanner.NewReconcileProcessor(
		database,
		directorySource,
		mediaSignal,
		discoveryCoordinator,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"construct automatic discovery processor: %w",
			err,
		)
	}
	limitedReconcileProcessor, err := resourcecontrol.LimitBackground(
		resourceController,
		reconcileProcessor,
	)
	if err != nil {
		return nil, fmt.Errorf("limit automatic discovery processor: %w", err)
	}
	reconcileWorker, err := jobs.NewWorkerPool(
		reconcileJobQueue{database: database},
		limitedReconcileProcessor,
		reconcileSignal,
		jobs.WorkerOptions{Workers: scanner.MaxConcurrentReconciles},
	)
	if err != nil {
		return nil, fmt.Errorf(
			"construct automatic discovery worker: %w",
			err,
		)
	}
	automaticDiscoveryComponent, err := newAutomaticDiscoveryComponent(
		reconcileWorker,
		discoveryCoordinator,
	)
	if err != nil {
		return nil, err
	}
	cachePublisher, err := cachefs.New(
		filepath.Join(configuration.dataRoot, "cache"),
	)
	if err != nil {
		return nil, fmt.Errorf("construct thumbnail cache: %w", err)
	}
	cacheManager, err := thumbnail.NewCacheManager(database, cachePublisher)
	if err != nil {
		return nil, fmt.Errorf("construct thumbnail cache manager: %w", err)
	}
	imageRuntimeComponent, err := newImageRuntimeComponent(imagevips.NewRuntime())
	if err != nil {
		return nil, fmt.Errorf("construct image runtime: %w", err)
	}
	videoProcessor, err := videoffmpeg.New(videoffmpeg.Options{
		StoryboardTempRoot: filepath.Join(
			configuration.dataRoot,
			"tmp",
			"storyboard",
		),
	})
	if err != nil {
		return nil, fmt.Errorf("construct video processor: %w", err)
	}
	thumbnailService, err := thumbnail.NewService(
		database,
		directorySource,
		cachePublisher,
		cacheManager,
		imagevips.New(),
		videoProcessor,
		thumbnail.ServiceOptions{},
	)
	if err != nil {
		return nil, fmt.Errorf("construct thumbnail service: %w", err)
	}
	storyboardService, err := thumbnail.NewStoryboardService(
		database,
		directorySource,
		cachePublisher,
		cacheManager,
		videoProcessor,
		thumbnail.ServiceOptions{},
	)
	if err != nil {
		return nil, fmt.Errorf("construct storyboard service: %w", err)
	}
	thumbnailDelivery, err := thumbnail.NewDeliveryService(
		database, cachePublisher, mediaSignal,
	)
	if err != nil {
		return nil, fmt.Errorf("construct thumbnail delivery service: %w", err)
	}
	mediaProgress, err := thumbnail.NewProgressService(database)
	if err != nil {
		return nil, fmt.Errorf("construct media progress service: %w", err)
	}
	mediaProcessor, err := thumbnail.NewClaimedProcessor(
		thumbnailService, storyboardService, database,
	)
	if err != nil {
		return nil, fmt.Errorf("construct claimed media processor: %w", err)
	}
	limitedMediaProcessor, err := resourcecontrol.LimitBackground(
		resourceController,
		mediaProcessor,
	)
	if err != nil {
		return nil, fmt.Errorf("limit media processor: %w", err)
	}
	mediaWorker, err := jobs.NewWorkerPool(
		mediaJobQueue{database: database},
		limitedMediaProcessor,
		mediaSignal,
		jobs.WorkerOptions{Workers: thumbnail.MediaWorkerCount},
	)
	if err != nil {
		return nil, fmt.Errorf("construct media worker: %w", err)
	}
	mediaComponent, err := newMediaWorkerComponent(
		mediaWorker, cacheManager, cacheSignal,
	)
	if err != nil {
		return nil, err
	}
	libraries, err := library.NewLifecycleService(
		database,
		directorySource,
		multiWaker{scanSignal, discoveryConfigSignal},
		multiWaker{removalWorker, discoveryConfigSignal},
		library.LifecycleOptions{},
	)
	if err != nil {
		return nil, fmt.Errorf("construct library lifecycle service: %w", err)
	}
	routes, err := api.NewRoutes(api.RouteDependencies{
		Readiness:        readiness.snapshot,
		Authentication:   authentication,
		Account:          authentication,
		Cache:            cacheManager,
		SystemStatus:     systemStatusProvider(input.Version, readiness, authentication),
		LibraryPaths:     libraryPaths,
		Libraries:        libraries,
		ScanAdmission:    scanAdmission,
		Scans:            scanQueries,
		MediaProgress:    mediaProgress,
		Settings:         settingsService,
		Catalog:          catalogService,
		Thumbnails:       thumbnailDelivery,
		Content:          contentService,
		ContentAdmission: resourceController,
	})
	if err != nil {
		return nil, err
	}
	httpComponent, httpService := newHTTPComponent(
		configuration.listenAddress,
		api.NewHandlerWithTransport(
			webassets.NewHandler(routes),
			logger,
			api.TransportConfig{
				TrustedProxyPrefixes: parseTrustedProxyPrefixes(configuration.trustedProxies),
				RequireTrustedProxy:  configuration.requireProxy,
			},
		),
		logger,
	)
	application, err := newApplication(
		[]component{
			databaseComponent,
			resourceProfileComponent,
			mediaRootComponent,
			imageRuntimeComponent,
			mediaComponent,
			scanComponent,
			automaticDiscoveryComponent,
			removalComponent,
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
