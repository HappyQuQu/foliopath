package ocr

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/HappyQuQu/foliopath/internal/jobs"
)

type Status struct {
	Settings            Settings
	Coverage            Coverage
	ExpectedFingerprint string
	LatestJob           *Job
	State               string
}
type Service struct {
	repository  Repository
	identity    Identity
	fingerprint string
	signal      *jobs.Signal
}

func NewService(repository Repository, identity Identity, signal *jobs.Signal) (*Service, error) {
	fp, err := identity.Fingerprint()
	if repository == nil || signal == nil || err != nil || identity.InputContractSHA256 != InputContractSHA256() || identity.OutputContractSHA256 != OutputContractSHA256() {
		return nil, ErrInvalidIdentity
	}
	return &Service{repository: repository, identity: identity, fingerprint: fp, signal: signal}, nil
}
func (s *Service) Get(ctx context.Context, id int64) (Status, error) {
	v, err := s.repository.GetOCROverview(ctx, id)
	if err != nil {
		return v, err
	}
	v.ExpectedFingerprint = s.fingerprint
	latest := v.LatestJob

	switch {
	case v.Settings.Clearing:
		v.State = "clearing"
	case !v.Settings.Enabled:
		v.State = "disabled"
	case v.Settings.Fingerprint != s.fingerprint:
		v.State = "model_mismatch"
	case latest != nil && (latest.State == "queued" || latest.State == "running" || latest.State == "cancelling"):
		v.State = latest.State
	case v.Coverage.Failed > 0:
		v.State = "degraded"
	case v.Coverage.Pending > 0:
		v.State = "pending"
	default:
		v.State = "ready"
	}
	return v, nil
}
func (s *Service) Update(ctx context.Context, id, revision int64, enabled bool) (Status, error) {
	if _, err := s.repository.ConfigureOCR(ctx, id, revision, enabled, s.identity, time.Now()); err != nil {
		return Status{}, err
	}
	s.signal.Wake()
	return s.Get(ctx, id)
}
func (s *Service) Request(ctx context.Context, id, revision int64, mode Mode, key string) (Job, bool, error) {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return Job{}, false, err
	}
	j, replay, err := s.repository.AdmitOCRJob(ctx, id, revision, mode, key, "ocrjob_"+hex.EncodeToString(bytes[:]), time.Now())
	if err == nil {
		s.signal.Wake()
	}
	return j, replay, err
}
func (s *Service) Job(ctx context.Context, id int64, jobID string) (Job, error) {
	j, err := s.repository.GetOCRJob(ctx, jobID)
	if err != nil {
		return j, err
	}
	if j.LibraryID != id {
		return Job{}, ErrNotFound
	}
	return j, nil
}
func (s *Service) Cancel(ctx context.Context, id int64, jobID string, revision int64) (Job, error) {
	if _, err := s.Job(ctx, id, jobID); err != nil {
		return Job{}, err
	}
	if err := s.repository.CancelOCRJob(ctx, jobID, revision, time.Now()); err != nil {
		return Job{}, err
	}
	s.signal.Wake()
	return s.Job(ctx, id, jobID)
}
func (s *Service) Clear(ctx context.Context, id, revision int64) (Status, error) {
	if _, err := s.repository.BeginOCRClear(ctx, id, revision, time.Now()); err != nil {
		return Status{}, err
	}
	s.signal.Wake()
	return s.Get(ctx, id)
}

// Run drains only explicitly requested, durable clears. It never auto-enables
// libraries or schedules inference. Notifications are an optimization; polling
// and a bounded keyset pass make restart and shared wake consumption safe.
func (s *Service) Run(ctx context.Context) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	var after int64
	for {
		values, err := s.repository.ListOCRClearingLibraries(ctx, after, MaximumActiveJobs)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		for _, v := range values {
			if _, err = s.repository.DrainOCRClear(ctx, v.LibraryID, v.Epoch, MaximumClearBatch, time.Now()); err != nil && !errors.Is(err, ErrConflict) && !errors.Is(err, ErrNotFound) {
				if ctx.Err() != nil {
					return nil
				}
				return err
			}
			after = v.LibraryID
		}
		if len(values) < MaximumActiveJobs {
			after = 0
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		case <-s.signal.Notifications():
		}
	}
}
