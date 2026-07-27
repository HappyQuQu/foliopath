// Package scanner owns full-scan orchestration and generation semantics.
package scanner

import (
	"context"
	"errors"
	"fmt"
	"path"
	"time"

	"github.com/HappyQuQu/foliopath/internal/pathpolicy"
)

var (
	ErrScanActive            = errors.New("a full scan is already active for the library")
	ErrScanRunNotFound       = errors.New("scan run not found")
	ErrScanRunNotActive      = errors.New("scan run is not active")
	ErrLibraryNotFound       = errors.New("scan library not found")
	ErrLibraryOffline        = errors.New("library root is offline")
	ErrLibraryRootSymlink    = errors.New("library root is a symbolic link")
	ErrLibraryMountBoundary  = errors.New("library contains a mount boundary")
	ErrPartialTreeUnreadable = errors.New("part of the library tree is unreadable")
	ErrScanIO                = errors.New("scan filesystem I/O failed")
	ErrRootIdentityChanged   = errors.New("library root identity changed during scan")
	ErrInvalidRootIdentity   = errors.New("invalid library root identity")
	ErrInvalidEntry          = errors.New("invalid scan entry")
	ErrBatchTooLarge         = errors.New("scan batch exceeds configured limit")
	ErrDatabaseUnavailable   = errors.New("scan database unavailable")
)

type Trigger string

const (
	TriggerCreation  Trigger = "library_created"
	TriggerStartup   Trigger = "startup"
	TriggerManual    Trigger = "manual"
	TriggerScheduled Trigger = "scheduled"
)

func (t Trigger) Valid() bool {
	switch t {
	case TriggerCreation, TriggerStartup, TriggerManual, TriggerScheduled:
		return true
	default:
		return false
	}
}

type RunStatus string

const (
	RunStatusQueued      RunStatus = "queued"
	RunStatusRunning     RunStatus = "running"
	RunStatusSucceeded   RunStatus = "succeeded"
	RunStatusFailed      RunStatus = "failed"
	RunStatusCancelled   RunStatus = "cancelled"
	RunStatusOffline     RunStatus = "offline"
	RunStatusInterrupted RunStatus = "interrupted"
)

type ScanRun struct {
	ID                    int64
	LibraryID             int64
	Generation            int64
	Trigger               Trigger
	Status                RunStatus
	DiscoveredDirectories int64
	DiscoveredAssets      int64
	SkippedCount          int64
	ErrorCode             string
	CreatedAtMS           int64
	StartedAtMS           *int64
	FinishedAtMS          *int64
	Revision              int64
	Phase                 string
	ProcessedAssets       int64
	SkippedDirectories    int64
	SkippedFiles          int64
	ErrorCount            int64
	IssuesTruncated       bool
	CancelRequestedAtMS   *int64
}

type SkipCounts struct {
	Directories int64
	Files       int64
}

func (counts SkipCounts) Total() int64 {
	return counts.Directories + counts.Files
}

func (counts SkipCounts) Valid() bool {
	return counts.Directories >= 0 && counts.Files >= 0
}

type CatalogEntryKind string

const (
	CatalogEntryDirectory CatalogEntryKind = "directory"
	CatalogEntryAsset     CatalogEntryKind = "asset"
)

type AssetKind string

const (
	AssetKindImage    AssetKind = "image"
	AssetKindAnimated AssetKind = "animated"
	AssetKindVideo    AssetKind = "video"
)

type MediaFormat string

const (
	MediaFormatJPEG MediaFormat = "jpeg"
	MediaFormatPNG  MediaFormat = "png"
	MediaFormatWebP MediaFormat = "webp"
	MediaFormatGIF  MediaFormat = "gif"
	MediaFormatMP4  MediaFormat = "mp4"
	MediaFormatMOV  MediaFormat = "mov"
	MediaFormatMKV  MediaFormat = "mkv"
)

type CatalogEntry struct {
	Kind         CatalogEntryKind
	RelativePath string
	ParentPath   string
	Name         string
	MTimeNS      int64
	AssetKind    AssetKind
	MediaFormat  MediaFormat
	MIMEType     string
	SizeBytes    int64
}

// Repository is owned by the scanner capability. CompleteFullScan is the only
// method allowed to remove stale catalog rows.
type Repository interface {
	GetLibraryRoot(context.Context, int64) (string, error)
	BeginFullScan(context.Context, int64, Trigger) (ScanRun, error)
	SetFullScanPhase(context.Context, int64, string) error
	UpsertCatalogBatch(context.Context, int64, []CatalogEntry) error
	CompleteFullScan(context.Context, int64, SkipCounts) (ScanRun, error)
	FailFullScan(context.Context, int64, SkipCounts, string) (ScanRun, error)
	CancelFullScan(context.Context, int64, SkipCounts) (ScanRun, error)
	OfflineFullScan(context.Context, int64, SkipCounts, string) (ScanRun, error)
	InterruptActiveScans(context.Context) (int64, error)
	GetScanRun(context.Context, int64) (ScanRun, error)
}

// RootIdentity is captured from the opened library root. Production walkers
// populate it from the root file descriptor's device and inode.
type RootIdentity struct {
	Device uint64
	Inode  uint64
}

func (i RootIdentity) Valid() bool { return i.Inode != 0 }

type WalkDecision uint8

const (
	WalkContinue WalkDecision = iota
	WalkSkipDirectory
)

type WalkEntry struct {
	RelativePath string
	IsDirectory  bool
	SizeBytes    int64
	MTimeNS      int64
	Skipped      bool
}

// Walker supplies a streaming, pre-order traversal. It must invoke visit
// serially, visit a directory before its descendants, honor SkipDirectory, and
// never expose an absolute path. CaptureRoot and VerifyRoot must inspect the
// same library root using a stable device/inode identity.
type Walker interface {
	CaptureRoot(context.Context, string) (RootIdentity, error)
	Walk(context.Context, string, func(WalkEntry) (WalkDecision, error)) error
	VerifyRoot(context.Context, string, RootIdentity) error
}

type FullScanRequest struct {
	LibraryID int64
	Trigger   Trigger
	Walker    Walker
}

const (
	PhaseCheckingRoot = "checking_root"
	PhaseWalking      = "walking"
	PhaseFinalizing   = "finalizing"
)

type Config struct {
	BatchSize       int
	FinalizeTimeout time.Duration
}

const DefaultBatchSize = 256

type Service struct {
	repository      Repository
	batchSize       int
	finalizeTimeout time.Duration
}

func NewService(repository Repository, config Config) (*Service, error) {
	if repository == nil {
		return nil, errors.New("scanner repository is required")
	}
	if config.BatchSize == 0 {
		config.BatchSize = DefaultBatchSize
	}
	if config.BatchSize < 1 || config.BatchSize > 1000 {
		return nil, fmt.Errorf("batch size must be between 1 and 1000")
	}
	if config.FinalizeTimeout == 0 {
		config.FinalizeTimeout = 5 * time.Second
	}
	if config.FinalizeTimeout < 0 {
		return nil, errors.New("finalize timeout must be positive")
	}
	return &Service{
		repository:      repository,
		batchSize:       config.BatchSize,
		finalizeTimeout: config.FinalizeTimeout,
	}, nil
}

func (s *Service) RunFullScan(ctx context.Context, request FullScanRequest) (ScanRun, error) {
	if request.LibraryID <= 0 {
		return ScanRun{}, ErrLibraryNotFound
	}
	if request.Walker == nil {
		return ScanRun{}, errors.New("scanner walker is required")
	}
	if !request.Trigger.Valid() {
		return ScanRun{}, fmt.Errorf("invalid scan trigger %q", request.Trigger)
	}
	if err := ctx.Err(); err != nil {
		return ScanRun{}, err
	}

	run, err := s.repository.BeginFullScan(ctx, request.LibraryID, request.Trigger)
	if err != nil {
		return ScanRun{}, err
	}
	return s.RunClaimedFullScan(ctx, run, request.Walker)
}

// RunClaimedFullScan executes a durable run that a QueueRepository has already
// claimed. It never creates a second queue record or allocates a generation.
func (s *Service) RunClaimedFullScan(
	ctx context.Context,
	run ScanRun,
	walker Walker,
) (ScanRun, error) {
	if run.ID <= 0 || run.LibraryID <= 0 || run.Generation <= 0 ||
		run.Status != RunStatusRunning {
		return ScanRun{}, ErrScanRunNotActive
	}
	if walker == nil {
		return ScanRun{}, errors.New("scanner walker is required")
	}
	if err := ctx.Err(); err != nil {
		return ScanRun{}, err
	}
	rootRelativePath, err := s.repository.GetLibraryRoot(ctx, run.LibraryID)
	if err != nil {
		return s.abort(ctx, run, SkipCounts{}, err)
	}
	if err := validateRoot(rootRelativePath); err != nil {
		return s.abort(ctx, run, SkipCounts{}, err)
	}

	identity, err := walker.CaptureRoot(ctx, rootRelativePath)
	if err != nil {
		return s.abort(ctx, run, SkipCounts{}, err)
	}
	if !identity.Valid() {
		return s.abort(ctx, run, SkipCounts{}, ErrInvalidRootIdentity)
	}
	if err := s.repository.SetFullScanPhase(ctx, run.ID, PhaseWalking); err != nil {
		return s.abort(ctx, run, SkipCounts{}, err)
	}

	batch := make([]CatalogEntry, 0, s.batchSize)
	skipped := SkipCounts{}
	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		if err := s.repository.UpsertCatalogBatch(ctx, run.ID, batch); err != nil {
			return err
		}
		batch = batch[:0]
		return nil
	}
	appendEntry := func(entry CatalogEntry) error {
		batch = append(batch, entry)
		if len(batch) >= s.batchSize {
			return flush()
		}
		return nil
	}

	if err := appendEntry(CatalogEntry{Kind: CatalogEntryDirectory}); err != nil {
		return s.abort(ctx, run, skipped, err)
	}

	walkErr := walker.Walk(ctx, rootRelativePath, func(entry WalkEntry) (WalkDecision, error) {
		if err := ctx.Err(); err != nil {
			return WalkContinue, err
		}
		if entry.Skipped {
			if entry.IsDirectory {
				skipped.Directories++
				return WalkSkipDirectory, nil
			}
			skipped.Files++
			return WalkContinue, nil
		}
		relativePath, err := normalizeEntryPath(entry.RelativePath)
		if err != nil {
			return WalkContinue, err
		}
		if relativePath == "" {
			if entry.IsDirectory {
				return WalkContinue, nil
			}
			return WalkContinue, ErrInvalidEntry
		}

		if entry.IsDirectory {
			if IsSystemDirectory(path.Base(relativePath)) {
				skipped.Directories++
				return WalkSkipDirectory, nil
			}
			err := appendEntry(CatalogEntry{
				Kind:         CatalogEntryDirectory,
				RelativePath: relativePath,
				ParentPath:   parentPath(relativePath),
				Name:         path.Base(relativePath),
				MTimeNS:      entry.MTimeNS,
			})
			return WalkContinue, err
		}

		assetKind, format, mimeType, supported := ClassifyPath(relativePath)
		if !supported {
			skipped.Files++
			return WalkContinue, nil
		}
		if entry.SizeBytes < 0 {
			return WalkContinue, ErrInvalidEntry
		}
		err = appendEntry(CatalogEntry{
			Kind:         CatalogEntryAsset,
			RelativePath: relativePath,
			ParentPath:   parentPath(relativePath),
			Name:         path.Base(relativePath),
			MTimeNS:      entry.MTimeNS,
			AssetKind:    assetKind,
			MediaFormat:  format,
			MIMEType:     mimeType,
			SizeBytes:    entry.SizeBytes,
		})
		return WalkContinue, err
	})
	if walkErr != nil {
		return s.abort(ctx, run, skipped, walkErr)
	}
	if err := flush(); err != nil {
		return s.abort(ctx, run, skipped, err)
	}
	if err := ctx.Err(); err != nil {
		return s.abort(ctx, run, skipped, err)
	}
	if err := walker.VerifyRoot(ctx, rootRelativePath, identity); err != nil {
		return s.abort(ctx, run, skipped, err)
	}
	if err := ctx.Err(); err != nil {
		return s.abort(ctx, run, skipped, err)
	}
	if err := s.repository.SetFullScanPhase(ctx, run.ID, PhaseFinalizing); err != nil {
		return s.abort(ctx, run, skipped, err)
	}

	completed, err := s.repository.CompleteFullScan(ctx, run.ID, skipped)
	if err != nil {
		return s.abort(ctx, run, skipped, err)
	}
	return completed, nil
}

func (s *Service) RecoverInterruptedScans(ctx context.Context) (int64, error) {
	return s.repository.InterruptActiveScans(ctx)
}

func (s *Service) abort(
	ctx context.Context,
	run ScanRun,
	skipped SkipCounts,
	cause error,
) (ScanRun, error) {
	finishCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), s.finalizeTimeout)
	defer cancel()

	var (
		finished ScanRun
		err      error
	)
	switch {
	case errors.Is(cause, context.Canceled), errors.Is(cause, context.DeadlineExceeded):
		finished, err = s.repository.CancelFullScan(finishCtx, run.ID, skipped)
	case errors.Is(cause, ErrLibraryOffline):
		finished, err = s.repository.OfflineFullScan(
			finishCtx,
			run.ID,
			skipped,
			"library_root_unavailable",
		)
	default:
		finished, err = s.repository.FailFullScan(finishCtx, run.ID, skipped, safeErrorCode(cause))
	}
	if err != nil {
		return run, errors.Join(cause, fmt.Errorf("record scan termination: %w", err))
	}
	return finished, cause
}

func safeErrorCode(err error) string {
	switch {
	case errors.Is(err, ErrRootIdentityChanged):
		return "library_root_identity_changed"
	case errors.Is(err, ErrLibraryRootSymlink):
		return "library_root_symlink"
	case errors.Is(err, ErrLibraryMountBoundary):
		return "library_root_mount_boundary"
	case errors.Is(err, ErrPartialTreeUnreadable):
		return "partial_tree_unreadable"
	case errors.Is(err, ErrScanIO):
		return "scan_io_error"
	case errors.Is(err, ErrDatabaseUnavailable):
		return "database_unavailable"
	default:
		return "internal_error"
	}
}

func validateRoot(root string) error {
	if root == "" {
		return nil
	}
	_, err := normalizeEntryPath(root)
	return err
}

func normalizeEntryPath(value string) (string, error) {
	normalized, err := pathpolicy.Normalize(value)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrInvalidEntry, err)
	}
	return normalized, nil
}

func parentPath(relativePath string) string {
	parent := path.Dir(relativePath)
	if parent == "." {
		return ""
	}
	return parent
}
