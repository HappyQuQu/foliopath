package search

import (
	"context"
	"strings"
	"testing"
)

func TestOCRExcerptsBoundedLiteralAndUnicode(t *testing.T) {
	lines := []string{strings.Repeat("前", 300) + "中文 Straße <img onerror=1>" + strings.Repeat("后", 100), "CAFÉ"}
	got := OCRExcerpts([]string{"中文", "café"}, lines)
	if len(got) != 2 || got[0].Line != 1 || len([]rune(got[0].Text)) > 162 || !strings.Contains(got[0].Text, "中文") {
		t.Fatal(got)
	}
	if len(OCRExcerpts([]string{"absent"}, lines)) != 0 {
		t.Fatal("invented hit")
	}
}

type ocrRepo struct{ fakeRepo }

func (f *ocrRepo) GetOCRSearchCoverage(context.Context, Scan) (OCRCoverage, error) {
	return OCRCoverage{Eligible: 2, Ready: 2}, nil
}
func TestOCRFusionOfflineEncoderAndNoDuplicate(t *testing.T) {
	repo := &ocrRepo{fakeRepo: fakeRepo{revision: 1, candidates: []Candidate{{SourceFingerprint: "v1:test", LibraryID: 1, AssetID: 1, Name: "中文.png", RelativePath: "中文.png", OCRLines: []string{"中文 receipt"}}}}}
	encoder := &fakeEncoder{err: ErrUnavailable}
	svc, err := New(repo, encoder, make([]byte, 32), strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"ocr", "all"} {
		page, e := svc.Search(context.Background(), Request{Query: "中文", Mode: mode, Limit: 20})
		if e != nil || len(page.Hits) != 1 || len(page.Hits[0].OCR) != 1 {
			t.Fatal(page, e)
		}
		if mode == "ocr" && encoder.calls != 0 {
			t.Fatal("OCR encoded query")
		}
		if mode == "all" && (len(page.Hits[0].Sources) != 2 || page.SemanticState != "unavailable") {
			t.Fatal(page)
		}
	}
}

func TestOCRAndVisualCombinedEvidenceRanksOnce(t *testing.T) {
	repo := &ocrRepo{fakeRepo: fakeRepo{revision: 1, candidates: []Candidate{{SourceFingerprint: "v1:test", LibraryID: 1, AssetID: 1, Name: "中文.png", RelativePath: "中文.png", Vector: vector(t, []float32{1, 0}), OCRLines: []string{"中文"}}}}}
	svc, err := New(repo, &fakeEncoder{}, make([]byte, 32), strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	page, err := svc.Search(context.Background(), Request{Query: "中文", Mode: "all", Limit: 20})
	if err != nil || len(page.Hits) != 1 || len(page.Hits[0].Sources) != 3 || page.Hits[0].SemanticScore == nil || page.Hits[0].Rank != 7 {
		t.Fatal(page, err)
	}
}
