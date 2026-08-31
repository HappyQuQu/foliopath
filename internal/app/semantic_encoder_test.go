package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
	"github.com/HappyQuQu/foliopath/internal/inference/onnx"
	"github.com/HappyQuQu/foliopath/internal/semantic"
)

type semanticSessionFactoryStub struct {
	sessions []*semanticImageSessionStub
	opened   []string
}

func (*semanticSessionFactoryStub) ValidateSemanticImageSession(context.Context, string) error {
	return nil
}

func (factory *semanticSessionFactoryStub) OpenSemanticImageSession(_ context.Context, generationID string) (onnx.ImageSession, error) {
	index := len(factory.opened)
	factory.opened = append(factory.opened, generationID)
	if index >= len(factory.sessions) {
		return nil, errors.New("unexpected session open")
	}
	return factory.sessions[index], nil
}

type semanticImageSessionStub struct {
	outputs int
	closed  int
	err     error
}

func (session *semanticImageSessionStub) Encode(context.Context, []float32) ([]float32, error) {
	session.outputs++
	if session.err != nil {
		return nil, session.err
	}
	return []float32{3, 4}, nil
}

func (session *semanticImageSessionStub) Close() error { session.closed++; return nil }

func TestSemanticSessionOwnerReusesOneGenerationAndClosesBeforeSwitch(t *testing.T) {
	first, second := &semanticImageSessionStub{}, &semanticImageSessionStub{}
	factory := &semanticSessionFactoryStub{sessions: []*semanticImageSessionStub{first, second}}
	now := time.Date(2026, 8, 28, 7, 0, 0, 0, time.UTC)
	owner, err := newSemanticSessionOwner(factory, time.Second, time.Minute, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	if _, err := owner.EncodeSemanticImage(context.Background(), "generation-a", []float32{1}); err != nil {
		t.Fatal(err)
	}
	if _, err := owner.EncodeSemanticImage(context.Background(), "generation-a", []float32{2}); err != nil {
		t.Fatal(err)
	}
	if len(factory.opened) != 1 || first.outputs != 2 || first.closed != 0 {
		t.Fatalf("reused state = opened %#v, outputs %d, closed %d", factory.opened, first.outputs, first.closed)
	}
	if _, err := owner.EncodeSemanticImage(context.Background(), "generation-b", []float32{3}); err != nil {
		t.Fatal(err)
	}
	if len(factory.opened) != 2 || first.closed != 1 || second.outputs != 1 {
		t.Fatalf("switch state = opened %#v, first closed %d, second outputs %d", factory.opened, first.closed, second.outputs)
	}
	if err := owner.Close(); err != nil || second.closed != 1 {
		t.Fatalf("close = %v, count %d", err, second.closed)
	}
	if resources := owner.Resources(); resources != (semanticSessionResources{
		TotalLoads: 2, TotalRuns: 3, TotalUnloads: 2,
	}) {
		t.Fatalf("resources = %#v", resources)
	}
}

type blockingSemanticImageSessionStub struct {
	started chan struct{}
	release chan struct{}
}

func (session *blockingSemanticImageSessionStub) Encode(context.Context, []float32) ([]float32, error) {
	close(session.started)
	<-session.release
	return []float32{3, 4}, nil
}

func (*blockingSemanticImageSessionStub) Close() error { return nil }

type blockingSemanticSessionFactoryStub struct {
	session *blockingSemanticImageSessionStub
}

func (*blockingSemanticSessionFactoryStub) ValidateSemanticImageSession(context.Context, string) error {
	return nil
}

func (factory *blockingSemanticSessionFactoryStub) OpenSemanticImageSession(context.Context, string) (onnx.ImageSession, error) {
	return factory.session, nil
}

func TestSemanticSessionOwnerAccountsForResidentAndActiveResources(t *testing.T) {
	session := &blockingSemanticImageSessionStub{started: make(chan struct{}), release: make(chan struct{})}
	owner, err := newSemanticSessionOwner(&blockingSemanticSessionFactoryStub{session: session}, time.Second, time.Minute, nil)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		_, encodeErr := owner.EncodeSemanticImage(context.Background(), "generation-a", []float32{1})
		done <- encodeErr
	}()
	<-session.started
	if resources := owner.Resources(); resources != (semanticSessionResources{
		ResidentSessions: 1, ActiveRuns: 1, TotalLoads: 1, TotalRuns: 1,
	}) {
		t.Fatalf("running resources = %#v", resources)
	}
	close(session.release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if resources := owner.Resources(); resources != (semanticSessionResources{
		ResidentSessions: 1, TotalLoads: 1, TotalRuns: 1,
	}) {
		t.Fatalf("idle resources = %#v", resources)
	}
	if err := owner.Close(); err != nil {
		t.Fatal(err)
	}
	if resources := owner.Resources(); resources != (semanticSessionResources{
		TotalLoads: 1, TotalRuns: 1, TotalUnloads: 1,
	}) {
		t.Fatalf("closed resources = %#v", resources)
	}
}

func TestSemanticSessionOwnerDropsFaultedSession(t *testing.T) {
	faulted, recovered := &semanticImageSessionStub{err: errors.New("runtime failed")}, &semanticImageSessionStub{}
	factory := &semanticSessionFactoryStub{sessions: []*semanticImageSessionStub{faulted, recovered}}
	owner, err := newSemanticSessionOwner(factory, time.Second, time.Minute, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := owner.EncodeSemanticImage(context.Background(), "generation-a", []float32{1}); err == nil || faulted.closed != 1 {
		t.Fatalf("fault = %v, closed %d", err, faulted.closed)
	}
	if _, err := owner.EncodeSemanticImage(context.Background(), "generation-a", []float32{1}); err != nil || len(factory.opened) != 2 {
		t.Fatalf("reopen = %v, opened %#v", err, factory.opened)
	}
}

type semanticGenerationRuntimeStub struct {
	generation semantic.GenerationRuntime
}

func (stub semanticGenerationRuntimeStub) GetSemanticGenerationRuntime(context.Context, string) (semantic.GenerationRuntime, error) {
	return stub.generation, nil
}

type semanticModelRepositoryStub struct {
	model aimodel.Model
}

func (stub *semanticModelRepositoryStub) ListAIModels(context.Context) (aimodel.Snapshot, error) {
	return aimodel.Snapshot{Items: []aimodel.Model{stub.model}, Revision: 1}, nil
}
func (stub *semanticModelRepositoryStub) GetAIModel(context.Context, string) (aimodel.Model, error) {
	return stub.model, nil
}
func (stub *semanticModelRepositoryStub) RegisterAIModel(context.Context, aimodel.Model) (aimodel.Model, bool, error) {
	return aimodel.Model{}, false, errors.New("not used")
}
func (stub *semanticModelRepositoryStub) SetAIModelAvailability(context.Context, string, int64, aimodel.State, time.Time) (aimodel.Model, error) {
	return aimodel.Model{}, errors.New("not used")
}

type semanticActivationSourceStub struct{ err error }

func (stub semanticActivationSourceStub) ValidateActivationSource(context.Context, aimodel.Model, aimodel.Manifest) error {
	return stub.err
}
func (semanticActivationSourceStub) OpenActivationModelFile(context.Context, aimodel.Model, string) (aimodel.RuntimeModelFile, error) {
	return nil, errors.New("not used")
}

type semanticAvailabilityMarkerStub struct {
	modelID  string
	revision int64
}

func (stub *semanticAvailabilityMarkerStub) MarkUnavailable(_ context.Context, modelID string, revision int64) error {
	stub.modelID, stub.revision = modelID, revision
	return nil
}

type semanticProductionRuntimeStub struct{}

func (semanticProductionRuntimeStub) OpenImageSession(context.Context, aimodel.Manifest, aimodel.RuntimeFileOpener) (onnx.ImageSession, error) {
	return nil, errors.New("not used")
}

func TestSemanticProductionColdLoadMarksFailedSourceUnavailable(t *testing.T) {
	manifest := aimodel.Manifest{
		FormatVersion: 1, PackageID: "semantic-runtime-v1", Purpose: aimodel.PurposeSemanticImageText,
		Version: "1.0.0", Architecture: "portable-onnx", LicenseID: "Apache-2.0",
		Files: []aimodel.ManifestFile{
			{Name: "image.onnx", Size: 1, SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Role: "image_encoder"},
			{Name: "text.onnx", Size: 1, SHA256: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", Role: "text_encoder"},
			{Name: "tokenizer.json", Size: 1, SHA256: "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", Role: "tokenizer"},
		},
	}
	catalog, err := aimodel.NewCatalog([]aimodel.CatalogEntry{{
		Manifest: manifest, ContentHash: "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
		RuntimeArchitectures: []string{"arm64"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 28, 20, 0, 0, 0, time.UTC)
	model := aimodel.Model{
		ID: "aim_semantic_runtime", StorageMode: aimodel.StorageDirect, State: aimodel.StateAvailable,
		SourceIdentity: "source:runtime", AvailabilityRevision: 7, CreatedAt: now, UpdatedAt: now,
		Package: aimodel.VerifiedPackage{
			PackageID: manifest.PackageID, Purpose: aimodel.PurposeSemanticImageText, Version: manifest.Version,
			Architecture: "arm64", ContentHash: "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
			LicenseID: manifest.LicenseID, PackageSizeByte: 3,
		},
	}
	models, err := aimodel.NewService(&semanticModelRepositoryStub{model: model}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	marker := &semanticAvailabilityMarkerStub{}
	factory := semanticProductionSessionFactory{
		generations: semanticGenerationRuntimeStub{generation: semantic.GenerationRuntime{
			ID: "aig_semantic_runtime", ModelID: model.ID, EmbeddingDimension: int(onnx.EmbeddingDimension), State: "active",
		}},
		models: models, catalog: catalog,
		source:       semanticActivationSourceStub{err: aimodel.ErrModelSourceUnavailable},
		availability: marker, runtime: semanticProductionRuntimeStub{},
	}
	if _, err := factory.OpenSemanticImageSession(context.Background(), "aig_semantic_runtime"); !errors.Is(err, aimodel.ErrModelSourceUnavailable) {
		t.Fatalf("open error = %v", err)
	}
	if marker.modelID != model.ID || marker.revision != model.AvailabilityRevision {
		t.Fatalf("availability mark = %q r%d", marker.modelID, marker.revision)
	}

	marker.modelID, marker.revision = "", 0
	factory.source = semanticActivationSourceStub{err: context.Canceled}
	if _, err := factory.OpenSemanticImageSession(context.Background(), "aig_semantic_runtime"); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel error = %v", err)
	}
	if marker.modelID != "" || marker.revision != 0 {
		t.Fatalf("cancellation marked unavailable = %q r%d", marker.modelID, marker.revision)
	}
}
