package integration_test

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/jobs"
	"github.com/HappyQuQu/foliopath/internal/library"
	"github.com/HappyQuQu/foliopath/internal/scanner"
	sqlitestore "github.com/HappyQuQu/foliopath/internal/store/sqlite"
)

type integrationScanQueue struct {
	store *sqlitestore.Store
}

func (queue integrationScanQueue) RecoverExpired(
	ctx context.Context,
) (jobs.RecoverySummary, error) {
	return queue.store.RecoverExpiredFullScans(ctx)
}

func (queue integrationScanQueue) Claim(
	ctx context.Context,
	lease time.Duration,
) (scanner.ScanRun, bool, error) {
	return queue.store.ClaimNextFullScan(ctx, lease)
}

func (queue integrationScanQueue) RefreshLease(
	ctx context.Context,
	run scanner.ScanRun,
	lease time.Duration,
) (bool, error) {
	refreshed, err := queue.store.RefreshFullScanLease(ctx, run.ID, lease)
	if err != nil {
		return false, err
	}
	return refreshed.CancelRequestedAtMS != nil, nil
}

type boundedScanWalker struct {
	mutex     sync.Mutex
	roots     map[string]scanner.RootIdentity
	started   chan string
	release   <-chan struct{}
	active    int
	maxActive int
}

func (walker *boundedScanWalker) CaptureRoot(
	_ context.Context,
	relativeRoot string,
) (scanner.RootIdentity, error) {
	identity, ok := walker.roots[relativeRoot]
	if !ok {
		return scanner.RootIdentity{}, scanner.ErrLibraryNotFound
	}
	return identity, nil
}

func (walker *boundedScanWalker) Walk(
	ctx context.Context,
	relativeRoot string,
	_ func(scanner.WalkEntry) (scanner.WalkDecision, error),
) error {
	walker.mutex.Lock()
	walker.active++
	if walker.active > walker.maxActive {
		walker.maxActive = walker.active
	}
	walker.mutex.Unlock()
	walker.started <- relativeRoot
	select {
	case <-walker.release:
	case <-ctx.Done():
		return ctx.Err()
	}
	walker.mutex.Lock()
	walker.active--
	walker.mutex.Unlock()
	return nil
}

func (walker *boundedScanWalker) VerifyRoot(
	_ context.Context,
	relativeRoot string,
	expected scanner.RootIdentity,
) error {
	if walker.roots[relativeRoot] != expected {
		return scanner.ErrRootIdentityChanged
	}
	return nil
}

func TestDurableScanWorkerEnforcesCrossLibraryConcurrencyAndOrder(t *testing.T) {
	store, err := sqlitestore.Open(
		context.Background(),
		filepath.Join(t.TempDir(), "scan-worker-capacity.db"),
		sqlitestore.Options{},
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	libraries, err := library.NewService(store)
	if err != nil {
		t.Fatal(err)
	}

	roots := make(map[string]scanner.RootIdentity, 3)
	runs := make([]scanner.ScanRun, 0, 3)
	for index := 1; index <= 3; index++ {
		root := fmt.Sprintf("library-%d", index)
		record, err := libraries.Create(
			context.Background(),
			fmt.Sprintf("Library %d", index),
			root,
		)
		if err != nil {
			t.Fatal(err)
		}
		roots[root] = scanner.RootIdentity{Device: 1, Inode: uint64(index)}
		admitted, err := store.AdmitFullScan(
			context.Background(),
			record.ID,
			scanner.TriggerManual,
		)
		if err != nil {
			t.Fatal(err)
		}
		runs = append(runs, admitted.Run)
	}

	release := make(chan struct{})
	walker := &boundedScanWalker{
		roots:   roots,
		started: make(chan string, 3),
		release: release,
	}
	service, err := scanner.NewService(store, scanner.Config{})
	if err != nil {
		t.Fatal(err)
	}
	processor, err := scanner.NewClaimedProcessor(service, walker)
	if err != nil {
		t.Fatal(err)
	}
	signal := jobs.NewSignal()
	pool, err := jobs.NewWorkerPool(
		integrationScanQueue{store: store},
		processor,
		signal,
		jobs.WorkerOptions{
			HeartbeatInterval: 20 * time.Millisecond,
			LeaseDuration:     200 * time.Millisecond,
			IdlePollInterval:  5 * time.Millisecond,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() { result <- pool.Run(ctx) }()
	defer func() {
		cancel()
		if err := <-result; err != nil {
			t.Errorf("WorkerPool.Run() error = %v", err)
		}
	}()

	firstStarted := <-walker.started
	secondStarted := <-walker.started
	firstWave := map[string]bool{
		firstStarted:  true,
		secondStarted: true,
	}
	if !firstWave["library-1"] || !firstWave["library-2"] || len(firstWave) != 2 {
		t.Fatalf(
			"first claimed roots = %q, %q; want library-1 and library-2",
			firstStarted,
			secondStarted,
		)
	}
	select {
	case third := <-walker.started:
		t.Fatalf("third library %q started before worker capacity was released", third)
	case <-time.After(25 * time.Millisecond):
	}
	close(release)
	select {
	case third := <-walker.started:
		if third != "library-3" {
			t.Fatalf("third claimed root = %q, want library-3", third)
		}
	case <-time.After(time.Second):
		t.Fatal("third library did not start after worker capacity was released")
	}

	deadline := time.Now().Add(2 * time.Second)
	for _, run := range runs {
		for {
			current, err := store.GetScanRun(context.Background(), run.ID)
			if err != nil {
				t.Fatal(err)
			}
			if current.Status == scanner.RunStatusSucceeded {
				break
			}
			if time.Now().After(deadline) {
				t.Fatalf("scan %d status = %q, want succeeded", run.ID, current.Status)
			}
			time.Sleep(time.Millisecond)
		}
	}
	walker.mutex.Lock()
	maxActive := walker.maxActive
	walker.mutex.Unlock()
	if maxActive != jobs.DefaultWorkerCount {
		t.Fatalf("maximum active scans = %d, want %d", maxActive, jobs.DefaultWorkerCount)
	}
}
