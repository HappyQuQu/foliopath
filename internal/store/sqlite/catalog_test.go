package sqlite

import (
	"context"
	"errors"
	"path/filepath"
	"slices"
	"strings"
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

func seedSearchCatalog(
	t *testing.T,
	store *Store,
	name, root string,
	entries []scanner.CatalogEntry,
) int64 {
	t.Helper()
	service, err := library.NewService(store)
	if err != nil {
		t.Fatal(err)
	}
	record, err := service.Create(context.Background(), name, root)
	if err != nil {
		t.Fatal(err)
	}
	run, err := store.BeginFullScan(context.Background(), record.ID, scanner.TriggerManual)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.UpsertCatalogBatch(context.Background(), run.ID, entries); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CompleteFullScan(
		context.Background(), run.ID, scanner.SkipCounts{},
	); err != nil {
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

func TestCatalogDirectoryPageFiltersDirectChildrenByNormalizedTerms(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	service := catalogServiceForStore(t, store)

	query := "  ＡＬＢＵＭ　２ "
	page, err := service.ListDirectories(context.Background(), catalog.DirectoryRequest{
		LibraryID:   libraryID,
		SearchQuery: &query,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].Name != "Album 2" {
		t.Fatalf("filtered directory page = %#v", page)
	}

	empty := " \t "
	if _, err := service.ListDirectories(context.Background(), catalog.DirectoryRequest{
		LibraryID:   libraryID,
		SearchQuery: &empty,
	}); !errors.Is(err, catalog.ErrInvalidQuery) {
		t.Fatalf("empty directory filter error = %v", err)
	}
}

func TestCatalogDirectoryDetailUsesIndexedLineageAndLibraryRootName(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	service := catalogServiceForStore(t, store)

	var rootID, albumID, nestedID int64
	for relative, target := range map[string]*int64{
		"":               &rootID,
		"Album 2":        &albumID,
		"Album 2/Nested": &nestedID,
	} {
		if err := store.db.QueryRowContext(context.Background(), `
            SELECT id FROM directories WHERE library_id = ? AND relative_path = ?`,
			libraryID, relative,
		).Scan(target); err != nil {
			t.Fatal(err)
		}
	}

	root, err := service.GetDirectory(context.Background(), rootID)
	if err != nil {
		t.Fatal(err)
	}
	if root.Name != "Library" || root.ParentID != nil || root.RelativePath != "" ||
		len(root.Breadcrumbs) != 1 || root.Breadcrumbs[0].Name != "Library" ||
		!root.HasChildren {
		t.Fatalf("root detail = %#v", root)
	}

	nested, err := service.GetDirectory(context.Background(), nestedID)
	if err != nil {
		t.Fatal(err)
	}
	gotNames := make([]string, len(nested.Breadcrumbs))
	for index, item := range nested.Breadcrumbs {
		gotNames[index] = item.Name
	}
	if !slices.Equal(gotNames, []string{"Library", "Album 2", "Nested"}) ||
		nested.ParentID == nil || *nested.ParentID != albumID {
		t.Fatalf("nested detail = %#v", nested)
	}
}

func TestCatalogDirectoryDetailSupportsThousandLevelIndexedLineage(t *testing.T) {
	store, _ := openTestStore(t)
	record := createTestLibrary(t, store)
	tx, err := store.db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	result, err := tx.ExecContext(context.Background(), `
        INSERT INTO directories(
            library_id, parent_id, relative_path, name, natural_name_key,
            mtime_ns, last_seen_generation
        ) VALUES (?, NULL, '', '', ?, 0, 1)`,
		record.ID, catalog.NaturalNameKey(""),
	)
	if err != nil {
		t.Fatal(err)
	}
	parentID, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	components := make([]string, 0, 1000)
	for depth := 1; depth <= 1000; depth++ {
		components = append(components, "d")
		relative := strings.Join(components, "/")
		result, err = tx.ExecContext(context.Background(), `
            INSERT INTO directories(
                library_id, parent_id, relative_path, name, natural_name_key,
                mtime_ns, last_seen_generation
            ) VALUES (?, ?, ?, 'd', ?, 0, 1)`,
			record.ID, parentID, relative, catalog.NaturalNameKey("d"),
		)
		if err != nil {
			t.Fatalf("insert depth %d: %v", depth, err)
		}
		parentID, err = result.LastInsertId()
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	detail, err := catalogServiceForStore(t, store).GetDirectory(
		context.Background(), parentID,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(detail.Breadcrumbs) != 1001 ||
		detail.Breadcrumbs[0].Name != "Library" ||
		detail.Breadcrumbs[1000].RelativePath != strings.Join(components, "/") {
		t.Fatalf("deep breadcrumb endpoints = %d, %#v, %#v",
			len(detail.Breadcrumbs), detail.Breadcrumbs[0], detail.Breadcrumbs[len(detail.Breadcrumbs)-1])
	}
}

func TestCatalogDirectoryDetailFailsClosedForCorruptParentTopology(t *testing.T) {
	for _, test := range []struct {
		name    string
		corrupt func(*testing.T, *Store, int64, int64)
	}{
		{
			name: "cycle",
			corrupt: func(t *testing.T, store *Store, _, directoryID int64) {
				if _, err := store.db.ExecContext(context.Background(), `
                    UPDATE directories SET parent_id = id WHERE id = ?`,
					directoryID,
				); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "missing parent",
			corrupt: func(t *testing.T, store *Store, _, directoryID int64) {
				connection, err := store.db.Conn(context.Background())
				if err != nil {
					t.Fatal(err)
				}
				defer connection.Close()
				if _, err := connection.ExecContext(
					context.Background(), `PRAGMA foreign_keys = OFF`,
				); err != nil {
					t.Fatal(err)
				}
				if _, err := connection.ExecContext(context.Background(), `
                    UPDATE directories SET parent_id = 9223372036854775806 WHERE id = ?`,
					directoryID,
				); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "cross library parent",
			corrupt: func(t *testing.T, store *Store, _ int64, directoryID int64) {
				other, err := store.CreateLibrary(context.Background(), library.CreateParams{
					Name: "Other", RootRelativePath: "other",
				})
				if err != nil {
					t.Fatal(err)
				}
				run, err := store.BeginFullScan(
					context.Background(), other.ID, scanner.TriggerManual,
				)
				if err != nil {
					t.Fatal(err)
				}
				if err := store.UpsertCatalogBatch(
					context.Background(), run.ID,
					[]scanner.CatalogEntry{{Kind: scanner.CatalogEntryDirectory}},
				); err != nil {
					t.Fatal(err)
				}
				if _, err := store.CompleteFullScan(
					context.Background(), run.ID, scanner.SkipCounts{},
				); err != nil {
					t.Fatal(err)
				}
				var otherRootID int64
				if err := store.db.QueryRowContext(context.Background(), `
                    SELECT id FROM directories
                    WHERE library_id = ? AND relative_path = ''`,
					other.ID,
				).Scan(&otherRootID); err != nil {
					t.Fatal(err)
				}
				connection, err := store.db.Conn(context.Background())
				if err != nil {
					t.Fatal(err)
				}
				defer connection.Close()
				if _, err := connection.ExecContext(
					context.Background(), `PRAGMA foreign_keys = OFF`,
				); err != nil {
					t.Fatal(err)
				}
				if _, err := connection.ExecContext(context.Background(), `
                    UPDATE directories SET parent_id = ? WHERE id = ?`,
					otherRootID, directoryID,
				); err != nil {
					t.Fatal(err)
				}
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			store, _ := openTestStore(t)
			libraryID := seedBrowseCatalog(t, store)
			var directoryID int64
			if err := store.db.QueryRowContext(context.Background(), `
                SELECT id FROM directories
                WHERE library_id = ? AND relative_path = 'Album 2/Nested'`,
				libraryID,
			).Scan(&directoryID); err != nil {
				t.Fatal(err)
			}
			test.corrupt(t, store, libraryID, directoryID)
			if _, err := catalogServiceForStore(t, store).GetDirectory(
				context.Background(), directoryID,
			); !errors.Is(err, catalog.ErrInvalidTopology) {
				t.Fatalf("topology error = %v", err)
			}
		})
	}
}

func TestCatalogDirectoryDetailPreservesReliableCountsWhileScanningAndOffline(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	var directoryID int64
	if err := store.db.QueryRowContext(context.Background(), `
        SELECT id FROM directories
        WHERE library_id = ? AND relative_path = 'Album 2'`,
		libraryID,
	).Scan(&directoryID); err != nil {
		t.Fatal(err)
	}
	run, err := store.BeginFullScan(context.Background(), libraryID, scanner.TriggerManual)
	if err != nil {
		t.Fatal(err)
	}
	scanning, err := catalogServiceForStore(t, store).GetDirectory(
		context.Background(), directoryID,
	)
	if err != nil {
		t.Fatal(err)
	}
	if scanning.DirectAssetCount != 1 || scanning.RecursiveAssetCount != 2 {
		t.Fatalf("scanning counts = %d, %d",
			scanning.DirectAssetCount, scanning.RecursiveAssetCount)
	}
	if _, err := store.OfflineFullScan(
		context.Background(), run.ID, scanner.SkipCounts{}, "source_unavailable",
	); err != nil {
		t.Fatal(err)
	}
	offline, err := catalogServiceForStore(t, store).GetDirectory(
		context.Background(), directoryID,
	)
	if err != nil {
		t.Fatal(err)
	}
	if offline.DirectAssetCount != 1 || offline.RecursiveAssetCount != 2 {
		t.Fatalf("offline counts = %d, %d",
			offline.DirectAssetCount, offline.RecursiveAssetCount)
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
	if recursive.Counts != (catalog.AssetCounts{All: 4, Images: 3, Videos: 1}) {
		t.Fatalf("recursive asset counts = %#v", recursive.Counts)
	}

	var folderThenName []string
	var cursor string
	for {
		page, pageErr := service.ListAssets(context.Background(), catalog.AssetRequest{
			LibraryID: libraryID, Recursive: true, Sort: catalog.SortName,
			Order: catalog.OrderAsc, Cursor: cursor, Limit: 1,
		})
		if pageErr != nil {
			t.Fatal(pageErr)
		}
		for _, item := range page.Items {
			folderThenName = append(folderThenName, item.RelativePath)
		}
		if page.NextCursor == "" {
			break
		}
		cursor = page.NextCursor
	}
	if want := []string{
		"photo-2.jpg", "photo-10.jpg", "Album 2/clip.mp4", "Album 2/Nested/photo.jpg",
	}; !slices.Equal(folderThenName, want) {
		t.Fatalf("folder-then-name order = %v, want %v", folderThenName, want)
	}
	descendingByFolder, err := service.ListAssets(context.Background(), catalog.AssetRequest{
		LibraryID: libraryID, Recursive: true, Sort: catalog.SortName, Order: catalog.OrderDesc,
	})
	if err != nil {
		t.Fatal(err)
	}
	descendingPaths := make([]string, 0, len(descendingByFolder.Items))
	for _, item := range descendingByFolder.Items {
		descendingPaths = append(descendingPaths, item.RelativePath)
	}
	if want := []string{
		"Album 2/Nested/photo.jpg", "Album 2/clip.mp4", "photo-10.jpg", "photo-2.jpg",
	}; !slices.Equal(descendingPaths, want) {
		t.Fatalf("descending folder-then-name order = %v, want %v", descendingPaths, want)
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

	if _, err := store.db.Exec(`
		UPDATE assets SET size_bytes = 30 WHERE name = 'photo-2.jpg'
	`); err != nil {
		t.Fatal(err)
	}
	largest, err := service.ListAssets(context.Background(), catalog.AssetRequest{
		LibraryID: libraryID, Recursive: true,
		Sort: catalog.SortSize, Order: catalog.OrderDesc, Limit: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(largest.Items) != 2 ||
		largest.Items[0].SizeBytes != 40 ||
		largest.Items[1].SizeBytes != 30 ||
		largest.NextCursor == "" {
		t.Fatalf("largest asset page = %#v", largest)
	}
	smallest, err := service.ListAssets(context.Background(), catalog.AssetRequest{
		LibraryID: libraryID, Recursive: true,
		Sort: catalog.SortSize, Order: catalog.OrderDesc,
		Cursor: largest.NextCursor, Limit: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(smallest.Items) != 2 ||
		smallest.Items[0].SizeBytes != 30 ||
		smallest.Items[1].SizeBytes != 10 {
		t.Fatalf("second size page = %#v", smallest)
	}
}

func TestCatalogSearchSupportsUnicodeScopesFiltersAndGlobalRevision(t *testing.T) {
	store, _ := openTestStore(t)
	firstLibraryID := seedSearchCatalog(t, store, "第一库", "first", []scanner.CatalogEntry{
		{Kind: scanner.CatalogEntryDirectory},
		{
			Kind: scanner.CatalogEntryDirectory, RelativePath: "Trips",
			ParentPath: "", Name: "Trips", MTimeNS: 1,
		},
		{
			Kind: scanner.CatalogEntryDirectory, RelativePath: "Trips/Nested",
			ParentPath: "Trips", Name: "Nested", MTimeNS: 2,
		},
		{
			Kind: scanner.CatalogEntryAsset, RelativePath: "Trips/上海-Photo.JPG",
			ParentPath: "Trips", Name: "上海-Photo.JPG", MTimeNS: 100,
			AssetKind: scanner.AssetKindImage, MediaFormat: scanner.MediaFormatJPEG,
			MIMEType: "image/jpeg", SizeBytes: 10,
		},
		{
			Kind: scanner.CatalogEntryAsset, RelativePath: "Trips/Nested/Photo-Straße.jpg",
			ParentPath: "Trips/Nested", Name: "Photo-Straße.jpg", MTimeNS: 200,
			AssetKind: scanner.AssetKindImage, MediaFormat: scanner.MediaFormatJPEG,
			MIMEType: "image/jpeg", SizeBytes: 20,
		},
		{
			Kind: scanner.CatalogEntryAsset, RelativePath: "100%_real.jpg",
			ParentPath: "", Name: "100%_real.jpg", MTimeNS: 300,
			AssetKind: scanner.AssetKindImage, MediaFormat: scanner.MediaFormatJPEG,
			MIMEType: "image/jpeg", SizeBytes: 30,
		},
		{
			Kind: scanner.CatalogEntryAsset, RelativePath: "café.jpg",
			ParentPath: "", Name: "café.jpg", MTimeNS: 400,
			AssetKind: scanner.AssetKindImage, MediaFormat: scanner.MediaFormatJPEG,
			MIMEType: "image/jpeg", SizeBytes: 40,
		},
	})
	secondLibraryID := seedSearchCatalog(t, store, "第二库", "second", []scanner.CatalogEntry{
		{Kind: scanner.CatalogEntryDirectory},
		{
			Kind: scanner.CatalogEntryAsset, RelativePath: "上海-other.jpg",
			ParentPath: "", Name: "上海-other.jpg", MTimeNS: 500,
			AssetKind: scanner.AssetKindImage, MediaFormat: scanner.MediaFormatJPEG,
			MIMEType: "image/jpeg", SizeBytes: 50,
		},
	})
	service := catalogServiceForStore(t, store)

	query := " 上海 PHOTO "
	wholeLibrary, err := service.ListAssets(context.Background(), catalog.AssetRequest{
		LibraryID: firstLibraryID, SearchQuery: &query,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(wholeLibrary.Items) != 1 ||
		wholeLibrary.Items[0].RelativePath != "Trips/上海-Photo.JPG" {
		t.Fatalf("whole-library search = %#v", wholeLibrary)
	}

	var tripsID int64
	if err := store.db.QueryRowContext(context.Background(), `
        SELECT id FROM directories
        WHERE library_id = ? AND relative_path = 'Trips'`,
		firstLibraryID,
	).Scan(&tripsID); err != nil {
		t.Fatal(err)
	}
	strasse := "STRASSE"
	direct, err := service.ListAssets(context.Background(), catalog.AssetRequest{
		LibraryID: firstLibraryID, DirectoryID: tripsID, DirectorySet: true,
		SearchQuery: &strasse,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(direct.Items) != 0 {
		t.Fatalf("direct directory search = %#v", direct)
	}
	recursive, err := service.ListAssets(context.Background(), catalog.AssetRequest{
		LibraryID: firstLibraryID, DirectoryID: tripsID, DirectorySet: true,
		Recursive: true, RecursiveSet: true, SearchQuery: &strasse,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(recursive.Items) != 1 ||
		recursive.Items[0].RelativePath != "Trips/Nested/Photo-Straße.jpg" {
		t.Fatalf("recursive directory search = %#v", recursive)
	}
	pathAndName := "trips STRASSE"
	mixedFields, err := service.ListAssets(context.Background(), catalog.AssetRequest{
		LibraryID: firstLibraryID, SearchQuery: &pathAndName,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(mixedFields.Items) != 1 ||
		mixedFields.Items[0].RelativePath != "Trips/Nested/Photo-Straße.jpg" {
		t.Fatalf("mixed-field AND search = %#v", mixedFields)
	}

	for search, want := range map[string]string{
		"上":          "Trips/上海-Photo.JPG",
		"%_":         "100%_real.jpg",
		"cafe\u0301": "café.jpg",
	} {
		page, err := service.ListAssets(context.Background(), catalog.AssetRequest{
			LibraryID: firstLibraryID, SearchQuery: &search,
		})
		if err != nil {
			t.Fatalf("search %q: %v", search, err)
		}
		if len(page.Items) != 1 || page.Items[0].RelativePath != want {
			t.Fatalf("search %q page = %#v", search, page)
		}
	}
	unaccented := "cafe"
	page, err := service.ListAssets(context.Background(), catalog.AssetRequest{
		LibraryID: firstLibraryID, SearchQuery: &unaccented,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 0 {
		t.Fatalf("unaccented search = %#v", page)
	}

	from, before := int64(150), int64(250)
	photo := "photo"
	filtered, err := service.ListAssets(context.Background(), catalog.AssetRequest{
		LibraryID: firstLibraryID, SearchQuery: &photo,
		Kinds:          []catalog.AssetKind{catalog.KindImage},
		ModifiedFromNS: &from, ModifiedBeforeNS: &before,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered.Items) != 1 ||
		filtered.Items[0].RelativePath != "Trips/Nested/Photo-Straße.jpg" {
		t.Fatalf("filtered search = %#v", filtered)
	}

	shanghai := "上海"
	global, err := service.SearchAssets(context.Background(), catalog.GlobalSearchRequest{
		SearchQuery: shanghai, Limit: 1, Sort: catalog.SortName,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(global.Items) != 1 || global.NextCursor == "" {
		t.Fatalf("global first page = %#v", global)
	}
	next, err := service.SearchAssets(context.Background(), catalog.GlobalSearchRequest{
		SearchQuery: shanghai, Limit: 1, Sort: catalog.SortName,
		Cursor: global.NextCursor,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(next.Items) != 1 || next.Items[0].LibraryID == global.Items[0].LibraryID {
		t.Fatalf("global second page = %#v", next)
	}
	otherQuery := "other"
	if _, err := service.SearchAssets(context.Background(), catalog.GlobalSearchRequest{
		SearchQuery: otherQuery, Limit: 1, Sort: catalog.SortName,
		Cursor: global.NextCursor,
	}); !errors.Is(err, catalog.ErrInvalidCursor) {
		t.Fatalf("cross-query global cursor error = %v", err)
	}

	if _, err := store.db.ExecContext(context.Background(), `
        UPDATE libraries SET status = 'offline' WHERE id = ?`,
		secondLibraryID,
	); err != nil {
		t.Fatal(err)
	}
	offline, err := service.SearchAssets(context.Background(), catalog.GlobalSearchRequest{
		SearchQuery: shanghai,
	})
	if err != nil {
		t.Fatal(err)
	}
	foundOffline := false
	for _, item := range offline.Items {
		if item.LibraryID == secondLibraryID {
			foundOffline = item.Availability == catalog.SourceOffline
		}
	}
	if !foundOffline {
		t.Fatalf("offline global search = %#v", offline)
	}

	if _, err := store.db.ExecContext(context.Background(), `
        UPDATE libraries SET current_generation = current_generation + 1 WHERE id = ?`,
		secondLibraryID,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := service.SearchAssets(context.Background(), catalog.GlobalSearchRequest{
		SearchQuery: shanghai, Limit: 1, Sort: catalog.SortName,
		Cursor: global.NextCursor,
	}); !errors.Is(err, catalog.ErrInvalidCursor) {
		t.Fatalf("advanced global revision cursor error = %v", err)
	}
}

func TestCatalogSearchIndexIntegrityRepairIsCancellableAndPreservesAssets(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedSearchCatalog(t, store, "Repair", "repair", []scanner.CatalogEntry{
		{Kind: scanner.CatalogEntryDirectory},
		{
			Kind: scanner.CatalogEntryAsset, RelativePath: "repair-target.jpg",
			ParentPath: "", Name: "repair-target.jpg", MTimeNS: 100,
			AssetKind: scanner.AssetKindImage, MediaFormat: scanner.MediaFormatJPEG,
			MIMEType: "image/jpeg", SizeBytes: 10,
		},
	})
	var assetID int64
	var searchNameKey, searchPathKey string
	if err := store.db.QueryRowContext(context.Background(), `
        SELECT id, search_name_key, search_path_key
        FROM assets
        WHERE library_id = ? AND relative_path = 'repair-target.jpg'`,
		libraryID,
	).Scan(&assetID, &searchNameKey, &searchPathKey); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(context.Background(), `
        INSERT INTO asset_search(
            asset_search, rowid, search_name_key, search_path_key
        ) VALUES('delete', ?, ?, ?)`,
		assetID, searchNameKey, searchPathKey,
	); err != nil {
		t.Fatalf("corrupt derived search index: %v", err)
	}

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if err := store.ensureCatalogSearchIndex(cancelled); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled search-index repair error = %v", err)
	}
	if err := store.ensureCatalogSearchIndex(context.Background()); err != nil {
		t.Fatalf("repair derived search index: %v", err)
	}

	service := catalogServiceForStore(t, store)
	query := "repair-target"
	page, err := service.ListAssets(context.Background(), catalog.AssetRequest{
		LibraryID: libraryID, SearchQuery: &query,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].ID != assetID {
		t.Fatalf("repaired search page = %#v", page)
	}
	var assets int
	if err := store.db.QueryRowContext(context.Background(), `
        SELECT count(*) FROM assets WHERE library_id = ?`,
		libraryID,
	).Scan(&assets); err != nil {
		t.Fatal(err)
	}
	if assets != 1 {
		t.Fatalf("assets after derived-index repair = %d, want 1", assets)
	}
}

func TestCatalogSearchIndexRepairFailsClosedWithoutDeletingCatalog(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedSearchCatalog(t, store, "Failed repair", "failed-repair", []scanner.CatalogEntry{
		{Kind: scanner.CatalogEntryDirectory},
		{
			Kind: scanner.CatalogEntryAsset, RelativePath: "preserved.jpg",
			ParentPath: "", Name: "preserved.jpg", MTimeNS: 100,
			AssetKind: scanner.AssetKindImage, MediaFormat: scanner.MediaFormatJPEG,
			MIMEType: "image/jpeg", SizeBytes: 10,
		},
	})
	if _, err := store.db.ExecContext(
		context.Background(), `DROP TABLE asset_search`,
	); err != nil {
		t.Fatal(err)
	}
	err := store.ensureCatalogSearchIndex(context.Background())
	if err == nil || !strings.Contains(err.Error(), "rebuild catalog search index") {
		t.Fatalf("missing-index repair error = %v", err)
	}
	var assets int
	if err := store.db.QueryRowContext(context.Background(), `
        SELECT count(*) FROM assets WHERE library_id = ?`,
		libraryID,
	).Scan(&assets); err != nil {
		t.Fatal(err)
	}
	if assets != 1 {
		t.Fatalf("assets after failed derived-index repair = %d, want 1", assets)
	}
}

func TestOpenRepairsInconsistentCatalogSearchIndex(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "search-repair.db")
	store, err := Open(context.Background(), filename, Options{MaxBatchSize: 16})
	if err != nil {
		t.Fatal(err)
	}
	libraryID := seedSearchCatalog(t, store, "Startup repair", "startup-repair", []scanner.CatalogEntry{
		{Kind: scanner.CatalogEntryDirectory},
		{
			Kind: scanner.CatalogEntryAsset, RelativePath: "startup-target.jpg",
			ParentPath: "", Name: "startup-target.jpg", MTimeNS: 100,
			AssetKind: scanner.AssetKindImage, MediaFormat: scanner.MediaFormatJPEG,
			MIMEType: "image/jpeg", SizeBytes: 10,
		},
	})
	if _, err := store.db.ExecContext(context.Background(), `
        INSERT INTO asset_search(
            asset_search, rowid, search_name_key, search_path_key
        )
        SELECT 'delete', id, search_name_key, search_path_key
        FROM assets
        WHERE library_id = ? AND relative_path = 'startup-target.jpg'`,
		libraryID,
	); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := Open(context.Background(), filename, Options{MaxBatchSize: 16})
	if err != nil {
		t.Fatalf("reopen and repair search index: %v", err)
	}
	t.Cleanup(func() {
		if err := reopened.Close(); err != nil {
			t.Errorf("close reopened store: %v", err)
		}
	})
	service := catalogServiceForStore(t, reopened)
	query := "startup-target"
	page, err := service.ListAssets(context.Background(), catalog.AssetRequest{
		LibraryID: libraryID, SearchQuery: &query,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].RelativePath != "startup-target.jpg" {
		t.Fatalf("startup-repaired search page = %#v", page)
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
