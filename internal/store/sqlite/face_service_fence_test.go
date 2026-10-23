package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
	"github.com/HappyQuQu/foliopath/internal/face"
)

type busyFaceAnalyzer struct{}

type emptyFaceAnalyzer struct{}

func (emptyFaceAnalyzer) Analyze(context.Context, int64, int64, string) ([]face.Observation, error) {
	return nil, nil
}

func (busyFaceAnalyzer) Analyze(context.Context, int64, int64, string) ([]face.Observation, error) {
	return nil, face.ErrRuntimeBusy
}

func TestFaceBusyKeepsBoundedRecoverableClaim(t *testing.T) {
	store, _ := openTestStore(t)
	library := seedBrowseCatalog(t, store)
	now := time.Now().UTC()
	ctx := context.Background()
	const generation = "face_busy_claim"
	seedFaceGeneration(t, store, generation, 2, "active", now)
	seedFaceReadySettings(t, store, library, generation, now)
	service, _ := face.NewJobService(store, store, &faceClearWaker{}, func() time.Time { return now }, faceClearIDs())
	if _, err := service.Request(ctx, library, generation, face.JobAll, "busy-claim-fixture"); err != nil {
		t.Fatal(err)
	}
	job, found, err := store.ClaimFaceJob(ctx, now, time.Minute)
	if err != nil || !found {
		t.Fatal(err)
	}
	processor, err := face.NewJobProcessor(store, store, busyFaceAnalyzer{}, store, face.ClusterProfile{CoreSimilarity: .8, EdgeSimilarity: .6, MinCoreSize: 2}, func() time.Time { return now }, 64)
	if err != nil {
		t.Fatal(err)
	}
	if err := processor.Process(ctx, job); !errors.Is(err, face.ErrRuntimeBusy) {
		t.Fatal(err)
	}
	progress, _, err := store.GetFaceJobProgress(ctx, generation, library)
	if err != nil || progress.CheckpointID != 0 || progress.Failed != 0 {
		t.Fatal(progress, err)
	}
	settings, err := store.GetFaceLibrarySettings(ctx, library)
	if err != nil || settings.State == "awaiting_model" {
		t.Fatal(settings, err)
	}
	recovery, err := store.RecoverExpiredFaceJobs(ctx, now.Add(3*time.Minute))
	if err != nil || recovery.Requeued != 1 {
		t.Fatal(recovery, err)
	}
	// Once contention clears, the same durable operation must complete, not
	// require toggling library settings or create a replacement operation.
	now = now.Add(3 * time.Minute)
	retried, found, err := store.ClaimFaceJob(ctx, now, time.Minute)
	if err != nil || !found || retried.OperationID != job.OperationID || retried.AttemptCount != 2 {
		t.Fatal(retried, err)
	}
	ready, err := face.NewJobProcessor(store, store, emptyFaceAnalyzer{}, store, face.ClusterProfile{CoreSimilarity: .8, EdgeSimilarity: .6, MinCoreSize: 2}, func() time.Time { return now }, 64)
	if err != nil {
		t.Fatal(err)
	}
	if err := ready.Process(ctx, retried); err != nil {
		t.Fatal(err)
	}
	op, err := store.GetAIOperation(ctx, job.OperationID)
	if err != nil || op.State != aimodel.OperationSucceeded {
		t.Fatal(op, err)
	}
	settings, err = store.GetFaceLibrarySettings(ctx, library)
	if err != nil || settings.State != "ready" || settings.Coverage.Failed != 0 {
		t.Fatal(settings, err)
	}
}

func TestFaceServicePublicationFence(t *testing.T) {
	for _, scenario := range []string{"available", "offline", "retired"} {
		t.Run(scenario, func(t *testing.T) {
			ctx := context.Background()
			store, _ := openTestStore(t)
			library := seedBrowseCatalog(t, store)
			now := time.Now().UTC()
			model := faceActivationModelFixture(now)
			model.StorageMode = aimodel.StorageService
			model.SourceIdentity = model.Package.ContentHash
			if _, _, err := store.RegisterAIModel(ctx, model); err != nil {
				t.Fatal(err)
			}
			const generation = "face_service_fence"
			seedFaceGeneration(t, store, generation, 2, "active", now)
			if _, err := store.db.Exec(`UPDATE face_generations SET model_id=? WHERE id=?`, model.ID, generation); err != nil {
				t.Fatal(err)
			}
			seedFaceReadySettings(t, store, library, generation, now)
			service, _ := face.NewJobService(store, store, &faceClearWaker{}, func() time.Time { return now }, faceClearIDs())
			if _, err := service.Request(ctx, library, generation, face.JobAll, "service-fence-request"); err != nil {
				t.Fatal(err)
			}
			job, found, err := store.ClaimFaceJob(ctx, now, time.Minute)
			if err != nil || !found {
				t.Fatal(found, err)
			}
			progress, _, err := store.GetFaceJobProgress(ctx, generation, library)
			if err != nil {
				t.Fatal(err)
			}
			page, err := store.ListFaceJobCandidates(ctx, library, generation, face.JobAll, 0, 1)
			if err != nil || len(page.Items) != 1 {
				t.Fatal(page, err)
			}
			candidate := page.Items[0]
			if scenario == "offline" {
				if _, err := store.db.Exec(`UPDATE ai_models SET state='unavailable' WHERE id=?`, model.ID); err != nil {
					t.Fatal(err)
				}
			}
			if scenario == "retired" {
				if _, err := store.db.Exec(`UPDATE face_generations SET state='retired' WHERE id=?`, generation); err != nil {
					t.Fatal(err)
				}
			}
			_, err = store.CommitFaceJobProgress(ctx, face.JobProgressCommit{JobID: job.ID, ClaimedRevision: job.ClaimedRevision, ExpectedProgressRevision: progress.Revision, ExpectedCheckpointID: 0, NextCheckpointID: candidate.AssetID, SourceFingerprint: candidate.SourceFingerprint, UpdatedAt: now,
				Batch: face.ObservationBatch{GenerationID: generation, LibraryID: library, AssetID: candidate.AssetID, UpdatedAt: now}})
			if scenario == "available" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if !errors.Is(err, face.ErrFaceModelUnavailable) {
				t.Fatal(err)
			}
			after, _, err := store.GetFaceJobProgress(ctx, generation, library)
			if err != nil || after.CheckpointID != 0 || after.Completed != 0 || after.Revision != progress.Revision {
				t.Fatal(after, err)
			}
			if err := store.RebuildFaceClusters(ctx, generation, library, job.ID, job.ClaimedRevision, face.ClusterProfile{CoreSimilarity: .8, EdgeSimilarity: .6, MinCoreSize: 2}, now); err == nil {
				t.Fatal("unavailable cluster published")
			}
			var count int
			if err := store.db.QueryRow(`SELECT COUNT(*) FROM face_asset_results WHERE generation_id=?`, generation).Scan(&count); err != nil || count != 0 {
				t.Fatal(count, err)
			}
			// Simulate the zero-work completion boundary independently of progress.
			if _, err := store.db.Exec(`UPDATE ai_model_operations SET completed_items=0,total_items=0 WHERE id=?`, job.OperationID); err != nil {
				t.Fatal(err)
			}
			if _, err := store.FinishFaceJob(ctx, job, true, "", now); !errors.Is(err, face.ErrFaceModelUnavailable) {
				t.Fatal("offline completion", err)
			}
			op, err := store.GetAIOperation(ctx, job.OperationID)
			if err != nil || op.State != aimodel.OperationRunning {
				t.Fatal(op, err)
			}
		})
	}
}

func TestFaceAdmissionCancellationPreservesShutdownClaim(t *testing.T) {
	for _, userCancel := range []bool{false, true} {
		t.Run(map[bool]string{false: "shutdown", true: "user-cancel"}[userCancel], func(t *testing.T) {
			store, _ := openTestStore(t)
			library := seedBrowseCatalog(t, store)
			now := time.Now().UTC()
			const generation = "face_admission_cancel"
			seedFaceGeneration(t, store, generation, 2, "active", now)
			seedFaceReadySettings(t, store, library, generation, now)
			service, _ := face.NewJobService(store, store, &faceClearWaker{}, func() time.Time { return now }, faceClearIDs())
			requested, err := service.Request(context.Background(), library, generation, face.JobAll, "admission-cancel-fixture")
			if err != nil {
				t.Fatal(err)
			}
			job, found, err := store.ClaimFaceJob(context.Background(), now, time.Minute)
			if err != nil || !found {
				t.Fatal(err)
			}
			if userCancel {
				if _, err := store.CancelFaceJobOperation(context.Background(), requested.Job.OperationID, job.OperationRevision, now); err != nil {
					t.Fatal(err)
				}
			}
			processor, err := face.NewJobProcessor(store, store, &scriptedFaceAnalyzer{}, store, face.ClusterProfile{CoreSimilarity: .8, EdgeSimilarity: .6, MinCoreSize: 2}, func() time.Time { return now }, 64)
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			processor.BackgroundAdmissionCancelled(ctx, job)
			op, err := store.GetAIOperation(context.Background(), job.OperationID)
			if err != nil {
				t.Fatal(err)
			}
			want := aimodel.OperationRunning
			if userCancel {
				want = aimodel.OperationCancelled
			}
			if op.State != want {
				t.Fatal(op.State, want)
			}
		})
	}
}
