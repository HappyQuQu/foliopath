package catalog

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/media"
)

type repositoryStub struct {
	scope          Scope
	directories    []Directory
	assets         []Asset
	directoryCalls []DirectoryListParams
	assetCalls     []AssetListParams
	resolveErr     error
	lineage        DirectoryLineage
	lineageErr     error
}

func (stub *repositoryStub) ResolveScope(
	ctx context.Context,
	libraryID, directoryID int64,
) (Scope, error) {
	if err := ctx.Err(); err != nil {
		return Scope{}, err
	}
	if stub.resolveErr != nil {
		return Scope{}, stub.resolveErr
	}
	scope := stub.scope
	if directoryID == 0 || directoryID == scope.RootDirectoryID {
		scope.DirectoryID = scope.RootDirectoryID
	} else {
		scope.DirectoryID = directoryID
	}
	return scope, nil
}

func (stub *repositoryStub) ListDirectoryPage(
	ctx context.Context,
	params DirectoryListParams,
) ([]Directory, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	stub.directoryCalls = append(stub.directoryCalls, params)
	return append([]Directory(nil), stub.directories...), nil
}

func (stub *repositoryStub) ListAssetPage(
	ctx context.Context,
	params AssetListParams,
) ([]Asset, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	stub.assetCalls = append(stub.assetCalls, params)
	return append([]Asset(nil), stub.assets...), nil
}

func (stub *repositoryStub) GetDirectoryLineage(
	ctx context.Context,
	_ int64,
	_ int,
) (DirectoryLineage, error) {
	if err := ctx.Err(); err != nil {
		return DirectoryLineage{}, err
	}
	return stub.lineage, stub.lineageErr
}

func newTestService(t *testing.T, repository Repository) *Service {
	t.Helper()
	service, err := NewService(repository, []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func TestNormalizeRootScope(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name                     string
		rootID, selectedID, want int64
		wantErr                  bool
	}{
		{name: "unindexed logical root", rootID: 0, selectedID: 0, want: 0},
		{name: "indexed logical root", rootID: 4, selectedID: 4, want: 0},
		{name: "child", rootID: 4, selectedID: 8, want: 8},
		{name: "missing root", rootID: 0, selectedID: 8, wantErr: true},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := NormalizeRootScope(test.rootID, test.selectedID)
			if (err != nil) != test.wantErr {
				t.Fatalf("NormalizeRootScope() error = %v, wantErr %v", err, test.wantErr)
			}
			if got != test.want {
				t.Fatalf("NormalizeRootScope() = %d, want %d", got, test.want)
			}
		})
	}
}

func TestNaturalNameKeyUsesNumericOrdering(t *testing.T) {
	t.Parallel()
	names := []string{"Album 10", "Album 2", "Album 1"}
	slices.SortFunc(names, func(left, right string) int {
		return slices.Compare(NaturalNameKey(left), NaturalNameKey(right))
	})
	want := []string{"Album 1", "Album 2", "Album 10"}
	if !slices.Equal(names, want) {
		t.Fatalf("natural order = %v, want %v", names, want)
	}
}

func TestDirectoryCursorBindsNormalizedRootAndGeneration(t *testing.T) {
	repository := &repositoryStub{
		scope: Scope{LibraryID: 7, RootDirectoryID: 11, Generation: 3},
		directories: []Directory{
			{ID: 20, LibraryID: 7, Name: "Album 1", NaturalNameKey: NaturalNameKey("Album 1")},
			{ID: 21, LibraryID: 7, Name: "Album 2", NaturalNameKey: NaturalNameKey("Album 2")},
		},
	}
	service := newTestService(t, repository)
	first, err := service.ListDirectories(context.Background(), DirectoryRequest{
		LibraryID: 7,
		Limit:     1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Items) != 1 || first.NextCursor == "" {
		t.Fatalf("first page = %#v", first)
	}

	repository.directories = nil
	second, err := service.ListDirectories(context.Background(), DirectoryRequest{
		LibraryID: 7, ParentDirectoryID: 11,
		Cursor: first.NextCursor, Limit: 1,
	})
	if err != nil {
		t.Fatalf("explicit root cursor failed: %v", err)
	}
	if len(second.Items) != 0 || len(repository.directoryCalls) != 2 {
		t.Fatalf("second page = %#v, calls = %d", second, len(repository.directoryCalls))
	}
	after := repository.directoryCalls[1].After
	if after == nil || after.ID != 20 || after.Name != "Album 1" {
		t.Fatalf("decoded position = %#v", after)
	}

	repository.scope.Generation = 4
	_, err = service.ListDirectories(context.Background(), DirectoryRequest{
		LibraryID: 7, Cursor: first.NextCursor, Limit: 1,
	})
	if !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("generation change error = %v", err)
	}
	repository.scope.Generation = 3
	tamperAt := len(first.NextCursor) / 2
	replacement := byte('A')
	if first.NextCursor[tamperAt] == replacement {
		replacement = 'B'
	}
	tampered := first.NextCursor[:tamperAt] + string(replacement) + first.NextCursor[tamperAt+1:]
	_, err = service.ListDirectories(context.Background(), DirectoryRequest{
		LibraryID: 7, Cursor: tampered, Limit: 1,
	})
	if !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("tampered cursor error = %v", err)
	}
	_, err = service.ListDirectories(context.Background(), DirectoryRequest{
		LibraryID: 7, ParentDirectoryID: 12,
		Cursor: first.NextCursor, Limit: 1,
	})
	if !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("scope change error = %v", err)
	}
}

func TestAssetQueryDefaultsCanonicalizesKindsAndBindsCursor(t *testing.T) {
	repository := &repositoryStub{
		scope: Scope{LibraryID: 9, RootDirectoryID: 30, Generation: 5},
		assets: []Asset{
			{
				ID: 40, LibraryID: 9, DirectoryID: 30, RelativePath: "photo-2.jpg",
				Name: "photo-2.jpg", NaturalNameKey: NaturalNameKey("photo-2.jpg"),
				ModifiedAtNS: 10, LibraryName: "Family", Kind: KindImage,
				SourceFingerprint: "v1:1:10", ProbeStatus: media.ProbePending,
				PlaybackStatus: media.PlaybackUnknown, ThumbnailStatus: "pending",
			},
			{
				ID: 41, LibraryID: 9, DirectoryID: 30, RelativePath: "photo-10.jpg",
				Name: "photo-10.jpg", NaturalNameKey: NaturalNameKey("photo-10.jpg"),
				ModifiedAtNS: 11, LibraryName: "Family", Kind: KindImage,
				SourceFingerprint: "v1:1:11", ProbeStatus: media.ProbePending,
				PlaybackStatus: media.PlaybackUnknown, ThumbnailStatus: "pending",
			},
		},
	}
	service := newTestService(t, repository)
	first, err := service.ListAssets(context.Background(), AssetRequest{
		LibraryID: 9,
		Kinds:     []AssetKind{KindVideo, KindImage, KindVideo},
		Limit:     1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.NextCursor == "" || len(repository.assetCalls) != 1 {
		t.Fatalf("first page = %#v", first)
	}
	query := repository.assetCalls[0].Query
	if query.Sort != SortName || query.Order != OrderAsc ||
		!slices.Equal(query.Kinds, []AssetKind{KindImage, KindVideo}) {
		t.Fatalf("normalized direct query = %#v", query)
	}

	repository.assets = nil
	if _, err := service.ListAssets(context.Background(), AssetRequest{
		LibraryID: 9, DirectoryID: 30,
		Kinds:  []AssetKind{KindImage, KindVideo},
		Cursor: first.NextCursor, Limit: 1,
	}); err != nil {
		t.Fatalf("equivalent root/kind query rejected cursor: %v", err)
	}
	if repository.assetCalls[1].After == nil || repository.assetCalls[1].After.ID != 40 {
		t.Fatalf("decoded asset position = %#v", repository.assetCalls[1].After)
	}

	_, err = service.ListAssets(context.Background(), AssetRequest{
		LibraryID: 9, Recursive: true,
		Kinds:  []AssetKind{KindImage, KindVideo},
		Cursor: first.NextCursor, Limit: 1,
	})
	if !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("recursive scope cursor error = %v", err)
	}
}

func TestAssetRecursiveDefaultAndSearchIsNotSilentlyIgnored(t *testing.T) {
	repository := &repositoryStub{
		scope: Scope{LibraryID: 2, RootDirectoryID: 3, Generation: 1},
	}
	service := newTestService(t, repository)
	if _, err := service.ListAssets(context.Background(), AssetRequest{
		LibraryID: 2, Recursive: true,
	}); err != nil {
		t.Fatal(err)
	}
	query := repository.assetCalls[0].Query
	if query.Sort != SortModifiedAt || query.Order != OrderDesc {
		t.Fatalf("recursive default = %#v", query)
	}

	search := "holiday"
	_, err := service.ListAssets(context.Background(), AssetRequest{
		LibraryID: 2, SearchQuery: &search,
	})
	if !errors.Is(err, ErrSearchUnavailable) {
		t.Fatalf("search error = %v", err)
	}
	if len(repository.assetCalls) != 1 {
		t.Fatal("unavailable search reached repository")
	}
}

func TestCatalogRejectsInvalidAndCancelledRequests(t *testing.T) {
	repository := &repositoryStub{
		scope: Scope{LibraryID: 1, RootDirectoryID: 2, Generation: 1},
	}
	service := newTestService(t, repository)
	for _, request := range []AssetRequest{
		{LibraryID: 1, Limit: MaxPageSize + 1},
		{LibraryID: 1, Sort: "size"},
		{LibraryID: 1, Order: "sideways"},
		{LibraryID: 1, Kinds: []AssetKind{"raw"}},
		{LibraryID: 1, Cursor: strings.Repeat("a", MaxCursorBytes+1)},
	} {
		if _, err := service.ListAssets(context.Background(), request); err == nil {
			t.Fatalf("request %#v unexpectedly succeeded", request)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := service.ListDirectories(ctx, DirectoryRequest{LibraryID: 1}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled directory request error = %v", err)
	}
	if len(repository.directoryCalls) != 0 {
		t.Fatal("cancelled request reached repository")
	}
}

func TestDirectoryDetailMapsRootAndBuildsCompleteBreadcrumb(t *testing.T) {
	rootID, albumID := int64(10), int64(11)
	repository := &repositoryStub{lineage: DirectoryLineage{
		LibraryName: "Family",
		Items: []Directory{
			{ID: rootID, LibraryID: 3},
			{
				ID: albumID, LibraryID: 3, ParentID: &rootID,
				RelativePath: "Album", Name: "Album", HasChildren: true,
			},
		},
	}}
	service := newTestService(t, repository)
	detail, err := service.GetDirectory(context.Background(), albumID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.ID != albumID || len(detail.Breadcrumbs) != 2 ||
		detail.Breadcrumbs[0].Name != "Family" ||
		detail.Breadcrumbs[0].RelativePath != "" ||
		detail.Breadcrumbs[1].Name != "Album" {
		t.Fatalf("directory detail = %#v", detail)
	}
}

func TestDirectoryDetailFailsClosedForBrokenTopology(t *testing.T) {
	rootID, albumID := int64(10), int64(11)
	tests := []DirectoryLineage{
		{LibraryName: "Family"},
		{
			LibraryName: "Family",
			Items: []Directory{
				{ID: rootID, LibraryID: 3},
				{
					ID: albumID, LibraryID: 3, ParentID: &rootID,
					RelativePath: "Other/Album", Name: "Album",
				},
			},
		},
		{
			LibraryName: "Family",
			Items: []Directory{
				{ID: rootID, LibraryID: 3},
				{
					ID: rootID, LibraryID: 3, ParentID: &rootID,
					RelativePath: "Again", Name: "Again",
				},
			},
		},
	}
	for index, lineage := range tests {
		service := newTestService(t, &repositoryStub{lineage: lineage})
		if _, err := service.GetDirectory(context.Background(), albumID); !errors.Is(err, ErrInvalidTopology) {
			t.Fatalf("case %d error = %v", index, err)
		}
	}
}
