package thumbnail

import (
	"context"
	"errors"

	"github.com/HappyQuQu/foliopath/internal/media"
)

const MaxFailurePageSize = 100

var (
	ErrInvalidDiagnosticsRequest  = errors.New("invalid media diagnostics request")
	ErrDiagnosticsLibraryNotFound = errors.New("media diagnostics library not found")
	ErrDiagnosticsFailureNotFound = errors.New("media diagnostics failure not found")
)

type MediaFailureAttempt struct {
	AttemptNumber int
	Outcome       JobOutcome
	Stage         media.FailureStage
	Reason        media.FailureReason
	Tool          string
	ExitCode      *int
	DurationMS    int64
	FinishedAtMS  int64
}

type MediaFailure struct {
	JobID          int64
	LibraryID      int64
	AssetID        int64
	LibraryName    string
	RelativePath   string
	Variant        Variant
	ErrorCode      JobErrorCode
	AttemptCount   int
	FinishedAtMS   int64
	LatestAttempt  *MediaFailureAttempt
	AttemptHistory []MediaFailureAttempt
}

type FailureQuery struct {
	LibraryID int64
	Variant   Variant
	ErrorCode JobErrorCode
	BeforeID  int64
	Limit     int
}

type FailureRevision struct {
	FinishedAtMS int64
	JobID        int64
}

type RetrySummary struct {
	Requeued          int64
	RemainingEligible int64
	PermanentFailures int64
}

type RequeueMode string

const (
	RequeueMissing RequeueMode = "missing"
	RequeueAll     RequeueMode = "all"
)

type DiagnosticsRepository interface {
	ListMediaFailures(context.Context, FailureQuery) ([]MediaFailure, error)
	LatestMediaFailureRevision(context.Context, FailureQuery) (FailureRevision, bool, error)
	GetMediaFailure(context.Context, int64) (MediaFailure, error)
	RequeueMediaProcessing(context.Context, int64, RequeueMode, int) (RetrySummary, error)
}

func (service *DiagnosticsService) LatestFailureRevision(
	ctx context.Context,
	query FailureQuery,
) (FailureRevision, bool, error) {
	if query.LibraryID < 0 ||
		(query.Variant != "" && query.Variant != VariantGrid && query.Variant != VariantStoryboard) ||
		(query.ErrorCode != "" && !validDiagnosticErrorCode(query.ErrorCode)) {
		return FailureRevision{}, false, ErrInvalidDiagnosticsRequest
	}
	return service.repository.LatestMediaFailureRevision(ctx, query)
}

type DiagnosticsService struct {
	repository DiagnosticsRepository
	waker      interface{ Wake() }
}

func (service *DiagnosticsService) GetFailure(
	ctx context.Context,
	jobID int64,
) (MediaFailure, error) {
	if jobID <= 0 {
		return MediaFailure{}, ErrInvalidDiagnosticsRequest
	}
	return service.repository.GetMediaFailure(ctx, jobID)
}

func NewDiagnosticsService(
	repository DiagnosticsRepository,
	waker interface{ Wake() },
) (*DiagnosticsService, error) {
	if repository == nil || waker == nil {
		return nil, errors.New("media diagnostics dependencies are required")
	}
	return &DiagnosticsService{repository: repository, waker: waker}, nil
}

func (service *DiagnosticsService) ListFailures(
	ctx context.Context,
	query FailureQuery,
) ([]MediaFailure, error) {
	if query.LibraryID < 0 || query.BeforeID < 0 || query.Limit < 1 ||
		query.Limit > MaxFailurePageSize ||
		(query.Variant != "" && query.Variant != VariantGrid &&
			query.Variant != VariantStoryboard) ||
		(query.ErrorCode != "" && !validDiagnosticErrorCode(query.ErrorCode)) {
		return nil, ErrInvalidDiagnosticsRequest
	}
	return service.repository.ListMediaFailures(ctx, query)
}

func (service *DiagnosticsService) ProcessMedia(
	ctx context.Context,
	libraryID int64,
	mode RequeueMode,
	limit int,
) (RetrySummary, error) {
	if libraryID <= 0 || (mode != RequeueMissing && mode != RequeueAll) ||
		limit < 1 || limit > 256 {
		return RetrySummary{}, ErrInvalidDiagnosticsRequest
	}
	summary, err := service.repository.RequeueMediaProcessing(ctx, libraryID, mode, limit)
	if err == nil && summary.Requeued > 0 {
		service.waker.Wake()
	}
	return summary, err
}

func validDiagnosticErrorCode(code JobErrorCode) bool {
	switch code {
	case JobErrorInvalidMedia, JobErrorUnsupportedMedia, JobErrorProcessing,
		JobErrorTimeout, JobErrorSource, JobErrorCache:
		return true
	default:
		return false
	}
}
