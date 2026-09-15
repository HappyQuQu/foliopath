//go:build linux && cgo && onnxruntime && sentencepiece && inferencecandidate

package integration_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
	"github.com/HappyQuQu/foliopath/internal/files"
	"github.com/HappyQuQu/foliopath/internal/inference/onnx"
	"github.com/HappyQuQu/foliopath/internal/store/sqlite"
)

// Explicit candidate evidence, not release admission. All writes go to t.TempDir;
// only pinned model artifacts are read, never a real media library or live database.
func TestSemanticCandidateManagedInstallAndNativeActivation(t *testing.T) {
	directory := os.Getenv("FOLIOPATH_SIGLIP_CANDIDATE_DIR")
	if directory == "" {
		t.Skip("set FOLIOPATH_SIGLIP_CANDIDATE_DIR to converted candidate artifacts")
	}
	manifest := aimodel.Manifest{
		FormatVersion: aimodel.SemanticFormatVersion, PackageID: "siglip-isolated-candidate",
		Purpose: aimodel.PurposeSemanticImageText, Version: "candidate-20260910", Architecture: "portable-onnx", LicenseID: "Apache-2.0",
		Contracts: &aimodel.PackageContracts{
			ImagePreprocess: aimodel.SemanticImagePreprocessContract, TextCanonical: aimodel.SemanticTextCanonicalContract,
			Tokenizer: aimodel.SemanticTokenizerContract, EmbeddingAndStorage: aimodel.SemanticEmbeddingContract,
		},
		Files: []aimodel.ManifestFile{
			{Name: "image_encoder.onnx", Role: "image_encoder", Size: 371674312, SHA256: "941bc2649234f3dbbc4ae3c77059bb9969f2aa47ca57d6b7a4e081bb05157a2e"},
			{Name: "text_encoder.onnx", Role: "text_encoder", Size: 441295454, SHA256: "0b9cdf435028e23ddee07c0943c881ab3b017203a409c9918742f63a866ed91f"},
			{Name: "spiece.model", Role: "sentencepiece_model", Size: 798330, SHA256: "1e5036bed065526c3c212dfbe288752391797c4bb1a284aa18c9a0b23fcaf8ec"},
		},
	}
	encoded, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(encoded)
	contentHash := hex.EncodeToString(hash[:])
	catalog, err := aimodel.NewCatalog([]aimodel.CatalogEntry{{Manifest: manifest, ContentHash: contentHash, RuntimeArchitectures: []string{"amd64"}}})
	if err != nil {
		t.Fatal(err)
	}
	verified, _, found := catalog.PackageByContentHash(contentHash, "amd64")
	if !found {
		t.Fatal("candidate catalog entry missing")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	databasePath := filepath.Join(t.TempDir(), "candidate.db")
	store, err := sqlite.Open(ctx, databasePath, sqlite.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()
	models, err := aimodel.NewService(store, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	managed, err := files.NewManagedModelStore(filepath.Join(t.TempDir(), "models"), 0)
	if err != nil {
		t.Fatal(err)
	}
	installer, err := aimodel.NewInstaller(models, candidateArtifactSource{directory, manifest}, managed)
	if err != nil {
		t.Fatal(err)
	}
	candidate := aimodel.Candidate{Package: verified, Manifest: manifest, Compatibility: "compatible", SourceIdentity: "isolated-candidate"}
	model, created, err := installer.Install(ctx, candidate, aimodel.StorageManaged)
	if err != nil || !created {
		t.Fatalf("managed install: created=%v error=%v", created, err)
	}
	again, created, err := installer.Install(ctx, candidate, aimodel.StorageManaged)
	if err != nil || created || again.ID != model.ID {
		t.Fatalf("idempotent install: created=%v error=%v", created, err)
	}
	source := candidateManagedActivation{managed}
	availability, err := aimodel.NewAvailabilityService(models, catalog, source)
	if err != nil {
		t.Fatal(err)
	}
	operations, err := aimodel.NewOperationService(store, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	wake := downloadWake{make(chan struct{}, 1)}
	admission, err := aimodel.NewActivationAdmissionService(store, wake, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	worker, err := aimodel.NewActivationWorker(store, models, operations, catalog, source, onnx.New(), availability, wake.channel, 10*time.Millisecond, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	runCtx, stop := context.WithCancel(ctx)
	done := make(chan error, 1)
	go func() { done <- worker.Run(runCtx) }()
	defer func() {
		stop()
		if err := <-done; err != nil {
			t.Error(err)
		}
	}()
	result, err := admission.StartActivation(ctx, model, "isolated-native-activation")
	if err != nil {
		t.Fatal(err)
	}
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	for {
		operation, err := operations.Get(ctx, result.Operation.ID)
		if err != nil {
			t.Fatal(err)
		}
		if operation.State == aimodel.OperationSucceeded {
			break
		}
		if operation.State == aimodel.OperationFailed || operation.State == aimodel.OperationCancelled {
			t.Fatalf("activation failed: %+v", operation)
		}
		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case <-ticker.C:
		}
	}
	replay, found, err := admission.ReplayActivation(ctx, model.ID, model.AvailabilityRevision, "isolated-native-activation")
	if err != nil || !found || !replay.Replayed || replay.Operation.ID != result.Operation.ID {
		t.Fatal("activation replay failed", err)
	}
	snapshot, err := models.List(ctx)
	if err != nil || snapshot.ActiveModelID != model.ID {
		t.Fatal("active model commit missing", err)
	}
	// Reopen a separate read of the persisted state while the worker remains idle.
	reopened, err := sqlite.Open(ctx, databasePath, sqlite.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	persisted, err := reopened.ListAIModels(ctx)
	if err != nil || persisted.ActiveModelID != model.ID {
		t.Fatal("active model did not persist", err)
	}
	t.Log("verified managed install, idempotent reinstall, native activation worker, SQLite active-model commit and reopen")
}

type candidateArtifactSource struct {
	directory string
	manifest  aimodel.Manifest
}

func (s candidateArtifactSource) OpenModelPackageFile(_ context.Context, _ string, name string) (io.ReadCloser, int64, error) {
	for _, file := range s.manifest.Files {
		if name == file.Name {
			r, err := os.Open(filepath.Join(s.directory, name))
			return r, file.Size, err
		}
	}
	return nil, 0, errors.New("unexpected candidate artifact")
}
func (candidateArtifactSource) ValidateDirectModelSource(context.Context, string) error {
	return errors.New("direct mode forbidden for candidate test")
}

type candidateManagedActivation struct{ store *files.ManagedModelStore }

func (s candidateManagedActivation) ValidateActivationSource(ctx context.Context, m aimodel.Model, v aimodel.Manifest) error {
	return s.store.ValidateManagedModelPackage(ctx, m, v)
}
func (s candidateManagedActivation) OpenActivationModelFile(ctx context.Context, m aimodel.Model, name string) (aimodel.RuntimeModelFile, error) {
	return s.store.OpenManagedRuntimeModelFile(ctx, m, name)
}
