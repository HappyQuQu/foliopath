package sqlite

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/resourcecontrol"
	"github.com/HappyQuQu/foliopath/internal/semantic"
)

type cancellingVideoBuilder func(context.Context, string, int64, int64) (semantic.VideoEmbeddingPlan, error)

func (build cancellingVideoBuilder) BuildPlan(ctx context.Context, generation string, libraryID, assetID int64) (semantic.VideoEmbeddingPlan, error) {
	return build(ctx, generation, libraryID, assetID)
}

func TestVideoCancellationBeforeResourceAdmission(t *testing.T) {
	for _, userCancel := range []bool{true, false} {
		t.Run(map[bool]string{true: "user cancellation", false: "shutdown"}[userCancel], func(t *testing.T) {
			store, _ := openTestStore(t)
			seedStoryboardAsset(t, store.db)
			generation := seedEmbeddingGeneration(t, store, 2)
			seedSemanticLibrarySettings(t, store, 1)
			now := time.Now().UTC()
			service, err := semantic.NewVideoJobService(store, store, &semanticWakeCounter{}, func() time.Time { return now }, nil)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := service.Request(context.Background(), 1, generation, semantic.JobMissing, "admission-cancel-test"); err != nil {
				t.Fatal(err)
			}
			job, found, err := store.ClaimVideoJob(context.Background(), now, time.Minute)
			if err != nil || !found {
				t.Fatal(err)
			}
			controller, _ := resourcecontrol.NewController(resourcecontrol.Limits{Background: 1, Content: 1})
			release, err := controller.AcquireBackground(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			defer release()
			processor, _ := semantic.NewVideoJobProcessor(store, store, cancellingVideoBuilder(func(context.Context, string, int64, int64) (semantic.VideoEmbeddingPlan, error) {
				t.Error("inference must not start without resource admission")
				return semantic.VideoEmbeddingPlan{}, semantic.ErrImageEncoderUnavailable
			}), nil)
			limited, _ := resourcecontrol.LimitBackground(controller, processor)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			done := make(chan error, 1)
			go func() { done <- limited.Process(ctx, job) }()
			if userCancel {
				if _, err := store.CancelVideoJobOperation(context.Background(), job.OperationID, job.OperationRevision, now); err != nil {
					t.Fatal(err)
				}
			}
			cancel()
			select {
			case err := <-done:
				if !errors.Is(err, context.Canceled) {
					t.Fatal(err)
				}
			case <-time.After(6 * time.Second):
				t.Fatal("cancellation waited for resource holder")
			}
			stored, err := scanVideoJob(store.db.QueryRow(videoJobSelect+` WHERE job.id=?`, job.ID))
			want := semantic.JobRunning
			if userCancel {
				want = semantic.JobCancelled
			}
			if err != nil || stored.State != want || stored.CompletedItems != 0 {
				t.Fatalf("stored=%#v err=%v want=%s", stored, err, want)
			}
			var state string
			if err := store.db.QueryRow(`SELECT state FROM ai_model_operations WHERE id=?`, job.OperationID).Scan(&state); err != nil || state != string(want) {
				t.Fatalf("operation=%s err=%v", state, err)
			}
		})
	}
}

func TestVideoJobProcessorCancellationFinalizesWithoutLeaseRecovery(t *testing.T) {
	for _, scenario := range []struct {
		name                         string
		requestCancel, cancelContext bool
		buildErr                     error
		wantState                    semantic.JobState
		wantCompleted                int64
	}{
		{"heartbeat cancellation", true, true, context.Canceled, semantic.JobCancelled, 0},
		{"cancel races final commit", true, false, semantic.ErrStoryboardNotReady, semantic.JobCancelled, 1},
		{"cancel races model failure", true, false, semantic.ErrImageEncoderUnavailable, semantic.JobCancelled, 0},
		{"shutdown is not user cancellation", false, true, context.Canceled, semantic.JobRunning, 0},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			store, _ := openTestStore(t)
			seedStoryboardAsset(t, store.db)
			generation := seedEmbeddingGeneration(t, store, 2)
			seedSemanticLibrarySettings(t, store, 1)
			now := time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC)
			service, err := semantic.NewVideoJobService(store, store, &semanticWakeCounter{}, func() time.Time { return now }, nil)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := service.Request(context.Background(), 1, generation, semantic.JobMissing, "video-cancellation-regression"); err != nil {
				t.Fatal(err)
			}
			job, found, err := store.ClaimVideoJob(context.Background(), now, time.Minute)
			if err != nil || !found {
				t.Fatalf("claim: found=%v err=%v", found, err)
			}
			if _, err := store.db.Exec(`
				INSERT INTO libraries(id,name,name_key,root_rel_path,status,current_generation,created_at_ms,updated_at_ms)
				VALUES(2,'Other','other','other','ready',1,1,1);
				INSERT INTO directories(id,library_id,parent_id,relative_path,name,mtime_ns,last_seen_generation,direct_asset_count,recursive_asset_count)
				VALUES(2,2,NULL,'','',0,1,1,1);
				INSERT INTO assets(id,library_id,directory_id,relative_path,name,kind,media_format,mime_type,size_bytes,mtime_ns,last_seen_generation,source_fingerprint,width,height,duration_ms,probe_status,playback_status)
				SELECT 2,2,2,relative_path,name,kind,media_format,mime_type,size_bytes,mtime_ns,last_seen_generation,source_fingerprint,width,height,duration_ms,probe_status,playback_status FROM assets WHERE id=1;
			`); err != nil {
				t.Fatal(err)
			}
			seedSemanticLibrarySettings(t, store, 2)
			other, err := service.Request(context.Background(), 2, generation, semantic.JobMissing, "other-library-video-job")
			if err != nil {
				t.Fatal(err)
			}
			before, beforeFound, err := store.GetVideoJobProgress(context.Background(), generation, 2)
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			builder := cancellingVideoBuilder(func(_ context.Context, gotGeneration string, libraryID, assetID int64) (semantic.VideoEmbeddingPlan, error) {
				if gotGeneration != generation || libraryID != 1 || assetID != 1 {
					t.Fatalf("unexpected candidate %s/%d/%d", gotGeneration, libraryID, assetID)
				}
				if scenario.requestCancel {
					if _, err := store.CancelVideoJobOperation(context.Background(), job.OperationID, job.OperationRevision, now); err != nil {
						t.Fatal(err)
					}
				}
				if scenario.cancelContext {
					cancel()
				}
				return semantic.VideoEmbeddingPlan{}, scenario.buildErr
			})
			processor, err := semantic.NewVideoJobProcessor(store, store, builder, func() time.Time { return now })
			if err != nil {
				t.Fatal(err)
			}
			err = processor.Process(ctx, job)
			if scenario.cancelContext && !errors.Is(err, context.Canceled) {
				t.Fatalf("expected cancellation error: %v", err)
			}
			stored, err := scanVideoJob(store.db.QueryRow(videoJobSelect+` WHERE job.id=?`, job.ID))
			if err != nil || stored.State != scenario.wantState || stored.CompletedItems != scenario.wantCompleted {
				t.Fatalf("stored=%#v err=%v; want %s completed=%d", stored, err, scenario.wantState, scenario.wantCompleted)
			}
			var operationState string
			if err := store.db.QueryRow(`SELECT state FROM ai_model_operations WHERE id=?`, job.OperationID).Scan(&operationState); err != nil || operationState != string(scenario.wantState) {
				t.Fatalf("operation=%s err=%v", operationState, err)
			}
			if scenario.requestCancel && (stored.ErrorCode != "cancelled" || stored.LeaseExpiresAt != nil) {
				t.Fatalf("cancellation not finalized: %#v", stored)
			}
			after, afterFound, err := store.GetVideoJobProgress(context.Background(), generation, 2)
			if err != nil || beforeFound != afterFound || !reflect.DeepEqual(before, after) {
				t.Fatalf("other library progress changed: before=%#v after=%#v err=%v", before, after, err)
			}
			otherStored, err := scanVideoJob(store.db.QueryRow(videoJobSelect+` WHERE job.id=?`, other.Job.ID))
			if err != nil || !reflect.DeepEqual(other.Job, otherStored) {
				t.Fatalf("other library job changed: %#v err=%v", otherStored, err)
			}
		})
	}
}
