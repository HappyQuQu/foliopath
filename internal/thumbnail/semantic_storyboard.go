package thumbnail

import (
	"context"
	"errors"

	"github.com/HappyQuQu/foliopath/internal/media"
)

var ErrSemanticStoryboardNotReady = errors.New("semantic storyboard not ready")

type SemanticStoryboardState struct {
	LibraryID         int64
	AssetID           int64
	SourceFingerprint media.SourceFingerprint
	DurationMS        int64
	TransformVersion  int
	CacheRelativePath string
	ByteSize          int64
	FrameCount        int
	Columns           int
	Rows              int
	CellWidth         int
	CellHeight        int
}

type SemanticStoryboardRepository interface {
	GetCompleteSemanticStoryboard(context.Context, int64, int64) (SemanticStoryboardState, error)
}

func (value SemanticStoryboardState) Validate() error {
	if value.LibraryID < 1 || value.AssetID < 1 || !value.SourceFingerprint.Valid() ||
		value.DurationMS < StoryboardMinimumDurationMS || value.TransformVersion != StoryboardTransformVersion ||
		value.CacheRelativePath == "" || value.ByteSize < 1 ||
		(value.FrameCount != StoryboardShortFrameCount && value.FrameCount != StoryboardLongFrameCount) ||
		value.Columns != min(value.FrameCount, StoryboardMaximumColumns) ||
		value.Rows != (value.FrameCount+value.Columns-1)/value.Columns ||
		value.CellWidth < 1 || value.CellWidth > StoryboardMaximumCellPixels ||
		value.CellHeight < 1 || value.CellHeight > StoryboardMaximumCellPixels {
		return ErrSemanticStoryboardNotReady
	}
	return nil
}

func NewStoryboardPlanWithFrameCount(durationMS int64, frameCount int) (StoryboardPlan, error) {
	return newStoryboardPlan(durationMS, frameCount)
}
