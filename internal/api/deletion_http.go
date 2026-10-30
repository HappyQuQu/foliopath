package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
	"github.com/HappyQuQu/foliopath/internal/deletion"
)

type DeletionService interface {
	Settings(context.Context) (deletion.Settings, error)
	UpdateSettings(context.Context, int64, deletion.SettingsUpdate) (deletion.Settings, error)
	Preflight(context.Context, []deletion.Target) (deletion.Preflight, error)
	Submit(context.Context, string, string, string) (deletion.Operation, bool, error)
	GetOperation(context.Context, int64, string, int) (deletion.Operation, error)
	ChangeOperation(context.Context, int64, int64, string, string, string) (deletion.Operation, error)
	ListOperations(context.Context, int64, string, int) (deletion.OperationPage, error)
}

type deletionTargetRequest struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

func registerDeletionRoutes(mux *http.ServeMux, service DeletionService) {
	mux.HandleFunc("GET /api/v1/deletion/settings", func(w http.ResponseWriter, r *http.Request) {
		value, err := service.Settings(r.Context())
		if err != nil {
			writeDeletionError(w, r, err)
			return
		}
		writeDeletionSettings(w, value)
	})
	mux.HandleFunc("PATCH /api/v1/deletion/settings", func(w http.ResponseWriter, r *http.Request) {
		revision, err := parseAIRevisionETag(r.Header.Get("If-Match"), "deletion-settings")
		if err != nil {
			writeDeletionError(w, r, err)
			return
		}
		var payload struct {
			Enabled     *bool           `json:"enabled"`
			Recycle     *bool           `json:"recycleBinEnabled"`
			Days        json.RawMessage `json:"retentionDays"`
			Acknowledge *bool           `json:"acknowledge"`
		}
		if err := decodeCurationJSON(w, r, &payload); err != nil || payload.Enabled == nil || payload.Recycle == nil || payload.Acknowledge == nil || len(payload.Days) == 0 {
			writeDeletionError(w, r, deletion.ErrInvalid)
			return
		}
		var days *int64
		if err := json.Unmarshal(payload.Days, &days); err != nil {
			writeDeletionError(w, r, deletion.ErrInvalid)
			return
		}
		value, err := service.UpdateSettings(r.Context(), revision, deletion.SettingsUpdate{Enabled: *payload.Enabled, RecycleBinEnabled: *payload.Recycle, RetentionDays: days, Acknowledge: *payload.Acknowledge})
		if err != nil {
			writeDeletionError(w, r, err)
			return
		}
		writeDeletionSettings(w, value)
	})
	mux.HandleFunc("POST /api/v1/deletion/preflight", func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Targets []deletionTargetRequest `json:"targets"`
		}
		if err := decodeCurationJSON(w, r, &payload); err != nil {
			writeDeletionError(w, r, deletion.ErrInvalid)
			return
		}
		targets := make([]deletion.Target, 0, len(payload.Targets))
		for _, target := range payload.Targets {
			prefix := "ast_"
			if target.Type == "directory" {
				prefix = "dir_"
			}
			id, err := parseResourceID(target.ID, prefix)
			if err != nil {
				writeDeletionError(w, r, deletion.ErrInvalid)
				return
			}
			targets = append(targets, deletion.Target{Type: target.Type, ID: id})
		}
		value, err := service.Preflight(r.Context(), targets)
		if err != nil {
			writeDeletionError(w, r, err)
			return
		}
		items := make([]map[string]any, 0, len(value.Items))
		for _, item := range value.Items {
			targetID := assetIDString(item.Target.ID)
			if item.Target.Type == "directory" {
				targetID = directoryIDString(item.Target.ID)
			}
			var libraryReference, libraryName, name, relative any
			if item.LibraryID > 0 {
				libraryReference = libraryID(item.LibraryID)
				libraryName = item.LibraryName
				name = item.Name
				relative = item.RelativePath
			}
			items = append(items, map[string]any{"target": map[string]any{"type": item.Target.Type, "id": targetID}, "libraryId": libraryReference, "libraryName": libraryName, "name": name, "relativePath": relative, "kind": item.Kind, "blockedReason": deletionNullable(item.BlockedReason)})
		}
		var expires any
		if value.ExpiresAtMS > 0 {
			expires = timestamp(value.ExpiresAtMS)
		}
		writeJSON(w, http.StatusOK, map[string]any{"id": deletionNullable(value.ID), "mode": value.Mode, "retentionDays": value.RetentionDays, "canSubmit": value.CanSubmit, "expiresAt": expires, "items": items})
	})
	mux.HandleFunc("POST /api/v1/deletion/operations", func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			ID           string `json:"preflightId"`
			Confirmation string `json:"confirmation"`
		}
		if err := decodeCurationJSON(w, r, &payload); err != nil {
			writeDeletionError(w, r, deletion.ErrInvalid)
			return
		}
		value, _, err := service.Submit(r.Context(), payload.ID, payload.Confirmation, r.Header.Get("Idempotency-Key"))
		if err != nil {
			writeDeletionError(w, r, err)
			return
		}
		writeDeletionOperation(w, http.StatusAccepted, value)
	})
	mux.HandleFunc("GET /api/v1/deletion/operations", func(w http.ResponseWriter, r *http.Request) {
		libraryID, cursor, limit, err := parseDeletionPageQuery(r.URL.RawQuery, true)
		if err != nil {
			writeDeletionError(w, r, err)
			return
		}
		page, err := service.ListOperations(r.Context(), libraryID, cursor, limit)
		if err != nil {
			writeDeletionError(w, r, err)
			return
		}
		items := make([]map[string]any, 0, len(page.Items))
		for _, item := range page.Items {
			items = append(items, deletionOperationWire(item))
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items, "nextCursor": deletionNullable(page.NextCursor)})
	})
	mux.HandleFunc("GET /api/v1/deletion/operations/{operationId}", func(w http.ResponseWriter, r *http.Request) {
		id, err := parseResourceID(r.PathValue("operationId"), "deletion_")
		if err != nil {
			writeDeletionError(w, r, deletion.ErrNotFound)
			return
		}
		_, cursor, limit, err := parseDeletionPageQuery(r.URL.RawQuery, false)
		if err != nil {
			writeDeletionError(w, r, err)
			return
		}
		value, err := service.GetOperation(r.Context(), id, cursor, limit)
		if err != nil {
			writeDeletionError(w, r, err)
			return
		}
		writeDeletionOperation(w, http.StatusOK, value)
	})
	for _, action := range []string{"stop", "resume"} {
		mux.HandleFunc("POST /api/v1/deletion/operations/{operationId}/"+action, func(w http.ResponseWriter, r *http.Request) {
			id, err := parseResourceID(r.PathValue("operationId"), "deletion_")
			if err != nil {
				writeDeletionError(w, r, deletion.ErrNotFound)
				return
			}
			revision, err := parseAIRevisionETag(r.Header.Get("If-Match"), r.PathValue("operationId"))
			if err != nil {
				writeDeletionError(w, r, err)
				return
			}
			confirmation := ""
			if action == "resume" {
				var payload struct {
					Confirmation string `json:"confirmation"`
				}
				if err := decodeCurationJSON(w, r, &payload); err != nil {
					writeDeletionError(w, r, deletion.ErrInvalid)
					return
				}
				confirmation = payload.Confirmation
			} else if r.ContentLength != 0 {
				writeDeletionError(w, r, deletion.ErrInvalid)
				return
			}
			value, err := service.ChangeOperation(r.Context(), id, revision, action, confirmation, r.Header.Get("Idempotency-Key"))
			if err != nil {
				writeDeletionError(w, r, err)
				return
			}
			writeDeletionOperation(w, http.StatusAccepted, value)
		})
	}
}

func deletionResourceID(prefix string, id int64) string { return prefix + strconv.FormatInt(id, 10) }
func deletionNullable(value string) any {
	if value == "" {
		return nil
	}
	return value
}
func writeDeletionSettings(w http.ResponseWriter, value deletion.Settings) {
	w.Header().Set("ETag", fmt.Sprintf(`"deletion-settings-r%d"`, value.Revision))
	writeJSON(w, http.StatusOK, map[string]any{"enabled": value.Enabled, "recycleBinEnabled": value.RecycleBinEnabled, "retentionDays": value.RetentionDays, "available": value.Available, "unavailableReason": deletionNullable(value.UnavailableReason), "revision": value.Revision})
}
func writeDeletionOperation(w http.ResponseWriter, status int, value deletion.Operation) {
	id := deletionResourceID("deletion_", value.ID)
	w.Header().Set("ETag", fmt.Sprintf(`"%s-r%d"`, id, value.Revision))
	w.Header().Set("Location", "/api/v1/deletion/operations/"+id)
	writeJSON(w, status, deletionOperationWire(value))
}
func deletionOperationWire(value deletion.Operation) map[string]any {
	items := make([]map[string]any, 0, len(value.Items))
	for _, item := range value.Items {
		var trash any
		if item.TrashEntryID > 0 {
			trash = deletionResourceID("trash_", item.TrashEntryID)
		}
		items = append(items, map[string]any{"id": deletionResourceID("deletion_item_", item.ID), "libraryId": libraryID(item.LibraryID), "libraryName": item.LibraryName, "name": item.Name, "relativePath": item.RelativePath, "kind": item.Kind, "state": item.State, "removedEntries": item.RemovedEntries, "errorCode": deletionNullable(item.ErrorCode), "trashEntryId": trash})
	}
	return map[string]any{"id": deletionResourceID("deletion_", value.ID), "kind": value.Kind, "state": value.State, "createdAt": timestamp(value.CreatedAtMS), "updatedAt": timestamp(value.UpdatedAtMS), "revision": value.Revision, "canStop": value.CanStop, "canResume": value.CanResume, "items": items, "itemsNextCursor": deletionNullable(value.ItemsNextCursor), "totalItems": value.TotalItems, "completedItems": value.CompletedItems, "failedItems": value.FailedItems}
}
func parseDeletionPageQuery(raw string, allowLibrary bool) (int64, string, int, error) {
	values, err := url.ParseQuery(raw)
	if err != nil {
		return 0, "", 0, deletion.ErrInvalid
	}
	for key, entries := range values {
		if len(entries) != 1 || key != "cursor" && key != "limit" && !(allowLibrary && key == "libraryId") {
			return 0, "", 0, deletion.ErrInvalid
		}
	}
	var libraryID int64
	if value, present := values["libraryId"]; present {
		libraryID, err = parseResourceID(value[0], "lib_")
		if err != nil {
			return 0, "", 0, deletion.ErrInvalid
		}
	}
	limit := 0
	if value, present := values["limit"]; present {
		limit, err = strconv.Atoi(value[0])
		if err != nil || limit < 1 || limit > deletion.MaxPageSize {
			return 0, "", 0, deletion.ErrInvalid
		}
	}
	if value, present := values["cursor"]; present && value[0] == "" {
		return 0, "", 0, deletion.ErrInvalid
	}
	return libraryID, values.Get("cursor"), limit, nil
}
func writeDeletionError(w http.ResponseWriter, r *http.Request, err error) {
	status, code, message := http.StatusInternalServerError, codeInternalError, "An unexpected error occurred."
	switch {
	case errors.Is(err, errPreconditionRequired):
		status, code, message = 428, "precondition_required", "Refresh this page before trying again."
	case errors.Is(err, deletion.ErrPrecondition), errors.Is(err, aimodel.ErrPreconditionFailed):
		status, code, message = 412, "precondition_failed", "The selection or settings changed. Refresh and try again."
	case errors.Is(err, deletion.ErrInvalid):
		status, code, message = 400, codeInvalidRequest, "The request is invalid."
	case errors.Is(err, deletion.ErrDisabled):
		status, code, message = 409, "deletion_disabled", "Media changes are disabled."
	case errors.Is(err, deletion.ErrConfirmation):
		status, code, message = 409, "deletion_confirmation_required", "Confirm this media change before continuing."
	case errors.Is(err, deletion.ErrExpired):
		status, code, message = 409, "deletion_confirmation_expired", "Review and confirm the selection again."
	case errors.Is(err, deletion.ErrBusy):
		status, code, message = 409, "deletion_busy", "Wait for the existing media change to finish."
	case errors.Is(err, deletion.ErrConflict):
		status, code, message = 409, "deletion_conflict", "The destination or requested operation is already in use."
	case errors.Is(err, deletion.ErrSourceChanged):
		status, code, message = 409, "deletion_source_changed", "The source changed. Refresh and select it again."
	case errors.Is(err, deletion.ErrNotFound), errors.Is(err, deletion.ErrSourceMissing):
		status, code, message = 404, "deletion_not_found", "The requested file or operation was not found."
	case errors.Is(err, deletion.ErrQueueFull):
		status, code, message = 429, "rate_limited", "The media-change queue is full. Try again later."
	case errors.Is(err, deletion.ErrPermission):
		status, code, message = 409, "deletion_permission_denied", "Storage permissions prevent this media change."
	case errors.Is(err, deletion.ErrWriteUnavailable), errors.Is(err, deletion.ErrSourceIO):
		status, code, message = 503, "deletion_unavailable", "Media changes are unavailable on this storage."
	}
	writePublicError(w, r, status, code, message)
}
