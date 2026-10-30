package deletion

import (
	"context"
	"errors"
	"time"

	"github.com/HappyQuQu/foliopath/internal/jobs"
)

const (
	MaxTargets          = 200
	MaxActiveOperations = 32
	MaxPageSize         = 50
	BatchSize           = 256
	PreflightLifetime   = 5 * time.Minute
)

var (
	ErrDisabled     = errors.New("original deletion disabled")
	ErrPrecondition = errors.New("deletion state changed")
	ErrConfirmation = errors.New("deletion confirmation required")
	ErrQueueFull    = errors.New("deletion queue full")
	ErrNotFound     = errors.New("deletion record not found")
	ErrExpired      = errors.New("deletion confirmation expired")
	ErrBusy         = errors.New("media has an active operation")
)

type Settings struct {
	Enabled           bool
	RecycleBinEnabled bool
	RetentionDays     *int64
	Revision          int64
	Instance          string
	Available         bool
	UnavailableReason string
}

type SettingsUpdate struct {
	Enabled           bool
	RecycleBinEnabled bool
	RetentionDays     *int64
	Acknowledge       bool
}

type Target struct {
	Type string
	ID   int64
}

type Selection struct {
	Target          Target
	LibraryID       int64
	LibraryName     string
	Name            string
	RelativePath    string
	Kind            string
	BlockedReason   string
	Root            string
	Size            int64
	ModifiedNS      int64
	Identity        SourceIdentity
	LibraryIdentity SourceIdentity
}

type Preflight struct {
	RetentionDays    *int64
	ID               string
	Mode             string
	CanSubmit        bool
	ExpiresAtMS      int64
	SettingsRevision int64
	Items            []Selection
}

type OperationItem struct {
	ID              int64
	LibraryID       int64
	LibraryName     string
	Name            string
	RelativePath    string
	Kind            string
	State           string
	RemovedEntries  int64
	ErrorCode       string
	TrashEntryID    int64
	Target          Target
	Root            string
	Identity        SourceIdentity
	LibraryIdentity SourceIdentity
	Stage           string
	TrashToken      string
}

type Operation struct {
	ID              int64
	Kind            string
	State           string
	CreatedAtMS     int64
	UpdatedAtMS     int64
	Revision        int64
	CanStop         bool
	CanResume       bool
	Items           []OperationItem
	ItemsNextCursor string
	TotalItems      int64
	CompletedItems  int64
	FailedItems     int64
}

type OperationPage struct {
	Items      []Operation
	NextCursor string
}

type TrashEntry struct {
	ID           int64
	LibraryID    int64
	LibraryName  string
	Name         string
	RelativePath string
	Kind         string
	DeletedAtMS  int64
	ExpiresAtMS  *int64
	State        string
	Revision     int64
	CanRestore   bool
}

type TrashPage struct {
	Items      []TrashEntry
	NextCursor string
}

type TrashSelection struct {
	ID       int64
	Revision int64
}

type EnqueueCommand struct {
	Preflight   Preflight
	KeyHash     [32]byte
	RequestHash [32]byte
}

type TrashCommand struct {
	Kind        string
	Entries     []TrashSelection
	Empty       bool
	LibraryID   int64
	KeyHash     [32]byte
	RequestHash [32]byte
}

type ChangeCommand struct {
	ID          int64
	Revision    int64
	Action      string
	KeyHash     [32]byte
	RequestHash [32]byte
}

type Repository interface {
	GetDeletionSettings(context.Context) (Settings, error)
	UpdateDeletionSettings(context.Context, int64, SettingsUpdate) (Settings, error)
	PreviewDeletionRetention(context.Context, int64) (int64, error)
	ResolveDeletionTargets(context.Context, []Target) ([]Selection, error)
	SaveDeletionPreflight(context.Context, Preflight) (Preflight, error)
	GetDeletionPreflight(context.Context, string) (Preflight, error)
	FindDeletionReplay(context.Context, [32]byte, [32]byte) (Operation, bool, error)
	EnqueueDeletion(context.Context, EnqueueCommand) (Operation, error)
	GetDeletionOperation(context.Context, int64) (Operation, error)
	ListDeletionOperationItems(context.Context, int64, int64, int) ([]OperationItem, error)
	ListDeletionOperations(context.Context, int64, int64, int) ([]Operation, error)
	ChangeDeletionOperation(context.Context, ChangeCommand) (Operation, error)
	ListTrashEntries(context.Context, int64, int64, int) ([]TrashEntry, error)
	EnqueueTrashOperation(context.Context, TrashCommand) (Operation, error)
}

type MutationSource interface {
	OpenMutation(context.Context) (Mutator, error)
	InspectMutationSource(context.Context, string) (SourceIdentity, error)
	MutationAvailability(context.Context) (bool, string)
}

type Claim struct {
	ID       int64
	Fence    int64
	Kind     string
	Instance string
}

type RepairRequest struct {
	OperationID int64
	LibraryID   int64
	Root        string
	Identity    SourceIdentity
	ScanID      int64
}

type Queue interface {
	RecoverExpired(context.Context) (jobs.RecoverySummary, error)
	Claim(context.Context, time.Duration) (Claim, bool, error)
	RefreshLease(context.Context, Claim, time.Duration) (bool, error)
	NextDeletionItem(context.Context, Claim) (OperationItem, bool, error)
	CheckDeletionPermission(context.Context, Claim) error
	CheckDeletionRestore(context.Context, Claim, OperationItem) error
	BeginDeletionItem(context.Context, Claim, OperationItem) error
	RecordDeletionFilesystem(context.Context, Claim, OperationItem, int64, bool, bool, string) error
	FinalizeDeletionItem(context.Context, Claim, OperationItem) (bool, error)
	FinishDeletionClaim(context.Context, Claim, string) error
	ListDeletionRepairs(context.Context, int) ([]RepairRequest, error)
	AssignDeletionRepair(context.Context, RepairRequest, int64) error
	CompleteDeletionRepair(context.Context, RepairRequest, bool) error
	HandoffDeletion(context.Context, Claim) error
}
