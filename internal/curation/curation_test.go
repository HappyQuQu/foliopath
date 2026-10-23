package curation

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/catalog"
)

func TestNormalizeTagNameUsesNFCWhitespaceAndCaseFold(t *testing.T) {
	name, key, err := NormalizeTagName("  Cafe\u0301　 TRIP  ")
	if err != nil {
		t.Fatal(err)
	}
	if name != "Café TRIP" || key != "café trip" {
		t.Fatalf("normalized tag = %q / %q", name, key)
	}
}

func TestNormalizeTagNameRejectsControlsEmptyAndOversize(t *testing.T) {
	for _, value := range []string{"", "   ", "trip\nname", strings.Repeat("界", MaxTagRunes+1)} {
		if _, _, err := NormalizeTagName(value); !errors.Is(err, ErrInvalidRequest) {
			t.Fatalf("NormalizeTagName(%q) error = %v", value, err)
		}
	}
}

type tagListRepositoryStub struct {
	Repository
	revision int64
	params   TagListParams
	calls    int
}

func (stub *tagListRepositoryStub) Revision(context.Context) (int64, error) {
	return stub.revision, nil
}

func (stub *tagListRepositoryStub) ResolveLibrary(_ context.Context, id int64) error {
	if id != 7 && id != 8 {
		return ErrLibraryNotFound
	}
	return nil
}

func (stub *tagListRepositoryStub) ListTagPage(_ context.Context, params TagListParams) ([]Tag, error) {
	stub.params, stub.calls = params, stub.calls+1
	now := time.Unix(100, 0).UTC()
	if params.After != nil {
		return []Tag{{ID: 2, Name: "Beta", NormalizedName: "beta", AssetCount: 2, CreatedAt: now, UpdatedAt: now}}, nil
	}
	return []Tag{
		{ID: 1, Name: "Alpha", NormalizedName: "alpha", AssetCount: 3, CreatedAt: now, UpdatedAt: now},
		{ID: 2, Name: "Beta", NormalizedName: "beta", AssetCount: 2, CreatedAt: now, UpdatedAt: now},
	}, nil
}

func TestTagListNormalizesScopeAndBindsCursor(t *testing.T) {
	ctx := context.Background()
	repository := &tagListRepositoryStub{revision: 4}
	service, err := NewService(repository, []byte("0123456789abcdef0123456789abcdef"), nil)
	if err != nil {
		t.Fatal(err)
	}
	kinds := []catalog.AssetKind{catalog.KindImage, catalog.KindAnimated, catalog.KindImage}
	request := TagListRequest{LibraryID: 7, Kinds: kinds, Search: " ALPHA ", Limit: 1}
	page, err := service.ListTags(ctx, request)
	if err != nil || len(page.Items) != 1 || page.NextCursor == "" {
		t.Fatalf("first tag page = %#v, %v", page, err)
	}
	if repository.params.LibraryID != 7 || repository.params.SearchKey != "alpha" || repository.params.Limit != 2 ||
		!slices.Equal(repository.params.Kinds, []catalog.AssetKind{catalog.KindAnimated, catalog.KindImage}) ||
		!slices.Equal(kinds, []catalog.AssetKind{catalog.KindImage, catalog.KindAnimated, catalog.KindImage}) {
		t.Fatalf("scope was not normalized without mutating input: %#v, input=%v", repository.params, kinds)
	}
	request.Cursor, request.Search = page.NextCursor, "alpha"
	request.Kinds = []catalog.AssetKind{catalog.KindAnimated, catalog.KindImage}
	if _, err := service.ListTags(ctx, request); err != nil || repository.params.After == nil || repository.params.After.ID != 1 {
		t.Fatalf("equivalent scope did not accept cursor: %#v, %v", repository.params, err)
	}
	for _, test := range []struct {
		name   string
		change func(*TagListRequest)
	}{
		{"other library", func(r *TagListRequest) { r.LibraryID = 8 }},
		{"global scope", func(r *TagListRequest) { r.LibraryID = 0 }},
		{"other kind", func(r *TagListRequest) { r.Kinds = []catalog.AssetKind{catalog.KindVideo} }},
		{"all kinds", func(r *TagListRequest) { r.Kinds = nil }},
		{"other search", func(r *TagListRequest) { r.Search = "beta" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			changed := request
			test.change(&changed)
			calls := repository.calls
			if _, err := service.ListTags(ctx, changed); !errors.Is(err, ErrInvalidCursor) || repository.calls != calls {
				t.Fatalf("changed scope listed rows: calls=%d err=%v", repository.calls, err)
			}
		})
	}
	repository.revision++
	if _, err := service.ListTags(ctx, request); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("stale revision accepted: %v", err)
	}
}

func TestTagListRejectsInvalidScopeWithoutGlobalFallback(t *testing.T) {
	repository := &tagListRepositoryStub{revision: 1}
	service, err := NewService(repository, []byte("0123456789abcdef0123456789abcdef"), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, request := range []TagListRequest{
		{LibraryID: -1}, {Kinds: []catalog.AssetKind{"unknown"}}, {Kinds: []catalog.AssetKind{""}}, {Limit: MaxPageSize + 1},
	} {
		if _, err := service.ListTags(context.Background(), request); !errors.Is(err, ErrInvalidRequest) {
			t.Fatalf("invalid scope %#v: %v", request, err)
		}
	}
	if _, err := service.ListTags(context.Background(), TagListRequest{LibraryID: 999}); !errors.Is(err, ErrLibraryNotFound) {
		t.Fatalf("unknown library: %v", err)
	}
	if repository.calls != 0 {
		t.Fatalf("invalid scope queried global rows: %d calls", repository.calls)
	}
}

func TestUnfilteredTagCursorRetainsExistingFingerprint(t *testing.T) {
	repository := &tagListRepositoryStub{revision: 1}
	service, err := NewService(repository, []byte("0123456789abcdef0123456789abcdef"), nil)
	if err != nil {
		t.Fatal(err)
	}
	legacyFingerprint := fingerprint(struct {
		Search string `json:"q"`
	}{""})
	cursor, err := service.encodeTagCursor(1, legacyFingerprint, TagPosition{NormalizedName: "alpha", ID: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.ListTags(context.Background(), TagListRequest{Cursor: cursor}); err != nil || repository.params.After == nil {
		t.Fatalf("unfiltered legacy cursor rejected: %v", err)
	}
}
