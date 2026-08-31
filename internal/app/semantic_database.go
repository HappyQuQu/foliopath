package app

import (
	"context"
	"time"

	"github.com/HappyQuQu/foliopath/internal/jobs"
	"github.com/HappyQuQu/foliopath/internal/semantic"
)

func (service *databaseService) PutTagEmbeddingBatch(ctx context.Context, batch semantic.TagEmbeddingBatch) error {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return errAIRepositoryNotReady
	}
	return service.store.PutTagEmbeddingBatch(ctx, batch)
}

func (service *databaseService) GetActiveTagVocabulary(ctx context.Context) (semantic.TagVocabulary, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return semantic.TagVocabulary{}, errAIRepositoryNotReady
	}
	return service.store.GetActiveTagVocabulary(ctx)
}

func (service *databaseService) GetTagSuggestionListSnapshot(ctx context.Context, libraryID int64) (semantic.TagSuggestionListSnapshot, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return semantic.TagSuggestionListSnapshot{}, errAIRepositoryNotReady
	}
	return service.store.GetTagSuggestionListSnapshot(ctx, libraryID)
}

func (service *databaseService) ListTagSuggestionViews(ctx context.Context, query semantic.TagSuggestionListQuery) ([]semantic.TagSuggestionView, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return nil, errAIRepositoryNotReady
	}
	return service.store.ListTagSuggestionViews(ctx, query)
}

func (service *databaseService) PublishTagVocabulary(ctx context.Context, id string, revision int64, tagIDs []int64, now time.Time) (semantic.TagVocabulary, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return semantic.TagVocabulary{}, errAIRepositoryNotReady
	}
	return service.store.PublishTagVocabulary(ctx, id, revision, tagIDs, now)
}

func (service *databaseService) ReplacePendingTagSuggestions(ctx context.Context, libraryID, assetID int64, generationID, snapshotID, sourceFingerprint string, items []semantic.TagSuggestion, now time.Time) error {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return errAIRepositoryNotReady
	}
	return service.store.ReplacePendingTagSuggestions(ctx, libraryID, assetID, generationID, snapshotID, sourceFingerprint, items, now)
}

func (service *databaseService) GetTagSuggestion(ctx context.Context, id string) (semantic.TagSuggestion, bool, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return semantic.TagSuggestion{}, false, errAIRepositoryNotReady
	}
	return service.store.GetTagSuggestion(ctx, id)
}

func (service *databaseService) GetTagReviewBySuggestion(ctx context.Context, id string) (semantic.TagReview, bool, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return semantic.TagReview{}, false, errAIRepositoryNotReady
	}
	return service.store.GetTagReviewBySuggestion(ctx, id)
}

func (service *databaseService) BeginTagReviewRequest(ctx context.Context, value semantic.TagReviewRequestRecord) (semantic.TagReviewRequestRecord, bool, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return semantic.TagReviewRequestRecord{}, false, errAIRepositoryNotReady
	}
	return service.store.BeginTagReviewRequest(ctx, value)
}

func (service *databaseService) CommitTagReviewRequestOutcome(ctx context.Context, keyHash string, ordinal int, outcome semantic.TagReviewOutcome, now time.Time) error {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return errAIRepositoryNotReady
	}
	return service.store.CommitTagReviewRequestOutcome(ctx, keyHash, ordinal, outcome, now)
}

func (service *databaseService) CompleteTagReviewRequest(ctx context.Context, keyHash string, now time.Time) error {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return errAIRepositoryNotReady
	}
	return service.store.CompleteTagReviewRequest(ctx, keyHash, now)
}

func (service *databaseService) CommitTagReview(ctx context.Context, id string, revision int64, decision semantic.TagReviewDecision, curationRevision int64, now time.Time) (semantic.TagReview, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return semantic.TagReview{}, errAIRepositoryNotReady
	}
	return service.store.CommitTagReview(ctx, id, revision, decision, curationRevision, now)
}

func (service *databaseService) ReplaceVideoEmbeddingPlan(ctx context.Context, plan semantic.VideoEmbeddingPlan) error {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return errAIRepositoryNotReady
	}
	return service.store.ReplaceVideoEmbeddingPlan(ctx, plan)
}

func (service *databaseService) FindVideoJob(ctx context.Context, keyHash string) (semantic.VideoJobAdmission, bool, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return semantic.VideoJobAdmission{}, false, errAIRepositoryNotReady
	}
	return service.store.FindVideoJob(ctx, keyHash)
}

func (service *databaseService) CreateVideoJob(ctx context.Context, value semantic.VideoJobAdmission) (semantic.VideoJobAdmission, bool, bool, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return semantic.VideoJobAdmission{}, false, false, errAIRepositoryNotReady
	}
	return service.store.CreateVideoJob(ctx, value)
}

func (service *databaseService) ClaimVideoJob(ctx context.Context, now time.Time, lease time.Duration) (semantic.VideoJob, bool, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return semantic.VideoJob{}, false, errAIRepositoryNotReady
	}
	return service.store.ClaimVideoJob(ctx, now, lease)
}

func (service *databaseService) RefreshVideoJobLease(ctx context.Context, job semantic.VideoJob, now time.Time, lease time.Duration) (bool, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return false, errAIRepositoryNotReady
	}
	return service.store.RefreshVideoJobLease(ctx, job, now, lease)
}

func (service *databaseService) GetVideoJobProgress(ctx context.Context, generationID string, libraryID int64) (semantic.VideoJobProgress, bool, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return semantic.VideoJobProgress{}, false, errAIRepositoryNotReady
	}
	return service.store.GetVideoJobProgress(ctx, generationID, libraryID)
}

func (service *databaseService) CommitVideoJobProgress(ctx context.Context, value semantic.VideoJobProgressCommit) (semantic.VideoJobProgress, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return semantic.VideoJobProgress{}, errAIRepositoryNotReady
	}
	return service.store.CommitVideoJobProgress(ctx, value)
}

func (service *databaseService) CancelVideoJobOperation(ctx context.Context, operationID string, revision int64, now time.Time) (semantic.VideoJob, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return semantic.VideoJob{}, errAIRepositoryNotReady
	}
	return service.store.CancelVideoJobOperation(ctx, operationID, revision, now)
}

func (service *databaseService) FinishVideoJob(ctx context.Context, job semantic.VideoJob, state semantic.JobState, code string, now time.Time) (semantic.VideoJob, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return semantic.VideoJob{}, errAIRepositoryNotReady
	}
	return service.store.FinishVideoJob(ctx, job, state, code, now)
}

func (service *databaseService) RecoverExpiredVideoJobs(ctx context.Context, now time.Time) (jobs.RecoverySummary, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return jobs.RecoverySummary{}, errAIRepositoryNotReady
	}
	return service.store.RecoverExpiredVideoJobs(ctx, now)
}

func (service *databaseService) CountVideoJobCandidates(ctx context.Context, libraryID int64, generationID string, mode semantic.JobMode) (semantic.VideoJobCandidateCounts, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return semantic.VideoJobCandidateCounts{}, errAIRepositoryNotReady
	}
	return service.store.CountVideoJobCandidates(ctx, libraryID, generationID, mode)
}

func (service *databaseService) ListVideoJobCandidates(ctx context.Context, libraryID int64, generationID string, mode semantic.JobMode, checkpoint int64, limit int) (semantic.VideoJobCandidatePage, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return semantic.VideoJobCandidatePage{}, errAIRepositoryNotReady
	}
	return service.store.ListVideoJobCandidates(ctx, libraryID, generationID, mode, checkpoint, limit)
}

func (service *databaseService) SearchVideoSemanticVectors(ctx context.Context, request semantic.VideoVectorSearchRequest) ([]semantic.VideoVectorMatch, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return nil, errAIRepositoryNotReady
	}
	return service.store.SearchVideoSemanticVectors(ctx, request)
}

func (service *databaseService) GetVideoSemanticSearchSnapshot(ctx context.Context, scope semantic.SearchScope) (semantic.VideoSearchSnapshot, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return semantic.VideoSearchSnapshot{}, errAIRepositoryNotReady
	}
	return service.store.GetVideoSemanticSearchSnapshot(ctx, scope)
}

func (service *databaseService) FindSemanticBackfill(ctx context.Context, keyHash string) (semantic.BackfillAdmission, bool, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return semantic.BackfillAdmission{}, false, errAIRepositoryNotReady
	}
	return service.store.FindSemanticBackfill(ctx, keyHash)
}

func (service *databaseService) CreateSemanticBackfill(ctx context.Context, value semantic.BackfillAdmission) (semantic.BackfillAdmission, bool, bool, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return semantic.BackfillAdmission{}, false, false, errAIRepositoryNotReady
	}
	return service.store.CreateSemanticBackfill(ctx, value)
}

func (service *databaseService) ClaimSemanticBackfill(ctx context.Context, now time.Time, lease time.Duration) (semantic.BackfillJob, bool, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return semantic.BackfillJob{}, false, errAIRepositoryNotReady
	}
	return service.store.ClaimSemanticBackfill(ctx, now, lease)
}

func (service *databaseService) RefreshSemanticBackfillLease(ctx context.Context, job semantic.BackfillJob, now time.Time, lease time.Duration) (bool, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return false, errAIRepositoryNotReady
	}
	return service.store.RefreshSemanticBackfillLease(ctx, job, now, lease)
}

func (service *databaseService) CancelSemanticBackfill(ctx context.Context, jobID string, revision int64, now time.Time) (semantic.BackfillJob, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return semantic.BackfillJob{}, errAIRepositoryNotReady
	}
	return service.store.CancelSemanticBackfill(ctx, jobID, revision, now)
}

func (service *databaseService) CancelSemanticBackfillOperation(ctx context.Context, operationID string, revision int64, now time.Time) (semantic.BackfillJob, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return semantic.BackfillJob{}, errAIRepositoryNotReady
	}
	return service.store.CancelSemanticBackfillOperation(ctx, operationID, revision, now)
}

func (service *databaseService) FinishSemanticBackfill(ctx context.Context, job semantic.BackfillJob, state semantic.JobState, code string, now time.Time) (semantic.BackfillJob, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return semantic.BackfillJob{}, errAIRepositoryNotReady
	}
	return service.store.FinishSemanticBackfill(ctx, job, state, code, now)
}

func (service *databaseService) RecoverExpiredSemanticBackfills(ctx context.Context, now time.Time) (jobs.RecoverySummary, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return jobs.RecoverySummary{}, errAIRepositoryNotReady
	}
	return service.store.RecoverExpiredSemanticBackfills(ctx, now)
}

func (service *databaseService) CountSemanticBackfillCandidates(ctx context.Context, libraryID int64, generationID string, mode semantic.JobMode) (semantic.BackfillCandidateCounts, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return semantic.BackfillCandidateCounts{}, errAIRepositoryNotReady
	}
	return service.store.CountSemanticBackfillCandidates(ctx, libraryID, generationID, mode)
}

func (service *databaseService) ListSemanticBackfillCandidates(ctx context.Context, libraryID int64, generationID string, mode semantic.JobMode, checkpoint int64, limit int) (semantic.BackfillCandidatePage, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return semantic.BackfillCandidatePage{}, errAIRepositoryNotReady
	}
	return service.store.ListSemanticBackfillCandidates(ctx, libraryID, generationID, mode, checkpoint, limit)
}

func (service *databaseService) PutSemanticEmbeddingBatch(ctx context.Context, batch semantic.EmbeddingBatch) error {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return errAIRepositoryNotReady
	}
	return service.store.PutSemanticEmbeddingBatch(ctx, batch)
}

func (service *databaseService) GetSemanticEmbedding(ctx context.Context, generationID string, libraryID, assetID int64) (semantic.StoredEmbedding, bool, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return semantic.StoredEmbedding{}, false, errAIRepositoryNotReady
	}
	return service.store.GetSemanticEmbedding(ctx, generationID, libraryID, assetID)
}

func (service *databaseService) DeleteSemanticEmbeddingIfSourceChanged(ctx context.Context, generationID string, libraryID, assetID int64, fingerprint string) (bool, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return false, errAIRepositoryNotReady
	}
	return service.store.DeleteSemanticEmbeddingIfSourceChanged(ctx, generationID, libraryID, assetID, fingerprint)
}

func (service *databaseService) GetSemanticEmbeddingProgress(ctx context.Context, generationID string, libraryID int64) (semantic.EmbeddingProgress, bool, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return semantic.EmbeddingProgress{}, false, errAIRepositoryNotReady
	}
	return service.store.GetSemanticEmbeddingProgress(ctx, generationID, libraryID)
}

func (service *databaseService) CommitSemanticEmbeddingProgress(ctx context.Context, commit semantic.EmbeddingProgressCommit) (semantic.EmbeddingProgress, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return semantic.EmbeddingProgress{}, errAIRepositoryNotReady
	}
	return service.store.CommitSemanticEmbeddingProgress(ctx, commit)
}

func (service *databaseService) GetSemanticGenerationRuntime(ctx context.Context, generationID string) (semantic.GenerationRuntime, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return semantic.GenerationRuntime{}, errAIRepositoryNotReady
	}
	return service.store.GetSemanticGenerationRuntime(ctx, generationID)
}

func (service *databaseService) GetSemanticLibrarySettings(ctx context.Context, libraryID int64) (semantic.LibrarySettings, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return semantic.LibrarySettings{}, errAIRepositoryNotReady
	}
	return service.store.GetSemanticLibrarySettings(ctx, libraryID)
}

func (service *databaseService) UpdateSemanticLibrarySettings(ctx context.Context, libraryID int64, enabled bool, revision int64, now time.Time) (semantic.LibrarySettings, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return semantic.LibrarySettings{}, errAIRepositoryNotReady
	}
	return service.store.UpdateSemanticLibrarySettings(ctx, libraryID, enabled, revision, now)
}

func (service *databaseService) SearchSemanticVectors(ctx context.Context, request semantic.VectorSearchRequest) ([]semantic.VectorMatch, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return nil, errAIRepositoryNotReady
	}
	return service.store.SearchSemanticVectors(ctx, request)
}

func (service *databaseService) FindSemanticClear(ctx context.Context, keyHash string) (semantic.ClearAdmission, bool, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return semantic.ClearAdmission{}, false, errAIRepositoryNotReady
	}
	return service.store.FindSemanticClear(ctx, keyHash)
}

func (service *databaseService) CreateSemanticClear(ctx context.Context, value semantic.ClearAdmission) (semantic.ClearAdmission, bool, bool, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return semantic.ClearAdmission{}, false, false, errAIRepositoryNotReady
	}
	return service.store.CreateSemanticClear(ctx, value)
}

func (service *databaseService) ClaimSemanticClear(ctx context.Context, now time.Time, lease time.Duration) (semantic.ClearJob, bool, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return semantic.ClearJob{}, false, errAIRepositoryNotReady
	}
	return service.store.ClaimSemanticClear(ctx, now, lease)
}

func (service *databaseService) RefreshSemanticClearLease(ctx context.Context, job semantic.ClearJob, now time.Time, lease time.Duration) (bool, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return false, errAIRepositoryNotReady
	}
	return service.store.RefreshSemanticClearLease(ctx, job, now, lease)
}

func (service *databaseService) DeleteSemanticClearBatch(ctx context.Context, job semantic.ClearJob, limit int, now time.Time) (int64, bool, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return 0, false, errAIRepositoryNotReady
	}
	return service.store.DeleteSemanticClearBatch(ctx, job, limit, now)
}

func (service *databaseService) FinishSemanticClear(ctx context.Context, job semantic.ClearJob, state semantic.JobState, code string, now time.Time) (semantic.ClearJob, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return semantic.ClearJob{}, errAIRepositoryNotReady
	}
	return service.store.FinishSemanticClear(ctx, job, state, code, now)
}

func (service *databaseService) CancelSemanticClearOperation(ctx context.Context, operationID string, revision int64, now time.Time) (semantic.ClearJob, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return semantic.ClearJob{}, errAIRepositoryNotReady
	}
	return service.store.CancelSemanticClearOperation(ctx, operationID, revision, now)
}

func (service *databaseService) RecoverExpiredSemanticClears(ctx context.Context, now time.Time) (jobs.RecoverySummary, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return jobs.RecoverySummary{}, errAIRepositoryNotReady
	}
	return service.store.RecoverExpiredSemanticClears(ctx, now)
}

func (service *databaseService) FindTagReviewClear(ctx context.Context, keyHash string) (semantic.TagReviewClearAdmission, bool, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return semantic.TagReviewClearAdmission{}, false, errAIRepositoryNotReady
	}
	return service.store.FindTagReviewClear(ctx, keyHash)
}

func (service *databaseService) CreateTagReviewClear(ctx context.Context, value semantic.TagReviewClearAdmission) (semantic.TagReviewClearAdmission, bool, bool, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return semantic.TagReviewClearAdmission{}, false, false, errAIRepositoryNotReady
	}
	return service.store.CreateTagReviewClear(ctx, value)
}

func (service *databaseService) ClaimTagReviewClear(ctx context.Context, now time.Time, lease time.Duration) (semantic.TagReviewClearJob, bool, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return semantic.TagReviewClearJob{}, false, errAIRepositoryNotReady
	}
	return service.store.ClaimTagReviewClear(ctx, now, lease)
}

func (service *databaseService) RefreshTagReviewClearLease(ctx context.Context, job semantic.TagReviewClearJob, now time.Time, lease time.Duration) (bool, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return false, errAIRepositoryNotReady
	}
	return service.store.RefreshTagReviewClearLease(ctx, job, now, lease)
}

func (service *databaseService) DeleteTagReviewClearBatch(ctx context.Context, job semantic.TagReviewClearJob, limit int, now time.Time) (int64, bool, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return 0, false, errAIRepositoryNotReady
	}
	return service.store.DeleteTagReviewClearBatch(ctx, job, limit, now)
}

func (service *databaseService) FinishTagReviewClear(ctx context.Context, job semantic.TagReviewClearJob, state semantic.JobState, code string, now time.Time) (semantic.TagReviewClearJob, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return semantic.TagReviewClearJob{}, errAIRepositoryNotReady
	}
	return service.store.FinishTagReviewClear(ctx, job, state, code, now)
}

func (service *databaseService) CancelTagReviewClearOperation(ctx context.Context, operationID string, revision int64, now time.Time) (semantic.TagReviewClearJob, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return semantic.TagReviewClearJob{}, errAIRepositoryNotReady
	}
	return service.store.CancelTagReviewClearOperation(ctx, operationID, revision, now)
}

func (service *databaseService) RecoverExpiredTagReviewClears(ctx context.Context, now time.Time) (jobs.RecoverySummary, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return jobs.RecoverySummary{}, errAIRepositoryNotReady
	}
	return service.store.RecoverExpiredTagReviewClears(ctx, now)
}

func (service *databaseService) FindTagJob(ctx context.Context, key string) (semantic.TagJobAdmission, bool, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return semantic.TagJobAdmission{}, false, errAIRepositoryNotReady
	}
	return service.store.FindTagJob(ctx, key)
}
func (service *databaseService) CreateTagJob(ctx context.Context, value semantic.TagJobAdmission) (semantic.TagJobAdmission, bool, bool, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return semantic.TagJobAdmission{}, false, false, errAIRepositoryNotReady
	}
	return service.store.CreateTagJob(ctx, value)
}
func (service *databaseService) ClaimTagJob(ctx context.Context, now time.Time, lease time.Duration) (semantic.TagJob, bool, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return semantic.TagJob{}, false, errAIRepositoryNotReady
	}
	return service.store.ClaimTagJob(ctx, now, lease)
}
func (service *databaseService) RefreshTagJobLease(ctx context.Context, job semantic.TagJob, now time.Time, lease time.Duration) (bool, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return false, errAIRepositoryNotReady
	}
	return service.store.RefreshTagJobLease(ctx, job, now, lease)
}
func (service *databaseService) GetTagJobProgress(ctx context.Context, generation string, library int64, snapshot string) (semantic.TagJobProgress, bool, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return semantic.TagJobProgress{}, false, errAIRepositoryNotReady
	}
	return service.store.GetTagJobProgress(ctx, generation, library, snapshot)
}
func (service *databaseService) CommitTagJobProgress(ctx context.Context, value semantic.TagJobProgressCommit) (semantic.TagJobProgress, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return semantic.TagJobProgress{}, errAIRepositoryNotReady
	}
	return service.store.CommitTagJobProgress(ctx, value)
}
func (service *databaseService) CancelTagJobOperation(ctx context.Context, id string, revision int64, now time.Time) (semantic.TagJob, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return semantic.TagJob{}, errAIRepositoryNotReady
	}
	return service.store.CancelTagJobOperation(ctx, id, revision, now)
}
func (service *databaseService) FinishTagJob(ctx context.Context, job semantic.TagJob, state semantic.JobState, code string, now time.Time) (semantic.TagJob, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return semantic.TagJob{}, errAIRepositoryNotReady
	}
	return service.store.FinishTagJob(ctx, job, state, code, now)
}
func (service *databaseService) RecoverExpiredTagJobs(ctx context.Context, now time.Time) (jobs.RecoverySummary, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return jobs.RecoverySummary{}, errAIRepositoryNotReady
	}
	return service.store.RecoverExpiredTagJobs(ctx, now)
}
func (service *databaseService) CountTagJobCandidates(ctx context.Context, library int64, generation, snapshot string, mode semantic.JobMode) (semantic.TagJobCandidateCounts, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return semantic.TagJobCandidateCounts{}, errAIRepositoryNotReady
	}
	return service.store.CountTagJobCandidates(ctx, library, generation, snapshot, mode)
}
func (service *databaseService) ListTagJobCandidates(ctx context.Context, library int64, generation, snapshot string, mode semantic.JobMode, checkpoint int64, limit int) (semantic.TagJobCandidatePage, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return semantic.TagJobCandidatePage{}, errAIRepositoryNotReady
	}
	return service.store.ListTagJobCandidates(ctx, library, generation, snapshot, mode, checkpoint, limit)
}
func (service *databaseService) LoadTagSuggestionInputs(ctx context.Context, generation, snapshot string, library, asset int64) (string, []float32, map[int64][]float32, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return "", nil, nil, errAIRepositoryNotReady
	}
	return service.store.LoadTagSuggestionInputs(ctx, generation, snapshot, library, asset)
}
func (service *databaseService) ListMissingTagEmbeddingInputs(ctx context.Context, generation, snapshot string, limit int) ([]semantic.TagEmbeddingInput, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return nil, errAIRepositoryNotReady
	}
	return service.store.ListMissingTagEmbeddingInputs(ctx, generation, snapshot, limit)
}

func (service *databaseService) GetSemanticSearchSnapshot(ctx context.Context, scope semantic.SearchScope) (semantic.SearchSnapshot, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return semantic.SearchSnapshot{}, errAIRepositoryNotReady
	}
	return service.store.GetSemanticSearchSnapshot(ctx, scope)
}

var _ semantic.BackfillQueue = (*databaseService)(nil)
var _ semantic.VideoJobQueue = (*databaseService)(nil)
var _ semantic.VideoJobCatalog = (*databaseService)(nil)
var _ semantic.ClearQueue = (*databaseService)(nil)
var _ semantic.TagReviewClearQueue = (*databaseService)(nil)
var _ semantic.TagJobQueue = (*databaseService)(nil)
var _ semantic.TagJobCatalog = (*databaseService)(nil)
var _ semantic.TagSuggestionInputSource = (*databaseService)(nil)
var _ semantic.TagEmbeddingInputRepository = (*databaseService)(nil)
var _ semantic.BackfillCatalog = (*databaseService)(nil)
var _ semantic.EmbeddingRepository = (*databaseService)(nil)
var _ semantic.GenerationRuntimeRepository = (*databaseService)(nil)
var _ semantic.LibrarySettingsRepository = (*databaseService)(nil)
var _ semantic.VectorSearchRepository = (*databaseService)(nil)
var _ semantic.SearchSnapshotRepository = (*databaseService)(nil)
