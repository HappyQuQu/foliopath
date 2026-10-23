package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/resourcecontrol"
	"github.com/HappyQuQu/foliopath/internal/semantic"
)

func TestSemanticClearAdmissionCancellationPreservesEmbeddings(t *testing.T) {
	for _, userCancel := range []bool{true, false} {
		name := "shutdown"
		if userCancel {
			name = "user-cancel"
		}
		t.Run(name, func(t *testing.T) {
			store, _ := openTestStore(t)
			library := seedBrowseCatalog(t, store)
			generation := seedEmbeddingGeneration(t, store, 2)
			seedSemanticLibrarySettings(t, store, library)
			vector, err := semantic.EncodeEmbedding([]float32{1, 0}, 2)
			if err != nil {
				t.Fatal(err)
			}
			now := time.Now().UTC()
			if err := store.PutSemanticEmbeddingBatch(context.Background(), semantic.EmbeddingBatch{GenerationID: generation, LibraryID: library,
				Items: []semantic.EmbeddingItem{{AssetID: catalogAssetID(t, store, "photo-10.jpg"), SourceFingerprint: "v1:10:10", Vector: vector, CreatedAt: now}},
			}); err != nil {
				t.Fatal(err)
			}
			service, err := semantic.NewClearService(store, &semanticWakeCounter{}, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := service.Request(context.Background(), library, 1, "clear-admission-cancel"); err != nil {
				t.Fatal(err)
			}
			job, found, err := store.ClaimSemanticClear(context.Background(), time.Now().UTC(), time.Minute)
			if err != nil || !found {
				t.Fatal(err)
			}
			// A shutdown alone must not be accepted as a user cancellation.
			if _, err := store.FinishSemanticClear(context.Background(), job, semantic.JobCancelled, "", now); !errors.Is(err, semantic.ErrSemanticClearConflict) {
				t.Fatalf("unrequested cancellation: %v", err)
			}
			controller, _ := resourcecontrol.NewController(resourcecontrol.Limits{Background: 1, Content: 1})
			release, err := controller.AcquireBackground(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			defer release()
			processor, _ := semantic.NewClearProcessor(store, nil, 1)
			limited, _ := resourcecontrol.LimitBackground(controller, processor)
			if userCancel {
				if _, err := store.CancelSemanticClearOperation(context.Background(), job.OperationID, job.OperationRevision, now); err != nil {
					t.Fatal(err)
				}
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			if err := limited.Process(ctx, job); !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
			stored, err := scanSemanticClearJob(store.db.QueryRow(semanticClearJobSelect+` WHERE job.id=?`, job.ID))
			want := semantic.JobRunning
			wantSettings := "clearing"
			if userCancel {
				want, wantSettings = semantic.JobCancelled, "degraded"
			}
			if err != nil || stored.State != want || stored.CompletedItems != 0 {
				t.Fatalf("job=%+v err=%v want=%s", stored, err, want)
			}
			var count, enabled int
			var state, operation string
			if err := store.db.QueryRow(`SELECT COUNT(*) FROM semantic_embeddings WHERE library_id=?`, library).Scan(&count); err != nil || count != 1 {
				t.Fatalf("embeddings=%d err=%v", count, err)
			}
			if err := store.db.QueryRow(`SELECT enabled,state FROM ai_library_settings WHERE library_id=?`, library).Scan(&enabled, &state); err != nil || enabled != 0 || state != wantSettings {
				t.Fatalf("settings=%d/%s err=%v", enabled, state, err)
			}
			if err := store.db.QueryRow(`SELECT state FROM ai_model_operations WHERE id=?`, job.OperationID).Scan(&operation); err != nil || operation != string(want) {
				t.Fatalf("operation=%s err=%v", operation, err)
			}
		})
	}
}
