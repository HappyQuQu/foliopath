package semantic

import (
	"context"
	"errors"
	"time"
)

type VideoPlanBuilder interface {
	BuildPlan(context.Context, string, int64, int64) (VideoEmbeddingPlan, error)
}

type VideoJobProcessor struct {
	catalog VideoJobCatalog
	queue   VideoJobQueue
	builder VideoPlanBuilder
	now     func() time.Time
}

func NewVideoJobProcessor(catalog VideoJobCatalog, queue VideoJobQueue, builder VideoPlanBuilder, now func() time.Time) (*VideoJobProcessor, error) {
	if catalog == nil || queue == nil || builder == nil {
		return nil, errors.New("video semantic worker dependencies are required")
	}
	if now == nil {
		now = time.Now
	}
	return &VideoJobProcessor{catalog: catalog, queue: queue, builder: builder, now: now}, nil
}

func (processor *VideoJobProcessor) Process(ctx context.Context, job VideoJob) error {
	if job.ID == "" || job.LibraryID < 1 || len(job.GenerationID) < 8 || job.OperationID == "" ||
		job.State != JobRunning || job.ClaimedRevision < 1 || job.CompletedItems > job.TotalItems {
		return ErrInvalidVideoJob
	}
	progress, found, err := processor.queue.GetVideoJobProgress(ctx, job.GenerationID, job.LibraryID)
	if err != nil || !found {
		return processor.fail(ctx, job, "internal_error", firstNonNil(err, ErrVideoJobConflict))
	}
	checkpoint, completed := job.CheckpointID, job.CompletedItems
	for completed < job.TotalItems {
		if err := ctx.Err(); err != nil {
			return err
		}
		page, err := processor.catalog.ListVideoJobCandidates(ctx, job.LibraryID, job.GenerationID, job.Mode, checkpoint, 1)
		if err != nil {
			return processor.fail(ctx, job, videoJobErrorCode(err), err)
		}
		if len(page.Items) != 1 || page.Checkpoint <= checkpoint {
			return processor.fail(ctx, job, "source_changed", ErrVideoJobConflict)
		}
		candidate := page.Items[0]
		plan, buildErr := processor.builder.BuildPlan(ctx, job.GenerationID, job.LibraryID, candidate.AssetID)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		commit := VideoJobProgressCommit{JobID: job.ID, ClaimedRevision: job.ClaimedRevision,
			ExpectedProgressRevision: progress.Revision, ExpectedCheckpointID: checkpoint,
			NextCheckpointID: candidate.AssetID, UpdatedAt: processor.now().UTC()}
		switch {
		case buildErr == nil:
			commit.Plan = &plan
		case errors.Is(buildErr, ErrSemanticGenerationUnavailable), errors.Is(buildErr, ErrImageEncoderUnavailable):
			return processor.fail(ctx, job, "model_unavailable", buildErr)
		case errors.Is(buildErr, ErrStoryboardSourceChanged):
			commit.StaleCount = 1
		default:
			// Missing storyboards and any partial-frame failure are retryable
			// degraded outcomes. No incomplete plan is handed to the store.
			commit.DegradedCount = 1
		}
		progress, err = processor.queue.CommitVideoJobProgress(ctx, commit)
		if err != nil {
			return processor.fail(ctx, job, videoJobErrorCode(err), errors.Join(buildErr, err))
		}
		checkpoint, completed = candidate.AssetID, completed+1
	}
	_, err = processor.queue.FinishVideoJob(ctx, job, JobSucceeded, "", processor.now().UTC())
	return err
}

func (processor *VideoJobProcessor) fail(ctx context.Context, job VideoJob, code string, cause error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	_, finishErr := processor.queue.FinishVideoJob(ctx, job, JobFailed, code, processor.now().UTC())
	return errors.Join(cause, finishErr)
}

func videoJobErrorCode(err error) string {
	switch {
	case errors.Is(err, ErrSemanticGenerationUnavailable), errors.Is(err, ErrImageEncoderUnavailable):
		return "model_unavailable"
	case errors.Is(err, ErrStoryboardSourceChanged):
		return "source_changed"
	default:
		return "internal_error"
	}
}

var _ VideoPlanBuilder = (*VideoProcessor)(nil)
