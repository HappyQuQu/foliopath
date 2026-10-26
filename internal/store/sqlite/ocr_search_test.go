package sqlite

import (
	"context"
	"errors"
	"github.com/HappyQuQu/foliopath/internal/catalog"
	"github.com/HappyQuQu/foliopath/internal/ocr"
	"github.com/HappyQuQu/foliopath/internal/search"
	"github.com/HappyQuQu/foliopath/migrations"
	"github.com/pressly/goose/v3"
	"strings"
	"testing"
)

func TestOCRFulltextFusionCurrentSourceScopesAndClear(t *testing.T) {
	s, id, v, now := setupOCR(t)
	ctx := context.Background()
	admitOCR(t, s, v, ocr.Missing, "search001", now)
	j := claimOCR(t, s, now)
	var ids []int64
	for i := 0; i < 3; i++ {
		c := resultOCR(t, s, j, now)
		c.Value.Result.Lines[0].Text = "photo 中文 Ｃａｆé %_ quote\" <script>alert(1)</script>"
		if err := s.CommitOCRResult(ctx, c, now); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, c.Value.AssetID)
		j = reloadOCR(t, s, j)
	}
	if err := s.FinishOCRJob(ctx, j, "succeeded", "", now); err != nil {
		t.Fatal(err)
	}
	svc, err := search.New(s, fusedTestEncoder{}, make([]byte, 32), v.Fingerprint)
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{"中", "中文", "café", "ＣＡＦÉ", "%_", "quote\"", "中文 café"} {
		page, e := svc.Search(ctx, search.Request{Query: q, LibraryID: id, Mode: "ocr", Limit: 20})
		if e != nil || len(page.Hits) != 3 {
			t.Fatalf("%q: %+v %v", q, page, e)
		}
		if page.OCRCoverage.Ready != 3 || page.SemanticState != "not_requested" {
			t.Fatal(page)
		}
		for _, hit := range page.Hits {
			if len(hit.Sources) != 1 || hit.Sources[0] != search.OCR || len(hit.OCR) != 1 || hit.OCR[0].Line != 1 || !strings.Contains(hit.OCR[0].Text, "<script>") {
				t.Fatal(hit)
			}
		}
	}
	// Name and OCR evidence are deduplicated in one ranked row.
	page, err := svc.Search(ctx, search.Request{Query: "photo", LibraryID: id, Mode: "all", Limit: 20})
	if err != nil || len(page.Hits) != 3 || len(page.Hits[0].Sources) != 2 {
		t.Fatal(page, err)
	}
	request := search.Request{Query: "中文", LibraryID: id, Mode: "ocr", Limit: 1}
	first, err := svc.Search(ctx, request)
	if err != nil || first.NextCursor == "" {
		t.Fatal(first, err)
	}
	request.Cursor = first.NextCursor
	second, err := svc.Search(ctx, request)
	if err != nil || second.Hits[0].AssetID == first.Hits[0].AssetID {
		t.Fatal(second, err)
	}
	video, err := svc.Search(ctx, search.Request{Query: "中文", LibraryID: id, Mode: "ocr", Kinds: []catalog.AssetKind{catalog.KindVideo}, Limit: 20})
	if err != nil || len(video.Hits) != 0 || video.OCRCoverage.Eligible != 0 {
		t.Fatal(video, err)
	}
	var directory int64
	if err = s.db.QueryRow(`SELECT directory_id FROM assets WHERE id=?`, ids[0]).Scan(&directory); err != nil {
		t.Fatal(err)
	}
	scoped, err := svc.Search(ctx, search.Request{Query: "中文", LibraryID: id, DirectoryID: directory, Mode: "ocr", Limit: 20})
	if err != nil || len(scoped.Hits) == 0 || len(scoped.Hits) >= 3 {
		t.Fatal(scoped, err)
	}
	bound := int64(1)
	dated, e := svc.Search(ctx, search.Request{Query: "中文", LibraryID: id, Mode: "ocr", ModifiedBeforeNS: &bound, Limit: 20})
	if e != nil || len(dated.Hits) != 0 || dated.OCRCoverage.Eligible != 0 {
		t.Fatal(dated, e)
	}
	if _, e = s.db.Exec(`UPDATE libraries SET status='offline' WHERE id=?`, id); e != nil {
		t.Fatal(e)
	}
	offline, e := svc.Search(ctx, search.Request{Query: "中文", LibraryID: id, Mode: "ocr", Limit: 20})
	if e != nil || len(offline.Hits) != 0 {
		t.Fatal(offline, e)
	}
	if _, e = s.db.Exec(`UPDATE libraries SET status='ready' WHERE id=?`, id); e != nil {
		t.Fatal(e)
	}
	// Verify prevents leaking a now-hidden OCR excerpt after asset hydration.
	if _, err = s.db.Exec(`UPDATE assets SET source_fingerprint='v1:changed:1' WHERE id=?`, ids[0]); err != nil {
		t.Fatal(err)
	}
	if err = svc.Verify(ctx, search.Request{Query: "中文", LibraryID: id, Mode: "ocr", Limit: 1}, first); !errors.Is(err, search.ErrStale) {
		t.Fatal(err)
	}
	if _, err = svc.Search(ctx, request); !errors.Is(err, search.ErrStale) {
		t.Fatal(err)
	}
	current, err := svc.Search(ctx, search.Request{Query: "中文", LibraryID: id, Mode: "ocr", Limit: 20})
	if err != nil || len(current.Hits) != 2 || current.OCRCoverage.Ready != 2 {
		t.Fatal(current, err)
	}
	unconfigured, _ := search.New(s, fusedTestEncoder{}, make([]byte, 32))
	hidden, err := unconfigured.Search(ctx, search.Request{Query: "中文", LibraryID: id, Mode: "ocr", Limit: 20})
	if err != nil || len(hidden.Hits) != 0 || hidden.OCRState != "disabled" {
		t.Fatal(hidden, err)
	}
	if _, err = s.ConfigureOCR(ctx, id, v.Revision, false, ocr.Identity{}, now); err != nil {
		t.Fatal(err)
	}
	hidden, err = svc.Search(ctx, search.Request{Query: "中文", LibraryID: id, Mode: "ocr", Limit: 20})
	if err != nil || len(hidden.Hits) != 0 {
		t.Fatal(hidden, err)
	}
	settings, err := s.GetOCRSettings(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.BeginOCRClear(ctx, id, settings.Revision, now); err != nil {
		t.Fatal(err)
	}
	for {
		done, e := s.DrainOCRClear(ctx, id, settings.Epoch+1, 1, now)
		if e != nil {
			t.Fatal(e)
		}
		if done {
			break
		}
	}
	var n int
	if err = s.db.QueryRow(`SELECT COUNT(*) FROM ocr_search_documents`).Scan(&n); err != nil || n != 0 {
		t.Fatal(n, err)
	}
	if _, err = s.db.Exec(`INSERT INTO ocr_search(ocr_search,rank) VALUES('integrity-check',1)`); err != nil {
		t.Fatal(err)
	}
}

func TestOCRFulltextUpgradeBackfillAndAtomicRollback(t *testing.T) {
	s, id, v, now := setupOCR(t)
	ctx := context.Background()
	admitOCR(t, s, v, ocr.Missing, "search002", now)
	j := claimOCR(t, s, now)
	c := resultOCR(t, s, j, now)
	// Fail the checkpoint after writing result/index. Nothing in that transaction survives.
	if _, err := s.db.Exec(`CREATE TRIGGER fail_ocr_checkpoint BEFORE UPDATE OF checkpoint_id ON ocr_jobs BEGIN SELECT RAISE(ABORT,'synthetic'); END`); err != nil {
		t.Fatal(err)
	}
	if err := s.CommitOCRResult(ctx, c, now); err == nil {
		t.Fatal("expected rollback")
	}
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM ocr_search_documents`).Scan(&n); err != nil || n != 0 {
		t.Fatal(n, err)
	}
	if _, err := s.db.Exec(`DROP TRIGGER fail_ocr_checkpoint`); err != nil {
		t.Fatal(err)
	}
	if err := s.CommitOCRResult(ctx, c, now); err != nil {
		t.Fatal(err)
	}
	provider, err := goose.NewProvider(goose.DialectSQLite3, s.db, migrations.FS)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = provider.DownTo(ctx, 46); err != nil {
		t.Fatal(err)
	}
	if _, err = provider.Up(ctx); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err = s.backfillOCRSearch(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if err = s.db.QueryRow(`SELECT COUNT(*) FROM ocr_search_documents`).Scan(&n); err != nil || n != 1 {
		t.Fatal(n, err)
	}
	svc, _ := search.New(s, fusedTestEncoder{}, make([]byte, 32), v.Fingerprint)
	page, err := svc.Search(ctx, search.Request{Query: "café", LibraryID: id, Mode: "ocr", Limit: 20})
	if err != nil || len(page.Hits) != 1 || page.Hits[0].OCR[0].Text != "中文 Café" {
		t.Fatal(page, err)
	}
	if _, err = s.db.Exec(`DELETE FROM assets WHERE id=?`, c.Value.AssetID); err != nil {
		t.Fatal(err)
	}
	if err = s.db.QueryRow(`SELECT COUNT(*) FROM ocr_search_documents`).Scan(&n); err != nil || n != 0 {
		t.Fatal(n, err)
	}
	if _, err = s.db.Exec(`INSERT INTO ocr_search(ocr_search,rank) VALUES('integrity-check',1)`); err != nil {
		t.Fatal(err)
	}
}

func TestOCRFulltextIdentityVisibilityDoesNotRelabel(t *testing.T) {
	s, id, v, now := setupOCR(t)
	ctx := context.Background()
	admitOCR(t, s, v, ocr.Missing, "search003", now)
	j := claimOCR(t, s, now)
	c := resultOCR(t, s, j, now)
	if err := s.CommitOCRResult(ctx, c, now); err != nil {
		t.Fatal(err)
	}
	identity := persistenceIdentity()
	identity.RuntimeContractSHA256 = strings.Repeat("c", 64)
	next, err := s.ConfigureOCR(ctx, id, v.Revision, true, identity, now)
	if err != nil {
		t.Fatal(err)
	}
	for _, fp := range []string{v.Fingerprint, next.Fingerprint} {
		svc, _ := search.New(s, fusedTestEncoder{}, make([]byte, 32), fp)
		page, err := svc.Search(ctx, search.Request{Query: "中文", LibraryID: id, Mode: "ocr", Limit: 20})
		if err != nil || len(page.Hits) != 0 {
			t.Fatal(page, err)
		}
	}
	var n int
	if err = s.db.QueryRow(`SELECT COUNT(*) FROM ocr_search_documents WHERE fingerprint=?`, v.Fingerprint).Scan(&n); err != nil || n != 1 {
		t.Fatal("old document relabelled/deleted", n, err)
	}
}
