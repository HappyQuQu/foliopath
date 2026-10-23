package semantic

import (
	"context"
	"testing"
)

type videoSnapshotStub struct{ snapshot VideoSearchSnapshot }

func (stub *videoSnapshotStub) GetVideoSemanticSearchSnapshot(context.Context, SearchScope) (VideoSearchSnapshot, error) {
	return stub.snapshot, nil
}

type videoVectorStub struct {
	requests []VideoVectorSearchRequest
	pages    [][]VideoVectorMatch
}

func (stub *videoVectorStub) SearchVideoSemanticVectors(_ context.Context, request VideoVectorSearchRequest) ([]VideoVectorMatch, error) {
	stub.requests = append(stub.requests, request)
	index := len(stub.requests) - 1
	if index >= len(stub.pages) {
		return nil, nil
	}
	return stub.pages[index], nil
}

func TestVideoSearchServiceBindsBestFrameCursorAndCoverage(t *testing.T) {
	snapshots := &videoSnapshotStub{snapshot: VideoSearchSnapshot{GenerationID: "generation_1", CatalogRevision: 7,
		Members: []VideoSearchSnapshotMember{{LibraryID: 3, LibraryGeneration: 2, SettingsRevision: 4,
			Coverage: VideoCoverage{Eligible: 4, Ready: 3, Degraded: 1, Revision: 8}}}}}
	vectors := &videoVectorStub{pages: [][]VideoVectorMatch{
		{{LibraryID: 3, AssetID: 10, PlanSize: 10, Ordinal: 3, TimestampMS: 3000, Score: .9},
			{LibraryID: 3, AssetID: 11, PlanSize: 4, Ordinal: 1, TimestampMS: 1000, Score: .8}},
		{{LibraryID: 3, AssetID: 12, PlanSize: 4, Ordinal: 0, TimestampMS: 500, Score: .7}},
	}}
	service, err := NewVideoSearchService(snapshots, vectors, &textEncoderStub{}, []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	first, err := service.Search(context.Background(), SearchRequest{Query: "red dress", LibraryID: 3, Limit: 2})
	if err != nil || first.NextCursor == "" || first.Coverage.Ready != 3 || first.Coverage.Complete() {
		t.Fatalf("first = %#v, %v", first, err)
	}
	second, err := service.Search(context.Background(), SearchRequest{Query: "red dress", LibraryID: 3, Cursor: first.NextCursor, Limit: 2})
	if err != nil || len(second.Matches) != 1 || vectors.requests[1].After == nil ||
		vectors.requests[1].After.AssetID != 11 || vectors.requests[1].After.Score != .8 {
		t.Fatalf("second = %#v request=%#v err=%v", second, vectors.requests[1], err)
	}
}
