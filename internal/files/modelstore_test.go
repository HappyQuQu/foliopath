package files

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
)

func managedPackageFixture() (aimodel.VerifiedPackage, aimodel.Manifest, map[string][]byte) {
	contents := map[string][]byte{
		"image_encoder.onnx": []byte("managed image graph"),
		"text_encoder.onnx":  []byte("managed text graph"),
		"tokenizer.json":     []byte("managed tokenizer"),
	}
	manifest := aimodel.Manifest{
		FormatVersion: 1,
		PackageID:     "semantic-managed-v1",
		Purpose:       aimodel.PurposeSemanticImageText,
		Version:       "1.0.0",
		Architecture:  "portable-onnx",
		LicenseID:     "Apache-2.0",
	}
	var total int64
	roles := map[string]string{"image_encoder.onnx": "image_encoder", "text_encoder.onnx": "text_encoder", "tokenizer.json": "tokenizer"}
	for _, name := range []string{"image_encoder.onnx", "text_encoder.onnx", "tokenizer.json"} {
		content := contents[name]
		digest := sha256.Sum256(content)
		manifest.Files = append(manifest.Files, aimodel.ManifestFile{
			Name: name, Size: int64(len(content)), SHA256: hex.EncodeToString(digest[:]), Role: roles[name],
		})
		total += int64(len(content))
	}
	model := aimodel.VerifiedPackage{
		PackageID:       manifest.PackageID,
		Purpose:         manifest.Purpose,
		Version:         manifest.Version,
		Architecture:    "arm64",
		ContentHash:     "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
		LicenseID:       manifest.LicenseID,
		PackageSizeByte: total,
	}
	return model, manifest, contents
}

func allowManagedPublishForTest(store *ManagedModelStore) {
	store.spaceProbe = func(string) (int64, int64, error) { return 2 << 30, 2 << 30, nil }
}

func TestManagedModelStorePublishesAndRevalidatesIdempotentPackage(t *testing.T) {
	root := filepath.Join(t.TempDir(), "models")
	store, err := NewManagedModelStore(root, 1<<30)
	if err != nil {
		t.Fatal(err)
	}
	allowManagedPublishForTest(store)
	model, manifest, contents := managedPackageFixture()
	opener := func(_ context.Context, name string) (io.ReadCloser, int64, error) {
		content, exists := contents[name]
		if !exists {
			return nil, 0, fs.ErrNotExist
		}
		return io.NopCloser(bytes.NewReader(content)), int64(len(content)), nil
	}
	identity, err := store.PublishModelPackage(context.Background(), model, manifest, opener)
	if err != nil || identity != "managed:"+model.ContentHash {
		t.Fatalf("publish = %q, %v", identity, err)
	}
	identity, err = store.PublishModelPackage(context.Background(), model, manifest, opener)
	if err != nil || identity != "managed:"+model.ContentHash {
		t.Fatalf("repeat = %q, %v", identity, err)
	}
	finalFile := filepath.Join(root, model.ContentHash+".foliomodel", "tokenizer.json")
	if err := os.WriteFile(finalFile, []byte("corrupt"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.PublishModelPackage(context.Background(), model, manifest, opener); !errors.Is(err, aimodel.ErrModelIncompatible) {
		t.Fatalf("corrupt existing package error = %v", err)
	}
}

func TestManagedModelActivationSourceRevalidatesIdentityAndFiles(t *testing.T) {
	root := filepath.Join(t.TempDir(), "models")
	store, err := NewManagedModelStore(root, 1<<30)
	if err != nil {
		t.Fatal(err)
	}
	allowManagedPublishForTest(store)
	verified, manifest, contents := managedPackageFixture()
	opener := func(_ context.Context, name string) (io.ReadCloser, int64, error) {
		content := contents[name]
		return io.NopCloser(bytes.NewReader(content)), int64(len(content)), nil
	}
	identity, err := store.PublishModelPackage(context.Background(), verified, manifest, opener)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	model := aimodel.Model{ID: "aim_managed_activation", Package: verified, StorageMode: aimodel.StorageManaged,
		State: aimodel.StateAvailable, SourceIdentity: identity, AvailabilityRevision: 1, CreatedAt: now, UpdatedAt: now}
	if err := store.ValidateManagedModelPackage(context.Background(), model, manifest); err != nil {
		t.Fatalf("validate = %v", err)
	}
	reader, size, err := store.OpenManagedModelPackageFile(context.Background(), model, "image_encoder.onnx")
	if err != nil || size != int64(len(contents["image_encoder.onnx"])) {
		t.Fatalf("open = %d, %v", size, err)
	}
	_ = reader.Close()
	wrong := model
	wrong.SourceIdentity = "managed:" + strings.Repeat("f", 64)
	if _, _, err := store.OpenManagedModelPackageFile(context.Background(), wrong, "image_encoder.onnx"); !errors.Is(err, aimodel.ErrInvalidModel) {
		t.Fatalf("wrong identity error = %v", err)
	}
	if _, _, err := store.OpenManagedModelPackageFile(context.Background(), model, "../image_encoder.onnx"); !errors.Is(err, aimodel.ErrInvalidModel) {
		t.Fatalf("traversal error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, verified.ContentHash+".foliomodel", "text_encoder.onnx"), []byte("tampered"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := store.ValidateManagedModelPackage(context.Background(), model, manifest); !errors.Is(err, aimodel.ErrModelIncompatible) {
		t.Fatalf("tampered validation error = %v", err)
	}
}

func TestManagedModelStoreReconcileRemovesOnlyStaging(t *testing.T) {
	root := filepath.Join(t.TempDir(), "models")
	store, err := NewManagedModelStore(root, 1<<30)
	if err != nil {
		t.Fatal(err)
	}
	staging := filepath.Join(root, ".partial-interrupted")
	if err := os.Mkdir(staging, 0o700); err != nil {
		t.Fatal(err)
	}
	unknown := filepath.Join(root, "operator-note.txt")
	if err := os.WriteFile(unknown, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	final := filepath.Join(root, strings.Repeat("a", 64)+".foliomodel")
	if err := os.Mkdir(final, 0o700); err != nil {
		t.Fatal(err)
	}
	report, err := store.Reconcile(context.Background())
	if err != nil || report.RemovedStaging != 1 || report.KnownFinals != 1 || report.UnknownEntries != 1 {
		t.Fatalf("reconcile = %#v, %v", report, err)
	}
	if len(report.FinalContentHashes) != 1 || report.FinalContentHashes[0] != strings.Repeat("a", 64) {
		t.Fatalf("reconciled final hashes = %v", report.FinalContentHashes)
	}
	if _, err := os.Stat(staging); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("staging still exists: %v", err)
	}
	if _, err := os.Stat(unknown); err != nil {
		t.Fatalf("unknown entry removed: %v", err)
	}
	if _, err := os.Stat(final); err != nil {
		t.Fatalf("final entry removed: %v", err)
	}
}
