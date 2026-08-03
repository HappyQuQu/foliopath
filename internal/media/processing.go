package media

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"
)

const (
	GridThumbnailWidth         = 512
	GridThumbnailHeight        = 512
	GridWebPQuality            = 82
	MaxToolOutputBytes         = 8 << 20
	MaxImageSourceBytes        = 256 << 20
	MaxVideoSourceBytes        = int64(4) << 30
	MaxDecodedPixels           = 100_000_000
	MaxMediaDimension          = 32_768
	DefaultProbeTimeout        = 15 * time.Second
	StoryboardMinFrames        = 4
	StoryboardMaxFrames        = 10
	StoryboardMaxColumns       = 5
	StoryboardMaxRows          = 2
	StoryboardMaxCellDimension = 320
	StoryboardMaxPixels        = 1_024_000
	StoryboardMaxFrameBytes    = 1 << 20
	StoryboardMaxTempBytes     = 10 << 20
	StoryboardMaxOutputBytes   = 8 << 20
	DefaultStoryboardTimeout   = 45 * time.Second
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
	ReasonDecoderUnavailable FailureReason = "decoder_unavailable"
	ReasonDecodeFailed       FailureReason = "decode_failed"
	ReasonNoFrame            FailureReason = "frame_unavailable"
	ReasonOutputLimit        FailureReason = "output_limit_exceeded"
	ReasonToolFailed         FailureReason = "tool_failed"
	ReasonSourceUnavailable  FailureReason = "source_unavailable"
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
			StoryboardMaxPixels {
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

func ValidateProcessingResult(kind Kind, result ProcessingResult) error {
	if ValidateDimensions(result.Metadata.Width, result.Metadata.Height) != nil ||
		result.Thumbnail.Width < 1 || result.Thumbnail.Height < 1 ||
		result.Thumbnail.Width > GridThumbnailWidth ||
		result.Thumbnail.Height > GridThumbnailHeight ||
		len(result.Thumbnail.Bytes) == 0 ||
		len(result.Thumbnail.Bytes) > MaxToolOutputBytes {
		return ErrInvalidResult
	}
	switch kind {
	case KindImage, KindAnimated:
		if result.Metadata.DurationMS != nil ||
			result.Metadata.PlaybackStatus != PlaybackNotApplicable {
			return ErrInvalidResult
		}
	case KindVideo:
		if result.Metadata.DurationMS == nil || *result.Metadata.DurationMS < 0 ||
			(result.Metadata.PlaybackStatus != PlaybackPlayable &&
				result.Metadata.PlaybackStatus != PlaybackUnsupportedCodec &&
				result.Metadata.PlaybackStatus != PlaybackUnknown) {
			return ErrInvalidResult
		}
	default:
		return ErrInvalidResult
	}
	return nil
}

func ValidateSourceSize(format Format, sizeBytes int64) error {
	if sizeBytes < 1 {
		return ErrInvalidMedia
	}
	switch format {
	case FormatJPEG, FormatPNG, FormatWebP, FormatGIF:
		if sizeBytes > MaxImageSourceBytes {
			return ErrInvalidMedia
		}
	case FormatMP4, FormatMOV, FormatMKV, FormatAVI:
		if sizeBytes > MaxVideoSourceBytes {
			return ErrInvalidMedia
		}
	default:
		return ErrUnsupportedMedia
	}
	return nil
}

func ValidateDimensions(width, height int) error {
	if width < 1 || height < 1 ||
		width > MaxMediaDimension || height > MaxMediaDimension ||
		int64(width)*int64(height) > MaxDecodedPixels {
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
