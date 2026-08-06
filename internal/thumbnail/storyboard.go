package thumbnail

import (
	"errors"
	"math"

	"github.com/HappyQuQu/foliopath/internal/media"
)

const (
	StoryboardMinimumDurationMS  = int64(2_000)
	StoryboardTenFrameDurationMS = int64(5_000)
	StoryboardShortFrameCount    = 4
	StoryboardLongFrameCount     = 10
	StoryboardMaximumColumns     = 5
	StoryboardMaximumCellPixels  = media.StoryboardMaxCellDimension
)

var (
	ErrStoryboardNotEligible     = errors.New("storyboard not eligible")
	ErrStoryboardBudgetExhausted = errors.New("storyboard processing budget exhausted")
	ErrInvalidStoryboardPlan     = errors.New("invalid storyboard plan")
	ErrInvalidStoryboardLayout   = errors.New("invalid storyboard layout")
)

type StoryboardPlan struct {
	DurationMS   int64
	TimestampsMS []int64
	Columns      int
	Rows         int
}

type StoryboardLayout struct {
	FrameCount int
	Columns    int
	Rows       int
	CellWidth  int
	CellHeight int
}

func StoryboardEligible(
	kind media.Kind,
	probeStatus media.ProbeStatus,
	durationMS *int64,
	gridReady bool,
) bool {
	return kind == media.KindVideo &&
		probeStatus == media.ProbeReady &&
		durationMS != nil &&
		*durationMS >= StoryboardMinimumDurationMS &&
		gridReady
}

func NewStoryboardPlan(durationMS int64) (StoryboardPlan, error) {
	if durationMS < StoryboardMinimumDurationMS {
		return StoryboardPlan{}, ErrStoryboardNotEligible
	}

	frameCount := StoryboardShortFrameCount
	if durationMS >= StoryboardTenFrameDurationMS {
		frameCount = StoryboardLongFrameCount
	}
	return newStoryboardPlan(durationMS, frameCount)
}

func newStoryboardPlan(durationMS int64, frameCount int) (StoryboardPlan, error) {
	if durationMS < StoryboardMinimumDurationMS ||
		(frameCount != StoryboardShortFrameCount &&
			frameCount != StoryboardLongFrameCount) {
		return StoryboardPlan{}, ErrInvalidStoryboardPlan
	}
	columns := min(frameCount, StoryboardMaximumColumns)
	plan := StoryboardPlan{
		DurationMS:   durationMS,
		TimestampsMS: make([]int64, frameCount),
		Columns:      columns,
		Rows:         (frameCount + columns - 1) / columns,
	}
	divisor := int64(frameCount + 1)
	quotient, remainder := durationMS/divisor, durationMS%divisor
	for index := range plan.TimestampsMS {
		position := int64(index + 1)
		plan.TimestampsMS[index] =
			quotient*position + remainder*position/divisor
	}
	if err := plan.Validate(); err != nil {
		return StoryboardPlan{}, err
	}
	return plan, nil
}

func StoryboardCellDimensions(width, height int) (int, int, error) {
	if media.ValidateDimensions(width, height) != nil {
		return 0, 0, ErrInvalidStoryboardLayout
	}
	scale := math.Min(
		float64(StoryboardMaximumCellPixels)/float64(width),
		float64(StoryboardMaximumCellPixels)/float64(height),
	)
	if scale > 1 {
		scale = 1
	}
	cellWidth := max(1, int(math.Round(float64(width)*scale)))
	cellHeight := max(1, int(math.Round(float64(height)*scale)))
	if (StoryboardLayout{
		FrameCount: StoryboardShortFrameCount,
		Columns:    StoryboardShortFrameCount,
		Rows:       1,
		CellWidth:  cellWidth,
		CellHeight: cellHeight,
	}).Validate() != nil {
		return 0, 0, ErrInvalidStoryboardLayout
	}
	return cellWidth, cellHeight, nil
}

func (plan StoryboardPlan) Validate() error {
	if plan.DurationMS < StoryboardMinimumDurationMS ||
		(len(plan.TimestampsMS) != StoryboardShortFrameCount &&
			len(plan.TimestampsMS) != StoryboardLongFrameCount) ||
		plan.Columns != min(len(plan.TimestampsMS), StoryboardMaximumColumns) ||
		plan.Rows != (len(plan.TimestampsMS)+plan.Columns-1)/plan.Columns {
		return ErrInvalidStoryboardPlan
	}
	var previous int64
	for _, timestamp := range plan.TimestampsMS {
		if timestamp <= previous || timestamp >= plan.DurationMS {
			return ErrInvalidStoryboardPlan
		}
		previous = timestamp
	}
	return nil
}

func (layout StoryboardLayout) Validate() error {
	if (layout.FrameCount != StoryboardShortFrameCount &&
		layout.FrameCount != StoryboardLongFrameCount) ||
		layout.Columns != min(layout.FrameCount, StoryboardMaximumColumns) ||
		layout.Rows != (layout.FrameCount+layout.Columns-1)/layout.Columns ||
		layout.CellWidth < 1 ||
		layout.CellWidth > StoryboardMaximumCellPixels ||
		layout.CellHeight < 1 ||
		layout.CellHeight > StoryboardMaximumCellPixels {
		return ErrInvalidStoryboardLayout
	}
	return nil
}
