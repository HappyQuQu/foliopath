package files

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
)

func TestModelSourceHashesOnlyBoundedRegularPackageFiles(t *testing.T) {
	rootPath := t.TempDir()
	packagePath := filepath.Join(rootPath, "local.foliomodel")
	if err := os.Mkdir(packagePath, 0o755); err != nil {
		t.Fatal(err)
	}
	contents := map[string][]byte{
		"image_encoder.onnx": []byte("image graph"),
		"text_encoder.onnx":  []byte("text graph"),
		"tokenizer.json":     []byte("tokenizer"),
	}
	manifest := aimodel.Manifest{
		FormatVersion: 1,
		PackageID:     "semantic-test-v1",
		Purpose:       aimodel.PurposeSemanticImageText,
		Version:       "1.0.0",
		Architecture:  "portable-onnx",
		LicenseID:     "Apache-2.0",
	}
	roles := map[string]string{"image_encoder.onnx": "image_encoder", "text_encoder.onnx": "text_encoder", "tokenizer.json": "tokenizer"}
	for _, name := range []string{"image_encoder.onnx", "text_encoder.onnx", "tokenizer.json"} {
		content := contents[name]
		digest := sha256.Sum256(content)
		manifest.Files = append(manifest.Files, aimodel.ManifestFile{Name: name, Size: int64(len(content)), SHA256: hex.EncodeToString(digest[:]), Role: roles[name]})
		if err := os.WriteFile(filepath.Join(packagePath, name), content, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	encoded, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(packagePath, "manifest.json"), encoded, 0o644); err != nil {
		t.Fatal(err)
	}
	root, err := OpenRoot(rootPath)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	source, err := NewModelSource(root)
	if err != nil {
		t.Fatal(err)
	}
	items, truncated, err := source.ScanModelPackages(context.Background(), 64, 16, 4<<30)
	if err != nil || truncated || len(items) != 1 || items[0].Failure != nil || len(items[0].Files) != 3 || items[0].SourceIdentity == "" {
		t.Fatalf("scan = %#v, %v, %v", items, truncated, err)
	}
}

func TestModelSourceRejectsSymlinkPackage(t *testing.T) {
	rootPath := t.TempDir()
	target := t.TempDir()
	if err := os.Symlink(target, filepath.Join(rootPath, "unsafe.foliomodel")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	root, err := OpenRoot(rootPath)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	source, err := NewModelSource(root)
	if err != nil {
		t.Fatal(err)
	}
	items, _, err := source.ScanModelPackages(context.Background(), 64, 16, 4<<30)
	if err != nil || len(items) != 1 || items[0].Failure == nil {
		t.Fatalf("symlink scan = %#v, %v", items, err)
	}
}
