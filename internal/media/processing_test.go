package media

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestValidateProcessingResultSeparatesImageAndVideoSemantics(t *testing.T) {
	image := ProcessingResult{
		Metadata: Metadata{
			Width: 96, Height: 64, PlaybackStatus: PlaybackNotApplicable,
		},
		Thumbnail: Thumbnail{Bytes: []byte("webp"), Width: 48, Height: 32},
	}
	if err := ValidateProcessingResult(KindImage, FormatJPEG, image); err != nil {
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
	if err := ValidateProcessingResult(KindVideo, FormatMP4, video); err != nil {
		t.Fatal(err)
	}
	video.Metadata.DurationMS = nil
	if err := ValidateProcessingResult(KindVideo, FormatMP4, video); err != nil {
		t.Fatalf("video with unknown duration rejected: %v", err)
	}

	image.Metadata.DurationMS = &duration
	if !errors.Is(ValidateProcessingResult(KindImage, FormatJPEG, image), ErrInvalidResult) {
		t.Fatal("image duration unexpectedly accepted")
	}
	video.Thumbnail.Width = GridThumbnailWidth + 1
	if !errors.Is(ValidateProcessingResult(KindVideo, FormatMP4, video), ErrInvalidResult) {
		t.Fatal("oversized thumbnail unexpectedly accepted")
	}
}

func TestValidateProcessingResultRestrictsRecoveredJPEGWarning(t *testing.T) {
	warning := FailureDiagnostic{
		Stage: StageValidation, Reason: ReasonDecodeRecovered, Tool: "libvips",
	}
	result := ProcessingResult{
		Metadata: Metadata{
			Width: 96, Height: 64, PlaybackStatus: PlaybackNotApplicable,
		},
		Thumbnail: Thumbnail{Bytes: []byte("webp"), Width: 48, Height: 32},
		Warning:   &warning,
	}
	if err := ValidateProcessingResult(KindImage, FormatJPEG, result); err != nil {
		t.Fatalf("bounded recovered JPEG warning rejected: %v", err)
	}
	for _, test := range []struct {
		name   string
		kind   Kind
		format Format
		mutate func(*ProcessingResult)
	}{
		{name: "png", kind: KindImage, format: FormatPNG},
		{name: "video", kind: KindVideo, format: FormatMP4},
		{
			name: "over recovery pixel limit", kind: KindImage, format: FormatJPEG,
			mutate: func(value *ProcessingResult) {
				value.Metadata.Width, value.Metadata.Height = 12_000, 9_000
			},
		},
		{
			name: "wrong reason", kind: KindImage, format: FormatJPEG,
			mutate: func(value *ProcessingResult) {
				value.Warning.Reason = ReasonDecodeFailed
			},
		},
		{
			name: "wrong tool", kind: KindImage, format: FormatJPEG,
			mutate: func(value *ProcessingResult) { value.Warning.Tool = "ffmpeg" },
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			candidate := result
			candidateWarning := *result.Warning
			candidate.Warning = &candidateWarning
			if test.kind == KindVideo {
				duration := int64(1_000)
				candidate.Metadata.DurationMS = &duration
				candidate.Metadata.PlaybackStatus = PlaybackPlayable
			}
			if test.mutate != nil {
				test.mutate(&candidate)
			}
			if !errors.Is(
				ValidateProcessingResult(test.kind, test.format, candidate),
				ErrInvalidResult,
			) {
				t.Fatal("invalid recovery warning unexpectedly accepted")
			}
		})
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
		want   error
	}{
		{name: "empty image", format: FormatJPEG, size: 0, want: ErrInvalidMedia},
		{name: "oversized image", format: FormatPNG, size: MaxImageSourceBytes + 1, want: ErrSourceTooLarge},
		{name: "oversized video", format: FormatMP4, size: MaxVideoSourceBytes + 1, want: ErrSourceTooLarge},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateSourceSize(test.format, test.size)
			if !errors.Is(err, test.want) {
				t.Fatal("unsafe source size unexpectedly accepted")
			}
			if errors.Is(err, ErrSourceTooLarge) {
				diagnostic, ok := DiagnoseFailure(err)
				if !ok || diagnostic.Reason != ReasonSourceTooLarge ||
					diagnostic.Stage != StageSourceRead {
					t.Fatalf("source size diagnostic = %#v, %v", diagnostic, ok)
				}
			}
			if test.size == 0 {
				diagnostic, ok := DiagnoseFailure(err)
				if !ok || diagnostic.Reason != ReasonInvalidData ||
					diagnostic.Stage != StageSourceRead ||
					diagnostic.Tool != "filesystem" {
					t.Fatalf("empty source diagnostic = %#v, %v", diagnostic, ok)
				}
			}
		})
	}
	if err := ValidateSourceSize(FormatMP4, int64(4)<<30+1); err != nil {
		t.Fatalf("video above the former 4 GiB limit rejected: %v", err)
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
	if err := ValidateImageDimensions(FormatJPEG, 11_520, 15_360); err != nil {
		t.Fatalf("shrink-on-load JPEG rejected: %v", err)
	}
	largeJPEG := ProcessingResult{
		Metadata: Metadata{
			Width: 11_520, Height: 15_360, PlaybackStatus: PlaybackNotApplicable,
		},
		Thumbnail: Thumbnail{Bytes: []byte("webp"), Width: 384, Height: 512},
	}
	if err := ValidateProcessingResult(KindImage, FormatJPEG, largeJPEG); err != nil {
		t.Fatalf("large JPEG result rejected: %v", err)
	}
	if !errors.Is(
		ValidateProcessingResult(KindImage, FormatPNG, largeJPEG),
		ErrInvalidResult,
	) {
		t.Fatal("large PNG result unexpectedly accepted")
	}
	if err := ValidateImageDimensions(FormatPNG, 11_520, 15_360); !errors.Is(err, ErrSourceTooLarge) {
		t.Fatalf("oversized full-decode PNG error = %v", err)
	}
	if err := ValidateImageDimensions(FormatJPEG, 20_000, 20_000); !errors.Is(err, ErrSourceTooLarge) {
		t.Fatalf("hostile JPEG dimensions error = %v", err)
	} else {
		diagnostic, ok := DiagnoseFailure(err)
		if !ok || diagnostic.Stage != StageProbe ||
			diagnostic.Reason != ReasonSourceTooLarge || diagnostic.Tool != "libvips" {
			t.Fatalf("JPEG pixel-limit diagnostic = %#v, %v", diagnostic, ok)
		}
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
	invalidRequest = request
	invalidRequest.Timeout = MaxStoryboardAttemptTimeout + time.Second
	if !errors.Is(ValidateStoryboardRequest(invalidRequest), ErrInvalidResult) {
		t.Fatal("oversized storyboard timeout unexpectedly accepted")
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

func TestStoryboardProcessingTimeoutScalesWithDecodeWork(t *testing.T) {
	tests := []struct {
		name       string
		width      int
		height     int
		frameCount int
		want       time.Duration
	}{
		{name: "1080p ten frames", width: 1920, height: 1080, frameCount: 10, want: 45 * time.Second},
		{name: "4k ten frames", width: 3840, height: 2160, frameCount: 10, want: 3 * time.Minute},
		{name: "4k four frames", width: 3840, height: 2160, frameCount: 4, want: 90 * time.Second},
		{name: "large input capped", width: 10_000, height: 10_000, frameCount: 10, want: 3 * time.Minute},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := StoryboardProcessingTimeout(
				test.width,
				test.height,
				test.frameCount,
			)
			if err != nil || got != test.want {
				t.Fatalf("timeout = %s, %v; want %s", got, err, test.want)
			}
		})
	}
	if _, err := StoryboardProcessingTimeout(0, 1080, 10); !errors.Is(err, ErrInvalidResult) {
		t.Fatalf("invalid dimensions error = %v", err)
	}
}

func TestStoryboardProcessingTimeoutExtendsLargeLongFramePlan(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		width       int
		height      int
		frames      int
		sourceBytes int64
		want        time.Duration
	}{
		{"ordinary 4k", 3840, 2160, 10, 2 << 30, 3 * time.Minute},
		{"large 4k", 3840, 2160, 10, StoryboardLargeSourceBytes, 4 * time.Minute},
		{"large 1080p", 1920, 1080, 10, StoryboardLargeSourceBytes, 4 * time.Minute},
		{"large fallback", 3840, 2160, 4, StoryboardLargeSourceBytes, 90 * time.Second},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := StoryboardProcessingTimeoutForSource(
				test.width,
				test.height,
				test.frames,
				test.sourceBytes,
			)
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("timeout = %s, want %s", got, test.want)
			}
		})
	}
	if _, err := StoryboardProcessingTimeoutForSource(1920, 1080, 10, 0); !errors.Is(err, ErrInvalidResult) {
		t.Fatalf("zero source error = %v", err)
	}
}
