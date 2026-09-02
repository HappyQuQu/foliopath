package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
)

const (
	minimumFaceIdentities       = 50
	minimumImagesPerIdentity    = 20
	minimumFaceSliceGroups      = 2
	minimumFaceSliceFaces       = 20
	minimumAnonymousCoreQuality = 0.995
)

type FaceQualityDataset struct {
	SchemaVersion         int                    `json:"schema_version"`
	DatasetID             string                 `json:"dataset_id"`
	DatasetManifestSHA256 string                 `json:"dataset_manifest_sha256"`
	ModelPackageSHA256    string                 `json:"model_package_sha256"`
	Approvals             FaceQualityApprovals   `json:"approvals"`
	Gate                  FaceQualityGate        `json:"gate"`
	Detections            []FaceDetectionItem    `json:"detections"`
	VerificationPairs     []FaceVerificationPair `json:"verification_pairs"`
	ClusterPairs          []FaceClusterPair      `json:"cluster_pairs"`
}

type FaceQualityApprovals struct {
	Product    string `json:"product"`
	ML         string `json:"ml"`
	QA         string `json:"qa"`
	Privacy    string `json:"privacy"`
	Compliance string `json:"compliance"`
}

type FaceQualityGate struct {
	MinimumDetectionRecall    float64 `json:"minimum_detection_recall"`
	MinimumSliceRecall        float64 `json:"minimum_slice_recall"`
	VerificationThreshold     float64 `json:"verification_threshold"`
	MinimumVerificationRecall float64 `json:"minimum_verification_recall"`
	MaximumVerificationFPR    float64 `json:"maximum_verification_fpr"`
	MinimumEdgePrecision      float64 `json:"minimum_edge_precision"`
}

type FaceDetectionItem struct {
	ItemID        string   `json:"item_id"`
	ExpectedFaces int      `json:"expected_faces"`
	DetectedFaces int      `json:"detected_faces"`
	MatchedFaces  int      `json:"matched_faces"`
	Slices        []string `json:"slices"`
}

type FaceVerificationPair struct {
	LeftItemID   string  `json:"left_item_id"`
	RightItemID  string  `json:"right_item_id"`
	SameIdentity bool    `json:"same_identity"`
	Similarity   float64 `json:"similarity"`
}

type FaceClusterPair struct {
	LeftItemID   string `json:"left_item_id"`
	RightItemID  string `json:"right_item_id"`
	SameIdentity bool   `json:"same_identity"`
	Decision     string `json:"decision"`
}

type BinomialMetric struct {
	Successes int     `json:"successes"`
	Total     int     `json:"total"`
	Rate      float64 `json:"rate"`
	Lower95   float64 `json:"lower95"`
	Upper95   float64 `json:"upper95"`
}
type FaceSliceMetric struct {
	Slice  string         `json:"slice"`
	Recall BinomialMetric `json:"recall"`
}
type FaceQualityReport struct {
	SchemaVersion          int                  `json:"schema_version"`
	SourceCommit           string               `json:"source_commit,omitempty"`
	DatasetID              string               `json:"dataset_id"`
	DatasetManifestSHA256  string               `json:"dataset_manifest_sha256"`
	ModelPackageDigest     string               `json:"model_package_digest"`
	Approvals              FaceQualityApprovals `json:"approvals"`
	IdentityCount          int                  `json:"identity_count"`
	ImageCount             int                  `json:"image_count"`
	DetectionRecall        BinomialMetric       `json:"detection_recall"`
	VerificationRecall     BinomialMetric       `json:"verification_recall"`
	VerificationFPR        BinomialMetric       `json:"verification_fpr"`
	CorePrecision          BinomialMetric       `json:"core_precision"`
	EdgePrecision          BinomialMetric       `json:"edge_precision"`
	Slices                 []FaceSliceMetric    `json:"slices"`
	GroupAssignmentAllowed bool                 `json:"group_assignment_allowed"`
	GatePass               bool                 `json:"gate_pass"`
	GateFailures           []string             `json:"gate_failures"`
}

func ReadFaceQualityDataset(filename string) (FaceQualityDataset, error) {
	var value FaceQualityDataset
	if err := decodeStrict(filename, &value); err != nil {
		return value, err
	}
	return value, value.Validate()
}

func (dataset FaceQualityDataset) Validate() error {
	if dataset.SchemaVersion != 1 || dataset.DatasetID == "" || !sha256Pattern.MatchString(dataset.DatasetManifestSHA256) || !sha256Pattern.MatchString(dataset.ModelPackageSHA256) {
		return errors.New("face quality requires schema_version 1, dataset_id, manifest SHA-256, and model package SHA-256")
	}
	for name, value := range map[string]string{"product": dataset.Approvals.Product, "ml": dataset.Approvals.ML, "qa": dataset.Approvals.QA, "privacy": dataset.Approvals.Privacy, "compliance": dataset.Approvals.Compliance} {
		if !governanceReferencePattern.MatchString(value) {
			return fmt.Errorf("%s approval must be an opaque safe reference", name)
		}
	}
	for name, value := range map[string]float64{"minimum_detection_recall": dataset.Gate.MinimumDetectionRecall, "minimum_slice_recall": dataset.Gate.MinimumSliceRecall, "minimum_verification_recall": dataset.Gate.MinimumVerificationRecall, "minimum_edge_precision": dataset.Gate.MinimumEdgePrecision} {
		if value <= 0 || value > 1 {
			return fmt.Errorf("%s must be in (0,1]", name)
		}
	}
	if dataset.Gate.VerificationThreshold < -1 || dataset.Gate.VerificationThreshold > 1 || dataset.Gate.MaximumVerificationFPR < 0 || dataset.Gate.MaximumVerificationFPR >= 1 {
		return errors.New("invalid verification threshold or maximum FPR")
	}
	if len(dataset.Detections) == 0 || len(dataset.VerificationPairs) == 0 || len(dataset.ClusterPairs) == 0 {
		return errors.New("face quality requires detection, verification, and clustering observations")
	}
	seen := map[string]struct{}{}
	dimensions := map[string]map[string]int{}
	for _, item := range dataset.Detections {
		if item.ItemID == "" || item.ExpectedFaces < 1 || item.DetectedFaces < 0 || item.MatchedFaces < 0 || item.MatchedFaces > item.ExpectedFaces || item.MatchedFaces > item.DetectedFaces {
			return fmt.Errorf("invalid detection item %q", item.ItemID)
		}
		if _, ok := seen[item.ItemID]; ok {
			return fmt.Errorf("duplicate detection item %q", item.ItemID)
		}
		seen[item.ItemID] = struct{}{}
		itemDimensions := map[string]bool{}
		for _, slice := range item.Slices {
			if !governanceReferencePattern.MatchString(slice) {
				return fmt.Errorf("invalid slice %q", slice)
			}
			for _, prefix := range []string{"skin-tone:", "age:", "lighting:", "occlusion:", "people-count:"} {
				if strings.HasPrefix(slice, prefix) {
					if itemDimensions[prefix] {
						return fmt.Errorf("detection item %q repeats %s slice", item.ItemID, strings.TrimSuffix(prefix, ":"))
					}
					value := strings.TrimPrefix(slice, prefix)
					if value == "" || value == "unknown" || value == "unlabeled" || value == "unspecified" {
						return fmt.Errorf("face quality has non-evaluable %s slice %q", strings.TrimSuffix(prefix, ":"), slice)
					}
					if dimensions[prefix] == nil {
						dimensions[prefix] = map[string]int{}
					}
					dimensions[prefix][value] += item.ExpectedFaces
					itemDimensions[prefix] = true
				}
			}
		}
		for _, prefix := range []string{"skin-tone:", "age:", "lighting:", "occlusion:", "people-count:"} {
			if !itemDimensions[prefix] {
				return fmt.Errorf("detection item %q lacks %s slice", item.ItemID, strings.TrimSuffix(prefix, ":"))
			}
		}
	}
	for _, prefix := range []string{"skin-tone:", "age:", "lighting:", "occlusion:", "people-count:"} {
		if len(dimensions[prefix]) < minimumFaceSliceGroups {
			return fmt.Errorf("face quality requires at least %d %s slice groups", minimumFaceSliceGroups, strings.TrimSuffix(prefix, ":"))
		}
		for value, faces := range dimensions[prefix] {
			if faces < minimumFaceSliceFaces {
				return fmt.Errorf("%s slice %q has %d expected faces, want at least %d", strings.TrimSuffix(prefix, ":"), value, faces, minimumFaceSliceFaces)
			}
		}
	}
	if err := validateFacePairs(dataset.VerificationPairs, dataset.ClusterPairs, seen); err != nil {
		return err
	}
	return nil
}

func validateFacePairs(verification []FaceVerificationPair, clusters []FaceClusterPair, items map[string]struct{}) error {
	seen := map[string]struct{}{}
	same, different := false, false
	for _, pair := range verification {
		key, err := facePairKey(pair.LeftItemID, pair.RightItemID, items)
		if err != nil {
			return err
		}
		if _, ok := seen[key]; ok {
			return fmt.Errorf("duplicate verification pair %q", key)
		}
		seen[key] = struct{}{}
		if math.IsNaN(pair.Similarity) || math.IsInf(pair.Similarity, 0) || pair.Similarity < -1 || pair.Similarity > 1 {
			return fmt.Errorf("invalid similarity for %q", key)
		}
		same = same || pair.SameIdentity
		different = different || !pair.SameIdentity
	}
	if !same || !different {
		return errors.New("verification pairs require same and different identities")
	}
	seen = map[string]struct{}{}
	core, edge := false, false
	for _, pair := range clusters {
		key, err := facePairKey(pair.LeftItemID, pair.RightItemID, items)
		if err != nil {
			return err
		}
		if _, ok := seen[key]; ok {
			return fmt.Errorf("duplicate cluster pair %q", key)
		}
		seen[key] = struct{}{}
		if pair.Decision != "core_same" && pair.Decision != "edge_same" && pair.Decision != "separate" {
			return fmt.Errorf("invalid cluster decision %q", pair.Decision)
		}
		core = core || pair.Decision == "core_same"
		edge = edge || pair.Decision == "edge_same"
	}
	if !core || !edge {
		return errors.New("cluster observations require core and edge decisions")
	}
	return nil
}

func facePairKey(left, right string, items map[string]struct{}) (string, error) {
	if left == right {
		return "", errors.New("face pair must contain different items")
	}
	if _, ok := items[left]; !ok {
		return "", fmt.Errorf("unknown face item %q", left)
	}
	if _, ok := items[right]; !ok {
		return "", fmt.Errorf("unknown face item %q", right)
	}
	if left > right {
		left, right = right, left
	}
	return left + "\x00" + right, nil
}

func ValidateFaceQualityManifest(dataset FaceQualityDataset, filename string) (map[string]string, error) {
	content, err := readEvidenceJSONFile(filename)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(content)
	if hex.EncodeToString(sum[:]) != dataset.DatasetManifestSHA256 {
		return nil, errors.New("governed face manifest SHA-256 mismatch")
	}
	manifest, err := ReadDatasetManifest(filename)
	if err != nil {
		return nil, err
	}
	if manifest.SchemaVersion != 2 || manifest.LegalBasis != "written-authorization" || manifest.Governance == nil || manifest.Governance.DataClass != "biometric-ground-truth" || !containsString(manifest.Governance.AllowedUses, "face-detection-evaluation") || !containsString(manifest.Governance.AllowedUses, "face-verification-evaluation") || !containsString(manifest.Governance.AllowedUses, "face-clustering-evaluation") {
		return nil, errors.New("face quality requires authorized schema v2 biometric ground truth for all face evaluation uses")
	}
	identities := map[string]int{}
	items := map[string]string{}
	for _, item := range manifest.Items {
		if item.MediaType != "image" || item.IdentityID == "" {
			return nil, fmt.Errorf("face manifest item %q is not an identity-labeled image", item.ID)
		}
		items[item.ID] = item.IdentityID
		identities[item.IdentityID]++
	}
	if len(identities) < minimumFaceIdentities {
		return nil, fmt.Errorf("face manifest has %d identities, want at least %d", len(identities), minimumFaceIdentities)
	}
	for id, count := range identities {
		if count < minimumImagesPerIdentity {
			return nil, fmt.Errorf("identity %q has %d images, want at least %d", id, count, minimumImagesPerIdentity)
		}
	}
	if len(dataset.Detections) != len(items) {
		return nil, errors.New("face detection results must cover every governed manifest item exactly once")
	}
	for _, item := range dataset.Detections {
		if _, ok := items[item.ItemID]; !ok {
			return nil, fmt.Errorf("detection item %q absent from manifest", item.ItemID)
		}
	}
	for _, pair := range dataset.VerificationPairs {
		if (items[pair.LeftItemID] == items[pair.RightItemID]) != pair.SameIdentity {
			return nil, fmt.Errorf("verification label disagrees with manifest for %q/%q", pair.LeftItemID, pair.RightItemID)
		}
	}
	for _, pair := range dataset.ClusterPairs {
		if (items[pair.LeftItemID] == items[pair.RightItemID]) != pair.SameIdentity {
			return nil, fmt.Errorf("cluster label disagrees with manifest for %q/%q", pair.LeftItemID, pair.RightItemID)
		}
	}
	return items, nil
}

func ScoreFaceQuality(dataset FaceQualityDataset, identities map[string]string) (FaceQualityReport, error) {
	if err := dataset.Validate(); err != nil {
		return FaceQualityReport{}, err
	}
	identitySet := map[string]struct{}{}
	for _, id := range identities {
		identitySet[id] = struct{}{}
	}
	report := FaceQualityReport{SchemaVersion: 1, DatasetID: dataset.DatasetID, DatasetManifestSHA256: dataset.DatasetManifestSHA256, ModelPackageDigest: "sha256:" + dataset.ModelPackageSHA256, Approvals: dataset.Approvals, IdentityCount: len(identitySet), ImageCount: len(dataset.Detections)}
	matched, expected := 0, 0
	slices := map[string][2]int{}
	for _, item := range dataset.Detections {
		matched += item.MatchedFaces
		expected += item.ExpectedFaces
		for _, name := range item.Slices {
			v := slices[name]
			v[0] += item.MatchedFaces
			v[1] += item.ExpectedFaces
			slices[name] = v
		}
	}
	report.DetectionRecall = binomial(matched, expected)
	names := make([]string, 0, len(slices))
	for name := range slices {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		v := slices[name]
		metric := binomial(v[0], v[1])
		report.Slices = append(report.Slices, FaceSliceMetric{Slice: name, Recall: metric})
		if metric.Lower95 < dataset.Gate.MinimumSliceRecall {
			report.GateFailures = append(report.GateFailures, "slice recall below approved threshold: "+name)
		}
	}
	tp, fn, fp, tn := 0, 0, 0, 0
	for _, pair := range dataset.VerificationPairs {
		predicted := pair.Similarity >= dataset.Gate.VerificationThreshold
		if pair.SameIdentity && predicted {
			tp++
		} else if pair.SameIdentity {
			fn++
		} else if predicted {
			fp++
		} else {
			tn++
		}
	}
	report.VerificationRecall = binomial(tp, tp+fn)
	report.VerificationFPR = binomial(fp, fp+tn)
	coreTP, coreTotal, edgeTP, edgeTotal := 0, 0, 0, 0
	for _, pair := range dataset.ClusterPairs {
		switch pair.Decision {
		case "core_same":
			coreTotal++
			if pair.SameIdentity {
				coreTP++
			}
		case "edge_same":
			edgeTotal++
			if pair.SameIdentity {
				edgeTP++
			}
		}
	}
	report.CorePrecision = binomial(coreTP, coreTotal)
	report.EdgePrecision = binomial(edgeTP, edgeTotal)
	report.GroupAssignmentAllowed = coreTotal > 0 && report.CorePrecision.Lower95 >= minimumAnonymousCoreQuality
	if report.DetectionRecall.Lower95 < dataset.Gate.MinimumDetectionRecall {
		report.GateFailures = append(report.GateFailures, "detection recall below approved threshold")
	}
	if report.VerificationRecall.Lower95 < dataset.Gate.MinimumVerificationRecall {
		report.GateFailures = append(report.GateFailures, "verification recall below approved threshold")
	}
	if report.VerificationFPR.Upper95 > dataset.Gate.MaximumVerificationFPR {
		report.GateFailures = append(report.GateFailures, "verification false-positive rate above approved threshold")
	}
	if !report.GroupAssignmentAllowed {
		report.GateFailures = append(report.GateFailures, "anonymous core precision below 99.5%")
	}
	if report.EdgePrecision.Lower95 < dataset.Gate.MinimumEdgePrecision {
		report.GateFailures = append(report.GateFailures, "edge precision below approved threshold")
	}
	report.GatePass = len(report.GateFailures) == 0
	return report, nil
}

func binomial(successes, total int) BinomialMetric {
	value := BinomialMetric{Successes: successes, Total: total}
	if total == 0 {
		return value
	}
	value.Rate = float64(successes) / float64(total)
	const z = 1.959963984540054
	n := float64(total)
	denom := 1 + z*z/n
	center := (value.Rate + z*z/(2*n)) / denom
	margin := z * math.Sqrt(value.Rate*(1-value.Rate)/n+z*z/(4*n*n)) / denom
	value.Lower95 = max(0.0, center-margin)
	value.Upper95 = min(1.0, center+margin)
	return value
}
