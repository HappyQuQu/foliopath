package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/semantic"
)

// Inject the accepted cancellation after processing, immediately before the
// real terminal transaction; all batching and persistence still use Store.
type cancelBeforeImageFinish struct {
	*Store
	t *testing.T
}

func (q cancelBeforeImageFinish) FinishSemanticBackfill(ctx context.Context, job semantic.BackfillJob, outcome semantic.JobState, code string, now time.Time) (semantic.BackfillJob, error) {
	var revision int64
	if err := q.db.QueryRow(`SELECT revision FROM ai_model_operations WHERE id=?`, job.OperationID).Scan(&revision); err != nil {
		q.t.Fatal(err)
	}
	if _, err := q.CancelSemanticBackfill(ctx, job.ID, revision, now); err != nil {
		q.t.Fatal(err)
	}
	return q.Store.FinishSemanticBackfill(ctx, job, outcome, code, now)
}

func TestImageProcessorAcceptedCancelAtTerminalBoundary(t *testing.T) {
	for _, fail := range []bool{false, true} {
		name := "all-pages-committed"
		if fail {
			name = "encoder-failed"
		}
		t.Run(name, func(t *testing.T) {
			store, _ := openTestStore(t)
			library := seedBrowseCatalog(t, store)
			generation := seedEmbeddingGeneration(t, store, 2)
			seedSemanticLibrarySettings(t, store, library)
			now := time.Now().UTC()
			admitSemanticBackfill(t, store, library, generation, now, "processor-terminal-cancel")
			job, found, err := store.ClaimSemanticBackfill(context.Background(), now, time.Minute)
			if err != nil || !found {
				t.Fatalf("claim found=%v err=%v", found, err)
			}
			encoder := semanticEncoderStub{}
			if fail {
				encoder.err = semantic.ErrImageEncoderUnavailable
			}
			processor, err := semantic.NewBackfillProcessor(store, semanticAssetSourceStub{store: store}, semanticPreprocessorStub{}, encoder, store, cancelBeforeImageFinish{store, t}, func() time.Time { return now }, 1)
			if err != nil {
				t.Fatal(err)
			}
			err = processor.Process(context.Background(), job)
			if fail && !errors.Is(err, semantic.ErrImageEncoderUnavailable) || !fail && err != nil {
				t.Fatalf("processor: %v", err)
			}
			stored, err := scanSemanticJob(store.db.QueryRow(semanticJobSelect+` WHERE job.id=?`, job.ID))
			want := job.TotalItems
			if fail {
				want = 0
			}
			if err != nil || stored.State != semantic.JobCancelled || stored.CompletedItems != want || stored.LeaseExpiresAt != nil || stored.ErrorCode != "cancelled" {
				t.Fatalf("stored=%#v err=%v", stored, err)
			}
			var count int64
			if err := store.db.QueryRow(`SELECT COUNT(*) FROM semantic_embeddings WHERE library_id=? AND generation_id=?`, library, generation).Scan(&count); err != nil || count != want {
				t.Fatalf("vectors=%d want=%d err=%v", count, want, err)
			}
			var state string
			if err := store.db.QueryRow(`SELECT state FROM ai_model_operations WHERE id=?`, job.OperationID).Scan(&state); err != nil || state != "cancelled" {
				t.Fatalf("operation=%s err=%v", state, err)
			}
		})
	}
}
