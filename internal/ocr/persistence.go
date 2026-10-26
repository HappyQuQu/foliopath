package ocr

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/HappyQuQu/foliopath/internal/jobs"
	"github.com/HappyQuQu/foliopath/internal/media"
)

var (
	ErrInvalidPersistence = errors.New("invalid OCR persistence request")
	ErrConflict           = errors.New("OCR state conflict")
	ErrNotFound           = errors.New("OCR resource not found")
	ErrOffline            = fmt.Errorf("%w: OCR library not ready", ErrConflict)
	ErrScanning           = errors.New("OCR catalog scan in progress")
	ErrDisabled           = errors.New("OCR library disabled")
	ErrRevision           = fmt.Errorf("%w: revision mismatch", ErrConflict)
)

const (
	MaximumActiveJobs = 32
	MaximumAttempts   = 3
	MaximumClearBatch = 500
)

type Mode string

const (
	Missing     Mode = "missing"
	RetryFailed Mode = "retry_failed"
)

type Settings struct {
	LibraryID, Revision, Epoch int64
	Enabled, Clearing          bool
	Fingerprint                string
}

type Candidate struct {
	AssetID           int64
	SourceFingerprint string
}

type Job struct {
	ID                                                                     string
	LibraryID, Epoch                                                       int64
	Fingerprint                                                            string
	Mode                                                                   Mode
	State                                                                  string
	CheckpointID, UpperAssetID, ClaimRevision, Attempts, Completed, Failed int64
	ErrorCode                                                              string
}

type StoredResult struct {
	Candidate
	Fingerprint, Outcome, ErrorCode string
	Width, Height                   int
	Result                          Result
}

type Coverage struct{ Eligible, Ready, Empty, Failed, Pending, Stale int64 }

type Commit struct {
	Job   Job
	Value StoredResult
}

// Repository is internal-only. Adapters own durable transitions; runtime work
// reuses jobs.WorkerPool and the existing media/inference boundaries.
// No method reads originals, authorizes API access or starts a worker.
type Repository interface {
	GetOCRSettings(context.Context, int64) (Settings, error)
	GetOCROverview(context.Context, int64) (Status, error)
	GetOCRCoverage(context.Context, int64) (Coverage, error)
	ListOCRClearingLibraries(context.Context, int64, int) ([]Settings, error)
	DeferOCRJob(context.Context, Job, time.Time, time.Duration) error
	HandoffOCRJob(context.Context, Job, time.Time) error
	ConfigureOCR(context.Context, int64, int64, bool, Identity, time.Time) (Settings, error)
	AdmitOCRJob(context.Context, int64, int64, Mode, string, string, time.Time) (Job, bool, error)
	GetOCRJob(context.Context, string) (Job, error)
	GetLatestOCRJob(context.Context, int64) (Job, bool, error)
	ClaimOCRJob(context.Context, time.Time, time.Duration) (Job, bool, error)
	RefreshOCRLease(context.Context, Job, time.Time, time.Duration) error
	RecoverOCRJobs(context.Context, time.Time) (jobs.RecoverySummary, error)
	NextOCRCandidate(context.Context, Job, time.Time) (Candidate, bool, error)
	CommitOCRResult(context.Context, Commit, time.Time) error
	FinishOCRJob(context.Context, Job, string, string, time.Time) error
	CancelOCRJob(context.Context, string, int64, time.Time) error
	GetCurrentOCRResult(context.Context, int64, int64) (StoredResult, bool, error)
	BeginOCRClear(context.Context, int64, int64, time.Time) (Settings, error)
	DrainOCRClear(context.Context, int64, int64, int, time.Time) (bool, error)
}

var opaque = regexp.MustCompile(`^[A-Za-z0-9._~-]{8,128}$`)

func ValidOpaque(value string) bool { return opaque.MatchString(value) }
func ValidMode(mode Mode) bool      { return mode == Missing || mode == RetryFailed }
func RequestHashes(libraryID, epoch int64, fingerprint string, mode Mode, key string) (string, string, error) {
	if libraryID < 1 || epoch < 1 || !ValidMode(mode) || !ValidOpaque(key) {
		return "", "", ErrInvalidPersistence
	}
	hash := func(value string) string { v := sha256.Sum256([]byte(value)); return hex.EncodeToString(v[:]) }
	return hash("ocr-key-v1\x00" + key), hash(fmt.Sprintf("ocr-request-v1\x00%d\x00%d\x00%s\x00%s", libraryID, epoch, fingerprint, mode)), nil
}

func ValidateStoredResult(value StoredResult) (StoredResult, error) {
	if value.AssetID < 1 {
		return value, ErrInvalidPersistence
	}
	if !media.SourceFingerprint(value.SourceFingerprint).Valid() {
		return value, ErrInvalidPersistence
	}
	switch value.Outcome {
	case "ready":
		if value.ErrorCode != "" || value.Width < MinImageDimension || value.Height < MinImageDimension || value.Width > MaxPreparedDimension || value.Height > MaxPreparedDimension || int64(value.Width)*int64(value.Height) > MaxPreparedPixels {
			return value, ErrInvalidPersistence
		}
		result, err := Normalize(value.Result.Lines)
		if err != nil {
			return value, err
		}
		value.Result = result
	case "failed":
		if value.ErrorCode != "invalid_media" && value.ErrorCode != "input_limit" {
			return value, ErrInvalidPersistence
		}
		if value.Width != 0 || value.Height != 0 || value.Result.Lines != nil {
			return value, ErrInvalidPersistence
		}
	default:
		return value, ErrInvalidPersistence
	}
	return value, nil
}
