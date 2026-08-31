package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/curation"
	"github.com/HappyQuQu/foliopath/internal/semantic"
)

func TestTagReviewClearIsRevisionGuardedIdempotentAndPreservesCuration(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	generationID := seedEmbeddingGeneration(t, store, 2)
	assetID := catalogAssetID(t, store, "photo-10.jpg")
	now := time.Date(2026, 8, 30, 1, 0, 0, 0, time.UTC)
	tag, _, err := store.CreateTag(context.Background(), "clear-test", "clear-test", now)
	if err != nil {
		t.Fatal(err)
	}
	const snapshotID = "aiv_clear_test"
	if _, err := store.db.ExecContext(context.Background(), `UPDATE ai_tag_vocabulary_snapshots SET state='retired' WHERE state='active'`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(context.Background(), `INSERT INTO ai_tag_vocabulary_snapshots(id,revision,state,created_at_ms) VALUES(?,2,'active',?)`, snapshotID, now.UnixMilli()); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(context.Background(), `INSERT INTO ai_tag_vocabulary_entries(snapshot_id,tag_id) VALUES(?,?)`, snapshotID, tag.ID); err != nil {
		t.Fatal(err)
	}
	var fingerprint string
	if err := store.db.QueryRowContext(context.Background(), `SELECT source_fingerprint FROM assets WHERE id=?`, assetID).Scan(&fingerprint); err != nil {
		t.Fatal(err)
	}
	suggestion := semantic.TagSuggestion{ID: "ais_clear_test", GenerationID: generationID, LibraryID: libraryID,
		AssetID: assetID, SnapshotID: snapshotID, TagID: tag.ID, SourceFingerprint: fingerprint,
		Confidence: .9, State: "pending", Revision: 1, CreatedAt: now, UpdatedAt: now}
	if err := store.ReplacePendingTagSuggestions(context.Background(), libraryID, assetID, generationID, snapshotID, fingerprint, []semantic.TagSuggestion{suggestion}, now); err != nil {
		t.Fatal(err)
	}
	curationService, err := curation.NewService(store, nil, func() time.Time { return now.Add(time.Minute) })
	if err != nil {
		t.Fatal(err)
	}
	reviewService, err := semantic.NewTagReviewService(store, curationService, func() time.Time { return now.Add(time.Minute) })
	if err != nil {
		t.Fatal(err)
	}
	curationRevision, err := store.Revision(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	idempotentReviews, err := semantic.NewIdempotentTagReviewService(reviewService, store, func() time.Time { return now.Add(time.Minute) })
	if err != nil {
		t.Fatal(err)
	}
	if _, err := idempotentReviews.Review(context.Background(), "tag-review-before-clear", []semantic.TagReviewItem{{SuggestionID: suggestion.ID,
		Action: semantic.TagReviewAccept, ExpectedSuggestionRevision: 1, ExpectedCurationRevision: curationRevision}}); err != nil {
		t.Fatal(err)
	}
	var reviewRevision int64
	if err := store.db.QueryRowContext(context.Background(), `SELECT revision FROM ai_tag_review_state WHERE library_id=?`, libraryID).Scan(&reviewRevision); err != nil {
		t.Fatal(err)
	}
	wake := &semanticWakeCounter{}
	idCounter := 0
	service, err := semantic.NewTagReviewClearService(store, wake, func() time.Time { return now.Add(2 * time.Minute) }, func(string) (string, error) {
		idCounter++
		if idCounter%2 == 1 {
			return "tagclear_test_job_" + string(rune('a'+idCounter)), nil
		}
		return "aio_tagclear_test_" + string(rune('a'+idCounter)), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Request(context.Background(), libraryID, reviewRevision-1, "tag-review-clear-stale"); !errors.Is(err, semantic.ErrTagReviewClearConflict) {
		t.Fatalf("stale revision error=%v", err)
	}
	first, err := service.Request(context.Background(), libraryID, reviewRevision, "tag-review-clear-key")
	if err != nil || !first.Created || first.Job.TotalItems != 1 || wake.count != 1 {
		t.Fatalf("first=%#v wake=%d err=%v", first, wake.count, err)
	}
	replay, err := service.Request(context.Background(), libraryID, reviewRevision, "tag-review-clear-key")
	if err != nil || !replay.Replayed || replay.Job.ID != first.Job.ID || wake.count != 1 {
		t.Fatalf("replay=%#v wake=%d err=%v", replay, wake.count, err)
	}
	claimed, found, err := store.ClaimTagReviewClear(context.Background(), now.Add(3*time.Minute), time.Minute)
	if err != nil || !found {
		t.Fatalf("claim=%#v found=%v err=%v", claimed, found, err)
	}
	processor, err := semantic.NewTagReviewClearProcessor(store, func() time.Time { return now.Add(3*time.Minute + time.Second) }, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := processor.Process(context.Background(), claimed); err != nil {
		t.Fatal(err)
	}
	var reviews, requests int
	if err := store.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM ai_tag_reviews WHERE library_id=?`, libraryID).Scan(&reviews); err != nil {
		t.Fatal(err)
	}
	if err := store.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM ai_tag_review_requests`).Scan(&requests); err != nil {
		t.Fatal(err)
	}
	state, err := store.GetAssetState(context.Background(), assetID)
	if err != nil || reviews != 0 || requests != 0 || len(state.Tags) != 1 || state.Tags[0].ID != tag.ID {
		t.Fatalf("reviews=%d requests=%d curation=%#v err=%v", reviews, requests, state, err)
	}
}

func TestTagReviewClearLeaseRecoveryKeepsCommittedProgress(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	now := time.Date(2026, 8, 30, 2, 0, 0, 0, time.UTC)
	if _, err := store.db.ExecContext(context.Background(), `INSERT INTO ai_model_operations(
		id,kind,state,phase,library_id,completed_items,total_items,revision,created_at_ms,updated_at_ms
	) VALUES('aio_tagclear_recovery','tag_review_clear','queued','queued',?,0,0,1,?,?);
	INSERT INTO ai_tag_review_clear_jobs(id,library_id,operation_id,expected_review_revision,state,requested_revision,attempt_count,created_at_ms,updated_at_ms)
	VALUES('tagclear_recovery',?,'aio_tagclear_recovery',1,'queued',1,0,?,?)`, libraryID, now.UnixMilli(), now.UnixMilli(), libraryID, now.UnixMilli(), now.UnixMilli()); err != nil {
		t.Fatal(err)
	}
	claimed, found, err := store.ClaimTagReviewClear(context.Background(), now, time.Minute)
	if err != nil || !found {
		t.Fatalf("claim=%#v found=%v err=%v", claimed, found, err)
	}
	summary, err := store.RecoverExpiredTagReviewClears(context.Background(), now.Add(2*time.Minute))
	if err != nil || summary.Requeued != 1 {
		t.Fatalf("recovery=%#v err=%v", summary, err)
	}
	second, found, err := store.ClaimTagReviewClear(context.Background(), now.Add(3*time.Minute), time.Minute)
	if err != nil || !found || second.ID != claimed.ID || second.AttemptCount != 2 {
		t.Fatalf("second=%#v found=%v err=%v", second, found, err)
	}
}
