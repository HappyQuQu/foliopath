package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
	"github.com/HappyQuQu/foliopath/internal/catalog"
	"github.com/HappyQuQu/foliopath/internal/curation"
	"github.com/HappyQuQu/foliopath/internal/semantic"
)

type AITagVocabularyService interface {
	Get(context.Context) (semantic.TagVocabulary, error)
	Publish(context.Context, int64, []int64) (semantic.TagVocabulary, error)
}

type AITagSuggestionListService interface {
	List(context.Context, semantic.TagSuggestionListRequest) (semantic.TagSuggestionPage, error)
}

type AITagReviewService interface {
	Review(context.Context, string, []semantic.TagReviewItem) (semantic.IdempotentTagReviewResult, error)
}

type AITagReviewClearService interface {
	RequestTagReviewClear(context.Context, int64, int64, string) (aimodel.Operation, bool, error)
}

type AITagJobService interface {
	RequestTagJob(context.Context, int64, semantic.JobMode, string) (aimodel.Operation, bool, error)
}

type aiTagVocabularyUpdateRequest struct {
	TagIDs []string `json:"tagIds"`
}

type aiTagVocabularyEntryResponse struct {
	TagID string `json:"tagId"`
	Name  string `json:"name"`
}

type aiTagVocabularyResponse struct {
	Revision    int64                          `json:"revision"`
	Entries     []aiTagVocabularyEntryResponse `json:"entries"`
	PublishedAt string                         `json:"publishedAt"`
}

type aiTagSuggestionResponse struct {
	ID                 string                           `json:"id"`
	LibraryID          string                           `json:"libraryId"`
	Asset              assetResponse                    `json:"asset"`
	Tag                aiTagVocabularyEntryResponse     `json:"tag"`
	Confidence         float32                          `json:"confidence"`
	Status             semantic.TagSuggestionListStatus `json:"status"`
	GenerationID       string                           `json:"generationId"`
	VocabularyRevision int64                            `json:"vocabularyRevision"`
	Revision           int64                            `json:"revision"`
	ReviewedAt         *string                          `json:"reviewedAt"`
}

type aiTagSuggestionPageResponse struct {
	Items      []aiTagSuggestionResponse `json:"items"`
	NextCursor *string                   `json:"nextCursor"`
	Coverage   derivedCoverageResponse   `json:"coverage"`
}

type aiTagReviewRequest struct {
	Items []struct {
		SuggestionID               string `json:"suggestionId"`
		Action                     string `json:"action"`
		ExpectedSuggestionRevision int64  `json:"expectedSuggestionRevision"`
		ExpectedCurationRevision   int64  `json:"expectedCurationRevision"`
	} `json:"items"`
}

type aiTagReviewClearRequest struct {
	Confirmation string `json:"confirmation"`
}

type aiTagReviewResponse struct {
	Items []struct {
		SuggestionID string `json:"suggestionId"`
		Outcome      string `json:"outcome"`
		Revision     int64  `json:"revision"`
	} `json:"items"`
}

func registerAITagVocabularyRoutes(mux *http.ServeMux, service AITagVocabularyService) {
	mux.HandleFunc("GET /api/v1/ai/tag-vocabulary", func(w http.ResponseWriter, r *http.Request) {
		value, err := service.Get(r.Context())
		if err != nil {
			writeAITagVocabularyError(w, r, err)
			return
		}
		w.Header().Set("ETag", aiTagVocabularyETag(value.Revision))
		writeJSON(w, http.StatusOK, aiTagVocabularyWire(value))
	})
	mux.HandleFunc("PUT /api/v1/ai/tag-vocabulary", func(w http.ResponseWriter, r *http.Request) {
		revision, err := parseAIRevisionETag(r.Header.Get("If-Match"), "ai-tag-vocabulary")
		if err != nil {
			writeAIModelError(w, r, err)
			return
		}
		var payload aiTagVocabularyUpdateRequest
		if decodeAIModelJSON(r, &payload) != nil || len(payload.TagIDs) > semantic.MaxControlledVocabularyEntries {
			writePublicError(w, r, http.StatusBadRequest, codeInvalidRequest, "The request is invalid.")
			return
		}
		ids := make([]int64, len(payload.TagIDs))
		seen := make(map[int64]struct{}, len(ids))
		for index, raw := range payload.TagIDs {
			id, parseErr := parseResourceID(raw, "tag_")
			if parseErr != nil {
				writePublicError(w, r, http.StatusBadRequest, codeInvalidRequest, "The request is invalid.")
				return
			}
			if _, exists := seen[id]; exists {
				writePublicError(w, r, http.StatusUnprocessableEntity, "validation_failed", "The controlled vocabulary contains duplicate tags.")
				return
			}
			seen[id], ids[index] = struct{}{}, id
		}
		value, err := service.Publish(r.Context(), revision, ids)
		if err != nil {
			writeAITagVocabularyError(w, r, err)
			return
		}
		w.Header().Set("ETag", aiTagVocabularyETag(value.Revision))
		writeJSON(w, http.StatusOK, aiTagVocabularyWire(value))
	})
}

func registerAITagReviewRoute(mux *http.ServeMux, service AITagReviewService) {
	mux.HandleFunc("POST /api/v1/ai/tag-suggestion-reviews", func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("Idempotency-Key")
		var payload aiTagReviewRequest
		if !idempotencyKeyPattern.MatchString(key) || decodeAIModelJSON(r, &payload) != nil ||
			len(payload.Items) < 1 || len(payload.Items) > semantic.MaxTagSuggestionReviewBatch {
			writePublicError(w, r, http.StatusBadRequest, codeInvalidRequest, "The request is invalid.")
			return
		}
		items := make([]semantic.TagReviewItem, len(payload.Items))
		for index, item := range payload.Items {
			action := semantic.TagReviewDecision("")
			switch item.Action {
			case "accept":
				action = semantic.TagReviewAccept
			case "dismiss":
				action = semantic.TagReviewDismiss
			}
			items[index] = semantic.TagReviewItem{SuggestionID: item.SuggestionID, Action: action,
				ExpectedSuggestionRevision: item.ExpectedSuggestionRevision,
				ExpectedCurationRevision:   item.ExpectedCurationRevision}
		}
		result, err := service.Review(r.Context(), key, items)
		if err != nil {
			writeAITagReviewError(w, r, err)
			return
		}
		response := aiTagReviewResponse{Items: make([]struct {
			SuggestionID string `json:"suggestionId"`
			Outcome      string `json:"outcome"`
			Revision     int64  `json:"revision"`
		}, len(result.Items))}
		for index, outcome := range result.Items {
			value := string(outcome.Outcome)
			if outcome.Conflict {
				value = "conflict"
			}
			response.Items[index].SuggestionID = outcome.SuggestionID
			response.Items[index].Outcome = value
			response.Items[index].Revision = outcome.Revision
		}
		w.Header().Set("Idempotency-Replayed", strconv.FormatBool(result.Replayed))
		writeJSON(w, http.StatusOK, response)
	})
}

func registerAITagReviewClearRoute(mux *http.ServeMux, service AITagReviewClearService) {
	mux.HandleFunc("POST /api/v1/libraries/{libraryId}/ai/tag-suggestion-reviews/clear", func(w http.ResponseWriter, r *http.Request) {
		libraryID, err := parseResourceID(r.PathValue("libraryId"), "lib_")
		key := r.Header.Get("Idempotency-Key")
		if err != nil || !idempotencyKeyPattern.MatchString(key) {
			writePublicError(w, r, http.StatusBadRequest, codeInvalidRequest, "The request is invalid.")
			return
		}
		revision, err := parseAIRevisionETag(r.Header.Get("If-Match"), fmt.Sprintf("ai-tag-reviews-%d", libraryID))
		if err != nil {
			writeAIModelError(w, r, err)
			return
		}
		var payload aiTagReviewClearRequest
		if decodeAIModelJSON(r, &payload) != nil || payload.Confirmation != "clear_tag_suggestion_reviews" {
			writePublicError(w, r, http.StatusBadRequest, codeInvalidRequest, "The request is invalid.")
			return
		}
		operation, replayed, err := service.RequestTagReviewClear(r.Context(), libraryID, revision, key)
		if err != nil {
			writeAITagReviewClearError(w, r, err)
			return
		}
		w.Header().Set("Location", "/api/v1/ai/operations/"+operation.ID)
		w.Header().Set("Idempotency-Replayed", strconv.FormatBool(replayed))
		w.Header().Set("ETag", aiOperationETag(operation))
		writeJSON(w, http.StatusAccepted, aiOperationWire(operation))
	})
}

func registerAITagJobRoute(mux *http.ServeMux, service AITagJobService) {
	mux.HandleFunc("POST /api/v1/libraries/{libraryId}/ai/tag-suggestions/jobs", func(w http.ResponseWriter, r *http.Request) {
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
		operation, replayed, err := service.RequestTagJob(r.Context(), libraryID, payload.Mode, key)
		if err != nil {
			writeAITagJobError(w, r, err)
			return
		}
		w.Header().Set("Location", "/api/v1/ai/operations/"+operation.ID)
		w.Header().Set("Idempotency-Replayed", strconv.FormatBool(replayed))
		w.Header().Set("ETag", aiOperationETag(operation))
		writeJSON(w, http.StatusAccepted, aiOperationWire(operation))
	})
}

func writeAITagJobError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, semantic.ErrSemanticLibraryNotFound):
		writePublicError(w, r, http.StatusNotFound, "library_not_found", "The library was not found.")
	case errors.Is(err, semantic.ErrSemanticDisabled):
		writePublicError(w, r, http.StatusConflict, "ai_disabled", "Semantic features are disabled for this library.")
	case errors.Is(err, semantic.ErrSemanticGenerationUnavailable):
		writePublicError(w, r, http.StatusConflict, "model_unavailable", "The active semantic model is unavailable.")
	case errors.Is(err, semantic.ErrTagVocabularyUnavailable):
		writePublicError(w, r, http.StatusConflict, "vocabulary_unavailable", "The controlled tag vocabulary is unavailable.")
	case errors.Is(err, semantic.ErrTagJobConflict):
		writePublicError(w, r, http.StatusConflict, "idempotency_conflict", "The idempotency key was used for another request.")
	case errors.Is(err, semantic.ErrInvalidTagJob):
		writePublicError(w, r, http.StatusBadRequest, codeInvalidRequest, "The request is invalid.")
	default:
		writePublicError(w, r, http.StatusInternalServerError, codeInternalError, "The request could not be completed.")
	}
}

func writeAITagReviewClearError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, semantic.ErrTagReviewClearNotFound):
		writePublicError(w, r, http.StatusNotFound, "library_not_found", "The library was not found.")
	case errors.Is(err, semantic.ErrTagReviewClearConflict):
		writePublicError(w, r, http.StatusPreconditionFailed, "precondition_failed", "The review snapshot changed.")
	case errors.Is(err, semantic.ErrInvalidTagReviewClear):
		writePublicError(w, r, http.StatusBadRequest, codeInvalidRequest, "The request is invalid.")
	default:
		writePublicError(w, r, http.StatusInternalServerError, codeInternalError, "The request could not be completed.")
	}
}

func writeAITagReviewError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, semantic.ErrTagReviewRequestConflict):
		writePublicError(w, r, http.StatusConflict, "idempotency_conflict", "The idempotency key was used for another request.")
	case errors.Is(err, semantic.ErrInvalidTagSuggestion):
		writePublicError(w, r, http.StatusUnprocessableEntity, "validation_failed", "The review request is invalid.")
	default:
		writePublicError(w, r, http.StatusInternalServerError, codeInternalError, "The request could not be completed.")
	}
}

func registerAITagSuggestionListRoute(mux *http.ServeMux, service AITagSuggestionListService, catalogService CatalogService) {
	mux.HandleFunc("GET /api/v1/libraries/{libraryId}/ai/tag-suggestions", func(w http.ResponseWriter, r *http.Request) {
		request, err := parseAITagSuggestionListRequest(r)
		if err != nil {
			writeAITagSuggestionListError(w, r, err)
			return
		}
		page, err := service.List(r.Context(), request)
		if err != nil {
			writeAITagSuggestionListError(w, r, err)
			return
		}
		w.Header().Set("ETag", aiTagReviewETag(request.LibraryID, page.ReviewRevision))
		ids := make([]int64, len(page.Items))
		for index, item := range page.Items {
			ids[index] = item.AssetID
		}
		assets := []catalog.Asset{}
		if len(ids) > 0 {
			assets, err = catalogService.GetAssetsByIDs(r.Context(), ids)
			if err != nil || len(assets) != len(page.Items) {
				writeAITagSuggestionListError(w, r, semantic.ErrTagSuggestionCursorStale)
				return
			}
		}
		items := make([]aiTagSuggestionResponse, len(page.Items))
		for index, item := range page.Items {
			asset := assets[index]
			if asset.ID != item.AssetID || asset.LibraryID != item.LibraryID || asset.Availability != catalog.SourceAvailable {
				writeAITagSuggestionListError(w, r, semantic.ErrTagSuggestionCursorStale)
				return
			}
			var reviewedAt *string
			if item.ReviewedAt != nil {
				value := item.ReviewedAt.UTC().Format(time.RFC3339Nano)
				reviewedAt = &value
			}
			items[index] = aiTagSuggestionResponse{ID: item.ID, LibraryID: libraryID(item.LibraryID), Asset: assetWire(asset),
				Tag:        aiTagVocabularyEntryResponse{TagID: "tag_" + strconv.FormatInt(item.TagID, 10), Name: item.TagName},
				Confidence: item.Confidence, Status: item.Status, GenerationID: item.GenerationID,
				VocabularyRevision: item.VocabularyRevision, Revision: item.Revision, ReviewedAt: reviewedAt}
		}
		var next *string
		if page.NextCursor != "" {
			next = &page.NextCursor
		}
		writeJSON(w, http.StatusOK, aiTagSuggestionPageResponse{Items: items, NextCursor: next,
			Coverage: derivedCoverageResponse{Eligible: page.Coverage.Eligible, Completed: page.Coverage.Completed,
				Degraded: page.Coverage.Degraded, Failed: page.Coverage.Failed, Stale: page.Coverage.Stale,
				Complete: page.Coverage.Complete(), Revision: page.Coverage.Revision}})
	})
}

func aiTagReviewETag(libraryID, revision int64) string {
	return fmt.Sprintf(`"ai-tag-reviews-%d-r%d"`, libraryID, revision)
}

func parseAITagSuggestionListRequest(r *http.Request) (semantic.TagSuggestionListRequest, error) {
	library, err := parseResourceID(r.PathValue("libraryId"), "lib_")
	if err != nil || len(r.URL.RawQuery) > maxSemanticSearchRawQueryBytes {
		return semantic.TagSuggestionListRequest{}, semantic.ErrInvalidTagSuggestion
	}
	values := r.URL.Query()
	for key := range values {
		if key != "status" && key != "tagId" && key != "cursor" && key != "limit" {
			return semantic.TagSuggestionListRequest{}, semantic.ErrInvalidTagSuggestion
		}
		if len(values[key]) != 1 {
			return semantic.TagSuggestionListRequest{}, semantic.ErrInvalidTagSuggestion
		}
	}
	request := semantic.TagSuggestionListRequest{LibraryID: library, Status: semantic.TagSuggestionPending, Limit: 50}
	if raw := values.Get("status"); raw != "" {
		request.Status = semantic.TagSuggestionListStatus(raw)
	}
	if raw := values.Get("tagId"); raw != "" {
		request.TagID, err = parseResourceID(raw, "tag_")
		if err != nil {
			return semantic.TagSuggestionListRequest{}, semantic.ErrInvalidTagSuggestion
		}
	}
	request.Cursor = values.Get("cursor")
	if raw := values.Get("limit"); raw != "" {
		value, parseErr := strconv.Atoi(raw)
		if parseErr != nil {
			return semantic.TagSuggestionListRequest{}, semantic.ErrInvalidTagSuggestion
		}
		request.Limit = value
	}
	return request, nil
}

func writeAITagSuggestionListError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, semantic.ErrInvalidTagSuggestionCursor):
		writePublicError(w, r, http.StatusBadRequest, "invalid_cursor", "The cursor is invalid.")
	case errors.Is(err, semantic.ErrTagSuggestionCursorStale):
		writePublicError(w, r, http.StatusConflict, "suggestion_cursor_stale", "The suggestion snapshot changed.")
	case errors.Is(err, semantic.ErrSemanticLibraryNotFound):
		writePublicError(w, r, http.StatusNotFound, "library_not_found", "The library was not found.")
	case errors.Is(err, semantic.ErrInvalidTagSuggestion):
		writePublicError(w, r, http.StatusBadRequest, codeInvalidRequest, "The request is invalid.")
	default:
		writePublicError(w, r, http.StatusInternalServerError, codeInternalError, "The request could not be completed.")
	}
}

func aiTagVocabularyWire(value semantic.TagVocabulary) aiTagVocabularyResponse {
	entries := make([]aiTagVocabularyEntryResponse, len(value.Entries))
	for index, entry := range value.Entries {
		entries[index] = aiTagVocabularyEntryResponse{TagID: "tag_" + strconv.FormatInt(entry.TagID, 10), Name: entry.Name}
	}
	return aiTagVocabularyResponse{Revision: value.Revision, Entries: entries, PublishedAt: value.PublishedAt.UTC().Format(time.RFC3339Nano)}
}

func aiTagVocabularyETag(revision int64) string {
	return fmt.Sprintf(`"ai-tag-vocabulary-r%d"`, revision)
}

func writeAITagVocabularyError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, semantic.ErrTagVocabularyConflict):
		writePublicError(w, r, http.StatusPreconditionFailed, "precondition_failed", "The resource revision changed.")
	case errors.Is(err, curation.ErrTagNotFound):
		writePublicError(w, r, http.StatusNotFound, "tag_not_found", "A tag was not found.")
	case errors.Is(err, semantic.ErrInvalidTagSuggestion):
		writePublicError(w, r, http.StatusUnprocessableEntity, "validation_failed", "The controlled vocabulary is invalid.")
	default:
		writePublicError(w, r, http.StatusInternalServerError, codeInternalError, "The request could not be completed.")
	}
}
