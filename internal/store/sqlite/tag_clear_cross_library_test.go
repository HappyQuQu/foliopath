package sqlite

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/semantic"
)

func TestTagReviewClearMixedLedgerCancellationPreservesOtherLibrary(t *testing.T) {
	store, _ := openTestStore(t)
	a := seedBrowseCatalog(t, store)
	b := createWorkerLibrary(t, store, "Other review library", "other-reviews")
	ctx := context.Background()
	now := time.Now().UTC()
	// Synthetic second-library catalog; no filesystem media is opened.
	if _, err := store.db.Exec(`INSERT INTO directories(library_id,parent_id,relative_path,name,mtime_ns,last_seen_generation,direct_asset_count,recursive_asset_count) VALUES(?,NULL,'','',0,1,1,1)`, b.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(`INSERT INTO assets(library_id,directory_id,relative_path,name,kind,media_format,mime_type,size_bytes,mtime_ns,last_seen_generation,source_fingerprint,probe_status,playback_status) SELECT ?, (SELECT id FROM directories WHERE library_id=? AND relative_path=''),relative_path,name,kind,media_format,mime_type,size_bytes,mtime_ns,1,source_fingerprint,probe_status,playback_status FROM assets WHERE library_id=? ORDER BY id LIMIT 1`, b.ID, b.ID, a); err != nil {
		t.Fatal(err)
	}
	var assetA, assetB int64
	if err := store.db.QueryRow(`SELECT MIN(id) FROM assets WHERE library_id=?`, a).Scan(&assetA); err != nil {
		t.Fatal(err)
	}
	if err := store.db.QueryRow(`SELECT MIN(id) FROM assets WHERE library_id=?`, b.ID).Scan(&assetB); err != nil {
		t.Fatal(err)
	}
	tag, _, err := store.CreateTag(ctx, "preserved", "preserved", now)
	if err != nil {
		t.Fatal(err)
	}
	rev, err := store.Revision(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.AddAssetTag(ctx, assetB, rev, tag.ID, now); err != nil {
		t.Fatal(err)
	}
	before, err := store.GetAssetState(ctx, assetB)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(`INSERT INTO ai_tag_reviews(library_id,asset_id,tag_id,decision,source_suggestion_id,revision,reviewed_at_ms) VALUES(?,?,?,'dismissed','ais_mixed_a',1,1),(?,?,?,'dismissed','ais_mixed_b',1,1)`, a, assetA, tag.ID, b.ID, assetB, tag.ID); err != nil {
		t.Fatal(err)
	}
	mixed, other := strings.Repeat("a", 64), strings.Repeat("b", 64)
	for _, key := range []string{mixed, other} {
		if _, err := store.db.Exec(`INSERT INTO ai_tag_review_requests VALUES(?,?,'completed',1,1)`, key, key); err != nil {
			t.Fatal(err)
		}
	}
	for _, item := range []struct {
		key     string
		ordinal int
		id      string
	}{{mixed, 0, "ais_mixed_a"}, {mixed, 1, "ais_mixed_b"}, {other, 0, "ais_mixed_b"}} {
		if _, err := store.db.Exec(`INSERT INTO ai_tag_review_request_items VALUES(?,?,?,'dismissed',1,0,'dismissed',1)`, item.key, item.ordinal, item.id); err != nil {
			t.Fatal(err)
		}
	}
	var reviewRevision int64
	if err := store.db.QueryRow(`SELECT revision FROM ai_tag_review_state WHERE library_id=?`, a).Scan(&reviewRevision); err != nil {
		t.Fatal(err)
	}
	svc, err := semantic.NewTagReviewClearService(store, &semanticWakeCounter{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Request(ctx, a, reviewRevision, "mixed-ledger-clear"); err != nil {
		t.Fatal(err)
	}
	job, found, err := store.ClaimTagReviewClear(ctx, now, time.Minute)
	if err != nil || !found {
		t.Fatal(err)
	}
	deleted, done, err := store.DeleteTagReviewClearBatch(ctx, job, 1, now)
	if err != nil || deleted != 1 || !done {
		t.Fatalf("batch=%d/%v err=%v", deleted, done, err)
	}
	var operationRev int64
	if err := store.db.QueryRow(`SELECT revision FROM ai_model_operations WHERE id=?`, job.OperationID).Scan(&operationRev); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CancelTagReviewClearOperation(ctx, job.OperationID, operationRev, now); err != nil {
		t.Fatal(err)
	}
	finished, err := store.FinishTagReviewClear(ctx, job, semantic.JobSucceeded, "", now)
	if err != nil || finished.State != semantic.JobCancelled {
		t.Fatalf("finish=%#v err=%v", finished, err)
	}
	for _, q := range []struct {
		sql  string
		arg  any
		want int
	}{
		{`SELECT COUNT(*) FROM ai_tag_reviews WHERE library_id=?`, a, 0},
		{`SELECT COUNT(*) FROM ai_tag_reviews WHERE library_id=?`, b.ID, 1},
		{`SELECT COUNT(*) FROM ai_tag_review_requests WHERE idempotency_key_hash=?`, mixed, 0},
		{`SELECT COUNT(*) FROM ai_tag_review_request_items WHERE idempotency_key_hash=?`, mixed, 0},
		{`SELECT COUNT(*) FROM ai_tag_review_requests WHERE idempotency_key_hash=?`, other, 1},
		{`SELECT COUNT(*) FROM ai_tag_review_request_items WHERE idempotency_key_hash=?`, other, 1},
	} {
		var count int
		if err := store.db.QueryRow(q.sql, q.arg).Scan(&count); err != nil || count != q.want {
			t.Fatalf("%s got=%d want=%d err=%v", q.sql, count, q.want, err)
		}
	}
	after, err := store.GetAssetState(ctx, assetB)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatalf("other curation changed: %#v => %#v err=%v", before, after, err)
	}
	if _, _, err := store.DeleteTagReviewClearBatch(ctx, job, 1, now); !errors.Is(err, semantic.ErrTagReviewClearConflict) {
		t.Fatalf("terminal delete: %v", err)
	}
}
