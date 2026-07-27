package scanner

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type queryRepositoryStub struct {
	runs    []ScanRun
	details map[int64]Details
}

func (stub queryRepositoryStub) ListScanRuns(
	_ context.Context,
	libraryID int64,
	before QueryPosition,
	limit int,
) ([]ScanRun, error) {
	items := make([]ScanRun, 0, limit)
	for _, run := range stub.runs {
		if run.LibraryID == libraryID &&
			(run.CreatedAtMS < before.CreatedAtMS ||
				run.CreatedAtMS == before.CreatedAtMS && run.ID < before.ID) {
			items = append(items, run)
			if len(items) == limit {
				break
			}
		}
	}
	return items, nil
}

func (stub queryRepositoryStub) GetScanDetails(_ context.Context, scanID int64) (Details, error) {
	item, ok := stub.details[scanID]
	if !ok {
		return Details{}, ErrScanRunNotFound
	}
	return item, nil
}

func (stub queryRepositoryStub) RequestScanCancellation(_ context.Context, scanID int64) (ScanRun, error) {
	item, ok := stub.details[scanID]
	if !ok {
		return ScanRun{}, ErrScanRunNotFound
	}
	return item.Run, nil
}

func TestQueryServiceCursorIsBoundToLibraryAndTamperEvident(t *testing.T) {
	runs := []ScanRun{
		{ID: 3, LibraryID: 7, CreatedAtMS: 30},
		{ID: 2, LibraryID: 7, CreatedAtMS: 20},
		{ID: 1, LibraryID: 7, CreatedAtMS: 10},
	}
	details := make(map[int64]Details, len(runs))
	for _, run := range runs {
		details[run.ID] = Details{Run: run, Issues: []Issue{}}
	}
	service, err := NewQueryService(
		queryRepositoryStub{runs: runs, details: details},
		make([]byte, 32),
	)
	if err != nil {
		t.Fatal(err)
	}
	first, err := service.List(context.Background(), 7, "", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Items) != 2 || first.NextCursor == "" {
		t.Fatalf("first page = %#v", first)
	}
	second, err := service.List(context.Background(), 7, first.NextCursor, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Items) != 1 || second.Items[0].Run.ID != 1 || second.NextCursor != "" {
		t.Fatalf("second page = %#v", second)
	}
	if _, err := service.List(context.Background(), 8, first.NextCursor, 2); !errors.Is(err, ErrInvalidScanCursor) {
		t.Fatalf("cross-library cursor error = %v", err)
	}
	replacement := "A"
	if first.NextCursor[0] == 'A' {
		replacement = "B"
	}
	tampered := replacement + first.NextCursor[1:]
	if _, err := service.List(context.Background(), 7, tampered, 2); !errors.Is(err, ErrInvalidScanCursor) {
		t.Fatalf("tampered cursor error = %v", err)
	}
	if _, err := service.List(
		context.Background(), 7, strings.Repeat("a", MaxScanCursorBytes+1), 2,
	); !errors.Is(err, ErrInvalidScanCursor) {
		t.Fatalf("oversized cursor error = %v", err)
	}
}
