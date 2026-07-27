package library

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type directorySourceStub struct {
	candidates []DirectoryCandidate
	err        error
	calls      int
	parent     string
}

func (stub *directorySourceStub) EnumerateDirectories(
	ctx context.Context,
	parent string,
	visit func(DirectoryCandidate) error,
) error {
	stub.calls++
	stub.parent = parent
	for _, candidate := range stub.candidates {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := visit(candidate); err != nil {
			return err
		}
	}
	return stub.err
}

func TestPathServiceNaturallySortsAndPaginatesWithOpaqueBoundCursor(t *testing.T) {
	t.Parallel()

	source := &directorySourceStub{candidates: []DirectoryCandidate{
		{Name: "Album 10"},
		{Name: "旅行"},
		{Name: "Album 2", HasChildren: true},
		{Name: ".hidden"},
		{Name: "Album 1"},
	}}
	service := testPathService(t, source)
	first, err := service.ListPaths(context.Background(), ListPathParams{
		Parent: "family",
		Limit:  3,
	})
	if err != nil {
		t.Fatalf("ListPaths(first): %v", err)
	}
	assertPathNames(t, first.Items, ".hidden", "Album 1", "Album 2")
	if first.NextCursor == "" {
		t.Fatal("first page has no next cursor")
	}
	for _, plaintext := range []string{"family", "Album", "旅行"} {
		if strings.Contains(first.NextCursor, plaintext) {
			t.Fatalf("cursor exposed plaintext %q: %q", plaintext, first.NextCursor)
		}
	}

	second, err := service.ListPaths(context.Background(), ListPathParams{
		Parent: "family",
		Cursor: first.NextCursor,
		Limit:  3,
	})
	if err != nil {
		t.Fatalf("ListPaths(second): %v", err)
	}
	assertPathNames(t, second.Items, "Album 10", "旅行")
	if second.NextCursor != "" {
		t.Fatalf("last page cursor = %q, want empty", second.NextCursor)
	}
	if source.parent != "family" {
		t.Fatalf("source parent = %q, want family", source.parent)
	}
}

func TestPathServiceCursorRejectsTamperingAndDifferentParentOrKey(t *testing.T) {
	t.Parallel()

	source := &directorySourceStub{candidates: []DirectoryCandidate{
		{Name: "a"},
		{Name: "b"},
	}}
	service := testPathService(t, source)
	first, err := service.ListPaths(context.Background(), ListPathParams{Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if first.NextCursor == "" {
		t.Fatal("missing cursor")
	}

	tamperedBytes := []byte(first.NextCursor)
	tamperIndex := len(tamperedBytes) / 2
	if tamperedBytes[tamperIndex] == 'A' {
		tamperedBytes[tamperIndex] = 'B'
	} else {
		tamperedBytes[tamperIndex] = 'A'
	}
	tampered := string(tamperedBytes)
	for name, request := range map[string]ListPathParams{
		"tampered":         {Cursor: tampered, Limit: 1},
		"different parent": {Parent: "other", Cursor: first.NextCursor, Limit: 1},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := service.ListPaths(context.Background(), request); !errors.Is(err, ErrInvalidCursor) {
				t.Fatalf("ListPaths() error = %v, want ErrInvalidCursor", err)
			}
		})
	}

	other, err := NewPathService(source, PathServiceOptions{
		CursorKey: []byte("abcdef0123456789abcdef0123456789"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := other.ListPaths(context.Background(), ListPathParams{
		Cursor: first.NextCursor,
		Limit:  1,
	}); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("cursor decoded with another key: %v", err)
	}
}

func TestPathServiceBuildsBreadcrumbsAndBlockedEntries(t *testing.T) {
	t.Parallel()

	service := testPathService(t, &directorySourceStub{candidates: []DirectoryCandidate{{
		Name:          "external",
		BlockedReason: SelectionBlockedMountBoundary,
	}}})
	page, err := service.ListPaths(context.Background(), ListPathParams{
		Parent: "家庭/2026",
	})
	if err != nil {
		t.Fatal(err)
	}
	if page.Location.Name != "2026" || page.Location.RelativePath != "家庭/2026" {
		t.Fatalf("location = %#v", page.Location)
	}
	if len(page.Breadcrumbs) != 3 ||
		page.Breadcrumbs[0].RelativePath != "" ||
		page.Breadcrumbs[1].RelativePath != "家庭" ||
		page.Breadcrumbs[2].RelativePath != "家庭/2026" {
		t.Fatalf("breadcrumbs = %#v", page.Breadcrumbs)
	}
	if len(page.Items) != 1 ||
		page.Items[0].Selectable ||
		page.Items[0].BlockedReason != SelectionBlockedMountBoundary ||
		page.Items[0].RelativePath != "家庭/2026/external" {
		t.Fatalf("items = %#v", page.Items)
	}
}

func TestPathServiceRejectsInvalidInputBeforeFilesystemAccess(t *testing.T) {
	t.Parallel()

	source := &directorySourceStub{}
	service := testPathService(t, source)
	for _, test := range []struct {
		params ListPathParams
		want   error
	}{
		{params: ListPathParams{Parent: "../outside"}, want: ErrInvalidParent},
		{params: ListPathParams{Parent: "%252e%252e"}, want: ErrInvalidParent},
		{params: ListPathParams{Parent: `family\work`}, want: ErrInvalidParent},
		{
			params: ListPathParams{Parent: strings.Repeat("界", MaxParentRunes+1)},
			want:   ErrInvalidParent,
		},
		{params: ListPathParams{Limit: -1}, want: ErrInvalidParent},
		{params: ListPathParams{Limit: MaxPathPageSize + 1}, want: ErrInvalidParent},
		{params: ListPathParams{Cursor: "short"}, want: ErrInvalidCursor},
		{
			params: ListPathParams{Cursor: strings.Repeat("a", MaxCursorBytes+1)},
			want:   ErrInvalidCursor,
		},
	} {
		if _, err := service.ListPaths(context.Background(), test.params); !errors.Is(err, test.want) {
			t.Fatalf("ListPaths(%#v) error = %v, want %v", test.params, err, test.want)
		}
	}
	if source.calls != 0 {
		t.Fatalf("invalid inputs performed %d filesystem enumerations", source.calls)
	}
}

func TestPathServicePropagatesCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	service := testPathService(t, &directorySourceStub{candidates: []DirectoryCandidate{{Name: "a"}}})
	if _, err := service.ListPaths(ctx, ListPathParams{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("ListPaths() error = %v, want context.Canceled", err)
	}
}

func testPathService(t *testing.T, source DirectorySource) *PathService {
	t.Helper()
	service, err := NewPathService(source, PathServiceOptions{
		CursorKey: []byte("0123456789abcdef0123456789abcdef"),
	})
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func assertPathNames(t *testing.T, entries []PathEntry, names ...string) {
	t.Helper()
	if len(entries) != len(names) {
		t.Fatalf("entries = %#v, want names %#v", entries, names)
	}
	for index, name := range names {
		if entries[index].Name != name {
			t.Fatalf("entry %d name = %q, want %q", index, entries[index].Name, name)
		}
	}
}
