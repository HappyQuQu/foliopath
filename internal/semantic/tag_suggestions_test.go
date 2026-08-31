package semantic

import (
	"errors"
	"math"
	"testing"
)

func TestRankControlledTagSuggestionsIsBoundedStableAndFinite(t *testing.T) {
	asset := []float32{1, 0}
	tags := map[int64][]float32{
		9: {1, 0},
		3: {1, 0},
		4: {0.8, 0.2},
		5: {0.7, 0.3},
		6: {0.6, 0.4},
		7: {0.5, 0.5},
	}
	items, err := RankControlledTagSuggestions(asset, tags, 0, MaxTagSuggestionsPerAsset)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != MaxTagSuggestionsPerAsset {
		t.Fatalf("items = %d", len(items))
	}
	if items[0].TagID != 3 || items[1].TagID != 9 {
		t.Fatalf("tie order = %d, %d", items[0].TagID, items[1].TagID)
	}
	for _, item := range items {
		if item.Confidence < 0 || item.Confidence > 1 || math.IsNaN(float64(item.Confidence)) || math.IsInf(float64(item.Confidence), 0) {
			t.Fatalf("invalid confidence %#v", item)
		}
	}
}

func TestRankControlledTagSuggestionsRejectsUntrustedVectors(t *testing.T) {
	for _, tags := range []map[int64][]float32{
		{0: {1, 0}},
		{1: {1}},
		{1: {float32(math.NaN()), 0}},
	} {
		if _, err := RankControlledTagSuggestions([]float32{1, 0}, tags, 0, 5); !errors.Is(err, ErrInvalidTagSuggestion) {
			t.Fatalf("error = %v", err)
		}
	}
}
