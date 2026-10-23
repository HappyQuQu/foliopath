package sqlite

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/semantic"
)

func TestSemanticBackfillAcceptedCancelWinsTerminalTransaction(t *testing.T) {
	for _, outcome := range []semantic.JobState{semantic.JobSucceeded, semantic.JobFailed} {
		t.Run(string(outcome), func(t *testing.T) {
			store, _ := openTestStore(t)
			library := seedBrowseCatalog(t, store)
			generation := seedEmbeddingGeneration(t, store, 2)
			seedSemanticLibrarySettings(t, store, library)
			ctx := context.Background()
			now := time.Now().UTC()
			admitSemanticBackfill(t, store, library, generation, now, "finish-cancel-regression")
			job, found, err := store.ClaimSemanticBackfill(ctx, now, time.Minute)
			if err != nil || !found {
				t.Fatalf("claim found=%v err=%v", found, err)
			}
			before, beforeFound, err := store.GetSemanticEmbeddingProgress(ctx, generation, library)
			if err != nil {
				t.Fatal(err)
			}
			// A stale owner must not be able to finalize even an accepted cancel.
			if _, err := store.CancelSemanticBackfill(ctx, job.ID, job.OperationRevision, now); err != nil {
				t.Fatal(err)
			}
			stale := job
			stale.ClaimedRevision++
			if _, err := store.FinishSemanticBackfill(ctx, stale, semantic.JobCancelled, "", now); !errors.Is(err, semantic.ErrSemanticJobConflict) {
				t.Fatalf("stale owner: %v", err)
			}
			code := ""
			if outcome == semantic.JobFailed {
				code = "model_unavailable"
			}
			finished, err := store.FinishSemanticBackfill(ctx, job, outcome, code, now)
			if err != nil || finished.State != semantic.JobCancelled || finished.ErrorCode != "cancelled" || finished.LeaseExpiresAt != nil {
				t.Fatalf("finished=%#v err=%v", finished, err)
			}
			var state, phase string
			if err := store.db.QueryRow(`SELECT state,phase FROM ai_model_operations WHERE id=?`, job.OperationID).Scan(&state, &phase); err != nil || state != "cancelled" || phase != "completed" {
				t.Fatalf("operation=%s/%s err=%v", state, phase, err)
			}
			after, afterFound, err := store.GetSemanticEmbeddingProgress(ctx, generation, library)
			if err != nil || beforeFound != afterFound || !reflect.DeepEqual(before, after) {
				t.Fatalf("progress changed: before=%#v after=%#v err=%v", before, after, err)
			}
			if _, err := store.FinishSemanticBackfill(ctx, job, outcome, code, now); !errors.Is(err, semantic.ErrSemanticJobConflict) {
				t.Fatalf("terminal overwritten: %v", err)
			}
		})
	}
}
