package media

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"
)

const (
	GridThumbnailWidth  = 512
	GridThumbnailHeight = 512
	GridWebPQuality     = 82
	MaxToolOutputBytes  = 8 << 20
	MaxImageSourceBytes = 256 << 20
	MaxVideoSourceBytes = int64(4) << 30
	MaxDecodedPixels    = 100_000_000
	MaxMediaDimension   = 32_768
	DefaultProbeTimeout = 15 * time.Second
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
	case FormatMP4, FormatMOV, FormatMKV:
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
