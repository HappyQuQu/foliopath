package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/curation"
	"github.com/HappyQuQu/foliopath/internal/semantic"
)

func TestControlledTagEmbeddingAndSuggestionRepository(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	generationID := seedEmbeddingGeneration(t, store, 2)
	assetID := catalogAssetID(t, store, "photo-10.jpg")
	now := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)

	first, _, err := store.CreateTag(context.Background(), "cat", "cat", now)
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := store.CreateTag(context.Background(), "travel", "travel", now)
	if err != nil {
		t.Fatal(err)
	}
	const snapshotID = "aiv_tag_snapshot_one"
	if _, err := store.db.ExecContext(context.Background(), `
        UPDATE ai_tag_vocabulary_snapshots SET state='retired' WHERE state='active';
        INSERT INTO ai_tag_vocabulary_snapshots(id, revision, state, created_at_ms)
        VALUES(?, 2, 'active', ?)`, snapshotID, now.UnixMilli()); err != nil {
		t.Fatal(err)
	}
	for _, tagID := range []int64{first.ID, second.ID} {
		if _, err := store.db.ExecContext(context.Background(), `
            INSERT INTO ai_tag_vocabulary_entries(snapshot_id, tag_id) VALUES(?, ?)`, snapshotID, tagID); err != nil {
			t.Fatal(err)
		}
	}
	vector, err := semantic.EncodeEmbedding([]float32{1, 0}, 2)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.PutTagEmbeddingBatch(context.Background(), semantic.TagEmbeddingBatch{
		GenerationID: generationID,
		SnapshotID:   snapshotID,
		CreatedAt:    now,
		Items:        []semantic.TagEmbedding{{TagID: first.ID, Vector: vector}, {TagID: second.ID, Vector: vector}},
	}); err != nil {
		t.Fatal(err)
	}
	var fingerprint string
	if err := store.db.QueryRowContext(context.Background(), `SELECT source_fingerprint FROM assets WHERE id = ?`, assetID).Scan(&fingerprint); err != nil {
		t.Fatal(err)
	}
	if err := store.PutSemanticEmbeddingBatch(context.Background(), semantic.EmbeddingBatch{GenerationID: generationID, LibraryID: libraryID, Items: []semantic.EmbeddingItem{{AssetID: assetID, SourceFingerprint: fingerprint, Vector: vector, CreatedAt: now}}}); err != nil {
		t.Fatal(err)
	}
	items := []semantic.TagSuggestion{
		{ID: "ais_suggestion_first", GenerationID: generationID, LibraryID: libraryID, AssetID: assetID, SnapshotID: snapshotID, TagID: first.ID, SourceFingerprint: fingerprint, Confidence: .9, State: "pending", Revision: 1, CreatedAt: now, UpdatedAt: now},
		{ID: "ais_suggestion_second", GenerationID: generationID, LibraryID: libraryID, AssetID: assetID, SnapshotID: snapshotID, TagID: second.ID, SourceFingerprint: fingerprint, Confidence: .8, State: "pending", Revision: 1, CreatedAt: now, UpdatedAt: now},
	}
	if err := store.ReplacePendingTagSuggestions(context.Background(), libraryID, assetID, generationID, snapshotID, fingerprint, items, now); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := store.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM ai_tag_suggestions WHERE state = 'pending'`).Scan(&count); err != nil || count != 2 {
		t.Fatalf("pending = %d, %v", count, err)
	}
	listService, err := semantic.NewTagSuggestionListService(store, make([]byte, 32))
	if err != nil {
		t.Fatal(err)
	}
	firstPage, err := listService.List(context.Background(), semantic.TagSuggestionListRequest{
		LibraryID: libraryID, Status: semantic.TagSuggestionPending, Limit: 1,
	})
	if err != nil || len(firstPage.Items) != 1 || firstPage.Items[0].ID != items[0].ID || firstPage.NextCursor == "" {
		t.Fatalf("first page = %#v, %v", firstPage, err)
	}
	secondPage, err := listService.List(context.Background(), semantic.TagSuggestionListRequest{
		LibraryID: libraryID, Status: semantic.TagSuggestionPending, Cursor: firstPage.NextCursor, Limit: 1,
	})
	if err != nil || len(secondPage.Items) != 1 || secondPage.Items[0].ID != items[1].ID || secondPage.NextCursor != "" {
		t.Fatalf("second page = %#v, %v", secondPage, err)
	}

	curationService, err := curation.NewService(store, nil, func() time.Time { return now.Add(time.Minute) })
	if err != nil {
		t.Fatal(err)
	}
	reviewNow := now.Add(time.Minute)
	reviewService, err := semantic.NewTagReviewService(store, curationService, func() time.Time { return reviewNow })
	if err != nil {
		t.Fatal(err)
	}
	curationRevision, err := store.Revision(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	idempotentReviews, err := semantic.NewIdempotentTagReviewService(reviewService, store, func() time.Time { return reviewNow })
	if err != nil {
		t.Fatal(err)
	}
	reviewItems := []semantic.TagReviewItem{{
		SuggestionID: items[0].ID, Action: semantic.TagReviewAccept,
		ExpectedSuggestionRevision: 1, ExpectedCurationRevision: curationRevision,
	}}
	idempotentResult, err := idempotentReviews.Review(context.Background(), "tag-review-key-001", reviewItems)
	if err != nil || idempotentResult.Replayed || len(idempotentResult.Items) != 1 ||
		idempotentResult.Items[0].Outcome != semantic.TagReviewAccept || idempotentResult.Items[0].Conflict {
		t.Fatalf("accept result = %#v, %v", idempotentResult, err)
	}
	replayed, err := idempotentReviews.Review(context.Background(), "tag-review-key-001", reviewItems)
	if err != nil || !replayed.Replayed || !slices.Equal(replayed.Items, idempotentResult.Items) {
		t.Fatalf("replayed result = %#v, %v", replayed, err)
	}
	if _, err := idempotentReviews.Review(context.Background(), "tag-review-key-001", []semantic.TagReviewItem{{
		SuggestionID: items[1].ID, Action: semantic.TagReviewDismiss, ExpectedSuggestionRevision: 1,
	}}); !errors.Is(err, semantic.ErrTagReviewRequestConflict) {
		t.Fatalf("idempotency conflict = %v", err)
	}
	if _, err := listService.List(context.Background(), semantic.TagSuggestionListRequest{
		LibraryID: libraryID, Status: semantic.TagSuggestionPending, Cursor: firstPage.NextCursor, Limit: 1,
	}); !errors.Is(err, semantic.ErrTagSuggestionCursorStale) {
		t.Fatalf("stale pending cursor = %v", err)
	}
	acceptedPage, err := listService.List(context.Background(), semantic.TagSuggestionListRequest{
		LibraryID: libraryID, Status: semantic.TagSuggestionAccepted, Limit: 10,
	})
	if err != nil || len(acceptedPage.Items) != 1 || acceptedPage.Items[0].ID != items[0].ID ||
		acceptedPage.Items[0].GenerationID != generationID || acceptedPage.Items[0].VocabularyRevision != 2 ||
		acceptedPage.Items[0].ReviewedAt == nil {
		t.Fatalf("accepted page = %#v, %v", acceptedPage, err)
	}
	state, err := store.GetAssetState(context.Background(), assetID)
	if err != nil || len(state.Tags) != 1 || state.Tags[0].ID != first.ID {
		t.Fatalf("accepted curation state = %#v, %v", state, err)
	}
	if err := store.ReplacePendingTagSuggestions(context.Background(), libraryID, assetID, generationID, snapshotID, fingerprint, items, now.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := store.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM ai_tag_suggestions WHERE state = 'pending'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("accepted-tag suppressed pending = %d, %v", count, err)
	}
	reviewNow = now.Add(3 * time.Minute)
	outcomes, err := reviewService.Review(context.Background(), []semantic.TagReviewItem{{
		SuggestionID: items[1].ID, Action: semantic.TagReviewDismiss, ExpectedSuggestionRevision: 1,
	}})
	if err != nil || len(outcomes) != 1 || outcomes[0].Outcome != semantic.TagReviewDismiss || outcomes[0].Conflict {
		t.Fatalf("dismiss outcomes = %#v, %v", outcomes, err)
	}
	if err := store.ReplacePendingTagSuggestions(context.Background(), libraryID, assetID, generationID, snapshotID, fingerprint, items, now.Add(4*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := store.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM ai_tag_suggestions WHERE state = 'pending'`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("reviewed tags reappeared = %d, %v", count, err)
	}
	if _, err := store.db.ExecContext(context.Background(), `UPDATE semantic_generations SET state='retired' WHERE id=?`, generationID); err != nil {
		t.Fatal(err)
	}
	reviewedWithoutModel, err := listService.List(context.Background(), semantic.TagSuggestionListRequest{LibraryID: libraryID, Status: semantic.TagSuggestionAccepted, Limit: 10})
	if err != nil || len(reviewedWithoutModel.Items) != 1 || reviewedWithoutModel.Items[0].ID != items[0].ID {
		t.Fatalf("reviewed without active model=%#v err=%v", reviewedWithoutModel, err)
	}
}

func TestTagReviewRequestClampsTimeAcrossClockRollback(t *testing.T) {
	store, _ := openTestStore(t)
	createdAt := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)
	const keyHash = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	record := semantic.TagReviewRequestRecord{
		IdempotencyKeyHash: keyHash,
		RequestHash:        "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		State:              "running",
		Items: []semantic.TagReviewRequestItemState{{Item: semantic.TagReviewItem{
			SuggestionID: "suggestion_clock_01", Action: semantic.TagReviewAccept,
			ExpectedSuggestionRevision: 1, ExpectedCurationRevision: 1,
		}}},
		CreatedAt: createdAt, UpdatedAt: createdAt,
	}
	if _, created, err := store.BeginTagReviewRequest(context.Background(), record); err != nil || !created {
		t.Fatalf("begin created=%t err=%v", created, err)
	}
	if err := store.CommitTagReviewRequestOutcome(context.Background(), keyHash, 0, semantic.TagReviewOutcome{
		SuggestionID: "suggestion_clock_01", Outcome: semantic.TagReviewAccept, Revision: 2,
	}, createdAt.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := store.CompleteTagReviewRequest(context.Background(), keyHash, createdAt.Add(-2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	stored, err := store.getTagReviewRequest(context.Background(), keyHash)
	if err != nil {
		t.Fatal(err)
	}
	if stored.State != "completed" || stored.UpdatedAt.Before(stored.CreatedAt) {
		t.Fatalf("stored=%+v", stored)
	}
}

func TestTagSuggestionInvalidationClampsTimeAcrossClockRollback(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	assetID := catalogAssetID(t, store, "photo-10.jpg")
	generationID := seedEmbeddingGeneration(t, store, 2)
	createdAt := time.Date(2026, 8, 29, 13, 0, 0, 0, time.UTC)
	tag, _, err := store.CreateTag(context.Background(), "clock", "clock", createdAt)
	if err != nil {
		t.Fatal(err)
	}
	reviewTag, _, err := store.CreateTag(context.Background(), "clock-review", "clock-review", createdAt)
	if err != nil {
		t.Fatal(err)
	}
	var snapshotID string
	if err := store.db.QueryRow(`SELECT id FROM ai_tag_vocabulary_snapshots WHERE state='active'`).Scan(&snapshotID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(`INSERT INTO ai_tag_vocabulary_entries(snapshot_id,tag_id) VALUES(?,?),(?,?)`, snapshotID, tag.ID, snapshotID, reviewTag.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(`UPDATE semantic_generations SET state='active' WHERE id=?`, generationID); err != nil {
		t.Fatal(err)
	}
	var fingerprint string
	if err := store.db.QueryRow(`SELECT source_fingerprint FROM assets WHERE library_id=? AND id=?`, libraryID, assetID).Scan(&fingerprint); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(`INSERT INTO ai_tag_suggestions(id,generation_id,library_id,asset_id,vocabulary_snapshot_id,tag_id,source_fingerprint,confidence,state,revision,created_at_ms,updated_at_ms)
		VALUES('suggestion_clock_stale',?,?,?,?,?,'stale-source',0.9,'pending',1,?,?)`, generationID, libraryID, assetID, snapshotID, tag.ID, createdAt.UnixMilli(), createdAt.UnixMilli()); err != nil {
		t.Fatal(err)
	}
	rollback := createdAt.Add(-time.Hour)
	if err := store.ReplacePendingTagSuggestions(context.Background(), libraryID, assetID, generationID, snapshotID, fingerprint, nil, rollback); err != nil {
		t.Fatal(err)
	}
	var state string
	var storedCreatedAt, storedUpdatedAt int64
	if err := store.db.QueryRow(`SELECT state,created_at_ms,updated_at_ms FROM ai_tag_suggestions WHERE id='suggestion_clock_stale'`).
		Scan(&state, &storedCreatedAt, &storedUpdatedAt); err != nil {
		t.Fatal(err)
	}
	if state != "invalidated" || storedUpdatedAt < storedCreatedAt {
		t.Fatalf("stale suggestion=%q %d/%d", state, storedCreatedAt, storedUpdatedAt)
	}

	const reviewSuggestionID = "suggestion_clock_review"
	if _, err := store.db.Exec(`INSERT INTO ai_tag_suggestions(id,generation_id,library_id,asset_id,vocabulary_snapshot_id,tag_id,source_fingerprint,confidence,state,revision,created_at_ms,updated_at_ms)
		VALUES(?,?,?,?,?,?,?,0.8,'pending',1,?,?)`, reviewSuggestionID, generationID, libraryID, assetID, snapshotID, reviewTag.ID, fingerprint, createdAt.UnixMilli(), createdAt.UnixMilli()); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CommitTagReview(context.Background(), reviewSuggestionID, 1, semantic.TagReviewDismiss, 0, rollback.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := store.db.QueryRow(`SELECT state,created_at_ms,updated_at_ms FROM ai_tag_suggestions WHERE id=?`, reviewSuggestionID).
		Scan(&state, &storedCreatedAt, &storedUpdatedAt); err != nil {
		t.Fatal(err)
	}
	if state != "invalidated" || storedUpdatedAt < storedCreatedAt {
		t.Fatalf("reviewed suggestion=%q %d/%d", state, storedCreatedAt, storedUpdatedAt)
	}
}

func TestTagSuggestionMigrationRejectsFreeTextAndCascadesDerivedState(t *testing.T) {
	store, _ := openTestStore(t)
	assertTableColumns(t, store.db, "ai_tag_suggestions", []string{
		"id", "generation_id", "library_id", "asset_id", "vocabulary_snapshot_id", "tag_id",
		"source_fingerprint", "confidence", "state", "revision", "created_at_ms", "updated_at_ms",
	})
	assertTableColumns(t, store.db, "ai_tag_reviews", []string{
		"library_id", "asset_id", "tag_id", "decision", "source_suggestion_id",
		"accepted_curation_revision", "revision", "reviewed_at_ms", "source_generation_id",
		"source_vocabulary_snapshot_id", "source_vocabulary_revision", "source_confidence",
		"source_suggestion_revision",
	})
}

func assertTableColumns(t *testing.T, db *sql.DB, table string, expected []string) {
	t.Helper()
	rows, err := db.Query(`SELECT name FROM pragma_table_info(?) ORDER BY cid`, table)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var actual []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatal(err)
		}
		actual = append(actual, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(actual, expected) {
		t.Fatalf("%s columns = %v, want %v", table, actual, expected)
	}
}
