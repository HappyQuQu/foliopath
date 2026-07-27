//go:build libvips

package imagevips

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"os"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/media"
)

func TestMain(main *testing.M) {
	runtime := NewRuntime()
	if err := runtime.Start(); err != nil {
		panic(err)
	}
	code := main.Run()
	runtime.Shutdown()
	os.Exit(code)
}

func TestProcessorGeneratesBoundedWebPAndRejectsCorruptInput(t *testing.T) {
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

func TestProcessorRejectsPixelBombBeforeThumbnailEvaluation(t *testing.T) {
	var encoded bytes.Buffer
	if err := jpeg.Encode(&encoded, syntheticImage(1, 1), nil); err != nil {
		t.Fatal(err)
	}
	value := encoded.Bytes()
	sof := bytes.Index(value, []byte{0xff, 0xc0})
	if sof < 0 || sof+9 > len(value) {
		t.Fatal("synthetic JPEG has no baseline SOF marker")
	}
	const hostileDimension = 20_000
	value[sof+5] = byte(hostileDimension >> 8)
	value[sof+6] = byte(hostileDimension & 0xff)
	value[sof+7] = byte(hostileDimension >> 8)
	value[sof+8] = byte(hostileDimension & 0xff)

	if _, err := New().Process(
		context.Background(),
		bytes.NewReader(value),
		media.FormatJPEG,
	); err != media.ErrInvalidMedia {
		t.Fatalf("pixel bomb error = %v", err)
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
