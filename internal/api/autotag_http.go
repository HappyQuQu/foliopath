package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
	"github.com/HappyQuQu/foliopath/internal/autotag"
)

type AutomaticTagService interface {
	Get(context.Context, int64) (autotag.Settings, error)
	Update(context.Context, int64, bool, int64) (autotag.Settings, error)
	Request(context.Context, int64, autotag.Mode, string) (aimodel.Operation, bool, error)
}
type automaticTagModelResponse struct {
	Identity    string `json:"identity"`
	DisplayName string `json:"displayName"`
	Protocol    string `json:"protocol"`
}
type automaticTagCoverageResponse struct {
	Eligible  int64 `json:"eligible"`
	Completed int64 `json:"completed"`
	Failed    int64 `json:"failed"`
	Stale     int64 `json:"stale"`
	Revision  int64 `json:"revision"`
}
type automaticTagResponse struct {
	LibraryID         string                       `json:"libraryId"`
	Enabled           bool                         `json:"enabled"`
	Revision          int64                        `json:"revision"`
	State             string                       `json:"state"`
	Offline           bool                         `json:"offline"`
	Model             *automaticTagModelResponse   `json:"model"`
	Coverage          automaticTagCoverageResponse `json:"coverage"`
	ActiveOperationID *string                      `json:"activeOperationId"`
}

func automaticTagWire(v autotag.Settings) automaticTagResponse {
	result := automaticTagResponse{LibraryID: fmt.Sprintf("lib_%d", v.LibraryID), Enabled: v.Enabled, Revision: v.Revision, State: v.State, Offline: v.Offline, Coverage: automaticTagCoverageResponse{v.Coverage.Eligible, v.Coverage.Completed, v.Coverage.Failed, v.Coverage.Stale, v.Coverage.Revision}}
	if v.Model != nil {
		result.Model = &automaticTagModelResponse{v.Model.Identity, v.Model.DisplayName, v.Model.Protocol}
	}
	if v.ActiveOperationID != "" {
		result.ActiveOperationID = &v.ActiveOperationID
	}
	return result
}
func registerAutomaticTagRoutes(mux *http.ServeMux, service AutomaticTagService) {
	for _, method := range []string{http.MethodGet, http.MethodPut} {
		mux.HandleFunc(method+" /api/v1/libraries/{libraryId}/ai/automatic-tags", func(w http.ResponseWriter, r *http.Request) {
			id, err := parseResourceID(r.PathValue("libraryId"), "lib_")
			if err != nil || r.URL.RawQuery != "" {
				writeAutomaticTagError(w, r, autotag.ErrInvalid)
				return
			}
			var value autotag.Settings
			if r.Method == http.MethodPut {
				revision, parseErr := parseAIRevisionETag(r.Header.Get("If-Match"), fmt.Sprintf("autotag-library-%d", id))
				if parseErr != nil {
					writeAIModelError(w, r, parseErr)
					return
				}
				var body struct {
					Enabled *bool `json:"enabled"`
				}
				if decodeAIModelJSON(r, &body) != nil || body.Enabled == nil {
					writeAutomaticTagError(w, r, autotag.ErrInvalid)
					return
				}
				value, err = service.Update(r.Context(), id, *body.Enabled, revision)
			} else {
				value, err = service.Get(r.Context(), id)
			}
			if err != nil {
				writeAutomaticTagError(w, r, err)
				return
			}
			w.Header().Set("ETag", fmt.Sprintf("\"autotag-library-%d-r%d\"", id, value.Revision))
			writeJSON(w, http.StatusOK, automaticTagWire(value))
		})
	}
	mux.HandleFunc("POST /api/v1/libraries/{libraryId}/ai/automatic-tags/jobs", func(w http.ResponseWriter, r *http.Request) {
		id, err := parseResourceID(r.PathValue("libraryId"), "lib_")
		key := r.Header.Get("Idempotency-Key")
		var body struct {
			Mode         autotag.Mode `json:"mode"`
			Confirmation string       `json:"confirmation"`
		}
		if err != nil || r.URL.RawQuery != "" || !idempotencyKeyPattern.MatchString(key) || decodeAIModelJSON(r, &body) != nil || (body.Mode != autotag.Missing && body.Mode != autotag.All) || (body.Mode == autotag.All && body.Confirmation != "reprocess_automatic_tags") || (body.Mode == autotag.Missing && body.Confirmation != "") {
			writeAutomaticTagError(w, r, autotag.ErrInvalid)
			return
		}
		value, replayed, err := service.Request(r.Context(), id, body.Mode, key)
		if err != nil {
			writeAutomaticTagError(w, r, err)
			return
		}
		w.Header().Set("Location", "/api/v1/ai/operations/"+value.ID)
		w.Header().Set("ETag", aiOperationETag(value))
		w.Header().Set("Idempotency-Replayed", strconv.FormatBool(replayed))
		writeJSON(w, http.StatusAccepted, aiOperationWire(value))
	})
}
func writeAutomaticTagError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, autotag.ErrInvalid):
		writePublicError(w, r, http.StatusBadRequest, codeInvalidRequest, "The request is invalid.")
	case errors.Is(err, autotag.ErrNotFound):
		writePublicError(w, r, http.StatusNotFound, "library_not_found", "The media library was not found.")
	case errors.Is(err, autotag.ErrDisabled):
		writePublicError(w, r, http.StatusConflict, "ai_disabled", "Automatic tags are disabled for this library.")
	case errors.Is(err, autotag.ErrOffline):
		writePublicError(w, r, http.StatusConflict, "source_offline", "The media library is offline.")
	case errors.Is(err, autotag.ErrUnavailable):
		writePublicError(w, r, http.StatusConflict, "model_unavailable", "The automatic tagging model is unavailable.")
	case errors.Is(err, autotag.ErrConflict):
		writePublicError(w, r, http.StatusConflict, "operation_conflict", "An incompatible operation is already active or the queue is full.")
	default:
		writeAIModelError(w, r, err)
	}
}
