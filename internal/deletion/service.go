package deletion

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"path"
	"regexp"
	"strings"
	"time"

	cursorcodec "github.com/HappyQuQu/foliopath/internal/cursor"
)

type WakeNotifier interface{ Wake() }

type Service struct {
	repository Repository
	source     MutationSource
	wake       WakeNotifier
	cursors    *cursorcodec.Codec
	now        func() time.Time
}

func NewService(repository Repository, source MutationSource, wake WakeNotifier, cursorKey []byte, now func() time.Time) (*Service, error) {
	if repository == nil || source == nil || wake == nil {
		return nil, errors.New("deletion dependencies required")
	}
	codec, err := cursorcodec.New(cursorKey)
	if err != nil {
		return nil, err
	}
	if now == nil {
		now = time.Now
	}
	return &Service{repository: repository, source: source, wake: wake, cursors: codec, now: now}, nil
}

func (service *Service) Settings(ctx context.Context) (Settings, error) {
	settings, err := service.repository.GetDeletionSettings(ctx)
	if err != nil {
		return Settings{}, err
	}
	settings.RecycleBinEnabled, settings.RetentionDays = false, nil
	settings.Available, settings.UnavailableReason = service.source.MutationAvailability(ctx)
	return settings, nil
}

func (service *Service) UpdateSettings(ctx context.Context, revision int64, update SettingsUpdate) (Settings, error) {
	if revision < 1 || update.RecycleBinEnabled || update.RetentionDays != nil {
		return Settings{}, ErrInvalid
	}
	current, err := service.repository.GetDeletionSettings(ctx)
	if err != nil {
		return Settings{}, err
	}
	if current.Revision != revision {
		return Settings{}, ErrPrecondition
	}
	if dangerousPolicyChange(current, update) && !update.Acknowledge {
		return Settings{}, ErrConfirmation
	}
	if update.Enabled {
		available, _ := service.source.MutationAvailability(ctx)
		if !available {
			return Settings{}, ErrWriteUnavailable
		}

	}
	result, err := service.repository.UpdateDeletionSettings(ctx, revision, update)
	if err != nil {
		return Settings{}, err
	}
	result.Available, result.UnavailableReason = service.source.MutationAvailability(ctx)
	service.wake.Wake()
	return result, nil
}

func dangerousPolicyChange(current Settings, next SettingsUpdate) bool {
	return !current.Enabled && next.Enabled || current.RecycleBinEnabled && !next.RecycleBinEnabled ||
		next.RetentionDays != nil && (current.RetentionDays == nil || *next.RetentionDays < *current.RetentionDays)
}

func (service *Service) Preflight(ctx context.Context, targets []Target) (Preflight, error) {
	if err := validateTargets(targets); err != nil {
		return Preflight{}, err
	}
	settings, err := service.repository.GetDeletionSettings(ctx)
	if err != nil {
		return Preflight{}, err
	}
	if !settings.Enabled {
		return Preflight{}, ErrDisabled
	}
	items, err := service.repository.ResolveDeletionTargets(ctx, targets)
	if err != nil {
		return Preflight{}, err
	}
	items = normalizeSelection(items)
	available, _ := service.source.MutationAvailability(ctx)
	if !available {
		return Preflight{}, ErrWriteUnavailable
	}
	canSubmit := true
	libraryIdentities := make(map[int64]SourceIdentity)
	for index := range items {
		item := &items[index]
		if item.BlockedReason == "" {
			identity, ok := libraryIdentities[item.LibraryID]
			if !ok {
				identity, err = service.source.InspectMutationSource(ctx, item.Root)
				if err == nil {
					libraryIdentities[item.LibraryID] = identity
				}
				item.BlockedReason = sourceBlockReason(err)
			}
			item.LibraryIdentity = identity
		}
		if item.BlockedReason == "" {
			item.Identity, err = service.source.InspectMutationSource(ctx, physicalPath(*item))
			item.BlockedReason = sourceBlockReason(err)
		}
		if item.BlockedReason == "" && item.Kind == "file" && (item.Size != item.Identity.Size || item.ModifiedNS != item.Identity.ModifiedNS) {
			item.BlockedReason = "changed"
		}
		if item.BlockedReason == "" && (item.Kind == "directory") != item.Identity.Directory {
			item.BlockedReason = "changed"
		}
		if item.BlockedReason != "" {
			canSubmit = false
		}
	}
	mode := "permanent"
	result := Preflight{Mode: mode, CanSubmit: canSubmit, SettingsRevision: settings.Revision, ExpiresAtMS: service.now().Add(PreflightLifetime).UnixMilli(), Items: items}
	if !canSubmit {
		result.ExpiresAtMS = 0
		return result, nil
	}
	return service.repository.SaveDeletionPreflight(ctx, result)
}

func validateTargets(targets []Target) error {
	if len(targets) < 1 || len(targets) > MaxTargets {
		return ErrInvalid
	}
	for _, target := range targets {
		if target.ID < 1 || target.Type != "asset" && target.Type != "directory" {
			return ErrInvalid
		}
	}
	return nil
}

func normalizeSelection(items []Selection) []Selection {
	result := make([]Selection, 0, len(items))
	seen := make(map[Target]bool)
	for index, item := range items {
		if seen[item.Target] {
			continue
		}
		seen[item.Target] = true
		covered := false
		for otherIndex, parent := range items {
			if index == otherIndex || parent.BlockedReason != "" || parent.LibraryID != item.LibraryID || parent.Kind != "directory" && parent.Kind != "archive" {
				continue
			}
			if parent.RelativePath != "" && strings.HasPrefix(item.RelativePath, parent.RelativePath+"/") {
				covered = true
				break
			}
		}
		if !covered {
			result = append(result, item)
		}
	}
	return result
}

func physicalPath(item Selection) string { return path.Join(item.Root, item.RelativePath) }

func sourceBlockReason(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, ErrSourceChanged):
		return "changed"
	case errors.Is(err, ErrSourceMissing):
		return "missing"
	case errors.Is(err, ErrPermission):
		return "unreadable"
	default:
		return "unavailable"
	}
}

func (service *Service) Submit(ctx context.Context, preflightID, confirmation, key string) (Operation, bool, error) {
	if len(preflightID) < 1 || len(preflightID) > 128 {
		return Operation{}, false, ErrInvalid
	}
	keyHash, err := deletionKey(key)
	if err != nil {
		return Operation{}, false, err
	}
	requestHash := deletionDigest([]string{"submit", preflightID, confirmation})
	if replay, found, err := service.repository.FindDeletionReplay(ctx, keyHash, requestHash); err != nil || found {
		return replay, found, err
	}
	preflight, err := service.repository.GetDeletionPreflight(ctx, preflightID)
	if err != nil {
		return Operation{}, false, err
	}
	if preflight.Mode != "permanent" {
		return Operation{}, false, ErrExpired
	}
	if preflight.ExpiresAtMS <= service.now().UnixMilli() {
		return Operation{}, false, ErrExpired
	}
	if confirmation != "permanently_delete" {
		return Operation{}, false, ErrConfirmation
	}
	available, _ := service.source.MutationAvailability(ctx)
	if !available {
		return Operation{}, false, ErrWriteUnavailable
	}
	checkedLibraries := make(map[int64]bool)
	for _, item := range preflight.Items {
		if !checkedLibraries[item.LibraryID] {
			current, err := service.source.InspectMutationSource(ctx, item.Root)
			if err != nil {
				return Operation{}, false, err
			}
			if !item.LibraryIdentity.Matches(current) {
				return Operation{}, false, ErrSourceChanged
			}
			checkedLibraries[item.LibraryID] = true
		}
		current, err := service.source.InspectMutationSource(ctx, physicalPath(item))
		if err != nil {
			return Operation{}, false, err
		}
		if !item.Identity.Matches(current) {
			return Operation{}, false, ErrSourceChanged
		}
	}
	result, err := service.repository.EnqueueDeletion(ctx, EnqueueCommand{Preflight: preflight, KeyHash: keyHash, RequestHash: requestHash})
	if err == nil {
		service.wake.Wake()
	}
	return result, false, err
}

func (service *Service) GetOperation(ctx context.Context, id int64, encoded string, limit int) (Operation, error) {
	if id < 1 {
		return Operation{}, ErrNotFound
	}
	before, limit, err := service.pageRequest(encoded, "operation-items", id, limit)
	if err != nil {
		return Operation{}, err
	}
	result, err := service.repository.GetDeletionOperation(ctx, id)
	if err != nil {
		return Operation{}, err
	}
	items, err := service.repository.ListDeletionOperationItems(ctx, id, before, limit+1)
	if err != nil {
		return Operation{}, err
	}
	result.Items = items
	if len(items) > limit {
		result.Items = items[:limit]
		result.ItemsNextCursor, err = service.encodePage("operation-items", id, items[limit-1].ID)
	}
	return result, err
}

func (service *Service) ChangeOperation(ctx context.Context, id, revision int64, action, confirmation, key string) (Operation, error) {
	if id < 1 || revision < 1 || action != "stop" && action != "resume" {
		return Operation{}, ErrInvalid
	}
	if action == "resume" && confirmation != "continue" {
		return Operation{}, ErrConfirmation
	}
	keyHash, err := deletionKey(key)
	if err != nil {
		return Operation{}, err
	}
	requestHash := deletionDigest([]any{action, id, revision, confirmation})
	if replay, found, err := service.repository.FindDeletionReplay(ctx, keyHash, requestHash); err != nil || found {
		return replay, err
	}
	if action == "resume" {
		if available, _ := service.source.MutationAvailability(ctx); !available {
			return Operation{}, ErrWriteUnavailable
		}
	}
	result, err := service.repository.ChangeDeletionOperation(ctx, ChangeCommand{ID: id, Revision: revision, Action: action, KeyHash: keyHash, RequestHash: requestHash})
	if err == nil {
		service.wake.Wake()
	}
	return result, err
}

func (service *Service) ListOperations(ctx context.Context, libraryID int64, encoded string, limit int) (OperationPage, error) {
	before, limit, err := service.pageRequest(encoded, "operations", libraryID, limit)
	if err != nil {
		return OperationPage{}, err
	}
	items, err := service.repository.ListDeletionOperations(ctx, libraryID, before, limit+1)
	if err != nil {
		return OperationPage{}, err
	}
	result := OperationPage{Items: items}
	if len(items) > limit {
		result.Items = items[:limit]
		result.NextCursor, err = service.encodePage("operations", libraryID, items[limit-1].ID)
	}
	if err != nil {
		return OperationPage{}, err
	}
	for index := range result.Items {
		result.Items[index], err = service.GetOperation(ctx, result.Items[index].ID, "", 1)
		if err != nil {
			return OperationPage{}, err
		}
	}
	return result, err
}

func (service *Service) pageRequest(encoded, kind string, libraryID int64, limit int) (int64, int, error) {
	if limit == 0 {
		limit = 25
	}
	if libraryID < 0 || limit < 1 || limit > MaxPageSize || len(encoded) > 2048 {
		return 0, 0, ErrInvalid
	}
	if encoded == "" {
		return 0, limit, nil
	}
	var value pageCursor
	if service.cursors.Decode(encoded, "foliopath:deletion:"+kind+":v1", &value) != nil || value.Version != 1 || value.ScopeID != libraryID || value.Before < 1 {
		return 0, 0, ErrInvalid
	}
	return value.Before, limit, nil
}

func (service *Service) encodePage(kind string, libraryID, before int64) (string, error) {
	return service.cursors.Encode(pageCursor{Version: 1, Before: before, ScopeID: libraryID}, "foliopath:deletion:"+kind+":v1")
}

var deletionKeyPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{15,127}$`)

func deletionKey(value string) ([32]byte, error) {
	if !deletionKeyPattern.MatchString(value) {
		return [32]byte{}, ErrInvalid
	}
	return sha256.Sum256([]byte(value)), nil
}
func deletionDigest(value any) [32]byte { data, _ := json.Marshal(value); return sha256.Sum256(data) }

type pageCursor struct {
	Version int   `json:"v"`
	Before  int64 `json:"before"`
	ScopeID int64 `json:"scope"`
}
