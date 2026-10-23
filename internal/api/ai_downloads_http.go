package api

import (
	"context"
	"net/http"
	"strconv"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
)

type AIModelDownloadService interface {
	ListDownloads(context.Context) (aimodel.DownloadPage, error)
	StartDownload(context.Context, string, string) (aimodel.InstallResult, error)
}

type aiDownloadResponse struct {
	ID               string               `json:"id"`
	Name             string               `json:"name"`
	SourceLabel      string               `json:"sourceLabel"`
	Purpose          string               `json:"purpose"`
	Version          string               `json:"version"`
	LicenseID        string               `json:"licenseId"`
	PackageSizeBytes int64                `json:"packageSizeBytes"`
	Compatibility    string               `json:"compatibility"`
	InstalledModelID *string              `json:"installedModelId"`
	Operation        *aiOperationResponse `json:"operation"`
}

type aiModelPresetResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	SourceLabel string `json:"sourceLabel"`
	Purpose     string `json:"purpose"`
	Readiness   string `json:"readiness"`
}

func registerAIModelDownloadRoutes(mux *http.ServeMux, service AIModelDownloadService) {
	mux.HandleFunc("GET /api/v1/ai/model-downloads", func(w http.ResponseWriter, r *http.Request) {
		page, err := service.ListDownloads(r.Context())
		if err != nil {
			writeAIModelError(w, r, err)
			return
		}
		items := make([]aiDownloadResponse, 0, len(page.Items))
		for _, offer := range page.Items {
			item := aiDownloadResponse{ID: offer.ID, Name: offer.Name, SourceLabel: offer.SourceLabel,
				Purpose: offer.Purpose, Version: offer.Version, LicenseID: offer.LicenseID,
				PackageSizeBytes: offer.PackageSizeBytes, Compatibility: offer.Compatibility}
			if offer.InstalledModelID != "" {
				id := offer.InstalledModelID
				item.InstalledModelID = &id
			}
			if offer.Operation != nil {
				operation := aiOperationWire(*offer.Operation)
				item.Operation = &operation
			}
			items = append(items, item)
		}
		presets := make([]aiModelPresetResponse, 0, len(page.Presets))
		for _, preset := range page.Presets {
			presets = append(presets, aiModelPresetResponse{preset.ID, preset.Name, preset.SourceLabel, preset.Purpose, preset.Readiness})
		}
		writeJSON(w, http.StatusOK, struct {
			Items            []aiDownloadResponse    `json:"items"`
			Presets          []aiModelPresetResponse `json:"presets"`
			RuntimeAvailable bool                    `json:"runtimeAvailable"`
		}{items, presets, page.RuntimeAvailable})
	})
	mux.HandleFunc("POST /api/v1/ai/model-downloads/{downloadId}/install", func(w http.ResponseWriter, r *http.Request) {
		id, key := r.PathValue("downloadId"), r.Header.Get("Idempotency-Key")
		if !aiOpaqueIDPattern.MatchString(id) || !idempotencyKeyPattern.MatchString(key) || r.ContentLength != 0 {
			writePublicError(w, r, http.StatusBadRequest, codeInvalidRequest, "The request is invalid.")
			return
		}
		result, err := service.StartDownload(r.Context(), id, key)
		if err != nil {
			writeAIModelError(w, r, err)
			return
		}
		w.Header().Set("Location", "/api/v1/ai/operations/"+result.Operation.ID)
		w.Header().Set("ETag", aiOperationETag(result.Operation))
		w.Header().Set("Idempotency-Replayed", strconv.FormatBool(result.Replayed))
		writeJSON(w, http.StatusAccepted, aiOperationWire(result.Operation))
	})
}
