package app

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
	"github.com/HappyQuQu/foliopath/internal/inference/onnx"
	"github.com/HappyQuQu/foliopath/internal/semantic"
)

const (
	semanticInferenceTimeout = 30 * time.Second
	semanticSessionIdle      = 5 * time.Minute
)

type semanticSessionFactory interface {
	ValidateSemanticImageSession(context.Context, string) error
	OpenSemanticImageSession(context.Context, string) (onnx.ImageSession, error)
}

type semanticSessionOwner struct {
	factory semanticSessionFactory
	timeout time.Duration
	idle    time.Duration
	now     func() time.Time

	mu           sync.Mutex
	session      onnx.ImageSession
	generationID string
	lastUsed     time.Time

	residentSessions atomic.Int64
	activeRuns       atomic.Int64
	totalLoads       atomic.Int64
	totalRuns        atomic.Int64
	totalUnloads     atomic.Int64
}

type semanticSessionResources struct {
	ResidentSessions int64
	ActiveRuns       int64
	TotalLoads       int64
	TotalRuns        int64
	TotalUnloads     int64
}

func newSemanticSessionOwner(factory semanticSessionFactory, timeout, idle time.Duration, now func() time.Time) (*semanticSessionOwner, error) {
	if factory == nil {
		return nil, errors.New("semantic session factory is required")
	}
	if timeout == 0 {
		timeout = semanticInferenceTimeout
	}
	if idle == 0 {
		idle = semanticSessionIdle
	}
	if timeout < time.Millisecond || timeout > time.Minute || idle < time.Second {
		return nil, errors.New("semantic session timing is invalid")
	}
	if now == nil {
		now = time.Now
	}
	return &semanticSessionOwner{factory: factory, timeout: timeout, idle: idle, now: now}, nil
}

func (owner *semanticSessionOwner) EncodeSemanticImage(ctx context.Context, generationID string, input []float32) ([]float32, error) {
	if generationID == "" {
		return nil, semantic.ErrImageEncoderUnavailable
	}
	owner.mu.Lock()
	defer owner.mu.Unlock()
	validateCtx, validateCancel := context.WithTimeout(ctx, owner.timeout)
	validateErr := owner.factory.ValidateSemanticImageSession(validateCtx, generationID)
	validateCancel()
	if validateErr != nil {
		owner.closeLocked()
		return nil, errors.Join(semantic.ErrImageEncoderUnavailable, validateErr)
	}
	if owner.session == nil || owner.generationID != generationID {
		owner.closeLocked()
		loadCtx, cancel := context.WithTimeout(ctx, owner.timeout)
		session, err := owner.factory.OpenSemanticImageSession(loadCtx, generationID)
		cancel()
		if err != nil {
			return nil, errors.Join(semantic.ErrImageEncoderUnavailable, err)
		}
		owner.session, owner.generationID = session, generationID
		owner.residentSessions.Store(1)
		owner.totalLoads.Add(1)
	}
	runCtx, cancel := context.WithTimeout(ctx, owner.timeout)
	owner.activeRuns.Add(1)
	owner.totalRuns.Add(1)
	output, err := owner.session.Encode(runCtx, input)
	owner.activeRuns.Add(-1)
	cancel()
	owner.lastUsed = owner.now().UTC()
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}
		owner.closeLocked()
		return nil, errors.Join(semantic.ErrImageEncoderUnavailable, err)
	}
	return output, nil
}

func (owner *semanticSessionOwner) Resources() semanticSessionResources {
	return semanticSessionResources{
		ResidentSessions: owner.residentSessions.Load(),
		ActiveRuns:       owner.activeRuns.Load(),
		TotalLoads:       owner.totalLoads.Load(),
		TotalRuns:        owner.totalRuns.Load(),
		TotalUnloads:     owner.totalUnloads.Load(),
	}
}

func (owner *semanticSessionOwner) Run(ctx context.Context) error {
	defer owner.Close()
	ticker := time.NewTicker(min(owner.idle/2, time.Minute))
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			owner.mu.Lock()
			if owner.session != nil && owner.now().UTC().Sub(owner.lastUsed) >= owner.idle {
				owner.closeLocked()
			}
			owner.mu.Unlock()
		}
	}
}

func (owner *semanticSessionOwner) Close() error {
	owner.mu.Lock()
	defer owner.mu.Unlock()
	return owner.closeLocked()
}

func (owner *semanticSessionOwner) closeLocked() error {
	session := owner.session
	owner.session, owner.generationID, owner.lastUsed = nil, "", time.Time{}
	if session != nil {
		err := session.Close()
		owner.residentSessions.Store(0)
		owner.totalUnloads.Add(1)
		return err
	}
	return nil
}

type semanticProductionSessionFactory struct {
	generations  semantic.GenerationRuntimeRepository
	models       *aimodel.Service
	catalog      *aimodel.Catalog
	source       aimodel.ActivationPackageSource
	availability aimodel.AvailabilityMarker
	runtime      interface {
		OpenImageSession(context.Context, aimodel.Manifest, aimodel.RuntimeFileOpener) (onnx.ImageSession, error)
	}
}

func (factory semanticProductionSessionFactory) OpenSemanticImageSession(ctx context.Context, generationID string) (onnx.ImageSession, error) {
	generation, model, manifest, err := factory.resolve(ctx, generationID)
	if err != nil {
		return nil, err
	}
	_ = generation
	if err := factory.source.ValidateActivationSource(ctx, model, manifest); err != nil {
		return nil, errors.Join(err, factory.markUnavailableAfterValidationFailure(ctx, model, err))
	}
	session, err := factory.runtime.OpenImageSession(ctx, manifest, func(openCtx context.Context, name string) (aimodel.RuntimeModelFile, error) {
		return factory.source.OpenActivationModelFile(openCtx, model, name)
	})
	if err != nil && (errors.Is(err, aimodel.ErrModelSourceUnavailable) || errors.Is(err, aimodel.ErrModelIncompatible)) {
		err = errors.Join(err, factory.markUnavailableAfterValidationFailure(ctx, model, err))
	}
	return session, err
}

func (factory semanticProductionSessionFactory) ValidateSemanticImageSession(ctx context.Context, generationID string) error {
	_, _, _, err := factory.resolve(ctx, generationID)
	return err
}

func (factory semanticProductionSessionFactory) resolve(ctx context.Context, generationID string) (semantic.GenerationRuntime, aimodel.Model, aimodel.Manifest, error) {
	if factory.generations == nil || factory.models == nil || factory.catalog == nil || factory.source == nil || factory.availability == nil || factory.runtime == nil {
		return semantic.GenerationRuntime{}, aimodel.Model{}, aimodel.Manifest{}, semantic.ErrImageEncoderUnavailable
	}
	generation, err := factory.generations.GetSemanticGenerationRuntime(ctx, generationID)
	if err != nil || generation.State != "active" || generation.EmbeddingDimension != int(onnx.EmbeddingDimension) {
		return semantic.GenerationRuntime{}, aimodel.Model{}, aimodel.Manifest{}, errors.Join(semantic.ErrSemanticGenerationUnavailable, err)
	}
	model, err := factory.models.Get(ctx, generation.ModelID)
	if err != nil || model.State != aimodel.StateAvailable {
		return semantic.GenerationRuntime{}, aimodel.Model{}, aimodel.Manifest{}, errors.Join(aimodel.ErrModelUnavailable, err)
	}
	manifest, found := factory.catalog.Manifest(model.Package.PackageID)
	if !found {
		return semantic.GenerationRuntime{}, aimodel.Model{}, aimodel.Manifest{}, aimodel.ErrModelIncompatible
	}
	if manifest.FormatVersion != aimodel.SemanticFormatVersion || manifest.Contracts == nil ||
		manifest.Contracts.ImagePreprocess != aimodel.SemanticImagePreprocessContract {
		return semantic.GenerationRuntime{}, aimodel.Model{}, aimodel.Manifest{}, aimodel.ErrModelIncompatible
	}
	return generation, model, manifest, nil
}

func (factory semanticProductionSessionFactory) markUnavailableAfterValidationFailure(ctx context.Context, model aimodel.Model, validationErr error) error {
	if errors.Is(validationErr, context.Canceled) || errors.Is(validationErr, context.DeadlineExceeded) || ctx.Err() != nil {
		return nil
	}
	return factory.availability.MarkUnavailable(ctx, model.ID, model.AvailabilityRevision)
}

var _ semantic.ImageEncoder = (*semanticSessionOwner)(nil)
var _ aiBackgroundWorker = (*semanticSessionOwner)(nil)
