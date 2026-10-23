package app

import (
	"context"
	"github.com/HappyQuQu/foliopath/internal/aimodel"
	"github.com/HappyQuQu/foliopath/internal/autotag"
	"github.com/HappyQuQu/foliopath/internal/jobs"
	"time"
)

func (s *databaseService) FindAutomaticTagRequest(ctx context.Context, key, request string) (aimodel.Operation, bool, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if s.store == nil {
		return aimodel.Operation{}, false, errAIRepositoryNotReady
	}
	return s.store.FindAutomaticTagRequest(ctx, key, request)
}

func (s *databaseService) GetAutomaticTagSettings(ctx context.Context, id int64, identity string) (autotag.Settings, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if s.store == nil {
		return autotag.Settings{}, errAIRepositoryNotReady
	}
	return s.store.GetAutomaticTagSettings(ctx, id, identity)
}

func (s *databaseService) UpdateAutomaticTagSettings(ctx context.Context, id int64, enabled bool, revision int64, now time.Time) error {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if s.store == nil {
		return errAIRepositoryNotReady
	}
	return s.store.UpdateAutomaticTagSettings(ctx, id, enabled, revision, now)
}

func (s *databaseService) ListAutomaticTagLibraries(ctx context.Context, after int64, limit int) ([]int64, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if s.store == nil {
		return nil, errAIRepositoryNotReady
	}
	return s.store.ListAutomaticTagLibraries(ctx, after, limit)
}

func (s *databaseService) AdmitAutomaticTagJob(ctx context.Context, a autotag.Admission, now time.Time) (aimodel.Operation, bool, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if s.store == nil {
		return aimodel.Operation{}, false, errAIRepositoryNotReady
	}
	return s.store.AdmitAutomaticTagJob(ctx, a, now)
}

func (s *databaseService) ClaimAutomaticTagJob(ctx context.Context, now time.Time, lease time.Duration) (autotag.Job, bool, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if s.store == nil {
		return autotag.Job{}, false, errAIRepositoryNotReady
	}
	return s.store.ClaimAutomaticTagJob(ctx, now, lease)
}

func (s *databaseService) RefreshAutomaticTagLease(ctx context.Context, j autotag.Job, now time.Time, lease time.Duration) (bool, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if s.store == nil {
		return false, errAIRepositoryNotReady
	}
	return s.store.RefreshAutomaticTagLease(ctx, j, now, lease)
}

func (s *databaseService) RecoverAutomaticTagJobs(ctx context.Context, now time.Time) (jobs.RecoverySummary, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if s.store == nil {
		return jobs.RecoverySummary{}, errAIRepositoryNotReady
	}
	return s.store.RecoverAutomaticTagJobs(ctx, now)
}

func (s *databaseService) NextAutomaticTagCandidate(ctx context.Context, j autotag.Job) (autotag.Candidate, bool, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if s.store == nil {
		return autotag.Candidate{}, false, errAIRepositoryNotReady
	}
	return s.store.NextAutomaticTagCandidate(ctx, j)
}

func (s *databaseService) CommitAutomaticTagResult(ctx context.Context, c autotag.Commit, now time.Time) error {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if s.store == nil {
		return errAIRepositoryNotReady
	}
	return s.store.CommitAutomaticTagResult(ctx, c, now)
}

func (s *databaseService) DeferAutomaticTagJob(ctx context.Context, j autotag.Job, now time.Time, delay time.Duration) error {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if s.store == nil {
		return errAIRepositoryNotReady
	}
	return s.store.DeferAutomaticTagJob(ctx, j, now, delay)
}

func (s *databaseService) FinishAutomaticTagJob(ctx context.Context, j autotag.Job, state, code string, now time.Time) error {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if s.store == nil {
		return errAIRepositoryNotReady
	}
	return s.store.FinishAutomaticTagJob(ctx, j, state, code, now)
}

func (s *databaseService) CancelAutomaticTagOperation(ctx context.Context, id string, revision int64, now time.Time) (aimodel.Operation, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if s.store == nil {
		return aimodel.Operation{}, errAIRepositoryNotReady
	}
	return s.store.CancelAutomaticTagOperation(ctx, id, revision, now)
}
