package videoffmpeg

import (
	"bytes"
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
printf '%s' '{"streams":[{"codec_type":"video","codec_name":"h264","width":96,"height":64,"duration":"1.250"}],"format":{"format_name":"mov,mp4,m4a,3gp,3g2,mj2","duration":"1.250"}}'
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
case "$*" in
  *"-map 0:v:0"*) ;;
  *) echo "missing explicit first-video-stream map" >&2; exit 2 ;;
esac
printf 'RIFF\004\000\000\000WEBP'
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
	if !errors.Is(err, media.ErrProcessingFailed) ||
		errors.Is(err, exec.ErrNotFound) {
		t.Fatalf("safe failure = %v", err)
	}
}

func TestVideoProcessorDerivesMPEGTSMismatchWithoutClaimingPlayback(t *testing.T) {
	directory := t.TempDir()
	probe := filepath.Join(directory, "ffprobe")
	ffmpeg := filepath.Join(directory, "ffmpeg")
	writeExecutable(t, probe, `#!/bin/sh
printf '%s' '{"streams":[{"codec_type":"video","codec_name":"h264","width":1920,"height":1080,"duration":"7290"}],"format":{"format_name":"mpegts","duration":"7290"}}'
`)
	writeExecutable(t, ffmpeg, `#!/bin/sh
case "$*" in
  *"-map 0:v:0"*) printf 'RIFF\004\000\000\000WEBP' ;;
  *) exit 2 ;;
esac
`)
	sourcePath := filepath.Join(directory, "mpegts-disguised-as-mp4.mp4")
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
	result, err := processor.Process(context.Background(), source, media.FormatMP4)
	if err != nil {
		t.Fatal(err)
	}
	if result.Metadata.PlaybackStatus != media.PlaybackUnknown ||
		result.Metadata.DurationMS == nil || *result.Metadata.DurationMS != 7_290_000 {
		t.Fatalf("MPEG-TS mismatch metadata = %#v", result.Metadata)
	}
}

func TestVideoProcessorRejectsDisallowedActualContainers(t *testing.T) {
	tests := []struct {
		name       string
		formatName string
	}{
		{name: "known container mismatch", formatName: "matroska,webm"},
		{name: "unknown container", formatName: "image2"},
		{name: "conflicting container aliases", formatName: "mov,mpegts"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			directory := t.TempDir()
			probe := filepath.Join(directory, "ffprobe")
			ffmpeg := filepath.Join(directory, "ffmpeg")
			posterCalled := filepath.Join(directory, "poster-called")
			writeExecutable(t, probe, "#!/bin/sh\nprintf '%s' '{\"streams\":[{\"codec_type\":\"video\",\"codec_name\":\"h264\",\"width\":96,\"height\":64}],\"format\":{\"format_name\":\""+test.formatName+"\"}}'\n")
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
			_, err = processor.Process(context.Background(), source, media.FormatMP4)
			diagnostic, ok := media.DiagnoseFailure(err)
			if !errors.Is(err, media.ErrUnsupportedMedia) || !ok ||
				diagnostic.Stage != media.StageProbe ||
				diagnostic.Reason != media.ReasonContainerMismatch ||
				diagnostic.Tool != "ffprobe" {
				t.Fatalf("container mismatch = %v, %#v, %v", err, diagnostic, ok)
			}
			if _, statErr := os.Stat(posterCalled); !errors.Is(statErr, os.ErrNotExist) {
				t.Fatalf("poster decoder was invoked: %v", statErr)
			}
		})
	}
}

func TestClassifyFFmpegReasonReturnsStableSafeReasons(t *testing.T) {
	tests := map[string]media.FailureReason{
		"moov atom not found":                                    media.ReasonMissingMoovAtom,
		"Invalid data found when processing input":               media.ReasonInvalidData,
		"Could not find codec parameters for stream 0":           media.ReasonInvalidData,
		"Error while decoding stream #0:0":                       media.ReasonDecodeFailed,
		"Decoder h265 not found":                                 media.ReasonDecoderUnavailable,
		"Failed to get pixel format; Function not implemented":   media.ReasonDecoderUnavailable,
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

func TestVideoProcessorRejectsOnlyExtremelyLargeSparseFileBeforeTools(t *testing.T) {
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
	); !errors.Is(err, media.ErrSourceTooLarge) {
		t.Fatalf("oversized video error = %v", err)
	}
	if _, err := os.Stat(called); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("media tool was invoked: %v", err)
	}
}

func TestVideoProcessorHandsFilesAboveFourGiBToMediaTools(t *testing.T) {
	directory := t.TempDir()
	probe := filepath.Join(directory, "ffprobe")
	ffmpeg := filepath.Join(directory, "ffmpeg")
	writeExecutable(t, probe, `#!/bin/sh
printf '%s' '{"streams":[{"codec_type":"video","codec_name":"hevc","width":1920,"height":1080,"duration":"N/A"}],"format":{"format_name":"mov,mp4,m4a,3gp,3g2,mj2","duration":"N/A"}}'
`)
	writeExecutable(t, ffmpeg, "#!/bin/sh\nprintf 'RIFF\\004\\000\\000\\000WEBP'\n")
	sourcePath := filepath.Join(directory, "large.mp4")
	source, err := os.Create(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	if err := source.Truncate((int64(4) << 30) + 1); err != nil {
		t.Fatal(err)
	}
	processor, err := New(Options{
		FFprobePath: probe, FFmpegPath: ffmpeg, Timeout: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := processor.Process(context.Background(), source, media.FormatMP4)
	if err != nil {
		t.Fatal(err)
	}
	if result.Metadata.DurationMS != nil ||
		result.Metadata.PlaybackStatus != media.PlaybackUnknown {
		t.Fatalf("large video metadata = %#v", result.Metadata)
	}
}

func TestVideoProcessorGivesProbeAndPosterSeparateTimeouts(t *testing.T) {
	directory := t.TempDir()
	probe := filepath.Join(directory, "ffprobe")
	ffmpeg := filepath.Join(directory, "ffmpeg")
	writeExecutable(t, probe, `#!/bin/sh
sleep 0.6
printf '%s' '{"streams":[{"codec_type":"video","codec_name":"h264","width":96,"height":64,"duration":"1"}],"format":{"format_name":"mov,mp4,m4a,3gp,3g2,mj2","duration":"1"}}'
`)
	writeExecutable(t, ffmpeg, "#!/bin/sh\nsleep 0.6\nprintf 'RIFF\\004\\000\\000\\000WEBP'\n")
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
	if _, err := processor.Process(context.Background(), source, media.FormatMP4); err != nil {
		t.Fatalf("independent probe and poster budgets failed: %v", err)
	}
}

func TestVideoProcessorSeparatesDamagedMediaFromTransientToolFailure(t *testing.T) {
	directory := t.TempDir()
	tool := filepath.Join(directory, "tool")
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
		FFprobePath: tool, FFmpegPath: tool, Timeout: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}

	writeExecutable(t, tool, "#!/bin/sh\necho 'moov atom not found' >&2\nexit 1\n")
	_, err = processor.Process(context.Background(), source, media.FormatMP4)
	diagnostic, ok := media.DiagnoseFailure(err)
	if !errors.Is(err, media.ErrInvalidMedia) || !ok ||
		diagnostic.Reason != media.ReasonMissingMoovAtom {
		t.Fatalf("missing moov classification = %v, %#v, %v", err, diagnostic, ok)
	}

	writeExecutable(t, tool, "#!/bin/sh\necho 'resource temporarily unavailable' >&2\nexit 1\n")
	_, err = processor.Process(context.Background(), source, media.FormatMP4)
	diagnostic, ok = media.DiagnoseFailure(err)
	if !errors.Is(err, media.ErrProcessingFailed) ||
		errors.Is(err, media.ErrInvalidMedia) || !ok ||
		diagnostic.Reason != media.ReasonToolFailed {
		t.Fatalf("transient tool classification = %v, %#v, %v", err, diagnostic, ok)
	}
}

func TestVideoProcessorRejectsHostileMetadataBeforePosterDecode(t *testing.T) {
	directory := t.TempDir()
	probe := filepath.Join(directory, "ffprobe")
	ffmpeg := filepath.Join(directory, "ffmpeg")
	posterCalled := filepath.Join(directory, "poster-called")
	writeExecutable(t, probe, `#!/bin/sh
printf '%s' '{"streams":[{"codec_type":"video","codec_name":"h264","width":20000,"height":20000,"duration":"1"}],"format":{"format_name":"mov,mp4,m4a,3gp,3g2,mj2","duration":"1"}}'
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
	); !errors.Is(err, media.ErrSourceTooLarge) {
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

func TestVideoProcessorKeepsDecoderReasonWhenStderrIsTruncated(t *testing.T) {
	directory := t.TempDir()
	tool := filepath.Join(directory, "ffmpeg")
	writeExecutable(t, tool, `#!/bin/sh
i=0
while [ "$i" -lt 4000 ]; do
  echo 'AV1 decoder failed to get pixel format: Function not implemented' >&2
  i=$((i + 1))
done
exit 1
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
		FFprobePath: tool, FFmpegPath: tool, Timeout: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = processor.run(context.Background(), tool, source)
	classified := classifyCommandError(
		context.Background(), err, media.StagePoster, "ffmpeg",
	)
	diagnostic, ok := media.DiagnoseFailure(classified)
	if !errors.Is(classified, media.ErrUnsupportedMedia) || !ok ||
		diagnostic.Reason != media.ReasonDecoderUnavailable {
		t.Fatalf("truncated stderr classification = %v, %#v, %v", classified, diagnostic, ok)
	}
}

func TestSuccessfulToolOutputSurvivesTruncatedWarnings(t *testing.T) {
	directory := t.TempDir()
	tool := filepath.Join(directory, "ffmpeg")
	writeExecutable(t, tool, `#!/bin/sh
i=0
while [ "$i" -lt 4000 ]; do
  echo 'recoverable damaged-frame warning' >&2
  i=$((i + 1))
done
printf 'usable-output'
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
		FFprobePath: tool, FFmpegPath: tool, Timeout: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	output, err := processor.run(context.Background(), tool, source)
	if err != nil || string(output) != "usable-output" {
		t.Fatalf("successful output = %q, %v", output, err)
	}
}

func TestPosterRejectsInvalidOutputFromSuccessfulTool(t *testing.T) {
	directory := t.TempDir()
	tool := filepath.Join(directory, "ffmpeg")
	writeExecutable(t, tool, "#!/bin/sh\nprintf 'not-a-webp'\n")
	sourcePath := filepath.Join(directory, "source.mp4")
	if err := os.WriteFile(sourcePath, []byte("source"), 0o600); err != nil {
		t.Fatal(err)
	}
	source, err := os.Open(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	processor, err := New(Options{FFmpegPath: tool, Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	_, err = processor.poster(context.Background(), source)
	diagnostic, ok := media.DiagnoseFailure(err)
	if !errors.Is(err, media.ErrProcessingFailed) || !ok ||
		diagnostic.Stage != media.StageValidation ||
		diagnostic.Reason != media.ReasonToolFailed {
		t.Fatalf("invalid successful poster output = %v, %#v, %v", err, diagnostic, ok)
	}
}

func TestStoryboardFrameUsesBoundedNeighborFallback(t *testing.T) {
	directory := t.TempDir()
	tool := filepath.Join(directory, "ffmpeg")
	calls := filepath.Join(directory, "calls")
	writeExecutable(t, tool, `#!/bin/sh
printf '%s\n' "$*" >> "`+calls+`"
case "$*" in
  *"-ss 1.000"*) printf '\211PNG\r\n\032\nneighbor' ;;
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
	processor, err := New(Options{FFmpegPath: tool, StoryboardTimeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	frame, err := processor.storyboardFrame(
		context.Background(), source, 2_000, 1_000, 1_000, 320, 180,
	)
	if err != nil || !bytes.HasPrefix(frame, []byte{'\x89', 'P', 'N', 'G'}) {
		t.Fatalf("neighbor frame = %q, %v", frame, err)
	}
	callBytes, err := os.ReadFile(calls)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(splitNonemptyLines(string(callBytes))); got != 2 {
		t.Fatalf("FFmpeg calls = %d, want exact plus one neighbor", got)
	}
}

func TestStoryboardFrameStopsAfterBoundedMissingFrameFallback(t *testing.T) {
	directory := t.TempDir()
	tool := filepath.Join(directory, "ffmpeg")
	calls := filepath.Join(directory, "calls")
	writeExecutable(t, tool, "#!/bin/sh\nprintf '%s\\n' \"$*\" >> \""+calls+"\"\n")
	sourcePath := filepath.Join(directory, "source.mp4")
	if err := os.WriteFile(sourcePath, []byte("source"), 0o600); err != nil {
		t.Fatal(err)
	}
	source, err := os.Open(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	processor, err := New(Options{FFmpegPath: tool, StoryboardTimeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	_, err = processor.storyboardFrame(
		context.Background(), source, 2_000, 1_000, 1_000, 320, 180,
	)
	diagnostic, ok := media.DiagnoseFailure(err)
	if !errors.Is(err, media.ErrFrameUnavailable) || !ok ||
		diagnostic.Reason != media.ReasonNoFrame {
		t.Fatalf("missing-frame failure = %v, %#v, %v", err, diagnostic, ok)
	}
	callBytes, readErr := os.ReadFile(calls)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if got := len(splitNonemptyLines(string(callBytes))); got != 3 {
		t.Fatalf("FFmpeg calls = %d, want bounded three attempts", got)
	}
}

func TestStoryboardNeighborFallbackPreservesShortVideoSampleOrder(t *testing.T) {
	for _, test := range []struct {
		name       string
		timestamps []int64
	}{
		{
			name: "five second ten frame plan",
			timestamps: []int64{
				454, 909, 1_363, 1_818, 2_272,
				2_727, 3_181, 3_636, 4_090, 4_545,
			},
		},
		{name: "two second four frame plan", timestamps: []int64{400, 800, 1_200, 1_600}},
	} {
		t.Run(test.name, func(t *testing.T) {
			previousUpper := int64(0)
			for index, timestampMS := range test.timestamps {
				backwardMS, forwardMS := storyboardNeighborOffsets(
					test.timestamps,
					index,
				)
				if backwardMS < 0 || backwardMS > 1_000 ||
					forwardMS < 0 || forwardMS > 1_000 {
					t.Fatalf("offsets[%d] = -%d/+%d", index, backwardMS, forwardMS)
				}
				lower := timestampMS - backwardMS
				upper := timestampMS + forwardMS
				if lower <= 0 || lower > timestampMS || upper < timestampMS {
					t.Fatalf("candidate range[%d] = %d..%d around %d", index, lower, upper, timestampMS)
				}
				if index > 0 && lower <= previousUpper {
					t.Fatalf("candidate ranges overlap at %d: previous upper %d, lower %d", index, previousUpper, lower)
				}
				previousUpper = upper
			}
		})
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
	callLines := splitNonemptyLines(string(callBytes))
	if got := len(callLines); got != 5 {
		t.Fatalf("FFmpeg calls = %d, want four seeks plus one compose", got)
	}
	for index, call := range callLines[:4] {
		if !strings.Contains(call, "-map 0:v:0") {
			t.Fatalf("frame extraction call %d does not map the first video stream: %q", index, call)
		}
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

func TestStoryboardHonorsAdaptiveRequestTimeout(t *testing.T) {
	directory := t.TempDir()
	ffmpeg := filepath.Join(directory, "ffmpeg")
	writeExecutable(t, ffmpeg, "#!/bin/sh\nsleep 0.4\n")
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
		StoryboardTimeout:  5 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	_, err = processor.Storyboard(
		context.Background(),
		source,
		media.FormatMP4,
		media.StoryboardRequest{
			TimestampsMS: []int64{1000, 2000, 3000, 4000},
			Columns:      4,
			Rows:         1,
			CellWidth:    320,
			CellHeight:   180,
			Timeout:      100 * time.Millisecond,
		},
	)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("storyboard timeout error = %v", err)
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("adaptive timeout took %s", elapsed)
	}
}

func TestDurationAndPlaybackClassification(t *testing.T) {
	if value := durationMilliseconds("", "1.2345"); value == nil || *value != 1235 {
		t.Fatalf("duration = %v", value)
	}
	if value := durationMilliseconds("N/A", "-1"); value != nil {
		t.Fatalf("invalid duration = %v, want unknown", *value)
	}
	if got := playbackStatus(
		media.FormatMP4, videoContainerISOBaseMedia, "h264",
	); got != media.PlaybackPlayable {
		t.Fatalf("MP4 H.264 playback = %q", got)
	}
	if got := playbackStatus(
		media.FormatMKV, videoContainerMatroska, "h264",
	); got != media.PlaybackUnknown {
		t.Fatalf("MKV H.264 playback = %q", got)
	}
	if got := playbackStatus(
		media.FormatMKV, videoContainerMatroska, "ffv1",
	); got != media.PlaybackUnknown {
		t.Fatalf("FFV1 playback = %q", got)
	}
	if got := playbackStatus(
		media.FormatMP4, videoContainerISOBaseMedia, "hevc",
	); got != media.PlaybackUnknown {
		t.Fatalf("MP4 HEVC playback = %q", got)
	}
	if got := playbackStatus(
		media.FormatMP4, videoContainerMPEGTS, "h264",
	); got != media.PlaybackUnknown {
		t.Fatalf("MPEG-TS disguised as MP4 playback = %q", got)
	}
}

func TestCanonicalVideoContainer(t *testing.T) {
	tests := []struct {
		name        string
		formatNames string
		want        videoContainer
		ok          bool
	}{
		{name: "ISO base media aliases", formatNames: "mov,mp4,m4a,3gp,3g2,mj2", want: videoContainerISOBaseMedia, ok: true},
		{name: "matroska aliases", formatNames: "matroska,webm", want: videoContainerMatroska, ok: true},
		{name: "AVI", formatNames: "avi", want: videoContainerAVI, ok: true},
		{name: "MPEG-TS", formatNames: "mpegts", want: videoContainerMPEGTS, ok: true},
		{name: "empty", formatNames: ""},
		{name: "unknown", formatNames: "image2"},
		{name: "mixed families", formatNames: "mov,mpegts"},
		{name: "known and unknown", formatNames: "mov,unknown"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := canonicalVideoContainer(test.formatNames)
			if got != test.want || ok != test.ok {
				t.Fatalf("canonicalVideoContainer(%q) = %q, %v, want %q, %v", test.formatNames, got, ok, test.want, test.ok)
			}
		})
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
