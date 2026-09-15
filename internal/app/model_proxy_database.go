package app

import (
	"context"
	"github.com/HappyQuQu/foliopath/internal/aimodel"
)

func (s *databaseService) GetModelProxy(ctx context.Context) (aimodel.ProxySettings, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if s.store == nil {
		return aimodel.ProxySettings{}, errAIRepositoryNotReady
	}
	return s.store.GetModelProxy(ctx)
}
func (s *databaseService) SaveModelProxy(ctx context.Context, v aimodel.ProxySettings) (aimodel.ProxySettings, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if s.store == nil {
		return aimodel.ProxySettings{}, errAIRepositoryNotReady
	}
	return s.store.SaveModelProxy(ctx, v)
}
