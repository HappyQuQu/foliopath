package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEvidenceJSONRejectsNestedDuplicateKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "manifest.json")
	content := `{"schema_version":2,"governance":{"data_class":"ordinary-media","data_class":"biometric-ground-truth"}}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	var manifest DatasetManifest
	if err := decodeStrict(path, &manifest); err == nil || !strings.Contains(err.Error(), "duplicate JSON key") {
		t.Fatalf("error=%v", err)
	}
}

func TestEvidenceJSONRejectsSymlinkAndOversizedFiles(t *testing.T) {
	directory := t.TempDir()
	target := filepath.Join(directory, "target.json")
	if err := os.WriteFile(target, []byte(`{"schema_version":1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(directory, "link.json")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if _, err := readEvidenceJSONFile(link); err == nil || !strings.Contains(err.Error(), "non-symlink regular file") {
		t.Fatalf("symlink error=%v", err)
	}
	oversized := filepath.Join(directory, "oversized.json")
	file, err := os.Create(oversized)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(maxEvidenceJSONBytes + 1); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := readEvidenceJSONFile(oversized); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("oversized error=%v", err)
	}
}

func TestExampleManifestsValidate(t *testing.T) {
	if _, err := ReadDatasetManifest("testdata/dataset-manifest.example.json"); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadDatasetManifest("testdata/dataset-manifest.authorized-face-template.json"); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadModelCatalog("testdata/model-catalog.example.json"); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadModelCatalog("testdata/model-catalog.package.example.json"); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadModelCatalog("testdata/model-catalog.siglip-candidates.json"); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadModelCatalog("testdata/model-catalog.arcface-alternative.json"); err != nil {
		t.Fatal(err)
	}
}

func TestAuthorizedFaceDatasetGovernance(t *testing.T) {
	manifest, err := ReadDatasetManifest("testdata/dataset-manifest.authorized-face-template.json")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		edit func(*DatasetManifest)
		want string
	}{
		{
			name: "public license is not biometric authority",
			edit: func(changed *DatasetManifest) { changed.LegalBasis = "public-license" },
			want: "public media license is not authority",
		},
		{
			name: "missing authority reference",
			edit: func(changed *DatasetManifest) { changed.Governance.ConsentOrAuthorityRef = "" },
			want: "consent_or_authority_ref",
		},
		{
			name: "missing privacy review",
			edit: func(changed *DatasetManifest) { changed.Governance.PrivacyReviewRef = "" },
			want: "privacy_review_ref",
		},
		{
			name: "missing retention",
			edit: func(changed *DatasetManifest) { changed.Governance.RetentionUntil = "" },
			want: "retention_until",
		},
		{
			name: "invalid retention",
			edit: func(changed *DatasetManifest) { changed.Governance.RetentionUntil = "until-finished" },
			want: "retention_until",
		},
		{
			name: "unknown data class",
			edit: func(changed *DatasetManifest) { changed.Governance.DataClass = "personal-media" },
			want: "unsupported dataset data_class",
		},
		{
			name: "unknown deletion procedure",
			edit: func(changed *DatasetManifest) { changed.Governance.DeletionProcedure = "keep-derived-data" },
			want: "unsupported deletion_procedure",
		},
		{
			name: "redistribution fields disagree",
			edit: func(changed *DatasetManifest) { changed.Redistributable = true },
			want: "must agree",
		},
		{
			name: "redistributable biometric data",
			edit: func(changed *DatasetManifest) {
				changed.Redistributable = true
				changed.Governance.Redistribution = "allowed"
			},
			want: "redistribution must be prohibited",
		},
		{
			name: "unknown allowed use",
			edit: func(changed *DatasetManifest) { changed.Governance.AllowedUses = []string{"training"} },
			want: "unsupported allowed_use",
		},
		{
			name: "unknown access role",
			edit: func(changed *DatasetManifest) { changed.Governance.AuthorizedRoles = []string{"all-developers"} },
			want: "unsupported authorized_role",
		},
		{
			name: "unsafe identity reference",
			edit: func(changed *DatasetManifest) { changed.Items[0].IdentityID = "person/alice" },
			want: "opaque safe reference",
		},
		{
			name: "missing clustering identity",
			edit: func(changed *DatasetManifest) { changed.Items[0].IdentityID = "" },
			want: "requires identity_id",
		},
		{
			name: "ordinary data cannot carry identity",
			edit: func(changed *DatasetManifest) {
				changed.Governance.DataClass = "ordinary-media"
				changed.Governance.AllowedUses = []string{"semantic-evaluation"}
			},
			want: "outside biometric ground truth",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			changed := cloneDatasetManifest(t, manifest)
			test.edit(&changed)
			err := changed.Validate()
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected error containing %q, got %v", test.want, err)
			}
		})
	}
}

func TestDatasetSchemaTwoRequiresGovernance(t *testing.T) {
	manifest, err := ReadDatasetManifest("testdata/dataset-manifest.example.json")
	if err != nil {
		t.Fatal(err)
	}
	manifest.SchemaVersion = 2
	if err := manifest.Validate(); err == nil || !strings.Contains(err.Error(), "requires governance") {
		t.Fatalf("expected governance error, got %v", err)
	}
}

func TestNonSyntheticDatasetRequiresSchemaTwoGovernance(t *testing.T) {
	manifest, err := ReadDatasetManifest("testdata/dataset-manifest.example.json")
	if err != nil {
		t.Fatal(err)
	}
	manifest.LegalBasis = "public-license"
	if err := manifest.Validate(); err == nil || !strings.Contains(err.Error(), "require schema_version 2") {
		t.Fatalf("expected schema version 2 governance error, got %v", err)
	}
}

func cloneDatasetManifest(t *testing.T, source DatasetManifest) DatasetManifest {
	t.Helper()
	changed := source
	governance := *source.Governance
	governance.AllowedUses = append([]string(nil), source.Governance.AllowedUses...)
	governance.AuthorizedRoles = append([]string(nil), source.Governance.AuthorizedRoles...)
	changed.Governance = &governance
	changed.Items = append([]DatasetItem(nil), source.Items...)
	return changed
}

func TestPackageManifestRejectsIdentityAndPathChanges(t *testing.T) {
	catalog, err := ReadModelCatalog("testdata/model-catalog.package.example.json")
	if err != nil {
		t.Fatal(err)
	}
	t.Run("package digest", func(t *testing.T) {
		changed := catalog
		changed.Models = append([]ModelEntry(nil), catalog.Models...)
		changed.Models[0].PackageSHA256 = "0000000000000000000000000000000000000000000000000000000000000000"
		if err := changed.Validate(); err == nil {
			t.Fatal("expected changed package digest to be rejected")
		}
	})
	t.Run("artifact traversal", func(t *testing.T) {
		changed := catalog
		changed.Models = append([]ModelEntry(nil), catalog.Models...)
		changed.Models[0].Artifacts = append([]ModelArtifact(nil), catalog.Models[0].Artifacts...)
		changed.Models[0].Artifacts[0].Filename = "../config.json"
		if err := changed.Validate(); err == nil {
			t.Fatal("expected artifact traversal to be rejected")
		}
	})
	t.Run("duplicate artifact", func(t *testing.T) {
		changed := catalog
		changed.Models = append([]ModelEntry(nil), catalog.Models...)
		changed.Models[0].Artifacts = append(changed.Models[0].Artifacts, changed.Models[0].Artifacts[0])
		if err := changed.Validate(); err == nil {
			t.Fatal("expected duplicate artifact to be rejected")
		}
	})
}

func TestDatasetRejectsTraversal(t *testing.T) {
	manifest, err := ReadDatasetManifest("testdata/dataset-manifest.example.json")
	if err != nil {
		t.Fatal(err)
	}
	manifest.Items[0].RelativePath = "../private.jpg"
	if err := manifest.Validate(); err == nil {
		t.Fatal("expected traversal to be rejected")
	}
}
