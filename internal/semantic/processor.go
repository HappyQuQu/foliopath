package semantic

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/HappyQuQu/foliopath/internal/media"
)

const DefaultBackfillPageSize = 16

var (
	ErrSemanticSourceChanged     = errors.New("semantic source changed")
	ErrSemanticSourceUnavailable = errors.New("semantic source unavailable")
	ErrImageEncoderUnavailable   = errors.New("semantic image encoder unavailable")
)

type SemanticAsset struct {
	File              io.ReadSeekCloser
	Format            media.Format
	SourceFingerprint string
}

type BackfillAssetSource interface {
	OpenSemanticAsset(context.Context, int64, int64) (SemanticAsset, error)
}

type ImagePreprocessor interface {
	PrepareSemanticImage(context.Context, io.ReadSeeker, media.Format) ([]float32, error)
}

type ImageEncoder interface {
	EncodeSemanticImage(context.Context, string, []float32) ([]float32, error)
}

type BackfillProcessor struct {
	catalog      BackfillCatalog
	assets       BackfillAssetSource
	preprocessor ImagePreprocessor
	encoder      ImageEncoder
	embeddings   EmbeddingRepository
	queue        BackfillQueue
	now          func() time.Time
	pageSize     int
}

func NewBackfillProcessor(
	catalog BackfillCatalog,
	assets BackfillAssetSource,
	preprocessor ImagePreprocessor,
	encoder ImageEncoder,
	embeddings EmbeddingRepository,
	queue BackfillQueue,
	now func() time.Time,
	pageSize int,
) (*BackfillProcessor, error) {
	if catalog == nil || assets == nil || preprocessor == nil || encoder == nil || embeddings == nil || queue == nil {
		return nil, errors.New("semantic backfill processor dependencies are required")
	}
	if now == nil {
		now = time.Now
	}
	if pageSize == 0 {
		pageSize = DefaultBackfillPageSize
	}
	if pageSize < 1 || pageSize > 1000 {
		return nil, ErrInvalidSemanticJob
	}
	return &BackfillProcessor{catalog: catalog, assets: assets, preprocessor: preprocessor,
		encoder: encoder, embeddings: embeddings, queue: queue, now: now, pageSize: pageSize}, nil
}

func (processor *BackfillProcessor) Process(ctx context.Context, job BackfillJob) error {
	if job.ID == "" || job.LibraryID < 1 || job.GenerationID == "" || job.OperationID == "" ||
		job.State != JobRunning || job.ClaimedRevision < 1 || job.TotalItems < job.CompletedItems {
		return ErrInvalidSemanticJob
	}
	progress, found, err := processor.embeddings.GetSemanticEmbeddingProgress(ctx, job.GenerationID, job.LibraryID)
	if err != nil || !found {
		return processor.fail(ctx, job, "internal_error", firstNonNil(err, ErrSemanticProgressConflict))
	}
	checkpoint := job.CheckpointID
	completed := job.CompletedItems
	for completed < job.TotalItems {
		if err := ctx.Err(); err != nil {
			return err
		}
		limit := min(processor.pageSize, int(job.TotalItems-completed))
		page, err := processor.catalog.ListSemanticBackfillCandidates(ctx, job.LibraryID, job.GenerationID, job.Mode, checkpoint, limit)
		if err != nil {
			return processor.fail(ctx, job, semanticProcessorErrorCode(err), err)
		}
		if len(page.Items) == 0 {
			remaining := job.TotalItems - completed
			progress, err = processor.commit(ctx, job, progress, checkpoint+1, nil, 0, remaining)
			if err != nil {
				return processor.fail(ctx, job, semanticProcessorErrorCode(err), err)
			}
			completed += remaining
			checkpoint++
			break
		}

		items := make([]EmbeddingItem, 0, len(page.Items))
		var failed, stale int64
		for _, candidate := range page.Items {
			item, outcome, err := processor.processCandidate(ctx, job, candidate)
			if err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				if errors.Is(err, ErrImageEncoderUnavailable) || errors.Is(err, ErrSemanticGenerationUnavailable) {
					return processor.fail(ctx, job, semanticProcessorErrorCode(err), err)
				}
			}
			switch outcome {
			case candidateCompleted:
				items = append(items, item)
			case candidateStale:
				stale++
			default:
				failed++
			}
		}
		progress, err = processor.commit(ctx, job, progress, page.Checkpoint, items, failed, stale)
		if err != nil {
			return processor.fail(ctx, job, semanticProcessorErrorCode(err), err)
		}
		processed := int64(len(items)) + failed + stale
		completed += processed
		checkpoint = page.Checkpoint
	}
	_, err = processor.queue.FinishSemanticBackfill(ctx, job, JobSucceeded, "", processor.now().UTC())
	return err
}

type candidateOutcome uint8

const (
	candidateFailed candidateOutcome = iota
	candidateCompleted
	candidateStale
)

func (processor *BackfillProcessor) processCandidate(ctx context.Context, job BackfillJob, candidate BackfillCandidate) (EmbeddingItem, candidateOutcome, error) {
	asset, err := processor.assets.OpenSemanticAsset(ctx, job.LibraryID, candidate.AssetID)
	if err != nil {
		if errors.Is(err, ErrSemanticSourceChanged) {
			return EmbeddingItem{}, candidateStale, err
		}
		return EmbeddingItem{}, candidateFailed, err
	}
	if asset.File == nil {
		return EmbeddingItem{}, candidateFailed, ErrSemanticSourceUnavailable
	}
	defer asset.File.Close()
	if asset.SourceFingerprint != candidate.SourceFingerprint {
		return EmbeddingItem{}, candidateStale, ErrSemanticSourceChanged
	}
	tensor, err := processor.preprocessor.PrepareSemanticImage(ctx, asset.File, asset.Format)
	if err != nil {
		return EmbeddingItem{}, candidateFailed, err
	}
	vector, err := processor.encoder.EncodeSemanticImage(ctx, job.GenerationID, tensor)
	if err != nil {
		return EmbeddingItem{}, candidateFailed, err
	}
	encoded, err := EncodeEmbedding(vector, len(vector))
	if err != nil {
		return EmbeddingItem{}, candidateFailed, err
	}
	return EmbeddingItem{AssetID: candidate.AssetID, SourceFingerprint: candidate.SourceFingerprint,
		Vector: encoded, CreatedAt: processor.now().UTC()}, candidateCompleted, nil
}

func (processor *BackfillProcessor) commit(
	ctx context.Context,
	job BackfillJob,
	progress EmbeddingProgress,
	nextCheckpoint int64,
	items []EmbeddingItem,
	failed, stale int64,
) (EmbeddingProgress, error) {
	return processor.embeddings.CommitSemanticEmbeddingProgress(ctx, EmbeddingProgressCommit{
		JobID: job.ID, ClaimedRevision: job.ClaimedRevision,
		ExpectedProgressRevision: progress.Revision,
		ExpectedCheckpointID:     progress.CheckpointID,
		NextCheckpointID:         nextCheckpoint,
		Batch:                    EmbeddingBatch{GenerationID: job.GenerationID, LibraryID: job.LibraryID, Items: items},
		FailedCount:              failed, StaleCount: stale, UpdatedAt: processor.now().UTC(),
	})
}

func (processor *BackfillProcessor) fail(ctx context.Context, job BackfillJob, code string, cause error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	_, finishErr := processor.queue.FinishSemanticBackfill(ctx, job, JobFailed, code, processor.now().UTC())
	return errors.Join(cause, finishErr)
}

func semanticProcessorErrorCode(err error) string {
	switch {
	case errors.Is(err, ErrImageEncoderUnavailable), errors.Is(err, ErrSemanticGenerationUnavailable):
		return "model_unavailable"
	case errors.Is(err, ErrSemanticSourceUnavailable):
		return "source_unreadable"
	default:
		return "internal_error"
	}
}

func firstNonNil(values ...error) error {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}
