package autotag

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
	"github.com/HappyQuQu/foliopath/internal/curation"
	"github.com/HappyQuQu/foliopath/internal/jobs"
)

type busyAnalyzer struct {
	calls, busy int
	verifyErr   error
	analyzeErr  error
	onVerify    func()
}

func (a *busyAnalyzer) Model() *Model { return &Model{Identity: "test-model"} }
func (a *busyAnalyzer) Verify(context.Context) error {
	if a.onVerify != nil {
		a.onVerify()
	}
	return a.verifyErr
}
func (a *busyAnalyzer) Analyze(context.Context, Candidate) ([]curation.AutomaticLabel, error) {
	a.calls++
	if a.analyzeErr != nil {
		return nil, a.analyzeErr
	}
	if a.calls <= a.busy {
		return nil, ErrBusy
	}
	return []curation.AutomaticLabel{{Name: "tree", Confidence: .8}}, nil
}

func TestProcessorDoesNotRetryModelFailuresAsBusy(t *testing.T) {
	r, a := &busyRepository{}, &busyAnalyzer{analyzeErr: ErrUnavailable}
	if err := (Processor{Repository: r, Analyzer: a}).Process(context.Background(), busyJob()); !errors.Is(err, ErrUnavailable) {
		t.Fatal(err)
	}
	if a.calls != 1 || r.commits != 0 || r.defers != 0 || len(r.terminal) != 1 || r.terminal[0] != "failed:model_unavailable" {
		t.Fatalf("model failure incorrectly retried: calls=%d repository=%+v", a.calls, r)
	}
}

type busyRepository struct {
	Repository
	commits, defers, admits int
	deferred                Job
	delay                   time.Duration
	deferErr                error
	terminal                []string
}

func (r *busyRepository) NextAutomaticTagCandidate(_ context.Context, j Job) (Candidate, bool, error) {
	return Candidate{ID: 4, LibraryID: j.LibraryID, SourceFingerprint: "v1:10:20"}, r.commits == 0, nil
}
func (r *busyRepository) CommitAutomaticTagResult(_ context.Context, c Commit, _ time.Time) error {
	if c.Outcome != "ready" || c.Plan == nil || len(c.Plan.Labels) != 1 {
		panic("busy committed a failed/empty result")
	}
	r.commits++
	return nil
}
func (r *busyRepository) DeferAutomaticTagJob(_ context.Context, j Job, _ time.Time, delay time.Duration) error {
	r.defers++
	r.deferred, r.delay = j, delay
	return r.deferErr
}
func (r *busyRepository) FinishAutomaticTagJob(_ context.Context, _ Job, state, code string, _ time.Time) error {
	r.terminal = append(r.terminal, state+":"+code)
	return nil
}
func (r *busyRepository) FindAutomaticTagRequest(context.Context, string, string) (aimodel.Operation, bool, error) {
	return aimodel.Operation{}, false, nil
}
func (r *busyRepository) GetAutomaticTagSettings(context.Context, int64, string) (Settings, error) {
	return Settings{LibraryID: 1, Enabled: true}, nil
}
func (r *busyRepository) AdmitAutomaticTagJob(context.Context, Admission, time.Time) (aimodel.Operation, bool, error) {
	r.admits++
	return aimodel.Operation{ID: "same-operation"}, false, nil
}

func busyJob() Job {
	return Job{ID: "same-job", OperationID: "same-operation", LibraryID: 1, ModelIdentity: "test-model", CheckpointID: 3, UpperAssetID: 10, ClaimRevision: 1}
}

func TestProcessorRetriesBusyThenCommitsOnce(t *testing.T) {
	t.Parallel()
	r, a := &busyRepository{}, &busyAnalyzer{busy: 2}
	if err := (Processor{Repository: r, Analyzer: a}).Process(context.Background(), busyJob()); err != nil {
		t.Fatal(err)
	}
	if a.calls != 3 || r.commits != 1 || r.defers != 0 || len(r.terminal) != 1 || r.terminal[0] != "succeeded:" {
		t.Fatalf("calls=%d repository=%+v", a.calls, r)
	}
}

func TestProcessorDefersBusyWithoutTerminalFailureOrProgress(t *testing.T) {
	t.Parallel()
	r, a := &busyRepository{}, &busyAnalyzer{busy: 100}
	start := time.Now()
	if err := (Processor{Repository: r, Analyzer: a}).Process(context.Background(), busyJob()); !errors.Is(err, ErrBusy) {
		t.Fatal(err)
	}
	if elapsed := time.Since(start); elapsed < 3*time.Second || elapsed > 6*time.Second {
		t.Fatalf("unbounded or tight retry: %v", elapsed)
	}
	if a.calls != 3 || r.commits != 0 || r.defers != 1 || len(r.terminal) != 0 || r.deferred != busyJob() || r.delay != 30*time.Second {
		t.Fatalf("calls=%d repository=%+v", a.calls, r)
	}
}

func TestProcessorBusyWaitIsCancellable(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r, a := &busyRepository{}, &busyAnalyzer{busy: 100}
	timer := time.AfterFunc(50*time.Millisecond, cancel)
	defer timer.Stop()
	start := time.Now()
	if err := (Processor{Repository: r, Analyzer: a}).Process(ctx, busyJob()); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if time.Since(start) > time.Second || a.calls != 1 || r.commits != 0 || r.defers != 0 || len(r.terminal) != 1 || r.terminal[0] != "cancelled:cancelled" {
		t.Fatalf("cancellation did not stop busy turn: %+v calls=%d", r, a.calls)
	}
}

func TestProcessorDeferralConflictFinalizesCancellationWithoutFailure(t *testing.T) {
	t.Parallel()
	r, a := &busyRepository{deferErr: ErrConflict}, &busyAnalyzer{busy: 100}
	if err := (Processor{Repository: r, Analyzer: a}).Process(context.Background(), busyJob()); !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}
	if r.commits != 0 || r.defers != 1 || len(r.terminal) != 1 || r.terminal[0] != "cancelled:cancelled" {
		t.Fatalf("repository=%+v", r)
	}
}

func TestBusyProbePreservesReadinessButNeverInventsIt(t *testing.T) {
	for _, wasReady := range []bool{false, true} {
		r, a := &busyRepository{}, &busyAnalyzer{verifyErr: ErrBusy}
		s, _ := New(r, a, jobs.NewSignal())
		s.ready.Store(wasReady)
		_, _, err := s.Request(context.Background(), 1, Missing, "request-key")
		if wasReady && (err != nil || r.admits != 1) || !wasReady && (!errors.Is(err, ErrUnavailable) || r.admits != 0) {
			t.Fatalf("ready=%v error=%v admitted=%d", wasReady, err, r.admits)
		}
		if s.ready.Load() != wasReady {
			t.Fatal("busy changed readiness")
		}
		ctx, cancel := context.WithCancel(context.Background())
		a.onVerify = cancel
		if err := s.Run(ctx); err != nil || s.ready.Load() != wasReady {
			t.Fatalf("reconciler busy changed readiness: %v", err)
		}
	}
}
