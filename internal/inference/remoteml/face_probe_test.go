//go:build libvips && remotemlprobe

package remoteml

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/face"
	"github.com/HappyQuQu/foliopath/internal/media"
	"github.com/HappyQuQu/foliopath/internal/media/imagevips"
	"github.com/davidbyttow/govips/v2/vips"
)

// Explicit local Docker probe: synthetic pixels only, deployment identity supplied
// independently of the HTTP response. No original media or database access.
func TestFaceServiceProbe(t *testing.T) {
	var identity ServiceIdentity
	if json.Unmarshal([]byte(os.Getenv("FOLIOPATH_TEST_FACE_IDENTITY")), &identity) != nil {
		t.Fatal("explicit face identity required")
	}
	c, err := New(os.Getenv("FOLIOPATH_TEST_ML_ORIGIN"))
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	f, err := c.BindFace(identity)
	if err != nil {
		t.Fatal(err)
	}
	input := image.NewRGBA(image.Rect(0, 0, 320, 240))
	for y := 0; y < 240; y++ {
		for x := 0; x < 320; x++ {
			input.SetRGBA(x, y, color.RGBA{25, 70, 130, 255})
		}
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, input); err != nil {
		t.Fatal(err)
	}
	faces, err := f.Detect(context.Background(), encoded.Bytes())
	if err != nil || len(faces) != 0 {
		t.Fatalf("faces=%d err=%v", len(faces), err)
	}
	decoderRuntime := imagevips.NewRuntime()
	if err := decoderRuntime.Start(); err != nil {
		t.Fatal(err)
	}
	defer decoderRuntime.Shutdown()
	runtime, err := NewFaceRuntime(imagevips.New(), f)
	if err != nil {
		t.Fatal(err)
	}
	var jpg, gifImage bytes.Buffer
	if err := jpeg.Encode(&jpg, input, nil); err != nil {
		t.Fatal(err)
	}
	if err := gif.Encode(&gifImage, input, nil); err != nil {
		t.Fatal(err)
	}
	webpImage, err := vips.LoadImageFromBuffer(encoded.Bytes(), nil)
	if err != nil {
		t.Fatal(err)
	}
	webp, _, err := webpImage.ExportWebp(vips.NewWebpExportParams())
	webpImage.Close()
	if err != nil {
		t.Fatal(err)
	}
	for _, fixture := range []struct {
		format media.Format
		data   []byte
	}{
		{media.FormatPNG, encoded.Bytes()}, {media.FormatJPEG, jpg.Bytes()},
		{media.FormatGIF, gifImage.Bytes()}, {media.FormatWebP, webp},
	} {
		t.Run(string(fixture.format), func(t *testing.T) {
			faces, err := runtime.AnalyzeFaces(context.Background(), bytes.NewReader(fixture.data), fixture.format, face.MaxInputBytes)
			if err != nil || len(faces) != 0 {
				t.Fatalf("faces=%d err=%v", len(faces), err)
			}
		})
	}
}
