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
	"math"
	"os"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/media"
	"github.com/HappyQuQu/foliopath/internal/media/imagevips"
	"github.com/davidbyttow/govips/v2/vips"
)

// Explicit isolated probe: generated pixels only, no file or database access.
func TestEncodedImageServiceProbe(t *testing.T) {
	origin := os.Getenv("FOLIOPATH_TEST_ML_ORIGIN")
	if origin == "" {
		t.Fatal("FOLIOPATH_TEST_ML_ORIGIN required")
	}
	runtime := imagevips.NewRuntime()
	if err := runtime.Start(); err != nil {
		t.Fatal(err)
	}
	defer runtime.Shutdown()
	client, err := New(origin)
	if raw := os.Getenv("FOLIOPATH_TEST_ML_IDENTITY"); raw != "" {
		if client != nil {
			client.Close()
		}
		var expected ServiceIdentity
		if err := json.Unmarshal([]byte(raw), &expected); err != nil {
			t.Fatal(err)
		}
		client, err = NewVerified(context.Background(), origin, expected)
	}
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	img := image.NewRGBA(image.Rect(0, 0, 1200, 600))
	for y := 0; y < 600; y++ {
		for x := 0; x < 1200; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 255, A: 255})
		}
	}
	var pngBytes, jpegBytes, gifBytes bytes.Buffer
	if err := png.Encode(&pngBytes, img); err != nil {
		t.Fatal(err)
	}
	if err := jpeg.Encode(&jpegBytes, img, nil); err != nil {
		t.Fatal(err)
	}
	if err := gif.Encode(&gifBytes, img, nil); err != nil {
		t.Fatal(err)
	}
	webpImage, err := vips.LoadImageFromBuffer(pngBytes.Bytes(), nil)
	if err != nil {
		t.Fatal(err)
	}
	webpBytes, _, err := webpImage.ExportWebp(vips.NewWebpExportParams())
	webpImage.Close()
	if err != nil {
		t.Fatal(err)
	}
	for _, fixture := range []struct {
		format media.Format
		data   []byte
	}{
		{media.FormatPNG, pngBytes.Bytes()}, {media.FormatJPEG, jpegBytes.Bytes()},
		{media.FormatGIF, gifBytes.Bytes()}, {media.FormatWebP, webpBytes},
	} {
		t.Run(string(fixture.format), func(t *testing.T) {
			prepared, err := imagevips.New().PrepareRemoteSemanticImage(ctx, bytes.NewReader(fixture.data), fixture.format)
			if err != nil {
				t.Fatal(err)
			}
			config, err := png.DecodeConfig(bytes.NewReader(prepared))
			if err != nil || config.Width != 1024 || config.Height != 512 {
				t.Fatalf("unexpected prepared dimensions: %+v %v", config, err)
			}
			vector, err := client.EncodeImage(ctx, prepared)
			if err != nil {
				t.Fatal(err)
			}
			assertProbeVector(t, vector)
		})
	}
	query, err := client.EncodeText(ctx, "a red rectangle")
	if err != nil {
		t.Fatal(err)
	}
	assertProbeVector(t, query)
	vips.AssertNoLeaks(t)
	t.Log("four generated source formats -> bounded PNG -> real offline ML -> normalized 512-vector; no indexing or quality claim")
}

func assertProbeVector(t *testing.T, vector []float32) {
	t.Helper()
	if len(vector) != Dimension {
		t.Fatal("unexpected dimension")
	}
	var norm float64
	for _, value := range vector {
		norm += float64(value) * float64(value)
	}
	if math.IsNaN(norm) || math.IsInf(norm, 0) || math.Abs(norm-1) > 1e-5 {
		t.Fatal("invalid normalized vector")
	}
}
