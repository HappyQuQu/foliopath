package face

import (
	"context"
	"errors"
	"testing"
	"time"
)

type scanWaitQueue struct {
	JobQueue
	calls, cancelAt int
	finished        string
}

func (q *scanWaitQueue) RefreshFaceJobLease(ctx context.Context, _ AnalysisJob, _ time.Time, _ time.Duration) (bool, error) {
	q.calls++
	return q.cancelAt > 0 && q.calls >= q.cancelAt, ctx.Err()
}
func (q *scanWaitQueue) FinishFaceJob(_ context.Context, j AnalysisJob, _ bool, code string, _ time.Time) (AnalysisJob, error) {
	q.finished = code
	return j, nil
}

type scanWaitCatalog struct {
	JobCatalog
	states []JobLibraryState
	calls  int
}

func (c *scanWaitCatalog) FaceJobLibraryState(context.Context, int64, string) (JobLibraryState, error) {
	i := min(c.calls, len(c.states)-1)
	c.calls++
	return c.states[i], nil
}

func TestScanWaitIsBoundedAndRechecksControl(t *testing.T) {
	for _, tc := range []struct {
		name     string
		states   []JobLibraryState
		cancelAt int
		wantStop bool
		code     string
	}{
		{"resumes", []JobLibraryState{JobLibraryScanning, JobLibraryScanning, JobLibraryReady}, 0, false, ""},
		{"times out", []JobLibraryState{JobLibraryScanning}, 0, true, "library_scanning"},
		{"cancels", []JobLibraryState{JobLibraryScanning}, 2, true, "cancelled"},
		{"offline", []JobLibraryState{JobLibraryScanning, JobLibraryOffline}, 0, true, "library_offline"},
		{"disabled", []JobLibraryState{JobLibraryScanning, JobLibraryDisabled}, 0, true, "face_disabled"},
		{"model unavailable", []JobLibraryState{JobLibraryScanning, JobLibraryModelUnavailable}, 0, true, "model_unavailable"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			q, c := &scanWaitQueue{cancelAt: tc.cancelAt}, &scanWaitCatalog{states: tc.states}
			p := &JobProcessor{queue: q, catalog: c, now: time.Now, scanWait: 20 * time.Millisecond, scanPoll: time.Millisecond}
			stop, err := p.refreshControl(context.Background(), AnalysisJob{})
			if err != nil || stop != tc.wantStop || q.finished != tc.code {
				t.Fatalf("stop=%v error=%v code=%q", stop, err, q.finished)
			}
			if q.calls < 2 {
				t.Fatal("control was not rechecked")
			}
		})
	}
}

func TestScanWaitHonorsShutdownWithoutTerminalFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	q := &scanWaitQueue{}
	p := &JobProcessor{queue: q, catalog: &scanWaitCatalog{states: []JobLibraryState{JobLibraryScanning}}, now: time.Now}
	_, err := p.refreshControl(ctx, AnalysisJob{})
	if !errors.Is(err, context.Canceled) || q.finished != "" {
		t.Fatalf("err=%v finished=%q", err, q.finished)
	}
}

func TestFailureCodesNeverExposeExceptionText(t *testing.T) {
	for _, tc := range []struct {
		err  error
		code string
	}{
		{ErrMediaUnreadable, "media_unreadable"}, {ErrInvalidInput, "invalid_media"}, {ErrInvalidOutput, "inference_invalid_output"},
		{context.DeadlineExceeded, "inference_timeout"}, {errors.New("/private/secret.jpg: SQL internal"), "analysis_failed"},
	} {
		if got := FailureCode(tc.err); got != tc.code || !ValidFailureCode(got) {
			t.Fatalf("code=%q", got)
		}
	}
}
