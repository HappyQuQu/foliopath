package app

import (
	"bytes"
	"context"
	"errors"
	"io"

	"github.com/HappyQuQu/foliopath/internal/media"
	"github.com/HappyQuQu/foliopath/internal/semantic"
	"github.com/HappyQuQu/foliopath/internal/thumbnail"
)

type semanticStoryboardRepository interface {
	thumbnail.SemanticStoryboardRepository
	TouchThumbnail(context.Context, int64, thumbnail.Variant, media.SourceFingerprint, string) error
	RequeueMissingThumbnail(context.Context, thumbnail.DeliveryState) error
}

type semanticStoryboardSplitter interface {
	SplitSemanticStoryboard(context.Context, io.ReadSeeker, int, int, int, int) ([][]byte, error)
}

type semanticStoryboardSource struct {
	repository semanticStoryboardRepository
	cache      thumbnail.CacheReader
	splitter   semanticStoryboardSplitter
	waker      interface{ Wake() }
}

func (source semanticStoryboardSource) OpenCompleteStoryboard(
	ctx context.Context,
	libraryID, assetID int64,
) (semantic.CompleteStoryboard, error) {
	if source.repository == nil || source.cache == nil || source.splitter == nil {
		return semantic.CompleteStoryboard{}, semantic.ErrStoryboardNotReady
	}
	state, err := source.repository.GetCompleteSemanticStoryboard(ctx, libraryID, assetID)
	if err != nil {
		return semantic.CompleteStoryboard{}, errors.Join(semantic.ErrStoryboardNotReady, err)
	}
	content, err := source.cache.Open(ctx, state.CacheRelativePath)
	if errors.Is(err, thumbnail.ErrCacheEntryMissing) || err == nil && content.Reader != nil && content.ByteSize != state.ByteSize {
		if content.Reader != nil {
			_ = content.Reader.Close()
		}
		delivery := thumbnail.DeliveryState{AssetID: state.AssetID, Variant: thumbnail.VariantStoryboard,
			SourceFingerprint: state.SourceFingerprint, Status: thumbnail.DeliveryReady,
			CacheRelativePath: state.CacheRelativePath, ByteSize: state.ByteSize}
		if repairErr := source.repository.RequeueMissingThumbnail(ctx, delivery); repairErr != nil {
			return semantic.CompleteStoryboard{}, errors.Join(semantic.ErrStoryboardSourceChanged, repairErr)
		}
		if source.waker != nil {
			source.waker.Wake()
		}
		return semantic.CompleteStoryboard{}, semantic.ErrStoryboardNotReady
	}
	if err != nil || content.Reader == nil {
		if content.Reader != nil {
			_ = content.Reader.Close()
		}
		return semantic.CompleteStoryboard{}, errors.Join(semantic.ErrStoryboardNotReady, err)
	}
	cells, splitErr := source.splitter.SplitSemanticStoryboard(
		ctx, content.Reader, state.FrameCount, state.Columns, state.CellWidth, state.CellHeight,
	)
	_ = content.Reader.Close()
	if splitErr != nil || len(cells) != state.FrameCount {
		return semantic.CompleteStoryboard{}, errors.Join(semantic.ErrStoryboardNotReady, splitErr)
	}
	plan, err := thumbnail.NewStoryboardPlanWithFrameCount(state.DurationMS, state.FrameCount)
	if err != nil {
		return semantic.CompleteStoryboard{}, semantic.ErrStoryboardNotReady
	}
	fingerprint, err := semantic.StoryboardFingerprint(state.SourceFingerprint.String(), state.TransformVersion, state.FrameCount)
	if err != nil {
		return semantic.CompleteStoryboard{}, err
	}
	frames := make([]semantic.StoryboardFrame, state.FrameCount)
	for ordinal, value := range cells {
		if len(value) == 0 {
			closeStoryboardFrameReaders(frames)
			return semantic.CompleteStoryboard{}, semantic.ErrStoryboardNotReady
		}
		frames[ordinal] = semantic.StoryboardFrame{Ordinal: ordinal, TimestampMS: plan.TimestampsMS[ordinal],
			Format: media.FormatWebP, Image: readSeekNopCloser{Reader: bytes.NewReader(value)}}
	}
	if err := source.repository.TouchThumbnail(ctx, assetID, thumbnail.VariantStoryboard, state.SourceFingerprint, state.CacheRelativePath); err != nil {
		closeStoryboardFrameReaders(frames)
		return semantic.CompleteStoryboard{}, errors.Join(semantic.ErrStoryboardSourceChanged, err)
	}
	return semantic.CompleteStoryboard{LibraryID: libraryID, AssetID: assetID,
		SourceFingerprint: state.SourceFingerprint.String(), StoryboardFingerprint: fingerprint,
		TransformVersion: state.TransformVersion, PlanSize: state.FrameCount, Frames: frames}, nil
}

type readSeekNopCloser struct{ *bytes.Reader }

func (readSeekNopCloser) Close() error { return nil }

func closeStoryboardFrameReaders(frames []semantic.StoryboardFrame) {
	for _, frame := range frames {
		if frame.Image != nil {
			_ = frame.Image.Close()
		}
	}
}

var _ semantic.CompleteStoryboardSource = semanticStoryboardSource{}
