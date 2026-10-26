package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/catalog"
	"github.com/HappyQuQu/foliopath/internal/search"
	"github.com/HappyQuQu/foliopath/internal/semantic"
)

type fusedTestEncoder struct{}

func (fusedTestEncoder) EncodeSemanticText(context.Context, string, string) ([]float32, error) {
	return []float32{1, 0}, nil
}
func TestFusedSQLiteScopesEvidenceStalenessAndDegradation(t *testing.T) {
	s, _ := openTestStore(t)
	ctx := context.Background()
	lib := seedBrowseCatalog(t, s)
	gen := seedEmbeddingGeneration(t, s, 2)
	seedSemanticLibrarySettings(t, s, lib)
	if _, e := s.db.ExecContext(ctx, `UPDATE ai_model_state SET active_model_id=(SELECT model_id FROM semantic_generations WHERE id=?),active_generation_id=? WHERE singleton_key=1`, gen, gen); e != nil {
		t.Fatal(e)
	}
	page, e := s.ListSemanticBackfillCandidates(ctx, lib, gen, semantic.JobAll, 0, 20)
	if e != nil {
		t.Fatal(e)
	}
	for _, a := range page.Items {
		v, _ := semantic.EncodeEmbedding([]float32{1, 0}, 2)
		if e = s.PutSemanticEmbeddingBatch(ctx, semantic.EmbeddingBatch{GenerationID: gen, LibraryID: lib, Items: []semantic.EmbeddingItem{{AssetID: a.AssetID, SourceFingerprint: a.SourceFingerprint, Vector: v, CreatedAt: time.Now()}}}); e != nil {
			t.Fatal(e)
		}
	}
	service, e := search.New(s, fusedTestEncoder{}, make([]byte, 32))
	if e != nil {
		t.Fatal(e)
	}
	r := search.Request{Query: "photo", LibraryID: lib, Limit: 1}
	seen := map[int64]bool{}
	for {
		out, err := service.Search(ctx, r)
		if err != nil {
			t.Fatal(err)
		}
		for _, h := range out.Hits {
			if seen[h.AssetID] {
				t.Fatal("duplicate pagination")
			}
			seen[h.AssetID] = true
			if len(h.Sources) != 2 || h.Sources[0] != search.Filename || h.Sources[1] != search.Visual {
				t.Fatalf("sources=%+v", h)
			}
		}
		if out.NextCursor == "" {
			break
		}
		r.Cursor = out.NextCursor
	}
	if len(seen) != 3 {
		t.Fatalf("images=%v", seen)
	}
	var album int64
	if e = s.db.QueryRowContext(ctx, `SELECT id FROM directories WHERE library_id=? AND relative_path='Album 2'`, lib).Scan(&album); e != nil {
		t.Fatal(e)
	}
	r = search.Request{Query: "photo", LibraryID: lib, DirectoryID: album, Mode: "all", Limit: 50}
	out, e := service.Search(ctx, r)
	if e != nil || len(out.Hits) != 0 {
		t.Fatalf("direct=%+v %v", out, e)
	}
	r.Recursive = true
	out, e = service.Search(ctx, r)
	if e != nil || len(out.Hits) != 1 {
		t.Fatalf("recursive=%+v %v", out, e)
	}
	r = search.Request{Query: "clip", LibraryID: lib, Kinds: []catalog.AssetKind{catalog.KindVideo}, Limit: 50}
	out, e = service.Search(ctx, r)
	if e != nil || len(out.Hits) != 1 || out.Hits[0].SemanticScore != nil {
		t.Fatalf("video false visual=%+v %v", out, e)
	}
	bound := int64(30)
	r.ModifiedBeforeNS = &bound
	out, e = service.Search(ctx, r)
	if e != nil || len(out.Hits) != 0 {
		t.Fatalf("date boundary=%+v %v", out, e)
	}
	// A changed source cannot reuse an old vector, while literal catalog results remain.
	if _, e = s.db.ExecContext(ctx, `UPDATE assets SET source_fingerprint='v1:source-changed' WHERE library_id=? AND name='photo-10.jpg'`, lib); e != nil {
		t.Fatal(e)
	}
	r = search.Request{Query: "photo-10", LibraryID: lib, Limit: 50}
	out, e = service.Search(ctx, r)
	if e != nil {
		t.Fatal(e)
	}
	for _, h := range out.Hits {
		if h.AssetID == page.Items[0].AssetID && h.SemanticScore != nil {
			t.Fatal("stale vector reused")
		}
	}
	if _, e = s.db.ExecContext(ctx, `UPDATE ai_library_settings SET enabled=0,state='disabled' WHERE library_id=?`, lib); e != nil {
		t.Fatal(e)
	}
	r = search.Request{Query: "photo", LibraryID: lib, Limit: 50}
	out, e = service.Search(ctx, r)
	if e != nil || out.SemanticState != "disabled" || len(out.Hits) != 3 {
		t.Fatalf("disabled fallback=%+v %v", out, e)
	}
	if _, e = s.db.ExecContext(ctx, `UPDATE libraries SET status='offline' WHERE id=?`, lib); e != nil {
		t.Fatal(e)
	}
	out, e = service.Search(ctx, r)
	if e != nil || out.SemanticState != "offline" || len(out.Hits) != 3 {
		t.Fatalf("offline catalog preserved=%+v %v", out, e)
	}
	r.DirectoryID = 999999
	if _, e = service.Search(ctx, r); e == nil {
		t.Fatal("missing directory accepted")
	}
}
