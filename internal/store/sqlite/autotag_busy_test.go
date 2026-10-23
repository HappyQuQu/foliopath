package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
	"github.com/HappyQuQu/foliopath/internal/autotag"
)

func TestAutomaticTagBusyDeferralKeepsProgressAndFencesOldClaim(t *testing.T) {
	s, id, j, now := setupAutomaticJob(t)
	ctx := context.Background()
	first := automaticCommit(t, s, j)
	if err := s.CommitAutomaticTagResult(ctx, first, now); err != nil {
		t.Fatal(err)
	}
	j.CheckpointID = first.Asset.ID
	for cycle := 0; cycle < 5; cycle++ {
		late := automaticCommit(t, s, j)
		if err := s.DeferAutomaticTagJob(ctx, j, now, autotag.BusyRetryDelay); err != nil {
			t.Fatal(err)
		}
		if err := s.CommitAutomaticTagResult(ctx, late, now); !errors.Is(err, autotag.ErrConflict) {
			t.Fatalf("late commit=%v", err)
		}
		if _, found, err := s.ClaimAutomaticTagJob(ctx, now.Add(autotag.BusyRetryDelay-time.Millisecond), time.Minute); err != nil || found {
			t.Fatalf("early claim found=%v err=%v", found, err)
		}
		op, err := s.GetAIOperation(ctx, j.OperationID)
		if err != nil || op.State != aimodel.OperationQueued || op.CompletedItems != 1 || op.TotalItems != nil || op.ErrorCode != "" {
			t.Fatalf("deferred operation=%+v error=%v", op, err)
		}
		settings, err := s.GetAutomaticTagSettings(ctx, id, j.ModelIdentity)
		if err != nil || settings.AutomaticBlocked || settings.Coverage.Completed != 1 || settings.Coverage.Failed != 0 {
			t.Fatalf("busy changed coverage/block=%+v %v", settings, err)
		}
		now = now.Add(autotag.BusyRetryDelay)
		next, found, err := s.ClaimAutomaticTagJob(ctx, now, time.Minute)
		if err != nil || !found || next.ID != j.ID || next.OperationID != j.OperationID || next.CheckpointID != j.CheckpointID || next.ClaimRevision <= j.ClaimRevision || next.AttemptCount != 1 {
			t.Fatalf("same-job resume=%+v %v", next, err)
		}
		j = next
	}
	if err := s.CommitAutomaticTagResult(ctx, automaticCommit(t, s, j), now); err != nil {
		t.Fatal(err)
	}
	var operations, jobs int
	if err := s.db.QueryRow(`SELECT (SELECT COUNT(*) FROM ai_model_operations WHERE kind LIKE 'automatic_tags_%'),(SELECT COUNT(*) FROM autotag_jobs)`).Scan(&operations, &jobs); err != nil || operations != 1 || jobs != 1 {
		t.Fatalf("operation storm %d %d %v", operations, jobs, err)
	}
}

func TestAutomaticTagBusyDeferralRespectsCancellationAndDisable(t *testing.T) {
	for _, deferredFirst := range []bool{false, true} {
		for _, disable := range []bool{false, true} {
			s, id, j, now := setupAutomaticJob(t)
			ctx := context.Background()
			if deferredFirst {
				if err := s.DeferAutomaticTagJob(ctx, j, now, autotag.BusyRetryDelay); err != nil {
					t.Fatal(err)
				}
			}
			if disable {
				if err := s.UpdateAutomaticTagSettings(ctx, id, false, 2, now); err != nil {
					t.Fatal(err)
				}
			} else {
				op, err := s.GetAIOperation(ctx, j.OperationID)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := s.CancelAutomaticTagOperation(ctx, op.ID, op.Revision, now); err != nil {
					t.Fatal(err)
				}
			}
			if err := s.DeferAutomaticTagJob(ctx, j, now, autotag.BusyRetryDelay); !errors.Is(err, autotag.ErrConflict) {
				t.Fatalf("cancelled/disabled defer=%v", err)
			}
			if _, found, err := s.ClaimAutomaticTagJob(ctx, now.Add(time.Minute), time.Minute); err != nil || found {
				t.Fatalf("cancelled/disabled claim found=%v err=%v", found, err)
			}
			if deferredFirst {
				if op, err := s.GetAIOperation(ctx, j.OperationID); err != nil || op.State != aimodel.OperationCancelled {
					t.Fatalf("queued cancel=%+v %v", op, err)
				}
			}
		}
	}
}

func TestAutomaticTagBusyDeferralPreservesPriorCrashAttempts(t *testing.T) {
	s, _, j, now := setupAutomaticJob(t)
	ctx := context.Background()
	now = now.Add(2 * time.Minute)
	if _, err := s.RecoverAutomaticTagJobs(ctx, now); err != nil {
		t.Fatal(err)
	}
	j, found, err := s.ClaimAutomaticTagJob(ctx, now, time.Minute)
	if err != nil || !found || j.AttemptCount != 2 {
		t.Fatalf("crash claim=%+v %v", j, err)
	}
	if err := s.DeferAutomaticTagJob(ctx, j, now, autotag.BusyRetryDelay); err != nil {
		t.Fatal(err)
	}
	now = now.Add(autotag.BusyRetryDelay)
	j, found, err = s.ClaimAutomaticTagJob(ctx, now, time.Minute)
	if err != nil || !found || j.AttemptCount != 2 {
		t.Fatalf("busy erased previous crash=%+v %v", j, err)
	}
	for _, mutate := range []func(*autotag.Job){
		func(j *autotag.Job) { j.CheckpointID++ }, func(j *autotag.Job) { j.ClaimRevision-- },
		func(j *autotag.Job) { j.ModelIdentity = "different" }, func(j *autotag.Job) { j.LibraryID++ },
	} {
		wrong := j
		mutate(&wrong)
		if err := s.DeferAutomaticTagJob(ctx, wrong, now, autotag.BusyRetryDelay); !errors.Is(err, autotag.ErrConflict) {
			t.Fatalf("unfenced defer=%v", err)
		}
	}
	if err := s.DeferAutomaticTagJob(ctx, j, now.Add(2*time.Minute), autotag.BusyRetryDelay); !errors.Is(err, autotag.ErrConflict) {
		t.Fatalf("expired claim deferred=%v", err)
	}
}

func TestAutomaticTagBusyDeferralSurvivesDatabaseReopen(t *testing.T) {
	s, filename := openTestStore(t)
	ctx := context.Background()
	id := seedBrowseCatalog(t, s)
	now := time.Now()
	if err := s.UpdateAutomaticTagSettings(ctx, id, true, 1, now); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.AdmitAutomaticTagJob(ctx, automaticAdmission(id, "c", autotag.Missing), now); err != nil {
		t.Fatal(err)
	}
	j, found, err := s.ClaimAutomaticTagJob(ctx, now, time.Minute)
	if err != nil || !found {
		t.Fatalf("claim=%+v %v", j, err)
	}
	if err := s.DeferAutomaticTagJob(ctx, j, now, autotag.BusyRetryDelay); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(ctx, filename, Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	if summary, err := reopened.RecoverAutomaticTagJobs(ctx, now.Add(time.Second)); err != nil || summary.Requeued != 0 || summary.Interrupted != 0 {
		t.Fatalf("busy is not crashed work: %+v %v", summary, err)
	}
	if _, found, err := reopened.ClaimAutomaticTagJob(ctx, now.Add(time.Second), time.Minute); err != nil || found {
		t.Fatalf("restart skipped persistent delay: found=%v err=%v", found, err)
	}
	next, found, err := reopened.ClaimAutomaticTagJob(ctx, now.Add(autotag.BusyRetryDelay), time.Minute)
	if err != nil || !found || next.ID != j.ID || next.OperationID != j.OperationID || next.AttemptCount != 1 {
		t.Fatalf("resumed=%+v %v", next, err)
	}
}
