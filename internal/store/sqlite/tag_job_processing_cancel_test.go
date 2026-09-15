package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/semantic"
)

type tagPlanHook struct {
	next semantic.TagPlanBuilder
	hook func()
}

func (b tagPlanHook) BuildTagPlan(ctx context.Context, g, s string, l, a int64) (semantic.TagSuggestionPlan, error) {
	plan, err := b.next.BuildTagPlan(ctx, g, s, l, a)
	b.hook()
	return plan, err
}

type tagFinishHook struct {
	*Store
	hook func()
}

func (q tagFinishHook) FinishTagJob(ctx context.Context, j semantic.TagJob, s semantic.JobState, c string, n time.Time) (semantic.TagJob, error) {
	q.hook()
	return q.Store.FinishTagJob(ctx, j, s, c, n)
}

func TestTagJobNonemptyProcessingCancellation(t *testing.T) {
	for _, mode := range []string{"before-commit", "heartbeat", "shutdown", "after-commit"} {
		t.Run(mode, func(t *testing.T) {
			store, _ := openTestStore(t)
			lib := seedBrowseCatalog(t, store)
			gen := seedEmbeddingGeneration(t, store, 2)
			ctx := context.Background()
			now := time.Now().UTC()
			tag, _, err := store.CreateTag(ctx, "candidate", "candidate", now)
			if err != nil {
				t.Fatal(err)
			}
			v, err := store.GetActiveTagVocabulary(ctx)
			if err != nil {
				t.Fatal(err)
			}
			v, err = store.PublishTagVocabulary(ctx, "aiv_processing_cancel", v.Revision, []int64{tag.ID}, now)
			if err != nil {
				t.Fatal(err)
			}
			asset := catalogAssetID(t, store, "photo-10.jpg")
			var fingerprint string
			if err := store.db.QueryRow(`SELECT source_fingerprint FROM assets WHERE id=?`, asset).Scan(&fingerprint); err != nil {
				t.Fatal(err)
			}
			vector, _ := semantic.EncodeEmbedding([]float32{1, 0}, 2)
			if err := store.PutSemanticEmbeddingBatch(ctx, semantic.EmbeddingBatch{GenerationID: gen, LibraryID: lib, Items: []semantic.EmbeddingItem{{AssetID: asset, SourceFingerprint: fingerprint, Vector: vector, CreatedAt: now}}}); err != nil {
				t.Fatal(err)
			}
			if err := store.PutTagEmbeddingBatch(ctx, semantic.TagEmbeddingBatch{GenerationID: gen, SnapshotID: v.ID, CreatedAt: now, Items: []semantic.TagEmbedding{{TagID: tag.ID, Vector: vector}}}); err != nil {
				t.Fatal(err)
			}
			svc, err := semantic.NewTagJobService(store, store, &semanticWakeCounter{}, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := svc.Request(ctx, lib, gen, v.ID, semantic.JobMissing, "processing-cancel-key"); err != nil {
				t.Fatal(err)
			}
			job, found, err := store.ClaimTagJob(ctx, now, time.Minute)
			if err != nil || !found || job.TotalItems != 1 {
				t.Fatalf("claim=%#v found=%v err=%v", job, found, err)
			}
			cancelRequest := func() {
				var rev int64
				if err := store.db.QueryRow(`SELECT revision FROM ai_model_operations WHERE id=?`, job.OperationID).Scan(&rev); err != nil {
					t.Fatal(err)
				}
				if _, err := store.CancelTagJobOperation(ctx, job.OperationID, rev, time.Now().UTC()); err != nil {
					t.Fatal(err)
				}
			}
			runCtx, cancel := context.WithCancel(ctx)
			defer cancel()
			builder, err := semantic.NewControlledTagPlanBuilder(store, nil, nil, semantic.DefaultTagSuggestionThreshold)
			if err != nil {
				t.Fatal(err)
			}
			var plans semantic.TagPlanBuilder = builder
			var queue semantic.TagJobQueue = store
			if mode == "after-commit" {
				queue = tagFinishHook{store, cancelRequest}
			} else {
				plans = tagPlanHook{builder, func() {
					if mode != "shutdown" {
						cancelRequest()
					}
					if mode == "heartbeat" || mode == "shutdown" {
						cancel()
					}
				}}
			}
			processor, err := semantic.NewTagJobProcessor(store, queue, plans, nil)
			if err != nil {
				t.Fatal(err)
			}
			err = processor.Process(runCtx, job)
			if mode == "after-commit" && err != nil {
				t.Fatal(err)
			}
			if (mode == "heartbeat" || mode == "shutdown") && !errors.Is(err, context.Canceled) {
				t.Fatalf("context error=%v", err)
			}
			if mode == "before-commit" && !errors.Is(err, semantic.ErrTagJobConflict) {
				t.Fatalf("commit error=%v", err)
			}
			var state, op string
			var completed, count int64
			var lease bool
			if err := store.db.QueryRow(`SELECT job.state,operation.state,operation.completed_items,job.lease_expires_ms IS NOT NULL FROM semantic_tag_jobs job JOIN ai_model_operations operation ON operation.id=job.operation_id WHERE job.id=?`, job.ID).Scan(&state, &op, &completed, &lease); err != nil {
				t.Fatal(err)
			}
			want := "cancelled"
			if mode == "shutdown" {
				want = "running"
			}
			wantCount := int64(0)
			if mode == "after-commit" {
				wantCount = 1
			}
			if state != want || op != want || completed != wantCount || lease != (mode == "shutdown") {
				t.Fatalf("state=%s/%s completed=%d lease=%v", state, op, completed, lease)
			}
			if err := store.db.QueryRow(`SELECT COUNT(*) FROM ai_tag_suggestions WHERE library_id=?`, lib).Scan(&count); err != nil || count != wantCount {
				t.Fatalf("suggestions=%d err=%v", count, err)
			}
		})
	}
}
