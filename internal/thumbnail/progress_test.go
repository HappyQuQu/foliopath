package thumbnail

import (
	"context"
	"errors"
	"testing"
)

type progressRepositoryStub struct {
	progress ProcessingProgress
	found    bool
	err      error
}

func (stub progressRepositoryStub) GetMediaProcessingProgress(
	context.Context,
	int64,
) (ProcessingProgress, bool, error) {
	return stub.progress, stub.found, stub.err
}

func TestProgressServiceReturnsAggregateAndStableActivity(t *testing.T) {
	want := ProcessingProgress{
		Grid:                         JobProgress{Queued: 2, Running: 1, Succeeded: 7, Failed: 1},
		Storyboard:                   JobProgress{Succeeded: 3},
		StoryboardPendingEligibility: 1,
	}
	service, err := NewProgressService(progressRepositoryStub{progress: want, found: true})
	if err != nil {
		t.Fatalf("construct progress service: %v", err)
	}
	got, err := service.Get(context.Background(), 7)
	if err != nil || got != want {
		t.Fatalf("progress = %#v, %v", got, err)
	}
	if got.Grid.Total() != 11 || got.Grid.Processed() != 8 || !got.Active() {
		t.Fatalf("derived progress helpers = %#v", got)
	}
}

func TestProgressServiceRejectsMissingLibrary(t *testing.T) {
	service, err := NewProgressService(progressRepositoryStub{})
	if err != nil {
		t.Fatalf("construct progress service: %v", err)
	}
	for _, libraryID := range []int64{0, 7} {
		_, err := service.Get(context.Background(), libraryID)
		if !errors.Is(err, ErrProgressLibraryNotFound) {
			t.Fatalf("library %d error = %v", libraryID, err)
		}
	}
}
