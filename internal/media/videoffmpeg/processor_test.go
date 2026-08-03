package videoffmpeg

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/media"
)

func TestVideoProcessorUsesInheritedDescriptorAndReturnsSafeFailure(t *testing.T) {
	directory := t.TempDir()
	probe := filepath.Join(directory, "ffprobe")
	ffmpeg := filepath.Join(directory, "ffmpeg")
	writeExecutable(t, probe, `#!/bin/sh
case "$*" in
  *"/dev/fd/3"*|*"/proc/self/fd/3"*) ;;
  *) echo "missing inherited descriptor" >&2; exit 2 ;;
esac
case "$*" in
  *"-nostdin"*) echo "ffprobe does not portably support -nostdin" >&2; exit 2 ;;
esac
case "$*" in
  *"-threads 1"*) ;;
  *) echo "missing decoder thread bound" >&2; exit 2 ;;
esac
printf '%s' '{"streams":[{"codec_type":"video","codec_name":"h264","width":96,"height":64,"duration":"1.250"}],"format":{"duration":"1.250"}}'
`)
	writeExecutable(t, ffmpeg, `#!/bin/sh
case "$*" in
  *"/dev/fd/3"*|*"/proc/self/fd/3"*) ;;
  *) echo "missing inherited descriptor" >&2; exit 2 ;;
esac
case "$*" in
  *"-threads 1"*"-filter_threads 1"*) ;;
  *) echo "missing decoder/filter thread bounds" >&2; exit 2 ;;
esac
printf 'synthetic-webp'
`)
	sourcePath := filepath.Join(directory, "source.mp4")
	if err := os.WriteFile(sourcePath, []byte("source"), 0o600); err != nil {
		t.Fatal(err)
	}
	source, err := os.Open(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	processor, err := New(Options{
		FFprobePath: probe, FFmpegPath: ffmpeg, Timeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := processor.Process(context.Background(), source, media.FormatMP4)
	if err != nil {
		t.Fatal(err)
	}
	if result.Metadata.Width != 96 || result.Metadata.Height != 64 ||
		result.Metadata.DurationMS == nil || *result.Metadata.DurationMS != 1250 ||
		result.Metadata.PlaybackStatus != media.PlaybackPlayable ||
		result.Thumbnail.Width != 96 || result.Thumbnail.Height != 64 {
		t.Fatalf("result = %#v", result)
	}

	writeExecutable(t, probe, "#!/bin/sh\necho '/library/private raw stderr' >&2\nexit 1\n")
	_, err = processor.Process(context.Background(), source, media.FormatMP4)
	if !errors.Is(err, media.ErrInvalidMedia) ||
		errors.Is(err, exec.ErrNotFound) {
		t.Fatalf("safe failure = %v", err)
	}
}

func TestClassifyFFmpegReasonReturnsStableSafeReasons(t *testing.T) {
	tests := map[string]media.FailureReason{
		"moov atom not found":                                    media.ReasonMissingMoovAtom,
		"Invalid data found when processing input":               media.ReasonInvalidData,
		"Error while decoding stream #0:0":                       media.ReasonDecodeFailed,
		"Decoder h265 not found":                                 media.ReasonDecoderUnavailable,
		"unrecognized diagnostic containing /private/source.mp4": media.ReasonToolFailed,
	}
	for stderr, want := range tests {
		if got := classifyFFmpegReason(stderr); got != want {
			t.Errorf("classifyFFmpegReason(%q) = %q, want %q", stderr, got, want)
		}
	}
}

func TestVideoProcessorPropagatesDeadlineAndRejectsWrongFormat(t *testing.T) {
	directory := t.TempDir()
	slow := filepath.Join(directory, "slow")
	marker := filepath.Join(directory, "descendant-survived")
	writeExecutable(t, slow, "#!/bin/sh\n(sleep 0.4; touch \""+marker+"\") &\nwait\n")
	sourcePath := filepath.Join(directory, "source.mp4")
	if err := os.WriteFile(sourcePath, []byte("source"), 0o600); err != nil {
		t.Fatal(err)
	}
	source, err := os.Open(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	processor, err := New(Options{
		FFprobePath: slow, FFmpegPath: slow, Timeout: 100 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := processor.Process(
		context.Background(), source, media.FormatJPEG,
	); !errors.Is(err, media.ErrUnsupportedMedia) {
		t.Fatalf("unsupported format error = %v", err)
	}
	started := time.Now()
	if _, err := processor.Process(
		context.Background(), source, media.FormatMP4,
	); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("deadline error = %v", err)
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("process tree cancellation took %s", elapsed)
	}
	time.Sleep(500 * time.Millisecond)
	if _, err := os.Stat(marker); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("cancelled descendant survived: %v", err)
	}
}

func TestVideoProcessorRejectsOversizedSparseFileBeforeTools(t *testing.T) {
	directory := t.TempDir()
	called := filepath.Join(directory, "called")
	tool := filepath.Join(directory, "tool")
	writeExecutable(t, tool, "#!/bin/sh\ntouch \""+called+"\"\nexit 1\n")
	sourcePath := filepath.Join(directory, "oversized.mp4")
	source, err := os.Create(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	if err := source.Truncate(media.MaxVideoSourceBytes + 1); err != nil {
		t.Fatal(err)
	}
	processor, err := New(Options{
		FFprobePath: tool, FFmpegPath: tool, Timeout: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := processor.Process(
		context.Background(), source, media.FormatMP4,
	); !errors.Is(err, media.ErrInvalidMedia) {
		t.Fatalf("oversized video error = %v", err)
	}
	if _, err := os.Stat(called); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("media tool was invoked: %v", err)
	}
}

func TestVideoProcessorRejectsHostileMetadataBeforePosterDecode(t *testing.T) {
	directory := t.TempDir()
	probe := filepath.Join(directory, "ffprobe")
	ffmpeg := filepath.Join(directory, "ffmpeg")
	posterCalled := filepath.Join(directory, "poster-called")
	writeExecutable(t, probe, `#!/bin/sh
printf '%s' '{"streams":[{"codec_type":"video","codec_name":"h264","width":20000,"height":20000,"duration":"1"}],"format":{"duration":"1"}}'
`)
	writeExecutable(t, ffmpeg, "#!/bin/sh\ntouch \""+posterCalled+"\"\n")
	sourcePath := filepath.Join(directory, "source.mp4")
	if err := os.WriteFile(sourcePath, []byte("source"), 0o600); err != nil {
		t.Fatal(err)
	}
	source, err := os.Open(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	processor, err := New(Options{
		FFprobePath: probe, FFmpegPath: ffmpeg, Timeout: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := processor.Process(
		context.Background(), source, media.FormatMP4,
	); !errors.Is(err, media.ErrInvalidMedia) {
		t.Fatalf("hostile dimensions error = %v", err)
	}
	if _, err := os.Stat(posterCalled); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("poster decoder was invoked: %v", err)
	}
}

func TestVideoProcessorCapsUntrustedToolOutput(t *testing.T) {
	directory := t.TempDir()
	probe := filepath.Join(directory, "ffprobe")
	writeExecutable(t, probe, `#!/bin/sh
dd if=/dev/zero bs=1048576 count=9 2>/dev/null
`)
	sourcePath := filepath.Join(directory, "source.mp4")
	if err := os.WriteFile(sourcePath, []byte("source"), 0o600); err != nil {
		t.Fatal(err)
	}
	source, err := os.Open(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	processor, err := New(Options{
		FFprobePath: probe, FFmpegPath: probe, Timeout: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := processor.run(
		context.Background(), probe, source,
	); !errors.Is(err, media.ErrProcessingFailed) {
		t.Fatalf("oversized tool output error = %v", err)
	}
}

func TestStoryboardUsesBoundedFastSeeksAndCleansTemporaryFrames(t *testing.T) {
	directory := t.TempDir()
	ffmpeg := filepath.Join(directory, "ffmpeg")
	tempRoot := filepath.Join(directory, "storyboard-temp")
	calls := filepath.Join(directory, "calls")
	writeExecutable(t, ffmpeg, `#!/bin/sh
printf '%s\n' "$*" >> "`+calls+`"
case "$*" in
  *"xstack="*) printf 'RIFF\004\000\000\000WEBP' ;;
  *"-ss "*"/dev/fd/3"*|*"-ss "*"/proc/self/fd/3"*)
    printf '\211PNG\r\n\032\nframe'
    ;;
  *) exit 2 ;;
esac
`)
	sourcePath := filepath.Join(directory, "source.mp4")
	if err := os.WriteFile(sourcePath, []byte("source"), 0o600); err != nil {
		t.Fatal(err)
	}
	source, err := os.Open(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	processor, err := New(Options{
		FFmpegPath: ffmpeg, StoryboardTempRoot: tempRoot,
		StoryboardTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	request := media.StoryboardRequest{
		TimestampsMS: []int64{1000, 2000, 3000, 4000},
		Columns:      4,
		Rows:         1,
		CellWidth:    320,
		CellHeight:   180,
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
		t.Fatal(err)
	}
	callBytes, err := os.ReadFile(calls)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(splitNonemptyLines(string(callBytes))); got != 5 {
		t.Fatalf("FFmpeg calls = %d, want four seeks plus one compose", got)
	}
	entries, err := os.ReadDir(tempRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("temporary storyboard entries remain: %v", entries)
	}
}

func TestStoryboardIsAllOrNothing(t *testing.T) {
	directory := t.TempDir()
	ffmpeg := filepath.Join(directory, "ffmpeg")
	writeExecutable(t, ffmpeg, `#!/bin/sh
case "$*" in
  *"-ss 2.000"*) exit 1 ;;
  *"xstack="*) printf 'RIFF\004\000\000\000WEBP' ;;
  *) printf '\211PNG\r\n\032\nframe' ;;
esac
`)
	sourcePath := filepath.Join(directory, "source.mp4")
	if err := os.WriteFile(sourcePath, []byte("source"), 0o600); err != nil {
		t.Fatal(err)
	}
	source, err := os.Open(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	processor, err := New(Options{
		FFmpegPath:         ffmpeg,
		StoryboardTempRoot: filepath.Join(directory, "temp"),
		StoryboardTimeout:  time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = processor.Storyboard(
		context.Background(),
		source,
		media.FormatMP4,
		media.StoryboardRequest{
			TimestampsMS: []int64{1000, 2000, 3000, 4000},
			Columns:      4, Rows: 1, CellWidth: 320, CellHeight: 180,
		},
	)
	if err == nil {
		t.Fatal("failed planned frame unexpectedly produced a storyboard")
	}
}

func TestDurationAndPlaybackClassification(t *testing.T) {
	if value, err := durationMilliseconds("", "1.2345"); err != nil || value != 1235 {
		t.Fatalf("duration = %d, %v", value, err)
	}
	if _, err := durationMilliseconds("N/A", "-1"); err == nil {
		t.Fatal("invalid durations unexpectedly accepted")
	}
	if got := playbackStatus(media.FormatMKV, "h264"); got != media.PlaybackUnknown {
		t.Fatalf("MKV H.264 playback = %q", got)
	}
	if got := playbackStatus(media.FormatMKV, "ffv1"); got != media.PlaybackUnsupportedCodec {
		t.Fatalf("FFV1 playback = %q", got)
	}
}

func splitNonemptyLines(value string) []string {
	var result []string
	for _, line := range strings.Split(value, "\n") {
		if line != "" {
			result = append(result, line)
		}
	}
	return result
}

func writeExecutable(t *testing.T, filename, contents string) {
	t.Helper()
	if err := os.WriteFile(filename, []byte(contents), 0o700); err != nil {
		t.Fatal(err)
	}
}
