package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/resourcecontrol"
	"github.com/HappyQuQu/foliopath/internal/semantic"
)

type forbiddenTagBuilder struct{ t *testing.T }

func (b forbiddenTagBuilder) BuildTagPlan(context.Context, string, string, int64, int64) (semantic.TagSuggestionPlan, error) {
	b.t.Error("unexpected plan build")
	return semantic.TagSuggestionPlan{}, semantic.ErrInvalidTagJob
}

func TestTagJobCancellationBoundaries(t *testing.T) {
	for _, mode := range []string{"admission-cancel", "shutdown", "finish-success", "finish-failure"} {
		t.Run(mode, func(t *testing.T) {
			store, _ := openTestStore(t)
			library := seedBrowseCatalog(t, store)
			generation := seedEmbeddingGeneration(t, store, 2)
			ctx := context.Background()
			now := time.Now().UTC()
			tag, _, err := store.CreateTag(ctx, "candidate", "candidate", now)
			if err != nil {
				t.Fatal(err)
			}
			vocab, err := store.GetActiveTagVocabulary(ctx)
			if err != nil {
				t.Fatal(err)
			}
			vocab, err = store.PublishTagVocabulary(ctx, "aiv_cancel_candidate", vocab.Revision, []int64{tag.ID}, now)
			if err != nil {
				t.Fatal(err)
			}
			service, err := semantic.NewTagJobService(store, store, &semanticWakeCounter{}, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := service.Request(ctx, library, generation, vocab.ID, semantic.JobMissing, "tag-job-cancel-test"); err != nil {
				t.Fatal(err)
			}
			job, found, err := store.ClaimTagJob(ctx, now, time.Minute)
			if err != nil || !found {
				t.Fatalf("claim found=%v err=%v", found, err)
			}
			if mode != "shutdown" {
				if _, err := store.CancelTagJobOperation(ctx, job.OperationID, job.OperationRevision, now); err != nil {
					t.Fatal(err)
				}
			}
			switch mode {
			case "finish-success", "finish-failure":
				outcome, code := semantic.JobSucceeded, ""
				if mode == "finish-failure" {
					outcome, code = semantic.JobFailed, "internal_error"
				}
				if _, err := store.FinishTagJob(ctx, job, outcome, code, now); err != nil {
					t.Fatal(err)
				}
			default:
				controller, _ := resourcecontrol.NewController(resourcecontrol.Limits{Background: 1, Content: 1})
				release, err := controller.AcquireBackground(ctx)
				if err != nil {
					t.Fatal(err)
				}
				defer release()
				processor, err := semantic.NewTagJobProcessor(store, store, forbiddenTagBuilder{t}, nil)
				if err != nil {
					t.Fatal(err)
				}
				limited, err := resourcecontrol.LimitBackground(controller, processor)
				if err != nil {
					t.Fatal(err)
				}
				cancelled, cancel := context.WithCancel(ctx)
				cancel()
				if err := limited.Process(cancelled, job); !errors.Is(err, context.Canceled) {
					t.Fatal(err)
				}
			}
			var state, operation string
			var lease bool
			if err := store.db.QueryRow(`SELECT job.state,operation.state,job.lease_expires_ms IS NOT NULL FROM semantic_tag_jobs job JOIN ai_model_operations operation ON operation.id=job.operation_id WHERE job.id=?`, job.ID).Scan(&state, &operation, &lease); err != nil {
				t.Fatal(err)
			}
			want := "cancelled"
			if mode == "shutdown" {
				want = "running"
			}
			if state != want || operation != want || lease != (mode == "shutdown") {
				t.Fatalf("job=%s operation=%s lease=%v", state, operation, lease)
			}
		})
	}
}
