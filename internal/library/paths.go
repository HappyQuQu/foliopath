package library

import (
	"bytes"
	"container/heap"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"sort"
	"unicode/utf8"

	"golang.org/x/text/collate"
	"golang.org/x/text/language"
)

const (
	DefaultPathPageSize = 50
	MaxPathPageSize     = 200
	MaxParentRunes      = 4096
	MaxCursorBytes      = 2048

	mediaRootDisplayName = "Media root"
	pathCursorVersion    = 1
)

var (
	ErrInvalidParent       = errors.New("invalid library path parent")
	ErrInvalidCursor       = errors.New("invalid library path cursor")
	ErrParentUnavailable   = errors.New("library path parent unavailable")
	ErrParentSymlink       = errors.New("library path parent is a symbolic link")
	ErrParentMountBoundary = errors.New("library path parent crosses a mount boundary")
)

type SelectionBlockedReason string

const (
	SelectionBlockedUnreadable    SelectionBlockedReason = "unreadable"
	SelectionBlockedSymlink       SelectionBlockedReason = "symlink"
	SelectionBlockedMountBoundary SelectionBlockedReason = "mount_boundary"
	SelectionBlockedUnavailable   SelectionBlockedReason = "unavailable"
)

// DirectoryCandidate is produced by the filesystem adapter after it has
// inspected one direct child without following a symlink or crossing a mount.
type DirectoryCandidate struct {
	Name          string
	HasChildren   bool
	BlockedReason SelectionBlockedReason
}

// DirectorySource is the capability-owned filesystem port. Implementations
// stream direct children and must keep all real filesystem I/O behind their
// boundary.
type DirectorySource interface {
	EnumerateDirectories(
		context.Context,
		string,
		func(DirectoryCandidate) error,
	) error
}

type ListPathParams struct {
	Parent string
	Cursor string
	Limit  int
}

type PathLocation struct {
	Name         string
	RelativePath string
}

type PathEntry struct {
	Name          string
	RelativePath  string
	HasChildren   bool
	Selectable    bool
	BlockedReason SelectionBlockedReason
}

type PathPage struct {
	Location    PathLocation
	Breadcrumbs []PathLocation
	Items       []PathEntry
	NextCursor  string
}

type PathServiceOptions struct {
	// CursorKey is injectable only for deterministic tests. Production callers
	// omit it and receive a fresh cryptographically random process key.
	CursorKey []byte
}

type PathService struct {
	source DirectorySource
	aead   cipher.AEAD
}

func NewPathService(source DirectorySource, options PathServiceOptions) (*PathService, error) {
	if source == nil {
		return nil, errors.New("library directory source is required")
	}
	key := append([]byte(nil), options.CursorKey...)
	if len(key) == 0 {
		key = make([]byte, 32)
		if _, err := io.ReadFull(rand.Reader, key); err != nil {
			return nil, fmt.Errorf("generate library path cursor key: %w", err)
		}
	}
	if len(key) != 32 {
		return nil, errors.New("library path cursor key must be 32 bytes")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("construct library path cursor cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("construct library path cursor AEAD: %w", err)
	}
	return &PathService{source: source, aead: aead}, nil
}

func (service *PathService) ListPaths(
	ctx context.Context,
	params ListPathParams,
) (PathPage, error) {
	if ctx == nil {
		return PathPage{}, errors.New("library path context is required")
	}
	if utf8.RuneCountInString(params.Parent) > MaxParentRunes {
		return PathPage{}, ErrInvalidParent
	}
	if params.Cursor != "" &&
		(len(params.Cursor) < 8 || len(params.Cursor) > MaxCursorBytes) {
		return PathPage{}, ErrInvalidCursor
	}
	parent, err := NormalizeRoot(params.Parent)
	if err != nil {
		return PathPage{}, fmt.Errorf("%w: %w", ErrInvalidParent, err)
	}
	limit := params.Limit
	if limit == 0 {
		limit = DefaultPathPageSize
	}
	if limit < 1 || limit > MaxPathPageSize {
		return PathPage{}, ErrInvalidParent
	}

	after := ""
	if params.Cursor != "" {
		after, err = service.decodeCursor(params.Cursor, parent)
		if err != nil {
			return PathPage{}, err
		}
	}

	collator := collate.New(language.Und, collate.Loose, collate.Numeric)
	var afterKey []byte
	if after != "" {
		afterKey = naturalKey(collator, after)
	}
	selected := &candidateHeap{maximum: true}
	heap.Init(selected)
	err = service.source.EnumerateDirectories(
		ctx,
		parent,
		func(candidate DirectoryCandidate) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			if candidate.Name == "" || path.Base(candidate.Name) != candidate.Name {
				return errors.New("directory source returned an invalid child name")
			}
			relative, normalizeErr := NormalizeRoot(joinRoot(parent, candidate.Name))
			if normalizeErr != nil {
				return nil
			}
			ranked := rankedCandidate{
				candidate: candidate,
				relative:  relative,
				sortKey:   naturalKey(collator, candidate.Name),
			}
			if after != "" && comparePosition(ranked.sortKey, candidate.Name, afterKey, after) <= 0 {
				return nil
			}
			heap.Push(selected, ranked)
			if selected.Len() > limit+1 {
				heap.Pop(selected)
			}
			return nil
		},
	)
	if err != nil {
		return PathPage{}, err
	}

	ranked := make([]rankedCandidate, selected.Len())
	for index := len(ranked) - 1; index >= 0; index-- {
		ranked[index] = heap.Pop(selected).(rankedCandidate)
	}
	sort.Slice(ranked, func(left, right int) bool {
		return compareRanked(ranked[left], ranked[right]) < 0
	})

	hasNext := len(ranked) > limit
	if hasNext {
		ranked = ranked[:limit]
	}
	items := make([]PathEntry, 0, len(ranked))
	for _, item := range ranked {
		items = append(items, PathEntry{
			Name:          item.candidate.Name,
			RelativePath:  item.relative,
			HasChildren:   item.candidate.HasChildren,
			Selectable:    item.candidate.BlockedReason == "",
			BlockedReason: item.candidate.BlockedReason,
		})
	}

	nextCursor := ""
	if hasNext && len(ranked) > 0 {
		nextCursor, err = service.encodeCursor(parent, ranked[len(ranked)-1].candidate.Name)
		if err != nil {
			return PathPage{}, err
		}
	}
	location, breadcrumbs := pathLocations(parent)
	return PathPage{
		Location:    location,
		Breadcrumbs: breadcrumbs,
		Items:       items,
		NextCursor:  nextCursor,
	}, nil
}

type pathCursor struct {
	Version int    `json:"v"`
	Parent  string `json:"p"`
	After   string `json:"a"`
}

func (service *PathService) encodeCursor(parent, after string) (string, error) {
	plaintext, err := json.Marshal(pathCursor{
		Version: pathCursorVersion,
		Parent:  parentFingerprint(parent),
		After:   after,
	})
	if err != nil {
		return "", fmt.Errorf("encode library path cursor: %w", err)
	}
	nonce := make([]byte, service.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate library path cursor nonce: %w", err)
	}
	sealed := service.aead.Seal(nonce, nonce, plaintext, []byte("foliopath:library-path:v1"))
	return base64.RawURLEncoding.EncodeToString(sealed), nil
}

func (service *PathService) decodeCursor(encoded, parent string) (string, error) {
	sealed, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil || len(sealed) <= service.aead.NonceSize() {
		return "", ErrInvalidCursor
	}
	nonce := sealed[:service.aead.NonceSize()]
	plaintext, err := service.aead.Open(
		nil,
		nonce,
		sealed[service.aead.NonceSize():],
		[]byte("foliopath:library-path:v1"),
	)
	if err != nil {
		return "", ErrInvalidCursor
	}
	var cursor pathCursor
	if err := json.Unmarshal(plaintext, &cursor); err != nil ||
		cursor.Version != pathCursorVersion ||
		cursor.Parent != parentFingerprint(parent) ||
		cursor.After == "" {
		return "", ErrInvalidCursor
	}
	if _, err := NormalizeRoot(joinRoot(parent, cursor.After)); err != nil ||
		path.Base(cursor.After) != cursor.After {
		return "", ErrInvalidCursor
	}
	return cursor.After, nil
}

func parentFingerprint(parent string) string {
	sum := sha256.Sum256([]byte("library-path-parent\x00" + parent))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func pathLocations(parent string) (PathLocation, []PathLocation) {
	root := PathLocation{Name: mediaRootDisplayName, RelativePath: ""}
	breadcrumbs := []PathLocation{root}
	if parent == "" {
		return root, breadcrumbs
	}
	current := ""
	for _, component := range splitRoot(parent) {
		current = joinRoot(current, component)
		breadcrumbs = append(breadcrumbs, PathLocation{
			Name:         component,
			RelativePath: current,
		})
	}
	return breadcrumbs[len(breadcrumbs)-1], breadcrumbs
}

func splitRoot(relative string) []string {
	if relative == "" {
		return nil
	}
	var components []string
	for relative != "" {
		parent, base := path.Split(relative)
		components = append([]string{base}, components...)
		relative = path.Clean(parent)
		if relative == "." || relative == "/" {
			relative = ""
		}
	}
	return components
}

func joinRoot(parent, child string) string {
	if parent == "" {
		return child
	}
	return parent + "/" + child
}

type rankedCandidate struct {
	candidate DirectoryCandidate
	relative  string
	sortKey   []byte
}

func naturalKey(collator *collate.Collator, value string) []byte {
	buffer := &collate.Buffer{}
	return append([]byte(nil), collator.KeyFromString(buffer, value)...)
}

func compareRanked(left, right rankedCandidate) int {
	return comparePosition(left.sortKey, left.candidate.Name, right.sortKey, right.candidate.Name)
}

func comparePosition(leftKey []byte, leftName string, rightKey []byte, rightName string) int {
	if compared := bytes.Compare(leftKey, rightKey); compared != 0 {
		return compared
	}
	return bytes.Compare([]byte(leftName), []byte(rightName))
}

// candidateHeap is a max heap so streaming enumeration retains only the
// smallest page-size candidates after the keyset position.
type candidateHeap struct {
	items   []rankedCandidate
	maximum bool
}

func (values candidateHeap) Len() int {
	return len(values.items)
}

func (values candidateHeap) Less(left, right int) bool {
	compared := compareRanked(values.items[left], values.items[right])
	if values.maximum {
		return compared > 0
	}
	return compared < 0
}

func (values candidateHeap) Swap(left, right int) {
	values.items[left], values.items[right] = values.items[right], values.items[left]
}

func (values *candidateHeap) Push(value any) {
	values.items = append(values.items, value.(rankedCandidate))
}

func (values *candidateHeap) Pop() any {
	last := len(values.items) - 1
	value := values.items[last]
	values.items = values.items[:last]
	return value
}
