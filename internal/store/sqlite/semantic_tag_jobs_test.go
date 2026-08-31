package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/semantic"
)

func TestTagSuggestionJobPersistsZeroSuggestionProgressAndResumes(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	generationID := seedEmbeddingGeneration(t, store, 2)
	now := time.Date(2026, 8, 30, 3, 0, 0, 0, time.UTC)
	tag, _, err := store.CreateTag(context.Background(), "unlikely", "unlikely", now)
	if err != nil {
		t.Fatal(err)
	}
	const snapshotID = "aiv_tag_job_test"
	if _, err := store.db.ExecContext(context.Background(), `UPDATE ai_tag_vocabulary_snapshots SET state='retired' WHERE state='active'`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(context.Background(), `INSERT INTO ai_tag_vocabulary_snapshots(id,revision,state,created_at_ms) VALUES(?,2,'active',?)`, snapshotID, now.UnixMilli()); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(context.Background(), `INSERT INTO ai_tag_vocabulary_entries(snapshot_id,tag_id) VALUES(?,?)`, snapshotID, tag.ID); err != nil {
		t.Fatal(err)
	}
	assetID := catalogAssetID(t, store, "photo-10.jpg")
	var fingerprint string
	if err := store.db.QueryRowContext(context.Background(), `SELECT source_fingerprint FROM assets WHERE id=?`, assetID).Scan(&fingerprint); err != nil {
		t.Fatal(err)
	}
	assetVector, _ := semantic.EncodeEmbedding([]float32{1, 0}, 2)
	tagVector, _ := semantic.EncodeEmbedding([]float32{-1, 0}, 2)
	if err := store.PutSemanticEmbeddingBatch(context.Background(), semantic.EmbeddingBatch{GenerationID: generationID, LibraryID: libraryID, Items: []semantic.EmbeddingItem{{AssetID: assetID, SourceFingerprint: fingerprint, Vector: assetVector, CreatedAt: now}}}); err != nil {
		t.Fatal(err)
	}
	if err := store.PutTagEmbeddingBatch(context.Background(), semantic.TagEmbeddingBatch{GenerationID: generationID, SnapshotID: snapshotID, CreatedAt: now, Items: []semantic.TagEmbedding{{TagID: tag.ID, Vector: tagVector}}}); err != nil {
		t.Fatal(err)
	}
	wake := &semanticWakeCounter{}
	ids := []string{"tagjob_zero_test", "aio_tagjob_zero"}
	service, err := semantic.NewTagJobService(store, store, wake, func() time.Time { return now }, func(string) (string, error) { id := ids[0]; ids = ids[1:]; return id, nil })
	if err != nil {
		t.Fatal(err)
	}
	admitted, err := service.Request(context.Background(), libraryID, generationID, snapshotID, semantic.JobMissing, "tag-job-zero-key")
	if err != nil || !admitted.Created || admitted.Job.TotalItems != 1 {
		t.Fatalf("admitted=%#v err=%v", admitted, err)
	}
	replayed, err := service.Request(context.Background(), libraryID, generationID, snapshotID, semantic.JobMissing, "tag-job-zero-key")
	if err != nil || !replayed.Replayed || replayed.Job.ID != admitted.Job.ID || wake.count != 1 {
		t.Fatalf("replay=%#v wake=%d err=%v", replayed, wake.count, err)
	}
	claimed, found, err := store.ClaimTagJob(context.Background(), now.Add(time.Minute), time.Minute)
	if err != nil || !found {
		t.Fatalf("claim=%#v found=%v err=%v", claimed, found, err)
	}
	recovery, err := store.RecoverExpiredTagJobs(context.Background(), now.Add(3*time.Minute))
	if err != nil || recovery.Requeued != 1 {
		t.Fatalf("recovery=%#v err=%v", recovery, err)
	}
	claimed, found, err = store.ClaimTagJob(context.Background(), now.Add(4*time.Minute), time.Minute)
	if err != nil || !found || claimed.AttemptCount != 2 {
		t.Fatalf("reclaim=%#v found=%v err=%v", claimed, found, err)
	}
	builder, err := semantic.NewControlledTagPlanBuilder(store, func() time.Time { return now.Add(time.Minute) }, func(string) (string, error) { return "ais_never_used", nil }, .99)
	if err != nil {
		t.Fatal(err)
	}
	processor, err := semantic.NewTagJobProcessor(store, store, builder, func() time.Time { return now.Add(time.Minute) })
	if err != nil {
		t.Fatal(err)
	}
	if err := processor.Process(context.Background(), claimed); err != nil {
		t.Fatal(err)
	}
	progress, found, err := store.GetTagJobProgress(context.Background(), generationID, libraryID, snapshotID)
	if err != nil || !found || progress.Ready != 1 || progress.Eligible < 1 {
		t.Fatalf("progress=%#v found=%v err=%v", progress, found, err)
	}
	var suggestions int
	if err := store.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM ai_tag_suggestions WHERE library_id=?`, libraryID).Scan(&suggestions); err != nil || suggestions != 0 {
		t.Fatalf("suggestions=%d err=%v", suggestions, err)
	}
	counts, err := store.CountTagJobCandidates(context.Background(), libraryID, generationID, snapshotID, semantic.JobMissing)
	if err != nil || counts.Pending != 0 {
		t.Fatalf("counts=%#v err=%v", counts, err)
	}
}
