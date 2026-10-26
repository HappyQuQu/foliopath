package api

import (
	"context"
	"errors"
	"net/http"

	"github.com/HappyQuQu/foliopath/internal/semantic"
)

type VideoSemanticStatusService interface {
	Get(context.Context, int64) (semantic.VideoStatus, error)
}

type videoSemanticStatusResponse struct {
	LibraryID          string                   `json:"libraryId"`
	Enabled            bool                     `json:"enabled"`
	Offline            bool                     `json:"offline"`
	ActiveGenerationID *string                  `json:"activeGenerationId"`
	Coverage           *derivedCoverageResponse `json:"coverage"`
}

func registerVideoSemanticStatusRoute(mux *http.ServeMux, service VideoSemanticStatusService) {
	mux.HandleFunc("GET /api/v1/libraries/{libraryId}/ai/video-semantic", func(w http.ResponseWriter, r *http.Request) {
		id, err := parseResourceID(r.PathValue("libraryId"), "lib_")
		if err != nil {
			writePublicError(w, r, http.StatusBadRequest, codeInvalidRequest, "The request is invalid.")
			return
		}
		status, err := service.Get(r.Context(), id)
		if errors.Is(err, semantic.ErrSemanticLibraryNotFound) {
			writePublicError(w, r, http.StatusNotFound, "library_not_found", "Library not found.")
			return
		}
		if err != nil {
			writePublicError(w, r, http.StatusInternalServerError, codeInternalError, "The request could not be completed.")
			return
		}
		result := videoSemanticStatusResponse{LibraryID: libraryID(status.LibraryID), Enabled: status.Enabled, Offline: status.Offline}
		if status.GenerationID != "" {
			result.ActiveGenerationID = &status.GenerationID
		}
		if c := status.Coverage; c != nil {
			result.Coverage = &derivedCoverageResponse{Eligible: c.Eligible, Completed: c.Ready, Degraded: c.Degraded,
				Failed: c.Failed, Stale: c.Stale, Revision: c.Revision, Complete: c.Complete()}
		}
		writeJSON(w, http.StatusOK, result)
	})
}
