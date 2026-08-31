package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
	"github.com/HappyQuQu/foliopath/internal/catalog"
	"github.com/HappyQuQu/foliopath/internal/semantic"
)

type SemanticService interface {
	GetLibrarySettings(context.Context, int64) (semantic.LibrarySettings, error)
	UpdateLibrarySettings(context.Context, int64, bool, int64) (semantic.LibrarySettings, error)
	RequestBackfill(context.Context, int64, semantic.JobMode, string) (aimodel.Operation, bool, error)
	RequestClear(context.Context, int64, int64, string) (aimodel.Operation, bool, error)
}

type SemanticSearchService interface {
	Search(context.Context, semantic.SearchRequest) (semantic.SearchResult, error)
}

type VideoSemanticSearchService interface {
	Search(context.Context, semantic.SearchRequest) (semantic.VideoSearchResult, error)
}

type VideoSemanticJobService interface {
	RequestVideoJob(context.Context, int64, semantic.JobMode, string) (aimodel.Operation, bool, error)
}

type semanticSettingsUpdateRequest struct {
	Enabled *bool `json:"enabled"`
}
type semanticJobRequest struct {
	Mode semantic.JobMode `json:"mode"`
}
type semanticClearRequest struct {
	Confirmation string `json:"confirmation"`
}

type semanticCoverageResponse struct {
	Eligible  int64 `json:"eligible"`
	Completed int64 `json:"completed"`
	Failed    int64 `json:"failed"`
	Stale     int64 `json:"stale"`
	Revision  int64 `json:"revision"`
	Complete  bool  `json:"complete"`
}
type semanticSettingsResponse struct {
	LibraryID          string                   `json:"libraryId"`
	Enabled            bool                     `json:"enabled"`
	State              semantic.LibraryState    `json:"state"`
	Revision           int64                    `json:"revision"`
	ActiveGenerationID *string                  `json:"activeGenerationId"`
	Coverage           semanticCoverageResponse `json:"coverage"`
}

type semanticExcludedLibraryResponse struct {
	LibraryID string `json:"libraryId"`
	Reason    string `json:"reason"`
}

type semanticSearchCoverageResponse struct {
	Eligible          int64                             `json:"eligible"`
	Completed         int64                             `json:"completed"`
	Failed            int64                             `json:"failed"`
	Stale             int64                             `json:"stale"`
	Complete          bool                              `json:"complete"`
	ExcludedLibraries []semanticExcludedLibraryResponse `json:"excludedLibraries"`
}

type semanticAssetPageResponse struct {
	Items      []assetResponse                `json:"items"`
	NextCursor *string                        `json:"nextCursor"`
	Coverage   semanticSearchCoverageResponse `json:"coverage"`
}

type semanticVideoMatchedFrameResponse struct {
	Ordinal     int   `json:"ordinal"`
	TimestampMS int64 `json:"timestampMs"`
	PlanSize    int   `json:"planSize"`
}

type semanticVideoHitResponse struct {
	Asset        assetResponse                     `json:"asset"`
	Score        float32                           `json:"score"`
	MatchedFrame semanticVideoMatchedFrameResponse `json:"matchedFrame"`
}

type derivedCoverageResponse struct {
	Eligible          int64                             `json:"eligible"`
	Completed         int64                             `json:"completed"`
	Degraded          int64                             `json:"degraded"`
	Failed            int64                             `json:"failed"`
	Stale             int64                             `json:"stale"`
	Complete          bool                              `json:"complete"`
	Revision          int64                             `json:"revision"`
	ExcludedLibraries []semanticExcludedLibraryResponse `json:"excludedLibraries,omitempty"`
}

type semanticVideoPageResponse struct {
	Items      []semanticVideoHitResponse `json:"items"`
	NextCursor *string                    `json:"nextCursor"`
	Coverage   derivedCoverageResponse    `json:"coverage"`
}

const maxSemanticSearchRawQueryBytes = 16 << 10

type semanticSearchAdmission struct{ slot chan struct{} }

func newSemanticSearchAdmission() *semanticSearchAdmission {
	return &semanticSearchAdmission{slot: make(chan struct{}, 1)}
}

func (admission *semanticSearchAdmission) acquire() bool {
	select {
	case admission.slot <- struct{}{}:
		return true
	default:
		return false
	}
}

func (admission *semanticSearchAdmission) release() { <-admission.slot }

func registerSemanticRoutes(mux *http.ServeMux, service SemanticService) {
	mux.HandleFunc("GET /api/v1/libraries/{libraryId}/ai/semantic", func(w http.ResponseWriter, r *http.Request) {
		libraryID, err := parseResourceID(r.PathValue("libraryId"), "lib_")
		if err != nil {
			writeSemanticError(w, r, semantic.ErrSemanticLibraryNotFound)
			return
		}
		settings, err := service.GetLibrarySettings(r.Context(), libraryID)
		if err != nil {
			writeSemanticError(w, r, err)
			return
		}
		w.Header().Set("ETag", semanticSettingsETag(settings))
		writeJSON(w, http.StatusOK, semanticSettingsWire(settings))
	})
	mux.HandleFunc("PUT /api/v1/libraries/{libraryId}/ai/semantic", func(w http.ResponseWriter, r *http.Request) {
		libraryID, err := parseResourceID(r.PathValue("libraryId"), "lib_")
		if err != nil {
			writeSemanticError(w, r, semantic.ErrSemanticLibraryNotFound)
			return
		}
		revision, err := parseAIRevisionETag(r.Header.Get("If-Match"), fmt.Sprintf("semantic-library-%d", libraryID))
		if err != nil {
			writeAIModelError(w, r, err)
			return
		}
		var payload semanticSettingsUpdateRequest
		if decodeAIModelJSON(r, &payload) != nil || payload.Enabled == nil {
			writePublicError(w, r, http.StatusBadRequest, codeInvalidRequest, "The request is invalid.")
			return
		}
		settings, err := service.UpdateLibrarySettings(r.Context(), libraryID, *payload.Enabled, revision)
		if err != nil {
			writeSemanticError(w, r, err)
			return
		}
		w.Header().Set("ETag", semanticSettingsETag(settings))
		writeJSON(w, http.StatusOK, semanticSettingsWire(settings))
	})
	mux.HandleFunc("POST /api/v1/libraries/{libraryId}/ai/semantic/jobs", func(w http.ResponseWriter, r *http.Request) {
		libraryID, err := parseResourceID(r.PathValue("libraryId"), "lib_")
		key := r.Header.Get("Idempotency-Key")
		if err != nil || libraryID < 1 || !idempotencyKeyPattern.MatchString(key) {
			writePublicError(w, r, http.StatusBadRequest, codeInvalidRequest, "The request is invalid.")
			return
		}
		var payload semanticJobRequest
		if decodeAIModelJSON(r, &payload) != nil || (payload.Mode != semantic.JobMissing && payload.Mode != semantic.JobAll) {
			writePublicError(w, r, http.StatusBadRequest, codeInvalidRequest, "The request is invalid.")
			return
		}
		operation, replayed, err := service.RequestBackfill(r.Context(), libraryID, payload.Mode, key)
		if err != nil {
			writeSemanticError(w, r, err)
			return
		}
		w.Header().Set("Location", "/api/v1/ai/operations/"+operation.ID)
		w.Header().Set("Idempotency-Replayed", strconv.FormatBool(replayed))
		w.Header().Set("ETag", aiOperationETag(operation))
		writeJSON(w, http.StatusAccepted, aiOperationWire(operation))
	})
	mux.HandleFunc("POST /api/v1/libraries/{libraryId}/ai/semantic/clear", func(w http.ResponseWriter, r *http.Request) {
		libraryID, err := parseResourceID(r.PathValue("libraryId"), "lib_")
		key := r.Header.Get("Idempotency-Key")
		if err != nil || !idempotencyKeyPattern.MatchString(key) {
			writePublicError(w, r, http.StatusBadRequest, codeInvalidRequest, "The request is invalid.")
			return
		}
		revision, err := parseAIRevisionETag(r.Header.Get("If-Match"), fmt.Sprintf("semantic-library-%d", libraryID))
		if err != nil {
			writeAIModelError(w, r, err)
			return
		}
		var payload semanticClearRequest
		if decodeAIModelJSON(r, &payload) != nil || payload.Confirmation != "clear_semantic_data" {
			writePublicError(w, r, http.StatusBadRequest, codeInvalidRequest, "The request is invalid.")
			return
		}
		operation, replayed, err := service.RequestClear(r.Context(), libraryID, revision, key)
		if err != nil {
			writeSemanticError(w, r, err)
			return
		}
		w.Header().Set("Location", "/api/v1/ai/operations/"+operation.ID)
		w.Header().Set("Idempotency-Replayed", strconv.FormatBool(replayed))
		w.Header().Set("ETag", aiOperationETag(operation))
		writeJSON(w, http.StatusAccepted, aiOperationWire(operation))
	})
}

func registerSemanticSearchRoute(mux *http.ServeMux, service SemanticSearchService, catalogService CatalogService) {
	registerSemanticSearchRouteWithAdmission(mux, service, catalogService, newSemanticSearchAdmission())
}

func registerSemanticSearchRouteWithAdmission(mux *http.ServeMux, service SemanticSearchService, catalogService CatalogService, admission *semanticSearchAdmission) {
	mux.HandleFunc("GET /api/v1/semantic/assets", func(w http.ResponseWriter, r *http.Request) {
		request, err := parseSemanticSearchQuery(r.URL.RawQuery)
		if err != nil {
			writeSemanticSearchError(w, r, err)
			return
		}
		if !admission.acquire() {
			w.Header().Set("Retry-After", "1")
			writePublicError(w, r, http.StatusTooManyRequests, "semantic_busy", "Semantic search is busy.")
			return
		}
		defer admission.release()
		result, err := service.Search(r.Context(), request)
		if err != nil {
			writeSemanticSearchError(w, r, err)
			return
		}
		assetIDs := make([]int64, 0, len(result.Matches))
		for _, match := range result.Matches {
			assetIDs = append(assetIDs, match.AssetID)
		}
		assets := []catalog.Asset{}
		if len(assetIDs) > 0 {
			assets, err = catalogService.GetAssetsByIDs(r.Context(), assetIDs)
			if err != nil {
				writeSemanticSearchError(w, r, err)
				return
			}
			if err := validateSemanticSearchAssets(result.Matches, assets); err != nil {
				writeSemanticSearchError(w, r, err)
				return
			}
		}
		items := make([]assetResponse, 0, len(assets))
		for _, asset := range assets {
			items = append(items, assetWire(asset))
		}
		var next *string
		if result.NextCursor != "" {
			next = &result.NextCursor
		}
		excluded := make([]semanticExcludedLibraryResponse, 0, len(result.Excluded))
		for _, value := range result.Excluded {
			excluded = append(excluded, semanticExcludedLibraryResponse{
				LibraryID: libraryID(value.LibraryID), Reason: value.Reason,
			})
		}
		writeJSON(w, http.StatusOK, semanticAssetPageResponse{
			Items: items, NextCursor: next,
			Coverage: semanticSearchCoverageResponse{
				Eligible: result.Coverage.Eligible, Completed: result.Coverage.Completed,
				Failed: result.Coverage.Failed, Stale: result.Coverage.Stale,
				Complete: result.Coverage.Complete(), ExcludedLibraries: excluded,
			},
		})
	})
}

func registerVideoSemanticSearchRoute(mux *http.ServeMux, service VideoSemanticSearchService, catalogService CatalogService) {
	registerVideoSemanticSearchRouteWithAdmission(mux, service, catalogService, newSemanticSearchAdmission())
}

func registerVideoSemanticJobRoute(mux *http.ServeMux, service VideoSemanticJobService) {
	mux.HandleFunc("POST /api/v1/libraries/{libraryId}/ai/video-semantic/jobs", func(w http.ResponseWriter, r *http.Request) {
		libraryID, err := parseResourceID(r.PathValue("libraryId"), "lib_")
		key := r.Header.Get("Idempotency-Key")
		if err != nil || !idempotencyKeyPattern.MatchString(key) {
			writePublicError(w, r, http.StatusBadRequest, codeInvalidRequest, "The request is invalid.")
			return
		}
		var payload semanticJobRequest
		if decodeAIModelJSON(r, &payload) != nil || (payload.Mode != semantic.JobMissing && payload.Mode != semantic.JobAll) {
			writePublicError(w, r, http.StatusBadRequest, codeInvalidRequest, "The request is invalid.")
			return
		}
		operation, replayed, err := service.RequestVideoJob(r.Context(), libraryID, payload.Mode, key)
		if err != nil {
			writeVideoSemanticJobError(w, r, err)
			return
		}
		w.Header().Set("Location", "/api/v1/ai/operations/"+operation.ID)
		w.Header().Set("Idempotency-Replayed", strconv.FormatBool(replayed))
		w.Header().Set("ETag", aiOperationETag(operation))
		writeJSON(w, http.StatusAccepted, aiOperationWire(operation))
	})
}

func writeVideoSemanticJobError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, semantic.ErrSemanticLibraryNotFound):
		writePublicError(w, r, http.StatusNotFound, "library_not_found", "The library was not found.")
	case errors.Is(err, semantic.ErrSemanticDisabled):
		writePublicError(w, r, http.StatusConflict, "ai_disabled", "Semantic features are disabled for this library.")
	case errors.Is(err, semantic.ErrSemanticGenerationUnavailable):
		writePublicError(w, r, http.StatusConflict, "model_unavailable", "The active semantic model is unavailable.")
	case errors.Is(err, semantic.ErrStoryboardNotReady):
		writePublicError(w, r, http.StatusConflict, "storyboard_not_ready", "Complete storyboards are not ready.")
	case errors.Is(err, semantic.ErrVideoJobConflict):
		writePublicError(w, r, http.StatusConflict, "idempotency_conflict", "The idempotency key was used for another request.")
	case errors.Is(err, semantic.ErrInvalidVideoJob):
		writePublicError(w, r, http.StatusBadRequest, codeInvalidRequest, "The request is invalid.")
	default:
		writePublicError(w, r, http.StatusInternalServerError, codeInternalError, "The request could not be completed.")
	}
}

func registerVideoSemanticSearchRouteWithAdmission(mux *http.ServeMux, service VideoSemanticSearchService, catalogService CatalogService, admission *semanticSearchAdmission) {
	mux.HandleFunc("GET /api/v1/semantic/videos", func(w http.ResponseWriter, r *http.Request) {
		request, err := parseSemanticSearchQuery(r.URL.RawQuery)
		if err != nil {
			writeVideoSemanticSearchError(w, r, err)
			return
		}
		if !admission.acquire() {
			w.Header().Set("Retry-After", "1")
			writePublicError(w, r, http.StatusTooManyRequests, "semantic_busy", "Semantic search is busy.")
			return
		}
		defer admission.release()
		result, err := service.Search(r.Context(), request)
		if err != nil {
			writeVideoSemanticSearchError(w, r, err)
			return
		}
		ids := make([]int64, len(result.Matches))
		for index, match := range result.Matches {
			ids[index] = match.AssetID
		}
		assets := []catalog.Asset{}
		if len(ids) > 0 {
			assets, err = catalogService.GetAssetsByIDs(r.Context(), ids)
			if err != nil || validateVideoSemanticSearchAssets(result.Matches, assets) != nil {
				writeVideoSemanticSearchError(w, r, catalog.ErrAssetNotFound)
				return
			}
		}
		items := make([]semanticVideoHitResponse, len(assets))
		for index, asset := range assets {
			match := result.Matches[index]
			items[index] = semanticVideoHitResponse{Asset: assetWire(asset), Score: match.Score,
				MatchedFrame: semanticVideoMatchedFrameResponse{Ordinal: match.Ordinal, TimestampMS: match.TimestampMS, PlanSize: match.PlanSize}}
		}
		var next *string
		if result.NextCursor != "" {
			next = &result.NextCursor
		}
		excluded := make([]semanticExcludedLibraryResponse, len(result.Excluded))
		for index, value := range result.Excluded {
			excluded[index] = semanticExcludedLibraryResponse{LibraryID: libraryID(value.LibraryID), Reason: value.Reason}
		}
		writeJSON(w, http.StatusOK, semanticVideoPageResponse{Items: items, NextCursor: next,
			Coverage: derivedCoverageResponse{Eligible: result.Coverage.Eligible, Completed: result.Coverage.Ready,
				Degraded: result.Coverage.Degraded, Failed: result.Coverage.Failed, Stale: result.Coverage.Stale,
				Complete: result.Coverage.Complete(), Revision: result.Coverage.Revision, ExcludedLibraries: excluded}})
	})
}

func validateVideoSemanticSearchAssets(matches []semantic.VideoVectorMatch, assets []catalog.Asset) error {
	if len(matches) != len(assets) {
		return catalog.ErrAssetNotFound
	}
	for index, match := range matches {
		asset := assets[index]
		if asset.ID != match.AssetID || asset.LibraryID != match.LibraryID || asset.Kind != catalog.KindVideo ||
			asset.Availability != catalog.SourceAvailable {
			return catalog.ErrAssetNotFound
		}
	}
	return nil
}

func validateSemanticSearchAssets(matches []semantic.VectorMatch, assets []catalog.Asset) error {
	if len(matches) != len(assets) {
		return catalog.ErrAssetNotFound
	}
	for index, match := range matches {
		asset := assets[index]
		if asset.ID != match.AssetID || asset.LibraryID != match.LibraryID {
			return catalog.ErrAssetNotFound
		}
		if asset.Availability != catalog.SourceAvailable {
			return semantic.ErrSemanticLibraryOffline
		}
	}
	return nil
}

func parseSemanticSearchQuery(raw string) (semantic.SearchRequest, error) {
	if len(raw) > maxSemanticSearchRawQueryBytes {
		return semantic.SearchRequest{}, semantic.ErrInvalidSemanticSearch
	}
	values, err := url.ParseQuery(raw)
	if err != nil {
		return semantic.SearchRequest{}, semantic.ErrInvalidSemanticSearch
	}
	allowed := map[string]bool{"q": true, "libraryId": true, "directoryId": true, "recursive": true, "cursor": true, "limit": true}
	for key, entries := range values {
		if !allowed[key] || len(entries) != 1 {
			return semantic.SearchRequest{}, semantic.ErrInvalidSemanticSearch
		}
	}
	request := semantic.SearchRequest{Query: values.Get("q"), Cursor: values.Get("cursor"), Limit: 50}
	if request.Query == "" {
		return semantic.SearchRequest{}, semantic.ErrInvalidQuery
	}
	if rawLibraryID := values.Get("libraryId"); rawLibraryID != "" {
		request.LibraryID, err = parseResourceID(rawLibraryID, "lib_")
		if err != nil {
			return semantic.SearchRequest{}, semantic.ErrInvalidSemanticSearch
		}
	}
	if rawDirectoryID := values.Get("directoryId"); rawDirectoryID != "" {
		request.DirectoryID, err = parseResourceID(rawDirectoryID, "dir_")
		if err != nil || request.LibraryID == 0 {
			return semantic.SearchRequest{}, semantic.ErrInvalidSemanticSearch
		}
	}
	if rawRecursive := values.Get("recursive"); rawRecursive != "" {
		request.Recursive, err = strconv.ParseBool(rawRecursive)
		if err != nil {
			return semantic.SearchRequest{}, semantic.ErrInvalidSemanticSearch
		}
	}
	if rawLimit := values.Get("limit"); rawLimit != "" {
		request.Limit, err = strconv.Atoi(rawLimit)
		if err != nil || request.Limit < 1 || request.Limit > semantic.MaxSemanticSearchLimit {
			return semantic.SearchRequest{}, semantic.ErrInvalidSemanticSearch
		}
	}
	return request, nil
}

func semanticSettingsWire(value semantic.LibrarySettings) semanticSettingsResponse {
	var generation *string
	if value.ActiveGenerationID != "" {
		current := value.ActiveGenerationID
		generation = &current
	}
	return semanticSettingsResponse{LibraryID: libraryID(value.LibraryID), Enabled: value.Enabled, State: value.State, Revision: value.Revision, ActiveGenerationID: generation,
		Coverage: semanticCoverageResponse{Eligible: value.Coverage.Eligible, Completed: value.Coverage.Completed, Failed: value.Coverage.Failed, Stale: value.Coverage.Stale, Revision: value.Coverage.Revision, Complete: value.Coverage.Complete()}}
}

func semanticSettingsETag(value semantic.LibrarySettings) string {
	return fmt.Sprintf(`"semantic-library-%d-r%d"`, value.LibraryID, value.Revision)
}

func writeSemanticError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, semantic.ErrSemanticLibraryNotFound):
		writePublicError(w, r, http.StatusNotFound, "library_not_found", "The library was not found.")
	case errors.Is(err, semantic.ErrSemanticRevisionConflict):
		writePublicError(w, r, http.StatusPreconditionFailed, "precondition_failed", "The resource revision changed.")
	case errors.Is(err, semantic.ErrSemanticDisabled):
		writePublicError(w, r, http.StatusConflict, "ai_disabled", "Semantic search is disabled.")
	case errors.Is(err, semantic.ErrSemanticGenerationUnavailable):
		writePublicError(w, r, http.StatusConflict, "model_unavailable", "No compatible model is available.")
	case errors.Is(err, semantic.ErrSemanticJobConflict):
		writePublicError(w, r, http.StatusConflict, "idempotency_conflict", "The request conflicts with active work.")
	case errors.Is(err, semantic.ErrSemanticClearConflict):
		writePublicError(w, r, http.StatusConflict, "idempotency_conflict", "The request conflicts with active work.")
	case errors.Is(err, semantic.ErrInvalidSemanticClear):
		writePublicError(w, r, http.StatusBadRequest, codeInvalidRequest, "The request is invalid.")
	default:
		writePublicError(w, r, http.StatusInternalServerError, codeInternalError, "The request could not be completed.")
	}
}

func writeSemanticSearchError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, semantic.ErrInvalidSemanticCursor):
		writePublicError(w, r, http.StatusBadRequest, "invalid_cursor", "The cursor is invalid.")
	case errors.Is(err, semantic.ErrInvalidSemanticSearch), errors.Is(err, semantic.ErrInvalidQuery),
		errors.Is(err, semantic.ErrSemanticScopeNotFound), errors.Is(err, semantic.ErrSemanticLibraryNotFound):
		writePublicError(w, r, http.StatusBadRequest, codeInvalidRequest, "The request is invalid.")
	case errors.Is(err, semantic.ErrSemanticCursorStale):
		writePublicError(w, r, http.StatusConflict, "semantic_cursor_stale", "The semantic search snapshot changed.")
	case errors.Is(err, semantic.ErrSemanticDisabled):
		writePublicError(w, r, http.StatusConflict, "ai_disabled", "Semantic search is disabled.")
	case errors.Is(err, semantic.ErrSemanticLibraryOffline):
		writePublicError(w, r, http.StatusConflict, "semantic_not_ready", "The semantic search scope is not ready.")
	case errors.Is(err, semantic.ErrSemanticGenerationUnavailable):
		writePublicError(w, r, http.StatusServiceUnavailable, "model_unavailable", "No compatible model is available.")
	case errors.Is(err, catalog.ErrAssetNotFound):
		writePublicError(w, r, http.StatusConflict, "semantic_cursor_stale", "The semantic search snapshot changed.")
	default:
		writePublicError(w, r, http.StatusInternalServerError, codeInternalError, "The request could not be completed.")
	}
}

func writeVideoSemanticSearchError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, semantic.ErrInvalidSemanticCursor):
		writePublicError(w, r, http.StatusBadRequest, "invalid_cursor", "The cursor is invalid.")
	case errors.Is(err, semantic.ErrInvalidSemanticSearch), errors.Is(err, semantic.ErrInvalidQuery),
		errors.Is(err, semantic.ErrSemanticScopeNotFound), errors.Is(err, semantic.ErrSemanticLibraryNotFound):
		writePublicError(w, r, http.StatusBadRequest, codeInvalidRequest, "The request is invalid.")
	case errors.Is(err, semantic.ErrSemanticCursorStale):
		writePublicError(w, r, http.StatusConflict, "semantic_cursor_stale", "The semantic search snapshot changed.")
	case errors.Is(err, semantic.ErrSemanticDisabled):
		writePublicError(w, r, http.StatusConflict, "ai_disabled", "Semantic search is disabled.")
	case errors.Is(err, semantic.ErrSemanticLibraryOffline), errors.Is(err, semantic.ErrStoryboardNotReady):
		writePublicError(w, r, http.StatusConflict, "video_semantic_not_ready", "Video semantic search is not ready.")
	case errors.Is(err, semantic.ErrSemanticGenerationUnavailable):
		writePublicError(w, r, http.StatusServiceUnavailable, "model_unavailable", "No compatible model is available.")
	case errors.Is(err, catalog.ErrAssetNotFound):
		writePublicError(w, r, http.StatusConflict, "semantic_cursor_stale", "The semantic search snapshot changed.")
	default:
		writePublicError(w, r, http.StatusInternalServerError, codeInternalError, "The request could not be completed.")
	}
}
