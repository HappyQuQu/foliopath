package videoffmpeg

import (
	"bytes"
	"context"
	"crypto/sha256"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
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
		StoryboardTempRoot: filepath.Join(t.TempDir(), "storyboard"),
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
		{"mkv ffv1", media.FormatMKV, ".mkv", "ffv1", media.PlaybackUnknown},
		{"avi mpeg4", media.FormatAVI, ".avi", "mpeg4", media.PlaybackUnknown},
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

	t.Run("four and ten frame storyboards", func(t *testing.T) {
		filename := filepath.Join(t.TempDir(), "storyboard.mp4")
		command := exec.Command(
			ffmpeg,
			"-v", "error",
			"-f", "lavfi",
			"-i", "testsrc=size=96x64:rate=4",
			"-t", "10",
			"-pix_fmt", "yuv420p",
			"-c:v", "libx264",
			filename,
		)
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("generate storyboard fixture: %v: %s", err, output)
		}
		source, err := os.Open(filename)
		if err != nil {
			t.Fatal(err)
		}
		defer source.Close()
		sourceBefore, err := os.ReadFile(filename)
		if err != nil {
			t.Fatal(err)
		}
		beforeHash := sha256.Sum256(sourceBefore)
		for _, request := range []media.StoryboardRequest{
			{
				TimestampsMS: []int64{2000, 4000, 6000, 8000},
				Columns:      4, Rows: 1, CellWidth: 96, CellHeight: 64,
			},
			{
				TimestampsMS: []int64{
					909, 1818, 2727, 3636, 4545,
					5454, 6363, 7272, 8181, 9090,
				},
				Columns: 5, Rows: 2, CellWidth: 96, CellHeight: 64,
			},
		} {
			if _, err := source.Seek(0, 0); err != nil {
				t.Fatal(err)
			}
			result, err := processor.Storyboard(
				context.Background(),
				source,
				media.FormatMP4,
				request,
			)
			if err != nil {
				t.Fatal(err)
			}
			if err := media.ValidateStoryboardResult(request, result); err != nil {
				t.Fatalf("storyboard result = %#v: %v", result, err)
			}
		}
		sourceAfter, err := os.ReadFile(filename)
		if err != nil {
			t.Fatal(err)
		}
		afterHash := sha256.Sum256(sourceAfter)
		if !slices.Equal(beforeHash[:], afterHash[:]) {
			t.Fatal("storyboard processing modified the source video")
		}
	})

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

func TestPinnedFixtureFourAndTenFrameStoryboards(t *testing.T) {
	fixture := os.Getenv("FOLIOPATH_TEST_STORYBOARD_FIXTURE")
	ffmpeg := os.Getenv("FOLIOPATH_TEST_FFMPEG")
	if fixture == "" || ffmpeg == "" {
		t.Skip("pinned fixture and production FFmpeg paths are required")
	}
	encoders, err := exec.Command(ffmpeg, "-hide_banner", "-encoders").CombinedOutput()
	if err != nil || !bytes.Contains(encoders, []byte("libwebp")) {
		t.Fatal("production FFmpeg does not provide the required libwebp encoder")
	}
	sourceBefore, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatal(err)
	}
	beforeHash := sha256.Sum256(sourceBefore)
	processor, err := New(Options{
		FFmpegPath: ffmpeg,
		Timeout:    10 * time.Second,
		StoryboardTempRoot: filepath.Join(
			t.TempDir(), "storyboard",
		),
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, request := range []media.StoryboardRequest{
		{
			TimestampsMS: []int64{2000, 4000, 6000, 8000},
			Columns:      4, Rows: 1, CellWidth: 96, CellHeight: 64,
		},
		{
			TimestampsMS: []int64{
				909, 1818, 2727, 3636, 4545,
				5454, 6363, 7272, 8181, 9090,
			},
			Columns: 5, Rows: 2, CellWidth: 96, CellHeight: 64,
		},
	} {
		source, err := os.Open(fixture)
		if err != nil {
			t.Fatal(err)
		}
		result, processErr := processor.Storyboard(
			context.Background(), source, media.FormatMP4, request,
		)
		closeErr := source.Close()
		if processErr != nil {
			t.Fatal(processErr)
		}
		if closeErr != nil {
			t.Fatal(closeErr)
		}
		if err := media.ValidateStoryboardResult(request, result); err != nil {
			t.Fatalf("storyboard result = %#v: %v", result, err)
		}
	}
	sourceAfter, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatal(err)
	}
	afterHash := sha256.Sum256(sourceAfter)
	if beforeHash != afterHash {
		t.Fatal("storyboard processing modified the source video")
	}
}
