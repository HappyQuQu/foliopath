package curation

import (
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"
)

func TestAutomaticTagPlanNormalizesDeduplicatesWithoutMutatingInput(t *testing.T) {
	input := AutomaticTagPlan{LibraryID: 1, AssetID: 2, SourceFingerprint: "v1:10:20", ModelIdentity: "model:test-v1", Labels: []AutomaticLabel{
		{Name: " Cafe\u0301  Trip ", Confidence: .3}, {Name: "café trip", Confidence: .8},
		{Name: "zebra", Confidence: .6}, {Name: "apple", Confidence: .6},
	}}
	before := append([]AutomaticLabel(nil), input.Labels...)
	got, err := NormalizeAutomaticTagPlan(input)
	if err != nil || !reflect.DeepEqual(got.Labels, []AutomaticLabel{{Name: "café trip", Confidence: .8}, {Name: "apple", Confidence: .6}, {Name: "zebra", Confidence: .6}}) {
		t.Fatalf("plan=%+v, err=%v", got, err)
	}
	if !reflect.DeepEqual(input.Labels, before) {
		t.Fatal("normalization mutated caller labels")
	}
	again, err := NormalizeAutomaticTagPlan(got)
	if err != nil || !reflect.DeepEqual(again, got) {
		t.Fatalf("normalization is not idempotent: %+v, %v", again, err)
	}
	input.Labels = nil
	if _, err := NormalizeAutomaticTagPlan(input); err != nil {
		t.Fatalf("successful zero-label plan rejected: %v", err)
	}
}

func TestAutomaticTagPlanRejectsUnboundedOrInvalidInference(t *testing.T) {
	valid := AutomaticTagPlan{LibraryID: 1, AssetID: 2, SourceFingerprint: "v1:10:20", ModelIdentity: "model:test-v1", Labels: []AutomaticLabel{{Name: "tree", Confidence: .8}}}
	for _, test := range []struct {
		name   string
		change func(*AutomaticTagPlan)
	}{
		{"missing library", func(p *AutomaticTagPlan) { p.LibraryID = 0 }},
		{"missing asset", func(p *AutomaticTagPlan) { p.AssetID = 0 }},
		{"missing fingerprint", func(p *AutomaticTagPlan) { p.SourceFingerprint = "" }},
		{"oversize identity", func(p *AutomaticTagPlan) { p.ModelIdentity = strings.Repeat("m", 257) }},
		{"control identity", func(p *AutomaticTagPlan) { p.ModelIdentity = "model\nsecret" }},
		{"blank identity", func(p *AutomaticTagPlan) { p.ModelIdentity = " " }},
		{"too many labels", func(p *AutomaticTagPlan) { p.Labels = make([]AutomaticLabel, MaxTagsPerAsset+1) }},
		{"control label", func(p *AutomaticTagPlan) { p.Labels[0].Name = "tree\n" }},
		{"oversize label", func(p *AutomaticTagPlan) { p.Labels[0].Name = strings.Repeat("界", MaxTagRunes+1) }},
		{"nan score", func(p *AutomaticTagPlan) { p.Labels[0].Confidence = math.NaN() }},
		{"infinite score", func(p *AutomaticTagPlan) { p.Labels[0].Confidence = math.Inf(1) }},
		{"negative score", func(p *AutomaticTagPlan) { p.Labels[0].Confidence = -.1 }},
		{"oversize score", func(p *AutomaticTagPlan) { p.Labels[0].Confidence = 1.1 }},
	} {
		t.Run(test.name, func(t *testing.T) {
			plan := valid
			plan.Labels = append([]AutomaticLabel(nil), valid.Labels...)
			test.change(&plan)
			if _, err := NormalizeAutomaticTagPlan(plan); !errors.Is(err, ErrInvalidRequest) {
				t.Fatalf("error=%v", err)
			}
		})
	}
}
