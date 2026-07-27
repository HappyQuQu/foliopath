package library

import (
	"bytes"
	"container/heap"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"
	"unicode/utf8"

	cursorcodec "github.com/HappyQuQu/foliopath/internal/cursor"
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
	SelectionBlockedOverlapping   SelectionBlockedReason = "overlapping_library"
	SelectionBlockedAncestor      SelectionBlockedReason = "ancestor_of_library"
	SelectionBlockedDescendant    SelectionBlockedReason = "descendant_of_library"
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

type LibraryReader interface {
	ListLibraries(context.Context) ([]Library, error)
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
	ConflictID    int64
	ConflictName  string
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
	source      DirectorySource
	reader      LibraryReader
	cursorCodec *cursorcodec.Codec
}

func NewPathService(
	source DirectorySource,
	reader LibraryReader,
	options PathServiceOptions,
) (*PathService, error) {
	if source == nil {
		return nil, errors.New("library directory source is required")
	}
	if reader == nil {
		return nil, errors.New("library reader is required")
	}
	cursorCodec, err := cursorcodec.New(options.CursorKey)
	if err != nil {
		return nil, fmt.Errorf("construct library path cursor codec: %w", err)
	}
	return &PathService{source: source, reader: reader, cursorCodec: cursorCodec}, nil
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

	libraries, err := service.reader.ListLibraries(ctx)
	if err != nil {
		return PathPage{}, fmt.Errorf("list libraries for path selection: %w", err)
	}
	for _, configured := range libraries {
		if configured.ID <= 0 {
			return PathPage{}, errors.New("library reader returned an invalid ID")
		}
		normalizedName, _, normalizeNameErr := NormalizeName(configured.Name)
		if normalizeNameErr != nil || normalizedName != configured.Name {
			return PathPage{}, errors.New("library reader returned an invalid name")
		}
		normalized, normalizeErr := NormalizeRoot(configured.RootRelativePath)
		if normalizeErr != nil || normalized != configured.RootRelativePath {
			return PathPage{}, errors.New("library reader returned an invalid root")
		}
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
		blockedReason := item.candidate.BlockedReason
		var conflict Library
		if blockedReason == "" {
			blockedReason, conflict = pathConflict(item.relative, libraries)
		}
		items = append(items, PathEntry{
			Name:          item.candidate.Name,
			RelativePath:  item.relative,
			HasChildren:   item.candidate.HasChildren,
			Selectable:    blockedReason == "",
			BlockedReason: blockedReason,
			ConflictID:    conflict.ID,
			ConflictName:  conflict.Name,
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

func pathConflict(candidate string, libraries []Library) (SelectionBlockedReason, Library) {
	var (
		reason   SelectionBlockedReason
		conflict Library
	)
	for _, configured := range libraries {
		candidateReason := SelectionBlockedReason("")
		switch {
		case candidate == configured.RootRelativePath:
			candidateReason = SelectionBlockedOverlapping
		case configured.RootRelativePath == "" ||
			strings.HasPrefix(candidate, configured.RootRelativePath+"/"):
			candidateReason = SelectionBlockedDescendant
		case strings.HasPrefix(configured.RootRelativePath, candidate+"/"):
			candidateReason = SelectionBlockedAncestor
		}
		if candidateReason != "" && (conflict.ID == 0 || configured.ID < conflict.ID) {
			reason = candidateReason
			conflict = configured
		}
	}
	return reason, conflict
}

type pathCursor struct {
	Version int    `json:"v"`
	Parent  string `json:"p"`
	After   string `json:"a"`
}

func (service *PathService) encodeCursor(parent, after string) (string, error) {
	encoded, err := service.cursorCodec.Encode(pathCursor{
		Version: pathCursorVersion,
		Parent:  parentFingerprint(parent),
		After:   after,
	}, "foliopath:library-path:v1")
	if err != nil {
		return "", fmt.Errorf("encode library path cursor: %w", err)
	}
	return encoded, nil
}

func (service *PathService) decodeCursor(encoded, parent string) (string, error) {
	var cursor pathCursor
	if err := service.cursorCodec.Decode(
		encoded, "foliopath:library-path:v1", &cursor,
	); err != nil ||
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
