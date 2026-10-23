//go:build libvips

package imagevips

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/media"
	"github.com/HappyQuQu/foliopath/internal/semantic"
	"github.com/davidbyttow/govips/v2/vips"
)

func pngProfile(t *testing.T) semantic.EncodedImagePreprocessor {
	t.Helper()
	p, err := NewRemoteImagePreprocessor(semantic.RemoteImagePNG)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// No source pixels or files come from a user's library. NoCompression makes the
// actual encoded PNG >32 MiB without trailing padding or a fake size reader.
func TestRemotePNGV4LargeFileAndBuffer(t *testing.T) {
	t.Cleanup(func() { vips.AssertNoLeaks(t) })
	f, err := os.CreateTemp(t.TempDir(), "synthetic-large-*.png")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	encoder := png.Encoder{CompressionLevel: png.NoCompression}
	if err := encoder.Encode(f, generatedRGB{image.Rect(0, 0, 5460, 3072)}); err != nil {
		t.Fatal(err)
	}
	size, _ := f.Seek(0, io.SeekEnd)
	if size <= semantic.RemoteImageMaxSourceBytes || size > semantic.RemotePNGMaxSourceBytes {
		t.Fatalf("fixture size: %d", size)
	}
	if output, err := New().PrepareRemoteSemanticImage(context.Background(), f, media.FormatPNG); output != nil || !errors.Is(err, semantic.ErrInvalidImageInput) {
		t.Fatal("legacy accepted enlarged PNG", err)
	}
	p := pngProfile(t)
	var first []byte
	for range 3 {
		output, err := p.PrepareRemoteSemanticImage(context.Background(), f, media.FormatPNG)
		if err != nil {
			t.Fatal(err)
		}
		config, err := png.DecodeConfig(bytes.NewReader(output))
		if err != nil || config.Width != 1024 || config.Height > 1024 || len(output) > semantic.RemoteImageMaxBytes {
			t.Fatalf("invalid output: %+v %v", config, err)
		}
		if first != nil && !bytes.Equal(first, output) {
			t.Fatal("repeated output changed")
		}
		first = output
		if vips.OpenImageRefs() != 0 {
			t.Fatal("native references retained")
		}
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	encoded, err := io.ReadAll(f)
	if err != nil {
		t.Fatal(err)
	}
	wantSource := sha256.Sum256(encoded)
	output, err := p.PrepareRemoteSemanticImage(context.Background(), bytes.NewReader(encoded), media.FormatPNG)
	if err != nil || !bytes.Equal(first, output) || sha256.Sum256(encoded) != wantSource {
		t.Fatal("file/buffer differ or source mutated", err)
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		t.Fatal("caller-owned file closed", err)
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, f); err != nil || !bytes.Equal(hash.Sum(nil), wantSource[:]) {
		t.Fatal("source file changed", err)
	}
	t.Logf("synthetic PNG bytes=%d pixels=%d; repeated FD and memory stream output identical", size, 5460*3072)
}

type generatedRGB struct{ rectangle image.Rectangle }

func (g generatedRGB) ColorModel() color.Model { return color.RGBAModel }
func (g generatedRGB) Bounds() image.Rectangle { return g.rectangle }
func (g generatedRGB) Opaque() bool            { return true }
func (g generatedRGB) At(x, y int) color.Color {
	return color.RGBA{uint8(x*37 + y*11), uint8(x*19 + y*29), uint8(x ^ y), 255}
}

func TestRemotePNGV4LegacyPixelsAlphaAndSixteenBit(t *testing.T) {
	t.Cleanup(func() { vips.AssertNoLeaks(t) })
	p := pngProfile(t)
	transparent := image.NewNRGBA(image.Rect(0, 0, 2051, 30))
	for i := 0; i < len(transparent.Pix); i += 4 {
		copy(transparent.Pix[i:i+4], []byte{200, 40, 17, 0})
	}
	sixteen := image.NewNRGBA64(image.Rect(0, 0, 2051, 30))
	for y := 0; y < 30; y++ {
		for x := 0; x < 2051; x++ {
			sixteen.SetNRGBA64(x, y, color.NRGBA64{R: 32768, G: 16384, B: 65535, A: 65535})
		}
	}
	for _, fixture := range []image.Image{transparent, sixteen, image.NewGray(image.Rect(0, 0, 37, 19))} {
		var encoded bytes.Buffer
		if err := png.Encode(&encoded, fixture); err != nil {
			t.Fatal(err)
		}
		want, err := New().PrepareRemoteSemanticImage(context.Background(), bytes.NewReader(encoded.Bytes()), media.FormatPNG)
		if err != nil {
			t.Fatal(err)
		}
		f, err := os.CreateTemp(t.TempDir(), "synthetic-small-*.png")
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if _, err := f.Write(encoded.Bytes()); err != nil {
			t.Fatal(err)
		}
		for _, source := range []io.ReadSeeker{bytes.NewReader(encoded.Bytes()), f} {
			got, err := p.PrepareRemoteSemanticImage(context.Background(), source, media.FormatPNG)
			if err != nil || !bytes.Equal(want, got) {
				t.Fatal("PNG profile changed legacy pixels", err)
			}
		}
	}
}

func TestRemotePNGV4BoundsCorruptionAndCancellation(t *testing.T) {
	t.Cleanup(func() { vips.AssertNoLeaks(t) })
	p := pngProfile(t)
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, image.NewGray(image.Rect(0, 0, 64, 32))); err != nil {
		t.Fatal(err)
	}
	overPixels := append([]byte(nil), encoded.Bytes()...)
	binary.BigEndian.PutUint32(overPixels[16:20], 8001)
	binary.BigEndian.PutUint32(overPixels[20:24], 5000)
	binary.BigEndian.PutUint32(overPixels[29:33], crc32.ChecksumIEEE(overPixels[12:29]))
	for name, source := range map[string]io.ReadSeeker{
		"pixels":    bytes.NewReader(overPixels),
		"size":      semanticSizedReader{bytes.NewReader(encoded.Bytes()), semantic.RemotePNGMaxSourceBytes + 1},
		"truncated": bytes.NewReader(encoded.Bytes()[:len(encoded.Bytes())/2]),
		"invalid":   bytes.NewReader([]byte("invalid PNG")),
	} {
		t.Run(name, func(t *testing.T) {
			out, err := p.PrepareRemoteSemanticImage(context.Background(), source, media.FormatPNG)
			if err == nil || out != nil {
				t.Fatalf("accepted invalid input: %s", name)
			}
			if vips.OpenImageRefs() != 0 {
				t.Fatal("rejection leaked native references")
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	source := &semanticCancellingReader{Reader: bytes.NewReader(encoded.Bytes()), cancel: cancel}
	if out, err := p.PrepareRemoteSemanticImage(ctx, source, media.FormatPNG); !errors.Is(err, context.Canceled) || out != nil || source.reads != 1 {
		t.Fatal("cancellation lost or extra reads", err, source.reads)
	}
	cancel()
	empty := &semanticChunkReader{Reader: bytes.NewReader(encoded.Bytes()), empty: 100}
	if _, err := p.PrepareRemoteSemanticImage(context.Background(), empty, media.FormatPNG); !errors.Is(err, semantic.ErrInvalidImageInput) || empty.reads != 100 {
		t.Fatal("no-progress guard lost", err, empty.reads)
	}
	if _, err := NewRemoteImagePreprocessor("unknown"); !errors.Is(err, semantic.ErrInvalidImageInput) {
		t.Fatal("unknown profile accepted")
	}
}

func TestRemotePNGV4PixelBoundary(t *testing.T) {
	t.Cleanup(func() { vips.AssertNoLeaks(t) })
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, image.NewGray(image.Rect(0, 0, 8000, 5000))); err != nil {
		t.Fatal(err)
	}
	p := pngProfile(t)
	if out, err := p.PrepareRemoteSemanticImage(context.Background(), bytes.NewReader(encoded.Bytes()), media.FormatPNG); err != nil || len(out) == 0 {
		t.Fatal("40 MP boundary rejected", err)
	}
	if _, err := New().PrepareRemoteSemanticImage(context.Background(), bytes.NewReader(encoded.Bytes()), media.FormatPNG); !errors.Is(err, semantic.ErrInvalidImageInput) {
		t.Fatal("legacy pixel contract changed")
	}
}

func TestRemotePNGV4HeaderBudget(t *testing.T) {
	t.Cleanup(func() { vips.AssertNoLeaks(t) })
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, image.NewPaletted(image.Rect(0, 0, 2, 2), color.Palette{color.Black, color.White})); err != nil {
		t.Fatal(err)
	}
	// A palette PNG needs PLTE to finish DecodeConfig. A valid ancillary chunk
	// before PLTE must not allow the bounded header probe to read past 1 MiB.
	chunk := make([]byte, semantic.RemotePNGMaxHeaderBytes+12)
	binary.BigEndian.PutUint32(chunk[:4], semantic.RemotePNGMaxHeaderBytes)
	copy(chunk[4:8], "tEXt")
	copy(chunk[8:], "Comment\x00")
	binary.BigEndian.PutUint32(chunk[len(chunk)-4:], crc32.ChecksumIEEE(chunk[4:len(chunk)-4]))
	value := append(append(append([]byte(nil), encoded.Bytes()[:33]...), chunk...), encoded.Bytes()[33:]...)
	if _, err := png.DecodeConfig(bytes.NewReader(value)); err != nil {
		t.Fatal("synthetic header must be valid without a budget", err)
	}
	if out, err := pngProfile(t).PrepareRemoteSemanticImage(context.Background(), bytes.NewReader(value), media.FormatPNG); !errors.Is(err, semantic.ErrInvalidImageInput) || out != nil {
		t.Fatal("oversized PNG header accepted", err)
	}
}

func TestRemotePNGV4UsesSharedAdmissionBeforeCallerReads(t *testing.T) {
	t.Cleanup(func() { vips.AssertNoLeaks(t) })
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, image.NewGray(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	release, err := beginLargeImageDecode(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	source := &semanticChunkReader{Reader: bytes.NewReader(encoded.Bytes())}
	if out, err := pngProfile(t).PrepareRemoteSemanticImage(ctx, source, media.FormatPNG); !errors.Is(err, context.DeadlineExceeded) || out != nil || source.reads != 0 {
		t.Fatal("PNG bypassed shared admission or read before cancellation", err, source.reads)
	}
}
