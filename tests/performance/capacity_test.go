package performance_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

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
	GOOS                    string   `json:"goos"`
	GOARCH                  string   `json:"goarch"`
	GOMAXPROCS              int      `json:"gomaxprocs"`
	DirectoryCount          int      `json:"directoryCount"`
	AssetCount              int      `json:"assetCount"`
	FixtureMaxDepth         int      `json:"fixtureMaxDepth"`
	FixtureDeepBranches     int      `json:"fixtureDeepBranches"`
	FixtureAssetTargets     int      `json:"fixtureAssetTargets"`
	FixtureDurationMS       int64    `json:"fixtureDurationMs"`
	ScanDurationMS          int64    `json:"scanDurationMs"`
	ConcurrentReads         int      `json:"concurrentReads"`
	ConcurrentReadP95US     int64    `json:"concurrentReadP95Us"`
	ConcurrentReadMaxUS     int64    `json:"concurrentReadMaxUs"`
	ConcurrentSearches      int      `json:"concurrentSearches"`
	ConcurrentSearchP95US   int64    `json:"concurrentSearchP95Us"`
	ConcurrentSearchMaxUS   int64    `json:"concurrentSearchMaxUs"`
	DirectoryListP50US      int64    `json:"directoryListP50Us"`
	DirectoryListP95US      int64    `json:"directoryListP95Us"`
	AssetListP50US          int64    `json:"assetListP50Us"`
	AssetListP95US          int64    `json:"assetListP95Us"`
	RecursiveBrowseP50US    int64    `json:"recursiveBrowseP50Us"`
	RecursiveBrowseP95US    int64    `json:"recursiveBrowseP95Us"`
	FTSSearchP50US          int64    `json:"ftsSearchP50Us"`
	FTSSearchP95US          int64    `json:"ftsSearchP95Us"`
	ShortSearchP50US        int64    `json:"shortSearchP50Us"`
	ShortSearchP95US        int64    `json:"shortSearchP95Us"`
	GlobalSearchP95US       int64    `json:"globalSearchP95Us"`
	SearchKeysetP95US       int64    `json:"searchKeysetP95Us"`
	SearchCancelLatencyUS   int64    `json:"searchCancelLatencyUs"`
	SearchRebuildDurationMS int64    `json:"searchRebuildDurationMs"`
	StoryboardVideoCount    int      `json:"storyboardVideoCount"`
	StoryboardAdmissionRuns int      `json:"storyboardAdmissionRuns"`
	StoryboardAdmissionMax  int64    `json:"storyboardAdmissionMax"`
	StoryboardAdmissionMS   int64    `json:"storyboardAdmissionMs"`
	StoryboardBrowseP95US   int64    `json:"storyboardBrowseP95Us"`
	PeakGoHeapAllocBytes    uint64   `json:"peakGoHeapAllocBytes"`
	PeakRSSBytes            uint64   `json:"peakRssBytes,omitempty"`
	PeakRSSSource           string   `json:"peakRssSource,omitempty"`
	DatabaseAndWALSizeBytes int64    `json:"databaseAndWalSizeBytes"`
	BudgetProfile           string   `json:"budgetProfile"`
	SearchBudgetProfile     string   `json:"searchBudgetProfile"`
	BudgetViolations        []string `json:"budgetViolations"`
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
	cancelLatency := measureActiveSearchCancellation(t, catalogService, created.ID)
	storyboardCapacity := measureStoryboardAdmissionCapacity(
		t,
		store,
		inspector,
		catalogService,
		created.ID,
		assetCount,
	)
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
		GOOS:                    runtime.GOOS,
		GOARCH:                  runtime.GOARCH,
		GOMAXPROCS:              runtime.GOMAXPROCS(0),
		DirectoryCount:          directoryCount,
		AssetCount:              assetCount,
		FixtureMaxDepth:         fixtureShape.maxDepth,
		FixtureDeepBranches:     fixtureShape.deepBranches,
		FixtureAssetTargets:     fixtureShape.assetTargets,
		FixtureDurationMS:       fixtureDuration.Milliseconds(),
		ScanDurationMS:          scanDuration.Milliseconds(),
		ConcurrentReads:         len(readLatencies),
		ConcurrentReadP95US:     percentile(readLatencies, 95).Microseconds(),
		ConcurrentReadMaxUS:     percentile(readLatencies, 100).Microseconds(),
		ConcurrentSearches:      len(searchLatencies),
		ConcurrentSearchP95US:   percentile(searchLatencies, 95).Microseconds(),
		ConcurrentSearchMaxUS:   percentile(searchLatencies, 100).Microseconds(),
		DirectoryListP50US:      percentile(directoryLatencies, 50).Microseconds(),
		DirectoryListP95US:      percentile(directoryLatencies, 95).Microseconds(),
		AssetListP50US:          percentile(assetLatencies, 50).Microseconds(),
		AssetListP95US:          percentile(assetLatencies, 95).Microseconds(),
		RecursiveBrowseP50US:    percentile(recursiveBrowseLatencies, 50).Microseconds(),
		RecursiveBrowseP95US:    percentile(recursiveBrowseLatencies, 95).Microseconds(),
		FTSSearchP50US:          percentile(ftsSearchLatencies, 50).Microseconds(),
		FTSSearchP95US:          percentile(ftsSearchLatencies, 95).Microseconds(),
		ShortSearchP50US:        percentile(shortSearchLatencies, 50).Microseconds(),
		ShortSearchP95US:        percentile(shortSearchLatencies, 95).Microseconds(),
		GlobalSearchP95US:       percentile(globalSearchLatencies, 95).Microseconds(),
		SearchKeysetP95US:       percentile(keysetLatencies, 95).Microseconds(),
		SearchCancelLatencyUS:   cancelLatency.Microseconds(),
		SearchRebuildDurationMS: rebuildDuration.Milliseconds(),
		StoryboardVideoCount:    storyboardCapacity.videoCount,
		StoryboardAdmissionRuns: storyboardCapacity.admissionRuns,
		StoryboardAdmissionMax:  storyboardCapacity.admissionMax,
		StoryboardAdmissionMS:   storyboardCapacity.admissionDuration.Milliseconds(),
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
