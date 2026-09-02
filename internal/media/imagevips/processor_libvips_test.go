//go:build libvips

package imagevips

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"strings"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/face"
	"github.com/HappyQuQu/foliopath/internal/media"
	"github.com/HappyQuQu/foliopath/internal/semantic"
	"github.com/davidbyttow/govips/v2/vips"
)

func TestPrepareSemanticImageDropsAlphaBeforeResizeAndUsesCHW(t *testing.T) {
	t.Cleanup(func() { vips.AssertNoLeaks(t) })
	source := image.NewNRGBA(image.Rect(0, 0, semantic.SigLIPImageWidth, semantic.SigLIPImageHeight))
	for offset := 0; offset < len(source.Pix); offset += 4 {
		source.Pix[offset] = 255
		source.Pix[offset+1] = 128
		source.Pix[offset+2] = 0
		source.Pix[offset+3] = 0
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, source); err != nil {
		t.Fatal(err)
	}
	tensor, err := New().PrepareSemanticImage(context.Background(), bytes.NewReader(encoded.Bytes()), media.FormatPNG)
	if err != nil {
		t.Fatal(err)
	}
	if len(tensor) != semantic.SigLIPImageValues || tensor[0] != 1 ||
		tensor[semantic.SigLIPImagePixels] <= 0 || tensor[2*semantic.SigLIPImagePixels] != -1 {
		t.Fatalf("unexpected tensor channels: len=%d rgb=%v", len(tensor), []float32{
			tensor[0], tensor[semantic.SigLIPImagePixels], tensor[2*semantic.SigLIPImagePixels],
		})
	}
}

func TestPrepareSemanticImageResizesBothAxesAndFailsClosed(t *testing.T) {
	t.Cleanup(func() { vips.AssertNoLeaks(t) })
	var encoded bytes.Buffer
	if err := jpeg.Encode(&encoded, syntheticImage(640, 240), nil); err != nil {
		t.Fatal(err)
	}
	tensor, err := New().PrepareSemanticImage(context.Background(), bytes.NewReader(encoded.Bytes()), media.FormatJPEG)
	if err != nil || len(tensor) != semantic.SigLIPImageValues {
		t.Fatalf("tensor len=%d error=%v", len(tensor), err)
	}
	if _, err := New().PrepareSemanticImage(context.Background(), bytes.NewReader(encoded.Bytes()), media.FormatPNG); !errors.Is(err, semantic.ErrInvalidImageInput) {
		t.Fatalf("format mismatch error = %v", err)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := New().PrepareSemanticImage(cancelled, bytes.NewReader(encoded.Bytes()), media.FormatJPEG); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled error = %v", err)
	}
}

func TestDecodeFaceImageProducesBoundedInterleavedRGB(t *testing.T) {
	t.Cleanup(func() { vips.AssertNoLeaks(t) })
	source := image.NewNRGBA(image.Rect(0, 0, 320, 160))
	for offset := 0; offset < len(source.Pix); offset += 4 {
		source.Pix[offset], source.Pix[offset+1], source.Pix[offset+2], source.Pix[offset+3] = 255, 64, 0, 0
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, source); err != nil {
		t.Fatal(err)
	}
	decoded, err := New().DecodeFaceImage(context.Background(), bytes.NewReader(encoded.Bytes()), media.FormatPNG,
		face.MaxInputBytes, 160)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Width != 160 || decoded.Height != 80 || len(decoded.RGB) != 160*80*3 ||
		decoded.RGB[0] < 250 || decoded.RGB[1] < 60 || decoded.RGB[2] != 0 {
		t.Fatalf("decoded=%dx%d len=%d rgb=%v", decoded.Width, decoded.Height, len(decoded.RGB), decoded.RGB[:3])
	}
	if _, err := New().DecodeFaceImage(context.Background(), bytes.NewReader(encoded.Bytes()), media.FormatJPEG,
		face.MaxInputBytes, 160); !errors.Is(err, face.ErrInvalidInput) {
		t.Fatalf("format mismatch error=%v", err)
	}
}

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
	t.Cleanup(func() { vips.AssertNoLeaks(t) })
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
	t.Cleanup(func() { vips.AssertNoLeaks(t) })
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
	); !errors.Is(err, media.ErrSourceTooLarge) {
		t.Fatalf("pixel bomb error = %v", err)
	}
}

func TestProcessorUsesShrinkOnLoadAndOrientationForLargeJPEG(t *testing.T) {
	t.Cleanup(func() { vips.AssertNoLeaks(t) })
	const width, height = 12_000, 9_000
	source, err := vips.Black(width, height)
	if err != nil {
		t.Fatal(err)
	}
	if err := source.SetOrientation(6); err != nil {
		source.Close()
		t.Fatal(err)
	}
	encoded, _, err := source.ExportJpeg(vips.NewJpegExportParams())
	source.Close()
	if err != nil {
		t.Fatal(err)
	}

	result, err := New().Process(
		context.Background(),
		bytes.NewReader(encoded),
		media.FormatJPEG,
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.Metadata.Width != height || result.Metadata.Height != width ||
		result.Thumbnail.Width != 384 ||
		result.Thumbnail.Height != media.GridThumbnailHeight ||
		len(result.Thumbnail.Bytes) < 12 ||
		string(result.Thumbnail.Bytes[:4]) != "RIFF" ||
		string(result.Thumbnail.Bytes[8:12]) != "WEBP" {
		t.Fatalf("large JPEG result = %#v", result)
	}
}

func TestProcessorRejectsExtensionFormatMismatch(t *testing.T) {
	t.Cleanup(func() { vips.AssertNoLeaks(t) })
	var encoded bytes.Buffer
	if err := gif.Encode(&encoded, syntheticImage(96, 64), nil); err != nil {
		t.Fatal(err)
	}
	_, err := New().Process(
		context.Background(),
		bytes.NewReader(encoded.Bytes()),
		media.FormatJPEG,
	)
	diagnostic, ok := media.DiagnoseFailure(err)
	if !errors.Is(err, media.ErrInvalidMedia) || !ok ||
		diagnostic.Stage != media.StageProbe ||
		diagnostic.Reason != media.ReasonInvalidData ||
		diagnostic.Tool != "libvips" {
		t.Fatalf("format mismatch = %v, %#v, %v", err, diagnostic, ok)
	}
}

func TestProcessorRecoversOnlyBoundedTruncatedJPEG(t *testing.T) {
	t.Cleanup(func() { vips.AssertNoLeaks(t) })
	var encoded bytes.Buffer
	if err := jpeg.Encode(&encoded, syntheticImage(512, 384), nil); err != nil {
		t.Fatal(err)
	}
	value := encoded.Bytes()
	if len(value) < 4 || !bytes.Equal(value[len(value)-2:], []byte{0xff, 0xd9}) {
		t.Fatal("synthetic JPEG has no EOI marker")
	}
	strict, err := New().Process(
		context.Background(),
		bytes.NewReader(value),
		media.FormatJPEG,
	)
	if err != nil || strict.Warning != nil {
		t.Fatalf("strict JPEG result = %#v, %v", strict, err)
	}

	result, err := New().Process(
		context.Background(),
		bytes.NewReader(value[:len(value)-2]),
		media.FormatJPEG,
	)
	if err != nil {
		t.Fatalf("recover truncated JPEG: %v", err)
	}
	if result.Metadata.Width != 512 || result.Metadata.Height != 384 ||
		result.Thumbnail.Width != media.GridThumbnailWidth ||
		result.Thumbnail.Height != 384 || result.Warning == nil ||
		result.Warning.Stage != media.StageValidation ||
		result.Warning.Reason != media.ReasonDecodeRecovered ||
		result.Warning.Tool != "libvips" || result.Warning.ExitCode != nil ||
		len(result.Thumbnail.Bytes) < 12 ||
		string(result.Thumbnail.Bytes[:4]) != "RIFF" ||
		string(result.Thumbnail.Bytes[8:12]) != "WEBP" {
		t.Fatalf("recovered JPEG result = %#v", result)
	}

	_, err = New().Process(
		context.Background(),
		bytes.NewReader(value[2:]),
		media.FormatJPEG,
	)
	if !errors.Is(err, media.ErrInvalidMedia) {
		t.Fatalf("JPEG without SOI error = %v", err)
	}
}

func TestJPEGRecoveryPredicateIsNarrowAndPixelBounded(t *testing.T) {
	t.Cleanup(func() { vips.AssertNoLeaks(t) })
	for _, test := range []struct {
		name   string
		format media.Format
		width  int
		height int
		err    error
		want   bool
	}{
		{
			name: "premature end", format: media.FormatJPEG,
			width: 100, height: 100, err: errors.New("Premature end of JPEG file"),
			want: true,
		},
		{
			name: "incomplete scan", format: media.FormatJPEG,
			width: 100, height: 100, err: errors.New("Incomplete scan detected"),
			want: true,
		},
		{
			name: "corrupt jpeg data", format: media.FormatJPEG,
			width: 100, height: 100, err: errors.New("Corrupt JPEG data"),
			want: true,
		},
		{
			name: "wrong format", format: media.FormatPNG,
			width: 100, height: 100, err: errors.New("premature end"),
		},
		{
			name: "over pixel bound", format: media.FormatJPEG,
			width: 10_001, height: 10_000, err: errors.New("premature end"),
		},
		{
			name: "generic corruption", format: media.FormatJPEG,
			width: 100, height: 100, err: errors.New("corrupt image data"),
		},
		{
			name: "generic truncation", format: media.FormatJPEG,
			width: 100, height: 100, err: errors.New("truncated input"),
		},
		{
			name: "unexpected end", format: media.FormatJPEG,
			width: 100, height: 100, err: errors.New("unexpected end"),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := jpegRecoveryAllowed(
				test.format, test.width, test.height, test.err,
			); got != test.want {
				t.Fatalf("jpegRecoveryAllowed = %t, want %t", got, test.want)
			}
		})
	}
}

func TestImageOutputErrorClassifiesKnownTruncationWithoutLeakingIt(t *testing.T) {
	t.Cleanup(func() { vips.AssertNoLeaks(t) })
	err := imageOutputError(errors.New("private/source.jpg: Incomplete scan detected\nStack: secret"))
	diagnostic, ok := media.DiagnoseFailure(err)
	if !errors.Is(err, media.ErrInvalidMedia) || !ok ||
		diagnostic.Stage != media.StageValidation ||
		diagnostic.Reason != media.ReasonDecodeFailed ||
		diagnostic.Tool != "libvips" ||
		strings.Contains(err.Error(), "private/source.jpg") {
		t.Fatalf("truncated image output = %v, %#v, %v", err, diagnostic, ok)
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
