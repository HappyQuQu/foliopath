package remoteml

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/face"
	"github.com/HappyQuQu/foliopath/internal/media"
)

type faceDecoderStub struct {
	decoded face.DecodedImage
	err     error
}

func (d faceDecoderStub) DecodeFaceImage(_ context.Context, _ io.ReadSeeker, _ media.Format, limit int64, dimension int) (face.DecodedImage, error) {
	if limit != 1024 || dimension != face.MaxDecodeDimension {
		return face.DecodedImage{}, face.ErrInvalidInput
	}
	return d.decoded, d.err
}

func TestFaceRuntimeReconstructsPixelsAndQuality(t *testing.T) {
	expected := faceIdentityFixture()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			_ = json.NewEncoder(w).Encode(expected)
			return
		}
		file, _, err := r.FormFile("image")
		if err != nil {
			t.Error(err)
			return
		}
		defer file.Close()
		defer r.MultipartForm.RemoveAll()
		decoded, err := png.Decode(file)
		if err != nil {
			t.Error(err)
			return
		}
		if decoded.Bounds().Dx() != 64 || decoded.Bounds().Dy() != 32 {
			t.Error(decoded.Bounds())
		}
		red, green, blue, alpha := decoded.At(0, 0).RGBA()
		if red != 65535 || green != 0 || blue != 0 || alpha != 65535 {
			t.Error("pixel mismatch")
		}
		vector := make([]float32, 512)
		vector[0] = 1
		w.Header().Set("X-FolioPath-Runtime", expected.RuntimeContractSHA256)
		w.Header().Set("X-FolioPath-Cache", expected.CacheManifestSHA256)
		_ = json.NewEncoder(w).Encode(map[string]any{"faces": []any{map[string]any{
			"box": map[string]float32{"x": 0, "y": 0, "width": 1, "height": 1}, "detection": float32(.9), "embedding": vector}}})
	}))
	defer server.Close()
	c, _ := New(server.URL)
	defer c.Close()
	f, _ := c.BindFace(expected)
	rgb := make([]byte, 64*32*3)
	for i := 0; i < len(rgb); i += 3 {
		rgb[i] = 255
	}
	runtime, err := NewFaceRuntime(faceDecoderStub{decoded: face.DecodedImage{Width: 64, Height: 32, RGB: rgb}}, f)
	if err != nil {
		t.Fatal(err)
	}
	result, err := runtime.AnalyzeFaces(context.Background(), bytes.NewReader([]byte("original metadata not forwarded")), media.FormatJPEG, 1024)
	if err != nil || len(result) != 1 {
		t.Fatal(result, err)
	}
	if result[0].Quality != .45 || result[0].Detection != .9 || result[0].Embedding[0] != 1 {
		t.Fatal(result)
	}
}

func TestFaceRuntimeRejectsInvalidDecodeAndCancellation(t *testing.T) {
	c, _ := New("http://127.0.0.1:1")
	defer c.Close()
	f, _ := c.BindFace(faceIdentityFixture())
	for _, decoded := range []face.DecodedImage{{}, {Width: 1601, Height: 1}, {Width: 1, Height: 1, RGB: []byte{1, 2}}} {
		runtime, _ := NewFaceRuntime(faceDecoderStub{decoded: decoded}, f)
		if _, err := runtime.AnalyzeFaces(context.Background(), bytes.NewReader(nil), media.FormatPNG, 1024); !errors.Is(err, face.ErrInvalidOutput) {
			t.Fatal(err)
		}
	}
	runtime, _ := NewFaceRuntime(faceDecoderStub{err: face.ErrInvalidInput}, f)
	if _, err := runtime.AnalyzeFaces(context.Background(), bytes.NewReader(nil), media.FormatPNG, 1024); !errors.Is(err, face.ErrInvalidInput) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := runtime.AnalyzeFaces(ctx, bytes.NewReader(nil), media.FormatPNG, 1024); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	w := &facePNGWriter{ctx: context.Background()}
	if _, err := w.Write(make([]byte, MaxImageBytes+1)); !errors.Is(err, face.ErrInvalidInput) || w.buffer.Len() != 0 {
		t.Fatal(err)
	}
}
