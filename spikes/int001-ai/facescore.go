package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"sort"
)

type FaceScoreDataset struct {
	SchemaVersion       int               `json:"schema_version"`
	DatasetID           string            `json:"dataset_id"`
	LegalBasis          string            `json:"legal_basis"`
	License             LicenseEvidence   `json:"license"`
	SimilarityThreshold float64           `json:"similarity_threshold"`
	CorePrecisionGate   float64           `json:"core_precision_gate"`
	Observations        []FaceObservation `json:"observations"`
	CannotLinks         [][2]string       `json:"cannot_links"`
	ManualAssignments   map[string]string `json:"manual_assignments"`
}

type FaceObservation struct {
	ID         string    `json:"id"`
	IdentityID string    `json:"identity_id"`
	ClusterID  string    `json:"cluster_id,omitempty"`
	Tier       string    `json:"tier,omitempty"`
	PersonID   string    `json:"person_id,omitempty"`
	Embedding  []float32 `json:"embedding"`
}

type FaceScoreReport struct {
	SchemaVersion              int           `json:"schema_version"`
	DatasetID                  string        `json:"dataset_id"`
	Observations               int           `json:"observations"`
	Identities                 int           `json:"identities"`
	Dimensions                 int           `json:"dimensions"`
	SimilarityThreshold        float64       `json:"similarity_threshold"`
	PairMetrics                BinaryMetrics `json:"pair_metrics"`
	ClusterPairPrecision       float64       `json:"cluster_pair_precision"`
	ClusterPairRecall          float64       `json:"cluster_pair_recall"`
	CoreClusterPairPrecision   float64       `json:"core_cluster_pair_precision"`
	CannotLinkViolations       int           `json:"cannot_link_violations"`
	ManualAssignmentViolations int           `json:"manual_assignment_violations"`
	GatePass                   bool          `json:"gate_pass"`
	GateFailures               []string      `json:"gate_failures"`
	ROCCurve                   []ROCPoint    `json:"roc_curve"`
}

type BinaryMetrics struct {
	TruePositive      int     `json:"true_positive"`
	FalsePositive     int     `json:"false_positive"`
	TrueNegative      int     `json:"true_negative"`
	FalseNegative     int     `json:"false_negative"`
	Precision         float64 `json:"precision"`
	Recall            float64 `json:"recall"`
	FalsePositiveRate float64 `json:"false_positive_rate"`
}

type ROCPoint struct {
	Threshold         float64 `json:"threshold"`
	TruePositiveRate  float64 `json:"true_positive_rate"`
	FalsePositiveRate float64 `json:"false_positive_rate"`
}

type facePair struct {
	SameIdentity bool
	Similarity   float64
	SameCluster  bool
	BothCore     bool
}

func ReadFaceScoreDataset(filename string) (FaceScoreDataset, error) {
	var dataset FaceScoreDataset
	file, err := os.Open(filename)
	if err != nil {
		return dataset, err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&dataset); err != nil {
		return dataset, fmt.Errorf("decode face score dataset: %w", err)
	}
	return dataset, dataset.Validate()
}

func (dataset FaceScoreDataset) Validate() error {
	if dataset.SchemaVersion != 1 || dataset.DatasetID == "" {
		return errors.New("face score dataset requires schema_version 1 and dataset_id")
	}
	if dataset.LegalBasis != "synthetic" && dataset.LegalBasis != "public-license" && dataset.LegalBasis != "written-authorization" {
		return errors.New("face score dataset has unsupported legal_basis")
	}
	if err := validateLicense("face score dataset", dataset.License); err != nil {
		return err
	}
	if dataset.SimilarityThreshold < -1 || dataset.SimilarityThreshold > 1 {
		return errors.New("similarity_threshold must be between -1 and 1")
	}
	if dataset.CorePrecisionGate <= 0 || dataset.CorePrecisionGate > 1 {
		return errors.New("core_precision_gate must be in (0, 1]")
	}
	if len(dataset.Observations) < 4 || len(dataset.Observations) > 5000 {
		return errors.New("face score dataset must contain 4 to 5000 observations")
	}
	dimensions := len(dataset.Observations[0].Embedding)
	if dimensions == 0 || dimensions > 4096 {
		return errors.New("embedding dimensions must be in [1, 4096]")
	}
	seen := make(map[string]struct{}, len(dataset.Observations))
	identities := make(map[string]struct{})
	for _, observation := range dataset.Observations {
		if observation.ID == "" || observation.IdentityID == "" || len(observation.Embedding) != dimensions {
			return errors.New("every observation requires an id, identity_id, and consistent embedding dimensions")
		}
		if _, exists := seen[observation.ID]; exists {
			return fmt.Errorf("duplicate observation %q", observation.ID)
		}
		seen[observation.ID] = struct{}{}
		identities[observation.IdentityID] = struct{}{}
		if observation.Tier != "" && observation.Tier != "core" && observation.Tier != "edge" {
			return fmt.Errorf("observation %q has unsupported tier", observation.ID)
		}
		var norm float64
		for _, value := range observation.Embedding {
			if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
				return fmt.Errorf("observation %q has non-finite embedding", observation.ID)
			}
			norm += float64(value * value)
		}
		if norm == 0 {
			return fmt.Errorf("observation %q has zero embedding", observation.ID)
		}
	}
	if len(identities) < 2 {
		return errors.New("face score dataset needs at least two identities")
	}
	for _, link := range dataset.CannotLinks {
		if link[0] == link[1] {
			return errors.New("cannot-link endpoints must differ")
		}
		if _, exists := seen[link[0]]; !exists {
			return fmt.Errorf("cannot-link references unknown observation %q", link[0])
		}
		if _, exists := seen[link[1]]; !exists {
			return fmt.Errorf("cannot-link references unknown observation %q", link[1])
		}
	}
	for observationID, personID := range dataset.ManualAssignments {
		if _, exists := seen[observationID]; !exists || personID == "" {
			return errors.New("manual assignment references unknown observation or empty person")
		}
	}
	return nil
}

func ScoreFaces(dataset FaceScoreDataset) (FaceScoreReport, error) {
	if err := dataset.Validate(); err != nil {
		return FaceScoreReport{}, err
	}
	byID := make(map[string]FaceObservation, len(dataset.Observations))
	identities := make(map[string]struct{})
	for _, observation := range dataset.Observations {
		byID[observation.ID] = observation
		identities[observation.IdentityID] = struct{}{}
	}
	pairs := make([]facePair, 0, len(dataset.Observations)*(len(dataset.Observations)-1)/2)
	for left := 0; left < len(dataset.Observations); left++ {
		for right := left + 1; right < len(dataset.Observations); right++ {
			a := dataset.Observations[left]
			b := dataset.Observations[right]
			pairs = append(pairs, facePair{
				SameIdentity: a.IdentityID == b.IdentityID,
				Similarity:   cosine(a.Embedding, b.Embedding),
				SameCluster:  a.ClusterID != "" && a.ClusterID == b.ClusterID,
				BothCore:     a.Tier == "core" && b.Tier == "core",
			})
		}
	}
	report := FaceScoreReport{
		SchemaVersion: 1, DatasetID: dataset.DatasetID, Observations: len(dataset.Observations),
		Identities: len(identities), Dimensions: len(dataset.Observations[0].Embedding),
		SimilarityThreshold: dataset.SimilarityThreshold,
		PairMetrics:         scoreThreshold(pairs, dataset.SimilarityThreshold),
	}
	report.ClusterPairPrecision, report.ClusterPairRecall = clusterPairMetrics(pairs, false)
	report.CoreClusterPairPrecision, _ = clusterPairMetrics(pairs, true)
	for _, link := range dataset.CannotLinks {
		left := byID[link[0]]
		right := byID[link[1]]
		if left.ClusterID != "" && left.ClusterID == right.ClusterID {
			report.CannotLinkViolations++
		}
	}
	for observationID, personID := range dataset.ManualAssignments {
		if byID[observationID].PersonID != personID {
			report.ManualAssignmentViolations++
		}
	}
	report.ROCCurve = rocCurve(pairs)
	if report.CoreClusterPairPrecision < dataset.CorePrecisionGate {
		report.GateFailures = append(report.GateFailures, "core cluster pair precision below gate")
	}
	if report.CannotLinkViolations != 0 {
		report.GateFailures = append(report.GateFailures, "cannot-link violation")
	}
	if report.ManualAssignmentViolations != 0 {
		report.GateFailures = append(report.GateFailures, "manual assignment violation")
	}
	report.GatePass = len(report.GateFailures) == 0
	return report, nil
}

func scoreThreshold(pairs []facePair, threshold float64) BinaryMetrics {
	var metrics BinaryMetrics
	for _, pair := range pairs {
		predicted := pair.Similarity >= threshold
		switch {
		case predicted && pair.SameIdentity:
			metrics.TruePositive++
		case predicted && !pair.SameIdentity:
			metrics.FalsePositive++
		case !predicted && pair.SameIdentity:
			metrics.FalseNegative++
		default:
			metrics.TrueNegative++
		}
	}
	metrics.Precision = ratio(metrics.TruePositive, metrics.TruePositive+metrics.FalsePositive)
	metrics.Recall = ratio(metrics.TruePositive, metrics.TruePositive+metrics.FalseNegative)
	metrics.FalsePositiveRate = ratio(metrics.FalsePositive, metrics.FalsePositive+metrics.TrueNegative)
	return metrics
}

func clusterPairMetrics(pairs []facePair, coreOnly bool) (float64, float64) {
	var predictedPairs, correctPredicted, identityPairs, recoveredIdentity int
	for _, pair := range pairs {
		if coreOnly && !pair.BothCore {
			continue
		}
		if pair.SameCluster {
			predictedPairs++
			if pair.SameIdentity {
				correctPredicted++
			}
		}
		if pair.SameIdentity {
			identityPairs++
			if pair.SameCluster {
				recoveredIdentity++
			}
		}
	}
	return ratio(correctPredicted, predictedPairs), ratio(recoveredIdentity, identityPairs)
}

func rocCurve(pairs []facePair) []ROCPoint {
	thresholds := []float64{-1, 0, 0.25, 0.5, 0.65, 0.75, 0.8, 0.85, 0.9, 0.95, 0.99, 1.000001}
	points := make([]ROCPoint, 0, len(thresholds))
	for _, threshold := range thresholds {
		metrics := scoreThreshold(pairs, threshold)
		points = append(points, ROCPoint{threshold, metrics.Recall, metrics.FalsePositiveRate})
	}
	sort.Slice(points, func(i, j int) bool { return points[i].Threshold < points[j].Threshold })
	return points
}

func cosine(a, b []float32) float64 {
	var dotProduct, normA, normB float64
	for index := range a {
		dotProduct += float64(a[index] * b[index])
		normA += float64(a[index] * a[index])
		normB += float64(b[index] * b[index])
	}
	return dotProduct / math.Sqrt(normA*normB)
}

func ratio(numerator, denominator int) float64 {
	if denominator == 0 {
		return 0
	}
	return float64(numerator) / float64(denominator)
}
