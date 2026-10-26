package sqlite

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/jobs"
	"github.com/HappyQuQu/foliopath/internal/ocr"
)

type ocrWorkerAnalyzer struct {
	calls       int
	run         func(context.Context, int64, ocr.Candidate) (ocr.StoredResult, error)
	fingerprint string
}

func (a *ocrWorkerAnalyzer) Analyze(ctx context.Context, id int64, c ocr.Candidate) (ocr.StoredResult, error) {
	a.calls++
	if a.run != nil {
		return a.run(ctx, id, c)
	}
	return resultWithCandidate(ocr.Job{Fingerprint: a.fingerprint}, c.AssetID, c.SourceFingerprint).Value, nil
}
func TestOCRWorkerCompletesMissingAndPreservesFailureClassifications(t *testing.T) {
	s, id, v, now := setupOCR(t)
	ctx := context.Background()
	admitOCR(t, s, v, ocr.Missing, "worker0001", now)
	j := claimOCR(t, s, now)
	analyzer := &ocrWorkerAnalyzer{fingerprint: v.Fingerprint}
	worker, err := ocr.NewWorker(s, analyzer, v.Fingerprint)
	if err != nil {
		t.Fatal(err)
	}
	if err = worker.Process(ctx, j); err != nil {
		t.Fatal(err)
	}
	terminal := reloadOCR(t, s, j)
	coverage, err := s.GetOCRCoverage(ctx, id)
	if err != nil || terminal.State != "succeeded" || terminal.Completed != 3 || coverage.Ready != 3 || coverage.Empty != 3 || coverage.Pending != 0 {
		t.Fatal(terminal, coverage, err)
	}
	admitOCR(t, s, v, ocr.Missing, "worker0002", now)
	j = claimOCR(t, s, now)
	if err = worker.Process(ctx, j); err != nil || analyzer.calls != 3 {
		t.Fatal(analyzer.calls, err)
	}
}
func TestOCRWorkerServiceErrorsNeverWriteBadSource(t *testing.T) {
	for _, cause := range []error{ocr.ErrRuntimeUnavailable, ocr.ErrRuntimeIdentity, ocr.ErrRuntimeOutput, ocr.ErrRuntimeOutputLimit, ocr.ErrSourceChanged, ocr.ErrPreprocessFailed} {
		t.Run(cause.Error(), func(t *testing.T) {
			s, _, v, now := setupOCR(t)
			admitOCR(t, s, v, ocr.Missing, "workererr1", now)
			j := claimOCR(t, s, now)
			analyzer := &ocrWorkerAnalyzer{run: func(context.Context, int64, ocr.Candidate) (ocr.StoredResult, error) {
				return ocr.StoredResult{}, cause
			}}
			worker, _ := ocr.NewWorker(s, analyzer, v.Fingerprint)
			if err := worker.Process(context.Background(), j); err != nil {
				t.Fatal(err)
			}
			terminal := reloadOCR(t, s, j)
			if terminal.State != "failed" || terminal.Completed != 0 || terminal.Failed != 0 {
				t.Fatal(terminal)
			}
			var n int
			if err := s.db.QueryRow(`SELECT COUNT(*) FROM ocr_asset_results`).Scan(&n); err != nil || n != 0 {
				t.Fatal(n, err)
			}
		})
	}
}
func TestOCRWorkerBusyDefersExactClaimWithoutBurningAttempt(t *testing.T) {
	s, _, v, now := setupOCR(t)
	admitOCR(t, s, v, ocr.Missing, "workerbusy", now)
	j := claimOCR(t, s, now)
	// The store owns the future delay and attempt transition; no time.Sleep in worker.
	if err := s.DeferOCRJob(context.Background(), j, now, 5*time.Second); err != nil {
		t.Fatal(err)
	}
	queued := reloadOCR(t, s, j)
	if queued.State != "queued" || queued.Attempts != 0 || queued.ClaimRevision <= j.ClaimRevision {
		t.Fatal(queued)
	}
	if _, found, err := s.ClaimOCRJob(context.Background(), now.Add(time.Second), time.Minute); err != nil || found {
		t.Fatal(found, err)
	}
	next := claimOCR(t, s, now.Add(5*time.Second))
	if next.Attempts != 1 {
		t.Fatal(next)
	}
	if err := s.RefreshOCRLease(context.Background(), j, now.Add(5*time.Second), time.Minute); !errors.Is(err, ocr.ErrConflict) {
		t.Fatal(err)
	}
}
func TestOCRWorkerPoolPlannedShutdownAndCancellationRace(t *testing.T) {
	for _, cancelJob := range []bool{false, true} {
		t.Run(map[bool]string{false: "planned-handoff", true: "cancellation-wins"}[cancelJob], func(t *testing.T) {
			s, id, v, _ := setupOCR(t)
			now := time.Now()
			admitOCR(t, s, v, ocr.Missing, "workerplan", now)
			entered := make(chan struct{})
			analyzer := &ocrWorkerAnalyzer{run: func(ctx context.Context, _ int64, _ ocr.Candidate) (ocr.StoredResult, error) {
				close(entered)
				<-ctx.Done()
				return ocr.StoredResult{}, ctx.Err()
			}}
			worker, _ := ocr.NewWorker(s, analyzer, v.Fingerprint)
			pool, err := jobs.NewWorkerPool(ocr.Queue{Repository: s}, worker, jobs.NewSignal(), jobs.WorkerOptions{Workers: 1, HeartbeatInterval: 10 * time.Millisecond, LeaseDuration: time.Second, IdlePollInterval: 10 * time.Millisecond})
			if err != nil {
				t.Fatal(err)
			}
			ctx, stop := context.WithCancelCause(context.Background())
			defer stop(context.Canceled)
			done := make(chan error, 1)
			go func() { done <- pool.Run(ctx) }()
			select {
			case <-entered:
			case <-time.After(5 * time.Second):
				t.Fatal("not processing")
			}
			j, found, err := s.GetLatestOCRJob(context.Background(), id)
			if err != nil || !found {
				t.Fatal(j, err)
			}
			if cancelJob {
				if err = s.CancelOCRJob(context.Background(), j.ID, j.ClaimRevision, time.Now()); err != nil {
					t.Fatal(err)
				}
			}
			stop(jobs.ErrPlannedShutdown)
			select {
			case err := <-done:
				if err != nil {
					t.Fatal(err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("worker failed to exit")
			}
			after := reloadOCR(t, s, j)
			if cancelJob {
				if after.State != "cancelled" || after.Attempts != 1 {
					t.Fatal(after)
				}
			} else {
				if after.State != "queued" || after.Attempts != 0 || after.Completed != 0 {
					t.Fatal(after)
				}
				var receipts int
				if err = s.db.QueryRow(`SELECT COUNT(*) FROM ocr_job_handoffs`).Scan(&receipts); err != nil || receipts != 1 {
					t.Fatal(receipts, err)
				}
			}
		})
	}
}
func TestOCRWorkerSourceFailureMissingNeverLoopsAndClearResumes(t *testing.T) {
	s, id, v, now := setupOCR(t)
	admitOCR(t, s, v, ocr.Missing, "workerbad1", now)
	j := claimOCR(t, s, now)
	analyzer := &ocrWorkerAnalyzer{run: func(_ context.Context, _ int64, c ocr.Candidate) (ocr.StoredResult, error) {
		return ocr.StoredResult{Candidate: c, Fingerprint: v.Fingerprint, Outcome: "failed", ErrorCode: "invalid_media"}, nil
	}}
	worker, _ := ocr.NewWorker(s, analyzer, v.Fingerprint)
	if err := worker.Process(context.Background(), j); err != nil {
		t.Fatal(err)
	}
	coverage, err := s.GetOCRCoverage(context.Background(), id)
	if err != nil || coverage.Failed != 3 || coverage.Pending != 0 {
		t.Fatal(coverage, err)
	}
	service, err := ocr.NewService(s, persistenceIdentity(), jobs.NewSignal())
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = service.Request(context.Background(), id, v.Revision, ocr.Missing, "workerbad2"); err != nil {
		t.Fatal(err)
	}
	j = claimOCR(t, s, time.Now())
	if err = worker.Process(context.Background(), j); err != nil || analyzer.calls != 3 {
		t.Fatal(analyzer.calls, err)
	}
	if _, err = service.Job(context.Background(), id+1, j.ID); !errors.Is(err, ocr.ErrNotFound) {
		t.Fatal(err)
	}
	if _, err = service.Clear(context.Background(), id, v.Revision); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- service.Run(ctx) }()
	deadline := time.After(5 * time.Second)
	for {
		status, err := service.Get(context.Background(), id)
		if err != nil {
			t.Fatal(err)
		}
		if !status.Settings.Clearing {
			if status.Coverage.Pending != 3 || status.Coverage.Failed != 0 || status.Settings.Enabled {
				t.Fatal(status)
			}
			break
		}
		select {
		case <-deadline:
			t.Fatal("clear did not finish")
		case <-time.After(10 * time.Millisecond):
		}
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
func TestOCRHandoffRetainsLatestProgressAndRejectsForgedOrExpiredClaims(t *testing.T) {
	s, _, v, now := setupOCR(t)
	admitOCR(t, s, v, ocr.Missing, "handoff001", now)
	j := claimOCR(t, s, now)
	c := resultOCR(t, s, j, now)
	if err := s.CommitOCRResult(context.Background(), c, now); err != nil {
		t.Fatal(err)
	}
	forged := j
	forged.Fingerprint = strings.Repeat("c", 64)
	if err := s.HandoffOCRJob(context.Background(), forged, now); err != nil {
		t.Fatal(err)
	}
	if reloadOCR(t, s, j).State != "running" {
		t.Fatal("forged handoff")
	}
	if err := s.HandoffOCRJob(context.Background(), j, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if reloadOCR(t, s, j).State != "running" {
		t.Fatal("expired handoff")
	}
	if err := s.HandoffOCRJob(context.Background(), j, now); err != nil {
		t.Fatal(err)
	}
	after := reloadOCR(t, s, j)
	if after.CheckpointID != c.Value.AssetID || after.Completed != 1 || after.Attempts != 0 || after.State != "queued" {
		t.Fatal(after)
	}
}

func TestOCRWorkerWaitsForTransientScanWithoutReanalysis(t *testing.T) {
	s, id, v, now := setupOCR(t)
	admitOCR(t, s, v, ocr.Missing, "scanwait01", now)
	j := claimOCR(t, s, now)
	analyzer := &ocrWorkerAnalyzer{fingerprint: v.Fingerprint}
	analyzer.run = func(ctx context.Context, _ int64, c ocr.Candidate) (ocr.StoredResult, error) {
		if analyzer.calls == 1 {
			if _, err := s.db.Exec(`UPDATE libraries SET status='scanning' WHERE id=?`, id); err != nil {
				return ocr.StoredResult{}, err
			}
			// Heartbeat must retain this claim even during a transient catalog scan.
			if err := s.RefreshOCRLease(ctx, j, time.Now(), time.Minute); err != nil {
				return ocr.StoredResult{}, err
			}
			go func() {
				time.Sleep(50 * time.Millisecond)
				_, _ = s.db.Exec(`UPDATE libraries SET status='ready' WHERE id=?`, id)
			}()
		}
		return resultWithCandidate(j, c.AssetID, c.SourceFingerprint).Value, nil
	}
	worker, _ := ocr.NewWorker(s, analyzer, v.Fingerprint)
	if err := worker.Process(context.Background(), j); err != nil {
		t.Fatal(err)
	}
	after := reloadOCR(t, s, j)
	if after.State != "succeeded" || after.Completed != 3 || analyzer.calls != 3 {
		t.Fatal(after, analyzer.calls)
	}
}

func TestOCRClearAcknowledgementAndSnapshotStayConsistentDuringDrain(t *testing.T) {
	s, id, v, now := setupOCR(t)
	admitOCR(t, s, v, ocr.Missing, "clearack01", now)
	j := claimOCR(t, s, now)
	analyzer := &ocrWorkerAnalyzer{fingerprint: v.Fingerprint}
	worker, _ := ocr.NewWorker(s, analyzer, v.Fingerprint)
	if err := worker.Process(context.Background(), j); err != nil {
		t.Fatal(err)
	}
	service, _ := ocr.NewService(s, persistenceIdentity(), jobs.NewSignal())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- service.Run(ctx) }()
	// Clear may finish between admission and its read response. It must still
	// acknowledge a successful write, with one internally consistent snapshot.
	status, err := service.Clear(context.Background(), id, v.Revision)
	if err != nil || status.Settings.Enabled {
		t.Fatal(status, err)
	}
	for i := 0; i < 20; i++ {
		status, err = service.Get(context.Background(), id)
		if err != nil || status.Settings.Enabled || status.Coverage.Ready+status.Coverage.Failed+status.Coverage.Pending != 3 {
			t.Fatal(status, err)
		}
	}
	cancel()
	if err = <-done; err != nil {
		t.Fatal(err)
	}
}
