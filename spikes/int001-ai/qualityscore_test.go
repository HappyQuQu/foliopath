package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScoreS2BQualityPassesFrozenTagAndVideoGates(t *testing.T) {
	dataset := validS2BQualityDataset()
	report, err := ScoreS2BQuality(dataset)
	if err != nil {
		t.Fatal(err)
	}
	if !report.GatePass || report.VideoCount != 100 || report.VideoTop20Success != 0.8 ||
		report.TagMacroPrecision < dataset.TagGate.MinimumMacroPrecision ||
		report.TagMacroRecall < dataset.TagGate.MinimumMacroRecall ||
		report.TagAcceptanceRate < dataset.TagGate.MinimumAcceptanceRate {
		t.Fatalf("unexpected quality report: %+v", report)
	}
}

func TestScoreS2BQualityFailsWithoutLoweringThresholds(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*S2BQualityDataset)
		want   string
	}{
		{
			name: "tag precision",
			mutate: func(dataset *S2BQualityDataset) {
				dataset.Tags[0].FalsePositive = 100
			},
			want: "tag macro precision",
		},
		{
			name: "video Top-20",
			mutate: func(dataset *S2BQualityDataset) {
				dataset.VideoQueries[7].Top20VideoIDs = []string{"video-099"}
			},
			want: "video Top-20",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dataset := validS2BQualityDataset()
			test.mutate(&dataset)
			report, err := ScoreS2BQuality(dataset)
			if err != nil {
				t.Fatal(err)
			}
			if report.GatePass || !containsFailure(report.GateFailures, test.want) {
				t.Fatalf("unexpected quality report: %+v", report)
			}
		})
	}
}

func TestS2BQualityDatasetRejectsIncompleteOrUnauditableEvidence(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*S2BQualityDataset)
		want   string
	}{
		{
			name: "fewer than 100 videos",
			mutate: func(dataset *S2BQualityDataset) {
				dataset.Videos = dataset.Videos[:99]
				dataset.VideoQueries[9].RelevantVideoIDs = []string{"video-000"}
				dataset.VideoQueries[9].Top20VideoIDs = []string{"video-000"}
			},
			want: "at least 100",
		},
		{
			name: "missing approval",
			mutate: func(dataset *S2BQualityDataset) {
				dataset.Approvals.QA = ""
			},
			want: "qa approval",
		},
		{
			name: "unknown result video",
			mutate: func(dataset *S2BQualityDataset) {
				dataset.VideoQueries[0].Top20VideoIDs = []string{"missing"}
			},
			want: "unknown video",
		},
		{
			name: "missing English queries",
			mutate: func(dataset *S2BQualityDataset) {
				for index := range dataset.VideoQueries {
					dataset.VideoQueries[index].Language = "zh"
				}
			},
			want: "both zh and en",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dataset := validS2BQualityDataset()
			test.mutate(&dataset)
			err := dataset.Validate()
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Validate() error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestValidateS2BQualityManifestBindsGovernedLicensedVideos(t *testing.T) {
	dataset := validS2BQualityDataset()
	manifestPath := writeS2BManifest(t, &dataset, "public-license")
	if err := ValidateS2BQualityManifest(dataset, manifestPath); err != nil {
		t.Fatal(err)
	}

	t.Run("digest mismatch", func(t *testing.T) {
		changed := dataset
		changed.DatasetManifestSHA256 = strings.Repeat("f", 64)
		err := ValidateS2BQualityManifest(changed, manifestPath)
		if err == nil || !strings.Contains(err.Error(), "SHA-256 mismatch") {
			t.Fatalf("ValidateS2BQualityManifest() error = %v", err)
		}
	})
	t.Run("synthetic manifest", func(t *testing.T) {
		changed := validS2BQualityDataset()
		path := writeS2BManifest(t, &changed, "synthetic")
		err := ValidateS2BQualityManifest(changed, path)
		if err == nil || !strings.Contains(err.Error(), "non-synthetic") {
			t.Fatalf("ValidateS2BQualityManifest() error = %v", err)
		}
	})
}

func TestRunQualityScoreWritesCommitAndDigestBoundSummary(t *testing.T) {
	dataset := validS2BQualityDataset()
	manifestPath := writeS2BManifest(t, &dataset, "public-license")
	inputPath := filepath.Join(t.TempDir(), "quality-input.json")
	content, err := json.Marshal(dataset)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(inputPath, content, 0o600); err != nil {
		t.Fatal(err)
	}
	outputPath := filepath.Join(t.TempDir(), "summary", "quality.json")
	commit := "0123456789abcdef0123456789abcdef01234567"
	if err := runQualityScore([]string{
		"-input", inputPath, "-dataset-manifest", manifestPath,
		"-commit", commit, "-output", outputPath,
	}); err != nil {
		t.Fatal(err)
	}
	var report S2BQualityReport
	content, err = os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(content, &report); err != nil {
		t.Fatal(err)
	}
	if report.SourceCommit != commit || report.DatasetManifestSHA256 != dataset.DatasetManifestSHA256 ||
		report.ModelPackageDigest != "sha256:"+dataset.ModelPackageSHA256 || !report.GatePass {
		t.Fatalf("unexpected bound quality summary: %+v", report)
	}
	if err := runQualityScore([]string{"-input", inputPath, "-dataset-manifest", manifestPath}); err == nil ||
		!strings.Contains(err.Error(), "valid -commit") {
		t.Fatalf("missing commit error = %v", err)
	}
}

func validS2BQualityDataset() S2BQualityDataset {
	dataset := S2BQualityDataset{
		SchemaVersion: 1, DatasetID: "approved-s2b-quality-v1",
		DatasetManifestSHA256: strings.Repeat("a", 64),
		ModelPackageSHA256:    strings.Repeat("b", 64),
		Approvals: S2BQualityApprovals{
			Product: "PRODUCT-APPROVAL-001", ML: "ML-APPROVAL-001", QA: "QA-APPROVAL-001",
		},
		TagGate: TagQualityGate{
			MinimumMacroPrecision: 0.8, MinimumMacroRecall: 0.8, MinimumAcceptanceRate: 0.75,
		},
		Tags: []TagQualityObservation{
			{TagID: "tag-a", TruePositive: 90, FalsePositive: 10, FalseNegative: 10, Accepted: 80, Reviewed: 100},
			{TagID: "tag-b", TruePositive: 85, FalsePositive: 15, FalseNegative: 15, Accepted: 75, Reviewed: 100},
		},
	}
	formats := []string{"mp4", "mov", "mkv"}
	for index := 0; index < 100; index++ {
		plan := 4
		if index%2 == 1 {
			plan = 10
		}
		motion := "static"
		if index%2 == 1 {
			motion = "motion"
		}
		setting := "indoor"
		if index%3 == 1 {
			setting = "outdoor"
		}
		dataset.Videos = append(dataset.Videos, VideoQualityItem{
			ID: videoID(index), Format: formats[index%len(formats)],
			DurationMS: int64(2_000 + index*1_000), FramePlan: plan,
			Motion: motion, Setting: setting,
		})
	}
	for index := 0; index < 10; index++ {
		resultID := videoID(index)
		if index >= 8 {
			resultID = videoID(99)
		}
		language := "zh"
		if index%2 == 1 {
			language = "en"
		}
		dataset.VideoQueries = append(dataset.VideoQueries, VideoQualityQuery{
			ID: "query-" + videoID(index), Language: language,
			RelevantVideoIDs: []string{videoID(index)}, Top20VideoIDs: []string{resultID},
		})
	}
	return dataset
}

func videoID(index int) string {
	const digits = "0123456789"
	return "video-" + string([]byte{digits[index/100], digits[(index/10)%10], digits[index%10]})
}

func containsFailure(failures []string, expected string) bool {
	for _, failure := range failures {
		if strings.Contains(failure, expected) {
			return true
		}
	}
	return false
}

func writeS2BManifest(t *testing.T, dataset *S2BQualityDataset, legalBasis string) string {
	t.Helper()
	manifest := DatasetManifest{
		SchemaVersion: 2, DatasetID: dataset.DatasetID + "-manifest",
		Purpose: "S2B quality evaluation", LegalBasis: legalBasis,
		License:         LicenseEvidence{ID: "CC-BY-4.0", URL: "https://creativecommons.org/licenses/by/4.0/"},
		Redistributable: true,
		Governance: &DatasetGovernance{
			DataClass:       "ordinary-media",
			AllowedUses:     []string{"semantic-evaluation", "video-evaluation"},
			AuthorizedRoles: []string{"evaluation-team"}, RetentionUntil: "2099-12-31",
			DeletionProcedure: "delete-fixtures-and-derived-evidence", Redistribution: "allowed",
		},
	}
	for _, video := range dataset.Videos {
		manifest.Items = append(manifest.Items, DatasetItem{
			ID: video.ID, RelativePath: "videos/" + video.ID + "." + video.Format,
			MediaType: "video", SHA256: strings.Repeat("c", 64),
		})
	}
	content, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	content = append(content, '\n')
	digest := sha256.Sum256(content)
	dataset.DatasetManifestSHA256 = hex.EncodeToString(digest[:])
	filename := filepath.Join(t.TempDir(), "dataset-manifest.json")
	if err := os.WriteFile(filename, content, 0o600); err != nil {
		t.Fatal(err)
	}
	return filename
}
