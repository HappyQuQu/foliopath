package library

import (
	"context"
	"testing"
)

type removalRepositoryStub struct {
	removal      Removal
	found        bool
	ready        bool
	cleanupDone  bool
	cleanupCalls int
}

func (stub *removalRepositoryStub) ClaimNextLibraryRemoval(
	context.Context,
) (Removal, bool, error) {
	return stub.removal, stub.found, nil
}

func (stub *removalRepositoryStub) LibraryRemovalReady(context.Context, int64) (bool, error) {
	return stub.ready, nil
}

func (stub *removalRepositoryStub) CleanupLibraryRemovalBatch(
	context.Context,
	int64,
	int,
) (bool, error) {
	stub.cleanupCalls++
	return stub.cleanupDone, nil
}

func (*removalRepositoryStub) FailLibraryRemoval(context.Context, int64, string) error {
	return nil
}

type cacheCleanerStub struct {
	libraryIDs []int64
}

func (stub *cacheCleanerStub) RemoveLibraryCache(_ context.Context, id int64) error {
	stub.libraryIDs = append(stub.libraryIDs, id)
	return nil
}

func TestRemovalWorkerWaitsForScanThenCleansOnlyDerivedPorts(t *testing.T) {
	repository := &removalRepositoryStub{
		removal: Removal{ID: 3, LibraryID: 7, Status: RemovalRunning},
		found:   true,
	}
	cache := &cacheCleanerStub{}
	worker, err := NewRemovalWorker(repository, cache, 5)
	if err != nil {
		t.Fatal(err)
	}
	progressed, err := worker.runOne(context.Background())
	if err != nil || progressed || len(cache.libraryIDs) != 0 || repository.cleanupCalls != 0 {
		t.Fatalf("waiting run = progressed %t, err %v, cache %#v, cleanup %d",
			progressed, err, cache.libraryIDs, repository.cleanupCalls)
	}

	repository.ready = true
	repository.cleanupDone = true
	progressed, err = worker.runOne(context.Background())
	if err != nil || progressed || len(cache.libraryIDs) != 1 ||
		cache.libraryIDs[0] != 7 || repository.cleanupCalls != 1 {
		t.Fatalf("cleanup run = progressed %t, err %v, cache %#v, cleanup %d",
			progressed, err, cache.libraryIDs, repository.cleanupCalls)
	}
}
