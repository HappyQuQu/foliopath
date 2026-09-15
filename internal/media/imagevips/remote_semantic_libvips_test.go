//go:build libvips

package imagevips

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/png"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/media"
	"github.com/HappyQuQu/foliopath/internal/semantic"
	"github.com/davidbyttow/govips/v2/vips"
)

func TestRemoteSemanticImage(t *testing.T) {
	t.Cleanup(func() { vips.AssertNoLeaks(t) })
	var source bytes.Buffer
	if err := png.Encode(&source, syntheticImage(2048, 1024)); err != nil {
		t.Fatal(err)
	}
	output, err := New().PrepareRemoteSemanticImage(context.Background(), bytes.NewReader(source.Bytes()), media.FormatPNG)
	if err != nil {
		t.Fatal(err)
	}
	config, err := png.DecodeConfig(bytes.NewReader(output))
	if err != nil || config.Width != 1024 || config.Height != 512 {
		t.Fatalf("invalid output: %+v %v", config, err)
	}
	for _, format := range []media.Format{media.FormatJPEG, media.FormatMP4} {
		if result, err := New().PrepareRemoteSemanticImage(context.Background(), bytes.NewReader(source.Bytes()), format); result != nil || !errors.Is(err, semantic.ErrInvalidImageInput) {
			t.Fatalf("accepted mismatch: %v", err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := New().PrepareRemoteSemanticImage(ctx, bytes.NewReader(source.Bytes()), media.FormatPNG); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := New().PrepareRemoteSemanticImage(context.Background(), bytes.NewReader([]byte("corrupt")), media.FormatPNG); !errors.Is(err, semantic.ErrInvalidImageInput) {
		t.Fatal(err)
	}
}

func TestRemoteSemanticImageBudgets(t *testing.T) {
	t.Cleanup(func() { vips.AssertNoLeaks(t) })
	ctx := context.Background()
	if _, err := New().PrepareRemoteSemanticImage(ctx, bytes.NewReader(make([]byte, semantic.RemoteImageMaxSourceBytes+1)), media.FormatPNG); !errors.Is(err, semantic.ErrInvalidImageInput) {
		t.Fatalf("accepted source over byte budget: %v", err)
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, image.NewGray(image.Rect(0, 0, 4001, 4000))); err != nil {
		t.Fatal(err)
	}
	if _, err := New().PrepareRemoteSemanticImage(ctx, bytes.NewReader(encoded.Bytes()), media.FormatPNG); !errors.Is(err, semantic.ErrInvalidImageInput) {
		t.Fatalf("accepted source over pixel budget: %v", err)
	}
}
