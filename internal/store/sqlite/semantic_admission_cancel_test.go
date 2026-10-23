package sqlite

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/media"
	"github.com/HappyQuQu/foliopath/internal/resourcecontrol"
	"github.com/HappyQuQu/foliopath/internal/semantic"
)

type noAdmissionEncoder struct{ t *testing.T }

func (e noAdmissionEncoder) EncodeSemanticStream(context.Context, string, io.ReadSeeker, media.Format) ([]float32, error) {
	e.t.Error("cancelled work must not start inference")
	return nil, semantic.ErrImageEncoderUnavailable
}

func TestImageBackfillCancellationBeforeAdmission(t *testing.T) {
	for _, userCancel := range []bool{true, false} {
		for _, limited := range []bool{true, false} {
			name := "shutdown"
			if userCancel {
				name = "user-cancel"
			}
			if limited {
				name += "/resource-wait"
			} else {
				name += "/processor-entry"
			}
			t.Run(name, func(t *testing.T) {
				store, _ := openTestStore(t)
				library := seedBrowseCatalog(t, store)
				generation := seedEmbeddingGeneration(t, store, 2)
				seedSemanticLibrarySettings(t, store, library)
				now := time.Now().UTC()
				admitSemanticBackfill(t, store, library, generation, now, "image-admission-test")
				job, found, err := store.ClaimSemanticBackfill(context.Background(), now, time.Minute)
				if err != nil || !found {
					t.Fatal(err)
				}
				processor, err := semantic.NewStreamBackfillProcessor(store, semanticAssetSourceStub{store}, noAdmissionEncoder{t}, store, store, nil, 1)
				if err != nil {
					t.Fatal(err)
				}
				var run resourcecontrol.Processor[semantic.BackfillJob] = processor
				if limited {
					controller, _ := resourcecontrol.NewController(resourcecontrol.Limits{Background: 1, Content: 1})
					release, err := controller.AcquireBackground(context.Background())
					if err != nil {
						t.Fatal(err)
					}
					defer release()
					run, err = resourcecontrol.LimitBackground(controller, processor)
					if err != nil {
						t.Fatal(err)
					}
				}
				if userCancel {
					if _, err := store.CancelSemanticBackfill(context.Background(), job.ID, job.OperationRevision, now); err != nil {
						t.Fatal(err)
					}
				}
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				if err := run.Process(ctx, job); !errors.Is(err, context.Canceled) {
					t.Fatal(err)
				}
				stored, err := scanSemanticJob(store.db.QueryRow(semanticJobSelect+` WHERE job.id=?`, job.ID))
				want := semantic.JobRunning
				if userCancel {
					want = semantic.JobCancelled
				}
				if err != nil || stored.State != want || stored.CompletedItems != 0 {
					t.Fatalf("job=%+v err=%v want=%s", stored, err, want)
				}
				var state string
				if err := store.db.QueryRow(`SELECT state FROM ai_model_operations WHERE id=?`, job.OperationID).Scan(&state); err != nil || state != string(want) {
					t.Fatalf("operation=%s err=%v", state, err)
				}
			})
		}
	}
}
