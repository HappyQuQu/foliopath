package app

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
	"github.com/HappyQuQu/foliopath/internal/inference/onnx"
	"github.com/HappyQuQu/foliopath/internal/inference/sentencepiece"
	"github.com/HappyQuQu/foliopath/internal/semantic"
)

type semanticTextRuntimeSession interface {
	Encode(context.Context, string) ([]float32, error)
	Close() error
}

type semanticTextSessionFactory interface {
	ValidateSemanticTextSession(context.Context, string) error
	OpenSemanticTextSession(context.Context, string) (semanticTextRuntimeSession, error)
}

type semanticTextSessionOwner struct {
	factory semanticTextSessionFactory
	timeout time.Duration
	idle    time.Duration
	now     func() time.Time

	mu           sync.Mutex
	session      semanticTextRuntimeSession
	generationID string
	lastUsed     time.Time
}

func newSemanticTextSessionOwner(factory semanticTextSessionFactory, timeout, idle time.Duration, now func() time.Time) (*semanticTextSessionOwner, error) {
	if factory == nil {
		return nil, errors.New("semantic text session factory is required")
	}
	if timeout == 0 {
		timeout = semanticInferenceTimeout
	}
	if idle == 0 {
		idle = semanticSessionIdle
	}
	if timeout < time.Millisecond || timeout > time.Minute || idle < time.Second {
		return nil, errors.New("semantic text session timing is invalid")
	}
	if now == nil {
		now = time.Now
	}
	return &semanticTextSessionOwner{factory: factory, timeout: timeout, idle: idle, now: now}, nil
}

func (owner *semanticTextSessionOwner) EncodeSemanticText(ctx context.Context, generationID, query string) ([]float32, error) {
	if generationID == "" {
		return nil, semantic.ErrSemanticGenerationUnavailable
	}
	owner.mu.Lock()
	defer owner.mu.Unlock()
	validateCtx, validateCancel := context.WithTimeout(ctx, owner.timeout)
	validateErr := owner.factory.ValidateSemanticTextSession(validateCtx, generationID)
	validateCancel()
	if validateErr != nil {
		owner.closeLocked()
		return nil, errors.Join(semantic.ErrSemanticGenerationUnavailable, validateErr)
	}
	if owner.session == nil || owner.generationID != generationID {
		owner.closeLocked()
		loadCtx, cancel := context.WithTimeout(ctx, owner.timeout)
		session, err := owner.factory.OpenSemanticTextSession(loadCtx, generationID)
		cancel()
		if err != nil {
			return nil, errors.Join(semantic.ErrSemanticGenerationUnavailable, err)
		}
		owner.session, owner.generationID = session, generationID
	}
	runCtx, cancel := context.WithTimeout(ctx, owner.timeout)
	output, err := owner.session.Encode(runCtx, query)
	cancel()
	owner.lastUsed = owner.now().UTC()
	if err != nil {
		if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
			owner.closeLocked()
		}
		return nil, err
	}
	normalized, err := semantic.NormalizeEmbedding(output, semantic.SigLIPEmbeddingDimension)
	if err != nil {
		owner.closeLocked()
		return nil, errors.Join(semantic.ErrSemanticGenerationUnavailable, err)
	}
	return normalized, nil
}

func (owner *semanticTextSessionOwner) Run(ctx context.Context) error {
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

func (owner *semanticTextSessionOwner) Close() error {
	owner.mu.Lock()
	defer owner.mu.Unlock()
	return owner.closeLocked()
}

func (owner *semanticTextSessionOwner) closeLocked() error {
	session := owner.session
	owner.session, owner.generationID, owner.lastUsed = nil, "", time.Time{}
	if session != nil {
		return session.Close()
	}
	return nil
}

type combinedSemanticTextSession struct {
	tokenizer sentencepiece.Session
	encoder   onnx.TextSession
}

func (session *combinedSemanticTextSession) Encode(ctx context.Context, query string) ([]float32, error) {
	ids, err := session.tokenizer.Encode(ctx, query)
	if err != nil {
		return nil, err
	}
	return session.encoder.EncodeText(ctx, ids[:])
}

func (session *combinedSemanticTextSession) Close() error {
	return errors.Join(session.encoder.Close(), session.tokenizer.Close())
}

type semanticProductionTextSessionFactory struct {
	generations  semantic.GenerationRuntimeRepository
	models       *aimodel.Service
	catalog      *aimodel.Catalog
	source       aimodel.ActivationPackageSource
	availability aimodel.AvailabilityMarker
	onnxRuntime  interface {
		OpenTextSession(context.Context, aimodel.Manifest, aimodel.RuntimeFileOpener) (onnx.TextSession, error)
	}
	tokenizerRuntime interface {
		Open(context.Context, aimodel.RuntimeModelFile) (sentencepiece.Session, error)
	}
}

func (factory semanticProductionTextSessionFactory) ValidateSemanticTextSession(ctx context.Context, generationID string) error {
	_, _, manifest, err := factory.resolve(ctx, generationID)
	if err != nil {
		return err
	}
	if manifest.FormatVersion != aimodel.SemanticFormatVersion || manifest.Contracts == nil ||
		manifest.Contracts.Tokenizer != aimodel.SemanticTokenizerContract {
		return aimodel.ErrModelIncompatible
	}
	return nil
}

func (factory semanticProductionTextSessionFactory) OpenSemanticTextSession(ctx context.Context, generationID string) (semanticTextRuntimeSession, error) {
	_, model, manifest, err := factory.resolve(ctx, generationID)
	if err != nil {
		return nil, err
	}
	if err := factory.ValidateSemanticTextSession(ctx, generationID); err != nil {
		return nil, err
	}
	if err := factory.source.ValidateActivationSource(ctx, model, manifest); err != nil {
		return nil, errors.Join(err, factory.markUnavailableAfterValidationFailure(ctx, model, err))
	}
	tokenizerManifestFile, err := semanticTokenizerManifestFile(manifest)
	if err != nil {
		return nil, err
	}
	tokenizerFile, err := factory.source.OpenActivationModelFile(ctx, model, tokenizerManifestFile.Name)
	if err != nil {
		return nil, errors.Join(err, factory.markUnavailableAfterValidationFailure(ctx, model, err))
	}
	tokenizer, err := factory.tokenizerRuntime.Open(ctx, tokenizerFile)
	if err != nil {
		return nil, errors.Join(err, factory.markUnavailableAfterValidationFailure(ctx, model, err))
	}
	encoder, err := factory.onnxRuntime.OpenTextSession(ctx, manifest, func(openCtx context.Context, name string) (aimodel.RuntimeModelFile, error) {
		return factory.source.OpenActivationModelFile(openCtx, model, name)
	})
	if err != nil {
		_ = tokenizer.Close()
		return nil, errors.Join(err, factory.markUnavailableAfterValidationFailure(ctx, model, err))
	}
	return &combinedSemanticTextSession{tokenizer: tokenizer, encoder: encoder}, nil
}

func semanticTokenizerManifestFile(manifest aimodel.Manifest) (aimodel.ManifestFile, error) {
	var tokenizerFile aimodel.ManifestFile
	found := false
	for _, file := range manifest.Files {
		if file.Role != "sentencepiece_model" {
			continue
		}
		if found {
			return aimodel.ManifestFile{}, aimodel.ErrModelIncompatible
		}
		tokenizerFile, found = file, true
	}
	if !found {
		return aimodel.ManifestFile{}, aimodel.ErrModelIncompatible
	}
	return tokenizerFile, nil
}

func (factory semanticProductionTextSessionFactory) resolve(ctx context.Context, generationID string) (semantic.GenerationRuntime, aimodel.Model, aimodel.Manifest, error) {
	if factory.generations == nil || factory.models == nil || factory.catalog == nil || factory.source == nil ||
		factory.availability == nil || factory.onnxRuntime == nil || factory.tokenizerRuntime == nil {
		return semantic.GenerationRuntime{}, aimodel.Model{}, aimodel.Manifest{}, semantic.ErrSemanticGenerationUnavailable
	}
	generation, err := factory.generations.GetSemanticGenerationRuntime(ctx, generationID)
	if err != nil || generation.State != "active" || generation.EmbeddingDimension != semantic.SigLIPEmbeddingDimension {
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
	return generation, model, manifest, nil
}

func (factory semanticProductionTextSessionFactory) markUnavailableAfterValidationFailure(ctx context.Context, model aimodel.Model, validationErr error) error {
	if errors.Is(validationErr, context.Canceled) || errors.Is(validationErr, context.DeadlineExceeded) || ctx.Err() != nil ||
		errors.Is(validationErr, sentencepiece.ErrTokenizerUnavailable) || errors.Is(validationErr, aimodel.ErrInferenceRuntimeUnavailable) {
		return nil
	}
	return factory.availability.MarkUnavailable(ctx, model.ID, model.AvailabilityRevision)
}

var _ semantic.TextEncoder = (*semanticTextSessionOwner)(nil)
var _ aiBackgroundWorker = (*semanticTextSessionOwner)(nil)
var _ semanticTextRuntimeSession = (*combinedSemanticTextSession)(nil)
