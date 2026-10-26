package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/semantic"
)

type retiringEmbeddingStore struct{ *Store }

func (s retiringEmbeddingStore) CommitSemanticEmbeddingProgress(ctx context.Context, commit semantic.EmbeddingProgressCommit) (semantic.EmbeddingProgress, error) {
	if _, err := s.db.ExecContext(ctx, `UPDATE semantic_generations SET state='retired' WHERE id=?`, commit.Batch.GenerationID); err != nil {
		return semantic.EmbeddingProgress{}, err
	}
	return s.Store.CommitSemanticEmbeddingProgress(ctx, commit)
}

func TestStreamBackfillPersistedLifecycle(t *testing.T) {
	for _, retire := range []bool{false, true} {
		name := "success"
		if retire {
			name = "retire-before-commit"
		}
		t.Run(name, func(t *testing.T) {
			store, _ := openTestStore(t)
			ctx := context.Background()
			library := seedBrowseCatalog(t, store)
			generation := seedEmbeddingGeneration(t, store, 512)
			if _, err := store.db.Exec(`UPDATE semantic_generations SET state='building' WHERE id=?`, generation); err != nil {
				t.Fatal(err)
			}
			if err := store.BindSemanticGenerationSpace(ctx, generation, spaceFixture()); err != nil {
				t.Fatal(err)
			}
			if _, err := store.db.Exec(`UPDATE semantic_generations SET state='active' WHERE id=?`, generation); err != nil {
				t.Fatal(err)
			}
			seedSemanticLibrarySettings(t, store, library)
			now := time.Date(2026, 9, 15, 1, 0, 0, 0, time.UTC)
			admitSemanticBackfill(t, store, library, generation, now, "stream-job-key")
			job, found, err := store.ClaimSemanticBackfill(ctx, now.Add(time.Second), time.Minute)
			if err != nil || !found {
				t.Fatalf("claim: %v", err)
			}
			transport := &spaceTransport{}
			encoder, err := semantic.NewBoundEncoder(store, transport, transport, spaceFixture())
			if err != nil {
				t.Fatal(err)
			}
			var embeddings semantic.EmbeddingRepository = store
			if retire {
				embeddings = retiringEmbeddingStore{store}
			}
			processor, err := semantic.NewStreamBackfillProcessor(store, semanticAssetSourceStub{store}, encoder, embeddings, store, func() time.Time { return now.Add(2 * time.Second) }, 1)
			if err != nil {
				t.Fatal(err)
			}
			err = processor.Process(ctx, job)
			if retire && err == nil {
				t.Fatal("retirement allowed commit")
			}
			if !retire && err != nil {
				t.Fatal(err)
			}
			var count int64
			if err := store.db.QueryRow(`SELECT COUNT(*) FROM semantic_embeddings WHERE generation_id=?`, generation).Scan(&count); err != nil {
				t.Fatal(err)
			}
			stored, err := scanSemanticJob(store.db.QueryRowContext(ctx, semanticJobSelect+` WHERE job.id=?`, job.ID))
			if err != nil {
				t.Fatal(err)
			}
			if retire {
				if count != 0 || stored.State != semantic.JobFailed {
					t.Fatalf("late result persisted: count=%d state=%s", count, stored.State)
				}
			} else if count != stored.TotalItems || stored.CompletedItems != stored.TotalItems || stored.State != semantic.JobSucceeded {
				t.Fatalf("incomplete: count=%d job=%+v", count, stored)
			}
		})
	}
}
