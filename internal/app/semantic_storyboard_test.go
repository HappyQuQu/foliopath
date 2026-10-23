package app

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/media"
	"github.com/HappyQuQu/foliopath/internal/semantic"
	"github.com/HappyQuQu/foliopath/internal/thumbnail"
)

type semanticStoryboardRepositoryStub struct {
	state    thumbnail.SemanticStoryboardState
	requeued int
	touched  int
}

func (stub *semanticStoryboardRepositoryStub) GetCompleteSemanticStoryboard(context.Context, int64, int64) (thumbnail.SemanticStoryboardState, error) {
	return stub.state, nil
}
func (stub *semanticStoryboardRepositoryStub) TouchThumbnail(context.Context, int64, thumbnail.Variant, media.SourceFingerprint, string) error {
	stub.touched++
	return nil
}
func (stub *semanticStoryboardRepositoryStub) RequeueMissingThumbnail(_ context.Context, state thumbnail.DeliveryState) error {
	if state.AssetID != stub.state.AssetID || state.Variant != thumbnail.VariantStoryboard || state.Status != thumbnail.DeliveryReady {
		return thumbnail.ErrInvalidState
	}
	stub.requeued++
	return nil
}

type missingStoryboardCache struct{}

func (missingStoryboardCache) Open(context.Context, string) (thumbnail.CacheContent, error) {
	return thumbnail.CacheContent{}, thumbnail.ErrCacheEntryMissing
}

type unusedStoryboardSplitter struct{}

func (unusedStoryboardSplitter) SplitSemanticStoryboard(context.Context, io.ReadSeeker, int, int, int, int) ([][]byte, error) {
	return nil, errors.New("unexpected split")
}

type semanticStoryboardWakeStub struct{ calls int }

func (stub *semanticStoryboardWakeStub) Wake() { stub.calls++ }

func TestSemanticStoryboardSourceRequeuesEvictedPublishedCache(t *testing.T) {
	repository := &semanticStoryboardRepositoryStub{state: thumbnail.SemanticStoryboardState{
		LibraryID: 1, AssetID: 2, SourceFingerprint: media.SourceFingerprint("v1:42:100"), DurationMS: 10_000,
		TransformVersion: thumbnail.StoryboardTransformVersion, CacheRelativePath: "libraries/lib_1/story.webp",
		ByteSize: 100, FrameCount: 4, Columns: 4, Rows: 1, CellWidth: 320, CellHeight: 180,
	}}
	waker := &semanticStoryboardWakeStub{}
	source := semanticStoryboardSource{repository: repository, cache: missingStoryboardCache{}, splitter: unusedStoryboardSplitter{}, waker: waker}
	_, err := source.OpenCompleteStoryboard(context.Background(), 1, 2)
	if !errors.Is(err, semantic.ErrStoryboardNotReady) || repository.requeued != 1 || repository.touched != 0 || waker.calls != 1 {
		t.Fatalf("err=%v repository=%#v wakes=%d", err, repository, waker.calls)
	}
}
