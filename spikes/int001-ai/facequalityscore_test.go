package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFaceQualityScoreRecomputesDetectionVerificationClusteringAndSlices(t *testing.T) {
	dataset := validFaceQualityDataset()
	manifestPath, identities := writeFaceQualityManifest(t, &dataset)
	bound, err := ValidateFaceQualityManifest(dataset, manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(bound) != len(identities) {
		t.Fatalf("bound=%d identities=%d", len(bound), len(identities))
	}
	report, err := ScoreFaceQuality(dataset, bound)
	if err != nil {
		t.Fatal(err)
	}
	if !report.GatePass || !report.GroupAssignmentAllowed || report.IdentityCount != 50 || report.ImageCount != 1000 || report.DetectionRecall.Rate != 1 || report.VerificationRecall.Rate != 1 || report.VerificationFPR.Rate != 0 || report.CorePrecision.Rate != 1 || len(report.Slices) < 5 {
		t.Fatalf("report=%+v", report)
	}
	if report.DetectionRecall.Lower95 <= 0 || report.CorePrecision.Lower95 <= 0 {
		t.Fatalf("missing confidence intervals: %+v %+v", report.DetectionRecall, report.CorePrecision)
	}
}

func TestFaceQualityScoreFailsClosedForCoreFalseMergeAndMissingBiasSlice(t *testing.T) {
	dataset := validFaceQualityDataset()
	dataset.ClusterPairs[0].SameIdentity = false
	report, err := ScoreFaceQuality(dataset, faceIdentityMap(dataset))
	if err != nil {
		t.Fatal(err)
	}
	if report.GatePass || report.GroupAssignmentAllowed || !containsFailure(report.GateFailures, "core precision") {
		t.Fatalf("report=%+v", report)
	}

	dataset = validFaceQualityDataset()
	for index := range dataset.Detections {
		filtered := dataset.Detections[index].Slices[:0]
		for _, value := range dataset.Detections[index].Slices {
			if !strings.HasPrefix(value, "skin-tone:") {
				filtered = append(filtered, value)
			}
		}
		dataset.Detections[index].Slices = filtered
	}
	if err := dataset.Validate(); err == nil || !strings.Contains(err.Error(), "skin-tone") {
		t.Fatalf("missing slice err=%v", err)
	}
}

func TestFaceQualityScoreRejectsUnlabeledOrSingleBucketBiasEvidence(t *testing.T) {
	dataset := validFaceQualityDataset()
	for index := range dataset.Detections {
		for sliceIndex, value := range dataset.Detections[index].Slices {
			if strings.HasPrefix(value, "skin-tone:") {
				dataset.Detections[index].Slices[sliceIndex] = "skin-tone:unlabeled"
			}
		}
	}
	if err := dataset.Validate(); err == nil || !strings.Contains(err.Error(), "non-evaluable skin-tone") {
		t.Fatalf("unlabeled slice err=%v", err)
	}

	dataset = validFaceQualityDataset()
	for index := range dataset.Detections {
		for sliceIndex, value := range dataset.Detections[index].Slices {
			if strings.HasPrefix(value, "lighting:") {
				dataset.Detections[index].Slices[sliceIndex] = "lighting:normal"
			}
		}
	}
	if err := dataset.Validate(); err == nil || !strings.Contains(err.Error(), "at least 2 lighting") {
		t.Fatalf("single bucket err=%v", err)
	}
}

func TestFaceQualityScoreUsesConfidenceBoundsNotPerfectSmallSamples(t *testing.T) {
	dataset := validFaceQualityDataset()
	dataset.VerificationPairs = dataset.VerificationPairs[:2]
	dataset.ClusterPairs = []FaceClusterPair{dataset.ClusterPairs[0], dataset.ClusterPairs[len(dataset.ClusterPairs)-1]}
	report, err := ScoreFaceQuality(dataset, faceIdentityMap(dataset))
	if err != nil {
		t.Fatal(err)
	}
	if report.GatePass || report.GroupAssignmentAllowed || report.CorePrecision.Rate != 1 || report.CorePrecision.Lower95 >= minimumAnonymousCoreQuality {
		t.Fatalf("small-sample report=%+v", report)
	}
}

func TestFaceQualityManifestRejectsPublicLicenseAndIdentityLabelMismatch(t *testing.T) {
	dataset := validFaceQualityDataset()
	manifestPath, _ := writeFaceQualityManifest(t, &dataset)
	dataset.VerificationPairs[0].SameIdentity = false
	if _, err := ValidateFaceQualityManifest(dataset, manifestPath); err == nil || !strings.Contains(err.Error(), "disagrees") {
		t.Fatalf("label mismatch err=%v", err)
	}

	var manifest DatasetManifest
	content, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(content, &manifest); err != nil {
		t.Fatal(err)
	}
	manifest.LegalBasis = "public-license"
	changed, _ := json.MarshalIndent(manifest, "", "  ")
	changed = append(changed, '\n')
	publicPath := filepath.Join(t.TempDir(), "public-face-manifest.json")
	if err := os.WriteFile(publicPath, changed, 0o600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(changed)
	dataset = validFaceQualityDataset()
	dataset.DatasetManifestSHA256 = hex.EncodeToString(sum[:])
	if _, err := ValidateFaceQualityManifest(dataset, publicPath); err == nil || !strings.Contains(err.Error(), "public media license") {
		t.Fatalf("public license err=%v", err)
	}
}

func TestRunFaceQualityScoreWritesCommitBoundSummary(t *testing.T) {
	dataset := validFaceQualityDataset()
	manifestPath, _ := writeFaceQualityManifest(t, &dataset)
	input := filepath.Join(t.TempDir(), "face-quality.json")
	content, _ := json.Marshal(dataset)
	if err := os.WriteFile(input, content, 0o600); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(t.TempDir(), "summary", "face-quality.json")
	commit := "0123456789abcdef0123456789abcdef01234567"
	if err := runFaceQualityScore([]string{"-input", input, "-dataset-manifest", manifestPath, "-commit", commit, "-output", output}); err != nil {
		t.Fatal(err)
	}
	var report FaceQualityReport
	content, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(content, &report); err != nil {
		t.Fatal(err)
	}
	if report.SourceCommit != commit || !report.GatePass || report.ModelPackageDigest != "sha256:"+dataset.ModelPackageSHA256 {
		t.Fatalf("report=%+v", report)
	}
}

func validFaceQualityDataset() FaceQualityDataset {
	value := FaceQualityDataset{
		SchemaVersion: 1, DatasetID: "authorized-face-quality-v1",
		DatasetManifestSHA256: strings.Repeat("a", 64), ModelPackageSHA256: strings.Repeat("b", 64),
		Approvals: FaceQualityApprovals{Product: "PRODUCT-APPROVAL-001", ML: "ML-APPROVAL-001", QA: "QA-APPROVAL-001", Privacy: "PRIVACY-APPROVAL-001", Compliance: "COMPLIANCE-APPROVAL-001"},
		Gate:      FaceQualityGate{MinimumDetectionRecall: .95, MinimumSliceRecall: .9, VerificationThreshold: .5, MinimumVerificationRecall: .95, MaximumVerificationFPR: .01, MinimumEdgePrecision: .8},
	}
	for identity := 0; identity < minimumFaceIdentities; identity++ {
		for image := 0; image < minimumImagesPerIdentity; image++ {
			id := faceQualityItemID(identity, image)
			value.Detections = append(value.Detections, FaceDetectionItem{ItemID: id, ExpectedFaces: 1, DetectedFaces: 1, MatchedFaces: 1, Slices: []string{
				fmt.Sprintf("skin-tone:tone-%d", identity%6+1), []string{"age:child", "age:adult", "age:older"}[identity%3], []string{"lighting:low", "lighting:normal"}[image%2], []string{"occlusion:clear", "occlusion:occluded"}[image%2], []string{"people-count:single", "people-count:multi"}[image%2],
			}})
		}
		for pair := 0; pair < minimumImagesPerIdentity; pair++ {
			left := faceQualityItemID(identity, pair)
			rightSame := faceQualityItemID(identity, (pair+1)%minimumImagesPerIdentity)
			rightDifferent := faceQualityItemID((identity+1)%minimumFaceIdentities, pair)
			value.VerificationPairs = append(value.VerificationPairs,
				FaceVerificationPair{LeftItemID: left, RightItemID: rightSame, SameIdentity: true, Similarity: .9},
				FaceVerificationPair{LeftItemID: left, RightItemID: rightDifferent, SameIdentity: false, Similarity: .1})
			value.ClusterPairs = append(value.ClusterPairs,
				FaceClusterPair{LeftItemID: left, RightItemID: rightSame, SameIdentity: true, Decision: "core_same"},
				FaceClusterPair{LeftItemID: left, RightItemID: rightDifferent, SameIdentity: false, Decision: "separate"})
		}
		for pair := 0; pair < minimumImagesPerIdentity; pair++ {
			value.ClusterPairs = append(value.ClusterPairs, FaceClusterPair{
				LeftItemID: faceQualityItemID(identity, pair), RightItemID: faceQualityItemID(identity, (pair+2)%minimumImagesPerIdentity),
				SameIdentity: true, Decision: "edge_same",
			})
		}
	}
	return value
}

func writeFaceQualityManifest(t *testing.T, dataset *FaceQualityDataset) (string, map[string]string) {
	t.Helper()
	manifest := DatasetManifest{SchemaVersion: 2, DatasetID: dataset.DatasetID + "-manifest", Purpose: "controlled face quality evaluation", LegalBasis: "written-authorization", License: LicenseEvidence{ID: "PRIVATE-EVALUATION-ONLY", URL: "https://example.invalid/approved-evidence"}, Redistributable: false, Governance: &DatasetGovernance{DataClass: "biometric-ground-truth", AllowedUses: []string{"face-detection-evaluation", "face-verification-evaluation", "face-clustering-evaluation"}, AuthorizedRoles: []string{"evaluation-team", "privacy-reviewer"}, RetentionUntil: "2099-12-31", DeletionProcedure: "delete-fixtures-and-derived-evidence", ConsentOrAuthorityRef: "AUTHORIZATION-001", PrivacyReviewRef: "PRIVACY-REVIEW-001", Redistribution: "prohibited"}}
	identities := map[string]string{}
	for identity := 0; identity < minimumFaceIdentities; identity++ {
		identityID := fmt.Sprintf("subject-%03d", identity)
		for image := 0; image < minimumImagesPerIdentity; image++ {
			id := faceQualityItemID(identity, image)
			identities[id] = identityID
			manifest.Items = append(manifest.Items, DatasetItem{ID: id, RelativePath: "private/" + id + ".jpg", MediaType: "image", SHA256: strings.Repeat("c", 64), IdentityID: identityID})
		}
	}
	content, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	content = append(content, '\n')
	sum := sha256.Sum256(content)
	dataset.DatasetManifestSHA256 = hex.EncodeToString(sum[:])
	filename := filepath.Join(t.TempDir(), "authorized-face-manifest.json")
	if err := os.WriteFile(filename, content, 0o600); err != nil {
		t.Fatal(err)
	}
	return filename, identities
}

func faceIdentityMap(dataset FaceQualityDataset) map[string]string {
	result := make(map[string]string, len(dataset.Detections))
	for _, item := range dataset.Detections {
		result[item.ItemID] = "subject-placeholder"
	}
	return result
}

func faceQualityItemID(identity, image int) string {
	return fmt.Sprintf("face-%03d-%03d", identity, image)
}
