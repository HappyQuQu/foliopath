package app

import (
	"context"
	"github.com/HappyQuQu/foliopath/internal/jobs"
	"github.com/HappyQuQu/foliopath/internal/ocr"
	"time"
)

// OCR forwards through the application's existing database lifecycle lock only.
// State transitions remain exclusively in the capability's SQLite adapter.
func (s *databaseService) GetOCRSettings(ctx context.Context, id int64) (ocr.Settings, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if s.store == nil {
		return ocr.Settings{}, errAIRepositoryNotReady
	}
	return s.store.GetOCRSettings(ctx, id)
}
func (s *databaseService) ConfigureOCR(ctx context.Context, id, rev int64, enabled bool, identity ocr.Identity, now time.Time) (ocr.Settings, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if s.store == nil {
		return ocr.Settings{}, errAIRepositoryNotReady
	}
	return s.store.ConfigureOCR(ctx, id, rev, enabled, identity, now)
}
func (s *databaseService) GetOCRCoverage(ctx context.Context, id int64) (ocr.Coverage, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if s.store == nil {
		return ocr.Coverage{}, errAIRepositoryNotReady
	}
	return s.store.GetOCRCoverage(ctx, id)
}
func (s *databaseService) ListOCRClearingLibraries(ctx context.Context, after int64, limit int) ([]ocr.Settings, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if s.store == nil {
		return nil, errAIRepositoryNotReady
	}
	return s.store.ListOCRClearingLibraries(ctx, after, limit)
}
func (s *databaseService) GetOCRJob(ctx context.Context, id string) (ocr.Job, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if s.store == nil {
		return ocr.Job{}, errAIRepositoryNotReady
	}
	return s.store.GetOCRJob(ctx, id)
}
func (s *databaseService) CommitOCRResult(ctx context.Context, v ocr.Commit, now time.Time) error {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if s.store == nil {
		return errAIRepositoryNotReady
	}
	return s.store.CommitOCRResult(ctx, v, now)
}
func (s *databaseService) RefreshOCRLease(ctx context.Context, j ocr.Job, now time.Time, lease time.Duration) error {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if s.store == nil {
		return errAIRepositoryNotReady
	}
	return s.store.RefreshOCRLease(ctx, j, now, lease)
}
func (s *databaseService) RecoverOCRJobs(ctx context.Context, now time.Time) (jobs.RecoverySummary, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if s.store == nil {
		return jobs.RecoverySummary{}, errAIRepositoryNotReady
	}
	return s.store.RecoverOCRJobs(ctx, now)
}
func (s *databaseService) DeferOCRJob(ctx context.Context, j ocr.Job, now time.Time, delay time.Duration) error {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if s.store == nil {
		return errAIRepositoryNotReady
	}
	return s.store.DeferOCRJob(ctx, j, now, delay)
}
func (s *databaseService) HandoffOCRJob(ctx context.Context, j ocr.Job, now time.Time) error {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if s.store == nil {
		return errAIRepositoryNotReady
	}
	return s.store.HandoffOCRJob(ctx, j, now)
}
func (s *databaseService) FinishOCRJob(ctx context.Context, j ocr.Job, state, code string, now time.Time) error {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if s.store == nil {
		return errAIRepositoryNotReady
	}
	return s.store.FinishOCRJob(ctx, j, state, code, now)
}
func (s *databaseService) CancelOCRJob(ctx context.Context, id string, rev int64, now time.Time) error {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if s.store == nil {
		return errAIRepositoryNotReady
	}
	return s.store.CancelOCRJob(ctx, id, rev, now)
}
func (s *databaseService) BeginOCRClear(ctx context.Context, id, rev int64, now time.Time) (ocr.Settings, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if s.store == nil {
		return ocr.Settings{}, errAIRepositoryNotReady
	}
	return s.store.BeginOCRClear(ctx, id, rev, now)
}
func (s *databaseService) DrainOCRClear(ctx context.Context, id, epoch int64, limit int, now time.Time) (bool, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if s.store == nil {
		return false, errAIRepositoryNotReady
	}
	return s.store.DrainOCRClear(ctx, id, epoch, limit, now)
}
func (s *databaseService) AdmitOCRJob(ctx context.Context, id, rev int64, mode ocr.Mode, key, jobID string, now time.Time) (ocr.Job, bool, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if s.store == nil {
		return ocr.Job{}, false, errAIRepositoryNotReady
	}
	return s.store.AdmitOCRJob(ctx, id, rev, mode, key, jobID, now)
}
func (s *databaseService) ClaimOCRJob(ctx context.Context, now time.Time, lease time.Duration) (ocr.Job, bool, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if s.store == nil {
		return ocr.Job{}, false, errAIRepositoryNotReady
	}
	return s.store.ClaimOCRJob(ctx, now, lease)
}
func (s *databaseService) NextOCRCandidate(ctx context.Context, j ocr.Job, now time.Time) (ocr.Candidate, bool, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if s.store == nil {
		return ocr.Candidate{}, false, errAIRepositoryNotReady
	}
	return s.store.NextOCRCandidate(ctx, j, now)
}
func (s *databaseService) GetLatestOCRJob(ctx context.Context, id int64) (ocr.Job, bool, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if s.store == nil {
		return ocr.Job{}, false, errAIRepositoryNotReady
	}
	return s.store.GetLatestOCRJob(ctx, id)
}
func (s *databaseService) GetCurrentOCRResult(ctx context.Context, id, asset int64) (ocr.StoredResult, bool, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if s.store == nil {
		return ocr.StoredResult{}, false, errAIRepositoryNotReady
	}
	return s.store.GetCurrentOCRResult(ctx, id, asset)
}

func (s *databaseService) GetOCROverview(ctx context.Context, id int64) (ocr.Status, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if s.store == nil {
		return ocr.Status{}, errAIRepositoryNotReady
	}
	return s.store.GetOCROverview(ctx, id)
}
