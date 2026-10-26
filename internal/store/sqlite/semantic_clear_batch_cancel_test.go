package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/semantic"
)

type cancelAfterClearBatch struct {
	*Store
	t        *testing.T
	accepted bool
}

func (q *cancelAfterClearBatch) DeleteSemanticClearBatch(ctx context.Context, job semantic.ClearJob, limit int, now time.Time) (int64, bool, error) {
	deleted, done, err := q.Store.DeleteSemanticClearBatch(ctx, job, limit, now)
	if err == nil && !q.accepted {
		var revision int64
		if err := q.db.QueryRow(`SELECT revision FROM ai_model_operations WHERE id=?`, job.OperationID).Scan(&revision); err != nil {
			q.t.Fatal(err)
		}
		if _, err := q.CancelSemanticClearOperation(ctx, job.OperationID, revision, now); err != nil {
			q.t.Fatal(err)
		}
		q.accepted = true
	}
	return deleted, done, err
}

func TestSemanticClearCancellationAfterCommittedBatch(t *testing.T) {
	for _, size := range []int{1, 1000} {
		name := "partial-batch"
		if size == 1000 {
			name = "final-batch"
		}
		t.Run(name, func(t *testing.T) {
			store, _ := openTestStore(t)
			library := seedBrowseCatalog(t, store)
			generation := seedEmbeddingGeneration(t, store, 2)
			seedSemanticLibrarySettings(t, store, library)
			ctx := context.Background()
			now := time.Now().UTC()
			admitSemanticBackfill(t, store, library, generation, now, "clear-batch-seed")
			seed, found, err := store.ClaimSemanticBackfill(ctx, now, time.Minute)
			if err != nil || !found {
				t.Fatal(err)
			}
			backfill, err := semantic.NewBackfillProcessor(store, semanticAssetSourceStub{store}, semanticPreprocessorStub{}, semanticEncoderStub{}, store, store, nil, 1)
			if err != nil {
				t.Fatal(err)
			}
			if err := backfill.Process(ctx, seed); err != nil {
				t.Fatal(err)
			}
			settings, err := store.GetSemanticLibrarySettings(ctx, library)
			if err != nil {
				t.Fatal(err)
			}
			service, err := semantic.NewClearService(store, &semanticWakeCounter{}, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := service.Request(ctx, library, settings.Revision, "clear-batch-cancel"); err != nil {
				t.Fatal(err)
			}
			job, found, err := store.ClaimSemanticClear(ctx, time.Now().UTC(), time.Minute)
			if err != nil || !found {
				t.Fatal(err)
			}
			queue := &cancelAfterClearBatch{Store: store, t: t}
			processor, err := semantic.NewClearProcessor(queue, nil, size)
			if err != nil {
				t.Fatal(err)
			}
			err = processor.Process(ctx, job)
			if size == 1 && !errors.Is(err, semantic.ErrSemanticClearConflict) || size == 1000 && err != nil {
				t.Fatalf("process: %v", err)
			}
			stored, err := scanSemanticClearJob(store.db.QueryRow(semanticClearJobSelect+` WHERE job.id=?`, job.ID))
			wantDeleted := min(int64(size), seed.TotalItems)
			if err != nil || stored.State != semantic.JobCancelled || stored.CompletedItems != wantDeleted || stored.LeaseExpiresAt != nil || stored.ErrorCode != "cancelled" {
				t.Fatalf("job=%#v err=%v", stored, err)
			}
			var remaining, assets int64
			if err := store.db.QueryRow(`SELECT (SELECT COUNT(*) FROM semantic_embeddings WHERE library_id=?),(SELECT COUNT(*) FROM assets WHERE library_id=?)`, library, library).Scan(&remaining, &assets); err != nil || remaining != seed.TotalItems-wantDeleted || assets < seed.TotalItems {
				t.Fatalf("remaining=%d assets=%d err=%v", remaining, assets, err)
			}
			settings, err = store.GetSemanticLibrarySettings(ctx, library)
			if err != nil || settings.Enabled || settings.State != semantic.LibraryDegraded {
				t.Fatalf("settings=%#v err=%v", settings, err)
			}
			var state string
			if err := store.db.QueryRow(`SELECT state FROM ai_model_operations WHERE id=?`, job.OperationID).Scan(&state); err != nil || state != "cancelled" {
				t.Fatalf("operation=%s err=%v", state, err)
			}
			if _, err := store.FinishSemanticClear(ctx, job, semantic.JobFailed, "internal_error", time.Now().UTC()); !errors.Is(err, semantic.ErrSemanticClearConflict) {
				t.Fatalf("terminal overwritten: %v", err)
			}
		})
	}
}
