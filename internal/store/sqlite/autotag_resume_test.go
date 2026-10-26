package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
	"github.com/HappyQuQu/foliopath/internal/autotag"
)

func TestAutomaticTagExplicitReenableResumesOnlyUnprocessedAfterModelFailure(t *testing.T) {
	s, id, j, now := setupAutomaticJob(t)
	ctx := context.Background()
	ready := automaticCommit(t, s, j)
	if err := s.CommitAutomaticTagResult(ctx, ready, now); err != nil {
		t.Fatal(err)
	}
	j.CheckpointID = ready.Asset.ID
	bad := automaticCommit(t, s, j)
	bad.Plan = nil
	bad.Outcome = "failed"
	if err := s.CommitAutomaticTagResult(ctx, bad, now); err != nil {
		t.Fatal(err)
	}
	j.CheckpointID = bad.Asset.ID
	if err := s.FinishAutomaticTagJob(ctx, j, "failed", "model_unavailable", now); err != nil {
		t.Fatal(err)
	}
	before, err := s.GetAutomaticTagSettings(ctx, id, j.ModelIdentity)
	if err != nil || !before.AutomaticBlocked {
		t.Fatalf("settings=%+v %v", before, err)
	}
	if err = s.UpdateAutomaticTagSettings(ctx, id, true, before.Revision, now); err != nil {
		t.Fatal(err)
	}
	a := automaticAdmission(id, "d", autotag.Missing)
	a.Automatic = true
	op, _, err := s.AdmitAutomaticTagJob(ctx, a, now)
	if err != nil || op.ID == j.OperationID {
		t.Fatalf("new explicit missing=%+v %v", op, err)
	}
	next, found, err := s.ClaimAutomaticTagJob(ctx, now, time.Minute)
	if err != nil || !found {
		t.Fatalf("claim=%+v %v", next, err)
	}
	remaining := automaticCommit(t, s, next)
	if remaining.Asset.ID <= bad.Asset.ID {
		t.Fatal("already completed or known failed source retried")
	}
	if err = s.CommitAutomaticTagResult(ctx, remaining, now); err != nil {
		t.Fatal(err)
	}
	next.CheckpointID = remaining.Asset.ID
	if _, found, err = s.NextAutomaticTagCandidate(ctx, next); err != nil || found {
		t.Fatalf("extra work=%v %v", found, err)
	}
	if err = s.FinishAutomaticTagJob(ctx, next, "succeeded", "", now); err != nil {
		t.Fatal(err)
	}
	old, err := s.GetAIOperation(ctx, j.OperationID)
	if err != nil || old.State != aimodel.OperationFailed || old.ErrorCode != "model_unavailable" || old.CompletedItems != 2 {
		t.Fatalf("history changed=%+v %v", old, err)
	}
	settings, err := s.GetAutomaticTagSettings(ctx, id, j.ModelIdentity)
	if err != nil || settings.Coverage.Completed != 2 || settings.Coverage.Failed != 1 || settings.AutomaticBlocked {
		t.Fatalf("lost progress=%+v %v", settings, err)
	}
}
