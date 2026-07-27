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
