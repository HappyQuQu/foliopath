//go:build libvips

package imagevips

import (
	"bytes"
	"context"
	"encoding/binary"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/face"
	"github.com/HappyQuQu/foliopath/internal/media"
	"github.com/davidbyttow/govips/v2/vips"
)

// Procedural fixture avoids allocating a full source pixel buffer in the test.
type largeFaceFixture struct{ width, height int }

func (p largeFaceFixture) ColorModel() color.Model { return color.YCbCrModel }
func (p largeFaceFixture) Bounds() image.Rectangle { return image.Rect(0, 0, p.width, p.height) }
func (p largeFaceFixture) At(x, y int) color.Color {
	return color.YCbCr{Y: uint8((x/32+y/32)%200 + 20), Cb: 110, Cr: 140}
}

func TestDecodeFaceLargePNGWorkingMemory(t *testing.T) {
	if testing.Short() {
		t.Skip("large synthetic decode; run under bounded Docker memory")
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, largeFaceFixture{9000, 9000}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		decoded, err := New().DecodeFaceImage(context.Background(), bytes.NewReader(encoded.Bytes()), media.FormatPNG, face.MaxInputBytes, face.MaxDecodeDimension)
		if err != nil {
			t.Fatal(err)
		}
		if decoded.Width != 1600 || decoded.Height != 1600 {
			t.Fatalf("invalid dimensions %dx%d", decoded.Width, decoded.Height)
		}
	}
}

func TestDecodeFaceLargeJPEGWorkingMemory(t *testing.T) {
	if testing.Short() {
		t.Skip("large synthetic decode; run under bounded Docker memory")
	}
	var encoded bytes.Buffer
	if err := jpeg.Encode(&encoded, largeFaceFixture{9000, 9000}, &jpeg.Options{Quality: 75}); err != nil {
		t.Fatal(err)
	}
	for _, orientation := range []byte{1, 6, 8} {
		// Minimal little-endian EXIF IFD with orientation. No real media fixture.
		exif := []byte{'E', 'x', 'i', 'f', 0, 0, 'I', 'I', 42, 0, 8, 0, 0, 0, 1, 0, 0x12, 1, 3, 0, 1, 0, 0, 0, orientation, 0, 0, 0, 0, 0, 0, 0}
		marker := []byte{0xff, 0xe1, 0, 0}
		binary.BigEndian.PutUint16(marker[2:], uint16(len(exif)+2))
		value := append([]byte{}, encoded.Bytes()[:2]...)
		value = append(value, marker...)
		value = append(value, exif...)
		value = append(value, encoded.Bytes()[2:]...)
		t.Logf("orientation=%d", orientation)
		decoded, err := New().DecodeFaceImage(context.Background(), bytes.NewReader(value), media.FormatJPEG, face.MaxInputBytes, face.MaxDecodeDimension)
		if err != nil {
			t.Fatal(err)
		}
		if decoded.Width != 1600 || decoded.Height != 1600 || len(decoded.RGB) != 1600*1600*3 {
			t.Fatalf("invalid output %dx%d", decoded.Width, decoded.Height)
		}
	}
}

func TestDecodeFaceProgressiveJPEGWorkingMemory(t *testing.T) {
	fixture := os.Getenv("FOLIOPATH_SYNTHETIC_PROGRESSIVE_JPEG")
	if fixture == "" {
		t.Skip("explicit generated progressive JPEG required")
	}
	if os.Getenv("FOLIOPATH_GENERATE_PROGRESSIVE") == "1" {
		var input bytes.Buffer
		if err := jpeg.Encode(&input, largeFaceFixture{9000, 9000}, &jpeg.Options{Quality: 75}); err != nil {
			t.Fatal(err)
		}
		im, err := vips.LoadImageFromBuffer(input.Bytes(), imageImportParams())
		if err != nil {
			t.Fatal(err)
		}
		defer im.Close()
		params := vips.NewJpegExportParams()
		params.Interlace = true
		data, _, err := im.ExportJpeg(params)
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(fixture, data, 0600); err != nil {
			t.Fatal(err)
		}
		return
	}
	encoded, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		decoded, err := New().DecodeFaceImage(context.Background(), bytes.NewReader(encoded), media.FormatJPEG, face.MaxInputBytes, face.MaxDecodeDimension)
		if err != nil {
			t.Fatal(err)
		}
		if decoded.Width != 1600 || decoded.Height != 1600 {
			t.Fatal("invalid dimensions")
		}
	}
}

func TestDecodeFaceVeryLargeJPEGKeepsShrinkContract(t *testing.T) {
	if testing.Short() {
		t.Skip("large synthetic JPEG")
	}
	var encoded bytes.Buffer
	if err := jpeg.Encode(&encoded, largeFaceFixture{12000, 9000}, &jpeg.Options{Quality: 75}); err != nil {
		t.Fatal(err)
	}
	decoded, err := New().DecodeFaceImage(context.Background(), bytes.NewReader(encoded.Bytes()), media.FormatJPEG, face.MaxInputBytes, face.MaxDecodeDimension)
	if err != nil {
		t.Fatal(err)
	}
	// Existing >100 MP contract uses shrink 8, without upscaling afterwards.
	if decoded.Width != 1500 || decoded.Height != 1125 {
		t.Fatalf("changed shrink output %dx%d", decoded.Width, decoded.Height)
	}
}

func TestDecodeFaceJPEGFileStreamsBeyondLegacyBufferThreshold(t *testing.T) {
	var encoded bytes.Buffer
	if err := jpeg.Encode(&encoded, largeFaceFixture{320, 240}, &jpeg.Options{Quality: 75}); err != nil {
		t.Fatal(err)
	}
	file, err := os.CreateTemp(t.TempDir(), "face-stream-*.jpg")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if _, err = file.Write(encoded.Bytes()); err != nil {
		t.Fatal(err)
	}
	// Valid JPEG data may have trailing bytes. Make the file larger than the
	// former 64 MiB operational threshold without allocating that payload.
	if err = file.Truncate(65 * 1024 * 1024); err != nil {
		t.Fatal(err)
	}
	decoded, err := New().DecodeFaceImage(context.Background(), file, media.FormatJPEG, face.MaxInputBytes, face.MaxDecodeDimension)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Width != 320 || decoded.Height != 240 || len(decoded.RGB) != 320*240*3 {
		t.Fatalf("invalid output %dx%d bytes=%d", decoded.Width, decoded.Height, len(decoded.RGB))
	}
}
