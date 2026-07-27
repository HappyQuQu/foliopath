package videoffmpeg

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/media"
)

func TestSyntheticVideoMatrixThroughProductionAdapter(t *testing.T) {
	if testing.Short() {
		t.Skip("synthetic FFmpeg integration is disabled in short mode")
	}
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("ffmpeg is not installed")
	}
	ffprobe, err := exec.LookPath("ffprobe")
	if err != nil {
		t.Skip("ffprobe is not installed")
	}
	encoders, err := exec.Command(ffmpeg, "-hide_banner", "-encoders").CombinedOutput()
	if err != nil || !bytes.Contains(encoders, []byte("libwebp")) {
		t.Skip("ffmpeg does not provide the required libwebp encoder")
	}
	processor, err := New(Options{
		FFmpegPath: ffmpeg, FFprobePath: ffprobe, Timeout: 10 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name      string
		format    media.Format
		extension string
		codec     string
		playback  media.PlaybackStatus
	}{
		{"mp4 h264", media.FormatMP4, ".mp4", "libx264", media.PlaybackPlayable},
		{"mov h264", media.FormatMOV, ".mov", "libx264", media.PlaybackPlayable},
		{"mkv h264", media.FormatMKV, ".mkv", "libx264", media.PlaybackUnknown},
		{"mkv ffv1", media.FormatMKV, ".mkv", "ffv1", media.PlaybackUnsupportedCodec},
	} {
		t.Run(test.name, func(t *testing.T) {
			filename := filepath.Join(t.TempDir(), "fixture"+test.extension)
			command := exec.Command(
				ffmpeg,
				"-v", "error",
				"-f", "lavfi",
				"-i", "testsrc=size=96x64:rate=4",
				"-t", "1",
				"-pix_fmt", "yuv420p",
				"-c:v", test.codec,
				filename,
			)
			if output, err := command.CombinedOutput(); err != nil {
				t.Fatalf("generate fixture: %v: %s", err, output)
			}
			source, err := os.Open(filename)
			if err != nil {
				t.Fatal(err)
			}
			defer source.Close()
			result, err := processor.Process(
				context.Background(), source, test.format,
			)
			if err != nil {
				t.Fatal(err)
			}
			if result.Metadata.Width != 96 || result.Metadata.Height != 64 ||
				result.Metadata.DurationMS == nil || *result.Metadata.DurationMS < 900 ||
				result.Metadata.PlaybackStatus != test.playback ||
				len(result.Thumbnail.Bytes) < 12 ||
				!bytes.Equal(result.Thumbnail.Bytes[:4], []byte("RIFF")) ||
				!bytes.Equal(result.Thumbnail.Bytes[8:12], []byte("WEBP")) {
				t.Fatalf("result = %#v", result)
			}
		})
	}

	corrupt := filepath.Join(t.TempDir(), "corrupt.mp4")
	if err := os.WriteFile(corrupt, []byte("not a video"), 0o600); err != nil {
		t.Fatal(err)
	}
	source, err := os.Open(corrupt)
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	if _, err := processor.Process(
		context.Background(), source, media.FormatMP4,
	); err == nil {
		t.Fatal("corrupt video unexpectedly processed")
	}
}
