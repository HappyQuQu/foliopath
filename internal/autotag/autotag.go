// Package autotag owns bounded, restart-safe automatic image tagging (ADR-0021).
// Curation remains the only owner of visible tags and manual protection policy.
package autotag

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
	"github.com/HappyQuQu/foliopath/internal/curation"
	"github.com/HappyQuQu/foliopath/internal/jobs"
)

var (
	ErrInvalid                = errors.New("invalid automatic tag request")
	ErrNotFound               = errors.New("automatic tag library not found")
	ErrDisabled               = errors.New("automatic tags disabled")
	ErrOffline                = errors.New("automatic tag source offline")
	ErrUnavailable            = errors.New("automatic tag model unavailable")
	ErrTemporarilyUnavailable = errors.New("automatic tag service temporarily unavailable")
	ErrBusy                   = errors.New("automatic tag inference busy")
	ErrConflict               = errors.New("automatic tag operation conflict")
	ErrSourceChanged          = errors.New("automatic tag source changed")
	ErrImage                  = errors.New("automatic tag image could not be processed")
)

type Mode string

const (
	Missing               Mode = "missing"
	All                   Mode = "all"
	MaximumActiveJobs          = 32
	BusyRetryDelay             = 30 * time.Second
	ServiceRecoveryWindow      = 60 * time.Second
)

type Model struct{ Identity, DisplayName, Protocol string }
type Coverage struct{ Eligible, Completed, Failed, Stale, Revision int64 }
type Settings struct {
	LibraryID         int64
	Enabled           bool
	Revision          int64
	State             string
	Offline           bool
	AutomaticBlocked  bool
	Model             *Model
	Coverage          Coverage
	ActiveOperationID string
}
type Job struct {
	ID, OperationID, ModelIdentity                          string
	LibraryID                                               int64
	Mode                                                    Mode
	State                                                   string
	CheckpointID, UpperAssetID, ClaimRevision, AttemptCount int64
}
type Admission struct {
	Job                  Job
	KeyHash, RequestHash string
	Automatic            bool
}
type Candidate struct {
	ID, LibraryID     int64
	SourceFingerprint string
}
type Commit struct {
	Job     Job
	Asset   Candidate
	Plan    *curation.AutomaticTagPlan
	Outcome string
}

// Repository implementations must atomically fence commits and curation writes.
type Repository interface {
	GetAutomaticTagSettings(context.Context, int64, string) (Settings, error)
	UpdateAutomaticTagSettings(context.Context, int64, bool, int64, time.Time) error
	ListAutomaticTagLibraries(context.Context, int64, int) ([]int64, error)
	FindAutomaticTagRequest(context.Context, string, string) (aimodel.Operation, bool, error)
	AdmitAutomaticTagJob(context.Context, Admission, time.Time) (aimodel.Operation, bool, error)
	ClaimAutomaticTagJob(context.Context, time.Time, time.Duration) (Job, bool, error)
	RefreshAutomaticTagLease(context.Context, Job, time.Time, time.Duration) (bool, error)
	RecoverAutomaticTagJobs(context.Context, time.Time) (jobs.RecoverySummary, error)
	NextAutomaticTagCandidate(context.Context, Job) (Candidate, bool, error)
	CommitAutomaticTagResult(context.Context, Commit, time.Time) error
	DeferAutomaticTagJob(context.Context, Job, time.Time, time.Duration) error
	HandoffAutomaticTagJob(context.Context, Job, time.Time) (bool, error)
	FinishAutomaticTagJob(context.Context, Job, string, string, time.Time) error
	CancelAutomaticTagOperation(context.Context, string, int64, time.Time) (aimodel.Operation, error)
}

// Analyzer owns bounded original-image reading/preprocessing outside storage.
type Analyzer interface {
	Model() *Model
	Verify(context.Context) error
	Analyze(context.Context, Candidate) ([]curation.AutomaticLabel, error)
}
type Service struct {
	repository Repository
	analyzer   Analyzer
	signal     *jobs.Signal
	reconcile  *jobs.Signal
	ready      atomic.Bool
}

func New(repository Repository, analyzer Analyzer, signal *jobs.Signal) (*Service, error) {
	if repository == nil || analyzer == nil || signal == nil {
		return nil, ErrInvalid
	}
	return &Service{repository: repository, analyzer: analyzer, signal: signal, reconcile: jobs.NewSignal()}, nil
}
func (s *Service) identity() string {
	if model := s.analyzer.Model(); model != nil {
		return model.Identity
	}
	return ""
}
func (s *Service) Get(ctx context.Context, libraryID int64) (Settings, error) {
	value, err := s.repository.GetAutomaticTagSettings(ctx, libraryID, s.identity())
	if err != nil {
		return value, err
	}
	value.Model = s.analyzer.Model()
	switch {
	case !value.Enabled:
		value.State = "disabled"
	case value.Model == nil || !s.ready.Load():
		value.State = "awaiting_model"
	case value.AutomaticBlocked:
		value.State = "degraded"
	case value.ActiveOperationID != "":
		value.State = "building"
	case value.Offline || value.Coverage.Failed > 0 || value.Coverage.Stale > 0:
		value.State = "degraded"
	case value.Coverage.Completed < value.Coverage.Eligible:
		value.State = "building"
	default:
		value.State = "ready"
	}
	return value, nil
}
func (s *Service) Update(ctx context.Context, libraryID int64, enabled bool, revision int64) (Settings, error) {
	if err := s.repository.UpdateAutomaticTagSettings(ctx, libraryID, enabled, revision, time.Now()); err != nil {
		return Settings{}, err
	}
	s.signal.Wake()
	s.reconcile.Wake()
	return s.Get(ctx, libraryID)
}
func digest(value string) string { v := sha256.Sum256([]byte(value)); return hex.EncodeToString(v[:]) }
func identifier() (string, error) {
	var v [16]byte
	if _, err := rand.Read(v[:]); err != nil {
		return "", err
	}
	return "autotag_" + hex.EncodeToString(v[:]), nil
}
func (s *Service) Request(ctx context.Context, libraryID int64, mode Mode, key string) (aimodel.Operation, bool, error) {
	if libraryID < 1 || (mode != Missing && mode != All) || len(key) < 8 || len(key) > 128 {
		return aimodel.Operation{}, false, ErrInvalid
	}
	operation, found, err := s.repository.FindAutomaticTagRequest(ctx, digest("autotag-key:"+key), digest(fmt.Sprintf("autotag-v1:%d:%s:%s", libraryID, mode, s.identity())))
	if err != nil || found {
		return operation, found, err
	}
	settings, err := s.repository.GetAutomaticTagSettings(ctx, libraryID, s.identity())
	if err != nil {
		return aimodel.Operation{}, false, err
	}
	if !settings.Enabled {
		return aimodel.Operation{}, false, ErrDisabled
	}
	if settings.Offline {
		return aimodel.Operation{}, false, ErrOffline
	}
	if err := s.analyzer.Verify(ctx); err != nil {
		if !errors.Is(err, ErrBusy) {
			s.ready.Store(false)
		}
		// A full shared slot budget is not evidence that a previously verified
		// model disappeared. The worker still verifies identity before each image.
		if !errors.Is(err, ErrBusy) || !s.ready.Load() {
			return aimodel.Operation{}, false, ErrUnavailable
		}
	} else {
		s.ready.Store(true)
	}
	return s.admit(ctx, libraryID, mode, key, false)
}
func (s *Service) admit(ctx context.Context, libraryID int64, mode Mode, key string, automatic bool) (aimodel.Operation, bool, error) {
	id, err := identifier()
	if err != nil {
		return aimodel.Operation{}, false, err
	}
	op, err := identifier()
	if err != nil {
		return aimodel.Operation{}, false, err
	}
	value := Admission{Job: Job{ID: id, OperationID: op, LibraryID: libraryID, ModelIdentity: s.identity(), Mode: mode}, Automatic: automatic}
	if !automatic {
		value.KeyHash = digest("autotag-key:" + key)
		value.RequestHash = digest(fmt.Sprintf("autotag-v1:%d:%s:%s", libraryID, mode, s.identity()))
	}
	operation, replayed, err := s.repository.AdmitAutomaticTagJob(ctx, value, time.Now())
	if err == nil {
		s.signal.Wake()
	}
	return operation, replayed, err
}
func (s *Service) CancelAutomaticTagOperation(ctx context.Context, id string, revision int64) (aimodel.Operation, error) {
	value, err := s.repository.CancelAutomaticTagOperation(ctx, id, revision, time.Now())
	if err == nil {
		s.signal.Wake()
	}
	return value, err
}

// Run reconciles at most 50 libraries per turn. The persistent cursor is not
// needed: restart restarts a bounded pass, and per-image identities are durable.
func (s *Service) Run(ctx context.Context) error {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	var after int64
	for {
		probeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err := s.analyzer.Verify(probeCtx)
		cancel()
		if !errors.Is(err, ErrBusy) {
			s.ready.Store(err == nil)
		}
		if err == nil {
			ids, listErr := s.repository.ListAutomaticTagLibraries(ctx, after, 50)
			if listErr != nil {
				return listErr
			}
			if len(ids) == 0 {
				after = 0
			} else {
				for _, id := range ids {
					after = id
					_, _, admitErr := s.admit(ctx, id, Missing, "", true)
					if admitErr != nil && !errors.Is(admitErr, ErrConflict) && !errors.Is(admitErr, ErrDisabled) && !errors.Is(admitErr, ErrOffline) && !errors.Is(admitErr, ErrNotFound) {
						return admitErr
					}
				}
				if len(ids) < 50 {
					after = 0
				}
			}
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		case <-s.reconcile.Notifications():
		}
	}
}

type Queue struct{ Repository Repository }

func (q Queue) RecoverExpired(ctx context.Context) (jobs.RecoverySummary, error) {
	return q.Repository.RecoverAutomaticTagJobs(ctx, time.Now())
}
func (q Queue) Claim(ctx context.Context, lease time.Duration) (Job, bool, error) {
	return q.Repository.ClaimAutomaticTagJob(ctx, time.Now(), lease)
}
func (q Queue) RefreshLease(ctx context.Context, j Job, lease time.Duration) (bool, error) {
	return q.Repository.RefreshAutomaticTagLease(ctx, j, time.Now(), lease)
}
func (q Queue) Handoff(ctx context.Context, j Job) error {
	_, err := q.Repository.HandoffAutomaticTagJob(ctx, j, time.Now())
	return err
}

type Processor struct {
	Repository Repository
	Analyzer   Analyzer
}

// Three attempts and two cancellable waits bound a busy turn. Persistent busy
// yields the same durable operation for a later claim; it never commits an empty
// result, consumes a crash-recovery attempt, or creates a replacement operation.
func (p Processor) analyzeBusy(ctx context.Context, asset Candidate) ([]curation.AutomaticLabel, error) {
	for attempt := 0; attempt < 3; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		labels, err := p.Analyzer.Analyze(ctx, asset)
		if !errors.Is(err, ErrBusy) || attempt == 2 {
			return labels, err
		}
		timer := time.NewTimer(time.Duration(attempt+1) * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
	return nil, ErrBusy
}

func (p Processor) analyze(ctx context.Context, asset Candidate) ([]curation.AutomaticLabel, error) {
	return p.analyzeRecover(ctx, asset, ServiceRecoveryWindow, time.Second)
}

// ADR-0030: preserve the same live claim/checkpoint during a short AI outage.
// Probe before resending input; hard identity/output failures never enter recovery.
func (p Processor) analyzeRecover(parent context.Context, asset Candidate, window, interval time.Duration) ([]curation.AutomaticLabel, error) {
	labels, err := p.analyzeBusy(parent, asset)
	if !errors.Is(err, ErrTemporarilyUnavailable) {
		return labels, err
	}
	ctx, cancel := context.WithTimeout(parent, window)
	defer cancel()
	for retry := 0; retry < 2; {
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			if parent.Err() != nil {
				return nil, parent.Err()
			}
			return nil, ErrTemporarilyUnavailable
		case <-timer.C:
		}
		err = p.Analyzer.Verify(ctx)
		if errors.Is(err, ErrTemporarilyUnavailable) || errors.Is(err, ErrBusy) {
			continue
		}
		if err != nil {
			return nil, err
		}
		labels, err = p.analyzeBusy(ctx, asset)
		retry++
		if !errors.Is(err, ErrTemporarilyUnavailable) {
			return labels, err
		}
	}
	return nil, ErrTemporarilyUnavailable
}

func (p Processor) BackgroundAdmissionCancelled(_ context.Context, j Job) {
	p.finish(j, "cancelled", "cancelled")
}
func (p Processor) finish(j Job, state, code string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return p.Repository.FinishAutomaticTagJob(ctx, j, state, code, time.Now())
}
func (p Processor) Process(ctx context.Context, j Job) error {
	if model := p.Analyzer.Model(); model == nil || model.Identity != j.ModelIdentity {
		p.finish(j, "failed", "model_unavailable")
		return ErrUnavailable
	}
	for {
		if ctx.Err() != nil {
			p.finish(j, "cancelled", "cancelled")
			return ctx.Err()
		}
		asset, found, err := p.Repository.NextAutomaticTagCandidate(ctx, j)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			p.finish(j, "failed", "operation_interrupted")
			return err
		}
		if !found {
			return p.finish(j, "succeeded", "")
		}
		labels, err := p.analyze(ctx, asset)
		if ctx.Err() != nil {
			p.finish(j, "cancelled", "cancelled")
			return ctx.Err()
		}
		commit := Commit{Job: j, Asset: asset, Outcome: "ready"}
		switch {
		case err == nil:
			plan, normalizeErr := curation.NormalizeAutomaticTagPlan(curation.AutomaticTagPlan{LibraryID: j.LibraryID, AssetID: asset.ID, SourceFingerprint: asset.SourceFingerprint, ModelIdentity: j.ModelIdentity, Labels: labels})
			if normalizeErr != nil {
				p.finish(j, "failed", "model_invalid_output")
				return normalizeErr
			}
			commit.Plan = &plan
		case errors.Is(err, ErrSourceChanged):
			commit.Outcome = "stale"
		case errors.Is(err, ErrImage):
			commit.Outcome = "failed"
		case errors.Is(err, ErrBusy):
			if deferErr := p.Repository.DeferAutomaticTagJob(ctx, j, time.Now(), BusyRetryDelay); deferErr != nil {
				if ctx.Err() != nil || errors.Is(deferErr, ErrConflict) {
					p.finish(j, "cancelled", "cancelled")
				}
				return deferErr
			}
			return ErrBusy
		case errors.Is(err, ErrOffline):
			p.finish(j, "failed", "source_offline")
			return err
		default:
			p.finish(j, "failed", "model_unavailable")
			return err
		}
		if err = p.Repository.CommitAutomaticTagResult(ctx, commit, time.Now()); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if errors.Is(err, ErrConflict) {
				p.finish(j, "cancelled", "cancelled")
			} else {
				p.finish(j, "failed", "operation_interrupted")
			}
			return err
		}
		j.CheckpointID = asset.ID
	}
}
