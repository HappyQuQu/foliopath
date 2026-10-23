//go:build libvips

package imagevips

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"image/jpeg"
	"os"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/face"
	"github.com/HappyQuQu/foliopath/internal/media"
	"github.com/davidbyttow/govips/v2/vips"
)

func TestLargeImageDecodeAdmissionCancellationAndRelease(t *testing.T) {
	release, err := beginLargeImageDecode(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	blockedRelease, err := beginLargeImageDecode(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		if blockedRelease != nil {
			blockedRelease()
		}
		release()
		t.Fatalf("waiting admission ignored cancellation: %v", err)
	}
	release()
	next, err := beginLargeImageDecode(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	next()
}

func TestLargeImageHeaderProbeRestoresPositionAndSkipsCallerReaders(t *testing.T) {
	var source bytes.Buffer
	if err := jpeg.Encode(&source, largeFaceFixture{320, 240}, nil); err != nil {
		t.Fatal(err)
	}
	r := bytes.NewReader(source.Bytes())
	r.Seek(9, 0)
	if largeImageSourceNeedsReclaim(r, media.FormatJPEG) {
		t.Fatal("small JPEG took large lane")
	}
	if position, _ := r.Seek(0, 1); position != 9 {
		t.Fatalf("header probe changed position: %d", position)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wrapped := cancellingFaceReader{bytes.NewReader(source.Bytes()), cancel}
	if !largeImageSourceNeedsReclaim(wrapped, media.FormatJPEG) {
		t.Fatal("unknown reader did not take conservative lane")
	}
	if ctx.Err() != nil {
		t.Fatal("resource probe read caller-owned reader")
	}
}

func TestLargeFaceMemoryHygieneKeepsPixelsAndClosesRefs(t *testing.T) {
	var source bytes.Buffer
	if err := jpeg.Encode(&source, largeFaceFixture{320, 240}, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatal(err)
	}
	want, err := New().DecodeFaceImage(context.Background(), bytes.NewReader(source.Bytes()), media.FormatJPEG, face.MaxInputBytes, face.MaxDecodeDimension)
	if err != nil {
		t.Fatal(err)
	}
	padded := make([]byte, largeImageReclaimBytes)
	copy(padded, source.Bytes()) // trailing padding keeps the encoded pixels identical
	f, err := os.CreateTemp(t.TempDir(), "synthetic-face-*.jpg")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := f.Write(padded); err != nil {
		t.Fatal(err)
	}
	for _, reader := range []interface {
		Read([]byte) (int, error)
		Seek(int64, int) (int64, error)
	}{bytes.NewReader(padded), f} {
		before := vips.OpenImageRefs()
		got, err := New().DecodeFaceImage(context.Background(), reader, media.FormatJPEG, face.MaxInputBytes, face.MaxDecodeDimension)
		if err != nil {
			t.Fatal(err)
		}
		if got.Width != want.Width || got.Height != want.Height || sha256.Sum256(got.RGB) != sha256.Sum256(want.RGB) {
			t.Fatal("reclamation changed pixels")
		}
		if vips.OpenImageRefs() != before {
			t.Fatal("decode leaked a native reference")
		}
	}
}

type cancellingFaceReader struct {
	*bytes.Reader
	cancel context.CancelFunc
}

func (r cancellingFaceReader) Read(p []byte) (int, error) {
	n, err := r.Reader.Read(p)
	r.cancel()
	return n, err
}

func TestLargeFaceMemoryHygieneCancellationClosesRefs(t *testing.T) {
	var source bytes.Buffer
	if err := jpeg.Encode(&source, largeFaceFixture{320, 240}, nil); err != nil {
		t.Fatal(err)
	}
	value := make([]byte, largeImageReclaimBytes)
	copy(value, source.Bytes())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	before := vips.OpenImageRefs()
	_, err := New().DecodeFaceImage(ctx, cancellingFaceReader{bytes.NewReader(value), cancel}, media.FormatJPEG, face.MaxInputBytes, face.MaxDecodeDimension)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
	if vips.OpenImageRefs() != before {
		t.Fatal("cancelled decode leaked native reference")
	}
}

func TestLargeFaceMemoryHygieneCorruptInputClosesRefs(t *testing.T) {
	var source bytes.Buffer
	if err := jpeg.Encode(&source, largeFaceFixture{320, 240}, nil); err != nil {
		t.Fatal(err)
	}
	value := make([]byte, largeImageReclaimBytes)
	copy(value, []byte("synthetic deliberately invalid JPEG; no original media"))
	before := vips.OpenImageRefs()
	for i := 0; i < 2; i++ {
		_, err := New().DecodeFaceImage(context.Background(), bytes.NewReader(value), media.FormatJPEG, face.MaxInputBytes, face.MaxDecodeDimension)
		if err == nil {
			t.Fatal("corrupt input accepted")
		}
		if vips.OpenImageRefs() != before {
			t.Fatal("failed decode leaked native reference")
		}
	}
}
