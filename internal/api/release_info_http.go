package api

import (
	"context"
	"net/http"
	"time"

	"github.com/HappyQuQu/foliopath/internal/releaseinfo"
)

type ReleaseInfoService interface {
	Get(context.Context, bool) (releaseinfo.Snapshot, error)
}

type releaseResponse struct {
	Version     string `json:"version"`
	Name        string `json:"name"`
	Summary     string `json:"summary"`
	PublishedAt string `json:"publishedAt"`
	URL         string `json:"url"`
}

type releaseInfoResponse struct {
	CurrentVersion  string            `json:"currentVersion"`
	LatestVersion   *string           `json:"latestVersion"`
	UpdateAvailable bool              `json:"updateAvailable"`
	CheckedAt       string            `json:"checkedAt"`
	Releases        []releaseResponse `json:"releases"`
}

func registerReleaseInfoRoute(mux *http.ServeMux, service ReleaseInfoService) {
	mux.HandleFunc("GET /api/v1/releases", func(writer http.ResponseWriter, request *http.Request) {
		values := request.URL.Query()
		refresh := false
		if len(values) > 0 {
			entries, ok := values["refresh"]
			if !ok || len(values) != 1 || len(entries) != 1 ||
				(entries[0] != "true" && entries[0] != "false") {
				writePublicError(writer, request, http.StatusUnprocessableEntity,
					"invalid_request", "The release query is invalid.")
				return
			}
			refresh = entries[0] == "true"
		}
		if request.URL.RawQuery != "" && len(values) == 0 {
			writePublicError(writer, request, http.StatusUnprocessableEntity,
				"invalid_request", "The release query is invalid.")
			return
		}
		snapshot, err := service.Get(request.Context(), refresh)
		if err != nil {
			writePublicError(writer, request, http.StatusServiceUnavailable,
				"update_check_unavailable", "Update information is temporarily unavailable.")
			return
		}
		writeJSON(writer, http.StatusOK, releaseInfoWire(snapshot))
	})
}

func releaseInfoWire(snapshot releaseinfo.Snapshot) releaseInfoResponse {
	response := releaseInfoResponse{
		CurrentVersion:  snapshot.CurrentVersion,
		UpdateAvailable: snapshot.UpdateAvailable,
		CheckedAt:       snapshot.CheckedAt.UTC().Format(time.RFC3339Nano),
		Releases:        make([]releaseResponse, 0, len(snapshot.Releases)),
	}
	if snapshot.LatestVersion != "" {
		response.LatestVersion = &snapshot.LatestVersion
	}
	for _, release := range snapshot.Releases {
		response.Releases = append(response.Releases, releaseResponse{
			Version: release.Version, Name: release.Name, Summary: release.Summary,
			PublishedAt: release.PublishedAt.UTC().Format(time.RFC3339Nano), URL: release.URL,
		})
	}
	return response
}
