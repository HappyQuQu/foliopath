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
	"github.com/HappyQuQu/foliopath/internal/aimodel/downloadhttp"
	"github.com/HappyQuQu/foliopath/internal/api"
	"github.com/HappyQuQu/foliopath/internal/auth"
	"github.com/HappyQuQu/foliopath/internal/autotag"
	"github.com/HappyQuQu/foliopath/internal/catalog"
	"github.com/HappyQuQu/foliopath/internal/curation"
	"github.com/HappyQuQu/foliopath/internal/deletion"
	"github.com/HappyQuQu/foliopath/internal/face"
	"github.com/HappyQuQu/foliopath/internal/files"
	"github.com/HappyQuQu/foliopath/internal/inference/onnx"
	"github.com/HappyQuQu/foliopath/internal/inference/remoteml"
	"github.com/HappyQuQu/foliopath/internal/inference/sentencepiece"
	"github.com/HappyQuQu/foliopath/internal/jobs"
	"github.com/HappyQuQu/foliopath/internal/library"
	"github.com/HappyQuQu/foliopath/internal/media"
	"github.com/HappyQuQu/foliopath/internal/media/imagevips"
	"github.com/HappyQuQu/foliopath/internal/media/videoffmpeg"
	"github.com/HappyQuQu/foliopath/internal/ocr"
	"github.com/HappyQuQu/foliopath/internal/releaseinfo"
	releasegithub "github.com/HappyQuQu/foliopath/internal/releaseinfo/github"
	"github.com/HappyQuQu/foliopath/internal/resourcecontrol"
	"github.com/HappyQuQu/foliopath/internal/scanner"
	"github.com/HappyQuQu/foliopath/internal/search"
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

	signalContext, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	go func() {
		select {
		case <-signalContext.Done():
			cancel(jobs.ErrPlannedShutdown)
		case <-ctx.Done():
		}
	}()

	return run(ctx, input)
}

func run(ctx context.Context, input Input) error {
	if ctx == nil {
		return errors.New("application context is nil")
	}
	if len(input.Args) == 1 && input.Args[0] == "database-check" {
		return checkDatabase(ctx)
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
	if err := directorySource.ConfigureArchiveTemp(filepath.Join(configuration.dataRoot, "tmp", "zip")); err != nil {
		return nil, fmt.Errorf("construct archive temporary storage: %w", err)
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
	mutationSource, err := newMediaMutationSource(directorySource, resourceController, configuration.mediaMutations)
	if err != nil {
		return nil, err
	}
	deletionSignal := jobs.NewSignal()
	deletionService, err := deletion.NewService(database, mutationSource, deletionSignal, nil, nil)
	if err != nil {
		return nil, err
	}
	deletionProcessor, err := resourcecontrol.LimitBackground(resourceController, deletion.Processor{Repository: database, Source: mutationSource})
	if err != nil {
		return nil, err
	}
	deletionQueue := deletion.MaintainedQueue{Repository: database, Source: mutationSource, Reconciler: deletionReconciler{admission: scanAdmission, queries: scanQueries}}
	deletionWorker, err := jobs.NewWorkerPool(deletionQueue, deletionProcessor, deletionSignal, jobs.WorkerOptions{Workers: 1})
	if err != nil {
		return nil, err
	}
	deletionComponent := newDeletionWorkerComponent(deletionWorker)
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
	downloadConfig, err := files.ReadModelDownloadCatalog(filepath.Join(configuration.dataRoot, "model-downloads.json"))
	if err != nil {
		return nil, fmt.Errorf("read model download configuration: %w", err)
	}
	downloadCatalog, err := aimodel.ParseDownloadCatalog(downloadConfig, runtime.GOARCH)
	if err != nil {
		return nil, fmt.Errorf("construct reviewed AI model catalog: %w", err)
	}
	aiCatalog := downloadCatalog.Packages()
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
	aiRemote, err := aimodel.NewProxySettingsService(database, downloadhttp.ProxySettingsTransport{}, configuration.modelDownloadProxy)
	if err != nil {
		return nil, fmt.Errorf("construct model download transport: %w", err)
	}
	aiDownloadSource := &aimodel.DownloadSource{Catalog: downloadCatalog, Remote: aiRemote, Offline: aiSource}
	aiInstaller, err := aimodel.NewInstaller(aiModels, aiDownloadSource, managedModels)
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
	var preparation aimodel.PreparationReader
	var deployment *deploymentBootstrap
	var faceDeployment *deploymentBootstrap
	var faceService *face.ControlService
	var faceWorkers []aiBackgroundWorker
	var ocrWorkers []aiBackgroundWorker
	var ocrService *ocr.Service
	var remoteEncoder *semantic.BoundEncoder
	var textEncoder semantic.TextEncoder = semanticTextSessions
	automaticRuntime := &automaticTagRuntime{content: contentService, preprocessor: imagevips.New()}
	if configuration.deployment != nil {
		config := *configuration.deployment
		client, createErr := remoteml.NewConfigured(config.Origin, config.Identity)
		if createErr != nil {
			return nil, createErr
		}
		if config.Managed {
			preparation = client
		}
		activationRuntime, createErr := newDeploymentActivationRuntime(client, database, config.space())
		if createErr != nil {
			return nil, createErr
		}
		if createErr := aiActivationWorker.EnableDeploymentRuntime(activationRuntime); createErr != nil {
			return nil, createErr
		}
		remoteImages, createErr := imagevips.NewRemoteImagePreprocessor(config.ImagePreprocessing)
		if createErr != nil {
			return nil, createErr
		}
		remoteEncoder, createErr = semantic.NewBoundEncoder(database, remoteImages, client, config.space())
		if createErr != nil {
			return nil, createErr
		}
		textEncoder = remoteEncoder
		deployment = &deploymentBootstrap{client: client, models: aiModels, admission: aiActivationAdmission, config: config}
		if config.OCR != nil {
			ocrService, ocrWorkers, createErr = composeOCR(database, contentService, client, *config.OCR, resourceController)
			if createErr != nil {
				return nil, createErr
			}
		}
		if config.Tag != nil {
			tagImages, err := imagevips.NewRemoteImagePreprocessor(config.Tag.ImagePreprocessing)
			if err != nil {
				return nil, err
			}
			tagClient, err := client.BindTag(config.Tag.Identity)
			if err != nil {
				return nil, err
			}
			automaticRuntime.client = tagClient
			automaticRuntime.preprocessor = tagImages
			automaticRuntime.model = &autotag.Model{Identity: config.Tag.fingerprint(), DisplayName: config.Tag.Identity.Model, Protocol: config.Tag.Identity.Protocol}
		}
		if config.Face != nil {
			faceClient, err := client.BindFace(config.Face.Identity)
			if err != nil {
				return nil, err
			}
			faceRuntime, err := newFaceDeploymentActivationRuntime(faceClient, database, config.Face.space())
			if err != nil {
				return nil, err
			}
			if err := aiActivationWorker.EnableFaceDeploymentRuntime(faceRuntime); err != nil {
				return nil, err
			}
			faceDeployment = &deploymentBootstrap{client: client, models: aiModels, admission: aiActivationAdmission, config: config, faceClient: faceClient}
			analysisRuntime, err := remoteml.NewFaceRuntime(imagevips.New(), faceClient)
			if err != nil {
				return nil, err
			}
			analyzer, err := face.NewProcessor(faceContentSource{content: contentService}, analysisRuntime)
			if err != nil {
				return nil, err
			}
			processor, err := face.NewDeploymentJobProcessor(database, database, analyzer, database, database, config.Face.space(), nil, 64)
			if err != nil {
				return nil, err
			}
			limited, err := resourcecontrol.LimitBackground(resourceController, processor)
			if err != nil {
				return nil, err
			}
			faceSignal, clearSignal := jobs.NewSignal(), jobs.NewSignal()
			worker, err := jobs.NewWorkerPool(faceJobQueue{database}, limited, faceSignal, jobs.WorkerOptions{Workers: 1})
			if err != nil {
				return nil, err
			}
			clearProcessor, err := face.NewClearProcessor(database, nil, 0)
			if err != nil {
				return nil, err
			}
			limitedClear, err := resourcecontrol.LimitBackground(resourceController, clearProcessor)
			if err != nil {
				return nil, err
			}
			clearWorker, err := jobs.NewWorkerPool(faceClearQueue{database}, limitedClear, clearSignal, jobs.WorkerOptions{Workers: 1})
			if err != nil {
				return nil, err
			}
			settings, err := face.NewSettingsService(database, nil)
			if err != nil {
				return nil, err
			}
			admission, err := face.NewJobService(database, database, faceSignal, nil, nil)
			if err != nil {
				return nil, err
			}
			clears, err := face.NewClearService(database, clearSignal, nil, nil)
			if err != nil {
				return nil, err
			}
			faceService, err = face.NewControlService(settings, admission, clears, aiOperations)
			if err != nil {
				return nil, err
			}
			faceWorkers = []aiBackgroundWorker{worker, clearWorker}
		}
	}
	semanticProcessor, err := semantic.NewBackfillProcessor(
		database, semanticContentSource{content: contentService}, imagevips.New(), semanticSessions,
		database, database, nil, 0,
	)
	if remoteEncoder != nil {
		semanticProcessor, err = semantic.NewStreamBackfillProcessor(database, semanticContentSource{content: contentService}, remoteEncoder, database, database, nil, 0)
	}
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
	if remoteEncoder != nil {
		videoSemanticProcessor, err = semantic.NewStreamVideoProcessor(semanticStoryboardSource{
			repository: database, cache: cachePublisher, splitter: imagevips.New(), waker: mediaSignal,
		}, remoteEncoder, database, nil)
	}
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
	tagSignal := jobs.NewSignal()
	tagEmbeddings, err := semantic.NewControlledTagEmbeddingBuilder(database, textEncoder, nil, 0)
	if err != nil {
		return nil, err
	}
	tagPlans, err := semantic.NewControlledTagPlanBuilder(database, nil, nil, semantic.DefaultTagSuggestionThreshold)
	if err != nil {
		return nil, err
	}
	tagProcessor, err := semantic.NewTagJobProcessor(database, database, tagPlans, nil, tagEmbeddings)
	if err != nil {
		return nil, err
	}
	limitedTagProcessor, err := resourcecontrol.LimitBackground(resourceController, tagProcessor)
	if err != nil {
		return nil, err
	}
	tagWorker, err := jobs.NewWorkerPool(tagJobQueue{database: database}, limitedTagProcessor, tagSignal, jobs.WorkerOptions{Workers: 1})
	if err != nil {
		return nil, err
	}
	tagJobs, err := semantic.NewTagJobService(database, database, tagSignal, nil, nil)
	if err != nil {
		return nil, err
	}
	if err := semanticService.EnableTagJobs(tagJobs, tagVocabulary); err != nil {
		return nil, err
	}
	tagSuggestionList, err := semantic.NewTagSuggestionListService(database, nil)
	if err != nil {
		return nil, fmt.Errorf("construct tag suggestion list service: %w", err)
	}
	tagReviews, err := semantic.NewTagReviewService(database, curationService, nil)
	if err != nil {
		return nil, fmt.Errorf("construct tag review service: %w", err)
	}
	semanticSearch, err := semantic.NewSearchService(database, database, textEncoder, nil)
	if err != nil {
		return nil, fmt.Errorf("construct semantic image search: %w", err)
	}
	var ocrSearchFingerprint []string
	if configuration.deployment != nil && configuration.deployment.OCR != nil {
		fp, fpErr := configuration.deployment.OCR.Identity.Fingerprint()
		if fpErr != nil {
			return nil, fpErr
		}
		ocrSearchFingerprint = []string{fp}
	}
	fusedSearch, err := search.New(database, fusedTextEncoder{textEncoder}, nil, ocrSearchFingerprint...)
	if err != nil {
		return nil, fmt.Errorf("construct fused search: %w", err)
	}
	videoSemanticSearch, err := semantic.NewVideoSearchService(database, database, textEncoder, nil)
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
	aiWorkers := []aiBackgroundWorker{aiInstallWorker, aiActivationWorker, semanticSessions, semanticTextSessions, semanticWorker, semanticClearWorker, tagReviewClearWorker, videoSemanticWorker, tagWorker}
	automaticSignal := jobs.NewSignal()
	automaticService, err := autotag.New(database, automaticRuntime, automaticSignal)
	if err != nil {
		return nil, err
	}
	automaticProcessor, err := resourcecontrol.LimitBackground(resourceController, autotag.Processor{Repository: database, Analyzer: automaticRuntime})
	if err != nil {
		return nil, err
	}
	automaticWorker, err := jobs.NewWorkerPool(autotag.Queue{Repository: database}, automaticProcessor, automaticSignal, jobs.WorkerOptions{Workers: 1})
	if err != nil {
		return nil, err
	}
	aiWorkers = append(aiWorkers, automaticWorker, automaticService)
	if deployment != nil {
		aiWorkers = append(aiWorkers, deployment)
	}
	if faceDeployment != nil {
		aiWorkers = append(aiWorkers, faceDeployment)
	}
	aiWorkers = append(aiWorkers, faceWorkers...)
	aiWorkers = append(aiWorkers, ocrWorkers...)
	aiWorkerComponent, err := newAIWorkerComponent(
		aiWorkers, aiOperations, aiOrphans, managedModels,
	)
	if err != nil {
		return nil, err
	}
	cancellers := aimodel.OperationCancellers{Semantic: semanticService, AutomaticTags: automaticService}
	if faceService != nil {
		cancellers.Face = faceService
	}
	aiManagement, err := aimodel.NewManagementService(aiModels, aiScanner, aiOperations, aiAdmission, aiActivationAdmission, aiAvailability, cancellers)
	if err != nil {
		return nil, fmt.Errorf("construct AI model management service: %w", err)
	}
	routeDependencies := api.RouteDependencies{
		Readiness:        readiness.snapshot,
		Authentication:   authentication,
		Account:          authentication,
		Cache:            cacheManager,
		SystemStatus:     systemStatusProvider(input.Version, readiness, authentication),
		Deletion:         deletionService,
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
		Exif:             media.NewExifService(contentService, imagevips.New()),
		Content:          contentService,
		ContentAdmission: resourceController,
		AIModels:         aiManagement,
		AIPreparation:    preparation,
		Semantic:         semanticService,
		SemanticSearch:   semanticSearch,
		FusedSearch:      fusedSearch,

		AITagVocabulary:   tagVocabulary,
		AutomaticTags:     automaticService,
		AITagJobs:         semanticService,
		AITagSuggestions:  tagSuggestionList,
		AITagReviews:      idempotentTagReviews,
		AITagReviewClear:  semanticService,
		VideoSemanticJobs: semanticService,
	}
	if ocrService != nil {
		routeDependencies.OCR = ocrService
	}
	if faceService != nil {
		routeDependencies.FaceSettings = faceService
		routeDependencies.FaceJobs = faceService
		routeDependencies.FaceClears = faceService
		if err := composeFaceResults(&routeDependencies, database); err != nil {
			return nil, fmt.Errorf("construct face results: %w", err)
		}
	}
	aiDownloads, err := aimodel.NewDownloadService(downloadCatalog, aiModels, aiAdmission, database, aiRuntimeIncluded)
	if err != nil {
		return nil, fmt.Errorf("construct model downloads: %w", err)
	}
	routeDependencies.AIModelDownloads = aiDownloads
	routeDependencies.ModelProxy = aiRemote
	routeDependencies.VideoSemanticSearch = videoSemanticSearch
	videoStatus, err := semantic.NewVideoStatusService(database)
	if err != nil {
		return nil, err
	}
	routeDependencies.VideoSemanticStatus = videoStatus
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
			deletionComponent,
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
