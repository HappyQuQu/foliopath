package curation

import (
	"errors"
	"math"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

type TagSource string

const (
	TagSourceManual TagSource = "manual"
	TagSourceAI     TagSource = "ai"
)

var ErrAutomaticTagSourceChanged = errors.New("automatic tag source changed")

// AutomaticTagPlan is a bounded, single-image result. Its queue owner must fence
// cancellation, enabled settings, the claimed job and model identity in the same
// transaction that applies this plan and advances its checkpoint (ADR-0021).
type AutomaticTagPlan struct {
	LibraryID         int64
	AssetID           int64
	SourceFingerprint string
	ModelIdentity     string
	Labels            []AutomaticLabel
}

type AutomaticLabel struct {
	Name       string
	Confidence float64
}

type AutomaticTagMergeResult struct {
	Added             int
	Removed           int
	SkippedSuppressed int
	SkippedLimit      int
}

// NormalizeAutomaticTagPlan owns normalization and deterministic selection. An
// empty successful result is valid; failures must never be encoded as empty plans.
// Confidence is a model score, not a calibrated probability.
func NormalizeAutomaticTagPlan(plan AutomaticTagPlan) (AutomaticTagPlan, error) {
	if plan.LibraryID < 1 || plan.AssetID < 1 || !validAutomaticIdentity(plan.SourceFingerprint) || !validAutomaticIdentity(plan.ModelIdentity) || len(plan.Labels) > MaxTagsPerAsset {
		return AutomaticTagPlan{}, ErrInvalidRequest
	}
	byKey := make(map[string]AutomaticLabel, len(plan.Labels))
	for _, label := range plan.Labels {
		name, key, err := NormalizeTagName(label.Name)
		if err != nil || math.IsNaN(label.Confidence) || math.IsInf(label.Confidence, 0) || label.Confidence < 0 || label.Confidence > 1 {
			return AutomaticTagPlan{}, ErrInvalidRequest
		}
		label.Name = name
		current, found := byKey[key]
		if !found || label.Confidence > current.Confidence || label.Confidence == current.Confidence && label.Name < current.Name {
			byKey[key] = label
		}
	}
	keys := make([]string, 0, len(byKey))
	for key := range byKey {
		keys = append(keys, key)
	}
	slices.SortFunc(keys, func(a, b string) int {
		if byKey[a].Confidence > byKey[b].Confidence {
			return -1
		}
		if byKey[a].Confidence < byKey[b].Confidence {
			return 1
		}
		return strings.Compare(a, b)
	})
	plan.Labels = make([]AutomaticLabel, 0, len(keys))
	for _, key := range keys {
		plan.Labels = append(plan.Labels, byKey[key])
	}
	return plan, nil
}

func validAutomaticIdentity(value string) bool {
	if value == "" || len(value) > 256 || !utf8.ValidString(value) || strings.TrimSpace(value) != value {
		return false
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}
