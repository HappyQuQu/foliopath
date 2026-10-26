package face

import (
	"encoding/json"
	"strings"
	"testing"
)

func validThresholdProfile() ThresholdProfile {
	return ThresholdProfile{
		SchemaVersion: 1, ProfileID: "face-reviewed-v1", CoreSimilarity: .75,
		EdgeSimilarity: .65, MinCoreSize: 2, QualitySummarySHA256: strings.Repeat("a", 64),
	}
}

func TestThresholdProfileAcceptsOnlyBoundedGovernedClusteringValues(t *testing.T) {
	encoded, err := json.Marshal(validThresholdProfile())
	if err != nil {
		t.Fatal(err)
	}
	profile, err := ParseThresholdProfile(encoded)
	if err != nil || profile.ProfileID != "face-reviewed-v1" || profile.CoreSimilarity != .75 {
		t.Fatalf("profile=%+v err=%v", profile, err)
	}
	for name, mutate := range map[string]func(*ThresholdProfile){
		"schema":       func(value *ThresholdProfile) { value.SchemaVersion = 2 },
		"profile path": func(value *ThresholdProfile) { value.ProfileID = "../face" },
		"core":         func(value *ThresholdProfile) { value.CoreSimilarity = 1.1 },
		"edge":         func(value *ThresholdProfile) { value.EdgeSimilarity = .8 },
		"minimum":      func(value *ThresholdProfile) { value.MinCoreSize = 1 },
		"quality hash": func(value *ThresholdProfile) { value.QualitySummarySHA256 = "pending" },
	} {
		t.Run(name, func(t *testing.T) {
			value := validThresholdProfile()
			mutate(&value)
			encoded, _ := json.Marshal(value)
			if _, err := ParseThresholdProfile(encoded); err == nil {
				t.Fatal("invalid threshold profile accepted")
			}
		})
	}
}

func TestThresholdProfileRejectsUnknownDuplicateTrailingAndOversizedJSON(t *testing.T) {
	encoded, _ := json.Marshal(validThresholdProfile())
	unknown := append(encoded[:len(encoded)-1], []byte(`,"groupAssignmentAllowed":true}`)...)
	duplicate := []byte(`{"schemaVersion":1,"schemaVersion":1}`)
	trailing := append(encoded, []byte(` {}`)...)
	oversized := make([]byte, MaxThresholdProfileBytes+1)
	for _, value := range [][]byte{unknown, duplicate, trailing, oversized} {
		if _, err := ParseThresholdProfile(value); err == nil {
			t.Fatal("hostile threshold profile accepted")
		}
	}
}
