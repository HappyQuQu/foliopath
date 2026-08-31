// Package aimodel owns reviewed model-package compatibility and lifecycle state.
package aimodel

import (
	"errors"
	"regexp"
	"time"
)

const PurposeSemanticImageText = "semantic_image_text"

var (
	ErrInvalidModel                = errors.New("invalid AI model")
	ErrModelNotFound               = errors.New("AI model not found")
	ErrPreconditionFailed          = errors.New("AI model precondition failed")
	ErrRepositoryState             = errors.New("invalid AI model repository state")
	ErrInsufficientSpace           = errors.New("insufficient managed model space")
	ErrModelSourceUnavailable      = errors.New("AI model source unavailable")
	ErrInferenceRuntimeUnavailable = errors.New("AI inference runtime unavailable")
	hexSHA256                      = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

type StorageMode string

const (
	StorageManaged StorageMode = "managed"
	StorageDirect  StorageMode = "direct"
)

type State string

const (
	StateInstalled   State = "installed"
	StateAvailable   State = "available"
	StateUnavailable State = "unavailable"
)

type VerifiedPackage struct {
	PackageID       string
	Purpose         string
	Version         string
	Architecture    string
	ContentHash     string
	LicenseID       string
	PackageSizeByte int64
}

type Model struct {
	ID                   string
	Package              VerifiedPackage
	StorageMode          StorageMode
	State                State
	SourceIdentity       string
	AvailabilityRevision int64
	CreatedAt            time.Time
	UpdatedAt            time.Time
	Active               bool
}

type Snapshot struct {
	Items         []Model
	ActiveModelID string
	Revision      int64
}

func ValidatePackage(value VerifiedPackage) error {
	if value.PackageID == "" || len(value.PackageID) > 128 || value.Purpose != PurposeSemanticImageText ||
		value.Version == "" || len(value.Version) > 64 ||
		(value.Architecture != "amd64" && value.Architecture != "arm64") ||
		!hexSHA256.MatchString(value.ContentHash) || value.LicenseID == "" || len(value.LicenseID) > 128 ||
		value.PackageSizeByte <= 0 {
		return ErrInvalidModel
	}
	return nil
}

func ValidateModel(value Model) error {
	if value.ID == "" || len(value.ID) > 128 || ValidatePackage(value.Package) != nil ||
		(value.StorageMode != StorageManaged && value.StorageMode != StorageDirect) ||
		(value.State != StateInstalled && value.State != StateAvailable && value.State != StateUnavailable) ||
		value.SourceIdentity == "" || len(value.SourceIdentity) > 256 || value.AvailabilityRevision < 1 ||
		value.CreatedAt.IsZero() || value.UpdatedAt.Before(value.CreatedAt) {
		return ErrRepositoryState
	}
	return nil
}
