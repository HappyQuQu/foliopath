package media

import (
	"context"
	"errors"
	"testing"
)

func TestValidateProcessingResultSeparatesImageAndVideoSemantics(t *testing.T) {
	image := ProcessingResult{
		Metadata: Metadata{
			Width: 96, Height: 64, PlaybackStatus: PlaybackNotApplicable,
		},
		Thumbnail: Thumbnail{Bytes: []byte("webp"), Width: 48, Height: 32},
	}
	if err := ValidateProcessingResult(KindImage, image); err != nil {
		t.Fatal(err)
	}
	duration := int64(1000)
	video := ProcessingResult{
		Metadata: Metadata{
			Width: 96, Height: 64, DurationMS: &duration,
			PlaybackStatus: PlaybackUnknown,
		},
		Thumbnail: Thumbnail{Bytes: []byte("webp"), Width: 48, Height: 32},
	}
	if err := ValidateProcessingResult(KindVideo, video); err != nil {
		t.Fatal(err)
	}

	image.Metadata.DurationMS = &duration
	if !errors.Is(ValidateProcessingResult(KindImage, image), ErrInvalidResult) {
		t.Fatal("image duration unexpectedly accepted")
	}
	video.Thumbnail.Width = GridThumbnailWidth + 1
	if !errors.Is(ValidateProcessingResult(KindVideo, video), ErrInvalidResult) {
		t.Fatal("oversized thumbnail unexpectedly accepted")
	}
}

func TestProcessingCodeDoesNotExposeAdapterErrors(t *testing.T) {
	tests := []struct {
		err  error
		want ProcessingErrorCode
	}{
		{ErrUnsupportedMedia, ErrorUnsupportedMedia},
		{ErrInvalidMedia, ErrorInvalidMedia},
		{context.DeadlineExceeded, ErrorProcessingTimed},
		{errors.New("/library/private failed: raw stderr"), ErrorProcessingFailed},
	}
	for _, test := range tests {
		if got := ProcessingCode(test.err); got != test.want {
			t.Fatalf("ProcessingCode(%v) = %q, want %q", test.err, got, test.want)
		}
	}
}

func TestMediaResourcePolicyRejectsOversizedSourcesAndDimensions(t *testing.T) {
	for _, test := range []struct {
		name   string
		format Format
		size   int64
	}{
		{name: "empty image", format: FormatJPEG, size: 0},
		{name: "oversized image", format: FormatPNG, size: MaxImageSourceBytes + 1},
		{name: "oversized video", format: FormatMP4, size: MaxVideoSourceBytes + 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			if !errors.Is(ValidateSourceSize(test.format, test.size), ErrInvalidMedia) {
				t.Fatal("unsafe source size unexpectedly accepted")
			}
		})
	}
	if err := ValidateSourceSize(FormatJPEG, MaxImageSourceBytes); err != nil {
		t.Fatalf("maximum image size rejected: %v", err)
	}
	if err := ValidateSourceSize(FormatMKV, MaxVideoSourceBytes); err != nil {
		t.Fatalf("maximum video size rejected: %v", err)
	}
	if err := ValidateSourceSize(FormatAVI, MaxVideoSourceBytes); err != nil {
		t.Fatalf("maximum AVI size rejected: %v", err)
	}
	for _, dimensions := range [][2]int{
		{0, 1},
		{MaxMediaDimension + 1, 1},
		{10_001, 10_000},
	} {
		if !errors.Is(
			ValidateDimensions(dimensions[0], dimensions[1]),
			ErrInvalidMedia,
		) {
			t.Fatalf("unsafe dimensions %v unexpectedly accepted", dimensions)
		}
	}
}

func TestStoryboardRequestAndResultValidation(t *testing.T) {
	request := StoryboardRequest{
		TimestampsMS: []int64{100, 200, 300, 400},
		Columns:      4,
		Rows:         1,
		CellWidth:    320,
		CellHeight:   180,
	}
	if err := ValidateStoryboardRequest(request); err != nil {
		t.Fatalf("valid request: %v", err)
	}
	result := StoryboardResult{
		Bytes:      []byte("RIFF\x04\x00\x00\x00WEBP"),
		FrameCount: 4,
		Columns:    4,
		Rows:       1,
		CellWidth:  320,
		CellHeight: 180,
	}
	if err := ValidateStoryboardResult(request, result); err != nil {
		t.Fatalf("valid result: %v", err)
	}

	invalidRequest := request
	invalidRequest.TimestampsMS = []int64{100, 200, 200, 400}
	if !errors.Is(ValidateStoryboardRequest(invalidRequest), ErrInvalidResult) {
		t.Fatal("duplicate storyboard timestamp unexpectedly accepted")
	}
	invalidResult := result
	invalidResult.Bytes = []byte("not a webp")
	if !errors.Is(
		ValidateStoryboardResult(request, invalidResult),
		ErrInvalidResult,
	) {
		t.Fatal("non-WebP storyboard unexpectedly accepted")
	}
}
