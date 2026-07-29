//go:build linux

package performance_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/catalog"
	"github.com/HappyQuQu/foliopath/internal/files"
	"github.com/HappyQuQu/foliopath/internal/library"
	"github.com/HappyQuQu/foliopath/internal/scanner"
	sqlitestore "github.com/HappyQuQu/foliopath/internal/store/sqlite"
)

const automaticDiscoveryCapacityChanges = 100

type automaticDiscoveryCapacityMetrics struct {
	GOARCH                  string `json:"goarch"`
	DirectoryCount          int    `json:"directoryCount"`
	AssetCount              int    `json:"assetCount"`
	ChangedDirectories      int    `json:"changedDirectories"`
	WatchRegistrationMS     int64  `json:"watchRegistrationMs"`
	EventCollectionMS       int64  `json:"eventCollectionMs"`
	ReconcileEndToEndMS     int64  `json:"reconcileEndToEndMs"`
	ReconcileP50US          int64  `json:"reconcileP50Us"`
	ReconcileP95US          int64  `json:"reconcileP95Us"`
	ReconcileMaxUS          int64  `json:"reconcileMaxUs"`
	DatabaseAllocatedGrowth int64  `json:"databaseAllocatedGrowthBytes"`
	ProcessWriteBytes       int64  `json:"processWriteBytes"`
	WriteBytesPerChange     int64  `json:"writeBytesPerChange"`
}

type atomicCapacityWaker struct{ count atomic.Int64 }

func (waker *atomicCapacityWaker) Wake() {
	waker.count.Add(1)
}

func TestAutomaticDiscoveryCapacity(t *testing.T) {
	if os.Getenv("FOLIOPATH_AUTOMATIC_DISCOVERY_CAPACITY") != "1" {
		t.Skip(
			"set FOLIOPATH_AUTOMATIC_DISCOVERY_CAPACITY=1 to run the explicit watcher capacity profile",
		)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	base := t.TempDir()
	mediaRoot := filepath.Join(base, "library")
	if err := os.MkdirAll(mediaRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	createCapacityFixture(t, mediaRoot, defaultDirectoryCount, defaultAssetCount)

	root, err := files.OpenRoot(mediaRoot)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = root.Close() })
	walker, err := files.NewScanWalker(root)
	if err != nil {
		t.Fatal(err)
	}
	databasePath := filepath.Join(base, "foliopath.db")
	store, err := sqlitestore.Open(ctx, databasePath, sqlitestore.Options{
		BusyTimeout:        5 * time.Second,
		MaxOpenConnections: 4,
		MaxBatchSize:       500,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	libraries, err := library.NewService(store)
	if err != nil {
		t.Fatal(err)
	}
	item, err := libraries.Create(ctx, "Automatic discovery capacity", "")
	if err != nil {
		t.Fatal(err)
	}
	fullScanner, err := scanner.NewService(store, scanner.Config{
		BatchSize:       scanner.DefaultBatchSize,
		FinalizeTimeout: 10 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fullScanner.RunFullScan(ctx, scanner.FullScanRequest{
		LibraryID: item.ID,
		Trigger:   scanner.TriggerCreation,
		Walker:    walker,
	}); err != nil {
		t.Fatal(err)
	}
	catalogService, err := catalog.NewService(store, nil)
	if err != nil {
		t.Fatal(err)
	}
	revisionBefore, err := catalogService.ContentRevision(ctx)
	if err != nil {
		t.Fatal(err)
	}

	watcher, err := files.NewLibraryWatcher(root, files.WatcherOptions{
		MaxWatches:  scanner.MaxDirectoryWatches,
		EventBuffer: scanner.MaxPendingWatchEvents,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = watcher.Close() })
	watchStarted := time.Now()
	if err := watcher.WatchLibrary(ctx, item.ID, ""); err != nil {
		t.Fatal(err)
	}
	watchRegistration := time.Since(watchStarted)
	watchContext, cancelWatch := context.WithCancel(ctx)
	runResult := make(chan error, 1)
	go func() {
		runResult <- watcher.Run(watchContext)
	}()
	t.Cleanup(func() {
		cancelWatch()
		select {
		case err := <-runResult:
			if err != nil {
				t.Errorf("watcher run: %v", err)
			}
		case <-time.After(time.Second):
			t.Error("watcher did not stop")
		}
	})

	changedDirectories := automaticDiscoveryCapacityDirectories()
	mutationStarted := time.Now()
	for index, relativeDirectory := range changedDirectories {
		filename := filepath.Join(
			mediaRoot,
			relativeDirectory,
			fmt.Sprintf("automatic-%03d.jpg", index),
		)
		if err := os.WriteFile(filename, []byte{0xff, 0xd8, 0xff, 0xd9}, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	dirty := make(map[string]struct{}, len(changedDirectories))
	for len(dirty) < len(changedDirectories) {
		select {
		case event := <-watcher.Events():
			if event.Kind != scanner.WatchEventDirty ||
				event.LibraryID != item.ID {
				t.Fatalf("unexpected capacity watch event %#v", event)
			}
			dirty[event.RelativeDirectory] = struct{}{}
		case <-ctx.Done():
			t.Fatalf(
				"collect capacity watch events: got %d/%d: %v",
				len(dirty),
				len(changedDirectories),
				ctx.Err(),
			)
		}
	}
	eventCollection := time.Since(mutationStarted)
	for relativeDirectory := range dirty {
		if _, err := store.EnqueueReconcile(
			ctx,
			item.ID,
			relativeDirectory,
			scanner.ReconcileDebounce,
			scanner.ReconcileMaximumDebounce,
		); err != nil {
			t.Fatal(err)
		}
	}
	databaseBefore := databaseFamilySize(t, databasePath)
	processWritesBefore := procSelfWChar(t)
	time.Sleep(scanner.ReconcileDebounce + 25*time.Millisecond)

	waker := &atomicCapacityWaker{}
	processor, err := scanner.NewReconcileProcessor(store, walker, waker, nil)
	if err != nil {
		t.Fatal(err)
	}
	latencies := make([]time.Duration, 0, len(dirty))
	var (
		latencyMu sync.Mutex
		processed atomic.Int64
	)
	workerErrors := make(chan error, scanner.MaxConcurrentReconciles)
	for worker := 0; worker < scanner.MaxConcurrentReconciles; worker++ {
		go func() {
			for processed.Load() < int64(len(dirty)) {
				job, found, err := store.ClaimNextReconcile(ctx, time.Minute)
				if err != nil {
					workerErrors <- err
					return
				}
				if !found {
					if processed.Load() >= int64(len(dirty)) {
						break
					}
					time.Sleep(time.Millisecond)
					continue
				}
				started := time.Now()
				if err := processor.Process(ctx, job); err != nil {
					workerErrors <- err
					return
				}
				latencyMu.Lock()
				latencies = append(latencies, time.Since(started))
				latencyMu.Unlock()
				processed.Add(1)
			}
			workerErrors <- nil
		}()
	}
	for worker := 0; worker < scanner.MaxConcurrentReconciles; worker++ {
		if err := <-workerErrors; err != nil {
			t.Fatal(err)
		}
	}
	reconcileEndToEnd := time.Since(mutationStarted)
	sort.Slice(latencies, func(left int, right int) bool {
		return latencies[left] < latencies[right]
	})
	if len(latencies) != len(dirty) ||
		waker.count.Load() != int64(len(dirty)) {
		t.Fatalf(
			"processed=%d latencies=%d wakes=%d dirty=%d",
			processed.Load(),
			len(latencies),
			waker.count.Load(),
			len(dirty),
		)
	}
	revisionAfter, err := catalogService.ContentRevision(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if revisionAfter != revisionBefore+int64(len(dirty)) {
		t.Fatalf(
			"catalog revision = %d -> %d, want +%d",
			revisionBefore,
			revisionAfter,
			len(dirty),
		)
	}
	searchQuery := "automatic"
	page, err := catalogService.ListAssets(ctx, catalog.AssetRequest{
		LibraryID:   item.ID,
		SearchQuery: &searchQuery,
		Limit:       automaticDiscoveryCapacityChanges,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != automaticDiscoveryCapacityChanges {
		t.Fatalf(
			"post-reconcile catalog search has %d items, want %d",
			len(page.Items),
			automaticDiscoveryCapacityChanges,
		)
	}
	databaseGrowth := databaseFamilySize(t, databasePath) - databaseBefore
	processWrites := procSelfWChar(t) - processWritesBefore
	metrics := automaticDiscoveryCapacityMetrics{
		GOARCH:                  runtime.GOARCH,
		DirectoryCount:          defaultDirectoryCount,
		AssetCount:              defaultAssetCount,
		ChangedDirectories:      len(dirty),
		WatchRegistrationMS:     watchRegistration.Milliseconds(),
		EventCollectionMS:       eventCollection.Milliseconds(),
		ReconcileEndToEndMS:     reconcileEndToEnd.Milliseconds(),
		ReconcileP50US:          percentile(latencies, 50).Microseconds(),
		ReconcileP95US:          percentile(latencies, 95).Microseconds(),
		ReconcileMaxUS:          percentile(latencies, 100).Microseconds(),
		DatabaseAllocatedGrowth: databaseGrowth,
		ProcessWriteBytes:       processWrites,
		WriteBytesPerChange:     processWrites / int64(len(dirty)),
	}
	encoded, err := json.Marshal(metrics)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("FOLIOPATH_AUTOMATIC_DISCOVERY_CAPACITY_METRICS %s", encoded)
	if watchRegistration > 30*time.Second {
		t.Fatalf("watch registration %s exceeds 30s budget", watchRegistration)
	}
	if eventCollection > 5*time.Second {
		t.Fatalf("event collection %s exceeds 5s budget", eventCollection)
	}
	if reconcileEndToEnd > 10*time.Second {
		t.Fatalf("reconcile end-to-end %s exceeds 10s budget", reconcileEndToEnd)
	}
	if processWrites > int64(len(dirty))*1024*1024 {
		t.Fatalf(
			"process writes %d exceed 1 MiB per changed directory budget",
			processWrites,
		)
	}
}

func procSelfWChar(t *testing.T) int64 {
	t.Helper()
	raw, err := os.ReadFile("/proc/self/io")
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(raw), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 || fields[0] != "wchar:" {
			continue
		}
		value, err := strconv.ParseInt(fields[1], 10, 64)
		if err != nil {
			t.Fatal(err)
		}
		return value
	}
	t.Fatal("/proc/self/io does not contain wchar")
	return 0
}

func automaticDiscoveryCapacityDirectories() []string {
	const groupCount = 97
	result := make([]string, 0, automaticDiscoveryCapacityChanges)
	for index := 0; index < automaticDiscoveryCapacityChanges; index++ {
		result = append(result, filepath.Join(
			fmt.Sprintf("group-%03d", index%groupCount),
			fmt.Sprintf("directory-%05d", index),
		))
	}
	return result
}
