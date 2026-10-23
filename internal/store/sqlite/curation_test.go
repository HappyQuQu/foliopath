package sqlite

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/catalog"
	"github.com/HappyQuQu/foliopath/internal/curation"
	"github.com/HappyQuQu/foliopath/internal/scanner"
)

func newCurationServiceForStore(t *testing.T, store *Store, now time.Time) *curation.Service {
	t.Helper()
	service, err := curation.NewService(
		store,
		[]byte("0123456789abcdef0123456789abcdef"),
		func() time.Time { return now },
	)
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func catalogAssetID(t *testing.T, store *Store, relativePath string) int64 {
	t.Helper()
	var id int64
	if err := store.db.QueryRowContext(context.Background(), `
        SELECT id FROM assets WHERE relative_path = ?`, relativePath).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func TestCurationFavoriteIsIdempotentAndInvalidatesOldCursor(t *testing.T) {
	store, _ := openTestStore(t)
	seedBrowseCatalog(t, store)
	now := time.Date(2026, 8, 10, 8, 0, 0, 0, time.UTC)
	service := newCurationServiceForStore(t, store, now)
	firstID := catalogAssetID(t, store, "photo-10.jpg")
	secondID := catalogAssetID(t, store, "photo-2.jpg")

	firstState, err := service.SetFavorite(context.Background(), firstID, true)
	if err != nil {
		t.Fatal(err)
	}
	firstRevision := firstState.Revision
	if !firstState.Favorite || firstState.FavoritedAt == nil || !firstState.FavoritedAt.Equal(now) {
		t.Fatalf("first favorite state = %#v", firstState)
	}
	repeated, err := service.SetFavorite(context.Background(), firstID, true)
	if err != nil {
		t.Fatal(err)
	}
	if repeated.Revision != firstRevision || repeated.FavoritedAt == nil || !repeated.FavoritedAt.Equal(now) {
		t.Fatalf("repeated favorite changed state = %#v", repeated)
	}

	now = now.Add(time.Minute)
	if _, err := service.SetFavorite(context.Background(), secondID, true); err != nil {
		t.Fatal(err)
	}
	page, err := service.ListAssets(context.Background(), curation.AssetListRequest{FavoriteOnly: true, Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].Asset.ID != secondID || page.NextCursor == "" || page.Counts.All != 2 {
		t.Fatalf("favorite page = %#v", page)
	}

	now = now.Add(time.Minute)
	if _, err := service.SetFavorite(context.Background(), secondID, false); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ListAssets(context.Background(), curation.AssetListRequest{FavoriteOnly: true, Limit: 1, Cursor: page.NextCursor}); !errors.Is(err, curation.ErrInvalidCursor) {
		t.Fatalf("stale favorite cursor error = %v", err)
	}
}

func TestCurationTagsNormalizeReplaceAndCascade(t *testing.T) {
	store, _ := openTestStore(t)
	seedBrowseCatalog(t, store)
	now := time.Date(2026, 8, 10, 9, 0, 0, 0, time.UTC)
	service := newCurationServiceForStore(t, store, now)
	assetID := catalogAssetID(t, store, "photo-10.jpg")

	tag, created, err := service.CreateTag(context.Background(), "  Café   Trip ")
	if err != nil || !created || tag.Name != "Café Trip" {
		t.Fatalf("created tag = %#v, %v, %v", tag, created, err)
	}
	equivalent, created, err := service.CreateTag(context.Background(), "cafe\u0301 trip")
	if err != nil || created || equivalent.ID != tag.ID {
		t.Fatalf("equivalent tag = %#v, %v, %v", equivalent, created, err)
	}
	second, _, err := service.CreateTag(context.Background(), "人物")
	if err != nil {
		t.Fatal(err)
	}
	state, err := service.GetAssetState(context.Background(), assetID)
	if err != nil {
		t.Fatal(err)
	}
	updated, err := service.ReplaceAssetTags(context.Background(), assetID, state.Revision, []int64{second.ID, tag.ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.Tags) != 2 {
		t.Fatalf("updated tags = %#v", updated.Tags)
	}
	if _, err := service.ReplaceAssetTags(context.Background(), assetID, state.Revision, []int64{tag.ID}); !errors.Is(err, curation.ErrPreconditionFailed) {
		t.Fatalf("stale replace error = %v", err)
	}

	page, err := service.ListAssets(context.Background(), curation.AssetListRequest{TagID: tag.ID})
	if err != nil || len(page.Items) != 1 || page.Items[0].Asset.ID != assetID {
		t.Fatalf("tagged page = %#v, %v", page, err)
	}
	if _, err := store.db.ExecContext(context.Background(), `DELETE FROM assets WHERE id = ?`, assetID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.GetAssetState(context.Background(), assetID); !errors.Is(err, curation.ErrAssetNotFound) {
		t.Fatalf("deleted asset state error = %v", err)
	}
	var associations int
	if err := store.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM asset_tags WHERE asset_id = ?`, assetID).Scan(&associations); err != nil {
		t.Fatal(err)
	}
	if associations != 0 {
		t.Fatalf("asset tag associations after cascade = %d", associations)
	}
}

func TestCurationMigrationConstraints(t *testing.T) {
	store, _ := openTestStore(t)
	defer store.Close()
	var version int
	if err := store.db.QueryRow(`SELECT MAX(version_id) FROM goose_db_version WHERE is_applied = 1`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != 44 {
		t.Fatalf("migration version = %d, want 44", version)
	}
	for _, table := range []string{"curation_state", "asset_favorites", "tags", "asset_tags"} {
		var count int
		if err := store.db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("table %s count = %d", table, count)
		}
	}
}

func TestCurationTagListCountsDistinctAssetsWithinOptionalScope(t *testing.T) {
	store, service, first, second, empty := seedScopedTagCatalog(t)
	ctx := context.Background()
	for _, test := range []struct {
		name    string
		request curation.TagListRequest
		want    map[string]int64
	}{
		{"global keeps unused", curation.TagListRequest{}, map[string]int64{"Alpha shared": 4, "Beta photos": 3, "Other only": 1, "Unused": 0, "Video only": 1}},
		{"first library", curation.TagListRequest{LibraryID: first}, map[string]int64{"Alpha shared": 3, "Beta photos": 3, "Video only": 1}},
		{"first images including animated", curation.TagListRequest{LibraryID: first, Kinds: []catalog.AssetKind{catalog.KindImage, catalog.KindAnimated}}, map[string]int64{"Alpha shared": 2, "Beta photos": 3}},
		{"second library images", curation.TagListRequest{LibraryID: second, Kinds: []catalog.AssetKind{catalog.KindImage, catalog.KindAnimated}}, map[string]int64{"Alpha shared": 1, "Other only": 1}},
		{"global static images only", curation.TagListRequest{Kinds: []catalog.AssetKind{catalog.KindImage}}, map[string]int64{"Alpha shared": 2, "Beta photos": 2, "Other only": 1}},
		{"animated only", curation.TagListRequest{Kinds: []catalog.AssetKind{catalog.KindAnimated}}, map[string]int64{"Alpha shared": 1, "Beta photos": 1}},
		{"video only", curation.TagListRequest{LibraryID: first, Kinds: []catalog.AssetKind{catalog.KindVideo}}, map[string]int64{"Alpha shared": 1, "Video only": 1}},
		{"empty library", curation.TagListRequest{LibraryID: empty}, map[string]int64{}},
		{"library without requested kind", curation.TagListRequest{LibraryID: second, Kinds: []catalog.AssetKind{catalog.KindVideo}}, map[string]int64{}},
		{"search is scoped and normalized", curation.TagListRequest{LibraryID: first, Kinds: []catalog.AssetKind{catalog.KindImage, catalog.KindAnimated}, Search: " SHARED "}, map[string]int64{"Alpha shared": 2}},
		{"duplicate kinds do not double count", curation.TagListRequest{LibraryID: first, Kinds: []catalog.AssetKind{catalog.KindAnimated, catalog.KindImage, catalog.KindImage}}, map[string]int64{"Alpha shared": 2, "Beta photos": 3}},
	} {
		t.Run(test.name, func(t *testing.T) {
			page, err := service.ListTags(ctx, test.request)
			if err != nil || page.NextCursor != "" {
				t.Fatalf("tag page = %#v, %v", page, err)
			}
			got := make(map[string]int64)
			for _, tag := range page.Items {
				got[tag.Name] = tag.AssetCount
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("tag counts = %v, want %v", got, test.want)
			}
		})
	}
	if _, err := service.ListTags(ctx, curation.TagListRequest{LibraryID: 999999}); !errors.Is(err, curation.ErrLibraryNotFound) {
		t.Fatalf("unknown library should not return global tags: %v", err)
	}
	// Offline means retained catalog state, not an empty or substituted library.
	if _, err := store.db.ExecContext(ctx, `UPDATE libraries SET status='offline' WHERE id=?`, first); err != nil {
		t.Fatal(err)
	}
	page, err := service.ListTags(ctx, curation.TagListRequest{LibraryID: first, Kinds: []catalog.AssetKind{catalog.KindImage, catalog.KindAnimated}})
	if err != nil || len(page.Items) != 2 || page.Items[0].AssetCount != 2 {
		t.Fatalf("offline retained tag counts = %#v, %v", page, err)
	}
}

func TestCurationScopedTagPaginationIsStableAndRevisionBound(t *testing.T) {
	_, service, first, second, _ := seedScopedTagCatalog(t)
	ctx := context.Background()
	request := curation.TagListRequest{LibraryID: first, Kinds: []catalog.AssetKind{catalog.KindImage, catalog.KindAnimated}, Limit: 1}
	page, err := service.ListTags(ctx, request)
	if err != nil || len(page.Items) != 1 || page.Items[0].Name != "Alpha shared" || page.NextCursor == "" {
		t.Fatalf("first scoped tag page = %#v, %v", page, err)
	}
	request.Cursor = page.NextCursor
	request.Kinds = []catalog.AssetKind{catalog.KindAnimated, catalog.KindImage, catalog.KindImage}
	next, err := service.ListTags(ctx, request)
	if err != nil || len(next.Items) != 1 || next.Items[0].Name != "Beta photos" || next.Items[0].AssetCount != 3 || next.NextCursor != "" {
		t.Fatalf("next normalized scope page = %#v, %v", next, err)
	}
	for _, changed := range []curation.TagListRequest{
		{LibraryID: second, Kinds: request.Kinds, Cursor: request.Cursor, Limit: 1},
		{LibraryID: first, Kinds: []catalog.AssetKind{catalog.KindVideo}, Cursor: request.Cursor, Limit: 1},
		{LibraryID: first, Kinds: request.Kinds, Search: "Beta", Cursor: request.Cursor, Limit: 1},
		{Cursor: request.Cursor, Limit: 1},
	} {
		if _, err := service.ListTags(ctx, changed); !errors.Is(err, curation.ErrInvalidCursor) {
			t.Fatalf("cross-scope cursor accepted: %#v err=%v", changed, err)
		}
	}
	if _, _, err := service.CreateTag(ctx, "New unused tag"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ListTags(ctx, request); !errors.Is(err, curation.ErrInvalidCursor) {
		t.Fatalf("stale curation revision accepted: %v", err)
	}
}

func seedScopedTagCatalog(t *testing.T) (*Store, *curation.Service, int64, int64, int64) {
	t.Helper()
	store, _ := openTestStore(t)
	ctx := context.Background()
	entries := []scanner.CatalogEntry{
		{Kind: scanner.CatalogEntryDirectory},
		{Kind: scanner.CatalogEntryAsset, Name: "photo.jpg", RelativePath: "photo.jpg", AssetKind: scanner.AssetKindImage, MediaFormat: scanner.MediaFormatJPEG, MIMEType: "image/jpeg", SizeBytes: 1, MTimeNS: 1},
		{Kind: scanner.CatalogEntryAsset, Name: "second.jpg", RelativePath: "second.jpg", AssetKind: scanner.AssetKindImage, MediaFormat: scanner.MediaFormatJPEG, MIMEType: "image/jpeg", SizeBytes: 2, MTimeNS: 2},
		{Kind: scanner.CatalogEntryAsset, Name: "animated.gif", RelativePath: "animated.gif", AssetKind: scanner.AssetKindAnimated, MediaFormat: scanner.MediaFormatGIF, MIMEType: "image/gif", SizeBytes: 3, MTimeNS: 3},
		{Kind: scanner.CatalogEntryAsset, Name: "video.mp4", RelativePath: "video.mp4", AssetKind: scanner.AssetKindVideo, MediaFormat: scanner.MediaFormatMP4, MIMEType: "video/mp4", SizeBytes: 4, MTimeNS: 4},
	}
	first := seedSearchCatalog(t, store, "Tag first", "tag-first", entries)
	second := seedSearchCatalog(t, store, "Tag second", "tag-second", entries[:2])
	empty := seedSearchCatalog(t, store, "Tag empty", "tag-empty", entries[:1])
	service := newCurationServiceForStore(t, store, time.Unix(100, 0).UTC())
	tags := make(map[string]int64)
	for _, name := range []string{"Alpha shared", "Beta photos", "Other only", "Unused", "Video only"} {
		tag, _, err := service.CreateTag(ctx, name)
		if err != nil {
			t.Fatal(err)
		}
		tags[name] = tag.ID
	}
	for _, item := range []struct {
		libraryID int64
		path      string
		tagNames  []string
	}{
		{first, "photo.jpg", []string{"Alpha shared", "Beta photos"}},
		{first, "second.jpg", []string{"Beta photos"}},
		{first, "animated.gif", []string{"Alpha shared", "Beta photos"}},
		{first, "video.mp4", []string{"Alpha shared", "Video only"}},
		{second, "photo.jpg", []string{"Alpha shared", "Other only"}},
	} {
		var assetID int64
		if err := store.db.QueryRowContext(ctx, `SELECT id FROM assets WHERE library_id=? AND relative_path=?`, item.libraryID, item.path).Scan(&assetID); err != nil {
			t.Fatal(err)
		}
		state, err := service.GetAssetState(ctx, assetID)
		if err != nil {
			t.Fatal(err)
		}
		ids := make([]int64, 0, len(item.tagNames))
		for _, name := range item.tagNames {
			ids = append(ids, tags[name])
		}
		slices.Sort(ids)
		if _, err := service.ReplaceAssetTags(ctx, assetID, state.Revision, ids); err != nil {
			t.Fatal(err)
		}
	}
	return store, service, first, second, empty
}
