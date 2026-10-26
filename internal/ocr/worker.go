package ocr

import (
	"context"
	"errors"
	"time"

	"github.com/HappyQuQu/foliopath/internal/jobs"
)

type AssetAnalyzer interface {
	Analyze(context.Context, int64, Candidate) (StoredResult, error)
}
type Worker struct {
	repository  Repository
	analyzer    AssetAnalyzer
	fingerprint string
	now         func() time.Time
}

func NewWorker(repository Repository, analyzer AssetAnalyzer, fingerprint string) (*Worker, error) {
	if repository == nil || analyzer == nil || len(fingerprint) != 64 {
		return nil, ErrInvalidPersistence
	}
	return &Worker{repository: repository, analyzer: analyzer, fingerprint: fingerprint, now: time.Now}, nil
}

// Queue adapts the existing durable owner to the existing generic worker pool.
// It has no scheduler, heartbeat goroutine or cancellation state of its own.
type Queue struct {
	Repository Repository
	Now        func() time.Time
}

func (q Queue) clock() time.Time {
	if q.Now != nil {
		return q.Now()
	}
	return time.Now()
}
func (q Queue) RecoverExpired(ctx context.Context) (jobs.RecoverySummary, error) {
	return q.Repository.RecoverOCRJobs(ctx, q.clock())
}
func (q Queue) Claim(ctx context.Context, lease time.Duration) (Job, bool, error) {
	return q.Repository.ClaimOCRJob(ctx, q.clock(), lease)
}
func (q Queue) RefreshLease(ctx context.Context, j Job, lease time.Duration) (bool, error) {
	err := q.Repository.RefreshOCRLease(ctx, j, q.clock(), lease)
	if errors.Is(err, ErrConflict) || errors.Is(err, ErrNotFound) {
		return true, nil
	}
	return false, err
}
func (q Queue) Handoff(ctx context.Context, j Job) error {
	return q.Repository.HandoffOCRJob(ctx, j, q.clock())
}
func (w *Worker) terminal(j Job, state, code string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := w.repository.FinishOCRJob(ctx, j, state, code, w.now())
	if errors.Is(err, ErrConflict) && state != "cancelled" {
		// Cancellation may win just before the processor's terminal write, before
		// the next heartbeat tick. Acknowledge it without waiting for lease expiry.
		err = w.repository.FinishOCRJob(ctx, j, "cancelled", "", w.now())
	}
	if errors.Is(err, ErrConflict) || errors.Is(err, ErrNotFound) {
		return nil
	}
	return err
}
func (w *Worker) Process(ctx context.Context, j Job) error {
	if j.Fingerprint != w.fingerprint {
		return w.terminal(j, "failed", "identity_mismatch")
	}
	for {
		if ctx.Err() != nil {
			if errors.Is(context.Cause(ctx), jobs.ErrPlannedShutdown) {
				return ctx.Err()
			}
			return w.terminal(j, "cancelled", "")
		}
		var c Candidate
		var found bool
		scanDeadline := w.now().Add(60 * time.Second)
		err := w.waitForScan(ctx, scanDeadline, func() error { var e error; c, found, e = w.repository.NextOCRCandidate(ctx, j, w.now()); return e })
		if err != nil {
			if ctx.Err() != nil {
				continue
			}
			return w.terminal(j, "failed", "source_unavailable")
		}
		if !found {
			return w.terminal(j, "succeeded", "")
		}
		value, err := w.analyzer.Analyze(ctx, j.LibraryID, c)
		if ctx.Err() != nil {
			continue
		}
		if errors.Is(err, ErrRuntimeBusy) {
			err := w.repository.DeferOCRJob(ctx, j, w.now(), 5*time.Second)
			if errors.Is(err, ErrConflict) {
				return w.terminal(j, "cancelled", "")
			}
			return err
		}
		if err != nil {
			code := "inference_failed"
			switch {
			case errors.Is(err, ErrRuntimeIdentity):
				code = "identity_mismatch"
			case errors.Is(err, ErrRuntimeUnavailable):
				code = "model_unavailable"
			case errors.Is(err, ErrRuntimeOutputLimit):
				code = "output_limit"
			case errors.Is(err, ErrPreprocessFailed), errors.Is(err, ErrPreprocessUnavailable):
				code = "preprocess_failed"
			case errors.Is(err, ErrSourceUnavailable), errors.Is(err, ErrSourceChanged):
				code = "source_unavailable"
			}
			return w.terminal(j, "failed", code)
		}
		if value.Candidate != c || value.Fingerprint != j.Fingerprint {
			return w.terminal(j, "failed", "identity_mismatch")
		}
		if err = w.waitForScan(ctx, scanDeadline, func() error { return w.repository.CommitOCRResult(ctx, Commit{Job: j, Value: value}, w.now()) }); err != nil {
			if ctx.Err() != nil {
				continue
			}
			if errors.Is(err, ErrConflict) {
				return w.terminal(j, "failed", "source_unavailable")
			}
			return err
		}
		j.CheckpointID = c.AssetID
		j.Completed++
		if value.Outcome == "failed" {
			j.Failed++
		}
	}
}

func (w *Worker) BackgroundAdmissionCancelled(ctx context.Context, j Job) {
	if !errors.Is(context.Cause(ctx), jobs.ErrPlannedShutdown) {
		_ = w.terminal(j, "cancelled", "")
	}
}

// A transient catalog reconciliation may change ready to scanning after claim.
// Preserve the same claim and bounded result, never hold a transaction or redo
// inference, while the generic heartbeat continues checking cancellation.
func (w *Worker) waitForScan(ctx context.Context, deadline time.Time, call func() error) error {
	for {
		err := call()
		if !errors.Is(err, ErrScanning) {
			return err
		}
		if !w.now().Before(deadline) {
			return ErrOffline
		}
		timer := time.NewTimer(time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}
