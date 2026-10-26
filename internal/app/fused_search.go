package app

import (
	"context"
	"errors"

	"github.com/HappyQuQu/foliopath/internal/inference/remoteml"
	"github.com/HappyQuQu/foliopath/internal/search"
	"github.com/HappyQuQu/foliopath/internal/semantic"
)

func (s *databaseService) StreamSearchCandidates(ctx context.Context, request search.Scan, offer func(search.Candidate) error) error {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if s.store == nil {
		return errAIRepositoryNotReady
	}
	return s.store.StreamSearchCandidates(ctx, request, offer)
}

// Translate only transport availability at the composition boundary. Invalid
// identity/vector/output errors are never converted into successful empty pages.
type fusedTextEncoder struct{ semantic.TextEncoder }

func (e fusedTextEncoder) EncodeSemanticText(ctx context.Context, generation, text string) ([]float32, error) {
	v, err := e.TextEncoder.EncodeSemanticText(ctx, generation, text)
	if errors.Is(err, remoteml.ErrUnavailable) || errors.Is(err, remoteml.ErrBusy) || errors.Is(err, context.DeadlineExceeded) && ctx.Err() == nil {
		return nil, errors.Join(search.ErrUnavailable, err)
	}
	return v, err
}

func (s *databaseService) GetOCRSearchCoverage(ctx context.Context, scan search.Scan) (search.OCRCoverage, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if s.store == nil {
		return search.OCRCoverage{}, errAIRepositoryNotReady
	}
	return s.store.GetOCRSearchCoverage(ctx, scan)
}
