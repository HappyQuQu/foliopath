//go:build libvips

package imagevips

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/media"
	"github.com/HappyQuQu/foliopath/internal/semantic"
	"github.com/davidbyttow/govips/v2/vips"
)

func TestRemoteSemanticImage(t *testing.T) {
	t.Cleanup(func() { vips.AssertNoLeaks(t) })
	var source bytes.Buffer
	if err := png.Encode(&source, syntheticImage(2048, 1024)); err != nil {
		t.Fatal(err)
	}
	output, err := New().PrepareRemoteSemanticImage(context.Background(), bytes.NewReader(source.Bytes()), media.FormatPNG)
	if err != nil {
		t.Fatal(err)
	}
	config, err := png.DecodeConfig(bytes.NewReader(output))
	if err != nil || config.Width != 1024 || config.Height != 512 {
		t.Fatalf("invalid output: %+v %v", config, err)
	}
	for _, format := range []media.Format{media.FormatJPEG, media.FormatMP4} {
		if result, err := New().PrepareRemoteSemanticImage(context.Background(), bytes.NewReader(source.Bytes()), format); result != nil || !errors.Is(err, semantic.ErrInvalidImageInput) {
			t.Fatalf("accepted mismatch: %v", err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := New().PrepareRemoteSemanticImage(ctx, bytes.NewReader(source.Bytes()), media.FormatPNG); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := New().PrepareRemoteSemanticImage(context.Background(), bytes.NewReader([]byte("corrupt")), media.FormatPNG); !errors.Is(err, semantic.ErrInvalidImageInput) {
		t.Fatal(err)
	}
}

func TestRemoteSemanticImageBudgets(t *testing.T) {
	t.Cleanup(func() { vips.AssertNoLeaks(t) })
	ctx := context.Background()
	if _, err := New().PrepareRemoteSemanticImage(ctx, bytes.NewReader(make([]byte, semantic.RemoteImageMaxSourceBytes+1)), media.FormatPNG); !errors.Is(err, semantic.ErrInvalidImageInput) {
		t.Fatalf("accepted source over byte budget: %v", err)
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, image.NewGray(image.Rect(0, 0, 4001, 4000))); err != nil {
		t.Fatal(err)
	}
	if _, err := New().PrepareRemoteSemanticImage(ctx, bytes.NewReader(encoded.Bytes()), media.FormatPNG); !errors.Is(err, semantic.ErrInvalidImageInput) {
		t.Fatalf("accepted source over pixel budget: %v", err)
	}
}

func TestRemoteSemanticJPEGShrink(t *testing.T) {
	for _, tc := range []struct{ width, height, factor int }{
		{4000, 4000, 1}, {4001, 4000, 2}, {8000, 8000, 2},
		{8001, 8000, 4}, {12000, 9000, 4}, {15000, 12000, 4},
		{15001, 12000, 0}, {0, 100, 0}, {-1, 1, 0},
	} {
		factor, ok := remoteSemanticJPEGShrink(tc.width, tc.height)
		if factor != tc.factor || ok != (tc.factor != 0) {
			t.Fatalf("%dx%d: got %d/%v want %d", tc.width, tc.height, factor, ok, tc.factor)
		}
	}
}

type semanticZeroReader struct{}

type semanticCancellingReader struct {
	*bytes.Reader
	cancel context.CancelFunc
	reads  int
	fail   bool
}

func (r *semanticCancellingReader) Read(p []byte) (int, error) {
	r.reads++
	r.cancel()
	if r.fail {
		return 0, io.ErrUnexpectedEOF
	}
	return r.Reader.Read(p[:min(len(p), 1)])
}

func TestRemoteSemanticCancellationDuringRead(t *testing.T) {
	for _, fail := range []bool{false, true} {
		ctx, cancel := context.WithCancel(context.Background())
		source := &semanticCancellingReader{Reader: bytes.NewReader(make([]byte, 32)), cancel: cancel, fail: fail}
		output, err := New().PrepareRemoteSemanticImage(ctx, source, media.FormatJPEG)
		cancel()
		if !errors.Is(err, context.Canceled) || output != nil || source.reads != 1 {
			t.Fatalf("fail=%v: error=%v reads=%d; cancellation must stop further reads", fail, err, source.reads)
		}
	}
}

type semanticChunkReader struct {
	*bytes.Reader
	empty, reads, maximum int
}

func (r *semanticChunkReader) Read(p []byte) (int, error) {
	r.reads++
	r.maximum = max(r.maximum, len(p))
	if r.empty > 0 {
		r.empty--
		return 0, nil
	}
	n, err := r.Reader.Read(p)
	if r.Len() == 0 && n > 0 {
		return n, io.EOF // Valid Reader contract: final data together with EOF.
	}
	return n, err
}

func TestRemoteSemanticChunkedReadAndNoProgress(t *testing.T) {
	var encoded bytes.Buffer
	if err := jpeg.Encode(&encoded, syntheticImage(32, 16), nil); err != nil {
		t.Fatal(err)
	}
	encoded.Write(make([]byte, 200*1024))
	for _, empty := range []int{0, 1, 100} {
		r := &semanticChunkReader{Reader: bytes.NewReader(encoded.Bytes()), empty: empty}
		output, err := New().PrepareRemoteSemanticImage(context.Background(), r, media.FormatJPEG)
		if empty == 100 {
			if !errors.Is(err, semantic.ErrInvalidImageInput) || output != nil || r.reads != 100 {
				t.Fatalf("no-progress guard: reads=%d err=%v", r.reads, err)
			}
		} else if err != nil || len(output) == 0 || r.reads < 4 {
			t.Fatalf("chunked read: reads=%d err=%v", r.reads, err)
		}
		if r.maximum > 64*1024 {
			t.Fatalf("read request too large: %d", r.maximum)
		}
	}
}

type semanticSizedReader struct {
	*bytes.Reader
	declared int64
}

func (r semanticSizedReader) Seek(offset int64, whence int) (int64, error) {
	if whence == io.SeekEnd {
		return r.declared, nil
	}
	return r.Reader.Seek(offset, whence)
}

func TestRemoteSemanticRejectsChangingSourceSize(t *testing.T) {
	for _, size := range []int64{2, 4} {
		source := semanticSizedReader{bytes.NewReader([]byte{1, 2, 3}), size}
		if _, err := New().PrepareRemoteSemanticImage(context.Background(), source, media.FormatJPEG); !errors.Is(err, semantic.ErrInvalidImageInput) {
			t.Fatalf("accepted changing source size %d: %v", size, err)
		}
	}
}

func (semanticZeroReader) Read(p []byte) (int, error) { clear(p); return len(p), nil }
func (semanticZeroReader) Seek(_ int64, whence int) (int64, error) {
	if whence == io.SeekEnd {
		return semantic.RemoteJPEGMaxSourceBytes + 1, nil
	}
	return 0, nil
}

func TestRemoteSemanticJPEGByteBudget(t *testing.T) {
	// Unbounded source is stopped at the configured byte budget, without a
	// second oversized test allocation.
	if _, err := New().PrepareRemoteSemanticImage(context.Background(), semanticZeroReader{}, media.FormatJPEG); !errors.Is(err, semantic.ErrInvalidImageInput) {
		t.Fatalf("accepted unbounded source: %v", err)
	}
	var encoded bytes.Buffer
	if err := jpeg.Encode(&encoded, syntheticImage(32, 16), nil); err != nil {
		t.Fatal(err)
	}
	// JPEG trailing bytes remain bounded, but verify the old 32 MiB ceiling
	// no longer rejects an otherwise valid JPEG by byte count alone.
	if _, err := io.CopyN(&encoded, semanticZeroReader{}, semantic.RemoteImageMaxSourceBytes); err != nil {
		t.Fatal(err)
	}
	if _, err := New().PrepareRemoteSemanticImage(context.Background(), bytes.NewReader(encoded.Bytes()), media.FormatJPEG); err != nil {
		t.Fatalf("rejected bounded JPEG above old byte budget: %v", err)
	}
}

func TestRemoteSemanticJPEGFileStreamsAboveLegacyBudget(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "semantic-large-*.jpg")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := jpeg.Encode(file, syntheticImage(64, 32), nil); err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate((64 << 20) + 1); err != nil {
		t.Fatal(err)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	output, err := New().PrepareRemoteSemanticImage(context.Background(), file, media.FormatJPEG)
	if err != nil {
		t.Fatal(err)
	}
	config, err := png.DecodeConfig(bytes.NewReader(output))
	if err != nil || config.Width != 64 || config.Height != 32 {
		t.Fatalf("invalid streamed output: %+v %v", config, err)
	}
}

func TestRemoteSemanticJPEGRecoversNarrowTruncation(t *testing.T) {
	var encoded bytes.Buffer
	if err := jpeg.Encode(&encoded, syntheticImage(2048, 1024), &jpeg.Options{Quality: 90}); err != nil {
		t.Fatal(err)
	}
	truncated := encoded.Bytes()[:encoded.Len()-2]
	output, err := New().PrepareRemoteSemanticImage(context.Background(), bytes.NewReader(truncated), media.FormatJPEG)
	if err != nil {
		t.Fatal(err)
	}
	config, err := png.DecodeConfig(bytes.NewReader(output))
	if err != nil || config.Width != 1024 || config.Height != 512 {
		t.Fatalf("invalid recovered output: %+v %v", config, err)
	}
}

func TestRemoteSemanticLargeJPEG(t *testing.T) {
	if testing.Short() {
		t.Skip("large synthetic JPEG; run under bounded Docker memory")
	}
	t.Cleanup(func() { vips.AssertNoLeaks(t) })
	var encoded bytes.Buffer
	if err := jpeg.Encode(&encoded, largeFaceFixture{12000, 9000}, &jpeg.Options{Quality: 75}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		output, err := New().PrepareRemoteSemanticImage(context.Background(), bytes.NewReader(encoded.Bytes()), media.FormatJPEG)
		if err != nil {
			t.Fatal(err)
		}
		config, err := png.DecodeConfig(bytes.NewReader(output))
		if err != nil || config.Width != 1024 || config.Height != 768 {
			t.Fatalf("invalid bounded output: %+v %v", config, err)
		}
	}
	if _, err := New().PrepareRemoteSemanticImage(context.Background(), bytes.NewReader(encoded.Bytes()), media.FormatPNG); !errors.Is(err, semantic.ErrInvalidImageInput) {
		t.Fatalf("accepted mismatched format: %v", err)
	}
}

func TestRemoteSemanticProgressiveJPEG(t *testing.T) {
	fixture := os.Getenv("FOLIOPATH_SYNTHETIC_PROGRESSIVE_JPEG")
	if fixture == "" {
		t.Skip("explicit generated progressive JPEG required")
	}
	encoded, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		output, err := New().PrepareRemoteSemanticImage(context.Background(), bytes.NewReader(encoded), media.FormatJPEG)
		if err != nil {
			t.Fatal(err)
		}
		config, err := png.DecodeConfig(bytes.NewReader(output))
		if err != nil || config.Width != 1024 || config.Height != 1024 {
			t.Fatalf("invalid progressive output: %+v %v", config, err)
		}
	}
}
