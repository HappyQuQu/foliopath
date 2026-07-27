package library

import (
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
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/text/collate"
	"golang.org/x/text/language"
)

const (
	DefaultLibraryPageSize = 50
	MaxLibraryPageSize     = 200
	idempotencyRetentionMS = int64(24 * 60 * 60 * 1000)
	libraryCursorVersion   = 1
)

var (
	ErrInvalidID             = errors.New("invalid library resource ID")
	ErrInvalidIdempotencyKey = errors.New("invalid idempotency key")
	ErrIdempotencyConflict   = errors.New("idempotency conflict")
	ErrInvalidLibraryCursor  = errors.New("invalid library cursor")
	ErrPreconditionFailed    = errors.New("library precondition failed")
	ErrRemovalActive         = errors.New("library removal is active")
	ErrRemovalNotFound       = errors.New("library removal not found")
	ErrRootUnavailable       = errors.New("library root is unavailable")
	ErrRootSymlink           = errors.New("library root contains a symbolic link")
	ErrRootMountBoundary     = errors.New("library root crosses a mount boundary")
	ErrRootOutsideAllowed    = errors.New("library root is outside the allowed root")
	ErrScanCapacity          = errors.New("scan admission capacity reached")
)

var idempotencyKeyPattern = regexp.MustCompile(`^[A-Za-z0-9._~-]{8,128}$`)

type Scan struct {
	ID                    int64
	LibraryID             int64
	Generation            int64
	Trigger               string
	Status                string
	Phase                 string
	Revision              int64
	DiscoveredDirectories int64
	DiscoveredAssets      int64
	ProcessedAssets       int64
	SkippedDirectories    int64
	SkippedFiles          int64
	ErrorCount            int64
	IssuesTruncated       bool
	ErrorCode             string
	CreatedAtMS           int64
	StartedAtMS           *int64
	FinishedAtMS          *int64
}

type Details struct {
	Library
	LastSuccessfulScanAtMS *int64
	LatestScanID           *int64
	AssetCount             int64
	DirectoryCount         int64
}

type CreateCommand struct {
	Name             string
	NameKey          string
	NameSortKey      []byte
	RootRelativePath string
	KeyHash          [32]byte
	RequestHash      [32]byte
	RetentionMS      int64
}

type CreateResult struct {
	Library  Details
	Scan     Scan
	Replayed bool
}

type ListPosition struct {
	NameSortKey []byte
	Name        string
	ID          int64
}

type ListParams struct {
	After *ListPosition
	Limit int
}

type Page struct {
	Items      []Details
	NextCursor string
}

type RenameCommand struct {
	ID               int64
	ExpectedRevision int64
	Name             string
	NameKey          string
	NameSortKey      []byte
}

type RemovalStatus string

const (
	RemovalQueued    RemovalStatus = "queued"
	RemovalRunning   RemovalStatus = "running"
	RemovalSucceeded RemovalStatus = "succeeded"
	RemovalFailed    RemovalStatus = "failed"
)

type Removal struct {
	ID           int64
	LibraryID    int64
	LibraryName  string
	Status       RemovalStatus
	Revision     int64
	ErrorCode    string
	CreatedAtMS  int64
	StartedAtMS  *int64
	FinishedAtMS *int64
}

type RemoveCommand struct {
	LibraryID        int64
	ExpectedRevision int64
	KeyHash          [32]byte
	RequestHash      [32]byte
	RetentionMS      int64
}

type RemoveResult struct {
	Removal  Removal
	Replayed bool
}

type LifecycleRepository interface {
	FindCreateReplay(context.Context, [32]byte, [32]byte) (CreateResult, bool, error)
	CreateLibraryWithScan(context.Context, CreateCommand) (CreateResult, error)
	ListLibraryPage(context.Context, ListParams) ([]Details, error)
	GetLibraryDetails(context.Context, int64) (Details, error)
	RenameLibraryIfRevision(context.Context, RenameCommand) (Details, error)
	RequestLibraryRemoval(context.Context, RemoveCommand) (RemoveResult, error)
	GetLibraryRemoval(context.Context, int64) (Removal, error)
}

type RootValidator interface {
	ValidateLibraryRoot(context.Context, string) error
}

type WakeNotifier interface {
	Wake()
}

type LifecycleOptions struct {
	CursorKey []byte
}

type LifecycleService struct {
	repository  LifecycleRepository
	roots       RootValidator
	scanWaker   WakeNotifier
	removeWaker WakeNotifier
	aead        cipher.AEAD
}

func NewLifecycleService(
	repository LifecycleRepository,
	roots RootValidator,
	scanWaker WakeNotifier,
	removeWaker WakeNotifier,
	options LifecycleOptions,
) (*LifecycleService, error) {
	if repository == nil || roots == nil || scanWaker == nil || removeWaker == nil {
		return nil, errors.New("library lifecycle dependencies are required")
	}
	key := append([]byte(nil), options.CursorKey...)
	if len(key) == 0 {
		key = make([]byte, 32)
		if _, err := io.ReadFull(rand.Reader, key); err != nil {
			return nil, fmt.Errorf("generate library cursor key: %w", err)
		}
	}
	if len(key) != 32 {
		return nil, errors.New("library cursor key must be 32 bytes")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("construct library cursor cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("construct library cursor AEAD: %w", err)
	}
	return &LifecycleService{
		repository:  repository,
		roots:       roots,
		scanWaker:   scanWaker,
		removeWaker: removeWaker,
		aead:        aead,
	}, nil
}

func (service *LifecycleService) Create(
	ctx context.Context,
	name, root, idempotencyKey string,
) (CreateResult, error) {
	displayName, nameKey, err := NormalizeName(name)
	if err != nil {
		return CreateResult{}, err
	}
	normalizedRoot, err := NormalizeRoot(root)
	if err != nil {
		return CreateResult{}, err
	}
	keyHash, err := hashIdempotencyKey(idempotencyKey)
	if err != nil {
		return CreateResult{}, err
	}
	requestHash := hashRequest("create_library", displayName, normalizedRoot)
	if result, found, err := service.repository.FindCreateReplay(ctx, keyHash, requestHash); err != nil {
		return CreateResult{}, err
	} else if found {
		return result, nil
	}
	if err := service.roots.ValidateLibraryRoot(ctx, normalizedRoot); err != nil {
		return CreateResult{}, err
	}
	result, err := service.repository.CreateLibraryWithScan(ctx, CreateCommand{
		Name:             displayName,
		NameKey:          nameKey,
		NameSortKey:      NaturalNameSortKey(displayName),
		RootRelativePath: normalizedRoot,
		KeyHash:          keyHash,
		RequestHash:      requestHash,
		RetentionMS:      idempotencyRetentionMS,
	})
	if err != nil {
		return CreateResult{}, err
	}
	if !result.Replayed {
		service.scanWaker.Wake()
	}
	return result, nil
}

func (service *LifecycleService) List(ctx context.Context, cursor string, limit int) (Page, error) {
	if limit == 0 {
		limit = DefaultLibraryPageSize
	}
	if limit < 1 || limit > MaxLibraryPageSize {
		return Page{}, ErrInvalidLibraryCursor
	}
	var after *ListPosition
	if cursor != "" {
		position, err := service.decodeCursor(cursor)
		if err != nil {
			return Page{}, err
		}
		after = &position
	}
	items, err := service.repository.ListLibraryPage(ctx, ListParams{
		After: after,
		Limit: limit + 1,
	})
	if err != nil {
		return Page{}, err
	}
	hasNext := len(items) > limit
	if hasNext {
		items = items[:limit]
	}
	page := Page{Items: items}
	if hasNext {
		_, _, err := NormalizeName(items[len(items)-1].Name)
		if err != nil {
			return Page{}, errors.New("repository returned invalid library name")
		}
		page.NextCursor, err = service.encodeCursor(ListPosition{
			NameSortKey: NaturalNameSortKey(items[len(items)-1].Name),
			Name:        items[len(items)-1].Name,
			ID:          items[len(items)-1].ID,
		})
		if err != nil {
			return Page{}, err
		}
	}
	return page, nil
}

func (service *LifecycleService) Get(ctx context.Context, id int64) (Details, error) {
	if id <= 0 {
		return Details{}, ErrNotFound
	}
	return service.repository.GetLibraryDetails(ctx, id)
}

func (service *LifecycleService) Rename(
	ctx context.Context,
	id, expectedRevision int64,
	name string,
) (Details, error) {
	if id <= 0 {
		return Details{}, ErrNotFound
	}
	if expectedRevision <= 0 {
		return Details{}, ErrPreconditionFailed
	}
	displayName, nameKey, err := NormalizeName(name)
	if err != nil {
		return Details{}, err
	}
	return service.repository.RenameLibraryIfRevision(ctx, RenameCommand{
		ID:               id,
		ExpectedRevision: expectedRevision,
		Name:             displayName,
		NameKey:          nameKey,
		NameSortKey:      NaturalNameSortKey(displayName),
	})
}

func (service *LifecycleService) Remove(
	ctx context.Context,
	id, expectedRevision int64,
	idempotencyKey string,
) (RemoveResult, error) {
	if id <= 0 {
		return RemoveResult{}, ErrNotFound
	}
	if expectedRevision <= 0 {
		return RemoveResult{}, ErrPreconditionFailed
	}
	keyHash, err := hashIdempotencyKey(idempotencyKey)
	if err != nil {
		return RemoveResult{}, err
	}
	result, err := service.repository.RequestLibraryRemoval(ctx, RemoveCommand{
		LibraryID:        id,
		ExpectedRevision: expectedRevision,
		KeyHash:          keyHash,
		RequestHash:      hashRequest("remove_library", strconv.FormatInt(id, 10)),
		RetentionMS:      idempotencyRetentionMS,
	})
	if err != nil {
		return RemoveResult{}, err
	}
	if !result.Replayed {
		service.removeWaker.Wake()
	}
	return result, nil
}

func (service *LifecycleService) GetRemoval(ctx context.Context, id int64) (Removal, error) {
	if id <= 0 {
		return Removal{}, ErrRemovalNotFound
	}
	return service.repository.GetLibraryRemoval(ctx, id)
}

func hashIdempotencyKey(key string) ([32]byte, error) {
	if !idempotencyKeyPattern.MatchString(key) {
		return [32]byte{}, ErrInvalidIdempotencyKey
	}
	return sha256.Sum256([]byte(key)), nil
}

func hashRequest(parts ...string) [32]byte {
	return sha256.Sum256([]byte(strings.Join(parts, "\x00")))
}

type libraryCursor struct {
	Version     int    `json:"v"`
	NameSortKey []byte `json:"k"`
	Name        string `json:"n"`
	ID          int64  `json:"i"`
}

func (service *LifecycleService) encodeCursor(position ListPosition) (string, error) {
	plaintext, err := json.Marshal(libraryCursor{
		Version:     libraryCursorVersion,
		NameSortKey: position.NameSortKey,
		Name:        position.Name,
		ID:          position.ID,
	})
	if err != nil {
		return "", fmt.Errorf("encode library cursor: %w", err)
	}
	nonce := make([]byte, service.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate library cursor nonce: %w", err)
	}
	sealed := service.aead.Seal(nonce, nonce, plaintext, []byte("foliopath:libraries:v1"))
	return base64.RawURLEncoding.EncodeToString(sealed), nil
}

func (service *LifecycleService) decodeCursor(encoded string) (ListPosition, error) {
	if len(encoded) < 8 || len(encoded) > MaxCursorBytes {
		return ListPosition{}, ErrInvalidLibraryCursor
	}
	sealed, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil || len(sealed) <= service.aead.NonceSize() {
		return ListPosition{}, ErrInvalidLibraryCursor
	}
	plaintext, err := service.aead.Open(
		nil,
		sealed[:service.aead.NonceSize()],
		sealed[service.aead.NonceSize():],
		[]byte("foliopath:libraries:v1"),
	)
	if err != nil {
		return ListPosition{}, ErrInvalidLibraryCursor
	}
	var cursor libraryCursor
	if err := json.Unmarshal(plaintext, &cursor); err != nil ||
		cursor.Version != libraryCursorVersion ||
		len(cursor.NameSortKey) == 0 ||
		cursor.Name == "" ||
		cursor.ID <= 0 {
		return ListPosition{}, ErrInvalidLibraryCursor
	}
	return ListPosition{
		NameSortKey: cursor.NameSortKey,
		Name:        cursor.Name,
		ID:          cursor.ID,
	}, nil
}

// NaturalNameSortKey is the canonical locale-neutral, numeric-aware ordering
// key for configured library display names.
func NaturalNameSortKey(name string) []byte {
	collator := collate.New(language.Und, collate.Loose, collate.Numeric)
	buffer := &collate.Buffer{}
	return append([]byte(nil), collator.KeyFromString(buffer, name)...)
}
