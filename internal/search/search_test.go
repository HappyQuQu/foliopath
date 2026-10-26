package search

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/catalog"
	"github.com/HappyQuQu/foliopath/internal/semantic"
)

type fakeRepo struct {
	candidates []Candidate
	revision   int64
	disabled   bool
	mutate     bool
}

func (f *fakeRepo) ResolveScope(context.Context, int64, int64) (catalog.Scope, error) {
	return catalog.Scope{}, nil
}
func (f *fakeRepo) ResolveGlobalCatalogRevision(context.Context) (int64, error) {
	return f.revision, nil
}
func (f *fakeRepo) GetSemanticSearchSnapshot(context.Context, semantic.SearchScope) (semantic.SearchSnapshot, error) {
	if f.disabled {
		return semantic.SearchSnapshot{}, semantic.ErrSemanticDisabled
	}
	return semantic.SearchSnapshot{GenerationID: "generation-active", CatalogRevision: f.revision, Members: []semantic.SearchSnapshotMember{{LibraryID: 1, SettingsRevision: 1, Coverage: semantic.Coverage{Revision: 1}}}}, nil
}
func (f *fakeRepo) StreamSearchCandidates(ctx context.Context, s Scan, offer func(Candidate) error) error {
	for _, c := range f.candidates {
		if s.GenerationID == "" {
			c.Vector = nil
		}
		if e := offer(c); e != nil {
			return e
		}
	}
	if f.mutate {
		f.revision++
	}
	return ctx.Err()
}

type fakeEncoder struct {
	err   error
	calls int
}

func (f *fakeEncoder) EncodeSemanticText(context.Context, string, string) ([]float32, error) {
	f.calls++
	return []float32{1, 0}, f.err
}
func vector(t *testing.T, v []float32) []byte {
	t.Helper()
	b, e := semantic.EncodeEmbedding(v, 2)
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func TestFusionSourcesRankingAndPagination(t *testing.T) {
	repo := &fakeRepo{revision: 1, candidates: []Candidate{
		{"v1:test", 1, 1, "车.jpg", "旅行/车.jpg", vector(t, []float32{1, 0}), nil},
		{"v1:test", 1, 2, "photo.jpg", "车展/photo.jpg", nil, nil},
		{"v1:test", 1, 3, "other.jpg", "other.jpg", vector(t, []float32{1, 1}), nil},
		{"v1:test", 1, 4, "none.jpg", "none.jpg", vector(t, []float32{-1, 0}), nil},
		{"v1:test", 1, 5, "车2.jpg", "车展/车2.jpg", vector(t, []float32{1, 0}), nil},
	}}
	service, e := New(repo, &fakeEncoder{}, make([]byte, 32))
	if e != nil {
		t.Fatal(e)
	}
	r := Request{Query: "车", Mode: "all", Limit: 2}
	first, e := service.Search(context.Background(), r)
	if e != nil {
		t.Fatal(e)
	}
	if len(first.Hits) != 2 || first.Hits[0].AssetID != 1 || first.Hits[1].AssetID != 5 || first.NextCursor == "" {
		t.Fatalf("first=%+v", first)
	}
	if !reflect.DeepEqual(first.Hits[0].Sources, []Source{Filename, Visual}) || !reflect.DeepEqual(first.Hits[1].Sources, []Source{Filename, ParentPath, Visual}) {
		t.Fatalf("sources=%+v", first.Hits)
	}
	r.Cursor = first.NextCursor
	second, e := service.Search(context.Background(), r)
	if e != nil {
		t.Fatal(e)
	}
	if len(second.Hits) != 2 || second.Hits[0].AssetID != 2 || second.Hits[1].AssetID != 3 || second.NextCursor != "" {
		t.Fatalf("second=%+v", second)
	}
	if second.Hits[0].SemanticScore != nil || !reflect.DeepEqual(second.Hits[0].Sources, []Source{ParentPath}) {
		t.Fatalf("false visual source=%+v", second.Hits[0])
	}
	r.Query = "boat"
	if _, e = service.Search(context.Background(), r); !errors.Is(e, ErrCursor) {
		t.Fatalf("query cursor=%v", e)
	}
	r.Query = "车"
	repo.revision++
	if _, e = service.Search(context.Background(), r); !errors.Is(e, ErrStale) {
		t.Fatalf("stale=%v", e)
	}
}
func TestLiteralEvidenceUnicodeMixedFieldsAndNoBasenamePathDuplication(t *testing.T) {
	for _, tc := range []struct {
		query, name, path string
		want              []Source
	}{
		{"ＣＡＲ", "car.jpg", "album/car.jpg", []Source{Filename}},
		{"车 蓝", "蓝.jpg", "车展/蓝.jpg", []Source{Filename, ParentPath}},
		{"notfound 车", "车.jpg", "car/车.jpg", nil},
		{"straße", "STRASSE.jpg", "STRASSE.jpg", []Source{Filename}},
		{"%_", "image.jpg", "%_/image.jpg", []Source{ParentPath}},
	} {
		terms, e := catalog.NormalizeSearchTerms(tc.query)
		if e != nil {
			t.Fatal(e)
		}
		if got := LiteralSources(terms, tc.name, tc.path); !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("%q: %v want %v", tc.query, got, tc.want)
		}
	}
}
func TestCopiedRelativePathKeepsLiteralCompatibilityWithoutDuplicateBasenameEvidence(t *testing.T) {
	terms, _ := catalog.NormalizeSearchTerms("旅行/京都夜景.jpg")
	if got := LiteralSources(terms, "京都夜景.jpg", "旅行/京都夜景.jpg"); !reflect.DeepEqual(got, []string{"path"}) {
		t.Fatalf("copied path lost: %v", got)
	}
	terms, _ = catalog.NormalizeSearchTerms("京都夜景.jpg")
	if got := LiteralSources(terms, "京都夜景.jpg", "旅行/京都夜景.jpg"); !reflect.DeepEqual(got, []string{"filename"}) {
		t.Fatalf("basename attributed twice: %v", got)
	}
}

func TestDegradedSearchDoesNotInventSemanticOrMaskCorruption(t *testing.T) {
	repo := &fakeRepo{revision: 1, candidates: []Candidate{{"v1:test", 1, 1, "car.jpg", "car.jpg", nil, nil}}}
	encoder := &fakeEncoder{err: ErrUnavailable}
	service, _ := New(repo, encoder, make([]byte, 32))
	r := Request{Query: "car", Limit: 50}
	page, e := service.Search(context.Background(), r)
	if e != nil || page.SemanticState != "unavailable" || len(page.Hits) != 1 || page.Hits[0].SemanticScore != nil {
		t.Fatalf("page=%+v err=%v", page, e)
	}
	r.Mode = "semantic"
	if _, e = service.Search(context.Background(), r); !errors.Is(e, ErrUnavailable) {
		t.Fatalf("semantic must fail: %v", e)
	}
	repo.disabled = true
	r.Mode = "all"
	page, e = service.Search(context.Background(), r)
	if e != nil || page.SemanticState != "disabled" || len(page.Hits) != 1 {
		t.Fatalf("disabled=%+v %v", page, e)
	}
	repo.disabled = false
	encoder.err = semantic.ErrInvalidEmbedding
	if _, e = service.Search(context.Background(), r); !errors.Is(e, semantic.ErrInvalidEmbedding) {
		t.Fatalf("corruption hidden: %v", e)
	}
	for _, invalid := range []error{semantic.ErrInvalidEmbedding, semantic.ErrInvalidSpaceIdentity, semantic.ErrIncompatibleSpace} {
		encoder.err = errors.Join(semantic.ErrSemanticGenerationUnavailable, invalid)
		if _, e = service.Search(context.Background(), r); !errors.Is(e, invalid) {
			t.Fatalf("wrapped data/identity failure hidden: %v", e)
		}
	}
	r.Mode = "filename"
	calls := encoder.calls
	if _, e = service.Search(context.Background(), r); e != nil || calls != encoder.calls {
		t.Fatalf("filename called AI: %v", e)
	}
}
func TestSnapshotMutationCancellationAndInvalidRequests(t *testing.T) {
	repo := &fakeRepo{revision: 1, mutate: true}
	service, _ := New(repo, &fakeEncoder{}, make([]byte, 32))
	r := Request{Query: "车", Limit: 50}
	if _, e := service.Search(context.Background(), r); !errors.Is(e, ErrStale) {
		t.Fatalf("mutation=%v", e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := service.Search(ctx, r); !errors.Is(e, context.Canceled) {
		t.Fatalf("cancel=%v", e)
	}
	for _, r := range []Request{{Query: "", Limit: 50}, {Query: "x", Limit: 201}, {Query: "x", Limit: 1, DirectoryID: 1}, {Query: "x", Limit: 1, Recursive: true}, {Query: "x", Limit: 1, Mode: "ocr_bad"}, {Query: "x", Limit: 1, Kinds: []catalog.AssetKind{"image", "image"}}} {
		if _, e := service.Search(context.Background(), r); !errors.Is(e, ErrInvalid) {
			t.Fatalf("invalid accepted=%+v %v", r, e)
		}
	}
}

func TestFusedSearchStreamsPastFirstPageAndAppliesVisualCutoff(t *testing.T) {
	repo := &fakeRepo{revision: 1}
	for id := int64(1); id <= 1000; id++ {
		repo.candidates = append(repo.candidates, Candidate{SourceFingerprint: "v1:test", LibraryID: 1, AssetID: id, Name: "other.jpg", RelativePath: "other.jpg", Vector: vector(t, []float32{-1, 0})})
	}
	repo.candidates = append(repo.candidates,
		Candidate{LibraryID: 1, AssetID: 1001, Name: "car.jpg", RelativePath: "car.jpg", Vector: vector(t, []float32{1, 0})},
		Candidate{LibraryID: 1, AssetID: 1002, Name: "low.jpg", RelativePath: "low.jpg", Vector: vector(t, []float32{.2, 1})},
		Candidate{LibraryID: 1, AssetID: 1003, Name: "visual.jpg", RelativePath: "visual.jpg", Vector: vector(t, []float32{.25, 1})},
	)
	service, err := New(repo, &fakeEncoder{}, make([]byte, 32))
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.Search(context.Background(), Request{Query: "car", Limit: 1})
	if err != nil || len(result.Hits) != 1 || result.Hits[0].AssetID != 1001 || result.NextCursor == "" {
		t.Fatalf("late candidate lost: %+v %v", result, err)
	}
	next, err := service.Search(context.Background(), Request{Query: "car", Limit: 1, Cursor: result.NextCursor})
	if err != nil || len(next.Hits) != 1 || next.Hits[0].AssetID != 1003 || next.NextCursor != "" {
		t.Fatalf("cutoff or pagination: %+v %v", next, err)
	}
}
