package app

import (
	"context"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/scanner"
)

type mediaWakeRepositoryStub struct {
	scanner.Repository
	completed scanner.ScanRun
}

func (stub mediaWakeRepositoryStub) CompleteFullScan(
	context.Context,
	int64,
	scanner.SkipCounts,
) (scanner.ScanRun, error) {
	return stub.completed, nil
}

type countingWaker struct{ calls int }

func (waker *countingWaker) Wake() { waker.calls++ }

func TestSuccessfulFullScanWakesCacheAndDiscoveryRecovery(t *testing.T) {
	media := &countingWaker{}
	cache := &countingWaker{}
	discovery := &countingWaker{}
	repository := mediaWakeScanRepository{
		Repository: mediaWakeRepositoryStub{
			completed: scanner.ScanRun{ID: 7, Status: scanner.RunStatusSucceeded},
		},
		waker:          media,
		cacheWaker:     cache,
		discoveryWaker: discovery,
	}

	if _, err := repository.CompleteFullScan(
		context.Background(),
		7,
		scanner.SkipCounts{},
	); err != nil {
		t.Fatalf("CompleteFullScan() error = %v", err)
	}
	if cache.calls != 1 || discovery.calls != 1 {
		t.Fatalf(
			"wake counts cache=%d discovery=%d, want 1 each",
			cache.calls,
			discovery.calls,
		)
	}
	if media.calls != 0 {
		t.Fatalf("media wake after full-scan completion = %d, want 0", media.calls)
	}
}
