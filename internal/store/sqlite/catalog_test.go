package sqlite

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/catalog"
	"github.com/HappyQuQu/foliopath/internal/library"
	"github.com/HappyQuQu/foliopath/internal/scanner"
)

func seedBrowseCatalog(t *testing.T, store *Store) int64 {
	t.Helper()
	record := createTestLibrary(t, store)
	run, err := store.BeginFullScan(context.Background(), record.ID, scanner.TriggerManual)
	if err != nil {
		t.Fatal(err)
	}
	entries := []scanner.CatalogEntry{
		{Kind: scanner.CatalogEntryDirectory},
		{
			Kind: scanner.CatalogEntryDirectory, RelativePath: "Album 10",
			ParentPath: "", Name: "Album 10", MTimeNS: 10,
		},
		{
			Kind: scanner.CatalogEntryDirectory, RelativePath: "Album 2",
			ParentPath: "", Name: "Album 2", MTimeNS: 20,
		},
		{
			Kind: scanner.CatalogEntryDirectory, RelativePath: "Album 1",
			ParentPath: "", Name: "Album 1", MTimeNS: 30,
		},
		{
			Kind: scanner.CatalogEntryDirectory, RelativePath: "Album 2/Nested",
			ParentPath: "Album 2", Name: "Nested", MTimeNS: 40,
		},
		{
			Kind: scanner.CatalogEntryAsset, RelativePath: "photo-10.jpg",
			ParentPath: "", Name: "photo-10.jpg", MTimeNS: 10,
			AssetKind: scanner.AssetKindImage, MediaFormat: scanner.MediaFormatJPEG,
			MIMEType: "image/jpeg", SizeBytes: 10,
		},
		{
			Kind: scanner.CatalogEntryAsset, RelativePath: "photo-2.jpg",
			ParentPath: "", Name: "photo-2.jpg", MTimeNS: 20,
			AssetKind: scanner.AssetKindImage, MediaFormat: scanner.MediaFormatJPEG,
			MIMEType: "image/jpeg", SizeBytes: 20,
		},
		{
			Kind: scanner.CatalogEntryAsset, RelativePath: "Album 2/clip.mp4",
			ParentPath: "Album 2", Name: "clip.mp4", MTimeNS: 30,
			AssetKind: scanner.AssetKindVideo, MediaFormat: scanner.MediaFormatMP4,
			MIMEType: "video/mp4", SizeBytes: 30,
		},
		{
			Kind: scanner.CatalogEntryAsset, RelativePath: "Album 2/Nested/photo.jpg",
			ParentPath: "Album 2/Nested", Name: "photo.jpg", MTimeNS: 40,
			AssetKind: scanner.AssetKindImage, MediaFormat: scanner.MediaFormatJPEG,
			MIMEType: "image/jpeg", SizeBytes: 40,
		},
	}
	if err := store.UpsertCatalogBatch(context.Background(), run.ID, entries); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CompleteFullScan(context.Background(), run.ID, scanner.SkipCounts{}); err != nil {
		t.Fatal(err)
	}
	return record.ID
}

func catalogServiceForStore(t *testing.T, store *Store) *catalog.Service {
	t.Helper()
	service, err := catalog.NewService(
		store,
		[]byte("0123456789abcdef0123456789abcdef"),
	)
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func TestCatalogDirectoryPageUsesNaturalKeysetAndNormalizesRoot(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	service := catalogServiceForStore(t, store)

	first, err := service.ListDirectories(context.Background(), catalog.DirectoryRequest{
		LibraryID: libraryID,
		Limit:     2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if names := []string{first.Items[0].Name, first.Items[1].Name}; !slices.Equal(
		names, []string{"Album 1", "Album 2"},
	) {
		t.Fatalf("first directory names = %v", names)
	}
	if first.NextCursor == "" {
		t.Fatal("first directory page has no cursor")
	}
	var rootID int64
	if err := store.db.QueryRowContext(context.Background(), `
        SELECT id FROM directories WHERE library_id = ? AND relative_path = ''`,
		libraryID,
	).Scan(&rootID); err != nil {
		t.Fatal(err)
	}
	second, err := service.ListDirectories(context.Background(), catalog.DirectoryRequest{
		LibraryID: libraryID, ParentDirectoryID: rootID,
		Cursor: first.NextCursor, Limit: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Items) != 1 || second.Items[0].Name != "Album 10" ||
		second.NextCursor != "" {
		t.Fatalf("second directory page = %#v", second)
	}
}

func TestCatalogAssetKeysetSupportsDirectRecursiveFilterAndDirection(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	service := catalogServiceForStore(t, store)

	direct, err := service.ListAssets(context.Background(), catalog.AssetRequest{
		LibraryID: libraryID,
		Limit:     1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(direct.Items) != 1 || direct.Items[0].Name != "photo-2.jpg" ||
		direct.NextCursor == "" {
		t.Fatalf("first direct asset page = %#v", direct)
	}
	directNext, err := service.ListAssets(context.Background(), catalog.AssetRequest{
		LibraryID: libraryID, Cursor: direct.NextCursor, Limit: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(directNext.Items) != 1 || directNext.Items[0].Name != "photo-10.jpg" {
		t.Fatalf("second direct asset page = %#v", directNext)
	}

	recursive, err := service.ListAssets(context.Background(), catalog.AssetRequest{
		LibraryID: libraryID, Recursive: true,
		Kinds: []catalog.AssetKind{catalog.KindVideo},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(recursive.Items) != 1 || recursive.Items[0].RelativePath != "Album 2/clip.mp4" {
		t.Fatalf("recursive video page = %#v", recursive)
	}

	ascending, err := service.ListAssets(context.Background(), catalog.AssetRequest{
		LibraryID: libraryID, Recursive: true,
		Sort: catalog.SortModifiedAt, Order: catalog.OrderAsc,
	})
	if err != nil {
		t.Fatal(err)
	}
	gotMTime := make([]int64, len(ascending.Items))
	for index, item := range ascending.Items {
		gotMTime[index] = item.ModifiedAtNS
	}
	if !slices.Equal(gotMTime, []int64{10, 20, 30, 40}) {
		t.Fatalf("ascending modified times = %v", gotMTime)
	}
}

func TestCatalogScopeIsolationOfflineStateGenerationAndCancellation(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	other, err := store.CreateLibrary(context.Background(), library.CreateParams{
		Name: "Other", RootRelativePath: "other",
	})
	if err != nil {
		t.Fatal(err)
	}
	service := catalogServiceForStore(t, store)

	var foreignDirectoryID int64
	if err := store.db.QueryRowContext(context.Background(), `
        SELECT id FROM directories
        WHERE library_id = ? AND relative_path = 'Album 2'`,
		libraryID,
	).Scan(&foreignDirectoryID); err != nil {
		t.Fatal(err)
	}
	_, err = service.ListAssets(context.Background(), catalog.AssetRequest{
		LibraryID: other.ID, DirectoryID: foreignDirectoryID,
	})
	if !errors.Is(err, catalog.ErrDirectoryNotFound) {
		t.Fatalf("cross-library scope error = %v", err)
	}

	first, err := service.ListAssets(context.Background(), catalog.AssetRequest{
		LibraryID: libraryID, Recursive: true, Limit: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(context.Background(), `
        UPDATE libraries SET current_generation = current_generation + 1, status = 'offline'
        WHERE id = ?`,
		libraryID,
	); err != nil {
		t.Fatal(err)
	}
	_, err = service.ListAssets(context.Background(), catalog.AssetRequest{
		LibraryID: libraryID, Recursive: true, Cursor: first.NextCursor, Limit: 1,
	})
	if !errors.Is(err, catalog.ErrInvalidCursor) {
		t.Fatalf("advanced generation cursor error = %v", err)
	}
	offline, err := service.ListAssets(context.Background(), catalog.AssetRequest{
		LibraryID: libraryID, Recursive: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range offline.Items {
		if item.Availability != catalog.SourceOffline {
			t.Fatalf("offline asset availability = %q", item.Availability)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = store.ListAssetPage(ctx, catalog.AssetListParams{
		Query: catalog.AssetQuery{
			Scope: catalog.Scope{
				LibraryID: libraryID, RootDirectoryID: 1, DirectoryID: 1,
			},
			Recursive: true, Sort: catalog.SortModifiedAt, Order: catalog.OrderDesc,
		},
		Limit: 10,
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled SQLite query error = %v", err)
	}
}
