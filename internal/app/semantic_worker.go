package app

import (
	"context"
	"errors"
	"time"

	"github.com/HappyQuQu/foliopath/internal/jobs"
	"github.com/HappyQuQu/foliopath/internal/media"
	"github.com/HappyQuQu/foliopath/internal/semantic"
)

type semanticJobQueue struct {
	database *databaseService
	now      func() time.Time
}

type semanticClearQueue struct {
	database *databaseService
	now      func() time.Time
}

type tagReviewClearQueue struct {
	database *databaseService
	now      func() time.Time
}

type tagJobQueue struct {
	database *databaseService
	now      func() time.Time
}

type videoJobQueue struct {
	database *databaseService
	now      func() time.Time
}

func (queue videoJobQueue) RecoverExpired(ctx context.Context) (jobs.RecoverySummary, error) {
	return queue.database.RecoverExpiredVideoJobs(ctx, queue.clock().UTC())
}
func (queue videoJobQueue) Claim(ctx context.Context, lease time.Duration) (semantic.VideoJob, bool, error) {
	return queue.database.ClaimVideoJob(ctx, queue.clock().UTC(), lease)
}
func (queue videoJobQueue) RefreshLease(ctx context.Context, job semantic.VideoJob, lease time.Duration) (bool, error) {
	return queue.database.RefreshVideoJobLease(ctx, job, queue.clock().UTC(), lease)
}
func (queue videoJobQueue) clock() time.Time {
	if queue.now == nil {
		return time.Now()
	}
	return queue.now()
}

func (queue tagJobQueue) RecoverExpired(ctx context.Context) (jobs.RecoverySummary, error) {
	return queue.database.RecoverExpiredTagJobs(ctx, queue.clock().UTC())
}
func (queue tagJobQueue) Claim(ctx context.Context, lease time.Duration) (semantic.TagJob, bool, error) {
	return queue.database.ClaimTagJob(ctx, queue.clock().UTC(), lease)
}
func (queue tagJobQueue) RefreshLease(ctx context.Context, job semantic.TagJob, lease time.Duration) (bool, error) {
	return queue.database.RefreshTagJobLease(ctx, job, queue.clock().UTC(), lease)
}
func (queue tagJobQueue) clock() time.Time {
	if queue.now == nil {
		return time.Now()
	}
	return queue.now()
}

func (queue tagReviewClearQueue) RecoverExpired(ctx context.Context) (jobs.RecoverySummary, error) {
	return queue.database.RecoverExpiredTagReviewClears(ctx, queue.clock().UTC())
}

func (queue tagReviewClearQueue) Claim(ctx context.Context, lease time.Duration) (semantic.TagReviewClearJob, bool, error) {
	return queue.database.ClaimTagReviewClear(ctx, queue.clock().UTC(), lease)
}

func (queue tagReviewClearQueue) RefreshLease(ctx context.Context, job semantic.TagReviewClearJob, lease time.Duration) (bool, error) {
	return queue.database.RefreshTagReviewClearLease(ctx, job, queue.clock().UTC(), lease)
}

func (queue tagReviewClearQueue) clock() time.Time {
	if queue.now == nil {
		return time.Now()
	}
	return queue.now()
}

func (queue semanticClearQueue) RecoverExpired(ctx context.Context) (jobs.RecoverySummary, error) {
	return queue.database.RecoverExpiredSemanticClears(ctx, queue.clock().UTC())
}

func (queue semanticClearQueue) Claim(ctx context.Context, lease time.Duration) (semantic.ClearJob, bool, error) {
	return queue.database.ClaimSemanticClear(ctx, queue.clock().UTC(), lease)
}

func (queue semanticClearQueue) RefreshLease(ctx context.Context, job semantic.ClearJob, lease time.Duration) (bool, error) {
	return queue.database.RefreshSemanticClearLease(ctx, job, queue.clock().UTC(), lease)
}

func (queue semanticClearQueue) clock() time.Time {
	if queue.now == nil {
		return time.Now()
	}
	return queue.now()
}

func (queue semanticJobQueue) RecoverExpired(ctx context.Context) (jobs.RecoverySummary, error) {
	return queue.database.RecoverExpiredSemanticBackfills(ctx, queue.clock().UTC())
}

func (queue semanticJobQueue) Claim(ctx context.Context, lease time.Duration) (semantic.BackfillJob, bool, error) {
	return queue.database.ClaimSemanticBackfill(ctx, queue.clock().UTC(), lease)
}

func (queue semanticJobQueue) RefreshLease(ctx context.Context, job semantic.BackfillJob, lease time.Duration) (bool, error) {
	return queue.database.RefreshSemanticBackfillLease(ctx, job, queue.clock().UTC(), lease)
}

func (queue semanticJobQueue) clock() time.Time {
	if queue.now == nil {
		return time.Now()
	}
	return queue.now()
}

type semanticContentSource struct{ content *media.ContentService }

func (source semanticContentSource) OpenSemanticAsset(ctx context.Context, libraryID, assetID int64) (semantic.SemanticAsset, error) {
	if source.content == nil || libraryID < 1 || assetID < 1 {
		return semantic.SemanticAsset{}, semantic.ErrSemanticSourceUnavailable
	}
	content, err := source.content.Open(ctx, assetID)
	if err != nil {
		switch {
		case errors.Is(err, media.ErrContentSourceChanged), errors.Is(err, media.ErrContentAssetNotFound):
			return semantic.SemanticAsset{}, semantic.ErrSemanticSourceChanged
		default:
			return semantic.SemanticAsset{}, errors.Join(semantic.ErrSemanticSourceUnavailable, err)
		}
	}
	if content.LibraryID != libraryID {
		_ = content.File.Close()
		return semantic.SemanticAsset{}, semantic.ErrSemanticSourceChanged
	}
	_, format, _, ok := media.ClassifyPath(content.Name)
	if !ok {
		_ = content.File.Close()
		return semantic.SemanticAsset{}, semantic.ErrSemanticSourceChanged
	}
	return semantic.SemanticAsset{File: content.File, Format: format,
		SourceFingerprint: content.Fingerprint.String()}, nil
}

var _ jobs.LeaseQueue[semantic.BackfillJob] = semanticJobQueue{}
var _ jobs.LeaseQueue[semantic.ClearJob] = semanticClearQueue{}
var _ jobs.LeaseQueue[semantic.TagReviewClearJob] = tagReviewClearQueue{}
var _ jobs.LeaseQueue[semantic.TagJob] = tagJobQueue{}
var _ jobs.LeaseQueue[semantic.VideoJob] = videoJobQueue{}
var _ semantic.BackfillAssetSource = semanticContentSource{}
