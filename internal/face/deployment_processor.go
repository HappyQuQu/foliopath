package face

import (
	"context"
	"errors"
	"time"
)

// DeploymentGenerationRuntime is read atomically from the generation and its
// model. Fingerprint is empty for legacy or non-service models.
type DeploymentGenerationRuntime struct {
	ID, Fingerprint, State string
	Dimension              int
	Available              bool
}

type DeploymentGenerationRepository interface {
	GetFaceDeploymentGenerationRuntime(context.Context, string) (DeploymentGenerationRuntime, error)
}

// DeploymentJobProcessor binds each claim to the configured deployment. The
// ordinary JobProcessor remains the sole owner of checkpoints and job states.
type DeploymentJobProcessor struct {
	queue      JobQueue
	catalog    JobCatalog
	analyzer   JobAnalyzer
	clusters   ClusterRebuilder
	repository DeploymentGenerationRepository
	identity   DeploymentIdentity
	now        func() time.Time
	pageSize   int
}

func NewDeploymentJobProcessor(queue JobQueue, catalog JobCatalog, analyzer JobAnalyzer, clusters ClusterRebuilder, repository DeploymentGenerationRepository, identity DeploymentIdentity, now func() time.Time, pageSize int) (*DeploymentJobProcessor, error) {
	if repository == nil {
		return nil, ErrInvalidFaceJob
	}
	if _, err := identity.Fingerprint(); err != nil {
		return nil, err
	}
	if now == nil {
		now = time.Now
	}
	p := &DeploymentJobProcessor{queue, catalog, analyzer, clusters, repository, identity, now, pageSize}
	if _, err := NewJobProcessor(queue, catalog, analyzer, clusters, p.profile(), now, pageSize); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *DeploymentJobProcessor) profile() ClusterProfile {
	return ClusterProfile{CoreSimilarity: float32(p.identity.Threshold.CoreSimilarity), EdgeSimilarity: float32(p.identity.Threshold.EdgeSimilarity), MinCoreSize: p.identity.Threshold.MinCoreSize}
}

func (p *DeploymentJobProcessor) Process(ctx context.Context, job AnalysisJob) error {
	bound := &deploymentJobBinding{JobCatalog: p.catalog, analyzer: p.analyzer, repository: p.repository, identity: p.identity, generationID: job.GenerationID}
	processor, err := NewJobProcessor(p.queue, bound, bound, p.clusters, p.profile(), p.now, p.pageSize)
	if err != nil {
		return err
	}
	return processor.Process(ctx, job)
}

func (p *DeploymentJobProcessor) BackgroundAdmissionCancelled(ctx context.Context, job AnalysisJob) {
	finishFaceAdmissionCancellation(ctx, p.queue, job, p.now)
}

type deploymentJobBinding struct {
	JobCatalog
	analyzer     JobAnalyzer
	repository   DeploymentGenerationRepository
	identity     DeploymentIdentity
	generationID string
}

func (b *deploymentJobBinding) admit(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	current, err := b.repository.GetFaceDeploymentGenerationRuntime(ctx, b.generationID)
	if err != nil {
		return errors.Join(ErrRuntimeUnavailable, err)
	}
	fingerprint, err := b.identity.Fingerprint()
	if err != nil || current.ID != b.generationID || current.Fingerprint != fingerprint || current.State != "active" || !current.Available || current.Dimension != b.identity.Dimension {
		return ErrRuntimeUnavailable
	}
	return nil
}

func (b *deploymentJobBinding) FaceJobLibraryState(ctx context.Context, libraryID int64, generationID string) (JobLibraryState, error) {
	state, err := b.JobCatalog.FaceJobLibraryState(ctx, libraryID, generationID)
	if err != nil || state != JobLibraryReady && state != JobLibraryScanning {
		return state, err
	}
	if generationID != b.generationID {
		return JobLibraryModelUnavailable, nil
	}
	if err := b.admit(ctx); err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return JobLibraryModelUnavailable, nil
	}
	return state, nil
}

func (b *deploymentJobBinding) Analyze(ctx context.Context, libraryID, assetID int64, fingerprint string) ([]Observation, error) {
	if err := b.admit(ctx); err != nil {
		return nil, err
	}
	items, err := b.analyzer.Analyze(ctx, libraryID, assetID, fingerprint)
	if err != nil {
		return nil, err
	}
	// Discard even an empty result when a switch/offline transition occurred
	// during inference. The repository still owns final transactional fencing.
	if err := b.admit(ctx); err != nil {
		return nil, err
	}
	for _, item := range items {
		if len(item.Embedding) != b.identity.Dimension {
			return nil, ErrInvalidOutput
		}
	}
	return items, nil
}
