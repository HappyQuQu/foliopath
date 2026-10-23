package face

import (
	"context"
	"errors"
	"time"
)

const FailureSampleLimit = 20

// FailureCode never contains an underlying exception or filesystem path.
func FailureCode(err error) string {
	switch {
	case errors.Is(err, ErrMediaUnreadable):
		return "media_unreadable"
	case errors.Is(err, ErrInvalidInput):
		return "invalid_media"
	case errors.Is(err, ErrInvalidOutput):
		return "inference_invalid_output"
	case errors.Is(err, context.DeadlineExceeded):
		return "inference_timeout"
	default:
		return "analysis_failed"
	}
}

func ValidFailureCode(code string) bool {
	switch code {
	case "media_unreadable", "invalid_media", "inference_invalid_output", "inference_timeout", "analysis_failed":
		return true
	}
	return false
}

type DiagnosticTask struct {
	OperationID, State, ErrorCode string
	UpdatedAt                     time.Time
}
type FailureReason struct {
	Code  string
	Count int64
}
type FailureSample struct {
	AssetID    int64
	SizeBytes  int64
	Name, Code string
	UpdatedAt  time.Time
}
type FailureDiagnostics struct {
	LatestTask                           *DiagnosticTask
	RecordedFailures, UnrecordedFailures int64
	Reasons                              []FailureReason
	Samples                              []FailureSample
}
