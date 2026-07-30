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
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/HappyQuQu/foliopath/internal/media"
)

type Options struct {
	FFprobePath        string
	FFmpegPath         string
	Timeout            time.Duration
	StoryboardTimeout  time.Duration
	StoryboardTempRoot string
}

type Processor struct {
	ffprobe            string
	ffmpeg             string
	timeout            time.Duration
	storyboardTimeout  time.Duration
	storyboardTempRoot string
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
	if options.StoryboardTimeout == 0 {
		options.StoryboardTimeout = media.DefaultStoryboardTimeout
	}
	if options.StoryboardTimeout < 100*time.Millisecond ||
		options.StoryboardTimeout > media.DefaultStoryboardTimeout {
		return nil, errors.New("invalid FFmpeg storyboard timeout")
	}
	return &Processor{
		ffprobe:            options.FFprobePath,
		ffmpeg:             options.FFmpegPath,
		timeout:            options.Timeout,
		storyboardTimeout:  options.StoryboardTimeout,
		storyboardTempRoot: options.StoryboardTempRoot,
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
	if format != media.FormatMP4 &&
		format != media.FormatMOV &&
		format != media.FormatMKV &&
		format != media.FormatAVI {
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

func (processor *Processor) Storyboard(
	ctx context.Context,
	source io.ReadSeeker,
	format media.Format,
	request media.StoryboardRequest,
) (media.StoryboardResult, error) {
	if format != media.FormatMP4 &&
		format != media.FormatMOV &&
		format != media.FormatMKV &&
		format != media.FormatAVI {
		return media.StoryboardResult{}, media.ErrUnsupportedMedia
	}
	if media.ValidateStoryboardRequest(request) != nil ||
		processor.storyboardTempRoot == "" {
		return media.StoryboardResult{}, media.ErrProcessingFailed
	}
	file, ok := source.(*os.File)
	if !ok {
		return media.StoryboardResult{}, media.ErrProcessingFailed
	}
	info, err := file.Stat()
	if err != nil {
		return media.StoryboardResult{}, media.ErrProcessingFailed
	}
	if err := media.ValidateSourceSize(format, info.Size()); err != nil {
		return media.StoryboardResult{}, err
	}
	if err := os.MkdirAll(processor.storyboardTempRoot, 0o700); err != nil {
		return media.StoryboardResult{}, media.ErrProcessingFailed
	}
	tempDirectory, err := os.MkdirTemp(
		processor.storyboardTempRoot,
		"storyboard-*",
	)
	if err != nil {
		return media.StoryboardResult{}, media.ErrProcessingFailed
	}
	defer os.RemoveAll(tempDirectory)

	runCtx, cancel := context.WithTimeout(ctx, processor.storyboardTimeout)
	defer cancel()
	framePaths := make([]string, 0, len(request.TimestampsMS))
	totalFrameBytes := 0
	for index, timestampMS := range request.TimestampsMS {
		frame, err := processor.storyboardFrame(
			runCtx,
			file,
			timestampMS,
			request.CellWidth,
			request.CellHeight,
		)
		if err != nil {
			return media.StoryboardResult{}, err
		}
		totalFrameBytes += len(frame)
		if totalFrameBytes > media.StoryboardMaxTempBytes {
			return media.StoryboardResult{}, media.ErrProcessingFailed
		}
		framePath := filepath.Join(
			tempDirectory,
			fmt.Sprintf("frame-%02d.png", index),
		)
		if err := os.WriteFile(framePath, frame, 0o600); err != nil {
			return media.StoryboardResult{}, media.ErrProcessingFailed
		}
		framePaths = append(framePaths, framePath)
	}
	sprite, err := processor.composeStoryboard(runCtx, framePaths, request)
	if err != nil {
		return media.StoryboardResult{}, err
	}
	result := media.StoryboardResult{
		Bytes:      sprite,
		FrameCount: len(request.TimestampsMS),
		Columns:    request.Columns,
		Rows:       request.Rows,
		CellWidth:  request.CellWidth,
		CellHeight: request.CellHeight,
	}
	if media.ValidateStoryboardResult(request, result) != nil {
		return media.StoryboardResult{}, media.ErrProcessingFailed
	}
	return result, nil
}

func (processor *Processor) storyboardFrame(
	ctx context.Context,
	source *os.File,
	timestampMS int64,
	width, height int,
) ([]byte, error) {
	output, err := processor.runWithLimit(
		ctx,
		processor.ffmpeg,
		source,
		media.StoryboardMaxFrameBytes,
		"-nostdin",
		"-v", "error",
		"-threads", "1",
		"-filter_threads", "1",
		"-ss", strconv.FormatFloat(float64(timestampMS)/1000, 'f', 3, 64),
		"-i", inheritedFilePath(),
		"-frames:v", "1",
		"-vf", fmt.Sprintf(
			"scale=%d:%d:flags=lanczos,setsar=1",
			width,
			height,
		),
		"-an",
		"-f", "image2pipe",
		"-vcodec", "png",
		"pipe:1",
	)
	if err != nil {
		return nil, classifyCommandError(ctx, err)
	}
	pngSignature := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}
	if len(output) < len(pngSignature) ||
		!bytes.Equal(output[:len(pngSignature)], pngSignature) {
		return nil, media.ErrProcessingFailed
	}
	return output, nil
}

func (processor *Processor) composeStoryboard(
	ctx context.Context,
	framePaths []string,
	request media.StoryboardRequest,
) ([]byte, error) {
	args := []string{
		"-nostdin",
		"-v", "error",
		"-threads", "1",
		"-filter_threads", "1",
	}
	inputs := make([]string, 0, len(framePaths))
	positions := make([]string, 0, len(framePaths))
	for index, framePath := range framePaths {
		args = append(args, "-i", framePath)
		inputs = append(inputs, fmt.Sprintf("[%d:v]", index))
		positions = append(
			positions,
			fmt.Sprintf(
				"%d_%d",
				(index%request.Columns)*request.CellWidth,
				(index/request.Columns)*request.CellHeight,
			),
		)
	}
	filter := strings.Join(inputs, "") +
		fmt.Sprintf(
			"xstack=inputs=%d:layout=%s:fill=black[v]",
			len(framePaths),
			strings.Join(positions, "|"),
		)
	args = append(
		args,
		"-filter_complex", filter,
		"-map", "[v]",
		"-frames:v", "1",
		"-an",
		"-f", "image2pipe",
		"-vcodec", "libwebp",
		"-q:v", "75",
		"pipe:1",
	)
	output, err := processor.runCommand(
		ctx,
		processor.ffmpeg,
		nil,
		media.StoryboardMaxOutputBytes,
		args...,
	)
	if err != nil {
		return nil, classifyCommandError(ctx, err)
	}
	return output, nil
}

func (processor *Processor) probe(ctx context.Context, source *os.File) (probeDocument, error) {
	output, err := processor.run(ctx, processor.ffprobe, source,
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
	return processor.runWithLimit(
		ctx,
		binary,
		source,
		media.MaxToolOutputBytes,
		args...,
	)
}

func (processor *Processor) runWithLimit(
	ctx context.Context,
	binary string,
	source *os.File,
	maximum int,
	args ...string,
) ([]byte, error) {
	if _, err := source.Seek(0, io.SeekStart); err != nil {
		return nil, media.ErrProcessingFailed
	}
	return processor.runCommand(
		ctx,
		binary,
		[]*os.File{source},
		maximum,
		args...,
	)
}

func (processor *Processor) runCommand(
	ctx context.Context,
	binary string,
	extraFiles []*os.File,
	maximum int,
	args ...string,
) ([]byte, error) {
	command := exec.CommandContext(ctx, binary, args...)
	configureCommandCancellation(command)
	command.ExtraFiles = extraFiles
	var stdout cappedBuffer
	stdout.maximum = maximum
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
