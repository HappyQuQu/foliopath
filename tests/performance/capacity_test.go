package performance_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/HappyQuQu/foliopath/internal/catalog"
	"github.com/HappyQuQu/foliopath/internal/files"
	"github.com/HappyQuQu/foliopath/internal/library"
	"github.com/HappyQuQu/foliopath/internal/scanner"
	sqlitestore "github.com/HappyQuQu/foliopath/internal/store/sqlite"
	"github.com/HappyQuQu/foliopath/internal/thumbnail"
	_ "modernc.org/sqlite"
)

const (
	defaultDirectoryCount = 10_000
	defaultAssetCount     = 100_000
	defaultDeepChainDepth = 1_000
)

type capacityMetrics struct {
	GOOS                        string           `json:"goos"`
	GOARCH                      string           `json:"goarch"`
	GOMAXPROCS                  int              `json:"gomaxprocs"`
	DirectoryCount              int              `json:"directoryCount"`
	AssetCount                  int              `json:"assetCount"`
	FixtureMaxDepth             int              `json:"fixtureMaxDepth"`
	FixtureDeepBranches         int              `json:"fixtureDeepBranches"`
	FixtureAssetTargets         int              `json:"fixtureAssetTargets"`
	FixtureDurationMS           int64            `json:"fixtureDurationMs"`
	ScanDurationMS              int64            `json:"scanDurationMs"`
	ConcurrentReads             int              `json:"concurrentReads"`
	ConcurrentReadP95US         int64            `json:"concurrentReadP95Us"`
	ConcurrentReadMaxUS         int64            `json:"concurrentReadMaxUs"`
	ConcurrentSearches          int              `json:"concurrentSearches"`
	ConcurrentSearchP95US       int64            `json:"concurrentSearchP95Us"`
	ConcurrentSearchMaxUS       int64            `json:"concurrentSearchMaxUs"`
	DirectoryListP50US          int64            `json:"directoryListP50Us"`
	DirectoryListP95US          int64            `json:"directoryListP95Us"`
	AssetListP50US              int64            `json:"assetListP50Us"`
	AssetListP95US              int64            `json:"assetListP95Us"`
	RecursiveBrowseP50US        int64            `json:"recursiveBrowseP50Us"`
	RecursiveBrowseP95US        int64            `json:"recursiveBrowseP95Us"`
	FTSSearchP50US              int64            `json:"ftsSearchP50Us"`
	FTSSearchP95US              int64            `json:"ftsSearchP95Us"`
	ShortSearchP50US            int64            `json:"shortSearchP50Us"`
	ShortSearchP95US            int64            `json:"shortSearchP95Us"`
	GlobalSearchP95US           int64            `json:"globalSearchP95Us"`
	SearchKeysetP95US           int64            `json:"searchKeysetP95Us"`
	SearchFirstPageP95US        int64            `json:"searchFirstPageP95Us"`
	SearchSecondPageP95US       int64            `json:"searchSecondPageP95Us"`
	SearchCountP95US            int64            `json:"searchCountP95Us"`
	SearchListFirstP95US        int64            `json:"searchListFirstP95Us"`
	SearchListSecondP95US       int64            `json:"searchListSecondP95Us"`
	SearchCountPlan             []string         `json:"searchCountPlan,omitempty"`
	SearchListFirstPlan         []string         `json:"searchListFirstPlan,omitempty"`
	SearchListSecondPlan        []string         `json:"searchListSecondPlan,omitempty"`
	SearchOrderFirstP95US       int64            `json:"searchOrderFirstP95Us,omitempty"`
	SearchOrderSecondP95US      int64            `json:"searchOrderSecondP95Us,omitempty"`
	SearchOrderHydrateP95US     int64            `json:"searchOrderHydrateP95Us,omitempty"`
	SearchOrderSparseP95US      int64            `json:"searchOrderSparseP95Us,omitempty"`
	SearchOrderCancelUS         int64            `json:"searchOrderCancelUs,omitempty"`
	SearchOrderMatrixCases      int              `json:"searchOrderMatrixCases,omitempty"`
	SearchOrderMatrixCursor     int              `json:"searchOrderMatrixCursorCases,omitempty"`
	SearchOrderMatrixAnimated   int              `json:"searchOrderMatrixAnimatedCount,omitempty"`
	SearchOrderMatrixFirst      map[string]int64 `json:"searchOrderMatrixFirstPageUs,omitempty"`
	SearchOrderMatrixSecond     map[string]int64 `json:"searchOrderMatrixSecondPageUs,omitempty"`
	SearchOrderMatrixFirstP95   map[string]int64 `json:"searchOrderMatrixFirstPageP95Us,omitempty"`
	SearchOrderMatrixSecondP95  map[string]int64 `json:"searchOrderMatrixSecondPageP95Us,omitempty"`
	SearchRepoMatrixFirst       map[string]int64 `json:"searchRepositoryMatrixFirstPageUs,omitempty"`
	SearchRepoMatrixSecond      map[string]int64 `json:"searchRepositoryMatrixSecondPageUs,omitempty"`
	SearchModifiedFirstPlan     []string         `json:"searchModifiedWindowFirstPlan,omitempty"`
	SearchModifiedSecondPlan    []string         `json:"searchModifiedWindowSecondPlan,omitempty"`
	SearchModifiedCandidatePlan []string         `json:"searchModifiedWindowCandidatePlan,omitempty"`
	SearchOrderFirstPlan        []string         `json:"searchOrderFirstPlan,omitempty"`
	SearchOrderSecondPlan       []string         `json:"searchOrderSecondPlan,omitempty"`
	SearchOrderHydratePlan      []string         `json:"searchOrderHydratePlan,omitempty"`
	SearchOrderSparsePlan       []string         `json:"searchOrderSparsePlan,omitempty"`
	SearchCancelLatencyUS       int64            `json:"searchCancelLatencyUs"`
	SearchRebuildDurationMS     int64            `json:"searchRebuildDurationMs"`
	StoryboardVideoCount        int              `json:"storyboardVideoCount"`
	StoryboardAdmissionRuns     int              `json:"storyboardAdmissionRuns"`
	StoryboardAdmissionMax      int64            `json:"storyboardAdmissionMax"`
	StoryboardAdmissionMS       int64            `json:"storyboardAdmissionMs"`
	StoryboardBrowseP95US       int64            `json:"storyboardBrowseP95Us"`
	PeakGoHeapAllocBytes        uint64           `json:"peakGoHeapAllocBytes"`
	PeakRSSBytes                uint64           `json:"peakRssBytes,omitempty"`
	PeakRSSSource               string           `json:"peakRssSource,omitempty"`
	DatabaseAndWALSizeBytes     int64            `json:"databaseAndWalSizeBytes"`
	BudgetProfile               string           `json:"budgetProfile"`
	SearchBudgetProfile         string           `json:"searchBudgetProfile"`
	BudgetViolations            []string         `json:"budgetViolations"`
}

const (
	capacityBudgetProfile       = "stage0-comparable-v1"
	searchCapacityBudgetProfile = "s4-search-v1"
)

type capacityFixtureShape struct {
	maxDepth                int
	deepBranches            int
	assetTargets            int
	firstDeepBranchPaths    []string
	firstDeepTargetAssetCnt int
}

func TestCapacityBaseline(t *testing.T) {
	if os.Getenv("FOLIOPATH_CAPACITY") != "1" {
		t.Skip("set FOLIOPATH_CAPACITY=1 to run the explicit capacity spike")
	}

	directoryCount := positiveEnv(t, "FOLIOPATH_CAPACITY_DIRS", defaultDirectoryCount)
	assetCount := positiveEnv(t, "FOLIOPATH_CAPACITY_ASSETS", defaultAssetCount)
	base := t.TempDir()
	allowedPath := filepath.Join(base, "library")
	dataPath := filepath.Join(base, "data")
	if err := os.MkdirAll(allowedPath, 0o755); err != nil {
		t.Fatalf("create allowed media root: %v", err)
	}
	if err := os.MkdirAll(dataPath, 0o755); err != nil {
		t.Fatalf("create data directory: %v", err)
	}

	fixtureStarted := time.Now()
	fixtureShape := createCapacityFixture(t, allowedPath, directoryCount, assetCount)
	fixtureDuration := time.Since(fixtureStarted)

	root, err := files.OpenRoot(allowedPath)
	if err != nil {
		t.Fatalf("open media root: %v", err)
	}
	t.Cleanup(func() {
		if err := root.Close(); err != nil {
			t.Errorf("close media root: %v", err)
		}
	})

	databasePath := filepath.Join(dataPath, "foliopath.db")
	store, err := sqlitestore.Open(context.Background(), databasePath, sqlitestore.Options{
		BusyTimeout:        5 * time.Second,
		MaxOpenConnections: 4,
		MaxBatchSize:       500,
	})
	if err != nil {
		t.Fatalf("open SQLite store: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Errorf("close SQLite store: %v", err)
		}
	})

	inspector, err := sql.Open("sqlite", databasePath)
	if err != nil {
		t.Fatalf("open SQLite inspector: %v", err)
	}
	inspector.SetMaxOpenConns(1)
	t.Cleanup(func() {
		if err := inspector.Close(); err != nil {
			t.Errorf("close SQLite inspector: %v", err)
		}
	})
	if _, err := inspector.Exec("PRAGMA busy_timeout = 5000"); err != nil {
		t.Fatalf("set inspector busy timeout: %v", err)
	}

	libraries, err := library.NewService(store)
	if err != nil {
		t.Fatalf("create library service: %v", err)
	}
	created, err := libraries.Create(context.Background(), "Capacity", "")
	if err != nil {
		t.Fatalf("create capacity library: %v", err)
	}
	catalogService, err := catalog.NewService(
		store,
		[]byte("0123456789abcdef0123456789abcdef"),
	)
	if err != nil {
		t.Fatalf("create catalog service: %v", err)
	}
	scans, err := scanner.NewService(store, scanner.Config{
		BatchSize:       scanner.DefaultBatchSize,
		FinalizeTimeout: 10 * time.Second,
	})
	if err != nil {
		t.Fatalf("create scanner service: %v", err)
	}
	walker, err := files.NewScanWalker(root)
	if err != nil {
		t.Fatalf("create filesystem walker: %v", err)
	}

	runtime.GC()
	var initialMemory runtime.MemStats
	runtime.ReadMemStats(&initialMemory)
	var peakHeap atomic.Uint64
	peakHeap.Store(initialMemory.Alloc)
	stopMemorySampling := make(chan struct{})
	memorySamplingDone := make(chan struct{})
	go sampleHeap(stopMemorySampling, memorySamplingDone, &peakHeap)

	type scanResult struct {
		run scanner.ScanRun
		err error
	}
	scanResults := make(chan scanResult, 1)
	scanContext, cancelScan := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancelScan()
	scanStarted := time.Now()
	go func() {
		run, scanErr := scans.RunFullScan(scanContext, scanner.FullScanRequest{
			LibraryID: created.ID,
			Trigger:   scanner.TriggerCreation,
			Walker:    walker,
		})
		scanResults <- scanResult{run: run, err: scanErr}
	}()

	readLatencies := make([]time.Duration, 0, 1024)
	searchLatencies := make([]time.Duration, 0, 1024)
	readTicker := time.NewTicker(25 * time.Millisecond)
	defer readTicker.Stop()
	var result scanResult
	var concurrentReadErr error
scanLoop:
	for {
		select {
		case result = <-scanResults:
			break scanLoop
		case <-readTicker.C:
			started := time.Now()
			if err := readAssetPage(context.Background(), inspector, created.ID); err != nil {
				concurrentReadErr = err
				cancelScan()
				result = <-scanResults
				break scanLoop
			}
			readLatencies = append(readLatencies, time.Since(started))
			started = time.Now()
			if err := readLibrarySearch(
				context.Background(), catalogService, created.ID, "asset-099",
				catalog.SortModifiedAt,
			); err != nil {
				concurrentReadErr = err
				cancelScan()
				result = <-scanResults
				break scanLoop
			}
			searchLatencies = append(searchLatencies, time.Since(started))
		}
	}
	scanDuration := time.Since(scanStarted)
	close(stopMemorySampling)
	<-memorySamplingDone

	if concurrentReadErr != nil {
		t.Fatalf("read assets while scan is active: %v; scan termination: %v", concurrentReadErr, result.err)
	}
	if result.err != nil {
		t.Fatalf("run full scan: %v", result.err)
	}
	if result.run.Status != scanner.RunStatusSucceeded {
		t.Fatalf("scan status = %q, want %q", result.run.Status, scanner.RunStatusSucceeded)
	}
	if result.run.DiscoveredDirectories != int64(directoryCount+1) {
		t.Fatalf("discovered directories = %d, want %d", result.run.DiscoveredDirectories, directoryCount+1)
	}
	if result.run.DiscoveredAssets != int64(assetCount) {
		t.Fatalf("discovered assets = %d, want %d", result.run.DiscoveredAssets, assetCount)
	}

	assertCatalogCounts(t, inspector, created.ID, directoryCount+1, assetCount)
	assertDirectoryRollup(t, inspector, created.ID, assetCount, fixtureShape)
	directoryLatencies := measureQuery(t, 30, func(ctx context.Context) error {
		return readDirectoryPage(ctx, inspector, created.ID)
	})
	assetLatencies := measureQuery(t, 30, func(ctx context.Context) error {
		return readAssetPage(ctx, inspector, created.ID)
	})
	recursiveBrowseLatencies := measureQuery(t, 10, func(ctx context.Context) error {
		return readRecursiveBrowse(ctx, catalogService, created.ID)
	})
	ftsSearchLatencies := measureQuery(t, 30, func(ctx context.Context) error {
		return readLibrarySearch(
			ctx, catalogService, created.ID, "asset-099", catalog.SortModifiedAt,
		)
	})
	shortSearchLatencies := measureQuery(t, 30, func(ctx context.Context) error {
		return readLibrarySearch(ctx, catalogService, created.ID, "99", catalog.SortModifiedAt)
	})
	globalSearchLatencies := measureQuery(t, 30, func(ctx context.Context) error {
		return readGlobalSearch(ctx, catalogService, "asset-099", catalog.SortModifiedAt)
	})
	keysetLatencies := measureQuery(t, 20, func(ctx context.Context) error {
		return readSearchKeyset(ctx, catalogService, created.ID)
	})
	keysetCursor, err := readSearchFirstPage(context.Background(), catalogService, created.ID)
	if err != nil {
		t.Fatalf("read diagnostic search first page: %v", err)
	}
	firstPageLatencies := measureQuery(t, 20, func(ctx context.Context) error {
		_, queryErr := readSearchFirstPage(ctx, catalogService, created.ID)
		return queryErr
	})
	secondPageLatencies := measureQuery(t, 20, func(ctx context.Context) error {
		return readSearchSecondPage(ctx, catalogService, created.ID, keysetCursor)
	})
	searchCountLatencies, searchListFirstLatencies, searchListSecondLatencies, searchAfter,
		searchFirstIDs, searchSecondIDs :=
		measureSearchRepositoryComponents(t, store, created.ID)
	searchPlans := explainSearchRepositoryPlans(t, inspector, created.ID, searchAfter)
	orderFirst := measureOrderFirstSearch(
		t, store, inspector, created.ID, searchAfter, searchFirstIDs, searchSecondIDs,
	)
	orderFirstCancel := measureOrderFirstCancellation(t, inspector, created.ID)
	cancelLatency := measureActiveSearchCancellation(t, catalogService, created.ID)
	storyboardCapacity := measureStoryboardAdmissionCapacity(
		t,
		store,
		inspector,
		catalogService,
		created.ID,
		assetCount,
	)
	animatedMatrixCount := seedAnimatedSearchMatrix(t, inspector, created.ID, assetCount)
	// Run the filter matrix after storyboard seeding so image and video filters
	// both exercise non-empty result sets without adding a second synthetic
	// catalog mutation path.
	orderFirstMatrix := verifyOrderFirstSearchMatrix(t, store, inspector, created.ID)
	rebuildStarted := time.Now()
	if _, err := inspector.ExecContext(context.Background(),
		`INSERT INTO asset_search(asset_search) VALUES('rebuild')`,
	); err != nil {
		t.Fatalf("rebuild capacity search index: %v", err)
	}
	rebuildDuration := time.Since(rebuildStarted)
	if _, err := inspector.ExecContext(context.Background(), `
        INSERT INTO asset_search(asset_search, rank)
        VALUES('integrity-check', 1)`); err != nil {
		t.Fatalf("verify rebuilt capacity search index: %v", err)
	}
	if err := readLibrarySearch(
		context.Background(), catalogService, created.ID, "asset-099", catalog.SortModifiedAt,
	); err != nil {
		t.Fatalf("search rebuilt capacity index: %v", err)
	}
	if _, err := inspector.Exec("PRAGMA wal_checkpoint(TRUNCATE)"); err != nil {
		t.Fatalf("checkpoint SQLite WAL: %v", err)
	}

	peakRSS, peakRSSSource, rssErr := processPeakRSS()
	if rssErr != nil {
		t.Fatalf("read process peak RSS: %v", rssErr)
	}
	metrics := capacityMetrics{
		GOOS:                        runtime.GOOS,
		GOARCH:                      runtime.GOARCH,
		GOMAXPROCS:                  runtime.GOMAXPROCS(0),
		DirectoryCount:              directoryCount,
		AssetCount:                  assetCount,
		FixtureMaxDepth:             fixtureShape.maxDepth,
		FixtureDeepBranches:         fixtureShape.deepBranches,
		FixtureAssetTargets:         fixtureShape.assetTargets,
		FixtureDurationMS:           fixtureDuration.Milliseconds(),
		ScanDurationMS:              scanDuration.Milliseconds(),
		ConcurrentReads:             len(readLatencies),
		ConcurrentReadP95US:         percentile(readLatencies, 95).Microseconds(),
		ConcurrentReadMaxUS:         percentile(readLatencies, 100).Microseconds(),
		ConcurrentSearches:          len(searchLatencies),
		ConcurrentSearchP95US:       percentile(searchLatencies, 95).Microseconds(),
		ConcurrentSearchMaxUS:       percentile(searchLatencies, 100).Microseconds(),
		DirectoryListP50US:          percentile(directoryLatencies, 50).Microseconds(),
		DirectoryListP95US:          percentile(directoryLatencies, 95).Microseconds(),
		AssetListP50US:              percentile(assetLatencies, 50).Microseconds(),
		AssetListP95US:              percentile(assetLatencies, 95).Microseconds(),
		RecursiveBrowseP50US:        percentile(recursiveBrowseLatencies, 50).Microseconds(),
		RecursiveBrowseP95US:        percentile(recursiveBrowseLatencies, 95).Microseconds(),
		FTSSearchP50US:              percentile(ftsSearchLatencies, 50).Microseconds(),
		FTSSearchP95US:              percentile(ftsSearchLatencies, 95).Microseconds(),
		ShortSearchP50US:            percentile(shortSearchLatencies, 50).Microseconds(),
		ShortSearchP95US:            percentile(shortSearchLatencies, 95).Microseconds(),
		GlobalSearchP95US:           percentile(globalSearchLatencies, 95).Microseconds(),
		SearchKeysetP95US:           percentile(keysetLatencies, 95).Microseconds(),
		SearchFirstPageP95US:        percentile(firstPageLatencies, 95).Microseconds(),
		SearchSecondPageP95US:       percentile(secondPageLatencies, 95).Microseconds(),
		SearchCountP95US:            percentile(searchCountLatencies, 95).Microseconds(),
		SearchListFirstP95US:        percentile(searchListFirstLatencies, 95).Microseconds(),
		SearchListSecondP95US:       percentile(searchListSecondLatencies, 95).Microseconds(),
		SearchCountPlan:             searchPlans.count,
		SearchListFirstPlan:         searchPlans.first,
		SearchListSecondPlan:        searchPlans.second,
		SearchOrderFirstP95US:       percentile(orderFirst.firstLatencies, 95).Microseconds(),
		SearchOrderSecondP95US:      percentile(orderFirst.secondLatencies, 95).Microseconds(),
		SearchOrderHydrateP95US:     percentile(orderFirst.hydratedLatencies, 95).Microseconds(),
		SearchOrderSparseP95US:      percentile(orderFirst.sparseLatencies, 95).Microseconds(),
		SearchOrderCancelUS:         orderFirstCancel.Microseconds(),
		SearchOrderMatrixCases:      orderFirstMatrix.cases,
		SearchOrderMatrixCursor:     orderFirstMatrix.cursorCases,
		SearchOrderMatrixAnimated:   animatedMatrixCount,
		SearchOrderMatrixFirst:      orderFirstMatrix.firstPageMicroseconds,
		SearchOrderMatrixSecond:     orderFirstMatrix.secondPageMicroseconds,
		SearchOrderMatrixFirstP95:   orderFirstMatrix.firstPageP95Microseconds,
		SearchOrderMatrixSecondP95:  orderFirstMatrix.secondPageP95Microseconds,
		SearchRepoMatrixFirst:       orderFirstMatrix.repositoryFirstPageMicroseconds,
		SearchRepoMatrixSecond:      orderFirstMatrix.repositorySecondPageMicroseconds,
		SearchModifiedFirstPlan:     orderFirstMatrix.modifiedWindowFirstPlan,
		SearchModifiedSecondPlan:    orderFirstMatrix.modifiedWindowSecondPlan,
		SearchModifiedCandidatePlan: orderFirstMatrix.modifiedWindowCandidatePlan,
		SearchOrderFirstPlan:        orderFirst.firstPlan,
		SearchOrderSecondPlan:       orderFirst.secondPlan,
		SearchOrderHydratePlan:      orderFirst.hydratedPlan,
		SearchOrderSparsePlan:       orderFirst.sparsePlan,
		SearchCancelLatencyUS:       cancelLatency.Microseconds(),
		SearchRebuildDurationMS:     rebuildDuration.Milliseconds(),
		StoryboardVideoCount:        storyboardCapacity.videoCount,
		StoryboardAdmissionRuns:     storyboardCapacity.admissionRuns,
		StoryboardAdmissionMax:      storyboardCapacity.admissionMax,
		StoryboardAdmissionMS:       storyboardCapacity.admissionDuration.Milliseconds(),
		StoryboardBrowseP95US: percentile(
			storyboardCapacity.browseLatencies,
			95,
		).Microseconds(),
		PeakGoHeapAllocBytes:    peakHeap.Load(),
		PeakRSSBytes:            peakRSS,
		PeakRSSSource:           peakRSSSource,
		DatabaseAndWALSizeBytes: databaseFamilySize(t, databasePath),
		BudgetProfile:           capacityBudgetProfile,
		SearchBudgetProfile:     searchCapacityBudgetProfile,
	}
	metrics.BudgetViolations = capacityBudgetViolations(metrics)
	encoded, err := json.Marshal(metrics)
	if err != nil {
		t.Fatalf("encode capacity metrics: %v", err)
	}
	t.Logf("FOLIOPATH_CAPACITY_METRICS %s", encoded)
	if os.Getenv("FOLIOPATH_CAPACITY_ENFORCE_BUDGET") == "1" &&
		len(metrics.BudgetViolations) > 0 {
		t.Fatalf(
			"%s/%s budget violations: %v",
			capacityBudgetProfile,
			searchCapacityBudgetProfile,
			metrics.BudgetViolations,
		)
	}
}

func TestOrderFirstMultiLibraryGlobalMatrix(t *testing.T) {
	if os.Getenv("FOLIOPATH_SEARCH_MATRIX_MULTILIB") != "1" {
		t.Skip("set FOLIOPATH_SEARCH_MATRIX_MULTILIB=1 to run the multi-library search matrix")
	}
	directoriesPerLibrary := positiveEnv(t, "FOLIOPATH_SEARCH_MATRIX_DIRS_PER_LIBRARY", 5_000)
	assetsPerLibrary := positiveEnv(t, "FOLIOPATH_SEARCH_MATRIX_ASSETS_PER_LIBRARY", 50_000)
	base := t.TempDir()
	allowedPath := filepath.Join(base, "library")
	dataPath := filepath.Join(base, "data")
	if err := os.MkdirAll(allowedPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dataPath, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, relativeRoot := range []string{"library-a", "library-b"} {
		fixtureRoot := filepath.Join(allowedPath, relativeRoot)
		if err := os.Mkdir(fixtureRoot, 0o755); err != nil {
			t.Fatal(err)
		}
		createCapacityFixture(t, fixtureRoot, directoriesPerLibrary, assetsPerLibrary)
	}

	root, err := files.OpenRoot(allowedPath)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	databasePath := filepath.Join(dataPath, "foliopath.db")
	store, err := sqlitestore.Open(context.Background(), databasePath, sqlitestore.Options{
		BusyTimeout: 5 * time.Second, MaxOpenConnections: 4, MaxBatchSize: 500,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	libraries, err := library.NewService(store)
	if err != nil {
		t.Fatal(err)
	}
	scans, err := scanner.NewService(store, scanner.Config{
		BatchSize: scanner.DefaultBatchSize, FinalizeTimeout: 10 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	walker, err := files.NewScanWalker(root)
	if err != nil {
		t.Fatal(err)
	}
	createdLibraryIDs := make([]int64, 0, 2)
	for index, relativeRoot := range []string{"library-a", "library-b"} {
		created, createErr := libraries.Create(
			context.Background(), fmt.Sprintf("Matrix %c", 'A'+index), relativeRoot,
		)
		if createErr != nil {
			t.Fatal(createErr)
		}
		run, scanErr := scans.RunFullScan(context.Background(), scanner.FullScanRequest{
			LibraryID: created.ID, Trigger: scanner.TriggerCreation, Walker: walker,
		})
		if scanErr != nil {
			t.Fatal(scanErr)
		}
		if run.Status != scanner.RunStatusSucceeded || run.DiscoveredAssets != int64(assetsPerLibrary) ||
			run.DiscoveredDirectories != int64(directoriesPerLibrary+1) {
			t.Fatalf("library %q scan = %#v", relativeRoot, run)
		}
		createdLibraryIDs = append(createdLibraryIDs, created.ID)
	}

	inspector, err := sql.Open("sqlite", databasePath)
	if err != nil {
		t.Fatal(err)
	}
	defer inspector.Close()
	inspector.SetMaxOpenConns(1)
	seedMultiLibrarySearchMatrix(t, inspector, createdLibraryIDs, assetsPerLibrary)
	seedMultiLibrarySearchTextFixtures(t, inspector, createdLibraryIDs[0])
	revision, err := store.ResolveGlobalCatalogRevision(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	terms, err := catalog.NormalizeSearchTerms("asset")
	if err != nil {
		t.Fatal(err)
	}
	type multiLibraryMatrixCase struct {
		name  string
		query catalog.AssetQuery
	}
	cases := make([]multiLibraryMatrixCase, 0, 17)
	for _, sortField := range []catalog.SortField{catalog.SortName, catalog.SortModifiedAt, catalog.SortSize} {
		for _, order := range []catalog.SortOrder{catalog.OrderAsc, catalog.OrderDesc} {
			cases = append(cases, multiLibraryMatrixCase{
				name: string(sortField) + "/" + string(order),
				query: catalog.AssetQuery{
					ScopeKind: catalog.ScopeGlobal, CatalogRevision: revision,
					SearchTerms: terms, Sort: sortField, Order: order,
				},
			})
		}
	}
	for _, kindCase := range []struct {
		name  string
		kinds []catalog.AssetKind
		order catalog.SortOrder
	}{
		{name: "kind-video/name/asc", kinds: []catalog.AssetKind{catalog.KindVideo}, order: catalog.OrderAsc},
		{name: "kind-animated/name/asc", kinds: []catalog.AssetKind{catalog.KindAnimated}, order: catalog.OrderAsc},
		{name: "kind-image-video/name/desc", kinds: []catalog.AssetKind{catalog.KindImage, catalog.KindVideo}, order: catalog.OrderDesc},
	} {
		cases = append(cases, multiLibraryMatrixCase{
			name: kindCase.name,
			query: catalog.AssetQuery{
				ScopeKind: catalog.ScopeGlobal, CatalogRevision: revision,
				SearchTerms: terms, Kinds: kindCase.kinds,
				Sort: catalog.SortName, Order: kindCase.order,
			},
		})
	}
	sparsePrefix := fmt.Sprintf("asset-%03d", (assetsPerLibrary-1)/1_000)
	sparseTerms, err := catalog.NormalizeSearchTerms(sparsePrefix)
	if err != nil {
		t.Fatal(err)
	}
	cases = append(cases, multiLibraryMatrixCase{
		name: "sparse/name/asc",
		query: catalog.AssetQuery{
			ScopeKind: catalog.ScopeGlobal, CatalogRevision: revision,
			SearchTerms: sparseTerms, Sort: catalog.SortName, Order: catalog.OrderAsc,
		},
	})
	var minimumModified, maximumModified int64
	if err := inspector.QueryRow(`SELECT MIN(mtime_ns), MAX(mtime_ns) FROM assets`).Scan(
		&minimumModified, &maximumModified,
	); err != nil {
		t.Fatal(err)
	}
	modifiedFrom := minimumModified + (maximumModified-minimumModified)/4
	modifiedBefore := minimumModified + 3*(maximumModified-minimumModified)/4
	cases = append(cases, multiLibraryMatrixCase{
		name: "selective-date/modifiedAt/asc",
		query: catalog.AssetQuery{
			ScopeKind: catalog.ScopeGlobal, CatalogRevision: revision,
			SearchTerms: terms, Sort: catalog.SortModifiedAt, Order: catalog.OrderAsc,
			ModifiedFromNS: &modifiedFrom, ModifiedBeforeNS: &modifiedBefore,
		},
	})
	libraryScope, err := store.ResolveScope(context.Background(), createdLibraryIDs[0], 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, textCase := range []struct {
		name     string
		rawQuery string
	}{
		{name: "text/chinese-two-rune", rawQuery: "上海"},
		{name: "text/combining-normalized", rawQuery: "café"},
		{name: "text/sharp-s-fold", rawQuery: "STRASSE"},
		{name: "text/punctuation", rawQuery: "hello-world,"},
		{name: "text/multi-word-and", rawQuery: "red blue"},
		{name: "text/quoted-fallback", rawQuery: `"quote"`},
	} {
		searchTerms, normalizeErr := catalog.NormalizeSearchTerms(textCase.rawQuery)
		if normalizeErr != nil {
			t.Fatalf("normalize %s: %v", textCase.name, normalizeErr)
		}
		cases = append(cases, multiLibraryMatrixCase{
			name: textCase.name,
			query: catalog.AssetQuery{
				Scope: libraryScope, ScopeKind: catalog.ScopeLibrary,
				SearchTerms: searchTerms, Sort: catalog.SortName, Order: catalog.OrderAsc,
			},
		})
	}

	firstPageMicroseconds := make(map[string]int64, len(cases))
	secondPageMicroseconds := make(map[string]int64, len(cases))
	hydratedPages := 0
	secondPages := 0
	for _, matrixCase := range cases {
		t.Run(matrixCase.name, func(t *testing.T) {
			query := matrixCase.query
			first, listErr := store.ListAssetPage(context.Background(), catalog.AssetListParams{
				Query: query, Limit: 101,
			})
			if listErr != nil {
				t.Fatal(listErr)
			}
			candidateQuery, candidateArgs := orderFirstMatrixQuery(query, nil, 101)
			started := time.Now()
			candidateIDs := queryAssetIDs(t, inspector, candidateQuery, candidateArgs...)
			firstPageMicroseconds[matrixCase.name] = time.Since(started).Microseconds()
			if !equalInt64s(candidateIDs, assetIDs(first)) {
				t.Fatal("multi-library candidate first page differs")
			}
			hydratedQuery, hydratedArgs := orderFirstHydratedMatrixQuery(query, nil, 101)
			if hydratedIDs := queryHydratedAssetIDs(t, inspector, hydratedQuery, hydratedArgs...); !equalInt64s(
				hydratedIDs, assetIDs(first),
			) {
				t.Fatal("multi-library hydrated candidate first page differs")
			}
			hydratedPages++
			if len(first) != 101 {
				if strings.HasPrefix(matrixCase.name, "text/") && len(first) == 1 {
					return
				}
				t.Fatalf("multi-library first page = %d, want 101", len(first))
			}
			after := assetPosition(first[99])
			second, secondErr := store.ListAssetPage(context.Background(), catalog.AssetListParams{
				Query: query, After: &after, Limit: 101,
			})
			if secondErr != nil {
				t.Fatal(secondErr)
			}
			candidateQuery, candidateArgs = orderFirstMatrixQuery(query, &after, 101)
			started = time.Now()
			candidateIDs = queryAssetIDs(t, inspector, candidateQuery, candidateArgs...)
			secondPageMicroseconds[matrixCase.name] = time.Since(started).Microseconds()
			if !equalInt64s(candidateIDs, assetIDs(second)) {
				t.Fatal("multi-library candidate second page differs")
			}
			hydratedQuery, hydratedArgs = orderFirstHydratedMatrixQuery(query, &after, 101)
			if hydratedIDs := queryHydratedAssetIDs(t, inspector, hydratedQuery, hydratedArgs...); !equalInt64s(
				hydratedIDs, assetIDs(second),
			) {
				t.Fatal("multi-library hydrated candidate second page differs")
			}
			hydratedPages++
			secondPages++
		})
	}

	type cardinalityCase struct {
		name      string
		rawQuery  string
		wantCount int
	}
	cardinalityCases := []cardinalityCase{
		{name: "cardinality/empty", rawQuery: "definitely-absent", wantCount: 0},
		{name: "cardinality/one", rawQuery: "asset-049999", wantCount: 1},
		{name: "cardinality/under-page", rawQuery: "asset-04999", wantCount: 10},
		{name: "cardinality/exact-page", rawQuery: "asset-0499", wantCount: 100},
		{name: "cardinality/over-page", rawQuery: "asset-049", wantCount: 101},
	}
	cardinalityCounts := make(map[string]int, len(cardinalityCases))
	for _, matrixCase := range cardinalityCases {
		t.Run(matrixCase.name, func(t *testing.T) {
			searchTerms, normalizeErr := catalog.NormalizeSearchTerms(matrixCase.rawQuery)
			if normalizeErr != nil {
				t.Fatal(normalizeErr)
			}
			query := catalog.AssetQuery{
				Scope: libraryScope, ScopeKind: catalog.ScopeLibrary,
				SearchTerms: searchTerms, Sort: catalog.SortName, Order: catalog.OrderAsc,
			}
			first, listErr := store.ListAssetPage(context.Background(), catalog.AssetListParams{Query: query, Limit: 101})
			if listErr != nil {
				t.Fatal(listErr)
			}
			candidateQuery, candidateArgs := orderFirstMatrixQuery(query, nil, 101)
			candidateIDs := queryAssetIDs(t, inspector, candidateQuery, candidateArgs...)
			if !equalInt64s(candidateIDs, assetIDs(first)) {
				t.Fatal("cardinality candidate first page differs")
			}
			if len(first) != matrixCase.wantCount {
				t.Fatalf("cardinality first page = %d, want %d", len(first), matrixCase.wantCount)
			}
			cardinalityCounts[matrixCase.name] = len(first)
			if len(first) <= 100 {
				return
			}
			after := assetPosition(first[99])
			second, secondErr := store.ListAssetPage(context.Background(), catalog.AssetListParams{
				Query: query, After: &after, Limit: 101,
			})
			if secondErr != nil {
				t.Fatal(secondErr)
			}
			candidateQuery, candidateArgs = orderFirstMatrixQuery(query, &after, 101)
			candidateIDs = queryAssetIDs(t, inspector, candidateQuery, candidateArgs...)
			if !equalInt64s(candidateIDs, assetIDs(second)) {
				t.Fatal("cardinality candidate second page differs")
			}
		})
	}

	type concurrentScanResult struct {
		status scanner.RunStatus
		err    error
	}
	scanDone := make(chan concurrentScanResult, 1)
	go func() {
		run, scanErr := scans.RunFullScan(context.Background(), scanner.FullScanRequest{
			LibraryID: createdLibraryIDs[0], Trigger: scanner.TriggerManual, Walker: walker,
		})
		scanDone <- concurrentScanResult{status: run.Status, err: scanErr}
	}()
	concurrentQuery := cases[9].query
	concurrentComparisons := 0
	for concurrentComparisons < 10 {
		production, listErr := store.ListAssetPage(context.Background(), catalog.AssetListParams{
			Query: concurrentQuery, Limit: 101,
		})
		if listErr != nil {
			t.Fatalf("list production page during concurrent scan: %v", listErr)
		}
		candidateQuery, candidateArgs := orderFirstHydratedMatrixQuery(concurrentQuery, nil, 101)
		if candidateIDs := queryHydratedAssetIDs(t, inspector, candidateQuery, candidateArgs...); !equalInt64s(
			candidateIDs, assetIDs(production),
		) {
			t.Fatal("hydrated candidate differs during concurrent scan")
		}
		concurrentComparisons++
		time.Sleep(5 * time.Millisecond)
	}
	concurrentResult := <-scanDone
	if concurrentResult.err != nil || concurrentResult.status != scanner.RunStatusSucceeded {
		t.Fatalf("concurrent full scan = %s, %v", concurrentResult.status, concurrentResult.err)
	}
	productionAfterScan, err := store.ListAssetPage(context.Background(), catalog.AssetListParams{
		Query: concurrentQuery, Limit: 101,
	})
	if err != nil {
		t.Fatalf("list production page after concurrent scan: %v", err)
	}
	postScanQuery, postScanArgs := orderFirstHydratedMatrixQuery(concurrentQuery, nil, 101)
	if postScanIDs := queryHydratedAssetIDs(t, inspector, postScanQuery, postScanArgs...); !equalInt64s(
		postScanIDs, assetIDs(productionAfterScan),
	) {
		t.Fatal("hydrated candidate differs after concurrent scan")
	}

	representativeQuery := cases[0].query
	if _, err := inspector.ExecContext(context.Background(),
		`INSERT INTO asset_search(asset_search) VALUES('rebuild')`,
	); err != nil {
		t.Fatalf("rebuild multi-library search index: %v", err)
	}
	if _, err := inspector.ExecContext(context.Background(), `
        INSERT INTO asset_search(asset_search, rank)
        VALUES('integrity-check', 1)`); err != nil {
		t.Fatalf("verify rebuilt multi-library search index: %v", err)
	}
	if err := inspector.Close(); err != nil {
		t.Fatalf("close multi-library inspector before reopen: %v", err)
	}
	reopenedInspector, err := sql.Open("sqlite", databasePath)
	if err != nil {
		t.Fatalf("reopen multi-library inspector: %v", err)
	}
	defer reopenedInspector.Close()
	reopenedInspector.SetMaxOpenConns(1)
	reopenedProduction, err := store.ListAssetPage(context.Background(), catalog.AssetListParams{
		Query: representativeQuery, Limit: 101,
	})
	if err != nil {
		t.Fatalf("list production page after FTS rebuild/reopen: %v", err)
	}
	reopenedQuery, reopenedArgs := orderFirstHydratedMatrixQuery(representativeQuery, nil, 101)
	if reopenedIDs := queryHydratedAssetIDs(t, reopenedInspector, reopenedQuery, reopenedArgs...); !equalInt64s(
		reopenedIDs, assetIDs(reopenedProduction),
	) {
		t.Fatal("hydrated candidate differs after FTS rebuild/database reopen")
	}
	t.Logf("FOLIOPATH_SEARCH_MULTILIB_METRICS %s", mustJSON(t, map[string]any{
		"goos": runtime.GOOS, "goarch": runtime.GOARCH,
		"gomaxprocs": runtime.GOMAXPROCS(0), "libraries": 2,
		"directories": 2 * directoriesPerLibrary, "assets": 2 * assetsPerLibrary,
		"firstPageCases": len(cases), "secondPageCases": secondPages, "orderedIDsEqual": true,
		"hydratedPages": hydratedPages, "rebuildReopenEquivalent": true,
		"concurrentScanComparisons": concurrentComparisons, "postScanEquivalent": true,
		"candidateFirstPageUs":  firstPageMicroseconds,
		"candidateSecondPageUs": secondPageMicroseconds,
		"cardinalityCounts":     cardinalityCounts,
	}))
}

func seedMultiLibrarySearchTextFixtures(
	t *testing.T,
	database *sql.DB,
	libraryID int64,
) {
	t.Helper()
	type assetPath struct {
		id           int64
		relativePath string
	}
	rows, err := database.QueryContext(context.Background(), `
        SELECT id, relative_path
        FROM assets
        WHERE library_id = ?
        ORDER BY id
        LIMIT 6`, libraryID)
	if err != nil {
		t.Fatalf("select special-text search fixtures: %v", err)
	}
	assets := make([]assetPath, 0, 6)
	for rows.Next() {
		var asset assetPath
		if err := rows.Scan(&asset.id, &asset.relativePath); err != nil {
			rows.Close()
			t.Fatalf("scan special-text search fixture: %v", err)
		}
		assets = append(assets, asset)
	}
	if err := rows.Close(); err != nil {
		t.Fatalf("close special-text search fixture rows: %v", err)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate special-text search fixtures: %v", err)
	}
	if len(assets) != 6 {
		t.Fatalf("special-text search fixtures = %d, want 6", len(assets))
	}
	names := []string{
		"上海 cosplay.jpg",
		"Cafe\u0301 memory.jpg",
		"Straße walk.jpg",
		"hello-world, night.jpg",
		"red blue portrait.jpg",
		`"quote" memory.jpg`,
	}
	for index, asset := range assets {
		directory := path.Dir(asset.relativePath)
		if directory == "." {
			directory = ""
		}
		relativePath := names[index]
		if directory != "" {
			relativePath = directory + "/" + names[index]
		}
		if _, err := database.ExecContext(context.Background(), `
            UPDATE assets
            SET relative_path = ?, name = ?, natural_name_key = ?,
                search_name_key = ?, search_path_key = ?
            WHERE id = ? AND library_id = ?`,
			relativePath, names[index], catalog.NaturalNameKey(names[index]),
			catalog.SearchTextKey(names[index]), catalog.SearchTextKey(relativePath),
			asset.id, libraryID,
		); err != nil {
			t.Fatalf("update special-text search fixture %q: %v", names[index], err)
		}
	}
}

func seedMultiLibrarySearchMatrix(
	t *testing.T,
	database *sql.DB,
	libraryIDs []int64,
	assetsPerLibrary int,
) {
	t.Helper()
	if len(libraryIDs) != 2 {
		t.Fatalf("multi-library fixture IDs = %d, want 2", len(libraryIDs))
	}
	for libraryIndex, libraryID := range libraryIDs {
		videoCount := assetsPerLibrary / (5 + libraryIndex)
		animatedCount := assetsPerLibrary / (10 - 2*libraryIndex)
		if _, err := database.ExecContext(context.Background(), `
            UPDATE assets SET kind = 'video'
            WHERE id IN (
                SELECT id FROM assets WHERE library_id = ? ORDER BY id LIMIT ?
            )`, libraryID, videoCount); err != nil {
			t.Fatalf("seed multi-library video kinds: %v", err)
		}
		if _, err := database.ExecContext(context.Background(), `
            UPDATE assets SET kind = 'animated'
            WHERE id IN (
                SELECT id FROM assets
                WHERE library_id = ? AND kind = 'image'
                ORDER BY id DESC LIMIT ?
            )`, libraryID, animatedCount); err != nil {
			t.Fatalf("seed multi-library animated kinds: %v", err)
		}
	}
}

type storyboardCapacityMetrics struct {
	videoCount        int
	admissionRuns     int
	admissionMax      int64
	admissionDuration time.Duration
	browseLatencies   []time.Duration
}

func measureStoryboardAdmissionCapacity(
	t *testing.T,
	store *sqlitestore.Store,
	inspector *sql.DB,
	catalogService *catalog.Service,
	libraryID int64,
	assetCount int,
) storyboardCapacityMetrics {
	t.Helper()
	videoCount := assetCount / 10
	if videoCount < 1 {
		videoCount = 1
	}
	tx, err := inspector.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("begin storyboard capacity seed: %v", err)
	}
	rollback := true
	defer func() {
		if rollback {
			_ = tx.Rollback()
		}
	}()
	if _, err := tx.Exec(`
        UPDATE assets
        SET kind = 'video', media_format = 'mp4', mime_type = 'video/mp4',
            width = 1920, height = 1080, duration_ms = 10000,
            probe_status = 'ready', probe_error_code = NULL,
            playback_status = 'playable'
        WHERE id IN (
            SELECT id FROM assets
            WHERE library_id = ?
            ORDER BY id LIMIT ?
        )`,
		libraryID,
		videoCount,
	); err != nil {
		t.Fatalf("seed storyboard video metadata: %v", err)
	}
	if _, err := tx.Exec(`
        INSERT INTO thumbnails(
            library_id, asset_id, variant, source_fingerprint,
            transform_version, cache_rel_path, status, width, height,
            byte_size, created_at_ms, last_accessed_at_ms
        )
        SELECT library_id, id, 'grid', source_fingerprint, ?,
               'capacity/grid/' || id || '.webp', 'ready', 320, 180,
               1, 1, 1
        FROM assets
        WHERE library_id = ? AND kind = 'video'
        ORDER BY id LIMIT ?`,
		thumbnail.GridTransformVersion,
		libraryID,
		videoCount,
	); err != nil {
		t.Fatalf("seed storyboard grid state: %v", err)
	}
	if _, err := tx.Exec(`
        UPDATE media_jobs
        SET status = 'succeeded', finished_at_ms = 1
        WHERE variant = 'grid'
          AND asset_id IN (
              SELECT id FROM assets
              WHERE library_id = ? AND kind = 'video'
              ORDER BY id LIMIT ?
          )`,
		libraryID,
		videoCount,
	); err != nil {
		t.Fatalf("seed completed grid jobs: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit storyboard capacity seed: %v", err)
	}
	rollback = false

	type admissionResult struct {
		total    int64
		runs     int
		max      int64
		duration time.Duration
		err      error
	}
	resultChannel := make(chan admissionResult, 1)
	go func() {
		started := time.Now()
		var result admissionResult
		for {
			admitted, admitErr := store.AdmitStoryboardJobs(
				context.Background(),
				sqlitestore.MaxStoryboardAdmissionBatch,
			)
			if admitErr != nil {
				result.err = admitErr
				break
			}
			result.runs++
			if admitted > result.max {
				result.max = admitted
			}
			result.total += admitted
			if admitted == 0 {
				break
			}
		}
		result.duration = time.Since(started)
		resultChannel <- result
	}()

	browseLatencies := make([]time.Duration, 0, 32)
	var admission admissionResult
	for {
		select {
		case admission = <-resultChannel:
			goto admissionComplete
		default:
			started := time.Now()
			if err := readRecursiveBrowse(
				context.Background(),
				catalogService,
				libraryID,
			); err != nil {
				t.Fatalf("browse during storyboard admission: %v", err)
			}
			browseLatencies = append(
				browseLatencies,
				time.Since(started),
			)
		}
	}

admissionComplete:
	if admission.err != nil {
		t.Fatalf("admit storyboard capacity jobs: %v", admission.err)
	}
	if admission.total != int64(videoCount) ||
		admission.max > sqlitestore.MaxStoryboardAdmissionBatch {
		t.Fatalf(
			"storyboard admission total/max = %d/%d, want %d/<=%d",
			admission.total,
			admission.max,
			videoCount,
			sqlitestore.MaxStoryboardAdmissionBatch,
		)
	}
	var queued int
	if err := inspector.QueryRow(`
        SELECT count(*) FROM media_jobs
        WHERE library_id = ? AND variant = 'storyboard'`,
		libraryID,
	).Scan(&queued); err != nil {
		t.Fatalf("count storyboard capacity jobs: %v", err)
	}
	if queued != videoCount {
		t.Fatalf("storyboard capacity jobs = %d, want %d", queued, videoCount)
	}
	if assetCount > videoCount {
		job, found, err := store.ClaimNextMediaJob(
			context.Background(),
			time.Minute,
		)
		if err != nil {
			t.Fatalf("claim beside storyboard capacity queue: %v", err)
		}
		if !found || job.Variant != thumbnail.VariantGrid {
			t.Fatalf(
				"capacity priority claim = %#v, found %t; want grid",
				job,
				found,
			)
		}
	}
	return storyboardCapacityMetrics{
		videoCount:        videoCount,
		admissionRuns:     admission.runs,
		admissionMax:      admission.max,
		admissionDuration: admission.duration,
		browseLatencies:   browseLatencies,
	}
}

func TestDirectoryRollupDeepChainBaseline(t *testing.T) {
	if os.Getenv("FOLIOPATH_CAPACITY") != "1" {
		t.Skip("set FOLIOPATH_CAPACITY=1 to run the explicit capacity spike")
	}

	depth := positiveEnv(t, "FOLIOPATH_CAPACITY_DEEP_CHAIN", defaultDeepChainDepth)
	base := t.TempDir()
	databasePath := filepath.Join(base, "deep-rollup.db")
	store, err := sqlitestore.Open(context.Background(), databasePath, sqlitestore.Options{
		BusyTimeout:        5 * time.Second,
		MaxOpenConnections: 1,
		MaxBatchSize:       500,
	})
	if err != nil {
		t.Fatalf("open deep-rollup SQLite store: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Errorf("close deep-rollup SQLite store: %v", err)
		}
	})
	libraries, err := library.NewService(store)
	if err != nil {
		t.Fatal(err)
	}
	created, err := libraries.Create(context.Background(), "Deep rollup", "")
	if err != nil {
		t.Fatal(err)
	}
	run, err := store.BeginFullScan(context.Background(), created.ID, scanner.TriggerCreation)
	if err != nil {
		t.Fatal(err)
	}

	batch := make([]scanner.CatalogEntry, 0, 500)
	appendEntry := func(entry scanner.CatalogEntry) {
		batch = append(batch, entry)
		if len(batch) < cap(batch) {
			return
		}
		if err := store.UpsertCatalogBatch(context.Background(), run.ID, batch); err != nil {
			t.Fatalf("upsert deep-chain batch: %v", err)
		}
		batch = batch[:0]
	}
	appendEntry(scanner.CatalogEntry{Kind: scanner.CatalogEntryDirectory})
	appendEntry(capacityAssetEntry("root.jpg", "", "root.jpg"))
	parent := ""
	for index := 0; index < depth; index++ {
		name := fmt.Sprintf("d%04d", index)
		relativePath := name
		if parent != "" {
			relativePath = parent + "/" + name
		}
		appendEntry(scanner.CatalogEntry{
			Kind:         scanner.CatalogEntryDirectory,
			RelativePath: relativePath,
			ParentPath:   parent,
			Name:         name,
		})
		appendEntry(capacityAssetEntry(relativePath+"/asset.jpg", relativePath, "asset.jpg"))
		parent = relativePath
	}
	if len(batch) > 0 {
		if err := store.UpsertCatalogBatch(context.Background(), run.ID, batch); err != nil {
			t.Fatalf("upsert final deep-chain batch: %v", err)
		}
	}

	finalizeStarted := time.Now()
	completed, err := store.CompleteFullScan(context.Background(), run.ID, scanner.SkipCounts{})
	finalizeDuration := time.Since(finalizeStarted)
	if err != nil {
		t.Fatalf("finalize %d-level deep chain: %v", depth, err)
	}
	if completed.Status != scanner.RunStatusSucceeded {
		t.Fatalf("deep-chain scan status = %q, want succeeded", completed.Status)
	}

	inspector, err := sql.Open("sqlite", databasePath)
	if err != nil {
		t.Fatal(err)
	}
	defer inspector.Close()
	var rootDirect, rootRecursive, leafDirect, leafRecursive, directTotal int
	if err := inspector.QueryRow(`
		SELECT direct_asset_count, recursive_asset_count
		FROM directories
		WHERE library_id = ? AND relative_path = ''`, created.ID).
		Scan(&rootDirect, &rootRecursive); err != nil {
		t.Fatal(err)
	}
	if err := inspector.QueryRow(`
		SELECT direct_asset_count, recursive_asset_count
		FROM directories
		WHERE library_id = ? AND relative_path = ?`, created.ID, parent).
		Scan(&leafDirect, &leafRecursive); err != nil {
		t.Fatal(err)
	}
	if err := inspector.QueryRow(`
		SELECT sum(direct_asset_count)
		FROM directories
		WHERE library_id = ?`, created.ID).Scan(&directTotal); err != nil {
		t.Fatal(err)
	}
	if rootDirect != 1 || rootRecursive != depth+1 ||
		leafDirect != 1 || leafRecursive != 1 || directTotal != depth+1 {
		t.Fatalf(
			"deep-chain counts root=(%d,%d) leaf=(%d,%d) total=%d, want (1,%d), (1,1), %d",
			rootDirect, rootRecursive, leafDirect, leafRecursive, directTotal, depth+1, depth+1,
		)
	}
	t.Logf(
		"FOLIOPATH_DEEP_ROLLUP_METRICS %s",
		mustJSON(t, map[string]any{
			"goos":               runtime.GOOS,
			"goarch":             runtime.GOARCH,
			"gomaxprocs":         runtime.GOMAXPROCS(0),
			"depth":              depth,
			"directories":        depth + 1,
			"assets":             depth + 1,
			"finalizeDurationMs": finalizeDuration.Milliseconds(),
		}),
	)
}

func TestCapacityBudgetViolations(t *testing.T) {
	if violations := capacityBudgetViolations(capacityMetrics{
		ScanDurationMS:          120_000,
		ConcurrentReadP95US:     250_000,
		ConcurrentSearchP95US:   500_000,
		DirectoryListP95US:      100_000,
		AssetListP95US:          100_000,
		FTSSearchP95US:          250_000,
		ShortSearchP95US:        250_000,
		GlobalSearchP95US:       250_000,
		SearchKeysetP95US:       250_000,
		SearchCancelLatencyUS:   250_000,
		SearchRebuildDurationMS: 120_000,
		PeakRSSBytes:            1 << 30,
		DatabaseAndWALSizeBytes: 1 << 30,
	}); len(violations) != 0 {
		t.Fatalf("budget boundary violations = %v, want none", violations)
	}

	violations := capacityBudgetViolations(capacityMetrics{
		ScanDurationMS:          120_001,
		ConcurrentReadP95US:     250_001,
		ConcurrentSearchP95US:   500_001,
		DirectoryListP95US:      100_001,
		AssetListP95US:          100_001,
		FTSSearchP95US:          250_001,
		ShortSearchP95US:        250_001,
		GlobalSearchP95US:       250_001,
		SearchKeysetP95US:       250_001,
		SearchCancelLatencyUS:   250_001,
		SearchRebuildDurationMS: 120_001,
		PeakRSSBytes:            1<<30 + 1,
		DatabaseAndWALSizeBytes: 1<<30 + 1,
	})
	if len(violations) != 13 {
		t.Fatalf("budget violations = %v, want all thirteen limits", violations)
	}
}

func positiveEnv(t *testing.T, name string, fallback int) int {
	t.Helper()
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		t.Fatalf("%s must be a positive integer, got %q", name, value)
	}
	return parsed
}

func createCapacityFixture(
	t *testing.T,
	root string,
	directoryCount int,
	assetCount int,
) capacityFixtureShape {
	t.Helper()

	maxDepth := directoryCount / 100
	if maxDepth < 1 {
		maxDepth = 1
	}
	if maxDepth > 32 {
		maxDepth = 32
	}
	deepBranches := directoryCount / (maxDepth * 10)
	if deepBranches < 1 {
		deepBranches = 1
	}
	if deepBranches > 8 {
		deepBranches = 8
	}
	if deepBranches*maxDepth > directoryCount {
		deepBranches = directoryCount / maxDepth
	}
	if deepBranches < 1 {
		deepBranches = 1
	}

	assetTargets := make([]string, 0, directoryCount)
	firstDeepBranchPaths := make([]string, 0, maxDepth)
	for branch := 0; branch < deepBranches; branch++ {
		parent := ""
		for level := 0; level < maxDepth; level++ {
			name := fmt.Sprintf("deep-%02d-%02d", branch, level)
			relativePath := name
			if parent != "" {
				relativePath = filepath.Join(parent, name)
			}
			if err := os.Mkdir(filepath.Join(root, relativePath), 0o755); err != nil {
				t.Fatalf("create deep fixture directory %q: %v", relativePath, err)
			}
			if branch == 0 {
				firstDeepBranchPaths = append(firstDeepBranchPaths, relativePath)
			}
			parent = relativePath
		}
		assetTargets = append(assetTargets, parent)
	}

	remainingDirectories := directoryCount - deepBranches*maxDepth
	if remainingDirectories > 0 {
		groupCount := remainingDirectories / 100
		if groupCount < 1 {
			groupCount = 1
		}
		if groupCount > 100 {
			groupCount = 100
		}
		if groupCount > remainingDirectories {
			groupCount = remainingDirectories
		}
		groups := make([]string, 0, groupCount)
		for index := 0; index < groupCount; index++ {
			relativePath := fmt.Sprintf("group-%03d", index)
			if err := os.Mkdir(filepath.Join(root, relativePath), 0o755); err != nil {
				t.Fatalf("create fixture group %q: %v", relativePath, err)
			}
			groups = append(groups, relativePath)
		}
		for index := 0; index < remainingDirectories-groupCount; index++ {
			relativePath := filepath.Join(
				groups[index%len(groups)],
				fmt.Sprintf("directory-%05d", index),
			)
			if err := os.Mkdir(filepath.Join(root, relativePath), 0o755); err != nil {
				t.Fatalf("create fixture directory %q: %v", relativePath, err)
			}
			assetTargets = append(assetTargets, relativePath)
		}
		if remainingDirectories == groupCount {
			assetTargets = append(assetTargets, groups...)
		}
	}

	contents := []byte{0xff, 0xd8, 0xff, 0xd9}
	for index := 0; index < assetCount; index++ {
		filename := filepath.Join(
			root,
			assetTargets[index%len(assetTargets)],
			fmt.Sprintf("asset-%06d.jpg", index),
		)
		if err := os.WriteFile(filename, contents, 0o644); err != nil {
			t.Fatalf("create fixture asset %q: %v", filepath.Base(filename), err)
		}
	}
	return capacityFixtureShape{
		maxDepth:                maxDepth,
		deepBranches:            deepBranches,
		assetTargets:            len(assetTargets),
		firstDeepBranchPaths:    firstDeepBranchPaths,
		firstDeepTargetAssetCnt: (assetCount + len(assetTargets) - 1) / len(assetTargets),
	}
}

func sampleHeap(stop <-chan struct{}, done chan<- struct{}, peak *atomic.Uint64) {
	defer close(done)
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			var memory runtime.MemStats
			runtime.ReadMemStats(&memory)
			for current := peak.Load(); memory.Alloc > current; current = peak.Load() {
				if peak.CompareAndSwap(current, memory.Alloc) {
					break
				}
			}
		}
	}
}

func readAssetPage(ctx context.Context, database *sql.DB, libraryID int64) error {
	rows, err := database.QueryContext(ctx, `
		SELECT id, name
		FROM assets
		WHERE library_id = ?
		ORDER BY mtime_ns DESC, id DESC
		LIMIT 100`, libraryID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return err
		}
	}
	return rows.Err()
}

func readLibrarySearch(
	ctx context.Context,
	service *catalog.Service,
	libraryID int64,
	query string,
	sortField catalog.SortField,
) error {
	page, err := service.ListAssets(ctx, catalog.AssetRequest{
		LibraryID:   libraryID,
		SearchQuery: &query,
		Sort:        sortField,
		Limit:       100,
	})
	if err != nil {
		return err
	}
	if len(page.Items) > 100 {
		return fmt.Errorf("library search returned %d items, want at most 100", len(page.Items))
	}
	return nil
}

func readRecursiveBrowse(
	ctx context.Context,
	service *catalog.Service,
	libraryID int64,
) error {
	page, err := service.ListAssets(ctx, catalog.AssetRequest{
		LibraryID: libraryID,
		Recursive: true,
		Limit:     100,
	})
	if err != nil {
		return err
	}
	if len(page.Items) != 100 {
		return fmt.Errorf("recursive browse returned %d items, want 100", len(page.Items))
	}
	return nil
}

func readGlobalSearch(
	ctx context.Context,
	service *catalog.Service,
	query string,
	sortField catalog.SortField,
) error {
	page, err := service.SearchAssets(ctx, catalog.GlobalSearchRequest{
		SearchQuery: query,
		Sort:        sortField,
		Limit:       100,
	})
	if err != nil {
		return err
	}
	if len(page.Items) > 100 {
		return fmt.Errorf("global search returned %d items, want at most 100", len(page.Items))
	}
	return nil
}

func readSearchKeyset(
	ctx context.Context,
	service *catalog.Service,
	libraryID int64,
) error {
	query := "asset"
	first, err := service.ListAssets(ctx, catalog.AssetRequest{
		LibraryID: libraryID, SearchQuery: &query,
		Sort: catalog.SortName, Limit: 100,
	})
	if err != nil {
		return err
	}
	if len(first.Items) != 100 || first.NextCursor == "" {
		return fmt.Errorf(
			"first search keyset page has %d items and cursor %t",
			len(first.Items), first.NextCursor != "",
		)
	}
	second, err := service.ListAssets(ctx, catalog.AssetRequest{
		LibraryID: libraryID, SearchQuery: &query,
		Sort: catalog.SortName, Limit: 100, Cursor: first.NextCursor,
	})
	if err != nil {
		return err
	}
	if len(second.Items) != 100 || second.Items[0].ID == first.Items[0].ID {
		return fmt.Errorf("second search keyset page is not a distinct full page")
	}
	return nil
}

func readSearchFirstPage(
	ctx context.Context,
	service *catalog.Service,
	libraryID int64,
) (string, error) {
	query := "asset"
	page, err := service.ListAssets(ctx, catalog.AssetRequest{
		LibraryID: libraryID, SearchQuery: &query,
		Sort: catalog.SortName, Limit: 100,
	})
	if err != nil {
		return "", err
	}
	if len(page.Items) != 100 || page.NextCursor == "" {
		return "", fmt.Errorf(
			"diagnostic first search page has %d items and cursor %t",
			len(page.Items), page.NextCursor != "",
		)
	}
	return page.NextCursor, nil
}

func readSearchSecondPage(
	ctx context.Context,
	service *catalog.Service,
	libraryID int64,
	cursor string,
) error {
	query := "asset"
	page, err := service.ListAssets(ctx, catalog.AssetRequest{
		LibraryID: libraryID, SearchQuery: &query,
		Sort: catalog.SortName, Limit: 100, Cursor: cursor,
	})
	if err != nil {
		return err
	}
	if len(page.Items) != 100 {
		return fmt.Errorf("diagnostic second search page has %d items, want 100", len(page.Items))
	}
	return nil
}

func measureSearchRepositoryComponents(
	t *testing.T,
	store *sqlitestore.Store,
	libraryID int64,
) ([]time.Duration, []time.Duration, []time.Duration, catalog.AssetPosition, []int64, []int64) {
	t.Helper()
	ctx := context.Background()
	scope, err := store.ResolveScope(ctx, libraryID, 0)
	if err != nil {
		t.Fatalf("resolve diagnostic repository scope: %v", err)
	}
	terms, err := catalog.NormalizeSearchTerms("asset")
	if err != nil {
		t.Fatalf("normalize diagnostic repository search: %v", err)
	}
	query := catalog.AssetQuery{
		Scope: scope, ScopeKind: catalog.ScopeLibrary,
		SearchTerms: terms, Sort: catalog.SortName, Order: catalog.OrderAsc,
	}
	first, err := store.ListAssetPage(ctx, catalog.AssetListParams{Query: query, Limit: 101})
	if err != nil {
		t.Fatalf("seed diagnostic repository first page: %v", err)
	}
	if len(first) != 101 {
		t.Fatalf("diagnostic repository first page = %d items, want 101", len(first))
	}
	last := first[99]
	directoryPath := path.Dir(last.RelativePath)
	if directoryPath == "." {
		directoryPath = ""
	}
	after := catalog.AssetPosition{
		DirectoryPath: directoryPath, NaturalNameKey: last.NaturalNameKey,
		Name: last.Name, LibraryID: last.LibraryID, RelativePath: last.RelativePath,
		ModifiedAtNS: last.ModifiedAtNS, SizeBytes: last.SizeBytes, ID: last.ID,
	}
	countLatencies := measureQuery(t, 20, func(queryContext context.Context) error {
		_, queryErr := store.CountAssets(queryContext, query)
		return queryErr
	})
	firstLatencies := measureQuery(t, 20, func(queryContext context.Context) error {
		_, queryErr := store.ListAssetPage(queryContext, catalog.AssetListParams{
			Query: query, Limit: 101,
		})
		return queryErr
	})
	secondLatencies := measureQuery(t, 20, func(queryContext context.Context) error {
		_, queryErr := store.ListAssetPage(queryContext, catalog.AssetListParams{
			Query: query, After: &after, Limit: 101,
		})
		return queryErr
	})
	second, err := store.ListAssetPage(ctx, catalog.AssetListParams{
		Query: query, After: &after, Limit: 101,
	})
	if err != nil {
		t.Fatalf("seed diagnostic repository second page: %v", err)
	}
	if len(second) != 101 {
		t.Fatalf("diagnostic repository second page = %d items, want 101", len(second))
	}
	return countLatencies, firstLatencies, secondLatencies, after,
		assetIDs(first), assetIDs(second)
}

func assetIDs(items []catalog.Asset) []int64 {
	ids := make([]int64, len(items))
	for index := range items {
		ids[index] = items[index].ID
	}
	return ids
}

type orderFirstSearchMetrics struct {
	firstLatencies    []time.Duration
	secondLatencies   []time.Duration
	hydratedLatencies []time.Duration
	sparseLatencies   []time.Duration
	firstPlan         []string
	secondPlan        []string
	hydratedPlan      []string
	sparsePlan        []string
}

func measureOrderFirstSearch(
	t *testing.T,
	store *sqlitestore.Store,
	database *sql.DB,
	libraryID int64,
	after catalog.AssetPosition,
	wantFirst []int64,
	wantSecond []int64,
) orderFirstSearchMetrics {
	t.Helper()
	firstQuery, firstArgs := orderFirstSearchQuery(libraryID, nil, "asset", false)
	secondQuery, secondArgs := orderFirstSearchQuery(libraryID, &after, "asset", false)
	hydratedQuery, hydratedArgs := orderFirstSearchQuery(libraryID, nil, "asset", true)
	firstIDs := queryAssetIDs(t, database, firstQuery, firstArgs...)
	secondIDs := queryAssetIDs(t, database, secondQuery, secondArgs...)
	if !equalInt64s(firstIDs, wantFirst) {
		t.Fatalf("order-first first page IDs differ from repository result")
	}
	if !equalInt64s(secondIDs, wantSecond) {
		t.Fatalf("order-first second page IDs differ from repository result")
	}
	if hydratedIDs := queryHydratedAssetIDs(t, database, hydratedQuery, hydratedArgs...); !equalInt64s(hydratedIDs, wantFirst) {
		t.Fatalf("order-first hydrated first page IDs differ from repository result")
	}
	sparseTerms, err := catalog.NormalizeSearchTerms("asset-099")
	if err != nil {
		t.Fatalf("normalize diagnostic sparse search: %v", err)
	}
	scope, err := store.ResolveScope(context.Background(), libraryID, 0)
	if err != nil {
		t.Fatalf("resolve diagnostic sparse scope: %v", err)
	}
	sparseWant, err := store.ListAssetPage(context.Background(), catalog.AssetListParams{
		Query: catalog.AssetQuery{
			Scope: scope, ScopeKind: catalog.ScopeLibrary,
			SearchTerms: sparseTerms, Sort: catalog.SortName, Order: catalog.OrderAsc,
		},
		Limit: 101,
	})
	if err != nil {
		t.Fatalf("seed diagnostic sparse repository page: %v", err)
	}
	sparseQuery, sparseArgs := orderFirstSearchQuery(libraryID, nil, "asset-099", false)
	if sparseIDs := queryAssetIDs(t, database, sparseQuery, sparseArgs...); !equalInt64s(sparseIDs, assetIDs(sparseWant)) {
		t.Fatalf("order-first sparse page IDs differ from repository result")
	}
	return orderFirstSearchMetrics{
		firstLatencies: measureQuery(t, 20, func(ctx context.Context) error {
			return queryAssetIDsContext(ctx, database, firstQuery, firstArgs...)
		}),
		secondLatencies: measureQuery(t, 20, func(ctx context.Context) error {
			return queryAssetIDsContext(ctx, database, secondQuery, secondArgs...)
		}),
		hydratedLatencies: measureQuery(t, 20, func(ctx context.Context) error {
			return queryRowsContext(ctx, database, hydratedQuery, hydratedArgs...)
		}),
		sparseLatencies: measureQuery(t, 20, func(ctx context.Context) error {
			return queryAssetIDsContext(ctx, database, sparseQuery, sparseArgs...)
		}),
		firstPlan:    explainQueryPlan(t, database, firstQuery, firstArgs...),
		secondPlan:   explainQueryPlan(t, database, secondQuery, secondArgs...),
		hydratedPlan: explainQueryPlan(t, database, hydratedQuery, hydratedArgs...),
		sparsePlan:   explainQueryPlan(t, database, sparseQuery, sparseArgs...),
	}
}

func orderFirstSearchQuery(
	libraryID int64,
	after *catalog.AssetPosition,
	term string,
	hydrate bool,
) (string, []any) {
	const orderExpression = `CASE
              WHEN length(a.relative_path) = length(a.name) THEN ''
              ELSE substr(a.relative_path, 1, length(a.relative_path) - length(a.name) - 1)
            END`
	var builder strings.Builder
	if hydrate {
		builder.WriteString(`
        SELECT a.id, a.library_id, l.name, a.directory_id, a.relative_path, a.name,
               a.natural_name_key, a.kind, a.media_format, a.mime_type,
               a.size_bytes, a.mtime_ns, a.source_fingerprint,
               a.width, a.height, a.duration_ms, a.probe_status,
               a.probe_error_code, a.playback_status,
               t.status, t.error_code,
               storyboard.status, storyboard.error_code,
               storyboard.frame_count, storyboard.sprite_columns,
               storyboard.sprite_rows, storyboard.cell_width,
               storyboard.cell_height, l.status,
               EXISTS(SELECT 1 FROM asset_favorites favorite WHERE favorite.asset_id = a.id)
        FROM assets a INDEXED BY assets_browse_folder_name_v2
        JOIN libraries l ON l.id = a.library_id
        LEFT JOIN thumbnails t ON t.asset_id = a.id AND t.variant = 'grid'
        LEFT JOIN thumbnails storyboard
          ON storyboard.asset_id = a.id AND storyboard.variant = 'storyboard'`)
	} else {
		builder.WriteString(`
        SELECT a.id
        FROM assets a INDEXED BY assets_browse_folder_name_v2`)
	}
	builder.WriteString(`
        WHERE a.library_id = ?
          AND EXISTS (
            SELECT 1
            FROM asset_search
            WHERE asset_search.rowid = a.id
              AND asset_search MATCH ?
          )
          AND (instr(a.search_name_key, ?) > 0 OR instr(a.search_path_key, ?) > 0)`)
	args := []any{libraryID, `"` + term + `"`, term, term}
	if after != nil {
		builder.WriteString(`
          AND (` + orderExpression + ` > ?
            OR (` + orderExpression + ` = ? AND a.natural_name_key > ?)
            OR (` + orderExpression + ` = ? AND a.natural_name_key = ? AND a.name > ?)
            OR (` + orderExpression + ` = ? AND a.natural_name_key = ? AND a.name = ?
              AND a.relative_path > ?)
            OR (` + orderExpression + ` = ? AND a.natural_name_key = ? AND a.name = ?
              AND a.relative_path = ? AND a.id > ?)
          )`)
		args = append(args,
			after.DirectoryPath,
			after.DirectoryPath, after.NaturalNameKey,
			after.DirectoryPath, after.NaturalNameKey, after.Name,
			after.DirectoryPath, after.NaturalNameKey, after.Name, after.RelativePath,
			after.DirectoryPath, after.NaturalNameKey, after.Name, after.RelativePath, after.ID,
		)
	}
	builder.WriteString(`
        ORDER BY ` + orderExpression + ` ASC,
          a.natural_name_key ASC, a.name ASC, a.relative_path ASC, a.id ASC
        LIMIT ?`)
	args = append(args, 101)
	return builder.String(), args
}

func queryAssetIDs(t *testing.T, database *sql.DB, query string, args ...any) []int64 {
	t.Helper()
	rows, err := database.QueryContext(context.Background(), query, args...)
	if err != nil {
		t.Fatalf("query diagnostic asset IDs: %v", err)
	}
	defer rows.Close()
	ids := make([]int64, 0, 101)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan diagnostic asset ID: %v", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate diagnostic asset IDs: %v", err)
	}
	return ids
}

func queryAssetIDsContext(
	ctx context.Context,
	database *sql.DB,
	query string,
	args ...any,
) error {
	rows, err := database.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return err
		}
	}
	return rows.Err()
}

func queryHydratedAssetIDs(
	t *testing.T,
	database *sql.DB,
	query string,
	args ...any,
) []int64 {
	t.Helper()
	ids, err := queryDynamicRows(context.Background(), database, query, true, args...)
	if err != nil {
		t.Fatalf("query hydrated diagnostic assets: %v", err)
	}
	return ids
}

func queryRowsContext(
	ctx context.Context,
	database *sql.DB,
	query string,
	args ...any,
) error {
	_, err := queryDynamicRows(ctx, database, query, false, args...)
	return err
}

func queryDynamicRows(
	ctx context.Context,
	database *sql.DB,
	query string,
	collectIDs bool,
	args ...any,
) ([]int64, error) {
	rows, err := database.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	values := make([]any, len(columns))
	destinations := make([]any, len(columns))
	for index := range values {
		destinations[index] = &values[index]
	}
	ids := make([]int64, 0, 101)
	for rows.Next() {
		if err := rows.Scan(destinations...); err != nil {
			return nil, err
		}
		if collectIDs {
			id, ok := values[0].(int64)
			if !ok {
				return nil, fmt.Errorf("diagnostic hydrated asset ID has type %T", values[0])
			}
			ids = append(ids, id)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return ids, nil
}

func equalInt64s(left, right []int64) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

type orderFirstSearchMatrixMetrics struct {
	cases                            int
	cursorCases                      int
	firstPageMicroseconds            map[string]int64
	secondPageMicroseconds           map[string]int64
	firstPageP95Microseconds         map[string]int64
	secondPageP95Microseconds        map[string]int64
	repositoryFirstPageMicroseconds  map[string]int64
	repositorySecondPageMicroseconds map[string]int64
	modifiedWindowFirstPlan          []string
	modifiedWindowSecondPlan         []string
	modifiedWindowCandidatePlan      []string
}

func seedAnimatedSearchMatrix(
	t *testing.T,
	database *sql.DB,
	libraryID int64,
	assetCount int,
) int {
	t.Helper()
	animatedCount := assetCount / 10
	if animatedCount < 1 {
		animatedCount = 1
	}
	result, err := database.ExecContext(context.Background(), `
        UPDATE assets SET kind = 'animated'
        WHERE id IN (
            SELECT id FROM assets
            WHERE library_id = ? AND kind = 'image'
            ORDER BY id LIMIT ?
        )`, libraryID, animatedCount)
	if err != nil {
		t.Fatalf("seed animated search matrix assets: %v", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		t.Fatalf("read animated search matrix seed count: %v", err)
	}
	if changed != int64(animatedCount) {
		t.Fatalf("animated search matrix seed = %d, want %d", changed, animatedCount)
	}
	return animatedCount
}

func verifyOrderFirstSearchMatrix(
	t *testing.T,
	store *sqlitestore.Store,
	database *sql.DB,
	libraryID int64,
) orderFirstSearchMatrixMetrics {
	t.Helper()
	ctx := context.Background()
	libraryScope, err := store.ResolveScope(ctx, libraryID, 0)
	if err != nil {
		t.Fatalf("resolve order-first library scope: %v", err)
	}
	libraryScope.CanonicalDirectoryID, err = catalog.NormalizeRootScope(
		libraryScope.RootDirectoryID,
		libraryScope.DirectoryID,
	)
	if err != nil {
		t.Fatalf("normalize order-first library scope: %v", err)
	}
	searchRevision, err := store.ResolveGlobalCatalogRevision(ctx)
	if err != nil {
		t.Fatalf("resolve order-first global revision: %v", err)
	}
	terms := func(raw string) []string {
		normalized, normalizeErr := catalog.NormalizeSearchTerms(raw)
		if normalizeErr != nil {
			t.Fatalf("normalize order-first matrix term %q: %v", raw, normalizeErr)
		}
		return normalized
	}
	baseLibrary := catalog.AssetQuery{
		Scope: libraryScope, ScopeKind: catalog.ScopeLibrary,
		SearchTerms: terms("asset"), Sort: catalog.SortName, Order: catalog.OrderAsc,
	}
	cases := make([]struct {
		name  string
		query catalog.AssetQuery
	}, 0, 24)
	for _, sortField := range []catalog.SortField{catalog.SortName, catalog.SortModifiedAt, catalog.SortSize} {
		for _, order := range []catalog.SortOrder{catalog.OrderAsc, catalog.OrderDesc} {
			query := baseLibrary
			query.Sort = sortField
			query.Order = order
			cases = append(cases, struct {
				name  string
				query catalog.AssetQuery
			}{name: "library/" + string(sortField) + "/" + string(order), query: query})
		}
	}
	for _, kindCase := range []struct {
		name  string
		kinds []catalog.AssetKind
	}{
		{name: "library/kind-image", kinds: []catalog.AssetKind{catalog.KindImage}},
		{name: "library/kind-video", kinds: []catalog.AssetKind{catalog.KindVideo}},
		{name: "library/kind-animated", kinds: []catalog.AssetKind{catalog.KindAnimated}},
		{name: "library/kind-image-video", kinds: []catalog.AssetKind{catalog.KindImage, catalog.KindVideo}},
	} {
		query := baseLibrary
		query.Kinds = kindCase.kinds
		cases = append(cases, struct {
			name  string
			query catalog.AssetQuery
		}{name: kindCase.name, query: query})
	}
	shortQuery := baseLibrary
	shortQuery.SearchTerms = terms("99")
	cases = append(cases, struct {
		name  string
		query catalog.AssetQuery
	}{name: "library/short-unanchored", query: shortQuery})
	var minimumModified, maximumModified int64
	if err := database.QueryRowContext(ctx, `
        SELECT MIN(mtime_ns), MAX(mtime_ns) FROM assets WHERE library_id = ?`,
		libraryID,
	).Scan(&minimumModified, &maximumModified); err != nil {
		t.Fatalf("read order-first modification bounds: %v", err)
	}
	modifiedBefore := maximumModified + 1
	dateQuery := baseLibrary
	dateQuery.ModifiedFromNS = &minimumModified
	dateQuery.ModifiedBeforeNS = &modifiedBefore
	cases = append(cases, struct {
		name  string
		query catalog.AssetQuery
	}{name: "library/modified-window", query: dateQuery})

	globalScope := catalog.Scope{}
	for _, sortField := range []catalog.SortField{catalog.SortName, catalog.SortModifiedAt, catalog.SortSize} {
		for _, order := range []catalog.SortOrder{catalog.OrderAsc, catalog.OrderDesc} {
			query := catalog.AssetQuery{
				Scope: globalScope, ScopeKind: catalog.ScopeGlobal,
				CatalogRevision: searchRevision, SearchTerms: terms("asset"),
				Sort: sortField, Order: order,
			}
			cases = append(cases, struct {
				name  string
				query catalog.AssetQuery
			}{name: "global/" + string(sortField) + "/" + string(order), query: query})
		}
	}

	rootRecursive := baseLibrary
	rootRecursive.ScopeKind = catalog.ScopeDirectory
	rootRecursive.Recursive = true
	cases = append(cases, struct {
		name  string
		query catalog.AssetQuery
	}{name: "directory-root/recursive/name/asc", query: rootRecursive})

	var directDirectoryID int64
	if err := database.QueryRowContext(ctx, `
        SELECT id FROM directories
        WHERE library_id = ? AND direct_asset_count > 0
        ORDER BY direct_asset_count DESC, id LIMIT 1`, libraryID,
	).Scan(&directDirectoryID); err != nil {
		t.Fatalf("resolve order-first direct directory: %v", err)
	}
	directScope := resolveMatrixDirectoryScope(t, store, libraryID, directDirectoryID)
	for _, matrixCase := range []struct {
		sort  catalog.SortField
		order catalog.SortOrder
	}{
		{sort: catalog.SortName, order: catalog.OrderAsc},
		{sort: catalog.SortName, order: catalog.OrderDesc},
		{sort: catalog.SortModifiedAt, order: catalog.OrderDesc},
		{sort: catalog.SortSize, order: catalog.OrderAsc},
	} {
		query := baseLibrary
		query.Scope = directScope
		query.ScopeKind = catalog.ScopeDirectory
		query.Sort = matrixCase.sort
		query.Order = matrixCase.order
		cases = append(cases, struct {
			name  string
			query catalog.AssetQuery
		}{name: "directory-direct/" + string(matrixCase.sort) + "/" + string(matrixCase.order), query: query})
	}

	var recursiveDirectoryID int64
	if err := database.QueryRowContext(ctx, `
        SELECT id FROM directories
        WHERE library_id = ? AND relative_path = 'group-000'`, libraryID,
	).Scan(&recursiveDirectoryID); err != nil {
		t.Fatalf("resolve order-first recursive directory: %v", err)
	}
	recursiveScope := resolveMatrixDirectoryScope(t, store, libraryID, recursiveDirectoryID)
	for _, matrixCase := range []struct {
		sort  catalog.SortField
		order catalog.SortOrder
	}{
		{sort: catalog.SortName, order: catalog.OrderAsc},
		{sort: catalog.SortModifiedAt, order: catalog.OrderDesc},
		{sort: catalog.SortSize, order: catalog.OrderAsc},
	} {
		query := baseLibrary
		query.Scope = recursiveScope
		query.ScopeKind = catalog.ScopeDirectory
		query.Recursive = true
		query.Sort = matrixCase.sort
		query.Order = matrixCase.order
		cases = append(cases, struct {
			name  string
			query catalog.AssetQuery
		}{name: "directory-recursive/" + string(matrixCase.sort) + "/" + string(matrixCase.order), query: query})
	}

	metrics := orderFirstSearchMatrixMetrics{
		firstPageMicroseconds:            make(map[string]int64, len(cases)),
		secondPageMicroseconds:           make(map[string]int64, len(cases)),
		firstPageP95Microseconds:         make(map[string]int64, 3),
		secondPageP95Microseconds:        make(map[string]int64, 3),
		repositoryFirstPageMicroseconds:  make(map[string]int64, len(cases)),
		repositorySecondPageMicroseconds: make(map[string]int64, len(cases)),
	}
	for _, matrixCase := range cases {
		t.Run("order-first-matrix/"+matrixCase.name, func(t *testing.T) {
			repositoryStarted := time.Now()
			first, listErr := store.ListAssetPage(ctx, catalog.AssetListParams{
				Query: matrixCase.query, Limit: 101,
			})
			metrics.repositoryFirstPageMicroseconds[matrixCase.name] = time.Since(repositoryStarted).Microseconds()
			if listErr != nil {
				t.Fatalf("list production first page: %v", listErr)
			}
			candidateQuery, candidateArgs := orderFirstMatrixQuery(matrixCase.query, nil, 101)
			if matrixCase.name == "library/modified-window" {
				repositoryQuery, repositoryArgs := modifiedWindowRepositoryQuery(matrixCase.query, nil, 101)
				metrics.modifiedWindowFirstPlan = explainQueryPlan(t, database, repositoryQuery, repositoryArgs...)
				metrics.modifiedWindowCandidatePlan = explainQueryPlan(t, database, candidateQuery, candidateArgs...)
			}
			candidateStarted := time.Now()
			candidateIDs := queryAssetIDs(t, database, candidateQuery, candidateArgs...)
			metrics.firstPageMicroseconds[matrixCase.name] = time.Since(candidateStarted).Microseconds()
			if !equalInt64s(candidateIDs, assetIDs(first)) {
				t.Fatalf("candidate first page differs: got %d IDs, want %d", len(candidateIDs), len(first))
			}
			if repeatOrderFirstMatrixCase(matrixCase.name) {
				latencies := measureQuery(t, 20, func(queryContext context.Context) error {
					return queryAssetIDsContext(queryContext, database, candidateQuery, candidateArgs...)
				})
				metrics.firstPageP95Microseconds[matrixCase.name] = percentile(latencies, 95).Microseconds()
			}
			metrics.cases++
			if len(first) != 101 {
				return
			}
			after := assetPosition(first[99])
			repositoryStarted = time.Now()
			second, secondErr := store.ListAssetPage(ctx, catalog.AssetListParams{
				Query: matrixCase.query, After: &after, Limit: 101,
			})
			metrics.repositorySecondPageMicroseconds[matrixCase.name] = time.Since(repositoryStarted).Microseconds()
			if secondErr != nil {
				t.Fatalf("list production second page: %v", secondErr)
			}
			candidateQuery, candidateArgs = orderFirstMatrixQuery(matrixCase.query, &after, 101)
			if matrixCase.name == "library/modified-window" {
				repositoryQuery, repositoryArgs := modifiedWindowRepositoryQuery(matrixCase.query, &after, 101)
				metrics.modifiedWindowSecondPlan = explainQueryPlan(t, database, repositoryQuery, repositoryArgs...)
			}
			candidateStarted = time.Now()
			candidateIDs = queryAssetIDs(t, database, candidateQuery, candidateArgs...)
			metrics.secondPageMicroseconds[matrixCase.name] = time.Since(candidateStarted).Microseconds()
			if !equalInt64s(candidateIDs, assetIDs(second)) {
				t.Fatalf("candidate second page differs: got %d IDs, want %d", len(candidateIDs), len(second))
			}
			if repeatOrderFirstMatrixCase(matrixCase.name) {
				latencies := measureQuery(t, 20, func(queryContext context.Context) error {
					return queryAssetIDsContext(queryContext, database, candidateQuery, candidateArgs...)
				})
				metrics.secondPageP95Microseconds[matrixCase.name] = percentile(latencies, 95).Microseconds()
			}
			metrics.cursorCases++
		})
	}
	return metrics
}

func repeatOrderFirstMatrixCase(name string) bool {
	switch name {
	case "library/kind-video", "library/modified-window", "directory-recursive/name/asc":
		return true
	default:
		return false
	}
}

func resolveMatrixDirectoryScope(
	t *testing.T,
	store *sqlitestore.Store,
	libraryID int64,
	directoryID int64,
) catalog.Scope {
	t.Helper()
	scope, err := store.ResolveScope(context.Background(), libraryID, directoryID)
	if err != nil {
		t.Fatalf("resolve order-first directory scope: %v", err)
	}
	scope.CanonicalDirectoryID, err = catalog.NormalizeRootScope(scope.RootDirectoryID, scope.DirectoryID)
	if err != nil {
		t.Fatalf("normalize order-first directory scope: %v", err)
	}
	return scope
}

func assetPosition(item catalog.Asset) catalog.AssetPosition {
	directoryPath := path.Dir(item.RelativePath)
	if directoryPath == "." {
		directoryPath = ""
	}
	return catalog.AssetPosition{
		DirectoryPath:  directoryPath,
		NaturalNameKey: item.NaturalNameKey,
		Name:           item.Name, LibraryID: item.LibraryID,
		RelativePath: item.RelativePath, ModifiedAtNS: item.ModifiedAtNS,
		SizeBytes: item.SizeBytes, ID: item.ID,
	}
}

func orderFirstMatrixQuery(
	query catalog.AssetQuery,
	after *catalog.AssetPosition,
	limit int,
) (string, []any) {
	return orderFirstMatrixQueryWithHydration(query, after, limit, false)
}

func orderFirstHydratedMatrixQuery(
	query catalog.AssetQuery,
	after *catalog.AssetPosition,
	limit int,
) (string, []any) {
	return orderFirstMatrixQueryWithHydration(query, after, limit, true)
}

func orderFirstMatrixQueryWithHydration(
	query catalog.AssetQuery,
	after *catalog.AssetPosition,
	limit int,
	hydrate bool,
) (string, []any) {
	var builder strings.Builder
	args := make([]any, 0, 48)
	if query.ScopeKind == catalog.ScopeDirectory && query.Recursive && query.Scope.CanonicalDirectoryID != 0 {
		builder.WriteString(`
        WITH RECURSIVE subtree(id) AS (
            SELECT ?
            UNION
            SELECT d.id
            FROM directories d
            JOIN subtree parent ON d.parent_id = parent.id
            WHERE d.library_id = ?
        )`)
		args = append(args, query.Scope.DirectoryID, query.Scope.LibraryID)
	}
	if hydrate {
		builder.WriteString(`
        SELECT a.id, a.library_id, l.name, a.directory_id, a.relative_path, a.name,
               a.natural_name_key, a.kind, a.media_format, a.mime_type,
               a.size_bytes, a.mtime_ns, a.source_fingerprint,
               a.width, a.height, a.duration_ms, a.probe_status,
               a.probe_error_code, a.playback_status,
               t.status, t.error_code,
               storyboard.status, storyboard.error_code,
               storyboard.frame_count, storyboard.sprite_columns,
               storyboard.sprite_rows, storyboard.cell_width,
               storyboard.cell_height, l.status,
               EXISTS(SELECT 1 FROM asset_favorites favorite WHERE favorite.asset_id = a.id)
        FROM assets a INDEXED BY ` + orderFirstMatrixIndex(query) + `
        JOIN libraries l ON l.id = a.library_id
        LEFT JOIN thumbnails t ON t.asset_id = a.id AND t.variant = 'grid'
        LEFT JOIN thumbnails storyboard
          ON storyboard.asset_id = a.id AND storyboard.variant = 'storyboard'
        WHERE 1 = 1`)
	} else {
		builder.WriteString(`
        SELECT a.id
        FROM assets a INDEXED BY ` + orderFirstMatrixIndex(query) + `
        WHERE 1 = 1`)
	}
	if anchor := matrixFTSAnchor(query.SearchTerms); anchor != "" {
		builder.WriteString(`
          AND EXISTS (
            SELECT 1 FROM asset_search
            WHERE asset_search.rowid = a.id AND asset_search MATCH ?
          )`)
		args = append(args, anchor)
	}
	switch query.ScopeKind {
	case catalog.ScopeLibrary:
		builder.WriteString(` AND a.library_id = ?`)
		args = append(args, query.Scope.LibraryID)
	case catalog.ScopeDirectory:
		builder.WriteString(` AND a.library_id = ?`)
		args = append(args, query.Scope.LibraryID)
		if !query.Recursive {
			builder.WriteString(` AND a.directory_id = ?`)
			args = append(args, query.Scope.DirectoryID)
		} else if query.Scope.CanonicalDirectoryID != 0 {
			builder.WriteString(` AND a.directory_id IN (SELECT id FROM subtree)`)
		}
	}
	for _, term := range query.SearchTerms {
		builder.WriteString(`
          AND (instr(a.search_name_key, ?) > 0 OR instr(a.search_path_key, ?) > 0)`)
		args = append(args, term, term)
	}
	if len(query.Kinds) > 0 {
		builder.WriteString(` AND a.kind IN (`)
		for index, kind := range query.Kinds {
			if index > 0 {
				builder.WriteByte(',')
			}
			builder.WriteByte('?')
			args = append(args, string(kind))
		}
		builder.WriteByte(')')
	}
	if query.ModifiedFromNS != nil {
		builder.WriteString(` AND a.mtime_ns >= ?`)
		args = append(args, *query.ModifiedFromNS)
	}
	if query.ModifiedBeforeNS != nil {
		builder.WriteString(` AND a.mtime_ns < ?`)
		args = append(args, *query.ModifiedBeforeNS)
	}
	if after != nil {
		appendOrderFirstMatrixKeyset(&builder, &args, query, *after)
	}
	appendOrderFirstMatrixOrder(&builder, query)
	builder.WriteString(` LIMIT ?`)
	args = append(args, limit)
	return builder.String(), args
}

func modifiedWindowRepositoryQuery(
	query catalog.AssetQuery,
	after *catalog.AssetPosition,
	limit int,
) (string, []any) {
	var builder strings.Builder
	builder.WriteString(`
        SELECT a.id, a.library_id, l.name, a.directory_id, a.relative_path, a.name,
               a.natural_name_key, a.kind, a.media_format, a.mime_type,
               a.size_bytes, a.mtime_ns, a.source_fingerprint,
               a.width, a.height, a.duration_ms, a.probe_status,
               a.probe_error_code, a.playback_status,
               t.status, t.error_code,
               storyboard.status, storyboard.error_code,
               storyboard.frame_count, storyboard.sprite_columns,
               storyboard.sprite_rows, storyboard.cell_width,
               storyboard.cell_height, l.status,
               EXISTS(SELECT 1 FROM asset_favorites favorite WHERE favorite.asset_id = a.id)
        FROM assets a
        JOIN libraries l ON l.id = a.library_id
        LEFT JOIN thumbnails t ON t.asset_id = a.id AND t.variant = 'grid'
        LEFT JOIN thumbnails storyboard
          ON storyboard.asset_id = a.id AND storyboard.variant = 'storyboard'
        JOIN asset_search ON asset_search.rowid = a.id
        WHERE asset_search MATCH ?
          AND a.library_id = ?`)
	args := []any{matrixFTSAnchor(query.SearchTerms), query.Scope.LibraryID}
	for _, term := range query.SearchTerms {
		builder.WriteString(`
          AND (instr(a.search_name_key, ?) > 0 OR instr(a.search_path_key, ?) > 0)`)
		args = append(args, term, term)
	}
	if query.ModifiedFromNS != nil {
		builder.WriteString(` AND a.mtime_ns >= ?`)
		args = append(args, *query.ModifiedFromNS)
	}
	if query.ModifiedBeforeNS != nil {
		builder.WriteString(` AND a.mtime_ns < ?`)
		args = append(args, *query.ModifiedBeforeNS)
	}
	if after != nil {
		appendOrderFirstMatrixKeyset(&builder, &args, query, *after)
	}
	appendOrderFirstMatrixOrder(&builder, query)
	builder.WriteString(` LIMIT ?`)
	args = append(args, limit)
	return builder.String(), args
}

func orderFirstMatrixIndex(query catalog.AssetQuery) string {
	if query.Sort == catalog.SortModifiedAt {
		if query.ScopeKind == catalog.ScopeGlobal {
			return "assets_search_global_modified"
		}
		if query.ScopeKind == catalog.ScopeDirectory && !query.Recursive {
			return "assets_browse_directory_modified"
		}
		return "assets_modified"
	}
	if query.Sort == catalog.SortSize {
		if query.ScopeKind == catalog.ScopeGlobal {
			return "assets_search_global_size"
		}
		if query.ScopeKind == catalog.ScopeDirectory && !query.Recursive {
			return "assets_browse_directory_size"
		}
		return "assets_browse_library_size"
	}
	return "assets_browse_folder_name_v2"
}

func matrixFTSAnchor(terms []string) string {
	longest := ""
	for _, term := range terms {
		if strings.ContainsRune(term, '"') || utf8.RuneCountInString(term) < 3 {
			continue
		}
		if utf8.RuneCountInString(term) > utf8.RuneCountInString(longest) {
			longest = term
		}
	}
	if longest == "" {
		return ""
	}
	return `"` + longest + `"`
}

func appendOrderFirstMatrixKeyset(
	builder *strings.Builder,
	args *[]any,
	query catalog.AssetQuery,
	after catalog.AssetPosition,
) {
	operator := ">"
	if query.Order == catalog.OrderDesc {
		operator = "<"
	}
	if query.Sort == catalog.SortModifiedAt {
		builder.WriteString(` AND (a.mtime_ns ` + operator + ` ? OR (a.mtime_ns = ? AND a.id ` + operator + ` ?))`)
		*args = append(*args, after.ModifiedAtNS, after.ModifiedAtNS, after.ID)
		return
	}
	if query.Sort == catalog.SortSize {
		builder.WriteString(` AND (a.size_bytes ` + operator + ` ? OR (a.size_bytes = ? AND a.id ` + operator + ` ?))`)
		*args = append(*args, after.SizeBytes, after.SizeBytes, after.ID)
		return
	}
	if query.ScopeKind == catalog.ScopeGlobal {
		builder.WriteString(` AND (
            a.library_id ` + operator + ` ?
            OR (a.library_id = ? AND ` + matrixDirectoryPathSQL + ` ` + operator + ` ?)
            OR (a.library_id = ? AND ` + matrixDirectoryPathSQL + ` = ? AND a.natural_name_key ` + operator + ` ?)
            OR (a.library_id = ? AND ` + matrixDirectoryPathSQL + ` = ? AND a.natural_name_key = ? AND a.name ` + operator + ` ?)
            OR (a.library_id = ? AND ` + matrixDirectoryPathSQL + ` = ? AND a.natural_name_key = ? AND a.name = ? AND a.relative_path ` + operator + ` ?)
            OR (a.library_id = ? AND ` + matrixDirectoryPathSQL + ` = ? AND a.natural_name_key = ? AND a.name = ? AND a.relative_path = ? AND a.id ` + operator + ` ?)
          )`)
		*args = append(*args,
			after.LibraryID,
			after.LibraryID, after.DirectoryPath,
			after.LibraryID, after.DirectoryPath, after.NaturalNameKey,
			after.LibraryID, after.DirectoryPath, after.NaturalNameKey, after.Name,
			after.LibraryID, after.DirectoryPath, after.NaturalNameKey, after.Name, after.RelativePath,
			after.LibraryID, after.DirectoryPath, after.NaturalNameKey, after.Name, after.RelativePath, after.ID,
		)
		return
	}
	builder.WriteString(` AND (
          ` + matrixDirectoryPathSQL + ` ` + operator + ` ?
          OR (` + matrixDirectoryPathSQL + ` = ? AND a.natural_name_key ` + operator + ` ?)
          OR (` + matrixDirectoryPathSQL + ` = ? AND a.natural_name_key = ? AND a.name ` + operator + ` ?)
          OR (` + matrixDirectoryPathSQL + ` = ? AND a.natural_name_key = ? AND a.name = ? AND a.relative_path ` + operator + ` ?)
          OR (` + matrixDirectoryPathSQL + ` = ? AND a.natural_name_key = ? AND a.name = ? AND a.relative_path = ? AND a.id ` + operator + ` ?)
        )`)
	*args = append(*args,
		after.DirectoryPath,
		after.DirectoryPath, after.NaturalNameKey,
		after.DirectoryPath, after.NaturalNameKey, after.Name,
		after.DirectoryPath, after.NaturalNameKey, after.Name, after.RelativePath,
		after.DirectoryPath, after.NaturalNameKey, after.Name, after.RelativePath, after.ID,
	)
}

const matrixDirectoryPathSQL = `CASE
    WHEN length(a.relative_path) = length(a.name) THEN ''
    ELSE substr(a.relative_path, 1, length(a.relative_path) - length(a.name) - 1)
END`

func appendOrderFirstMatrixOrder(builder *strings.Builder, query catalog.AssetQuery) {
	direction := " ASC"
	if query.Order == catalog.OrderDesc {
		direction = " DESC"
	}
	if query.Sort == catalog.SortModifiedAt {
		builder.WriteString(` ORDER BY a.mtime_ns` + direction + `, a.id` + direction)
		return
	}
	if query.Sort == catalog.SortSize {
		builder.WriteString(` ORDER BY a.size_bytes` + direction + `, a.id` + direction)
		return
	}
	if query.ScopeKind == catalog.ScopeGlobal {
		builder.WriteString(` ORDER BY a.library_id` + direction + `, ` + matrixDirectoryPathSQL + direction +
			`, a.natural_name_key` + direction + `, a.name` + direction +
			`, a.relative_path` + direction + `, a.id` + direction)
		return
	}
	builder.WriteString(` ORDER BY ` + matrixDirectoryPathSQL + direction +
		`, a.natural_name_key` + direction + `, a.name` + direction +
		`, a.relative_path` + direction + `, a.id` + direction)
}

func measureOrderFirstCancellation(
	t *testing.T,
	database *sql.DB,
	libraryID int64,
) time.Duration {
	t.Helper()
	query, args := orderFirstSearchQuery(
		libraryID, nil, "no-synthetic-asset-matches-this-term", false,
	)
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	started := time.Now()
	go func() {
		result <- queryAssetIDsContext(ctx, database, query, args...)
	}()
	time.Sleep(time.Millisecond)
	cancel()
	err := <-result
	elapsed := time.Since(started)
	if err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("order-first active cancellation: %v", err)
	}
	return elapsed
}

type searchRepositoryPlans struct {
	count  []string
	first  []string
	second []string
}

func explainSearchRepositoryPlans(
	t *testing.T,
	database *sql.DB,
	libraryID int64,
	after catalog.AssetPosition,
) searchRepositoryPlans {
	t.Helper()
	const countQuery = `
        SELECT COUNT(*),
               COALESCE(SUM(CASE WHEN a.kind IN ('image', 'animated') THEN 1 ELSE 0 END), 0),
               COALESCE(SUM(CASE WHEN a.kind = 'video' THEN 1 ELSE 0 END), 0)
        FROM assets a
        JOIN asset_search ON asset_search.rowid = a.id
        WHERE asset_search MATCH ?
          AND a.library_id = ?
          AND (instr(a.search_name_key, ?) > 0 OR instr(a.search_path_key, ?) > 0)`
	const firstQuery = `
        SELECT a.id, t.status, storyboard.status,
               EXISTS(SELECT 1 FROM asset_favorites favorite WHERE favorite.asset_id = a.id)
        FROM assets a
        JOIN libraries l ON l.id = a.library_id
        LEFT JOIN thumbnails t ON t.asset_id = a.id AND t.variant = 'grid'
        LEFT JOIN thumbnails storyboard
          ON storyboard.asset_id = a.id AND storyboard.variant = 'storyboard'
        JOIN asset_search ON asset_search.rowid = a.id
        WHERE asset_search MATCH ?
          AND a.library_id = ?
          AND (instr(a.search_name_key, ?) > 0 OR instr(a.search_path_key, ?) > 0)
        ORDER BY
          CASE
            WHEN length(a.relative_path) = length(a.name) THEN ''
            ELSE substr(a.relative_path, 1, length(a.relative_path) - length(a.name) - 1)
          END ASC,
          a.natural_name_key ASC, a.name ASC, a.relative_path ASC, a.id ASC
        LIMIT ?`
	const secondQuery = `
        SELECT a.id, t.status, storyboard.status,
               EXISTS(SELECT 1 FROM asset_favorites favorite WHERE favorite.asset_id = a.id)
        FROM assets a
        JOIN libraries l ON l.id = a.library_id
        LEFT JOIN thumbnails t ON t.asset_id = a.id AND t.variant = 'grid'
        LEFT JOIN thumbnails storyboard
          ON storyboard.asset_id = a.id AND storyboard.variant = 'storyboard'
        JOIN asset_search ON asset_search.rowid = a.id
        WHERE asset_search MATCH ?
          AND a.library_id = ?
          AND (instr(a.search_name_key, ?) > 0 OR instr(a.search_path_key, ?) > 0)
          AND (
            CASE
              WHEN length(a.relative_path) = length(a.name) THEN ''
              ELSE substr(a.relative_path, 1, length(a.relative_path) - length(a.name) - 1)
            END > ?
            OR (
              CASE
                WHEN length(a.relative_path) = length(a.name) THEN ''
                ELSE substr(a.relative_path, 1, length(a.relative_path) - length(a.name) - 1)
              END = ? AND a.natural_name_key > ?
            )
            OR (
              CASE
                WHEN length(a.relative_path) = length(a.name) THEN ''
                ELSE substr(a.relative_path, 1, length(a.relative_path) - length(a.name) - 1)
              END = ? AND a.natural_name_key = ? AND a.name > ?
            )
            OR (
              CASE
                WHEN length(a.relative_path) = length(a.name) THEN ''
                ELSE substr(a.relative_path, 1, length(a.relative_path) - length(a.name) - 1)
              END = ? AND a.natural_name_key = ? AND a.name = ? AND a.relative_path > ?
            )
            OR (
              CASE
                WHEN length(a.relative_path) = length(a.name) THEN ''
                ELSE substr(a.relative_path, 1, length(a.relative_path) - length(a.name) - 1)
              END = ? AND a.natural_name_key = ? AND a.name = ?
              AND a.relative_path = ? AND a.id > ?
            )
          )
        ORDER BY
          CASE
            WHEN length(a.relative_path) = length(a.name) THEN ''
            ELSE substr(a.relative_path, 1, length(a.relative_path) - length(a.name) - 1)
          END ASC,
          a.natural_name_key ASC, a.name ASC, a.relative_path ASC, a.id ASC
        LIMIT ?`
	const anchor = `"asset"`
	baseArgs := []any{anchor, libraryID, "asset", "asset"}
	secondArgs := append(append([]any{}, baseArgs...),
		after.DirectoryPath,
		after.DirectoryPath, after.NaturalNameKey,
		after.DirectoryPath, after.NaturalNameKey, after.Name,
		after.DirectoryPath, after.NaturalNameKey, after.Name, after.RelativePath,
		after.DirectoryPath, after.NaturalNameKey, after.Name, after.RelativePath, after.ID,
		101,
	)
	return searchRepositoryPlans{
		count: explainQueryPlan(t, database, countQuery, baseArgs...),
		first: explainQueryPlan(
			t, database, firstQuery, append(append([]any{}, baseArgs...), 101)...,
		),
		second: explainQueryPlan(t, database, secondQuery, secondArgs...),
	}
}

func explainQueryPlan(t *testing.T, database *sql.DB, query string, args ...any) []string {
	t.Helper()
	rows, err := database.QueryContext(context.Background(), "EXPLAIN QUERY PLAN "+query, args...)
	if err != nil {
		t.Fatalf("explain capacity query: %v", err)
	}
	defer rows.Close()
	details := make([]string, 0, 16)
	for rows.Next() {
		var id, parent, unused int
		var detail string
		if err := rows.Scan(&id, &parent, &unused, &detail); err != nil {
			t.Fatalf("scan capacity query plan: %v", err)
		}
		details = append(details, detail)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate capacity query plan: %v", err)
	}
	return details
}

func measureActiveSearchCancellation(
	t *testing.T,
	service *catalog.Service,
	libraryID int64,
) time.Duration {
	t.Helper()
	query := "a"
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	started := time.Now()
	go func() {
		_, err := service.ListAssets(ctx, catalog.AssetRequest{
			LibraryID: libraryID, SearchQuery: &query,
			Sort: catalog.SortName, Limit: 100,
		})
		result <- err
	}()
	time.Sleep(time.Millisecond)
	cancel()
	err := <-result
	elapsed := time.Since(started)
	if err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("active search cancellation: %v", err)
	}
	// A query that wins the race and completes before cancellation has no
	// outstanding work to stop; its observed completion latency is therefore
	// the stronger bound.
	return elapsed
}

func readDirectoryPage(ctx context.Context, database *sql.DB, libraryID int64) error {
	var rootID int64
	if err := database.QueryRowContext(ctx, `
		SELECT id FROM directories
		WHERE library_id = ? AND relative_path = ''`, libraryID).Scan(&rootID); err != nil {
		return err
	}
	rows, err := database.QueryContext(ctx, `
		SELECT id, name
		FROM directories
		WHERE library_id = ? AND parent_id = ?
		ORDER BY name, id
		LIMIT 100`, libraryID, rootID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return err
		}
	}
	return rows.Err()
}

func assertCatalogCounts(t *testing.T, database *sql.DB, libraryID int64, directories, assets int) {
	t.Helper()
	var actualDirectories int
	if err := database.QueryRow(`
		SELECT COUNT(*) FROM directories WHERE library_id = ?`, libraryID).Scan(&actualDirectories); err != nil {
		t.Fatalf("count indexed directories: %v", err)
	}
	if actualDirectories != directories {
		t.Fatalf("indexed directories = %d, want %d", actualDirectories, directories)
	}
	var actualAssets int
	if err := database.QueryRow(`
		SELECT COUNT(*) FROM assets WHERE library_id = ?`, libraryID).Scan(&actualAssets); err != nil {
		t.Fatalf("count indexed assets: %v", err)
	}
	if actualAssets != assets {
		t.Fatalf("indexed assets = %d, want %d", actualAssets, assets)
	}
}

func assertDirectoryRollup(
	t *testing.T,
	database *sql.DB,
	libraryID int64,
	assetCount int,
	shape capacityFixtureShape,
) {
	t.Helper()
	var rootDirect, rootRecursive int
	if err := database.QueryRow(`
		SELECT direct_asset_count, recursive_asset_count
		FROM directories
		WHERE library_id = ? AND relative_path = ''`, libraryID).
		Scan(&rootDirect, &rootRecursive); err != nil {
		t.Fatalf("read capacity root counts: %v", err)
	}
	if rootDirect != 0 || rootRecursive != assetCount {
		t.Fatalf("capacity root counts = (%d, %d), want (0, %d)",
			rootDirect, rootRecursive, assetCount)
	}

	var directTotal, invalidDirectRows, invalidRecursiveRows int
	if err := database.QueryRow(`
		SELECT COALESCE(sum(direct_asset_count), 0)
		FROM directories
		WHERE library_id = ?`, libraryID).Scan(&directTotal); err != nil {
		t.Fatalf("sum capacity direct counts: %v", err)
	}
	if err := database.QueryRow(`
		SELECT count(*)
		FROM (
			SELECT directory.id
			FROM directories AS directory
			LEFT JOIN assets AS asset
			  ON asset.library_id = directory.library_id
			 AND asset.directory_id = directory.id
			WHERE directory.library_id = ?
			GROUP BY directory.id, directory.direct_asset_count
			HAVING directory.direct_asset_count <> count(asset.id)
		)`, libraryID).Scan(&invalidDirectRows); err != nil {
		t.Fatalf("verify capacity direct counts: %v", err)
	}
	if err := database.QueryRow(`
		SELECT count(*)
		FROM directories
		WHERE library_id = ? AND recursive_asset_count < direct_asset_count`,
		libraryID).Scan(&invalidRecursiveRows); err != nil {
		t.Fatalf("verify capacity recursive count lower bounds: %v", err)
	}
	if directTotal != assetCount || invalidDirectRows != 0 || invalidRecursiveRows != 0 {
		t.Fatalf(
			"capacity rollup totals direct=%d invalidDirect=%d invalidRecursive=%d, want %d, 0, 0",
			directTotal, invalidDirectRows, invalidRecursiveRows, assetCount,
		)
	}

	for _, relativePath := range shape.firstDeepBranchPaths {
		var recursive int
		if err := database.QueryRow(`
			SELECT recursive_asset_count
			FROM directories
			WHERE library_id = ? AND relative_path = ?`,
			libraryID, relativePath).Scan(&recursive); err != nil {
			t.Fatalf("read deep fixture count %q: %v", relativePath, err)
		}
		if recursive != shape.firstDeepTargetAssetCnt {
			t.Fatalf("deep fixture count %q = %d, want %d",
				relativePath, recursive, shape.firstDeepTargetAssetCnt)
		}
	}
}

func capacityAssetEntry(relativePath, parentPath, name string) scanner.CatalogEntry {
	return scanner.CatalogEntry{
		Kind:         scanner.CatalogEntryAsset,
		RelativePath: relativePath,
		ParentPath:   parentPath,
		Name:         name,
		AssetKind:    scanner.AssetKindImage,
		MediaFormat:  scanner.MediaFormatJPEG,
		MIMEType:     "image/jpeg",
		SizeBytes:    4,
	}
}

func mustJSON(t *testing.T, value any) string {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("encode metrics: %v", err)
	}
	return string(encoded)
}

func measureQuery(t *testing.T, count int, query func(context.Context) error) []time.Duration {
	t.Helper()
	latencies := make([]time.Duration, 0, count)
	for index := 0; index < count; index++ {
		started := time.Now()
		if err := query(context.Background()); err != nil {
			t.Fatalf("measure query: %v", err)
		}
		latencies = append(latencies, time.Since(started))
	}
	return latencies
}

func percentile(values []time.Duration, requested int) time.Duration {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]time.Duration(nil), values...)
	sort.Slice(sorted, func(left, right int) bool { return sorted[left] < sorted[right] })
	index := (requested*len(sorted) + 99) / 100
	if index < 1 {
		index = 1
	}
	if index > len(sorted) {
		index = len(sorted)
	}
	return sorted[index-1]
}

func databaseFamilySize(t *testing.T, databasePath string) int64 {
	t.Helper()
	var total int64
	for _, suffix := range []string{"", "-wal", "-shm"} {
		info, err := os.Stat(databasePath + suffix)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			t.Fatalf("stat SQLite file %q: %v", suffix, err)
		}
		total += info.Size()
	}
	return total
}

func capacityBudgetViolations(metrics capacityMetrics) []string {
	violations := make([]string, 0)
	if metrics.ScanDurationMS > 120_000 {
		violations = append(violations, "scanDurationMs > 120000")
	}
	if metrics.ConcurrentReadP95US > 250_000 {
		violations = append(violations, "concurrentReadP95Us > 250000")
	}
	if metrics.ConcurrentSearchP95US > 500_000 {
		violations = append(violations, "concurrentSearchP95Us > 500000")
	}
	if metrics.DirectoryListP95US > 100_000 {
		violations = append(violations, "directoryListP95Us > 100000")
	}
	if metrics.AssetListP95US > 100_000 {
		violations = append(violations, "assetListP95Us > 100000")
	}
	if metrics.RecursiveBrowseP95US > 250_000 {
		violations = append(violations, "recursiveBrowseP95Us > 250000")
	}
	if metrics.FTSSearchP95US > 250_000 {
		violations = append(violations, "ftsSearchP95Us > 250000")
	}
	if metrics.ShortSearchP95US > 250_000 {
		violations = append(violations, "shortSearchP95Us > 250000")
	}
	if metrics.GlobalSearchP95US > 250_000 {
		violations = append(violations, "globalSearchP95Us > 250000")
	}
	if metrics.SearchKeysetP95US > 250_000 {
		violations = append(violations, "searchKeysetP95Us > 250000")
	}
	if metrics.SearchCancelLatencyUS > 250_000 {
		violations = append(violations, "searchCancelLatencyUs > 250000")
	}
	if metrics.SearchRebuildDurationMS > 120_000 {
		violations = append(violations, "searchRebuildDurationMs > 120000")
	}
	if metrics.StoryboardAdmissionMax > sqlitestore.MaxStoryboardAdmissionBatch {
		violations = append(violations, "storyboardAdmissionMax > 128")
	}
	if metrics.StoryboardAdmissionMS > 60_000 {
		violations = append(violations, "storyboardAdmissionMs > 60000")
	}
	if metrics.StoryboardBrowseP95US > 250_000 {
		violations = append(violations, "storyboardBrowseP95Us > 250000")
	}
	if metrics.PeakRSSBytes > 0 && metrics.PeakRSSBytes > 1<<30 {
		violations = append(violations, "peakRssBytes > 1073741824")
	}
	if metrics.DatabaseAndWALSizeBytes > 1<<30 {
		violations = append(violations, "databaseAndWalSizeBytes > 1073741824")
	}
	return violations
}
