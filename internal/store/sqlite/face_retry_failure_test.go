package sqlite

import (
	"context"
	"github.com/HappyQuQu/foliopath/internal/face"
	"testing"
	"time"
)

type countedEmptyFaceAnalyzer struct{ calls int }

func (a *countedEmptyFaceAnalyzer) Analyze(context.Context, int64, int64, string) ([]face.Observation, error) {
	a.calls++
	return nil, nil
}

func TestFaceMissingRetriesRebuildFailureDespitePriorSuccessfulResult(t *testing.T) {
	ctx := context.Background()
	store, _ := openTestStore(t)
	lib := seedBrowseCatalog(t, store)
	now := time.Now().UTC()
	const gen = "face_generation_retry_old_success"
	seedFaceGeneration(t, store, gen, 2, "active", now)
	seedFaceReadySettings(t, store, lib, gen, now)
	service, _ := face.NewJobService(store, store, &faceClearWaker{}, func() time.Time { return now }, faceClearIDs())
	run := func(mode face.JobMode, key string, analyzer face.JobAnalyzer) {
		t.Helper()
		if _, err := service.Request(ctx, lib, gen, mode, key); err != nil {
			t.Fatal(err)
		}
		j, found, err := store.ClaimFaceJob(ctx, now, time.Minute)
		if err != nil || !found {
			t.Fatalf("claim=%v %v", found, err)
		}
		p, _ := face.NewJobProcessor(store, store, analyzer, store, face.ClusterProfile{CoreSimilarity: .9, EdgeSimilarity: .8, MinCoreSize: 2}, func() time.Time { return now }, 10)
		if err := p.Process(ctx, j); err != nil {
			t.Fatal(err)
		}
	}
	first := &countedEmptyFaceAnalyzer{}
	run(face.JobMissing, "retry-diagnostics-first", first)
	failed := &failingFaceAnalyzer{}
	run(face.JobAll, "retry-diagnostics-rebuild", failed)
	if first.calls == 0 || failed.calls != first.calls {
		t.Fatalf("first=%d failed=%d", first.calls, failed.calls)
	}
	counts, err := store.CountFaceJobCandidates(ctx, lib, gen, face.JobMissing)
	if err != nil || counts.Pending != int64(first.calls) {
		t.Fatalf("pending=%+v %v", counts, err)
	}
	final := &countedEmptyFaceAnalyzer{}
	run(face.JobMissing, "retry-diagnostics-recover", final)
	if final.calls != first.calls {
		t.Fatalf("retried %d of %d", final.calls, first.calls)
	}
	snapshot, err := store.GetFaceLibrarySettings(ctx, lib)
	if err != nil || snapshot.State != "ready" || snapshot.Coverage.Failed != 0 || snapshot.Diagnostics.RecordedFailures != 0 {
		t.Fatalf("snapshot=%+v %v", snapshot, err)
	}
	var failures int
	store.db.QueryRow(`SELECT COUNT(*) FROM face_asset_failures WHERE library_id=?`, lib).Scan(&failures)
	if failures != 0 {
		t.Fatal("successful retry retained stale failure markers")
	}
	run(face.JobAll, "retry-diagnostics-fail-again", &failingFaceAnalyzer{})
	beforeClear, _ := store.GetFaceLibrarySettings(ctx, lib)
	clear, _ := face.NewClearService(store, &faceClearWaker{}, func() time.Time { return now }, nil)
	if _, err := clear.RequestDerived(ctx, lib, beforeClear.Revision, "diagnostic-clear-derived"); err != nil {
		t.Fatal(err)
	}
	clearJob, found, err := store.ClaimFaceClear(ctx, now, time.Minute)
	if err != nil || !found {
		t.Fatalf("clear=%v %v", found, err)
	}
	clearWorker, _ := face.NewClearProcessor(store, func() time.Time { return now }, 1)
	if err := clearWorker.Process(ctx, clearJob); err != nil {
		t.Fatal(err)
	}
	store.db.QueryRow(`SELECT COUNT(*) FROM face_asset_failures WHERE library_id=?`, lib).Scan(&failures)
	if failures != 0 {
		t.Fatal("derived clear retained diagnostic markers")
	}
	cleared, err := store.GetFaceLibrarySettings(ctx, lib)
	if err != nil || cleared.Diagnostics.LatestTask != nil || cleared.Diagnostics.RecordedFailures != 0 {
		t.Fatalf("cleared=%+v %v", cleared.Diagnostics, err)
	}
}
