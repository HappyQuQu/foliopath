package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/curation"
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
	if version != 32 {
		t.Fatalf("migration version = %d, want 32", version)
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
