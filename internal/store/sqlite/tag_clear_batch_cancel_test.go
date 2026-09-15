package sqlite

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/semantic"
)

type cancelAfterTagClearBatch struct {
	*Store
	t        *testing.T
	accepted bool
}

func (q *cancelAfterTagClearBatch) DeleteTagReviewClearBatch(ctx context.Context, job semantic.TagReviewClearJob, limit int, now time.Time) (int64, bool, error) {
	deleted, done, err := q.Store.DeleteTagReviewClearBatch(ctx, job, limit, now)
	if err == nil && !q.accepted {
		var revision int64
		if err := q.db.QueryRow(`SELECT revision FROM ai_model_operations WHERE id=?`, job.OperationID).Scan(&revision); err != nil {
			q.t.Fatal(err)
		}
		if _, err := q.CancelTagReviewClearOperation(ctx, job.OperationID, revision, now); err != nil {
			q.t.Fatal(err)
		}
		q.accepted = true
	}
	return deleted, done, err
}

func TestTagReviewClearCancellationAfterCommittedBatch(t *testing.T) {
	for _, size := range []int{1, 1000} {
		name := "partial-batch"
		if size == 1000 {
			name = "final-batch"
		}
		t.Run(name, func(t *testing.T) {
			store, _ := openTestStore(t)
			library := seedBrowseCatalog(t, store)
			ctx := context.Background()
			now := time.Now().UTC()
			tag, _, err := store.CreateTag(ctx, "preserved", "preserved", now)
			if err != nil {
				t.Fatal(err)
			}
			asset := catalogAssetID(t, store, "photo-10.jpg")
			revision, err := store.Revision(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if err := store.AddAssetTag(ctx, asset, revision, tag.ID, now); err != nil {
				t.Fatal(err)
			}
			before, err := store.GetAssetState(ctx, asset)
			if err != nil {
				t.Fatal(err)
			}
			// Synthetic review rows; formal curation was written through its owner.
			if _, err := store.db.Exec(`INSERT INTO ai_tag_reviews(library_id,asset_id,tag_id,decision,revision,reviewed_at_ms) SELECT library_id,id,?,'dismissed',1,? FROM assets WHERE library_id=? ORDER BY id LIMIT 2`, tag.ID, now.UnixMilli(), library); err != nil {
				t.Fatal(err)
			}
			service, err := semantic.NewTagReviewClearService(store, &semanticWakeCounter{}, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			var reviewRevision int64
			if err := store.db.QueryRow(`SELECT revision FROM ai_tag_review_state WHERE library_id=?`, library).Scan(&reviewRevision); err != nil {
				t.Fatal(err)
			}
			if _, err := service.Request(ctx, library, reviewRevision, "tagclear-batch-cancel"); err != nil {
				t.Fatal(err)
			}
			job, found, err := store.ClaimTagReviewClear(ctx, time.Now().UTC(), time.Minute)
			if err != nil || !found || job.TotalItems != 2 {
				t.Fatalf("job=%#v found=%v err=%v", job, found, err)
			}
			processor, err := semantic.NewTagReviewClearProcessor(&cancelAfterTagClearBatch{Store: store, t: t}, nil, size)
			if err != nil {
				t.Fatal(err)
			}
			err = processor.Process(ctx, job)
			if size == 1 && !errors.Is(err, semantic.ErrTagReviewClearConflict) || size == 1000 && err != nil {
				t.Fatalf("process: %v", err)
			}
			stored, err := scanTagReviewClearJob(store.db.QueryRow(tagReviewClearJobSelect+` WHERE job.id=?`, job.ID))
			want := min(int64(size), int64(2))
			if err != nil || stored.State != semantic.JobCancelled || stored.DeletedCount != want || stored.CompletedItems != want || stored.LeaseExpiresAt != nil {
				t.Fatalf("stored=%#v err=%v", stored, err)
			}
			var count int64
			if err := store.db.QueryRow(`SELECT COUNT(*) FROM ai_tag_reviews WHERE library_id=?`, library).Scan(&count); err != nil || count != 2-want {
				t.Fatalf("remaining=%d err=%v", count, err)
			}
			after, err := store.GetAssetState(ctx, asset)
			if err != nil || !reflect.DeepEqual(before, after) {
				t.Fatalf("curation changed: %#v => %#v err=%v", before, after, err)
			}
			var state string
			if err := store.db.QueryRow(`SELECT state FROM ai_model_operations WHERE id=?`, job.OperationID).Scan(&state); err != nil || state != "cancelled" {
				t.Fatalf("operation=%s err=%v", state, err)
			}
			if _, err := store.FinishTagReviewClear(ctx, job, semantic.JobFailed, "internal_error", time.Now().UTC()); !errors.Is(err, semantic.ErrTagReviewClearConflict) {
				t.Fatalf("terminal overwritten: %v", err)
			}
		})
	}
}
