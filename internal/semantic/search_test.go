package semantic

import "testing"

func TestBoundedVectorMatchesUsesScoreDescendingAssetAscending(t *testing.T) {
	values, err := BoundedVectorMatches(3, func(offer func(VectorMatch) error) error {
		for _, value := range []VectorMatch{{LibraryID: 1, AssetID: 9, Score: .5}, {LibraryID: 1, AssetID: 3, Score: .5}, {LibraryID: 1, AssetID: 7, Score: .9}, {LibraryID: 1, AssetID: 1, Score: -.2}} {
			if err := offer(value); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil || len(values) != 3 || values[0].AssetID != 7 || values[1].AssetID != 3 || values[2].AssetID != 9 {
		t.Fatalf("matches = %#v, err %v", values, err)
	}
}
