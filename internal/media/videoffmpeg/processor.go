// Package videoffmpeg implements the media video processor with ffprobe and
// FFmpeg. It never invokes a shell and never returns raw tool output.
package videoffmpeg

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"time"

	"github.com/HappyQuQu/foliopath/internal/media"
)

type Options struct {
	FFprobePath string
	FFmpegPath  string
	Timeout     time.Duration
}

type Processor struct {
	ffprobe string
	ffmpeg  string
	timeout time.Duration
}

func New(options Options) (*Processor, error) {
	if options.FFprobePath == "" {
		options.FFprobePath = "ffprobe"
	}
	if options.FFmpegPath == "" {
		options.FFmpegPath = "ffmpeg"
	}
	if options.Timeout == 0 {
		options.Timeout = media.DefaultProbeTimeout
	}
	if options.Timeout < 100*time.Millisecond || options.Timeout > time.Minute {
		return nil, errors.New("invalid FFmpeg processing timeout")
	}
	return &Processor{
		ffprobe: options.FFprobePath,
		ffmpeg:  options.FFmpegPath,
		timeout: options.Timeout,
	}, nil
}

type probeDocument struct {
	Streams []struct {
		CodecType string `json:"codec_type"`
		CodecName string `json:"codec_name"`
		Width     int    `json:"width"`
		Height    int    `json:"height"`
		Duration  string `json:"duration"`
	} `json:"streams"`
	Format struct {
		Duration string `json:"duration"`
	} `json:"format"`
}

func (processor *Processor) Process(
	ctx context.Context,
	source io.ReadSeeker,
	format media.Format,
) (media.ProcessingResult, error) {
	if format != media.FormatMP4 && format != media.FormatMOV && format != media.FormatMKV {
		return media.ProcessingResult{}, media.ErrUnsupportedMedia
	}
	file, ok := source.(*os.File)
	if !ok {
		return media.ProcessingResult{}, media.ErrProcessingFailed
	}
	info, err := file.Stat()
	if err != nil {
		return media.ProcessingResult{}, media.ErrProcessingFailed
	}
	if err := media.ValidateSourceSize(format, info.Size()); err != nil {
		return media.ProcessingResult{}, err
	}
	runCtx, cancel := context.WithTimeout(ctx, processor.timeout)
	defer cancel()

	document, err := processor.probe(runCtx, file)
	if err != nil {
		return media.ProcessingResult{}, err
	}
	stream, ok := firstVideoStream(document)
	if !ok || media.ValidateDimensions(stream.Width, stream.Height) != nil {
		return media.ProcessingResult{}, media.ErrInvalidMedia
	}
	durationMS, err := durationMilliseconds(stream.Duration, document.Format.Duration)
	if err != nil {
		return media.ProcessingResult{}, media.ErrInvalidMedia
	}
	poster, err := processor.poster(runCtx, file)
	if err != nil {
		return media.ProcessingResult{}, err
	}
	thumbnailWidth, thumbnailHeight := boundedDimensions(
		stream.Width, stream.Height,
		media.GridThumbnailWidth, media.GridThumbnailHeight,
	)
	result := media.ProcessingResult{
		Metadata: media.Metadata{
			Width: stream.Width, Height: stream.Height, DurationMS: &durationMS,
			PlaybackStatus: playbackStatus(format, stream.CodecName),
		},
		Thumbnail: media.Thumbnail{
			Bytes: poster, Width: thumbnailWidth, Height: thumbnailHeight,
		},
	}
	if err := media.ValidateProcessingResult(media.KindVideo, result); err != nil {
		return media.ProcessingResult{}, media.ErrProcessingFailed
	}
	return result, nil
}

func (processor *Processor) probe(ctx context.Context, source *os.File) (probeDocument, error) {
	output, err := processor.run(ctx, processor.ffprobe, source,
		"-nostdin",
		"-v", "error",
		"-threads", "1",
		"-print_format", "json",
		"-show_streams",
		"-show_format",
		"-i", inheritedFilePath(),
	)
	if err != nil {
		return probeDocument{}, classifyCommandError(ctx, err)
	}
	var document probeDocument
	if err := json.Unmarshal(output, &document); err != nil {
		return probeDocument{}, media.ErrInvalidMedia
	}
	return document, nil
}

func (processor *Processor) poster(ctx context.Context, source *os.File) ([]byte, error) {
	output, err := processor.run(ctx, processor.ffmpeg, source,
		"-nostdin",
		"-v", "error",
		"-threads", "1",
		"-filter_threads", "1",
		"-i", inheritedFilePath(),
		"-frames:v", "1",
		"-vf", fmt.Sprintf(
			"scale='min(%d,iw)':'min(%d,ih)':force_original_aspect_ratio=decrease",
			media.GridThumbnailWidth, media.GridThumbnailHeight,
		),
		"-an",
		"-f", "image2pipe",
		"-vcodec", "libwebp",
		"-q:v", "75",
		"pipe:1",
	)
	if err != nil {
		return nil, classifyCommandError(ctx, err)
	}
	if len(output) == 0 {
		return nil, media.ErrInvalidMedia
	}
	return output, nil
}

func (processor *Processor) run(
	ctx context.Context,
	binary string,
	source *os.File,
	args ...string,
) ([]byte, error) {
	if _, err := source.Seek(0, io.SeekStart); err != nil {
		return nil, media.ErrProcessingFailed
	}
	command := exec.CommandContext(ctx, binary, args...)
	configureCommandCancellation(command)
	command.ExtraFiles = []*os.File{source}
	var stdout cappedBuffer
	stdout.maximum = media.MaxToolOutputBytes
	var stderr cappedBuffer
	stderr.maximum = 64 << 10
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		if stdout.exceeded || stderr.exceeded {
			return nil, media.ErrProcessingFailed
		}
		return nil, err
	}
	if stdout.exceeded || stderr.exceeded {
		return nil, media.ErrProcessingFailed
	}
	return stdout.Bytes(), nil
}

type cappedBuffer struct {
	value    bytes.Buffer
	maximum  int
	exceeded bool
}

func (buffer *cappedBuffer) Write(value []byte) (int, error) {
	remaining := buffer.maximum - buffer.value.Len()
	if remaining <= 0 {
		buffer.exceeded = true
		return len(value), nil
	}
	if len(value) > remaining {
		_, _ = buffer.value.Write(value[:remaining])
		buffer.exceeded = true
		return len(value), nil
	}
	return buffer.value.Write(value)
}

func (buffer *cappedBuffer) Bytes() []byte {
	return buffer.value.Bytes()
}

func inheritedFilePath() string {
	if runtime.GOOS == "linux" {
		return "/proc/self/fd/3"
	}
	return "/dev/fd/3"
}

func classifyCommandError(ctx context.Context, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		return media.ErrInvalidMedia
	}
	return media.ErrProcessingFailed
}

func firstVideoStream(document probeDocument) (struct {
	CodecType string `json:"codec_type"`
	CodecName string `json:"codec_name"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Duration  string `json:"duration"`
}, bool) {
	for _, stream := range document.Streams {
		if stream.CodecType == "video" {
			return stream, true
		}
	}
	return struct {
		CodecType string `json:"codec_type"`
		CodecName string `json:"codec_name"`
		Width     int    `json:"width"`
		Height    int    `json:"height"`
		Duration  string `json:"duration"`
	}{}, false
}

func durationMilliseconds(values ...string) (int64, error) {
	for _, value := range values {
		if value == "" || value == "N/A" {
			continue
		}
		seconds, err := strconv.ParseFloat(value, 64)
		if err != nil || seconds < 0 || math.IsNaN(seconds) || math.IsInf(seconds, 0) {
			continue
		}
		return int64(math.Round(seconds * 1000)), nil
	}
	return 0, errors.New("video duration is unavailable")
}

func playbackStatus(format media.Format, codec string) media.PlaybackStatus {
	if codec == "h264" {
		if format == media.FormatMP4 || format == media.FormatMOV {
			return media.PlaybackPlayable
		}
		return media.PlaybackUnknown
	}
	return media.PlaybackUnsupportedCodec
}

func boundedDimensions(width, height, maximumWidth, maximumHeight int) (int, int) {
	scale := math.Min(
		float64(maximumWidth)/float64(width),
		float64(maximumHeight)/float64(height),
	)
	if scale > 1 {
		scale = 1
	}
	return max(1, int(math.Round(float64(width)*scale))),
		max(1, int(math.Round(float64(height)*scale)))
}
