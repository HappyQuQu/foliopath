package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/resourcecontrol"
	"github.com/HappyQuQu/foliopath/internal/semantic"
)

func TestTagReviewClearAdmissionCancellation(t *testing.T) {
	for _, requested := range []bool{true, false} {
		name := "shutdown"
		if requested {
			name = "user-cancel"
		}
		t.Run(name, func(t *testing.T) {
			store, _ := openTestStore(t)
			library := seedBrowseCatalog(t, store)
			ctx := context.Background()
			service, err := semantic.NewTagReviewClearService(store, &semanticWakeCounter{}, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := service.Request(ctx, library, 1, "tagclear-admission-cancel"); err != nil {
				t.Fatal(err)
			}
			job, found, err := store.ClaimTagReviewClear(ctx, time.Now().UTC(), time.Minute)
			if err != nil || !found {
				t.Fatal(err)
			}
			if requested {
				if _, err := store.CancelTagReviewClearOperation(ctx, job.OperationID, job.OperationRevision, time.Now().UTC()); err != nil {
					t.Fatal(err)
				}
			}
			controller, _ := resourcecontrol.NewController(resourcecontrol.Limits{Background: 1, Content: 1})
			release, err := controller.AcquireBackground(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer release()
			processor, err := semantic.NewTagReviewClearProcessor(store, nil, 1)
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
			stored, err := scanTagReviewClearJob(store.db.QueryRow(tagReviewClearJobSelect+` WHERE job.id=?`, job.ID))
			want := semantic.JobRunning
			if requested {
				want = semantic.JobCancelled
			}
			if err != nil || stored.State != want || stored.DeletedCount != 0 {
				t.Fatalf("job=%#v want=%s err=%v", stored, want, err)
			}
			var state string
			if err := store.db.QueryRow(`SELECT state FROM ai_model_operations WHERE id=?`, job.OperationID).Scan(&state); err != nil || state != string(want) {
				t.Fatalf("operation=%s err=%v", state, err)
			}
			if !requested {
				if _, err := store.FinishTagReviewClear(ctx, job, semantic.JobCancelled, "", time.Now().UTC()); !errors.Is(err, semantic.ErrTagReviewClearConflict) {
					t.Fatalf("unrequested cancel: %v", err)
				}
			}
		})
	}
}
