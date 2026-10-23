//go:build linux

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanModelsAllowsOnlyMatchingRegularFile(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "model.onnx"), []byte("model"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "unknown.onnx"), []byte("unknown"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("model.onnx", filepath.Join(root, "alias.onnx")); err != nil {
		t.Fatal(err)
	}
	catalog := ModelCatalog{SchemaVersion: 1, Models: []ModelEntry{{
		ID: "model", Status: "approved", Purpose: "test", Version: "1", Filename: "model.onnx",
		SHA256:    "9372c470eeadd5ecd9c3c74c2b3cb633f8e2f2fad799250a0f70d652b6b825e4",
		SizeBytes: 5, SourceURL: "https://example.invalid/model.onnx",
		CodeLicense:   LicenseEvidence{ID: "test", URL: "https://example.invalid/code"},
		WeightLicense: LicenseEvidence{ID: "test", URL: "https://example.invalid/weights"},
		Architectures: []string{"linux/amd64"},
	}}}
	report, err := ScanModels(root, catalog, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Accepted) != 1 || report.Accepted[0].Filename != "model.onnx" {
		t.Fatalf("unexpected accepted files: %#v", report.Accepted)
	}
	if len(report.Rejected) != 2 {
		t.Fatalf("expected unknown and symlink files to be rejected: %#v", report.Rejected)
	}
}

func TestScanModelsRequiresExactPackageContents(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(t *testing.T, packageRoot string)
		accept bool
	}{
		{name: "complete package", accept: true},
		{
			name: "missing artifact",
			mutate: func(t *testing.T, packageRoot string) {
				t.Helper()
				if err := os.Remove(filepath.Join(packageRoot, "tokenizer.json")); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "undeclared artifact",
			mutate: func(t *testing.T, packageRoot string) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(packageRoot, "injected.py"), []byte("payload"), 0o600); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "changed bytes",
			mutate: func(t *testing.T, packageRoot string) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(packageRoot, "model.safetensors"), []byte("other"), 0o600); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "artifact symlink",
			mutate: func(t *testing.T, packageRoot string) {
				t.Helper()
				if err := os.Remove(filepath.Join(packageRoot, "tokenizer.json")); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink("config.json", filepath.Join(packageRoot, "tokenizer.json")); err != nil {
					t.Fatal(err)
				}
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			packageRoot := filepath.Join(root, "synthetic-semantic-v1")
			if err := os.Mkdir(packageRoot, 0o700); err != nil {
				t.Fatal(err)
			}
			for filename, contents := range map[string]string{
				"config.json":       "config",
				"model.safetensors": "model",
				"tokenizer.json":    "tokenizer",
			} {
				if err := os.WriteFile(filepath.Join(packageRoot, filename), []byte(contents), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if test.mutate != nil {
				test.mutate(t, packageRoot)
			}
			catalog, err := ReadModelCatalog("testdata/model-catalog.package.example.json")
			if err != nil {
				t.Fatal(err)
			}
			report, err := ScanModels(root, catalog, false)
			if err != nil {
				t.Fatal(err)
			}
			if test.accept {
				if len(report.AcceptedPackages) != 1 || len(report.Rejected) != 0 {
					t.Fatalf("expected package acceptance, got %#v", report)
				}
				accepted := report.AcceptedPackages[0]
				if accepted.ArtifactCount != 3 || accepted.TotalBytes != 20 || accepted.PackageSHA256 != catalog.Models[0].PackageSHA256 {
					t.Fatalf("unexpected accepted package: %#v", accepted)
				}
				return
			}
			if len(report.AcceptedPackages) != 0 || len(report.Rejected) != 1 {
				t.Fatalf("expected package rejection, got %#v", report)
			}
		})
	}
}
