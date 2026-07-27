//go:build libvips

package imagevips

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/gif"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/media"
	"github.com/davidbyttow/govips/v2/vips"
)

func TestProcessorGeneratesBoundedWebPAndRejectsCorruptInput(t *testing.T) {
	vips.Startup(&vips.Config{
		ConcurrencyLevel: 1,
		MaxCacheFiles:    0,
		MaxCacheMem:      32 << 20,
		MaxCacheSize:     32,
	})
	t.Cleanup(vips.Shutdown)

	var encoded bytes.Buffer
	if err := gif.Encode(&encoded, syntheticImage(96, 64), nil); err != nil {
		t.Fatal(err)
	}
	result, err := New().Process(
		context.Background(),
		bytes.NewReader(encoded.Bytes()),
		media.FormatGIF,
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.Metadata.Width != 96 || result.Metadata.Height != 64 ||
		result.Thumbnail.Width != 96 || result.Thumbnail.Height != 64 ||
		len(result.Thumbnail.Bytes) == 0 {
		t.Fatalf("result = %#v", result)
	}
	if _, err := New().Process(
		context.Background(),
		bytes.NewReader([]byte("\x89PNG\r\n\x1a\ntruncated")),
		media.FormatPNG,
	); err == nil {
		t.Fatal("truncated PNG unexpectedly succeeded")
	}
}

func syntheticImage(width, height int) image.Image {
	value := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			value.SetNRGBA(x, y, color.NRGBA{
				R: uint8(x), G: uint8(y), B: uint8(x + y), A: 255,
			})
		}
	}
	return value
}
