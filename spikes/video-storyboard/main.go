package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	cellMaxWidth  = 320
	cellMaxHeight = 320
	maxOutput     = 8 << 20
)

type fixture struct {
	Name        string
	DurationS   float64
	FrameRate   int
	Width       int
	Height      int
	GOP         int
	RunBaseline bool
}

type commandMetric struct {
	ElapsedMS int64 `json:"elapsedMs"`
	UserMS    int64 `json:"userMs"`
	SystemMS  int64 `json:"systemMs"`
}

func (metric *commandMetric) add(value commandMetric) {
	metric.ElapsedMS += value.ElapsedMS
	metric.UserMS += value.UserMS
	metric.SystemMS += value.SystemMS
}

type fixtureResult struct {
	Name               string         `json:"name"`
	DurationS          float64        `json:"durationSeconds"`
	SourceBytes        int64          `json:"sourceBytes"`
	FrameCount         int            `json:"frameCount"`
	TimestampsS        []float64      `json:"timestampsSeconds"`
	Columns            int            `json:"columns"`
	Rows               int            `json:"rows"`
	SpriteWidth        int            `json:"spriteWidth"`
	SpriteHeight       int            `json:"spriteHeight"`
	SpriteBytes        int64          `json:"spriteBytes"`
	Generate           commandMetric  `json:"generate"`
	FastSeekStoryboard commandMetric  `json:"fastSeekStoryboard"`
	FullDecodeBaseline *commandMetric `json:"fullDecodeBaseline,omitempty"`
}

type report struct {
	GeneratedAt     string          `json:"generatedAt"`
	OS              string          `json:"os"`
	Architecture    string          `json:"architecture"`
	FFmpeg          string          `json:"ffmpeg"`
	FFprobe         string          `json:"ffprobe"`
	SpriteEncoder   string          `json:"spriteEncoder"`
	Fixtures        []fixtureResult `json:"fixtures"`
	CorruptRejected bool            `json:"corruptRejected"`
}

type probeStream struct {
	CodecName string `json:"codec_name"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
}

type probeDocument struct {
	Streams []probeStream `json:"streams"`
}

func main() {
	ffmpeg := flag.String("ffmpeg", "ffmpeg", "FFmpeg executable")
	ffprobe := flag.String("ffprobe", "ffprobe", "ffprobe executable")
	flag.Parse()

	if err := run(*ffmpeg, *ffprobe); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ffmpeg, ffprobe string) error {
	ffmpegPath, err := exec.LookPath(ffmpeg)
	if err != nil {
		return fmt.Errorf("find ffmpeg: %w", err)
	}
	ffprobePath, err := exec.LookPath(ffprobe)
	if err != nil {
		return fmt.Errorf("find ffprobe: %w", err)
	}
	useFFmpegWebP := hasEncoder(ffmpegPath, "libwebp")
	cwebpPath := ""
	if !useFFmpegWebP {
		cwebpPath, err = exec.LookPath("cwebp")
		if err != nil {
			return errors.New("ffmpeg has no libwebp encoder and cwebp is unavailable")
		}
	}

	work, err := os.MkdirTemp("", "foliopath-storyboard-spike-*")
	if err != nil {
		return fmt.Errorf("create temporary directory: %w", err)
	}
	defer os.RemoveAll(work)

	specs := []fixture{
		{Name: "two-seconds", DurationS: 2, FrameRate: 10, Width: 320, Height: 180, GOP: 20},
		{Name: "four-seconds", DurationS: 4, FrameRate: 10, Width: 320, Height: 180, GOP: 40},
		{Name: "ten-seconds", DurationS: 10, FrameRate: 10, Width: 320, Height: 180, GOP: 60},
		{Name: "ten-minutes", DurationS: 600, FrameRate: 15, Width: 320, Height: 180, GOP: 150, RunBaseline: true},
		{Name: "two-hours", DurationS: 7200, FrameRate: 5, Width: 160, Height: 90, GOP: 50, RunBaseline: true},
	}

	result := report{
		GeneratedAt:  time.Now().UTC().Format(time.RFC3339),
		OS:           runtime.GOOS,
		Architecture: runtime.GOARCH,
		FFmpeg:       firstVersionLine(ffmpegPath),
		FFprobe:      firstVersionLine(ffprobePath),
	}
	if useFFmpegWebP {
		result.SpriteEncoder = "ffmpeg/libwebp"
	} else {
		result.SpriteEncoder = "ffmpeg/png + cwebp"
	}

	for _, spec := range specs {
		item, err := exerciseFixture(
			work, ffmpegPath, ffprobePath, cwebpPath, useFFmpegWebP, spec,
		)
		if err != nil {
			return fmt.Errorf("%s: %w", spec.Name, err)
		}
		result.Fixtures = append(result.Fixtures, item)
	}

	corrupt := filepath.Join(work, "corrupt.mp4")
	if err := os.WriteFile(corrupt, []byte("not a video"), 0o600); err != nil {
		return fmt.Errorf("write corrupt fixture: %w", err)
	}
	_, err = execute(ffmpegPath,
		"-nostdin", "-hide_banner", "-v", "error", "-threads", "1",
		"-ss", "1", "-i", corrupt, "-frames:v", "1", "-f", "null", "-",
	)
	result.CorruptRejected = err != nil
	if !result.CorruptRejected {
		return errors.New("corrupt video was not rejected")
	}

	output, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("encode report: %w", err)
	}
	fmt.Println(string(output))
	return nil
}

func exerciseFixture(
	work string,
	ffmpeg string,
	ffprobe string,
	cwebp string,
	useFFmpegWebP bool,
	spec fixture,
) (fixtureResult, error) {
	directory := filepath.Join(work, spec.Name)
	if err := os.Mkdir(directory, 0o700); err != nil {
		return fixtureResult{}, fmt.Errorf("create fixture directory: %w", err)
	}
	source := filepath.Join(directory, "source.mp4")
	generate, err := execute(ffmpeg,
		"-nostdin", "-hide_banner", "-v", "error",
		"-f", "lavfi",
		"-i", fmt.Sprintf("testsrc2=size=%dx%d:rate=%d", spec.Width, spec.Height, spec.FrameRate),
		"-t", formatSeconds(spec.DurationS),
		"-threads", "1",
		"-c:v", "mpeg4",
		"-q:v", "8",
		"-g", strconv.Itoa(spec.GOP),
		"-an",
		"-y", source,
	)
	if err != nil {
		return fixtureResult{}, fmt.Errorf("generate source: %w", err)
	}
	sourceInfo, err := os.Stat(source)
	if err != nil {
		return fixtureResult{}, fmt.Errorf("stat source: %w", err)
	}

	count := frameCount(spec.DurationS)
	timestamps := sampleTimestamps(spec.DurationS, count)
	columns := min(5, count)
	rows := int(math.Ceil(float64(count) / float64(columns)))
	fastMetric := commandMetric{}
	for index, timestamp := range timestamps {
		frame := filepath.Join(directory, fmt.Sprintf("frame-%02d.png", index))
		metric, err := execute(ffmpeg,
			"-nostdin", "-hide_banner", "-v", "error",
			"-threads", "1",
			"-filter_threads", "1",
			"-ss", formatSeconds(timestamp),
			"-i", source,
			"-frames:v", "1",
			"-vf", fmt.Sprintf(
				"scale='min(%d,iw)':'min(%d,ih)':force_original_aspect_ratio=decrease",
				cellMaxWidth, cellMaxHeight,
			),
			"-an",
			"-f", "image2",
			"-vcodec", "png",
			"-y", frame,
		)
		if err != nil {
			return fixtureResult{}, fmt.Errorf("extract frame %d: %w", index, err)
		}
		fastMetric.add(metric)
	}

	sprite := filepath.Join(directory, "sprite.webp")
	composeOutput := sprite
	composeCodec := []string{"-c:v", "libwebp", "-q:v", "75"}
	if !useFFmpegWebP {
		composeOutput = filepath.Join(directory, "sprite.png")
		composeCodec = []string{"-c:v", "png"}
	}
	composeArguments := []string{
		"-nostdin", "-hide_banner", "-v", "error",
		"-threads", "1",
		"-filter_threads", "1",
		"-framerate", "1",
		"-start_number", "0",
		"-i", filepath.Join(directory, "frame-%02d.png"),
		"-frames:v", "1",
		"-vf", fmt.Sprintf("tile=%dx%d", columns, rows),
		"-an",
	}
	composeArguments = append(composeArguments, composeCodec...)
	composeArguments = append(composeArguments, "-y", composeOutput)
	compose, err := execute(ffmpeg, composeArguments...)
	if err != nil {
		return fixtureResult{}, fmt.Errorf("compose sprite: %w", err)
	}
	fastMetric.add(compose)
	if !useFFmpegWebP {
		encode, err := execute(cwebp, "-quiet", "-q", "75", composeOutput, "-o", sprite)
		if err != nil {
			return fixtureResult{}, fmt.Errorf("encode sprite WebP: %w", err)
		}
		fastMetric.add(encode)
	}

	spriteInfo, err := os.Stat(sprite)
	if err != nil {
		return fixtureResult{}, fmt.Errorf("stat sprite: %w", err)
	}
	stream, err := probeFirstStream(ffprobe, sprite)
	if err != nil {
		return fixtureResult{}, fmt.Errorf("probe sprite: %w", err)
	}
	if stream.CodecName != "webp" || stream.Width <= 0 || stream.Height <= 0 {
		return fixtureResult{}, fmt.Errorf("unexpected sprite stream: %+v", stream)
	}

	item := fixtureResult{
		Name: spec.Name, DurationS: spec.DurationS, SourceBytes: sourceInfo.Size(),
		FrameCount: count, TimestampsS: timestamps, Columns: columns, Rows: rows,
		SpriteWidth: stream.Width, SpriteHeight: stream.Height, SpriteBytes: spriteInfo.Size(),
		Generate: generate, FastSeekStoryboard: fastMetric,
	}

	if spec.RunBaseline {
		baseline := filepath.Join(directory, "full-decode.webp")
		baselineOutput := baseline
		baselineCodec := []string{"-c:v", "libwebp", "-q:v", "75"}
		if !useFFmpegWebP {
			baselineOutput = filepath.Join(directory, "full-decode.png")
			baselineCodec = []string{"-c:v", "png"}
		}
		baselineArguments := []string{
			"-nostdin", "-hide_banner", "-v", "error",
			"-threads", "1",
			"-filter_threads", "1",
			"-i", source,
			"-frames:v", "1",
			"-vf", fmt.Sprintf(
				"fps=%d/%s,scale='min(%d,iw)':'min(%d,ih)':force_original_aspect_ratio=decrease,tile=%dx%d",
				count, formatSeconds(spec.DurationS), cellMaxWidth, cellMaxHeight, columns, rows,
			),
			"-an",
		}
		baselineArguments = append(baselineArguments, baselineCodec...)
		baselineArguments = append(baselineArguments, "-y", baselineOutput)
		metric, err := execute(ffmpeg, baselineArguments...)
		if err != nil {
			return fixtureResult{}, fmt.Errorf("full-decode baseline: %w", err)
		}
		if !useFFmpegWebP {
			encode, err := execute(cwebp, "-quiet", "-q", "75", baselineOutput, "-o", baseline)
			if err != nil {
				return fixtureResult{}, fmt.Errorf("encode full-decode baseline WebP: %w", err)
			}
			metric.add(encode)
		}
		if _, err := probeFirstStream(ffprobe, baseline); err != nil {
			return fixtureResult{}, fmt.Errorf("probe full-decode baseline: %w", err)
		}
		item.FullDecodeBaseline = &metric
	}
	return item, nil
}

func frameCount(durationS float64) int {
	if durationS < 2 {
		return 0
	}
	if durationS >= 10 {
		return 10
	}
	return max(4, min(9, int(math.Floor(durationS))))
}

func sampleTimestamps(durationS float64, count int) []float64 {
	if count <= 0 {
		return nil
	}
	values := make([]float64, count)
	for index := range values {
		values[index] = durationS * (0.05 + 0.90*(float64(index)+0.5)/float64(count))
		values[index] = math.Round(values[index]*1000) / 1000
	}
	return values
}

func execute(name string, arguments ...string) (commandMetric, error) {
	started := time.Now()
	command := exec.Command(name, arguments...)
	var stderr bytes.Buffer
	command.Stderr = &cappedWriter{buffer: &stderr, maximum: 64 << 10}
	if err := command.Run(); err != nil {
		return commandMetric{}, fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
	}
	state := command.ProcessState
	return commandMetric{
		ElapsedMS: time.Since(started).Milliseconds(),
		UserMS:    state.UserTime().Milliseconds(),
		SystemMS:  state.SystemTime().Milliseconds(),
	}, nil
}

func probeFirstStream(ffprobe string, source string) (probeStream, error) {
	command := exec.Command(ffprobe,
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=codec_name,width,height",
		"-of", "json",
		source,
	)
	var stdout bytes.Buffer
	command.Stdout = &cappedWriter{buffer: &stdout, maximum: maxOutput}
	var stderr bytes.Buffer
	command.Stderr = &cappedWriter{buffer: &stderr, maximum: 64 << 10}
	if err := command.Run(); err != nil {
		return probeStream{}, fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
	}
	var document probeDocument
	if err := json.Unmarshal(stdout.Bytes(), &document); err != nil {
		return probeStream{}, fmt.Errorf("decode ffprobe JSON: %w", err)
	}
	if len(document.Streams) != 1 {
		return probeStream{}, errors.New("expected exactly one video stream")
	}
	return document.Streams[0], nil
}

func firstVersionLine(binary string) string {
	command := exec.Command(binary, "-version")
	output, err := command.Output()
	if err != nil {
		return filepath.Base(binary)
	}
	line, _, _ := strings.Cut(string(output), "\n")
	return strings.TrimSpace(line)
}

func hasEncoder(ffmpeg string, encoder string) bool {
	command := exec.Command(ffmpeg, "-hide_banner", "-encoders")
	output, err := command.CombinedOutput()
	return err == nil && bytes.Contains(output, []byte(encoder))
}

func formatSeconds(value float64) string {
	return strconv.FormatFloat(value, 'f', 3, 64)
}

type cappedWriter struct {
	buffer  *bytes.Buffer
	maximum int
}

func (writer *cappedWriter) Write(value []byte) (int, error) {
	remaining := writer.maximum - writer.buffer.Len()
	if remaining > 0 {
		if len(value) > remaining {
			_, _ = writer.buffer.Write(value[:remaining])
		} else {
			_, _ = writer.buffer.Write(value)
		}
	}
	return len(value), nil
}
