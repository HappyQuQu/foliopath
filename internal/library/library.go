// Package library owns media-library configuration and its invariants.
package library

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/HappyQuQu/foliopath/internal/pathpolicy"
	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"
)

const (
	maxLibraryNameRunes = 128
	maxLibraryRootRunes = 4096
)

var (
	ErrInvalidName        = errors.New("invalid library name")
	ErrInvalidRoot        = errors.New("invalid library root")
	ErrNameExists         = errors.New("library name already exists")
	ErrRootOverlap        = errors.New("library root overlaps another library")
	ErrRootImmutable      = errors.New("library root is immutable")
	ErrNotFound           = errors.New("library not found")
	ErrRepositoryNotReady = errors.New("library repository is not ready")
)

type Status string

const (
	StatusPending  Status = "pending"
	StatusScanning Status = "scanning"
	StatusReady    Status = "ready"
	StatusOffline  Status = "offline"
	StatusError    Status = "error"
)

type Library struct {
	ID                int64
	Name              string
	RootRelativePath  string
	Status            Status
	CurrentGeneration int64
	Revision          int64
	CreatedAtMS       int64
	UpdatedAtMS       int64
}

type CreateParams struct {
	Name             string
	RootRelativePath string
}

// Repository is implemented by persistence adapters. Implementations must
// enforce name uniqueness and root-overlap checks atomically.
type Repository interface {
	CreateLibrary(context.Context, CreateParams) (Library, error)
	GetLibrary(context.Context, int64) (Library, error)
	ListLibraries(context.Context) ([]Library, error)
	RenameLibrary(context.Context, int64, string) (Library, error)
}

type Service struct {
	repository Repository
}

func NewService(repository Repository) (*Service, error) {
	if repository == nil {
		return nil, errors.New("library repository is required")
	}
	return &Service{repository: repository}, nil
}

func (s *Service) Create(ctx context.Context, name, root string) (Library, error) {
	displayName, _, err := NormalizeName(name)
	if err != nil {
		return Library{}, err
	}
	normalizedRoot, err := NormalizeRoot(root)
	if err != nil {
		return Library{}, err
	}
	return s.repository.CreateLibrary(ctx, CreateParams{
		Name:             displayName,
		RootRelativePath: normalizedRoot,
	})
}

func (s *Service) Rename(ctx context.Context, id int64, name string) (Library, error) {
	if id <= 0 {
		return Library{}, ErrNotFound
	}
	displayName, _, err := NormalizeName(name)
	if err != nil {
		return Library{}, err
	}
	return s.repository.RenameLibrary(ctx, id, displayName)
}

func (s *Service) Get(ctx context.Context, id int64) (Library, error) {
	if id <= 0 {
		return Library{}, ErrNotFound
	}
	return s.repository.GetLibrary(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]Library, error) {
	return s.repository.ListLibraries(ctx)
}

// NormalizeName returns the trimmed display name and its instance-unique key.
func NormalizeName(name string) (string, string, error) {
	if !utf8.ValidString(name) {
		return "", "", ErrInvalidName
	}
	display := norm.NFC.String(strings.TrimSpace(name))
	if display == "" || utf8.RuneCountInString(display) > maxLibraryNameRunes {
		return "", "", ErrInvalidName
	}
	for _, character := range display {
		if unicode.IsControl(character) {
			return "", "", ErrInvalidName
		}
	}
	key := cases.Fold().String(norm.NFKC.String(display))
	if key == "" {
		return "", "", ErrInvalidName
	}
	return display, key, nil
}

// NormalizeRoot converts an API/library relative path into the canonical
// slash-separated form. Empty means the allowed /library root itself.
func NormalizeRoot(root string) (string, error) {
	if !utf8.ValidString(root) || utf8.RuneCountInString(root) > maxLibraryRootRunes {
		return "", ErrInvalidRoot
	}
	normalized, err := pathpolicy.Normalize(root)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrInvalidRoot, err)
	}
	return normalized, nil
}

// RootsOverlap compares canonical roots by path component rather than string
// prefix. The empty root overlaps every other root.
func RootsOverlap(left, right string) bool {
	if left == "" || right == "" || left == right {
		return true
	}
	return strings.HasPrefix(left, right+"/") || strings.HasPrefix(right, left+"/")
}

func ValidateStatus(status string) (Status, error) {
	switch Status(status) {
	case StatusPending, StatusScanning, StatusReady, StatusOffline, StatusError:
		return Status(status), nil
	default:
		return "", fmt.Errorf("unknown library status %q", status)
	}
}
