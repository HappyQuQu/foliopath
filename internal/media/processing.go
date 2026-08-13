package media

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"
)

const (
	GridThumbnailWidth           = 512
	GridThumbnailHeight          = 512
	GridWebPQuality              = 82
	MaxToolOutputBytes           = 8 << 20
	MaxImageSourceBytes          = 256 << 20
	MaxVideoSourceBytes          = int64(1) << 40
	MaxDecodedPixels             = 100_000_000
	MaxJPEGSourcePixels          = 180_000_000
	MaxMediaDimension            = 32_768
	DefaultProbeTimeout          = 60 * time.Second
	StoryboardMinFrames          = 4
	StoryboardMaxFrames          = 10
	StoryboardMaxColumns         = 5
	StoryboardMaxRows            = 2
	StoryboardMaxCellDimension   = 320
	StoryboardMaxPixels          = 1_024_000
	StoryboardMaxFrameBytes      = 1 << 20
	StoryboardMaxTempBytes       = 10 << 20
	StoryboardMaxOutputBytes     = 8 << 20
	DefaultStoryboardTimeout     = 45 * time.Second
	LargeStoryboardTimeout       = 4 * time.Minute
	MaxStoryboardAttemptTimeout  = LargeStoryboardTimeout
	MaxStoryboardFallbackTimeout = 2 * time.Minute
	MaxStoryboardTotalTimeout    = 6 * time.Minute
	StoryboardLargeSourceBytes   = int64(8) << 30
	storyboardReferencePixels    = int64(1920 * 1080)
)

type ProbeStatus string

const (
	ProbePending     ProbeStatus = "pending"
	ProbeReady       ProbeStatus = "ready"
	ProbeFailed      ProbeStatus = "failed"
	ProbeUnsupported ProbeStatus = "unsupported"
)

type PlaybackStatus string

const (
	PlaybackPlayable         PlaybackStatus = "playable"
	PlaybackUnsupportedCodec PlaybackStatus = "unsupported_codec"
	PlaybackNotApplicable    PlaybackStatus = "not_applicable"
	PlaybackUnknown          PlaybackStatus = "unknown"
)

type ProcessingErrorCode string

type FailureStage string
type FailureReason string

const (
	StageSourceRead   FailureStage = "source_read"
	StageProbe        FailureStage = "probe"
	StagePoster       FailureStage = "poster_extract"
	StageFrameExtract FailureStage = "frame_extract"
	StageCompose      FailureStage = "storyboard_compose"
	StageValidation   FailureStage = "output_validation"
	StageCachePublish FailureStage = "cache_publish"
)

const (
	ReasonTimedOut           FailureReason = "time_limit_exceeded"
	ReasonInvalidData        FailureReason = "invalid_media_data"
	ReasonMissingMoovAtom    FailureReason = "missing_moov_atom"
	ReasonContainerMismatch  FailureReason = "container_mismatch"
	ReasonDecoderUnavailable FailureReason = "decoder_unavailable"
	ReasonDecodeFailed       FailureReason = "decode_failed"
	ReasonDecodeRecovered    FailureReason = "decode_recovered"
	ReasonNoFrame            FailureReason = "frame_unavailable"
	ReasonOutputLimit        FailureReason = "output_limit_exceeded"
	ReasonToolFailed         FailureReason = "tool_failed"
	ReasonSourceUnavailable  FailureReason = "source_unavailable"
	ReasonSourceTooLarge     FailureReason = "source_too_large"
	ReasonCacheUnavailable   FailureReason = "cache_unavailable"
)

type FailureDiagnostic struct {
	Stage    FailureStage
	Reason   FailureReason
	Tool     string
	ExitCode *int
}

type DiagnosticError struct {
	cause      error
	diagnostic FailureDiagnostic
}

func (err *DiagnosticError) Error() string { return err.cause.Error() }
func (err *DiagnosticError) Unwrap() error { return err.cause }

func WithFailureDiagnostic(err error, diagnostic FailureDiagnostic) error {
	if err == nil {
		return nil
	}
	return &DiagnosticError{cause: err, diagnostic: diagnostic}
}

func DiagnoseFailure(err error) (FailureDiagnostic, bool) {
	var diagnosticError *DiagnosticError
	if !errors.As(err, &diagnosticError) {
		return FailureDiagnostic{}, false
	}
	return diagnosticError.diagnostic, true
}

const (
	ErrorUnsupportedMedia ProcessingErrorCode = "unsupported_media"
	ErrorInvalidMedia     ProcessingErrorCode = "invalid_media"
	ErrorProcessingFailed ProcessingErrorCode = "media_processing_failed"
	ErrorProcessingTimed  ProcessingErrorCode = "media_processing_timeout"
)

var (
	ErrInvalidMedia       = errors.New("invalid media")
	ErrFrameUnavailable   = errors.New("video frame unavailable")
	ErrSourceTooLarge     = errors.New("media source is too large")
	ErrUnsupportedMedia   = errors.New("unsupported media")
	ErrProcessingFailed   = errors.New("media processing failed")
	ErrProcessingTimedOut = errors.New("media processing timed out")
	ErrInvalidResult      = errors.New("invalid media processing result")
)

type Metadata struct {
	Width          int
	Height         int
	DurationMS     *int64
	PlaybackStatus PlaybackStatus
}

type Thumbnail struct {
	Bytes  []byte
	Width  int
	Height int
}

type ProcessingResult struct {
	Metadata  Metadata
	Thumbnail Thumbnail
	Warning   *FailureDiagnostic
}

type Processor interface {
	Process(context.Context, io.ReadSeeker, Format) (ProcessingResult, error)
}

type StoryboardRequest struct {
	TimestampsMS []int64
	Columns      int
	Rows         int
	CellWidth    int
	CellHeight   int
	Timeout      time.Duration
}

type StoryboardResult struct {
	Bytes      []byte
	FrameCount int
	Columns    int
	Rows       int
	CellWidth  int
	CellHeight int
}

type StoryboardProcessor interface {
	Storyboard(
		context.Context,
		io.ReadSeeker,
		Format,
		StoryboardRequest,
	) (StoryboardResult, error)
}

func ValidateStoryboardRequest(request StoryboardRequest) error {
	frameCount := len(request.TimestampsMS)
	if (frameCount != StoryboardMinFrames && frameCount != StoryboardMaxFrames) ||
		request.Columns != min(frameCount, StoryboardMaxColumns) ||
		request.Rows != (frameCount+request.Columns-1)/request.Columns ||
		request.Rows < 1 || request.Rows > StoryboardMaxRows ||
		request.CellWidth < 1 ||
		request.CellWidth > StoryboardMaxCellDimension ||
		request.CellHeight < 1 ||
		request.CellHeight > StoryboardMaxCellDimension ||
		request.Columns*request.CellWidth*request.Rows*request.CellHeight >
			StoryboardMaxPixels ||
		(request.Timeout != 0 &&
			(request.Timeout < 100*time.Millisecond ||
				request.Timeout > MaxStoryboardAttemptTimeout)) {
		return ErrInvalidResult
	}
	var previous int64
	for _, timestamp := range request.TimestampsMS {
		if timestamp <= previous {
			return ErrInvalidResult
		}
		previous = timestamp
	}
	return nil
}

func StoryboardProcessingTimeout(width, height, frameCount int) (time.Duration, error) {
	if ValidateDimensions(width, height) != nil ||
		(frameCount != StoryboardMinFrames && frameCount != StoryboardMaxFrames) {
		return 0, ErrInvalidResult
	}
	work := int64(width) * int64(height) * int64(frameCount)
	referenceWork := storyboardReferencePixels * int64(StoryboardMaxFrames)
	workUnits := (work + referenceWork - 1) / referenceWork
	timeout := time.Duration(max(int64(1), workUnits)) * DefaultStoryboardTimeout
	return min(timeout, 3*time.Minute), nil
}

// StoryboardProcessingTimeoutForSource extends only the long-frame plan for a
// very large video. File size is a conservative proxy for expensive remote
// seeks and high-bitrate decode work that dimensions alone cannot describe.
func StoryboardProcessingTimeoutForSource(
	width, height, frameCount int,
	sourceBytes int64,
) (time.Duration, error) {
	timeout, err := StoryboardProcessingTimeout(width, height, frameCount)
	if err != nil || sourceBytes < 1 || sourceBytes > MaxVideoSourceBytes {
		if err != nil {
			return 0, err
		}
		return 0, ErrInvalidResult
	}
	if frameCount == StoryboardMaxFrames &&
		sourceBytes >= StoryboardLargeSourceBytes {
		return max(timeout, LargeStoryboardTimeout), nil
	}
	return timeout, nil
}

func ValidateStoryboardResult(
	request StoryboardRequest,
	result StoryboardResult,
) error {
	if ValidateStoryboardRequest(request) != nil ||
		result.FrameCount != len(request.TimestampsMS) ||
		result.Columns != request.Columns ||
		result.Rows != request.Rows ||
		result.CellWidth != request.CellWidth ||
		result.CellHeight != request.CellHeight ||
		len(result.Bytes) < 12 ||
		len(result.Bytes) > StoryboardMaxOutputBytes ||
		string(result.Bytes[:4]) != "RIFF" ||
		string(result.Bytes[8:12]) != "WEBP" {
		return ErrInvalidResult
	}
	return nil
}

func ValidateProcessingResult(
	kind Kind,
	format Format,
	result ProcessingResult,
) error {
	if validateProcessingMetadata(kind, format, result.Metadata) != nil ||
		result.Thumbnail.Width < 1 || result.Thumbnail.Height < 1 ||
		result.Thumbnail.Width > GridThumbnailWidth ||
		result.Thumbnail.Height > GridThumbnailHeight ||
		len(result.Thumbnail.Bytes) == 0 ||
		len(result.Thumbnail.Bytes) > MaxToolOutputBytes ||
		!validProcessingWarning(kind, format, result) {
		return ErrInvalidResult
	}
	return nil
}

func validProcessingWarning(kind Kind, format Format, result ProcessingResult) bool {
	if result.Warning == nil {
		return true
	}
	warning := result.Warning
	return kind == KindImage && format == FormatJPEG &&
		ValidateDimensions(result.Metadata.Width, result.Metadata.Height) == nil &&
		warning.Stage == StageValidation &&
		warning.Reason == ReasonDecodeRecovered &&
		warning.Tool == "libvips" && warning.ExitCode == nil
}

func ValidateMetadata(kind Kind, metadata Metadata) error {
	if ValidateDimensions(metadata.Width, metadata.Height) != nil {
		return ErrInvalidResult
	}
	switch kind {
	case KindImage, KindAnimated:
		if metadata.DurationMS != nil ||
			metadata.PlaybackStatus != PlaybackNotApplicable {
			return ErrInvalidResult
		}
	case KindVideo:
		if (metadata.DurationMS != nil && *metadata.DurationMS < 0) ||
			(metadata.PlaybackStatus != PlaybackPlayable &&
				metadata.PlaybackStatus != PlaybackUnsupportedCodec &&
				metadata.PlaybackStatus != PlaybackUnknown) {
			return ErrInvalidResult
		}
	default:
		return ErrInvalidResult
	}
	return nil
}

func validateProcessingMetadata(kind Kind, format Format, metadata Metadata) error {
	if !processingKindMatchesFormat(kind, format) {
		return ErrInvalidResult
	}
	if kind == KindVideo {
		return ValidateMetadata(kind, metadata)
	}
	if ValidateImageDimensions(format, metadata.Width, metadata.Height) != nil ||
		metadata.DurationMS != nil ||
		metadata.PlaybackStatus != PlaybackNotApplicable {
		return ErrInvalidResult
	}
	return nil
}

func processingKindMatchesFormat(kind Kind, format Format) bool {
	switch kind {
	case KindImage:
		return format == FormatJPEG || format == FormatPNG || format == FormatWebP
	case KindAnimated:
		return format == FormatGIF
	case KindVideo:
		return format == FormatMP4 || format == FormatMOV ||
			format == FormatMKV || format == FormatAVI
	default:
		return false
	}
}

func ValidateSourceSize(format Format, sizeBytes int64) error {
	if sizeBytes < 1 {
		return WithFailureDiagnostic(ErrInvalidMedia, FailureDiagnostic{
			Stage: StageSourceRead, Reason: ReasonInvalidData, Tool: "filesystem",
		})
	}
	switch format {
	case FormatJPEG, FormatPNG, FormatWebP, FormatGIF:
		if sizeBytes > MaxImageSourceBytes {
			return WithFailureDiagnostic(ErrSourceTooLarge, FailureDiagnostic{
				Stage: StageSourceRead, Reason: ReasonSourceTooLarge, Tool: "filesystem",
			})
		}
	case FormatMP4, FormatMOV, FormatMKV, FormatAVI:
		if sizeBytes > MaxVideoSourceBytes {
			return WithFailureDiagnostic(ErrSourceTooLarge, FailureDiagnostic{
				Stage: StageSourceRead, Reason: ReasonSourceTooLarge, Tool: "filesystem",
			})
		}
	default:
		return ErrUnsupportedMedia
	}
	return nil
}

func ValidateDimensions(width, height int) error {
	return validateDimensions(width, height, MaxDecodedPixels)
}

// ValidateImageDimensions keeps full-resolution decode limits for formats that
// cannot shrink while loading. JPEG may exceed that working-pixel limit because
// the libvips adapter uses decoder-level shrink-on-load before evaluating pixels.
func ValidateImageDimensions(format Format, width, height int) error {
	if width < 1 || height < 1 {
		return WithFailureDiagnostic(ErrInvalidMedia, FailureDiagnostic{
			Stage: StageProbe, Reason: ReasonInvalidData, Tool: "libvips",
		})
	}
	maximumPixels := int64(MaxDecodedPixels)
	if format == FormatJPEG {
		maximumPixels = MaxJPEGSourcePixels
	}
	if err := validateDimensions(width, height, maximumPixels); err != nil {
		return WithFailureDiagnostic(ErrSourceTooLarge, FailureDiagnostic{
			Stage: StageProbe, Reason: ReasonSourceTooLarge, Tool: "libvips",
		})
	}
	return nil
}

func validateDimensions(width, height int, maximumPixels int64) error {
	if width < 1 || height < 1 ||
		width > MaxMediaDimension || height > MaxMediaDimension ||
		int64(width)*int64(height) > maximumPixels {
		return ErrInvalidMedia
	}
	return nil
}

func ProcessingCode(err error) ProcessingErrorCode {
	switch {
	case errors.Is(err, ErrUnsupportedMedia):
		return ErrorUnsupportedMedia
	case errors.Is(err, ErrInvalidMedia):
		return ErrorInvalidMedia
	case errors.Is(err, context.DeadlineExceeded),
		errors.Is(err, ErrProcessingTimedOut):
		return ErrorProcessingTimed
	default:
		return ErrorProcessingFailed
	}
}

func ProcessingError(code ProcessingErrorCode, cause error) error {
	switch code {
	case ErrorUnsupportedMedia:
		return fmt.Errorf("%w: %w", ErrUnsupportedMedia, cause)
	case ErrorInvalidMedia:
		return fmt.Errorf("%w: %w", ErrInvalidMedia, cause)
	case ErrorProcessingTimed:
		return fmt.Errorf("%w: %w", ErrProcessingTimedOut, cause)
	default:
		return fmt.Errorf("%w: %w", ErrProcessingFailed, cause)
	}
}
