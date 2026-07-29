package thumbnail

import (
	"errors"
	"math"
	"slices"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/media"
)

func TestStoryboardEligibility(t *testing.T) {
	t.Parallel()

	duration := int64(2_000)
	tests := []struct {
		name      string
		kind      media.Kind
		probe     media.ProbeStatus
		duration  *int64
		gridReady bool
		want      bool
	}{
		{"eligible", media.KindVideo, media.ProbeReady, &duration, true, true},
		{"not video", media.KindImage, media.ProbeReady, &duration, true, false},
		{"probe pending", media.KindVideo, media.ProbePending, &duration, true, false},
		{"duration absent", media.KindVideo, media.ProbeReady, nil, true, false},
		{"grid pending", media.KindVideo, media.ProbeReady, &duration, false, false},
	}
	short := StoryboardMinimumDurationMS - 1
	tests = append(tests, struct {
		name      string
		kind      media.Kind
		probe     media.ProbeStatus
		duration  *int64
		gridReady bool
		want      bool
	}{"too short", media.KindVideo, media.ProbeReady, &short, true, false})

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := StoryboardEligible(
				test.kind,
				test.probe,
				test.duration,
				test.gridReady,
			); got != test.want {
				t.Fatalf("eligible = %t, want %t", got, test.want)
			}
		})
	}
}

func TestNewStoryboardPlanBoundaries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		duration int64
		frames   int
		columns  int
		rows     int
	}{
		{2_000, 4, 4, 1},
		{4_999, 4, 4, 1},
		{5_000, 10, 5, 2},
		{10_000, 10, 5, 2},
		{math.MaxInt64, 10, 5, 2},
	}
	for _, test := range tests {
		plan, err := NewStoryboardPlan(test.duration)
		if err != nil {
			t.Fatalf("duration %d: %v", test.duration, err)
		}
		if len(plan.TimestampsMS) != test.frames ||
			plan.Columns != test.columns ||
			plan.Rows != test.rows {
			t.Fatalf("duration %d: plan = %#v", test.duration, plan)
		}
		if err := plan.Validate(); err != nil {
			t.Fatalf("duration %d validation: %v", test.duration, err)
		}
	}
}

func TestNewStoryboardPlanUsesUniformInteriorTimestamps(t *testing.T) {
	t.Parallel()

	plan, err := NewStoryboardPlan(10_000)
	if err != nil {
		t.Fatal(err)
	}
	want := []int64{909, 1818, 2727, 3636, 4545, 5454, 6363, 7272, 8181, 9090}
	if !slices.Equal(plan.TimestampsMS, want) {
		t.Fatalf("timestamps = %v, want %v", plan.TimestampsMS, want)
	}
}

func TestStoryboardPlanRejectsIneligibleOrInvalidValues(t *testing.T) {
	t.Parallel()

	if _, err := NewStoryboardPlan(1_999); !errors.Is(err, ErrStoryboardNotEligible) {
		t.Fatalf("short duration error = %v", err)
	}
	invalid := StoryboardPlan{
		DurationMS:   10_000,
		TimestampsMS: []int64{1, 2, 2, 4},
		Columns:      4,
		Rows:         1,
	}
	if !errors.Is(invalid.Validate(), ErrInvalidStoryboardPlan) {
		t.Fatal("duplicate timestamp unexpectedly accepted")
	}
}

func TestStoryboardLayoutValidation(t *testing.T) {
	t.Parallel()

	for _, layout := range []StoryboardLayout{
		{FrameCount: 4, Columns: 4, Rows: 1, CellWidth: 320, CellHeight: 180},
		{FrameCount: 10, Columns: 5, Rows: 2, CellWidth: 180, CellHeight: 320},
	} {
		if err := layout.Validate(); err != nil {
			t.Fatalf("valid layout %#v: %v", layout, err)
		}
	}
	for _, layout := range []StoryboardLayout{
		{FrameCount: 5, Columns: 5, Rows: 1, CellWidth: 320, CellHeight: 180},
		{FrameCount: 10, Columns: 4, Rows: 3, CellWidth: 320, CellHeight: 180},
		{FrameCount: 10, Columns: 5, Rows: 2, CellWidth: 321, CellHeight: 180},
		{FrameCount: 4, Columns: 4, Rows: 1, CellWidth: 320, CellHeight: 0},
	} {
		if !errors.Is(layout.Validate(), ErrInvalidStoryboardLayout) {
			t.Fatalf("invalid layout unexpectedly accepted: %#v", layout)
		}
	}
}

func TestStoryboardCellDimensionsPreserveOrientationWithinBounds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		width, height int
		wantWidth     int
		wantHeight    int
	}{
		{1920, 1080, 320, 180},
		{1080, 1920, 180, 320},
		{100, 50, 100, 50},
	}
	for _, test := range tests {
		width, height, err := StoryboardCellDimensions(test.width, test.height)
		if err != nil {
			t.Fatal(err)
		}
		if width != test.wantWidth || height != test.wantHeight {
			t.Fatalf(
				"%dx%d cell = %dx%d, want %dx%d",
				test.width,
				test.height,
				width,
				height,
				test.wantWidth,
				test.wantHeight,
			)
		}
	}
}
