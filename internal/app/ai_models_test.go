package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
)

func TestAIModelSourceUnavailableDoesNotPreventApplicationStartup(t *testing.T) {
	source, lifecycle := newAIModelSource(filepath.Join(t.TempDir(), "missing-models"))
	if err := lifecycle.start(context.Background()); err != nil {
		t.Fatalf("optional source start = %v", err)
	}
	defer lifecycle.stop(context.Background())
	if _, _, err := source.ScanModelPackages(context.Background(), 1, 4, 1024); !errors.Is(err, aimodel.ErrModelSourceUnavailable) {
		t.Fatalf("scan error = %v", err)
	}
}

func TestManagedAIModelStoreIsCreatedOnlyOnLifecycleStart(t *testing.T) {
	root := filepath.Join(t.TempDir(), "managed", "models")
	store, lifecycle, err := newManagedAIModelStore(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Reconcile(context.Background()); !errors.Is(err, errAIRepositoryNotReady) {
		t.Fatalf("pre-start reconcile error = %v", err)
	}
	if err := lifecycle.start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer lifecycle.stop(context.Background())
	if _, err := store.Reconcile(context.Background()); err != nil {
		t.Fatalf("started reconcile = %v", err)
	}
}

type aiAvailabilityRefresherStub struct {
	called bool
	err    error
}

func (stub *aiAvailabilityRefresherStub) Refresh(context.Context) (aimodel.AvailabilitySummary, error) {
	stub.called = true
	return aimodel.AvailabilitySummary{Checked: 1}, stub.err
}

func TestAIAvailabilityLifecycleRefreshesBeforeWorkersStart(t *testing.T) {
	stub := &aiAvailabilityRefresherStub{}
	lifecycle, err := newAIAvailabilityComponent(stub)
	if err != nil {
		t.Fatal(err)
	}
	if err := lifecycle.start(context.Background()); err != nil || !stub.called {
		t.Fatalf("availability start called=%v err=%v", stub.called, err)
	}

	stub = &aiAvailabilityRefresherStub{err: errors.New("database unavailable")}
	lifecycle, err = newAIAvailabilityComponent(stub)
	if err != nil {
		t.Fatal(err)
	}
	if err := lifecycle.start(context.Background()); err == nil || !stub.called {
		t.Fatalf("availability failure called=%v err=%v", stub.called, err)
	}
}

type managedAvailabilitySource struct{ store *managedAIModelStore }

func (source managedAvailabilitySource) ValidateActivationSource(ctx context.Context, model aimodel.Model, manifest aimodel.Manifest) error {
	return source.store.ValidateManagedModelPackage(ctx, model, manifest)
}
func (source managedAvailabilitySource) OpenActivationModelFile(ctx context.Context, model aimodel.Model, name string) (aimodel.RuntimeModelFile, error) {
	return source.store.OpenManagedRuntimeModelFile(ctx, model, name)
}

type availabilityWakeStub struct{}

func (availabilityWakeStub) Wake() {}

func TestManagedModelCorruptionMarksUnavailableWithoutRetiringActiveGeneration(t *testing.T) {
	ctx := context.Background()
	dataRoot := t.TempDir()
	mediaRoot := filepath.Join(t.TempDir(), "library")
	if err := os.MkdirAll(mediaRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	mediaPath := filepath.Join(mediaRoot, "image.jpg")
	mediaBytes := []byte("original media remains immutable")
	if err := os.WriteFile(mediaPath, mediaBytes, 0o400); err != nil {
		t.Fatal(err)
	}
	mediaMTime := time.Date(2026, 8, 28, 20, 0, 0, 123456789, time.UTC)
	if err := os.Chtimes(mediaPath, mediaMTime, mediaMTime); err != nil {
		t.Fatal(err)
	}
	mediaHashBefore := sha256.Sum256(mediaBytes)
	mediaInfoBefore, err := os.Stat(mediaPath)
	if err != nil {
		t.Fatal(err)
	}

	databaseComponent, database := newDatabaseComponent(dataRoot, newReadinessState())
	if err := databaseComponent.start(ctx); err != nil {
		t.Fatal(err)
	}
	defer databaseComponent.stop(ctx)

	managedRoot := filepath.Join(dataRoot, "models")
	managed, managedComponent, err := newManagedAIModelStore(managedRoot)
	if err != nil {
		t.Fatal(err)
	}
	if err := managedComponent.start(ctx); err != nil {
		t.Fatal(err)
	}
	defer managedComponent.stop(ctx)

	contents := map[string][]byte{
		"image_encoder.onnx": []byte("image graph"),
		"text_encoder.onnx":  []byte("text graph"),
		"tokenizer.json":     []byte("tokenizer"),
	}
	manifest := aimodel.Manifest{
		FormatVersion: 1, PackageID: "semantic-corruption-v1", Purpose: aimodel.PurposeSemanticImageText,
		Version: "1.0.0", Architecture: "portable-onnx", LicenseID: "Apache-2.0",
	}
	roles := map[string]string{
		"image_encoder.onnx": "image_encoder", "text_encoder.onnx": "text_encoder", "tokenizer.json": "tokenizer",
	}
	for _, name := range []string{"image_encoder.onnx", "text_encoder.onnx", "tokenizer.json"} {
		digest := sha256.Sum256(contents[name])
		manifest.Files = append(manifest.Files, aimodel.ManifestFile{
			Name: name, Size: int64(len(contents[name])), SHA256: hex.EncodeToString(digest[:]), Role: roles[name],
		})
	}
	const contentHash = "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
	finalRoot := filepath.Join(managedRoot, contentHash+".foliomodel")
	if err := os.Mkdir(finalRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(finalRoot, "manifest.json"), manifestBytes, 0o400); err != nil {
		t.Fatal(err)
	}
	for name, content := range contents {
		if err := os.WriteFile(filepath.Join(finalRoot, name), content, 0o400); err != nil {
			t.Fatal(err)
		}
	}
	catalog, err := aimodel.NewCatalog([]aimodel.CatalogEntry{{
		Manifest: manifest, ContentHash: contentHash, RuntimeArchitectures: []string{"arm64"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 28, 21, 0, 0, 0, time.UTC)
	models, err := aimodel.NewService(database, func() time.Time { now = now.Add(time.Second); return now }, func() (string, error) {
		return "aim_corruption_test", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	reconcileReport, err := managed.Reconcile(ctx)
	if err != nil {
		t.Fatal(err)
	}
	orphans, err := aimodel.NewManagedOrphanService(models, catalog, managed, "arm64")
	if err != nil {
		t.Fatal(err)
	}
	orphanSummary, err := orphans.Reconcile(ctx, reconcileReport.FinalContentHashes, !reconcileReport.Truncated)
	if err != nil || orphanSummary.Registered != 1 {
		t.Fatalf("orphan reconcile = %#v, %v", orphanSummary, err)
	}
	registered, err := models.List(ctx)
	if err != nil || len(registered.Items) != 1 || registered.ActiveModelID != "" || registered.Items[0].Active {
		t.Fatalf("registered orphan snapshot = %#v, %v", registered, err)
	}
	model := registered.Items[0]
	activation, err := aimodel.NewActivationAdmissionService(database, availabilityWakeStub{}, func() time.Time { return now }, func() (string, error) {
		return "aio_corruption_test", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := activation.StartActivation(ctx, model, "corruption-activation-key"); err != nil {
		t.Fatal(err)
	}
	work, found, err := database.ClaimAIModelActivation(ctx, now.Add(time.Second))
	if err != nil || !found {
		t.Fatalf("claim = %#v, %v, %v", work, found, err)
	}
	activatedAt := now.Add(2 * time.Second)
	if _, err := database.CommitAIModelActivation(ctx, aimodel.ActivationCommit{
		OperationID: work.Operation.ID, ExpectedRevision: work.Operation.Revision,
		ExpectedAvailabilityRevision: work.ExpectedAvailabilityRevision,
		Generation: aimodel.Generation{
			ID: "aig_corruption_test", ModelID: model.ID,
			TransformVersion: aimodel.SemanticTransformVersion, OutputSchemaVersion: aimodel.SemanticOutputSchemaVersion,
			IndexFormatVersion: aimodel.SemanticIndexFormatVersion, EmbeddingDimension: 768,
			State: aimodel.GenerationActive, CreatedAt: activatedAt, ActivatedAt: &activatedAt, UpdatedAt: activatedAt,
		},
		UpdatedAt: activatedAt,
	}); err != nil {
		t.Fatal(err)
	}
	inspector := openDatabaseInspector(t, filepath.Join(dataRoot, databaseFilename))
	seedAt := activatedAt.UnixMilli()
	if _, err := inspector.Exec(`
		INSERT INTO libraries(id, name, name_key, root_rel_path, status, current_generation, created_at_ms, updated_at_ms)
		VALUES(101, 'AI evidence', 'ai evidence', 'ai-evidence', 'ready', 1, ?, ?);
		INSERT INTO directories(id, library_id, relative_path, name, last_seen_generation)
		VALUES(101, 101, '', '', 1);
		INSERT INTO assets(id, library_id, directory_id, relative_path, name, kind, media_format, mime_type,
			size_bytes, mtime_ns, last_seen_generation, source_fingerprint)
		VALUES(101, 101, 101, 'image.jpg', 'image.jpg', 'image', 'jpeg', 'image/jpeg', 1, 1, 1, 'v1:1:1');
		INSERT INTO semantic_embeddings(generation_id, library_id, asset_id, source_fingerprint, vector, created_at_ms)
		VALUES('aig_corruption_test', 101, 101, 'v1:1:1', x'003c', ?)
	`, seedAt, seedAt, seedAt); err != nil {
		t.Fatal(err)
	}

	availability, err := aimodel.NewAvailabilityService(models, catalog, managedAvailabilitySource{store: managed})
	if err != nil {
		t.Fatal(err)
	}
	corruptPath := filepath.Join(finalRoot, "text_encoder.onnx")
	if err := os.Chmod(corruptPath, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(corruptPath, []byte("tampered"), 0o600); err != nil {
		t.Fatal(err)
	}
	summary, err := availability.Refresh(ctx)
	if err != nil || summary.Changed != 1 || summary.Unavailable != 1 {
		t.Fatalf("corrupt refresh = %#v, %v", summary, err)
	}
	snapshot, err := models.List(ctx)
	if err != nil || snapshot.ActiveModelID != model.ID || len(snapshot.Items) != 1 || snapshot.Items[0].State != aimodel.StateUnavailable {
		t.Fatalf("unavailable snapshot = %#v, %v", snapshot, err)
	}

	var activeGenerations, retainedEmbeddings int
	if err := inspector.QueryRow(`SELECT COUNT(*) FROM semantic_generations WHERE id=? AND model_id=? AND state='active'`, "aig_corruption_test", model.ID).Scan(&activeGenerations); err != nil {
		t.Fatal(err)
	}
	if err := inspector.QueryRow(`SELECT COUNT(*) FROM semantic_embeddings WHERE generation_id=?`, "aig_corruption_test").Scan(&retainedEmbeddings); err != nil {
		t.Fatal(err)
	}
	if activeGenerations != 1 || retainedEmbeddings != 1 {
		t.Fatalf("preserved state = active generations %d embeddings %d", activeGenerations, retainedEmbeddings)
	}
	if err := inspector.Close(); err != nil {
		t.Fatal(err)
	}
	if err := databaseComponent.stop(ctx); err != nil {
		t.Fatal(err)
	}
	if err := databaseComponent.start(ctx); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(corruptPath, contents["text_encoder.onnx"], 0o600); err != nil {
		t.Fatal(err)
	}
	availability, err = aimodel.NewAvailabilityService(models, catalog, managedAvailabilitySource{store: managed})
	if err != nil {
		t.Fatal(err)
	}
	summary, err = availability.Refresh(ctx)
	if err != nil || summary.Changed != 1 || summary.Unavailable != 0 {
		t.Fatalf("recovery refresh = %#v, %v", summary, err)
	}
	recovered, err := models.Get(ctx, model.ID)
	if err != nil || recovered.State != aimodel.StateAvailable || recovered.AvailabilityRevision != 3 {
		t.Fatalf("recovered model = %#v, %v", recovered, err)
	}

	inspector = openDatabaseInspector(t, filepath.Join(dataRoot, databaseFilename))
	defer inspector.Close()
	if err := inspector.QueryRow(`SELECT COUNT(*) FROM semantic_generations WHERE id=? AND model_id=? AND state='active'`, "aig_corruption_test", model.ID).Scan(&activeGenerations); err != nil {
		t.Fatal(err)
	}
	if err := inspector.QueryRow(`SELECT COUNT(*) FROM semantic_embeddings WHERE generation_id=?`, "aig_corruption_test").Scan(&retainedEmbeddings); err != nil {
		t.Fatal(err)
	}
	if activeGenerations != 1 || retainedEmbeddings != 1 {
		t.Fatalf("recovered state = active generations %d embeddings %d", activeGenerations, retainedEmbeddings)
	}
	mediaBytesAfter, err := os.ReadFile(mediaPath)
	if err != nil {
		t.Fatal(err)
	}
	mediaHashAfter := sha256.Sum256(mediaBytesAfter)
	mediaInfoAfter, err := os.Stat(mediaPath)
	if err != nil {
		t.Fatal(err)
	}
	if mediaHashAfter != mediaHashBefore || mediaInfoAfter.Size() != mediaInfoBefore.Size() || !mediaInfoAfter.ModTime().Equal(mediaInfoBefore.ModTime()) {
		t.Fatalf("original media changed: hash=%v size=%d mtime=%s", mediaHashAfter != mediaHashBefore, mediaInfoAfter.Size(), mediaInfoAfter.ModTime())
	}
}
