package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/autotag"
	"github.com/HappyQuQu/foliopath/internal/face"
	"github.com/HappyQuQu/foliopath/internal/jobs"
	"github.com/HappyQuQu/foliopath/internal/semantic"
)

type handoffFixture struct {
	s                                 *Store
	kind, table, fence, id, operation string
	now                               time.Time
	claim                             func(time.Time) error
	recover                           func(time.Time) (jobs.RecoverySummary, error)
}

func newHandoffFixture(t *testing.T, kind string) *handoffFixture {
	t.Helper()
	ctx := context.Background()
	f := &handoffFixture{kind: kind, now: time.Now().Add(time.Hour)}
	var libraryID int64
	if kind == "automatic_tags" {
		s, _, job, _ := setupAutomaticJob(t)
		f.s, f.table, f.fence, f.id, f.operation = s, "autotag_jobs", "claim_revision", job.ID, job.OperationID
		f.claim = func(now time.Time) error {
			_, found, err := s.ClaimAutomaticTagJob(ctx, now, time.Second)
			if err == nil && !found {
				return errors.New("claim missing")
			}
			return err
		}
		f.recover = func(now time.Time) (jobs.RecoverySummary, error) { return s.RecoverAutomaticTagJobs(ctx, now) }
		return f
	}
	f.s, _ = openTestStore(t)
	libraryID = seedBrowseCatalog(t, f.s)
	if kind == "semantic" {
		generation := seedEmbeddingGeneration(t, f.s, 2)
		seedSemanticLibrarySettings(t, f.s, libraryID)
		admitSemanticBackfill(t, f.s, libraryID, generation, f.now, "handoff-semantic-key")
		f.table, f.fence = "semantic_jobs", "claimed_revision"
		f.claim = func(now time.Time) error {
			j, found, err := f.s.ClaimSemanticBackfill(ctx, now, time.Second)
			if err == nil && !found {
				return errors.New("claim missing")
			}
			f.id, f.operation = j.ID, j.OperationID
			return err
		}
		f.recover = func(now time.Time) (jobs.RecoverySummary, error) {
			return f.s.RecoverExpiredSemanticBackfills(ctx, now)
		}
	} else {
		generation := "face_generation_handoff"
		seedFaceGeneration(t, f.s, generation, 2, "active", f.now)
		seedFaceReadySettings(t, f.s, libraryID, generation, f.now)
		service, err := face.NewJobService(f.s, f.s, &faceClearWaker{}, func() time.Time { return f.now }, faceClearIDs())
		if err != nil {
			t.Fatal(err)
		}
		if _, err = service.Request(ctx, libraryID, generation, face.JobMissing, "handoff-face-key"); err != nil {
			t.Fatal(err)
		}
		f.table, f.fence = "face_analysis_jobs", "claimed_revision"
		f.claim = func(now time.Time) error {
			j, found, err := f.s.ClaimFaceJob(ctx, now, time.Second)
			if err == nil && !found {
				return errors.New("claim missing")
			}
			f.id, f.operation = j.ID, j.OperationID
			return err
		}
		f.recover = func(now time.Time) (jobs.RecoverySummary, error) { return f.s.RecoverExpiredFaceJobs(ctx, now) }
	}
	if err := f.claim(f.now); err != nil {
		t.Fatal(err)
	}
	return f
}

func (f *handoffFixture) snapshot(t *testing.T) (jobs.HandoffClaim, string) {
	t.Helper()
	c := jobs.HandoffClaim{Kind: f.kind, JobID: f.id, OperationID: f.operation}
	var state string
	err := f.s.db.QueryRow(`SELECT j.`+f.fence+`,j.attempt_count,j.checkpoint_id,o.completed_items,j.state FROM `+f.table+` j JOIN ai_model_operations o ON o.id=j.operation_id WHERE j.id=?`, f.id).Scan(&c.ClaimRevision, &c.AttemptCount, &c.CheckpointID, &c.CompletedItems, &state)
	if err != nil {
		t.Fatal(err)
	}
	return c, state
}

func (f *handoffFixture) handoff(c jobs.HandoffClaim) (bool, error) {
	ctx := context.Background()
	switch f.kind {
	case "automatic_tags":
		return f.s.HandoffAutomaticTagJob(ctx, autotag.Job{ID: c.JobID, OperationID: c.OperationID, ClaimRevision: c.ClaimRevision}, f.now)
	case "semantic":
		return f.s.HandoffSemanticBackfill(ctx, semantic.BackfillJob{ID: c.JobID, OperationID: c.OperationID, ClaimedRevision: c.ClaimRevision}, f.now)
	default:
		return f.s.HandoffFaceJob(ctx, face.AnalysisJob{ID: c.JobID, OperationID: c.OperationID, ClaimedRevision: c.ClaimRevision}, f.now)
	}
}

func TestAnalysisHandoffThirdAttemptPreservesProgressAndCrashLimit(t *testing.T) {
	for _, kind := range []string{"automatic_tags", "semantic", "face"} {
		t.Run(kind, func(t *testing.T) {
			f := newHandoffFixture(t, kind)
			for i := 0; i < 2; i++ {
				f.now = f.now.Add(2 * time.Minute)
				r, err := f.recover(f.now)
				if err != nil || r.Requeued != 1 {
					t.Fatalf("recover %+v %v", r, err)
				}
				if err = f.claim(f.now); err != nil {
					t.Fatal(err)
				}
			}
			// A synthetic committed checkpoint; transition must not change either field.
			if _, err := f.s.db.Exec(`UPDATE `+f.table+` SET checkpoint_id=17 WHERE id=?`, f.id); err != nil {
				t.Fatal(err)
			}
			if _, err := f.s.db.Exec(`UPDATE ai_model_operations SET completed_items=1 WHERE id=?`, f.operation); err != nil {
				t.Fatal(err)
			}
			before, _ := f.snapshot(t)
			if before.AttemptCount != 3 {
				t.Fatal(before)
			}
			wrong := before
			wrong.ClaimRevision--
			if changed, err := f.handoff(wrong); changed || err != nil {
				t.Fatalf("stale claim: %v %v", changed, err)
			}
			if changed, err := f.handoff(before); !changed || err != nil {
				t.Fatalf("handoff: %v %v", changed, err)
			}
			if changed, err := f.handoff(before); changed || err != nil {
				t.Fatalf("duplicate: %v %v", changed, err)
			}
			var attempt, checkpoint, completed int64
			var state string
			if err := f.s.db.QueryRow(`SELECT j.attempt_count,j.checkpoint_id,o.completed_items,j.state FROM `+f.table+` j JOIN ai_model_operations o ON o.id=j.operation_id WHERE j.id=?`, f.id).Scan(&attempt, &checkpoint, &completed, &state); err != nil {
				t.Fatal(err)
			}
			if attempt != 2 || checkpoint != 17 || completed != 1 || state != "queued" {
				t.Fatalf("lost progress %d %d %d %s", attempt, checkpoint, completed, state)
			}
			var receipt int
			if err := f.s.db.QueryRow(`SELECT COUNT(*) FROM analysis_job_handoffs WHERE operation_id=? AND attempt_before=3 AND checkpoint_id=17 AND completed_items=1`, f.operation).Scan(&receipt); err != nil || receipt != 1 {
				t.Fatalf("audit %d %v", receipt, err)
			}
			if err := f.claim(f.now); err != nil {
				t.Fatal(err)
			}
			after, _ := f.snapshot(t)
			if after.AttemptCount != 3 || after.ClaimRevision <= before.ClaimRevision || after.OperationID != before.OperationID || after.CheckpointID != 17 || after.CompletedItems != 1 {
				t.Fatalf("resume %+v", after)
			}
			// An unplanned crash after resumption still exhausts the original budget.
			r, err := f.recover(f.now.Add(time.Minute))
			if err != nil || r.Interrupted != 1 || r.Requeued != 0 {
				t.Fatalf("crash budget %+v %v", r, err)
			}
			if changed, err := f.handoff(after); changed || err != nil {
				t.Fatalf("resurrected terminal %v %v", changed, err)
			}
		})
	}
}

func TestAnalysisHandoffCancellationWins(t *testing.T) {
	for _, kind := range []string{"automatic_tags", "semantic", "face"} {
		t.Run(kind, func(t *testing.T) {
			f := newHandoffFixture(t, kind)
			before, _ := f.snapshot(t)
			ctx := context.Background()
			op, err := f.s.GetAIOperation(ctx, f.operation)
			if err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "automatic_tags":
				_, err = f.s.CancelAutomaticTagOperation(ctx, f.operation, op.Revision, f.now)
			case "semantic":
				_, err = f.s.CancelSemanticBackfillOperation(ctx, f.operation, op.Revision, f.now)
			case "face":
				_, err = f.s.CancelFaceJobOperation(ctx, f.operation, op.Revision, f.now)
			}
			if err != nil {
				t.Fatal(err)
			}
			if changed, err := f.handoff(before); changed || err != nil {
				t.Fatalf("cancel overwritten %v %v", changed, err)
			}
			if _, err = f.recover(f.now.Add(2 * time.Minute)); err != nil {
				t.Fatal(err)
			}
			op, err = f.s.GetAIOperation(ctx, f.operation)
			if err != nil || op.State != "cancelled" {
				t.Fatalf("cancel outcome %+v %v", op, err)
			}
		})
	}
}

func TestStoppedAnalysisHandoffExactBatchRollback(t *testing.T) {
	f := newHandoffFixture(t, "automatic_tags")
	claim, _ := f.snapshot(t)
	wrong := claim
	wrong.AttemptCount++
	if err := f.s.HandoffStoppedAnalysisJobs(context.Background(), []jobs.HandoffClaim{claim, wrong}, f.now); !errors.Is(err, jobs.ErrHandoffConflict) {
		t.Fatal(err)
	}
	after, state := f.snapshot(t)
	if after != claim || state != "running" {
		t.Fatalf("partial commit %+v %s", after, state)
	}
	var count int
	if err := f.s.db.QueryRow(`SELECT COUNT(*) FROM analysis_job_handoffs`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("partial audit %d %v", count, err)
	}
	wrong = claim
	wrong.CheckpointID++
	if err := f.s.HandoffStoppedAnalysisJobs(context.Background(), []jobs.HandoffClaim{wrong}, f.now); !errors.Is(err, jobs.ErrHandoffConflict) {
		t.Fatal(err)
	}
	if err := f.s.HandoffStoppedAnalysisJobs(context.Background(), []jobs.HandoffClaim{claim}, f.now); err != nil {
		t.Fatal(err)
	}
	if err := f.s.HandoffStoppedAnalysisJobs(context.Background(), []jobs.HandoffClaim{claim}, f.now); !errors.Is(err, jobs.ErrHandoffConflict) {
		t.Fatal("snapshot replay must be refused", err)
	}
}
