package autotag

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/curation"
)

type recoveryAnalyzer struct {
	busyAnalyzer
	probes, outages int
	verifyFailure   error
	onProbe         func()
}

func (a *recoveryAnalyzer) Verify(ctx context.Context) error {
	a.probes++
	if a.onProbe != nil {
		a.onProbe()
	}
	if a.verifyFailure != nil {
		return a.verifyFailure
	}
	if a.probes <= a.outages {
		return ErrTemporarilyUnavailable
	}
	return ctx.Err()
}
func (a *recoveryAnalyzer) Analyze(context.Context, Candidate) ([]curation.AutomaticLabel, error) {
	a.calls++
	if a.calls == 1 || a.analyzeErr != nil {
		if a.analyzeErr != nil {
			return nil, a.analyzeErr
		}
		return nil, ErrTemporarilyUnavailable
	}
	return []curation.AutomaticLabel{{Name: "tree", Confidence: .8}}, nil
}

func TestAutomaticTagTransientRecoveryKeepsOneOperationAndCommitsOnce(t *testing.T) {
	r, a := &busyRepository{}, &recoveryAnalyzer{outages: 2}
	if err := (Processor{Repository: r, Analyzer: a}).Process(context.Background(), busyJob()); err != nil {
		t.Fatal(err)
	}
	if a.calls != 2 || a.probes != 3 || r.commits != 1 || r.defers != 0 || len(r.terminal) != 1 || r.terminal[0] != "succeeded:" {
		t.Fatalf("analyzer=%+v repository=%+v", a, r)
	}
}

func TestAutomaticTagTransientRecoveryIsBoundedAndCancellable(t *testing.T) {
	for _, cancelParent := range []bool{false, true} {
		ctx, cancel := context.WithCancel(context.Background())
		a := &recoveryAnalyzer{verifyFailure: ErrTemporarilyUnavailable}
		if cancelParent {
			a.onProbe = cancel
		}
		start := time.Now()
		labels, err := (Processor{Analyzer: a}).analyzeRecover(ctx, Candidate{}, 25*time.Millisecond, time.Millisecond)
		cancel()
		expected := ErrTemporarilyUnavailable
		if cancelParent {
			expected = context.Canceled
		}
		if !errors.Is(err, expected) || labels != nil || a.calls != 1 || time.Since(start) > time.Second {
			t.Fatalf("cancel=%v err=%v analyzer=%+v", cancelParent, err, a)
		}
	}
}

func TestAutomaticTagRecoveryLimitsInferenceRetriesAndRejectsHardFailures(t *testing.T) {
	a := &recoveryAnalyzer{busyAnalyzer: busyAnalyzer{analyzeErr: ErrTemporarilyUnavailable}}
	if _, err := (Processor{Analyzer: a}).analyzeRecover(context.Background(), Candidate{}, time.Second, time.Millisecond); !errors.Is(err, ErrTemporarilyUnavailable) || a.calls != 3 {
		t.Fatalf("unbounded inference retry: %+v %v", a, err)
	}
	hard := errors.New("identity mismatch")
	a = &recoveryAnalyzer{verifyFailure: hard}
	if _, err := (Processor{Analyzer: a}).analyzeRecover(context.Background(), Candidate{}, time.Second, time.Millisecond); !errors.Is(err, hard) || a.calls != 1 || a.probes != 1 {
		t.Fatalf("hard verification retried: %+v %v", a, err)
	}
	a = &recoveryAnalyzer{busyAnalyzer: busyAnalyzer{analyzeErr: hard}}
	if _, err := (Processor{Analyzer: a}).analyzeRecover(context.Background(), Candidate{}, time.Second, time.Millisecond); !errors.Is(err, hard) || a.calls != 1 || a.probes != 0 {
		t.Fatalf("hard inference retried: %+v %v", a, err)
	}
}
