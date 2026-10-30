// Package deletion owns explicit original-file deletion, trash and restoration.
package deletion

import (
	"context"
	"errors"

	"github.com/HappyQuQu/foliopath/internal/pathpolicy"
)

const TrashDirectory = pathpolicy.TrashDirectory

var (
	ErrWriteUnavailable = errors.New("media changes unavailable")
	ErrSourceChanged    = errors.New("media source changed")
	ErrSourceMissing    = errors.New("media source missing")
	ErrPermission       = errors.New("media changes denied")
	ErrInvalid          = errors.New("invalid deletion request")
	ErrConflict         = errors.New("deletion target occupied")
	ErrSourceIO         = errors.New("media storage unavailable")
)

// SourceIdentity belongs to the deletion journal, never to public media DTOs.
type SourceIdentity struct {
	Device     uint64 `json:"device"`
	Inode      uint64 `json:"inode"`
	Size       int64  `json:"size"`
	ModifiedNS int64  `json:"modifiedNS"`
	Directory  bool   `json:"directory"`
}

func (expected SourceIdentity) Matches(current SourceIdentity) bool {
	return expected.Device == current.Device && expected.Inode == current.Inode && expected.Directory == current.Directory &&
		(expected.Directory || expected.Size == current.Size && expected.ModifiedNS == current.ModifiedNS)
}

type RemovalProgress struct {
	Removed int64
	Done    bool
}

// Mutator is scoped to one admitted operation and must be reaped before that
// operation releases admission. Paths are server-resolved allowed-root-relative.
type Mutator interface {
	BindLibrary(context.Context, string, SourceIdentity) error
	Inspect(context.Context, string) (SourceIdentity, error)
	InspectTrash(context.Context, string, string) (SourceIdentity, error)
	PrepareTrash(context.Context, string) error
	Trash(context.Context, string, string, string, SourceIdentity) error
	Restore(context.Context, string, string, string, SourceIdentity) error
	RemoveBatch(context.Context, string, SourceIdentity) (RemovalProgress, error)
	PurgeBatch(context.Context, string, string, SourceIdentity) (RemovalProgress, error)
	Close() error
}
