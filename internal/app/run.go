// Package app is the application's sole composition and lifecycle boundary.
package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
	"github.com/HappyQuQu/foliopath/internal/api"
	"github.com/HappyQuQu/foliopath/internal/auth"
	"github.com/HappyQuQu/foliopath/internal/catalog"
	"github.com/HappyQuQu/foliopath/internal/curation"
	"github.com/HappyQuQu/foliopath/internal/inference/onnx"
	"github.com/HappyQuQu/foliopath/internal/inference/sentencepiece"
	"github.com/HappyQuQu/foliopath/internal/jobs"
	"github.com/HappyQuQu/foliopath/internal/library"
	"github.com/HappyQuQu/foliopath/internal/media"
	"github.com/HappyQuQu/foliopath/internal/media/imagevips"
	"github.com/HappyQuQu/foliopath/internal/media/videoffmpeg"
	"github.com/HappyQuQu/foliopath/internal/releaseinfo"
	releasegithub "github.com/HappyQuQu/foliopath/internal/releaseinfo/github"
	"github.com/HappyQuQu/foliopath/internal/resourcecontrol"
	"github.com/HappyQuQu/foliopath/internal/scanner"
	"github.com/HappyQuQu/foliopath/internal/semantic"
	appsettings "github.com/HappyQuQu/foliopath/internal/settings"
	"github.com/HappyQuQu/foliopath/internal/systemlog"
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
	aiInstallSignal := jobs.NewSignal()
	aiActivationSignal := jobs.NewSignal()
	semanticSignal := jobs.NewSignal()
	semanticClearSignal := jobs.NewSignal()
	tagReviewClearSignal := jobs.NewSignal()
	videoSemanticSignal := jobs.NewSignal()
	resourceController, err := resourcecontrol.NewController(resourcecontrol.Limits{Background: 1, Content: 1})
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
	mediaDiagnostics, err := thumbnail.NewDiagnosticsService(database, mediaSignal)
	if err != nil {
		return nil, fmt.Errorf("construct media diagnostics service: %w", err)
	}
	systemLogs, err := systemlog.NewService(database)
	if err != nil {
		return nil, fmt.Errorf("construct system log service: %w", err)
	}
	releaseSource, err := releasegithub.New(&http.Client{Timeout: 5 * time.Second}, "")
	if err != nil {
		return nil, fmt.Errorf("construct release source: %w", err)
	}
	releaseInformation, err := releaseinfo.NewService(
		normalizedApplicationVersion(input.Version),
		releaseSource,
	)
	if err != nil {
		return nil, fmt.Errorf("construct release information service: %w", err)
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
	resourceLimitsComponent, err := newResourceLimitsComponent(
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
	curationService, err := curation.NewService(database, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("construct curation service: %w", err)
	}
	contentService, err := media.NewContentService(database, directorySource)
	if err != nil {
		return nil, fmt.Errorf("construct media content service: %w", err)
	}
	scanService, err := scanner.NewService(
		mediaWakeScanRepository{
			Repository:     database,
			waker:          mediaSignal,
			cacheWaker:     cacheSignal,
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
		multiWaker{mediaSignal, cacheSignal},
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
	aiSource, aiSourceComponent := newAIModelSource(configuration.modelRoot)
	aiCatalog, err := aimodel.NewCatalog(nil)
	if err != nil {
		return nil, fmt.Errorf("construct reviewed AI model catalog: %w", err)
	}
	aiScanner, err := aimodel.NewScanner(aiSource, aiCatalog, runtime.GOARCH, nil)
	if err != nil {
		return nil, fmt.Errorf("construct AI model scanner: %w", err)
	}
	aiModels, err := aimodel.NewService(database, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("construct AI model service: %w", err)
	}
	aiOperations, err := aimodel.NewOperationService(database, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("construct AI operation service: %w", err)
	}
	managedModels, managedModelsComponent, err := newManagedAIModelStore(filepath.Join(configuration.dataRoot, "models"))
	if err != nil {
		return nil, fmt.Errorf("construct managed AI model store: %w", err)
	}
	aiInstaller, err := aimodel.NewInstaller(aiModels, aiSource, managedModels)
	if err != nil {
		return nil, fmt.Errorf("construct AI model installer: %w", err)
	}
	aiAdmission, err := aimodel.NewInstallAdmissionService(database, aiInstallSignal, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("construct AI model install admission: %w", err)
	}
	aiInstallWorker, err := aimodel.NewInstallWorker(database, aiInstaller, aiOperations, aiInstallSignal.Notifications(), 0)
	if err != nil {
		return nil, fmt.Errorf("construct AI model install worker: %w", err)
	}
	aiActivationSource, err := aimodel.NewActivationSourceRouter(aiSource, managedModels)
	if err != nil {
		return nil, fmt.Errorf("construct AI model activation source: %w", err)
	}
	aiAvailability, err := aimodel.NewAvailabilityService(aiModels, aiCatalog, aiActivationSource)
	if err != nil {
		return nil, fmt.Errorf("construct AI model availability service: %w", err)
	}
	aiAvailabilityComponent, err := newAIAvailabilityComponent(aiAvailability)
	if err != nil {
		return nil, fmt.Errorf("construct AI model availability lifecycle: %w", err)
	}
	aiActivationWorker, err := aimodel.NewActivationWorker(
		database, aiModels, aiOperations, aiCatalog, aiActivationSource, onnx.New(),
		aiAvailability, aiActivationSignal.Notifications(), 0, nil, nil,
	)
	if err != nil {
		return nil, fmt.Errorf("construct AI model activation worker: %w", err)
	}
	aiActivationAdmission, err := aimodel.NewActivationAdmissionService(database, aiActivationSignal, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("construct AI model activation admission: %w", err)
	}
	semanticSessions, err := newSemanticSessionOwner(semanticProductionSessionFactory{
		generations: database, models: aiModels, catalog: aiCatalog,
		source: aiActivationSource, availability: aiAvailability, runtime: onnx.New(),
	}, 0, 0, nil)
	if err != nil {
		return nil, fmt.Errorf("construct semantic inference sessions: %w", err)
	}
	semanticTextSessions, err := newSemanticTextSessionOwner(semanticProductionTextSessionFactory{
		generations: database, models: aiModels, catalog: aiCatalog,
		source: aiActivationSource, availability: aiAvailability,
		onnxRuntime: onnx.New(), tokenizerRuntime: sentencepiece.New(),
	}, 0, 0, nil)
	if err != nil {
		return nil, fmt.Errorf("construct semantic text inference sessions: %w", err)
	}
	semanticProcessor, err := semantic.NewBackfillProcessor(
		database, semanticContentSource{content: contentService}, imagevips.New(), semanticSessions,
		database, database, nil, 0,
	)
	if err != nil {
		return nil, fmt.Errorf("construct semantic backfill processor: %w", err)
	}
	limitedSemanticProcessor, err := resourcecontrol.LimitBackground(resourceController, semanticProcessor)
	if err != nil {
		return nil, fmt.Errorf("limit semantic backfill processor: %w", err)
	}
	semanticWorker, err := jobs.NewWorkerPool(
		semanticJobQueue{database: database}, limitedSemanticProcessor, semanticSignal,
		jobs.WorkerOptions{Workers: 1},
	)
	if err != nil {
		return nil, fmt.Errorf("construct semantic backfill worker: %w", err)
	}
	semanticSettings, err := semantic.NewSettingsService(database, nil)
	if err != nil {
		return nil, fmt.Errorf("construct semantic settings service: %w", err)
	}
	semanticAdmission, err := semantic.NewBackfillService(database, database, semanticSignal, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("construct semantic backfill admission: %w", err)
	}
	semanticClearProcessor, err := semantic.NewClearProcessor(database, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("construct semantic clear processor: %w", err)
	}
	limitedSemanticClearProcessor, err := resourcecontrol.LimitBackground(resourceController, semanticClearProcessor)
	if err != nil {
		return nil, fmt.Errorf("limit semantic clear processor: %w", err)
	}
	semanticClearWorker, err := jobs.NewWorkerPool(
		semanticClearQueue{database: database}, limitedSemanticClearProcessor, semanticClearSignal,
		jobs.WorkerOptions{Workers: 1},
	)
	if err != nil {
		return nil, fmt.Errorf("construct semantic clear worker: %w", err)
	}
	semanticClearAdmission, err := semantic.NewClearService(database, semanticClearSignal, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("construct semantic clear admission: %w", err)
	}
	semanticService, err := semantic.NewService(semanticSettings, semanticAdmission, aiOperations, semanticClearAdmission)
	if err != nil {
		return nil, fmt.Errorf("construct semantic service: %w", err)
	}
	tagReviewClearProcessor, err := semantic.NewTagReviewClearProcessor(database, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("construct tag review clear processor: %w", err)
	}
	limitedTagReviewClearProcessor, err := resourcecontrol.LimitBackground(resourceController, tagReviewClearProcessor)
	if err != nil {
		return nil, fmt.Errorf("limit tag review clear processor: %w", err)
	}
	tagReviewClearWorker, err := jobs.NewWorkerPool(tagReviewClearQueue{database: database}, limitedTagReviewClearProcessor,
		tagReviewClearSignal, jobs.WorkerOptions{Workers: 1})
	if err != nil {
		return nil, fmt.Errorf("construct tag review clear worker: %w", err)
	}
	tagReviewClearService, err := semantic.NewTagReviewClearService(database, tagReviewClearSignal, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("construct tag review clear service: %w", err)
	}
	if err := semanticService.EnableTagReviewClear(tagReviewClearService); err != nil {
		return nil, fmt.Errorf("compose tag review clear service: %w", err)
	}
	videoSemanticProcessor, err := semantic.NewVideoProcessor(semanticStoryboardSource{
		repository: database, cache: cachePublisher, splitter: imagevips.New(), waker: mediaSignal,
	}, imagevips.New(), semanticSessions, database, nil)
	if err != nil {
		return nil, fmt.Errorf("construct video semantic processor: %w", err)
	}
	videoSemanticJobProcessor, err := semantic.NewVideoJobProcessor(database, database, videoSemanticProcessor, nil)
	if err != nil {
		return nil, fmt.Errorf("construct video semantic job processor: %w", err)
	}
	limitedVideoSemanticProcessor, err := resourcecontrol.LimitBackground(resourceController, videoSemanticJobProcessor)
	if err != nil {
		return nil, fmt.Errorf("limit video semantic processor: %w", err)
	}
	videoSemanticWorker, err := jobs.NewWorkerPool(videoJobQueue{database: database}, limitedVideoSemanticProcessor,
		videoSemanticSignal, jobs.WorkerOptions{Workers: 1})
	if err != nil {
		return nil, fmt.Errorf("construct video semantic worker: %w", err)
	}
	videoSemanticJobs, err := semantic.NewVideoJobService(database, database, videoSemanticSignal, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("construct video semantic job service: %w", err)
	}
	if err := semanticService.EnableVideoJobs(videoSemanticJobs); err != nil {
		return nil, fmt.Errorf("compose video semantic job service: %w", err)
	}
	tagVocabulary, err := semantic.NewTagVocabularyService(database, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("construct tag vocabulary service: %w", err)
	}
	tagSuggestionList, err := semantic.NewTagSuggestionListService(database, nil)
	if err != nil {
		return nil, fmt.Errorf("construct tag suggestion list service: %w", err)
	}
	tagReviews, err := semantic.NewTagReviewService(database, curationService, nil)
	if err != nil {
		return nil, fmt.Errorf("construct tag review service: %w", err)
	}
	semanticSearch, err := semantic.NewSearchService(database, database, semanticTextSessions, nil)
	if err != nil {
		return nil, fmt.Errorf("construct semantic image search: %w", err)
	}
	videoSemanticSearch, err := semantic.NewVideoSearchService(database, database, semanticTextSessions, nil)
	if err != nil {
		return nil, fmt.Errorf("construct semantic video search: %w", err)
	}
	idempotentTagReviews, err := semantic.NewIdempotentTagReviewService(tagReviews, database, nil)
	if err != nil {
		return nil, fmt.Errorf("construct idempotent tag review service: %w", err)
	}
	aiOrphans, err := aimodel.NewManagedOrphanService(aiModels, aiCatalog, managedModels, runtime.GOARCH)
	if err != nil {
		return nil, fmt.Errorf("construct managed AI model orphan service: %w", err)
	}
	aiWorkerComponent, err := newAIWorkerComponent(
		[]aiBackgroundWorker{aiInstallWorker, aiActivationWorker, semanticSessions, semanticTextSessions, semanticWorker, semanticClearWorker, tagReviewClearWorker, videoSemanticWorker}, aiOperations, aiOrphans, managedModels,
	)
	if err != nil {
		return nil, err
	}
	aiManagement, err := aimodel.NewManagementService(aiModels, aiScanner, aiOperations, aiAdmission, aiActivationAdmission, aiAvailability,
		aimodel.OperationCancellers{Semantic: semanticService})
	if err != nil {
		return nil, fmt.Errorf("construct AI model management service: %w", err)
	}
	routeDependencies := api.RouteDependencies{
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
		MediaDiagnostics: mediaDiagnostics,
		SystemLogs:       systemLogs,
		ReleaseInfo:      releaseInformation,
		Settings:         settingsService,
		Catalog:          catalogService,
		Curation:         curationService,
		Thumbnails:       thumbnailDelivery,
		Content:          contentService,
		ContentAdmission: resourceController,
		AIModels:         aiManagement,
		Semantic:         semanticService,
		SemanticSearch:   semanticSearch,

		AITagVocabulary:   tagVocabulary,
		AITagSuggestions:  tagSuggestionList,
		AITagReviews:      idempotentTagReviews,
		AITagReviewClear:  semanticService,
		VideoSemanticJobs: semanticService,
	}
	routeDependencies.VideoSemanticSearch = videoSemanticSearch
	routes, err := api.NewRoutes(routeDependencies)
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
				SystemEvents:         systemLogs,
			},
		),
		logger,
	)
	application, err := newApplication(
		[]component{
			databaseComponent,
			newSystemEventComponent(systemLogs),
			resourceLimitsComponent,
			mediaRootComponent,
			aiSourceComponent,
			managedModelsComponent,
			aiAvailabilityComponent,
			imageRuntimeComponent,
			aiWorkerComponent,
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
