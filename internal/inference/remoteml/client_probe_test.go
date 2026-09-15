//go:build remotemlprobe

package remoteml

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"testing"
)

// Explicit opt-in only; this probe never opens media files.
func TestOfflineServiceProbe(t *testing.T) {
	origin := os.Getenv("FOLIOPATH_TEST_ML_ORIGIN")
	if origin == "" {
		t.Fatal("FOLIOPATH_TEST_ML_ORIGIN is required for explicit probe")
	}
	c, err := New(origin)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	img := image.NewRGBA(image.Rect(0, 0, 224, 224))
	for y := range 224 {
		for x := range 224 {
			img.SetRGBA(x, y, color.RGBA{R: 255, A: 255})
		}
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, img); err != nil {
		t.Fatal(err)
	}
	visual, err := c.EncodeImage(context.Background(), encoded.Bytes())
	if err != nil {
		t.Fatalf("image: %v", err)
	}
	text, err := c.EncodeText(context.Background(), "a red square")
	if err != nil {
		t.Fatalf("text: %v", err)
	}
	repeated, err := c.EncodeText(context.Background(), "a red square")
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range [][]float32{visual, text, repeated} {
		if len(v) != Dimension {
			t.Fatal("wrong dimension")
		}
		var norm float64
		for _, x := range v {
			norm += float64(x) * float64(x)
		}
		if math.IsNaN(norm) || math.Abs(norm-1) > 1e-5 {
			t.Fatal("not normalized")
		}
	}
	for i := range text {
		if math.Abs(float64(text[i]-repeated[i])) > 1e-5 {
			t.Fatal("unstable output")
		}
	}
	t.Log("Go HTTP adapter: image/text 512 dimensions, normalized, repeated text stable; no application/quality claim")
}
