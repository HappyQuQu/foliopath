package app

import (
	"context"
	"time"

	"github.com/HappyQuQu/foliopath/internal/autotag"
	"github.com/HappyQuQu/foliopath/internal/face"
	"github.com/HappyQuQu/foliopath/internal/semantic"
)

func (s *databaseService) HandoffAutomaticTagJob(ctx context.Context, j autotag.Job, now time.Time) (bool, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if s.store == nil {
		return false, errAIRepositoryNotReady
	}
	return s.store.HandoffAutomaticTagJob(ctx, j, now)
}
func (s *databaseService) HandoffSemanticBackfill(ctx context.Context, j semantic.BackfillJob, now time.Time) (bool, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if s.store == nil {
		return false, errAIRepositoryNotReady
	}
	return s.store.HandoffSemanticBackfill(ctx, j, now)
}
func (s *databaseService) HandoffFaceJob(ctx context.Context, j face.AnalysisJob, now time.Time) (bool, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if s.store == nil {
		return false, errAIRepositoryNotReady
	}
	return s.store.HandoffFaceJob(ctx, j, now)
}
func (q semanticJobQueue) Handoff(ctx context.Context, j semantic.BackfillJob) error {
	_, err := q.database.HandoffSemanticBackfill(ctx, j, q.clock().UTC())
	return err
}
func (q faceJobQueue) Handoff(ctx context.Context, j face.AnalysisJob) error {
	_, err := q.database.HandoffFaceJob(ctx, j, time.Now().UTC())
	return err
}
