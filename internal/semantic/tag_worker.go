package semantic

import (
	"context"
	"errors"
	"time"
)

type TagPlanBuilder interface {
	BuildTagPlan(context.Context, string, string, int64, int64) (TagSuggestionPlan, error)
}

type TagEmbeddingBuilder interface {
	EnsureTagEmbeddings(context.Context, string, string) error
}

type TagJobProcessor struct {
	catalog    TagJobCatalog
	queue      TagJobQueue
	builder    TagPlanBuilder
	embeddings TagEmbeddingBuilder
	now        func() time.Time
}

func NewTagJobProcessor(catalog TagJobCatalog, queue TagJobQueue, builder TagPlanBuilder, now func() time.Time, embeddings ...TagEmbeddingBuilder) (*TagJobProcessor, error) {
	if catalog == nil || queue == nil || builder == nil {
		return nil, errors.New("tag worker dependencies are required")
	}
	if now == nil {
		now = time.Now
	}
	var embeddingBuilder TagEmbeddingBuilder
	if len(embeddings) > 1 {
		return nil, errors.New("only one tag embedding builder is allowed")
	}
	if len(embeddings) == 1 {
		if embeddings[0] == nil {
			return nil, errors.New("tag embedding builder is invalid")
		}
		embeddingBuilder = embeddings[0]
	}
	return &TagJobProcessor{catalog: catalog, queue: queue, builder: builder, embeddings: embeddingBuilder, now: now}, nil
}

func (p *TagJobProcessor) Process(ctx context.Context, job TagJob) error {
	if job.ID == "" || job.LibraryID < 1 || len(job.GenerationID) < 8 || len(job.VocabularySnapshotID) < 8 || job.State != JobRunning || job.ClaimedRevision < 1 {
		return ErrInvalidTagJob
	}
	if p.embeddings != nil {
		if err := p.embeddings.EnsureTagEmbeddings(ctx, job.GenerationID, job.VocabularySnapshotID); err != nil {
			return p.fail(ctx, job, "model_unavailable", err)
		}
	}
	progress, found, err := p.queue.GetTagJobProgress(ctx, job.GenerationID, job.LibraryID, job.VocabularySnapshotID)
	if err != nil || !found {
		return p.fail(ctx, job, "internal_error", errors.Join(err, ErrTagJobConflict))
	}
	checkpoint, completed := job.CheckpointID, job.CompletedItems
	for completed < job.TotalItems {
		if err := ctx.Err(); err != nil {
			return err
		}
		page, err := p.catalog.ListTagJobCandidates(ctx, job.LibraryID, job.GenerationID, job.VocabularySnapshotID, job.Mode, checkpoint, 1)
		if err != nil {
			return p.fail(ctx, job, "internal_error", err)
		}
		if len(page.Items) != 1 || page.Checkpoint <= checkpoint {
			return p.fail(ctx, job, "source_changed", ErrTagJobConflict)
		}
		candidate := page.Items[0]
		plan, buildErr := p.builder.BuildTagPlan(ctx, job.GenerationID, job.VocabularySnapshotID, job.LibraryID, candidate.AssetID)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		commit := TagJobProgressCommit{JobID: job.ID, ClaimedRevision: job.ClaimedRevision, ExpectedProgressRevision: progress.Revision, ExpectedCheckpointID: checkpoint, NextCheckpointID: candidate.AssetID, UpdatedAt: p.now().UTC()}
		switch {
		case buildErr == nil:
			commit.Plan = &plan
		case errors.Is(buildErr, ErrSemanticSourceChanged):
			commit.StaleCount = 1
		case errors.Is(buildErr, ErrSemanticGenerationUnavailable):
			return p.fail(ctx, job, "model_unavailable", buildErr)
		default:
			commit.FailedCount = 1
		}
		progress, err = p.queue.CommitTagJobProgress(ctx, commit)
		if err != nil {
			return p.fail(ctx, job, "internal_error", errors.Join(buildErr, err))
		}
		checkpoint, completed = candidate.AssetID, completed+1
	}
	_, err = p.queue.FinishTagJob(ctx, job, JobSucceeded, "", p.now().UTC())
	return err
}

func (p *TagJobProcessor) fail(ctx context.Context, job TagJob, code string, cause error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	_, finishErr := p.queue.FinishTagJob(ctx, job, JobFailed, code, p.now().UTC())
	return errors.Join(cause, finishErr)
}
