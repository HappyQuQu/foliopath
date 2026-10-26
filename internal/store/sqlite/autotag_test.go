package sqlite

import (
	"context"
	"errors"
	"fmt"
	"github.com/HappyQuQu/foliopath/internal/aimodel"
	"github.com/HappyQuQu/foliopath/internal/autotag"
	"github.com/HappyQuQu/foliopath/internal/curation"
	"github.com/HappyQuQu/foliopath/internal/jobs"
	"github.com/HappyQuQu/foliopath/internal/library"
	"strings"
	"testing"
	"time"
)

func automaticAdmission(library int64, suffix string, mode autotag.Mode) autotag.Admission {
	return autotag.Admission{Job: autotag.Job{ID: "autotag_job_" + suffix, OperationID: "autotag_op_" + suffix, LibraryID: library, ModelIdentity: strings.Repeat("a", 64), Mode: mode}, KeyHash: strings.Repeat(suffix, 64)[:64], RequestHash: strings.Repeat("b", 64)}
}
func setupAutomaticJob(t *testing.T) (*Store, int64, autotag.Job, time.Time) {
	t.Helper()
	s, _ := openTestStore(t)
	id := seedBrowseCatalog(t, s)
	now := time.Now()
	ctx := context.Background()
	if err := s.UpdateAutomaticTagSettings(ctx, id, true, 1, now); err != nil {
		t.Fatal(err)
	}
	op, _, err := s.AdmitAutomaticTagJob(ctx, automaticAdmission(id, "c", autotag.Missing), now)
	if err != nil {
		t.Fatal(err)
	}
	if op.TotalItems != nil {
		t.Fatal("running total must be nullable")
	}
	j, found, err := s.ClaimAutomaticTagJob(ctx, now, time.Minute)
	if err != nil || !found {
		t.Fatalf("claim: %+v %v", j, err)
	}
	return s, id, j, now
}
func automaticCommit(t *testing.T, s *Store, j autotag.Job) autotag.Commit {
	t.Helper()
	asset, found, err := s.NextAutomaticTagCandidate(context.Background(), j)
	if err != nil || !found {
		t.Fatalf("candidate %+v %v", asset, err)
	}
	plan := curation.AutomaticTagPlan{LibraryID: j.LibraryID, AssetID: asset.ID, SourceFingerprint: asset.SourceFingerprint, ModelIdentity: j.ModelIdentity, Labels: []curation.AutomaticLabel{{Name: "tree", Confidence: .9}}}
	return autotag.Commit{Job: j, Asset: asset, Plan: &plan, Outcome: "ready"}
}
func TestAutomaticTagsCommitAtomicProgressManualProtectionAndMissing(t *testing.T) {
	s, id, j, now := setupAutomaticJob(t)
	ctx := context.Background()
	commit := automaticCommit(t, s, j)
	manual := newCurationServiceForStore(t, s, now)
	tag, _, err := manual.CreateTag(ctx, "manual")
	if err != nil {
		t.Fatal(err)
	}
	before, err := manual.GetAssetState(ctx, commit.Asset.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = manual.ReplaceAssetTags(ctx, commit.Asset.ID, before.Revision, []int64{tag.ID}); err != nil {
		t.Fatal(err)
	}
	if err = s.CommitAutomaticTagResult(ctx, commit, now); err != nil {
		t.Fatal(err)
	}
	state, err := manual.GetAssetState(ctx, commit.Asset.ID)
	if err != nil || len(state.Tags) != 2 {
		t.Fatalf("manual retained: %+v %v", state, err)
	}
	op, err := s.GetAIOperation(ctx, j.OperationID)
	if err != nil || op.CompletedItems != 1 || op.TotalItems != nil {
		t.Fatalf("progress %+v %v", op, err)
	}
	if err = s.CommitAutomaticTagResult(ctx, commit, now); !errors.Is(err, autotag.ErrConflict) {
		t.Fatalf("replay commit %v", err)
	}
	j.CheckpointID = commit.Asset.ID
	next := automaticCommit(t, s, j)
	next.Plan = nil
	next.Outcome = "failed"
	if err = s.CommitAutomaticTagResult(ctx, next, now); err != nil {
		t.Fatal(err)
	}
	j.CheckpointID = next.Asset.ID
	last := automaticCommit(t, s, j)
	if err = s.CommitAutomaticTagResult(ctx, last, now); err != nil {
		t.Fatal(err)
	}
	if err = s.FinishAutomaticTagJob(ctx, j, "succeeded", "", now); err != nil {
		t.Fatal(err)
	}
	op, _ = s.GetAIOperation(ctx, j.OperationID)
	if op.TotalItems == nil || *op.TotalItems != 3 || op.CompletedItems != 3 || op.State != aimodel.OperationSucceeded {
		t.Fatalf("terminal %+v", op)
	}
	settings, err := s.GetAutomaticTagSettings(ctx, id, j.ModelIdentity)
	if err != nil || settings.Coverage.Eligible != 3 || settings.Coverage.Completed != 2 || settings.Coverage.Failed != 1 {
		t.Fatalf("coverage %+v %v", settings, err)
	}
	automatic := automaticAdmission(id, "d", autotag.Missing)
	automatic.Automatic = true
	noWork, _, err := s.AdmitAutomaticTagJob(ctx, automatic, now)
	if err != nil || noWork.ID != "" {
		t.Fatalf("failed same identity retried: %+v %v", noWork, err)
	}
	all := automaticAdmission(id, "e", autotag.All)
	if _, _, err = s.AdmitAutomaticTagJob(ctx, all, now); err != nil {
		t.Fatal(err)
	}
	allJob, _, _ := s.ClaimAutomaticTagJob(ctx, now, time.Minute)
	asset, found, err := s.NextAutomaticTagCandidate(ctx, allJob)
	if err != nil || !found || asset.ID != commit.Asset.ID {
		t.Fatalf("all retry: %+v %v", asset, err)
	}
}
func TestAutomaticTagsDisableAndCancelFenceInFlightWrites(t *testing.T) {
	for _, disable := range []bool{false, true} {
		t.Run(map[bool]string{false: "cancel", true: "disable"}[disable], func(t *testing.T) {
			s, id, j, now := setupAutomaticJob(t)
			ctx := context.Background()
			commit := automaticCommit(t, s, j)
			if disable {
				if err := s.UpdateAutomaticTagSettings(ctx, id, false, 2, now); err != nil {
					t.Fatal(err)
				}
			} else {
				op, _ := s.GetAIOperation(ctx, j.OperationID)
				if _, err := s.CancelAutomaticTagOperation(ctx, op.ID, op.Revision, now); err != nil {
					t.Fatal(err)
				}
			}
			if err := s.CommitAutomaticTagResult(ctx, commit, now); !errors.Is(err, autotag.ErrConflict) {
				t.Fatalf("late commit=%v", err)
			}
			var count int
			if err := s.db.QueryRow(`SELECT COUNT(*) FROM asset_tags WHERE source='ai'`).Scan(&count); err != nil || count != 0 {
				t.Fatalf("late tags=%d %v", count, err)
			}
			if err := s.FinishAutomaticTagJob(ctx, j, "cancelled", "cancelled", now); err != nil {
				t.Fatal(err)
			}
			op, _ := s.GetAIOperation(ctx, j.OperationID)
			if op.State != aimodel.OperationCancelled || op.TotalItems == nil {
				t.Fatalf("cancelled %+v", op)
			}
		})
	}
}
func TestAutomaticTagsRecoveryFencesOldClaimAndStopsAfterThreeAttempts(t *testing.T) {
	s, _, j, now := setupAutomaticJob(t)
	ctx := context.Background()
	commit := automaticCommit(t, s, j)
	// A shutdown context is not a user cancellation.
	if err := s.FinishAutomaticTagJob(ctx, j, "cancelled", "cancelled", now); err != nil {
		t.Fatal(err)
	}
	op, _ := s.GetAIOperation(ctx, j.OperationID)
	if op.State != aimodel.OperationRunning {
		t.Fatal(op.State)
	}
	for attempt := 1; attempt <= 3; attempt++ {
		now = now.Add(2 * time.Minute)
		summary, err := s.RecoverAutomaticTagJobs(ctx, now)
		if err != nil {
			t.Fatal(err)
		}
		if attempt == 3 {
			if summary.Interrupted != 1 {
				t.Fatalf("summary %+v", summary)
			}
			break
		}
		if summary.Requeued != 1 {
			t.Fatalf("summary %+v", summary)
		}
		if err = s.CommitAutomaticTagResult(ctx, commit, now); !errors.Is(err, autotag.ErrConflict) {
			t.Fatalf("old commit %v", err)
		}
		newer, found, err := s.ClaimAutomaticTagJob(ctx, now, time.Minute)
		if err != nil || !found || newer.ClaimRevision <= j.ClaimRevision {
			t.Fatalf("new claim %+v %v", newer, err)
		}
	}
	op, _ = s.GetAIOperation(ctx, j.OperationID)
	if op.State != aimodel.OperationFailed || op.ErrorCode != "operation_interrupted" {
		t.Fatalf("final %+v", op)
	}
}
func TestAutomaticTagsSourceChangedCommitsNoTagsAndKeepsNewSourceEligible(t *testing.T) {
	s, _, j, now := setupAutomaticJob(t)
	ctx := context.Background()
	c := automaticCommit(t, s, j)
	if _, err := s.db.Exec(`UPDATE assets SET source_fingerprint='v1:99:99',size_bytes=99,mtime_ns=99 WHERE id=?`, c.Asset.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.CommitAutomaticTagResult(ctx, c, now); err != nil {
		t.Fatal(err)
	}
	var count int
	s.db.QueryRow(`SELECT COUNT(*) FROM asset_tags WHERE source='ai'`).Scan(&count)
	if count != 0 {
		t.Fatal("stale image received tags")
	}
	candidate, found, err := s.NextAutomaticTagCandidate(ctx, j)
	if err != nil || !found || candidate.ID != c.Asset.ID || candidate.SourceFingerprint == c.Asset.SourceFingerprint {
		t.Fatalf("new source eligible %+v %v", candidate, err)
	}
}

type automaticTestAnalyzer struct {
	model         autotag.Model
	verifyErr     error
	analyzeErr    error
	invalidOutput bool
}

func (a *automaticTestAnalyzer) Model() *autotag.Model        { return &a.model }
func (a *automaticTestAnalyzer) Verify(context.Context) error { return a.verifyErr }
func (a *automaticTestAnalyzer) Analyze(context.Context, autotag.Candidate) ([]curation.AutomaticLabel, error) {
	if a.analyzeErr != nil {
		return nil, a.analyzeErr
	}
	if a.invalidOutput {
		return []curation.AutomaticLabel{{Name: "", Confidence: .9}}, nil
	}
	return []curation.AutomaticLabel{{Name: "tree", Confidence: .9}}, nil
}
func TestAutomaticTagsRealProcessorAndIdempotentReplayWhileModelOffline(t *testing.T) {
	s, id, j, _ := setupAutomaticJob(t)
	ctx := context.Background()
	analyzer := &automaticTestAnalyzer{model: autotag.Model{Identity: j.ModelIdentity, DisplayName: "test", Protocol: "foliopath-tag-v1"}}
	if err := (autotag.Processor{Repository: s, Analyzer: analyzer}).Process(ctx, j); err != nil {
		t.Fatal(err)
	}
	op, _ := s.GetAIOperation(ctx, j.OperationID)
	if op.CompletedItems != 3 || op.State != aimodel.OperationSucceeded {
		t.Fatalf("process %+v", op)
	}
	service, _ := autotag.New(s, analyzer, jobs.NewSignal())
	created, _, err := service.Request(ctx, id, autotag.All, "replay-test-123")
	if err != nil {
		t.Fatal(err)
	}
	analyzer.verifyErr = autotag.ErrUnavailable
	replay, replayed, err := service.Request(ctx, id, autotag.All, "replay-test-123")
	if err != nil || !replayed || replay.ID != created.ID {
		t.Fatalf("offline replay %+v %v %v", replay, replayed, err)
	}
	if _, _, err = service.Request(ctx, id, autotag.Missing, "replay-test-123"); !errors.Is(err, aimodel.ErrIdempotencyConflict) {
		t.Fatalf("idempotency conflict %v", err)
	}
	if _, _, err = service.Request(ctx, 999999, autotag.All, "unknown-lib-key"); !errors.Is(err, autotag.ErrNotFound) {
		t.Fatalf("unknown lib=%v", err)
	}
}

func TestAutomaticTagsAdmissionBoundedQueueAndLibraryKeyset(t *testing.T) {
	s, _ := openTestStore(t)
	ctx := context.Background()
	now := time.Now()
	libraries, _ := library.NewService(s)
	var ids []int64
	for i := 0; i < 52; i++ {
		lib, err := libraries.Create(ctx, fmt.Sprintf("Library %d", i), fmt.Sprintf("root-%d", i))
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, lib.ID)
		if err = s.UpdateAutomaticTagSettings(ctx, lib.ID, true, 1, now); err != nil {
			t.Fatal(err)
		}
		if i <= 32 {
			a := automaticAdmission(lib.ID, "f", autotag.Missing)
			a.Job.ID = fmt.Sprintf("autotag_job_%d", i)
			a.Job.OperationID = fmt.Sprintf("autotag_op_%d", i)
			a.KeyHash = fmt.Sprintf("%064d", i)
			_, _, err = s.AdmitAutomaticTagJob(ctx, a, now)
			if i < 32 && err != nil {
				t.Fatal(err)
			}
			if i == 32 && !errors.Is(err, autotag.ErrConflict) {
				t.Fatalf("queue overflow=%v", err)
			}
		}
	}
	first, err := s.ListAutomaticTagLibraries(ctx, 0, 50)
	if err != nil || len(first) != 50 {
		t.Fatalf("first len%d %v", len(first), err)
	}
	second, err := s.ListAutomaticTagLibraries(ctx, first[49], 50)
	if err != nil || len(second) != 2 || second[0] != ids[50] {
		t.Fatalf("second %+v %v", second, err)
	}
	if _, err = s.ListAutomaticTagLibraries(ctx, 0, 51); !errors.Is(err, autotag.ErrInvalid) {
		t.Fatalf("oversized page=%v", err)
	}
	job, found, err := s.ClaimAutomaticTagJob(ctx, now, time.Minute)
	if err != nil || !found {
		t.Fatal(err)
	}
	// A queued cancellation frees capacity without touching another library.
	var operationID string
	if err = s.db.QueryRow(`SELECT operation_id FROM autotag_jobs WHERE library_id=?`, ids[1]).Scan(&operationID); err != nil {
		t.Fatal(err)
	}
	op, err := s.GetAIOperation(ctx, operationID)
	if err != nil {
		t.Fatal(err)
	}
	cancelled, err := s.CancelAutomaticTagOperation(ctx, op.ID, op.Revision, now)
	if err != nil || cancelled.State != aimodel.OperationCancelled {
		t.Fatalf("queued cancel %+v %v", cancelled, err)
	}
	active, err := s.GetAIOperation(ctx, job.OperationID)
	if err != nil || active.State != aimodel.OperationRunning {
		t.Fatalf("other library affected %+v %v", active, err)
	}
}

func TestAutomaticTagsSettingsConflictsAndOfflineAdmission(t *testing.T) {
	s, id, j, now := setupAutomaticJob(t)
	ctx := context.Background()
	if err := s.UpdateAutomaticTagSettings(ctx, id, false, 1, now); !errors.Is(err, aimodel.ErrPreconditionFailed) {
		t.Fatalf("lost update %v", err)
	}
	if _, err := s.GetAutomaticTagSettings(ctx, 999999, j.ModelIdentity); !errors.Is(err, autotag.ErrNotFound) {
		t.Fatal(err)
	}
	op, _ := s.GetAIOperation(ctx, j.OperationID)
	if _, err := s.CancelAutomaticTagOperation(ctx, op.ID, op.Revision+1, now); !errors.Is(err, aimodel.ErrPreconditionFailed) {
		t.Fatalf("stale cancel %v", err)
	}
	if _, err := s.db.Exec(`UPDATE libraries SET status='offline' WHERE id=?`, id); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.AdmitAutomaticTagJob(ctx, automaticAdmission(id, "f", autotag.All), now); !errors.Is(err, autotag.ErrOffline) {
		t.Fatalf("offline admission %v", err)
	}
	value, err := s.GetAutomaticTagSettings(ctx, id, j.ModelIdentity)
	if err != nil || !value.Offline || value.Coverage.Eligible != 3 {
		t.Fatalf("offline catalog preserved %+v %v", value, err)
	}
}

func TestAutomaticTagsTerminalFailuresBlockAutomaticReadmission(t *testing.T) {
	for _, reason := range []string{"service_error", "invalid_output", "recovery_exhausted", "cancelled"} {
		t.Run(reason, func(t *testing.T) {
			s, id, j, now := setupAutomaticJob(t)
			ctx := context.Background()
			switch reason {
			case "recovery_exhausted":
				for n := 0; n < 3; n++ {
					now = now.Add(2 * time.Minute)
					if _, err := s.RecoverAutomaticTagJobs(ctx, now); err != nil {
						t.Fatal(err)
					}
					if n < 2 {
						if _, _, err := s.ClaimAutomaticTagJob(ctx, now, time.Minute); err != nil {
							t.Fatal(err)
						}
					}
				}
			case "cancelled":
				op, _ := s.GetAIOperation(ctx, j.OperationID)
				if _, err := s.CancelAutomaticTagOperation(ctx, op.ID, op.Revision, now); err != nil {
					t.Fatal(err)
				}
				if err := s.FinishAutomaticTagJob(ctx, j, "cancelled", "cancelled", now); err != nil {
					t.Fatal(err)
				}
			default:
				analyzer := &automaticTestAnalyzer{model: autotag.Model{Identity: j.ModelIdentity}, invalidOutput: reason == "invalid_output"}
				if reason == "service_error" {
					analyzer.analyzeErr = autotag.ErrUnavailable
				}
				if err := (autotag.Processor{Repository: s, Analyzer: analyzer}).Process(ctx, j); err == nil {
					t.Fatal("expected processor failure")
				}
			}
			value, err := s.GetAutomaticTagSettings(ctx, id, j.ModelIdentity)
			if err != nil || !value.AutomaticBlocked {
				t.Fatalf("failure fence %+v %v", value, err)
			}
			a := automaticAdmission(id, "d", autotag.Missing)
			a.Automatic = true
			for n := 0; n < 3; n++ {
				op, _, err := s.AdmitAutomaticTagJob(ctx, a, now)
				if err != nil || op.ID != "" {
					t.Fatalf("auto retried terminal failure %+v %v", op, err)
				}
			}
			all := automaticAdmission(id, "e", autotag.All)
			if _, _, err = s.AdmitAutomaticTagJob(ctx, all, now); err != nil {
				t.Fatal(err)
			}
			value, err = s.GetAutomaticTagSettings(ctx, id, j.ModelIdentity)
			if err != nil || value.AutomaticBlocked {
				t.Fatalf("explicit all must clear block %+v %v", value, err)
			}
		})
	}
}

func TestAutomaticTagsExplicitEnableClearsCancelFenceEvenBeforeWorkerStops(t *testing.T) {
	s, id, j, now := setupAutomaticJob(t)
	ctx := context.Background()
	op, _ := s.GetAIOperation(ctx, j.OperationID)
	if _, err := s.CancelAutomaticTagOperation(ctx, op.ID, op.Revision, now); err != nil {
		t.Fatal(err)
	}
	if err := s.UpdateAutomaticTagSettings(ctx, id, true, 2, now); err != nil {
		t.Fatal(err)
	}
	if err := s.FinishAutomaticTagJob(ctx, j, "cancelled", "cancelled", now); err != nil {
		t.Fatal(err)
	}
	a := automaticAdmission(id, "d", autotag.Missing)
	a.Automatic = true
	next, _, err := s.AdmitAutomaticTagJob(ctx, a, now)
	if err != nil || next.ID == "" {
		t.Fatalf("explicit enable lost to late cancellation %+v %v", next, err)
	}
}

func TestAutomaticTagsRejectModelMismatchWithoutChangingTagsOrCheckpoint(t *testing.T) {
	for _, mismatch := range []string{"plan", "claimed_job"} {
		t.Run(mismatch, func(t *testing.T) {
			s, _, j, now := setupAutomaticJob(t)
			ctx := context.Background()
			commit := automaticCommit(t, s, j)
			commit.Plan.ModelIdentity = strings.Repeat("d", 64)
			want := autotag.ErrInvalid
			if mismatch == "claimed_job" {
				commit.Job.ModelIdentity = commit.Plan.ModelIdentity
				want = autotag.ErrConflict
			}
			if err := s.CommitAutomaticTagResult(ctx, commit, now); !errors.Is(err, want) {
				t.Fatalf("model mismatch=%v want %v", err, want)
			}
			var tags, runs, checkpoint int64
			if err := s.db.QueryRow(`SELECT COUNT(*) FROM asset_tags WHERE library_id=?`, j.LibraryID).Scan(&tags); err != nil {
				t.Fatal(err)
			}
			if err := s.db.QueryRow(`SELECT COUNT(*) FROM autotag_asset_runs WHERE library_id=?`, j.LibraryID).Scan(&runs); err != nil {
				t.Fatal(err)
			}
			if err := s.db.QueryRow(`SELECT checkpoint_id FROM autotag_jobs WHERE id=?`, j.ID).Scan(&checkpoint); err != nil {
				t.Fatal(err)
			}
			op, err := s.GetAIOperation(ctx, j.OperationID)
			if err != nil {
				t.Fatal(err)
			}
			if tags != 0 || runs != 0 || checkpoint != 0 || op.CompletedItems != 0 || op.State != aimodel.OperationRunning {
				t.Fatalf("rejected result mutated state: tags=%d runs=%d checkpoint=%d operation=%+v", tags, runs, checkpoint, op)
			}
		})
	}
}

func TestAutomaticTagsModelUpgradeMarksOldCoverageStaleAndRequeuesMissing(t *testing.T) {
	s, id, old, now := setupAutomaticJob(t)
	ctx := context.Background()
	analyzer := &automaticTestAnalyzer{model: autotag.Model{Identity: old.ModelIdentity}}
	if err := (autotag.Processor{Repository: s, Analyzer: analyzer}).Process(ctx, old); err != nil {
		t.Fatal(err)
	}
	before, err := s.GetAutomaticTagSettings(ctx, id, old.ModelIdentity)
	if err != nil || before.Coverage.Completed != 3 || before.Coverage.Stale != 0 {
		t.Fatalf("old coverage %+v %v", before.Coverage, err)
	}
	newIdentity := strings.Repeat("d", 64)
	changed, err := s.GetAutomaticTagSettings(ctx, id, newIdentity)
	if err != nil || changed.Coverage.Eligible != 3 || changed.Coverage.Completed != 0 || changed.Coverage.Failed != 0 || changed.Coverage.Stale != 3 {
		t.Fatalf("new identity coverage %+v %v", changed.Coverage, err)
	}
	admission := automaticAdmission(id, "d", autotag.Missing)
	admission.Automatic = true
	admission.Job.ModelIdentity = newIdentity
	op, _, err := s.AdmitAutomaticTagJob(ctx, admission, now)
	if err != nil || op.ID == "" {
		t.Fatalf("new identity missing admission %+v %v", op, err)
	}
	next, found, err := s.ClaimAutomaticTagJob(ctx, now, time.Minute)
	if err != nil || !found || next.ModelIdentity != newIdentity {
		t.Fatalf("new identity claim %+v %v", next, err)
	}
	first := automaticCommit(t, s, next)
	if err = s.CommitAutomaticTagResult(ctx, first, now); err != nil {
		t.Fatal(err)
	}
	progress, err := s.GetAutomaticTagSettings(ctx, id, newIdentity)
	if err != nil || progress.Coverage.Completed != 1 || progress.Coverage.Stale != 2 {
		t.Fatalf("mixed generation progress %+v %v", progress.Coverage, err)
	}
	next.CheckpointID = first.Asset.ID
	analyzer.model.Identity = newIdentity
	if err = (autotag.Processor{Repository: s, Analyzer: analyzer}).Process(ctx, next); err != nil {
		t.Fatal(err)
	}
	complete, err := s.GetAutomaticTagSettings(ctx, id, newIdentity)
	if err != nil || complete.Coverage.Completed != 3 || complete.Coverage.Stale != 0 {
		t.Fatalf("new coverage %+v %v", complete.Coverage, err)
	}
	var oldResults int
	if err = s.db.QueryRow(`SELECT COUNT(*) FROM autotag_asset_runs WHERE library_id=? AND model_identity<>?`, id, newIdentity).Scan(&oldResults); err != nil || oldResults != 0 {
		t.Fatalf("old run identities remain=%d %v", oldResults, err)
	}
}

func TestAutomaticTagsOldTaskCannotWriteAfterDeploymentModelChanges(t *testing.T) {
	s, id, old, now := setupAutomaticJob(t)
	ctx := context.Background()
	late := automaticCommit(t, s, old)
	newIdentity := strings.Repeat("d", 64)
	admission := automaticAdmission(id, "d", autotag.Missing)
	admission.Automatic = true
	admission.Job.ModelIdentity = newIdentity
	// The single-active-library invariant must not coalesce a new-model request
	// into a still-running old-model job.
	if _, _, err := s.AdmitAutomaticTagJob(ctx, admission, now); !errors.Is(err, autotag.ErrConflict) {
		t.Fatalf("coalesced incompatible models: %v", err)
	}
	// A restarted worker rejects an old persisted job before inference. Its
	// old-model failure fence must not permanently block the new deployment.
	newAnalyzer := &automaticTestAnalyzer{model: autotag.Model{Identity: newIdentity}}
	if err := (autotag.Processor{Repository: s, Analyzer: newAnalyzer}).Process(ctx, old); !errors.Is(err, autotag.ErrUnavailable) {
		t.Fatalf("old model processor=%v", err)
	}
	oldSettings, err := s.GetAutomaticTagSettings(ctx, id, old.ModelIdentity)
	if err != nil || !oldSettings.AutomaticBlocked {
		t.Fatalf("old model fence %+v %v", oldSettings, err)
	}
	newSettings, err := s.GetAutomaticTagSettings(ctx, id, newIdentity)
	if err != nil || newSettings.AutomaticBlocked {
		t.Fatalf("old model blocked new deployment %+v %v", newSettings, err)
	}
	if _, _, err = s.AdmitAutomaticTagJob(ctx, admission, now); err != nil {
		t.Fatal(err)
	}
	current, found, err := s.ClaimAutomaticTagJob(ctx, now, time.Minute)
	if err != nil || !found {
		t.Fatalf("new claim %+v %v", current, err)
	}
	result := automaticCommit(t, s, current)
	result.Plan.Labels = []curation.AutomaticLabel{{Name: "new model label", Confidence: .9}}
	if err = s.CommitAutomaticTagResult(ctx, result, now); err != nil {
		t.Fatal(err)
	}
	if err = s.CommitAutomaticTagResult(ctx, late, now); !errors.Is(err, autotag.ErrConflict) {
		t.Fatalf("late old result accepted: %v", err)
	}
	// Even retargeting the old output to a current claim cannot bypass the
	// plan/job identity check for another unprocessed image.
	current.CheckpointID = result.Asset.ID
	forged := automaticCommit(t, s, current)
	forged.Plan.ModelIdentity = old.ModelIdentity
	if err = s.CommitAutomaticTagResult(ctx, forged, now); !errors.Is(err, autotag.ErrInvalid) {
		t.Fatalf("old output on new claim accepted: %v", err)
	}
	var label, tagIdentity, runIdentity string
	if err = s.db.QueryRow(`SELECT tag.name,assignment.ai_model_identity FROM asset_tags assignment JOIN tags tag ON tag.id=assignment.tag_id WHERE assignment.library_id=? AND assignment.asset_id=? AND assignment.source='ai'`, id, result.Asset.ID).Scan(&label, &tagIdentity); err != nil {
		t.Fatal(err)
	}
	if err = s.db.QueryRow(`SELECT model_identity FROM autotag_asset_runs WHERE library_id=? AND asset_id=?`, id, result.Asset.ID).Scan(&runIdentity); err != nil {
		t.Fatal(err)
	}
	operation, err := s.GetAIOperation(ctx, current.OperationID)
	if err != nil {
		t.Fatal(err)
	}
	if label != "new model label" || tagIdentity != newIdentity || runIdentity != newIdentity || operation.CompletedItems != 1 {
		t.Fatalf("new result overwritten: label=%s tag=%s run=%s operation=%+v", label, tagIdentity, runIdentity, operation)
	}
}
