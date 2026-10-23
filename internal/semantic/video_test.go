package semantic

import "testing"

func TestBestVideoMatchesRequiresCompletePlanAndUsesStableTies(t *testing.T) {
	var candidates []VideoVectorCandidate
	for ordinal := 0; ordinal < 4; ordinal++ {
		candidates = append(candidates, VideoVectorCandidate{LibraryID: 1, AssetID: 9, PlanSize: 4, Ordinal: ordinal, TimestampMS: int64(ordinal * 1000), Score: .5})
	}
	for ordinal := 0; ordinal < 3; ordinal++ {
		candidates = append(candidates, VideoVectorCandidate{LibraryID: 1, AssetID: 3, PlanSize: 4, Ordinal: ordinal, TimestampMS: int64(ordinal * 1000), Score: .9})
	}
	for ordinal := 0; ordinal < 4; ordinal++ {
		score := float32(.5)
		candidates = append(candidates, VideoVectorCandidate{LibraryID: 1, AssetID: 2, PlanSize: 4, Ordinal: ordinal, TimestampMS: int64(ordinal * 1000), Score: score})
	}
	items, err := BestVideoMatches(candidates, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].AssetID != 2 || items[1].AssetID != 9 || items[0].Ordinal != 0 {
		t.Fatalf("matches = %#v", items)
	}
}
