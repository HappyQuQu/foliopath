package api

import (
	"context"
	"errors"
	"net/http"

	"github.com/HappyQuQu/foliopath/internal/thumbnail"
)

type MediaProgressService interface {
	Get(context.Context, int64) (thumbnail.ProcessingProgress, error)
}

type mediaJobProgressResponse struct {
	Total     int64 `json:"total"`
	Processed int64 `json:"processed"`
	Queued    int64 `json:"queued"`
	Running   int64 `json:"running"`
	Succeeded int64 `json:"succeeded"`
	Failed    int64 `json:"failed"`
}

type mediaProcessingProgressResponse struct {
	Active                          bool                     `json:"active"`
	Thumbnails                      mediaJobProgressResponse `json:"thumbnails"`
	VideoPreviews                   mediaJobProgressResponse `json:"videoPreviews"`
	VideoPreviewsPendingEligibility int64                    `json:"videoPreviewsPendingEligibility"`
}

func registerMediaProgressRoute(mux *http.ServeMux, service MediaProgressService) {
	mux.HandleFunc("GET /api/v1/libraries/{libraryId}/media-processing", func(
		writer http.ResponseWriter,
		request *http.Request,
	) {
		libraryID, err := parseResourceID(request.PathValue("libraryId"), "lib_")
		if err != nil {
			writePublicError(writer, request, http.StatusNotFound,
				"library_not_found", "The media library was not found.")
			return
		}
		progress, err := service.Get(request.Context(), libraryID)
		if errors.Is(err, thumbnail.ErrProgressLibraryNotFound) {
			writePublicError(writer, request, http.StatusNotFound,
				"library_not_found", "The media library was not found.")
			return
		}
		if err != nil {
			writeInternalError(writer, request)
			return
		}
		writeJSON(writer, http.StatusOK, mediaProgressWire(progress))
	})
}

func mediaProgressWire(
	progress thumbnail.ProcessingProgress,
) mediaProcessingProgressResponse {
	return mediaProcessingProgressResponse{
		Active:                          progress.Active(),
		Thumbnails:                      mediaJobProgressWire(progress.Grid),
		VideoPreviews:                   mediaJobProgressWire(progress.Storyboard),
		VideoPreviewsPendingEligibility: progress.StoryboardPendingEligibility,
	}
}

func mediaJobProgressWire(progress thumbnail.JobProgress) mediaJobProgressResponse {
	return mediaJobProgressResponse{
		Total: progress.Total(), Processed: progress.Processed(),
		Queued: progress.Queued, Running: progress.Running,
		Succeeded: progress.Succeeded, Failed: progress.Failed,
	}
}
