package performance_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/files"
	"github.com/HappyQuQu/foliopath/internal/library"
	"github.com/HappyQuQu/foliopath/internal/scanner"
	sqlitestore "github.com/HappyQuQu/foliopath/internal/store/sqlite"
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
	DirectoryListP50US      int64    `json:"directoryListP50Us"`
	DirectoryListP95US      int64    `json:"directoryListP95Us"`
	AssetListP50US          int64    `json:"assetListP50Us"`
	AssetListP95US          int64    `json:"assetListP95Us"`
	PeakGoHeapAllocBytes    uint64   `json:"peakGoHeapAllocBytes"`
	PeakRSSBytes            uint64   `json:"peakRssBytes,omitempty"`
	PeakRSSSource           string   `json:"peakRssSource,omitempty"`
	DatabaseAndWALSizeBytes int64    `json:"databaseAndWalSizeBytes"`
	BudgetProfile           string   `json:"budgetProfile"`
	BudgetViolations        []string `json:"budgetViolations"`
}

const capacityBudgetProfile = "stage0-comparable-v1"

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
	scans, err := scanner.NewService(store, scanner.Config{
		BatchSize:       500,
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
		DirectoryListP50US:      percentile(directoryLatencies, 50).Microseconds(),
		DirectoryListP95US:      percentile(directoryLatencies, 95).Microseconds(),
		AssetListP50US:          percentile(assetLatencies, 50).Microseconds(),
		AssetListP95US:          percentile(assetLatencies, 95).Microseconds(),
		PeakGoHeapAllocBytes:    peakHeap.Load(),
		PeakRSSBytes:            peakRSS,
		PeakRSSSource:           peakRSSSource,
		DatabaseAndWALSizeBytes: databaseFamilySize(t, databasePath),
		BudgetProfile:           capacityBudgetProfile,
	}
	metrics.BudgetViolations = capacityBudgetViolations(metrics)
	encoded, err := json.Marshal(metrics)
	if err != nil {
		t.Fatalf("encode capacity metrics: %v", err)
	}
	t.Logf("FOLIOPATH_CAPACITY_METRICS %s", encoded)
	if os.Getenv("FOLIOPATH_CAPACITY_ENFORCE_BUDGET") == "1" &&
		len(metrics.BudgetViolations) > 0 {
		t.Fatalf("%s budget violations: %v", capacityBudgetProfile, metrics.BudgetViolations)
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
	completed, err := store.CompleteFullScan(context.Background(), run.ID, 0)
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
		DirectoryListP95US:      100_000,
		AssetListP95US:          100_000,
		PeakRSSBytes:            1 << 30,
		DatabaseAndWALSizeBytes: 1 << 30,
	}); len(violations) != 0 {
		t.Fatalf("budget boundary violations = %v, want none", violations)
	}

	violations := capacityBudgetViolations(capacityMetrics{
		ScanDurationMS:          120_001,
		ConcurrentReadP95US:     250_001,
		DirectoryListP95US:      100_001,
		AssetListP95US:          100_001,
		PeakRSSBytes:            1<<30 + 1,
		DatabaseAndWALSizeBytes: 1<<30 + 1,
	})
	if len(violations) != 6 {
		t.Fatalf("budget violations = %v, want all six limits", violations)
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
	if metrics.DirectoryListP95US > 100_000 {
		violations = append(violations, "directoryListP95Us > 100000")
	}
	if metrics.AssetListP95US > 100_000 {
		violations = append(violations, "assetListP95Us > 100000")
	}
	if metrics.PeakRSSBytes > 0 && metrics.PeakRSSBytes > 1<<30 {
		violations = append(violations, "peakRssBytes > 1073741824")
	}
	if metrics.DatabaseAndWALSizeBytes > 1<<30 {
		violations = append(violations, "databaseAndWalSizeBytes > 1073741824")
	}
	return violations
}
