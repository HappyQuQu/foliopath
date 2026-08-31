package semantic

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"
)

type searchSnapshotStub struct {
	snapshot SearchSnapshot
	err      error
}

func (stub *searchSnapshotStub) GetSemanticSearchSnapshot(context.Context, SearchScope) (SearchSnapshot, error) {
	return stub.snapshot, stub.err
}

type textEncoderStub struct {
	generation string
	query      string
}

func (stub *textEncoderStub) EncodeSemanticText(_ context.Context, generationID, query string) ([]float32, error) {
	stub.generation, stub.query = generationID, query
	return []float32{1, 0}, nil
}

type vectorSearchStub struct {
	requests []VectorSearchRequest
	pages    [][]VectorMatch
}

func (stub *vectorSearchStub) SearchSemanticVectors(_ context.Context, request VectorSearchRequest) ([]VectorMatch, error) {
	stub.requests = append(stub.requests, request)
	index := len(stub.requests) - 1
	if index >= len(stub.pages) {
		return nil, nil
	}
	return stub.pages[index], nil
}

func TestSearchServiceCanonicalizesQueryAndBindsOpaqueCursor(t *testing.T) {
	snapshots := &searchSnapshotStub{snapshot: SearchSnapshot{
		GenerationID: "generation_1", CatalogRevision: 7,
		Members: []SearchSnapshotMember{{
			LibraryID: 3, LibraryGeneration: 9, SettingsRevision: 2,
			Coverage: Coverage{Eligible: 4, Completed: 3, Failed: 1, Revision: 5},
		}},
	}}
	encoder := &textEncoderStub{}
	vectors := &vectorSearchStub{pages: [][]VectorMatch{
		{{LibraryID: 3, AssetID: 10, Score: 0.9}, {LibraryID: 3, AssetID: 11, Score: 0.8}},
		{{LibraryID: 3, AssetID: 12, Score: 0.7}},
	}}
	service, err := NewSearchService(snapshots, vectors, encoder, []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	first, err := service.Search(context.Background(), SearchRequest{
		Query: "  Red,\tDRESS!  ", LibraryID: 3, DirectoryID: 8, Recursive: true, Limit: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if encoder.generation != "generation_1" || encoder.query != "red dress" {
		t.Fatalf("encoder received generation=%q query=%q", encoder.generation, encoder.query)
	}
	if first.NextCursor == "" || strings.Contains(first.NextCursor, "red") || first.Coverage.Eligible != 4 || first.Coverage.Completed != 3 {
		t.Fatalf("first result = %#v", first)
	}
	second, err := service.Search(context.Background(), SearchRequest{
		Query: "red dress", LibraryID: 3, DirectoryID: 8, Recursive: true, Cursor: first.NextCursor, Limit: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Matches) != 1 || len(vectors.requests) != 2 || vectors.requests[1].After == nil ||
		vectors.requests[1].After.AssetID != 11 || vectors.requests[1].After.Score != 0.8 {
		t.Fatalf("second result=%#v request=%#v", second, vectors.requests[1])
	}
}

func TestSearchServiceRejectsTokenizerControlLiteralsBeforeDependencies(t *testing.T) {
	encoder := &textEncoderStub{}
	vectors := &vectorSearchStub{}
	service, err := NewSearchService(&searchSnapshotStub{}, vectors, encoder, []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	for _, query := range []string{"portrait </s>", "blue <UNK> hair"} {
		if _, err := service.Search(context.Background(), SearchRequest{Query: query, Limit: 1}); !errors.Is(err, ErrInvalidQuery) {
			t.Fatalf("query %q error = %v", query, err)
		}
	}
	if encoder.query != "" || len(vectors.requests) != 0 {
		t.Fatalf("invalid control query reached dependencies: encoder=%q vectors=%d", encoder.query, len(vectors.requests))
	}
}

func TestSearchServiceRejectsTamperedMismatchedAndStaleCursors(t *testing.T) {
	snapshots := &searchSnapshotStub{snapshot: SearchSnapshot{
		GenerationID: "generation_1", CatalogRevision: 7,
		Members: []SearchSnapshotMember{{LibraryID: 3, LibraryGeneration: 9, SettingsRevision: 2, Coverage: Coverage{Revision: 1}}},
	}}
	vectors := &vectorSearchStub{pages: [][]VectorMatch{{{LibraryID: 3, AssetID: 10, Score: 0.9}}}}
	service, err := NewSearchService(snapshots, vectors, &textEncoderStub{}, []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	first, err := service.Search(context.Background(), SearchRequest{Query: "portrait", LibraryID: 3, Limit: 1})
	if err != nil || first.NextCursor == "" {
		t.Fatalf("first=%#v err=%v", first, err)
	}
	tamperAt := len(first.NextCursor) / 2
	replacement := byte('A')
	if first.NextCursor[tamperAt] == replacement {
		replacement = 'B'
	}
	tampered := first.NextCursor[:tamperAt] + string(replacement) + first.NextCursor[tamperAt+1:]
	if _, err := service.Search(context.Background(), SearchRequest{Query: "portrait", LibraryID: 3, Cursor: tampered, Limit: 1}); !errors.Is(err, ErrInvalidSemanticCursor) {
		t.Fatalf("tampered cursor error = %v", err)
	}
	if _, err := service.Search(context.Background(), SearchRequest{Query: "landscape", LibraryID: 3, Cursor: first.NextCursor, Limit: 1}); !errors.Is(err, ErrInvalidSemanticCursor) {
		t.Fatalf("query mismatch error = %v", err)
	}
	snapshots.snapshot.Members[0].Coverage.Revision++
	if _, err := service.Search(context.Background(), SearchRequest{Query: "portrait", LibraryID: 3, Cursor: first.NextCursor, Limit: 1}); !errors.Is(err, ErrSemanticCursorStale) {
		t.Fatalf("stale cursor error = %v", err)
	}
}

func TestSearchServiceRejectsOversizedCursorBeforeDecode(t *testing.T) {
	encoder := &textEncoderStub{}
	vectors := &vectorSearchStub{}
	service, err := NewSearchService(&searchSnapshotStub{snapshot: SearchSnapshot{
		GenerationID: "generation_1", CatalogRevision: 7,
		Members: []SearchSnapshotMember{{LibraryID: 3, SettingsRevision: 1, Coverage: Coverage{Revision: 1}}},
	}}, vectors, encoder, []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.Search(context.Background(), SearchRequest{
		Query: "portrait", LibraryID: 3, Cursor: strings.Repeat("A", MaxSemanticCursorBytes+1), Limit: 1,
	})
	if !errors.Is(err, ErrInvalidSemanticCursor) {
		t.Fatalf("error = %v, want invalid semantic cursor", err)
	}
	if encoder.query != "" || len(vectors.requests) != 0 {
		t.Fatalf("oversized cursor reached encoder/vector dependencies")
	}
}

func TestSearchServiceRejectsInvalidRepositoryPages(t *testing.T) {
	t.Parallel()
	snapshot := SearchSnapshot{
		GenerationID: "generation_1", CatalogRevision: 7,
		Members: []SearchSnapshotMember{
			{LibraryID: 3, SettingsRevision: 1, Coverage: Coverage{Revision: 1}},
			{LibraryID: 4, SettingsRevision: 1, Coverage: Coverage{Revision: 1}},
		},
	}
	for _, test := range []struct {
		name    string
		limit   int
		matches []VectorMatch
	}{
		{name: "over limit", limit: 1, matches: []VectorMatch{{LibraryID: 3, AssetID: 1, Score: .9}, {LibraryID: 3, AssetID: 2, Score: .8}}},
		{name: "unknown library", limit: 2, matches: []VectorMatch{{LibraryID: 9, AssetID: 1, Score: .9}}},
		{name: "wrong selected library", limit: 2, matches: []VectorMatch{{LibraryID: 4, AssetID: 1, Score: .9}}},
		{name: "invalid asset", limit: 2, matches: []VectorMatch{{LibraryID: 3, AssetID: 0, Score: .9}}},
		{name: "non finite score", limit: 2, matches: []VectorMatch{{LibraryID: 3, AssetID: 1, Score: float32(math.NaN())}}},
		{name: "score ascending", limit: 2, matches: []VectorMatch{{LibraryID: 3, AssetID: 1, Score: .8}, {LibraryID: 3, AssetID: 2, Score: .9}}},
		{name: "tie id descending", limit: 2, matches: []VectorMatch{{LibraryID: 3, AssetID: 2, Score: .9}, {LibraryID: 3, AssetID: 1, Score: .9}}},
		{name: "duplicate asset", limit: 2, matches: []VectorMatch{{LibraryID: 3, AssetID: 1, Score: .9}, {LibraryID: 3, AssetID: 1, Score: .8}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			vectors := &vectorSearchStub{pages: [][]VectorMatch{test.matches}}
			service, err := NewSearchService(&searchSnapshotStub{snapshot: snapshot}, vectors, &textEncoderStub{}, []byte("0123456789abcdef0123456789abcdef"))
			if err != nil {
				t.Fatal(err)
			}
			_, err = service.Search(context.Background(), SearchRequest{Query: "portrait", LibraryID: 3, Limit: test.limit})
			if !errors.Is(err, ErrInvalidEmbeddingRecord) {
				t.Fatalf("error = %v, want invalid embedding record", err)
			}
		})
	}
}

func TestSearchServiceRejectsRepositoryPageBeforeCursorPosition(t *testing.T) {
	snapshots := &searchSnapshotStub{snapshot: SearchSnapshot{
		GenerationID: "generation_1", CatalogRevision: 7,
		Members: []SearchSnapshotMember{{LibraryID: 3, SettingsRevision: 1, Coverage: Coverage{Revision: 1}}},
	}}
	vectors := &vectorSearchStub{pages: [][]VectorMatch{
		{{LibraryID: 3, AssetID: 10, Score: .9}},
		{{LibraryID: 3, AssetID: 9, Score: .9}},
	}}
	service, err := NewSearchService(snapshots, vectors, &textEncoderStub{}, []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	first, err := service.Search(context.Background(), SearchRequest{Query: "portrait", LibraryID: 3, Limit: 1})
	if err != nil || first.NextCursor == "" {
		t.Fatalf("first=%#v err=%v", first, err)
	}
	_, err = service.Search(context.Background(), SearchRequest{Query: "portrait", LibraryID: 3, Cursor: first.NextCursor, Limit: 1})
	if !errors.Is(err, ErrInvalidEmbeddingRecord) {
		t.Fatalf("error = %v, want invalid embedding record", err)
	}
}

func TestSemanticSearchSnapshotFingerprintRejectsUnorderedMembers(t *testing.T) {
	_, _, err := SemanticSearchSnapshotFingerprint(SearchSnapshot{
		GenerationID: "generation_1", CatalogRevision: 1,
		Members: []SearchSnapshotMember{
			{LibraryID: 2, SettingsRevision: 1, Coverage: Coverage{Revision: 1}},
			{LibraryID: 1, SettingsRevision: 1, Coverage: Coverage{Revision: 1}},
		},
	})
	if !errors.Is(err, ErrInvalidSemanticSnapshot) {
		t.Fatalf("unordered snapshot error = %v", err)
	}
}

func TestSemanticSearchSnapshotFingerprintRejectsInternalCorruption(t *testing.T) {
	t.Parallel()
	validMember := SearchSnapshotMember{LibraryID: 2, SettingsRevision: 1, Coverage: Coverage{Eligible: 2, Completed: 1, Revision: 1}}
	for _, snapshot := range []SearchSnapshot{
		{GenerationID: strings.Repeat("g", 129), CatalogRevision: 1, Members: []SearchSnapshotMember{validMember}},
		{GenerationID: "generation_1", CatalogRevision: 1, Members: []SearchSnapshotMember{{LibraryID: 2, SettingsRevision: 1, Coverage: Coverage{Eligible: 1, Completed: 1, Failed: 1, Revision: 1}}}},
		{GenerationID: "generation_1", CatalogRevision: 1, Members: []SearchSnapshotMember{validMember}, Excluded: []SearchExclusion{{LibraryID: 2, SettingsRevision: 1, Reason: "disabled"}}},
		{GenerationID: "generation_1", CatalogRevision: 1, Members: []SearchSnapshotMember{
			{LibraryID: 1, SettingsRevision: 1, Coverage: Coverage{Eligible: math.MaxInt64, Revision: 1}},
			{LibraryID: 2, SettingsRevision: 1, Coverage: Coverage{Eligible: 1, Revision: 1}},
		}},
		{GenerationID: "generation_1", CatalogRevision: 1, Members: []SearchSnapshotMember{
			{LibraryID: 1, SettingsRevision: 1, Coverage: Coverage{Revision: math.MaxInt64}},
		}},
	} {
		if _, _, err := SemanticSearchSnapshotFingerprint(snapshot); !errors.Is(err, ErrInvalidSemanticSnapshot) {
			t.Fatalf("snapshot=%#v error=%v, want invalid semantic snapshot", snapshot, err)
		}
	}
}
