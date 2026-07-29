package scanner

import (
	"context"
	"errors"
	"path"
	"time"

	"github.com/HappyQuQu/foliopath/internal/pathpolicy"
)

const (
	MaxDirectoryWatches      = 32_768
	MaxPendingWatchEvents    = 8_192
	MaxDirtyDirectories      = 4_096
	MaxConcurrentReconciles  = 2
	MaxLibraryReconciles     = 1
	MaxReconcileEntries      = 2_048
	MaxReconcileAttempts     = 5
	ReconcileDebounce        = 750 * time.Millisecond
	ReconcileMaximumDebounce = 5 * time.Second
)

var (
	ErrInvalidReconcileTarget = errors.New("invalid reconciliation target")
	ErrReconcileNotFound      = errors.New("reconciliation job not found")
	ErrReconcileNotActive     = errors.New("reconciliation job is not active")
	ErrReconcileCapacity      = errors.New("reconciliation admission capacity reached")
)

type ReconcileStatus string

const (
	ReconcileQueued  ReconcileStatus = "queued"
	ReconcileRunning ReconcileStatus = "running"
	ReconcileFailed  ReconcileStatus = "failed"
)

type WatchEventKind string

const (
	WatchEventDirty       WatchEventKind = "dirty"
	WatchEventOverflow    WatchEventKind = "overflow"
	WatchEventInvalidated WatchEventKind = "invalidated"
)

type WatchEvent struct {
	LibraryID         int64
	RelativeDirectory string
	Kind              WatchEventKind
}

type ReconcileJob struct {
	ID                int64
	LibraryID         int64
	RelativeDirectory string
	Status            ReconcileStatus
	RequestedRevision int64
	ClaimedRevision   *int64
	AvailableAtMS     int64
	LeaseExpiresAtMS  *int64
	AttemptCount      int64
	LastErrorCode     string
	CreatedAtMS       int64
	UpdatedAtMS       int64
}

type ReconcileRepository interface {
	EnqueueReconcile(context.Context, int64, string, time.Duration, time.Duration) (ReconcileJob, error)
}

type ReconcileReader interface {
	CaptureRoot(context.Context, string) (RootIdentity, error)
	ReadDirectory(
		context.Context,
		string,
		string,
		func(WalkEntry) error,
	) error
	VerifyRoot(context.Context, string, RootIdentity) error
}

type ReconcileCommitResult struct {
	Changed        bool
	Requeued       bool
	NewDirectories []string
}

type ReconcileExecutionRepository interface {
	GetLibraryRoot(context.Context, int64) (string, error)
	CommitDirectoryReconcile(
		context.Context,
		ReconcileJob,
		[]CatalogEntry,
	) (ReconcileCommitResult, error)
	FailDirectoryReconcile(context.Context, ReconcileJob, string) error
}

type AutomaticDiscoveryStateRepository interface {
	SetAutomaticDiscoveryState(
		context.Context,
		int64,
		string,
		string,
	) error
}

type ReconcileProcessor struct {
	repository ReconcileExecutionRepository
	reader     ReconcileReader
	mediaWaker WakeNotifier
	observer   ReconcileDirectoryObserver
}

type ReconcileDirectoryObserver interface {
	DirectoryDiscovered(context.Context, int64, string, string) error
	ReconcileFailed(context.Context, ReconcileJob, string) error
}

func NewReconcileProcessor(
	repository ReconcileExecutionRepository,
	reader ReconcileReader,
	mediaWaker WakeNotifier,
	observer ReconcileDirectoryObserver,
) (*ReconcileProcessor, error) {
	if repository == nil || reader == nil || mediaWaker == nil {
		return nil, errors.New("reconciliation processor dependencies are required")
	}
	return &ReconcileProcessor{
		repository: repository,
		reader:     reader,
		mediaWaker: mediaWaker,
		observer:   observer,
	}, nil
}

func (processor *ReconcileProcessor) Process(
	ctx context.Context,
	job ReconcileJob,
) error {
	if job.ID <= 0 || job.LibraryID <= 0 ||
		job.Status != ReconcileRunning || job.ClaimedRevision == nil {
		return ErrReconcileNotActive
	}
	root, err := processor.repository.GetLibraryRoot(ctx, job.LibraryID)
	if err != nil {
		return processor.fail(ctx, job, "source_unavailable", err)
	}
	identity, err := processor.reader.CaptureRoot(ctx, root)
	if err != nil {
		return processor.fail(ctx, job, "source_unavailable", err)
	}
	if !identity.Valid() {
		return processor.fail(ctx, job, "source_changed", ErrInvalidRootIdentity)
	}

	entries := make([]CatalogEntry, 0, 64)
	err = processor.reader.ReadDirectory(
		ctx,
		root,
		job.RelativeDirectory,
		func(entry WalkEntry) error {
			if entry.Skipped {
				return nil
			}
			relative, err := normalizeEntryPath(entry.RelativePath)
			if err != nil || relative == "" ||
				parentPath(relative) != job.RelativeDirectory {
				return ErrInvalidEntry
			}
			if entry.IsDirectory {
				if IsSystemDirectory(path.Base(relative)) {
					return nil
				}
				entries = append(entries, CatalogEntry{
					Kind:         CatalogEntryDirectory,
					RelativePath: relative,
					ParentPath:   job.RelativeDirectory,
					Name:         path.Base(relative),
					MTimeNS:      entry.MTimeNS,
				})
				return nil
			}
			assetKind, format, mimeType, supported := ClassifyPath(relative)
			if !supported {
				return nil
			}
			if entry.SizeBytes < 0 {
				return ErrInvalidEntry
			}
			entries = append(entries, CatalogEntry{
				Kind:         CatalogEntryAsset,
				RelativePath: relative,
				ParentPath:   job.RelativeDirectory,
				Name:         path.Base(relative),
				MTimeNS:      entry.MTimeNS,
				AssetKind:    assetKind,
				MediaFormat:  format,
				MIMEType:     mimeType,
				SizeBytes:    entry.SizeBytes,
			})
			return nil
		},
	)
	if err != nil {
		return processor.fail(ctx, job, reconcileFailureCode(err), err)
	}
	if err := processor.reader.VerifyRoot(ctx, root, identity); err != nil {
		return processor.fail(ctx, job, "source_changed", err)
	}
	result, err := processor.repository.CommitDirectoryReconcile(ctx, job, entries)
	if err != nil {
		return processor.fail(ctx, job, "internal_error", err)
	}
	if result.Changed {
		processor.mediaWaker.Wake()
	}
	if processor.observer != nil {
		for _, relativeDirectory := range result.NewDirectories {
			if err := processor.observer.DirectoryDiscovered(
				ctx,
				job.LibraryID,
				root,
				relativeDirectory,
			); err != nil {
				return err
			}
		}
	}
	return nil
}

func (processor *ReconcileProcessor) fail(
	ctx context.Context,
	job ReconcileJob,
	code string,
	cause error,
) error {
	if ctx.Err() != nil &&
		(errors.Is(cause, context.Canceled) ||
			errors.Is(cause, context.DeadlineExceeded)) {
		return cause
	}
	finishCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if err := processor.repository.FailDirectoryReconcile(finishCtx, job, code); err != nil {
		return errors.Join(cause, err)
	}
	if processor.observer != nil {
		if err := processor.observer.ReconcileFailed(
			finishCtx,
			job,
			code,
		); err != nil {
			return errors.Join(cause, err)
		}
	}
	return cause
}

func reconcileFailureCode(err error) string {
	switch {
	case errors.Is(err, ErrLibraryOffline),
		errors.Is(err, ErrPartialTreeUnreadable):
		return "source_unavailable"
	case errors.Is(err, ErrRootIdentityChanged),
		errors.Is(err, ErrInvalidRootIdentity):
		return "source_changed"
	default:
		return "internal_error"
	}
}

type ReconcileAdmission struct {
	repository ReconcileRepository
	waker      WakeNotifier
}

func NewReconcileAdmission(
	repository ReconcileRepository,
	waker WakeNotifier,
) (*ReconcileAdmission, error) {
	if repository == nil || waker == nil {
		return nil, errors.New("reconciliation admission dependencies are required")
	}
	return &ReconcileAdmission{repository: repository, waker: waker}, nil
}

func (service *ReconcileAdmission) MarkDirty(
	ctx context.Context,
	libraryID int64,
	relativeDirectory string,
) (ReconcileJob, error) {
	if libraryID <= 0 {
		return ReconcileJob{}, ErrInvalidReconcileTarget
	}
	normalized, err := pathpolicy.Normalize(relativeDirectory)
	if err != nil || normalized != relativeDirectory {
		return ReconcileJob{}, ErrInvalidReconcileTarget
	}
	job, err := service.repository.EnqueueReconcile(
		ctx,
		libraryID,
		relativeDirectory,
		ReconcileDebounce,
		ReconcileMaximumDebounce,
	)
	if err != nil {
		return ReconcileJob{}, err
	}
	service.waker.Wake()
	return job, nil
}
