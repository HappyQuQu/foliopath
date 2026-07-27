package library

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type repositoryStub struct {
	createParams CreateParams
	renameID     int64
	renameName   string
}

func (stub *repositoryStub) CreateLibrary(
	_ context.Context,
	params CreateParams,
) (Library, error) {
	stub.createParams = params
	return Library{Name: params.Name, RootRelativePath: params.RootRelativePath}, nil
}

func (stub *repositoryStub) GetLibrary(context.Context, int64) (Library, error) {
	return Library{}, ErrNotFound
}

func (stub *repositoryStub) ListLibraries(context.Context) ([]Library, error) {
	return []Library{}, nil
}

func (stub *repositoryStub) RenameLibrary(
	_ context.Context,
	id int64,
	name string,
) (Library, error) {
	stub.renameID = id
	stub.renameName = name
	return Library{ID: id, Name: name}, nil
}

func TestNormalizeNameUsesCanonicalDisplayAndCompatibilityFoldedKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input       string
		wantDisplay string
		wantKey     string
	}{
		{input: " Family ", wantDisplay: "Family", wantKey: "family"},
		{input: "Straße", wantDisplay: "Straße", wantKey: "strasse"},
		{input: "Ｓｕｍｍｅｒ", wantDisplay: "Ｓｕｍｍｅｒ", wantKey: "summer"},
		{input: "Cafe\u0301", wantDisplay: "Café", wantKey: "café"},
		{input: "家庭相册", wantDisplay: "家庭相册", wantKey: "家庭相册"},
	}
	for _, test := range tests {
		display, key, err := NormalizeName(test.input)
		if err != nil {
			t.Errorf("NormalizeName(%q) error = %v", test.input, err)
			continue
		}
		if display != test.wantDisplay || key != test.wantKey {
			t.Errorf(
				"NormalizeName(%q) = (%q, %q), want (%q, %q)",
				test.input,
				display,
				key,
				test.wantDisplay,
				test.wantKey,
			)
		}
	}
}

func TestNormalizeNameRejectsUnsafeOrOutOfContractValues(t *testing.T) {
	t.Parallel()

	for _, value := range []string{
		"",
		"   ",
		"Family\nArchive",
		"Family\x00Archive",
		string([]byte{0xff}),
		strings.Repeat("界", maxLibraryNameRunes+1),
	} {
		if _, _, err := NormalizeName(value); !errors.Is(err, ErrInvalidName) {
			t.Errorf("NormalizeName(%q) error = %v, want ErrInvalidName", value, err)
		}
	}
}

func TestServicePassesOnlyNormalizedValuesToRepository(t *testing.T) {
	t.Parallel()

	repository := &repositoryStub{}
	service, err := NewService(repository)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Create(
		context.Background(),
		"  Cafe\u0301  ",
		"家庭/2026",
	); err != nil {
		t.Fatal(err)
	}
	if repository.createParams.Name != "Café" ||
		repository.createParams.RootRelativePath != "家庭/2026" {
		t.Fatalf("create params = %#v", repository.createParams)
	}
	if _, err := service.Rename(context.Background(), 7, "  Ａrchive  "); err != nil {
		t.Fatal(err)
	}
	if repository.renameID != 7 || repository.renameName != "Ａrchive" {
		t.Fatalf("rename = (%d, %q)", repository.renameID, repository.renameName)
	}
}

func TestNormalizeRoot(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "allowed root", input: "", want: ""},
		{name: "nested", input: "family/2026", want: "family/2026"},
		{name: "literal percent", input: "旅行 100%/literal%20", want: "旅行 100%/literal%20"},
		{name: "absolute", input: "/etc", wantErr: true},
		{name: "traversal", input: "family/../work", wantErr: true},
		{name: "encoded traversal", input: "%252e%252e", wantErr: true},
		{name: "encoded separator", input: "family%252fwork", wantErr: true},
		{name: "empty component", input: "family//work", wantErr: true},
		{name: "backslash", input: `family\work`, wantErr: true},
		{name: "invalid UTF-8", input: string([]byte{'b', 'a', 'd', 0xff}), wantErr: true},
		{name: "too long", input: strings.Repeat("界", maxLibraryRootRunes+1), wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := NormalizeRoot(test.input)
			if (err != nil) != test.wantErr {
				t.Fatalf("NormalizeRoot(%q) error = %v, wantErr %v", test.input, err, test.wantErr)
			}
			if got != test.want {
				t.Fatalf("NormalizeRoot(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}

func TestRootsOverlapUsesComponentBoundaries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		left  string
		right string
		want  bool
	}{
		{left: "family", right: "family", want: true},
		{left: "family", right: "family/2026", want: true},
		{left: "family/2026", right: "family", want: true},
		{left: "", right: "family", want: true},
		{left: "family", right: "", want: true},
		{left: "family", right: "family-archive", want: false},
		{left: "旅行", right: "旅行者", want: false},
		{left: "旅行", right: "旅行/日本", want: true},
	}
	for _, test := range tests {
		if got := RootsOverlap(test.left, test.right); got != test.want {
			t.Errorf("RootsOverlap(%q, %q) = %t, want %t",
				test.left, test.right, got, test.want)
		}
	}
}
