package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/scanner"
)

func TestScanQueriesListDetailsAndCancellation(t *testing.T) {
	now := time.UnixMilli(5_000)
	store := openTimedScanStore(t, &now)
	ctx := context.Background()
	libraryRecord := createWorkerLibrary(t, store, "Queries", "queries")
	first, err := store.AdmitFullScan(ctx, libraryRecord.ID, scanner.TriggerManual)
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Millisecond)
	if _, err := store.RequestScanCancellation(ctx, first.Run.ID); err != nil {
		t.Fatal(err)
	}
	cancelled, err := store.GetScanDetails(ctx, first.Run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if cancelled.Run.Status != scanner.RunStatusCancelled ||
		cancelled.Run.CancelRequestedAtMS == nil ||
		cancelled.Run.FinishedAtMS == nil {
		t.Fatalf("cancelled details = %#v", cancelled)
	}
	if _, err := store.RequestScanCancellation(ctx, first.Run.ID); !errors.Is(err, scanner.ErrScanAlreadyFinished) {
		t.Fatalf("repeat terminal cancellation error = %v", err)
	}

	now = now.Add(time.Millisecond)
	second, err := store.AdmitFullScan(ctx, libraryRecord.ID, scanner.TriggerManual)
	if err != nil {
		t.Fatal(err)
	}
	running, found, err := store.ClaimNextFullScan(ctx, time.Minute)
	if err != nil || !found || running.ID != second.Run.ID {
		t.Fatalf("claimed run = %#v, found %t, err %v", running, found, err)
	}
	requested, err := store.RequestScanCancellation(ctx, running.ID)
	if err != nil {
		t.Fatal(err)
	}
	repeated, err := store.RequestScanCancellation(ctx, running.ID)
	if err != nil {
		t.Fatal(err)
	}
	if requested.CancelRequestedAtMS == nil || repeated.Revision != requested.Revision {
		t.Fatalf("idempotent running cancellation = first %#v, second %#v", requested, repeated)
	}

	page, err := store.ListScanRuns(
		ctx,
		libraryRecord.ID,
		scanner.QueryPosition{CreatedAtMS: 1<<63 - 1, ID: 1<<63 - 1},
		3,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(page) != 2 || page[0].ID != second.Run.ID || page[1].ID != first.Run.ID {
		t.Fatalf("scan history = %#v", page)
	}
	if _, err := store.ListScanRuns(
		ctx, libraryRecord.ID+99,
		scanner.QueryPosition{CreatedAtMS: 1<<63 - 1, ID: 1<<63 - 1}, 1,
	); !errors.Is(err, scanner.ErrLibraryNotFound) {
		t.Fatalf("missing library list error = %v", err)
	}
}
