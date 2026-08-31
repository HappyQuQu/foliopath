package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
)

const (
	minimumVideoCount     = 100
	minimumVideoDuration  = 2_000
	maximumVideoDuration  = 2 * 60 * 60 * 1_000
	minimumVideoTop20Rate = 0.80
)

type S2BQualityDataset struct {
	SchemaVersion         int                     `json:"schema_version"`
	DatasetID             string                  `json:"dataset_id"`
	DatasetManifestSHA256 string                  `json:"dataset_manifest_sha256"`
	ModelPackageSHA256    string                  `json:"model_package_sha256"`
	Approvals             S2BQualityApprovals     `json:"approvals"`
	TagGate               TagQualityGate          `json:"tag_gate"`
	Tags                  []TagQualityObservation `json:"tags"`
	Videos                []VideoQualityItem      `json:"videos"`
	VideoQueries          []VideoQualityQuery     `json:"video_queries"`
}

type S2BQualityApprovals struct {
	Product string `json:"product"`
	ML      string `json:"ml"`
	QA      string `json:"qa"`
}

type TagQualityGate struct {
	MinimumMacroPrecision float64 `json:"minimum_macro_precision"`
	MinimumMacroRecall    float64 `json:"minimum_macro_recall"`
	MinimumAcceptanceRate float64 `json:"minimum_acceptance_rate"`
}

type TagQualityObservation struct {
	TagID         string `json:"tag_id"`
	TruePositive  int    `json:"true_positive"`
	FalsePositive int    `json:"false_positive"`
	FalseNegative int    `json:"false_negative"`
	Accepted      int    `json:"accepted"`
	Reviewed      int    `json:"reviewed"`
}

type VideoQualityItem struct {
	ID         string `json:"id"`
	Format     string `json:"format"`
	DurationMS int64  `json:"duration_ms"`
	FramePlan  int    `json:"frame_plan"`
	Motion     string `json:"motion"`
	Setting    string `json:"setting"`
}

type VideoQualityQuery struct {
	ID               string   `json:"id"`
	Language         string   `json:"language"`
	RelevantVideoIDs []string `json:"relevant_video_ids"`
	Top20VideoIDs    []string `json:"top20_video_ids"`
}

type TagQualityMetric struct {
	TagID     string  `json:"tag_id"`
	Precision float64 `json:"precision"`
	Recall    float64 `json:"recall"`
}

type S2BQualityReport struct {
	SchemaVersion         int                 `json:"schema_version"`
	SourceCommit          string              `json:"source_commit,omitempty"`
	DatasetID             string              `json:"dataset_id"`
	DatasetManifestSHA256 string              `json:"dataset_manifest_sha256"`
	ModelPackageDigest    string              `json:"model_package_digest"`
	Approvals             S2BQualityApprovals `json:"approvals"`
	TagMetrics            []TagQualityMetric  `json:"tag_metrics"`
	TagMacroPrecision     float64             `json:"tag_macro_precision"`
	TagMacroRecall        float64             `json:"tag_macro_recall"`
	TagAcceptanceRate     float64             `json:"tag_acceptance_rate"`
	VideoCount            int                 `json:"video_count"`
	VideoQueryCount       int                 `json:"video_query_count"`
	VideoTop20Success     float64             `json:"video_top20_success"`
	GatePass              bool                `json:"gate_pass"`
	GateFailures          []string            `json:"gate_failures"`
}

func ReadS2BQualityDataset(filename string) (S2BQualityDataset, error) {
	var dataset S2BQualityDataset
	if err := decodeStrict(filename, &dataset); err != nil {
		return dataset, err
	}
	return dataset, dataset.Validate()
}

func ValidateS2BQualityManifest(quality S2BQualityDataset, filename string) error {
	content, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("read governed dataset manifest: %w", err)
	}
	digest := sha256.Sum256(content)
	if hex.EncodeToString(digest[:]) != quality.DatasetManifestSHA256 {
		return errors.New("governed dataset manifest SHA-256 mismatch")
	}
	manifest, err := ReadDatasetManifest(filename)
	if err != nil {
		return fmt.Errorf("validate governed dataset manifest: %w", err)
	}
	if manifest.SchemaVersion != 2 || manifest.LegalBasis == "synthetic" || manifest.Governance == nil ||
		manifest.Governance.DataClass != "ordinary-media" {
		return errors.New("S2B quality requires a non-synthetic schema v2 ordinary-media manifest")
	}
	if !containsString(manifest.Governance.AllowedUses, "semantic-evaluation") ||
		!containsString(manifest.Governance.AllowedUses, "video-evaluation") {
		return errors.New("S2B quality manifest requires semantic-evaluation and video-evaluation uses")
	}
	manifestVideos := make(map[string]struct{}, len(manifest.Items))
	for _, item := range manifest.Items {
		if item.MediaType == "video" {
			manifestVideos[item.ID] = struct{}{}
		}
	}
	if len(manifestVideos) < minimumVideoCount {
		return fmt.Errorf("governed manifest contains %d videos, want at least %d", len(manifestVideos), minimumVideoCount)
	}
	for _, video := range quality.Videos {
		if _, exists := manifestVideos[video.ID]; !exists {
			return fmt.Errorf("quality video %q is absent from governed manifest", video.ID)
		}
	}
	return nil
}

func (dataset S2BQualityDataset) Validate() error {
	if dataset.SchemaVersion != 1 || dataset.DatasetID == "" {
		return errors.New("S2B quality dataset requires schema_version 1 and dataset_id")
	}
	if !sha256Pattern.MatchString(dataset.DatasetManifestSHA256) ||
		!sha256Pattern.MatchString(dataset.ModelPackageSHA256) {
		return errors.New("S2B quality dataset requires manifest and model package SHA-256")
	}
	for name, reference := range map[string]string{
		"product": dataset.Approvals.Product,
		"ml":      dataset.Approvals.ML,
		"qa":      dataset.Approvals.QA,
	} {
		if !governanceReferencePattern.MatchString(reference) {
			return fmt.Errorf("%s approval must be an opaque safe reference", name)
		}
	}
	for name, threshold := range map[string]float64{
		"minimum_macro_precision": dataset.TagGate.MinimumMacroPrecision,
		"minimum_macro_recall":    dataset.TagGate.MinimumMacroRecall,
		"minimum_acceptance_rate": dataset.TagGate.MinimumAcceptanceRate,
	} {
		if threshold <= 0 || threshold > 1 {
			return fmt.Errorf("%s must be in (0, 1]", name)
		}
	}
	if len(dataset.Tags) == 0 {
		return errors.New("S2B quality dataset requires tag observations")
	}
	seenTags := map[string]struct{}{}
	for _, tag := range dataset.Tags {
		if tag.TagID == "" || tag.TruePositive < 0 || tag.FalsePositive < 0 ||
			tag.FalseNegative < 0 || tag.Accepted < 0 || tag.Reviewed <= 0 || tag.Accepted > tag.Reviewed {
			return fmt.Errorf("tag %q has invalid counts", tag.TagID)
		}
		if tag.TruePositive+tag.FalsePositive == 0 || tag.TruePositive+tag.FalseNegative == 0 {
			return fmt.Errorf("tag %q has undefined precision or recall", tag.TagID)
		}
		if _, exists := seenTags[tag.TagID]; exists {
			return fmt.Errorf("duplicate tag %q", tag.TagID)
		}
		seenTags[tag.TagID] = struct{}{}
	}
	if len(dataset.Videos) < minimumVideoCount {
		return fmt.Errorf("video quality set contains %d videos, want at least %d", len(dataset.Videos), minimumVideoCount)
	}
	seenVideos := map[string]struct{}{}
	coverage := map[string]bool{}
	for _, video := range dataset.Videos {
		if video.ID == "" || video.DurationMS < minimumVideoDuration || video.DurationMS > maximumVideoDuration {
			return fmt.Errorf("video %q has invalid ID or duration", video.ID)
		}
		if video.Format != "mp4" && video.Format != "mov" && video.Format != "mkv" {
			return fmt.Errorf("video %q has unsupported format %q", video.ID, video.Format)
		}
		if video.FramePlan != 4 && video.FramePlan != 10 {
			return fmt.Errorf("video %q has unsupported frame plan %d", video.ID, video.FramePlan)
		}
		if video.Motion != "motion" && video.Motion != "static" {
			return fmt.Errorf("video %q has unsupported motion class %q", video.ID, video.Motion)
		}
		if video.Setting != "indoor" && video.Setting != "outdoor" {
			return fmt.Errorf("video %q has unsupported setting %q", video.ID, video.Setting)
		}
		if _, exists := seenVideos[video.ID]; exists {
			return fmt.Errorf("duplicate video %q", video.ID)
		}
		seenVideos[video.ID] = struct{}{}
		coverage["format/"+video.Format] = true
		coverage[fmt.Sprintf("plan/%d", video.FramePlan)] = true
		coverage["motion/"+video.Motion] = true
		coverage["setting/"+video.Setting] = true
	}
	for _, required := range []string{
		"format/mp4", "format/mov", "format/mkv", "plan/4", "plan/10",
		"motion/motion", "motion/static", "setting/indoor", "setting/outdoor",
	} {
		if !coverage[required] {
			return fmt.Errorf("video quality set lacks %s coverage", required)
		}
	}
	if len(dataset.VideoQueries) == 0 {
		return errors.New("S2B quality dataset requires video queries")
	}
	seenQueries := map[string]struct{}{}
	languages := map[string]bool{}
	for _, query := range dataset.VideoQueries {
		if query.ID == "" || (query.Language != "zh" && query.Language != "en") ||
			len(query.RelevantVideoIDs) == 0 || len(query.Top20VideoIDs) == 0 || len(query.Top20VideoIDs) > 20 {
			return fmt.Errorf("video query %q has invalid identity, language, relevance, or Top-20", query.ID)
		}
		if _, exists := seenQueries[query.ID]; exists {
			return fmt.Errorf("duplicate video query %q", query.ID)
		}
		seenQueries[query.ID] = struct{}{}
		languages[query.Language] = true
		if err := validateVideoReferences(query.ID, query.RelevantVideoIDs, seenVideos); err != nil {
			return err
		}
		if err := validateVideoReferences(query.ID, query.Top20VideoIDs, seenVideos); err != nil {
			return err
		}
	}
	if !languages["zh"] || !languages["en"] {
		return errors.New("video queries require both zh and en coverage")
	}
	return nil
}

func validateVideoReferences(queryID string, references []string, videos map[string]struct{}) error {
	seen := map[string]struct{}{}
	for _, reference := range references {
		if _, exists := videos[reference]; !exists {
			return fmt.Errorf("video query %q references unknown video %q", queryID, reference)
		}
		if _, exists := seen[reference]; exists {
			return fmt.Errorf("video query %q repeats video %q", queryID, reference)
		}
		seen[reference] = struct{}{}
	}
	return nil
}

func ScoreS2BQuality(dataset S2BQualityDataset) (S2BQualityReport, error) {
	if err := dataset.Validate(); err != nil {
		return S2BQualityReport{}, err
	}
	report := S2BQualityReport{
		SchemaVersion: 1, DatasetID: dataset.DatasetID,
		DatasetManifestSHA256: dataset.DatasetManifestSHA256,
		ModelPackageDigest:    "sha256:" + dataset.ModelPackageSHA256,
		Approvals:             dataset.Approvals,
		TagMetrics:            make([]TagQualityMetric, 0, len(dataset.Tags)),
		VideoCount:            len(dataset.Videos), VideoQueryCount: len(dataset.VideoQueries),
	}
	var accepted, reviewed int
	for _, tag := range dataset.Tags {
		metric := TagQualityMetric{
			TagID:     tag.TagID,
			Precision: ratio(tag.TruePositive, tag.TruePositive+tag.FalsePositive),
			Recall:    ratio(tag.TruePositive, tag.TruePositive+tag.FalseNegative),
		}
		report.TagMetrics = append(report.TagMetrics, metric)
		report.TagMacroPrecision += metric.Precision
		report.TagMacroRecall += metric.Recall
		accepted += tag.Accepted
		reviewed += tag.Reviewed
	}
	report.TagMacroPrecision /= float64(len(report.TagMetrics))
	report.TagMacroRecall /= float64(len(report.TagMetrics))
	report.TagAcceptanceRate = ratio(accepted, reviewed)

	successes := 0
	for _, query := range dataset.VideoQueries {
		relevant := make(map[string]struct{}, len(query.RelevantVideoIDs))
		for _, videoID := range query.RelevantVideoIDs {
			relevant[videoID] = struct{}{}
		}
		for _, videoID := range query.Top20VideoIDs {
			if _, ok := relevant[videoID]; ok {
				successes++
				break
			}
		}
	}
	report.VideoTop20Success = ratio(successes, len(dataset.VideoQueries))
	if report.TagMacroPrecision < dataset.TagGate.MinimumMacroPrecision {
		report.GateFailures = append(report.GateFailures, "tag macro precision below approved threshold")
	}
	if report.TagMacroRecall < dataset.TagGate.MinimumMacroRecall {
		report.GateFailures = append(report.GateFailures, "tag macro recall below approved threshold")
	}
	if report.TagAcceptanceRate < dataset.TagGate.MinimumAcceptanceRate {
		report.GateFailures = append(report.GateFailures, "tag acceptance rate below approved threshold")
	}
	if report.VideoTop20Success < minimumVideoTop20Rate {
		report.GateFailures = append(report.GateFailures, "video Top-20 success below 80%")
	}
	report.GatePass = len(report.GateFailures) == 0
	return report, nil
}
