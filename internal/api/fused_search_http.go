package api

import (
	"context"
	"errors"
	"net/http"
	"net/url"

	"github.com/HappyQuQu/foliopath/internal/catalog"
	"github.com/HappyQuQu/foliopath/internal/search"
)

type FusedSearchService interface {
	Search(context.Context, search.Request) (search.Result, error)
	Verify(context.Context, search.Request, search.Result) error
}
type fusedHitResponse struct {
	OCR           []search.OCRExcerpt `json:"ocr"`
	Asset         assetResponse       `json:"asset"`
	Sources       []search.Source     `json:"sources"`
	SemanticScore *float32            `json:"semanticScore"`
}
type fusedPageResponse struct {
	OCRState      string                          `json:"ocrState"`
	OCRCoverage   *search.OCRCoverage             `json:"ocrCoverage"`
	Items         []fusedHitResponse              `json:"items"`
	NextCursor    *string                         `json:"nextCursor"`
	SemanticState string                          `json:"semanticState"`
	Coverage      *semanticSearchCoverageResponse `json:"coverage"`
}

func registerFusedSearchRoute(mux *http.ServeMux, service FusedSearchService, catalogService CatalogService, admission *semanticSearchAdmission) {
	mux.HandleFunc("GET /api/v1/search/assets", func(w http.ResponseWriter, r *http.Request) {
		request, e := parseFusedSearchQuery(r.URL.RawQuery)
		if e != nil {
			writeFusedSearchError(w, r, e)
			return
		}
		if !admission.acquire() {
			w.Header().Set("Retry-After", "1")
			writePublicError(w, r, http.StatusTooManyRequests, "semantic_busy", "Search is busy.")
			return
		}
		defer admission.release()
		result, e := service.Search(r.Context(), request)
		if e != nil {
			writeFusedSearchError(w, r, e)
			return
		}
		ids := make([]int64, len(result.Hits))
		for i, h := range result.Hits {
			ids[i] = h.AssetID
		}
		assets := []catalog.Asset{}
		if len(ids) > 0 {
			assets, e = catalogService.GetAssetsByIDs(r.Context(), ids)
			if e != nil {
				writeFusedSearchError(w, r, e)
				return
			}
		}
		if len(assets) != len(result.Hits) {
			writeFusedSearchError(w, r, search.ErrStale)
			return
		}
		page := fusedPageResponse{Items: make([]fusedHitResponse, 0, len(assets)), SemanticState: result.SemanticState, OCRState: result.OCRState, OCRCoverage: result.OCRCoverage}
		for i, a := range assets {
			h := result.Hits[i]
			if a.ID != h.AssetID || a.LibraryID != h.LibraryID || h.Evidence != search.Evidence(a.SourceFingerprint, a.Name, a.RelativePath) {
				writeFusedSearchError(w, r, search.ErrStale)
				return
			}
			page.Items = append(page.Items, fusedHitResponse{Asset: assetWire(a), Sources: h.Sources, SemanticScore: h.SemanticScore, OCR: h.OCR})
		}
		if result.NextCursor != "" {
			page.NextCursor = &result.NextCursor
		}
		if c := result.Coverage; c != nil {
			excluded := make([]semanticExcludedLibraryResponse, 0, len(result.Excluded))
			for _, x := range result.Excluded {
				excluded = append(excluded, semanticExcludedLibraryResponse{LibraryID: libraryID(x.LibraryID), Reason: x.Reason})
			}
			page.Coverage = &semanticSearchCoverageResponse{Eligible: c.Eligible, Completed: c.Completed, Failed: c.Failed, Stale: c.Stale, Complete: c.Complete(), ExcludedLibraries: excluded}
		}
		if e = service.Verify(r.Context(), request, result); e != nil {
			writeFusedSearchError(w, r, e)
			return
		}
		writeJSON(w, http.StatusOK, page)
	})
}
func parseFusedSearchQuery(raw string) (search.Request, error) {
	if len(raw) > maxSemanticSearchRawQueryBytes {
		return search.Request{}, search.ErrInvalid
	}
	v, e := url.ParseQuery(raw)
	if e != nil {
		return search.Request{}, search.ErrInvalid
	}
	allowed := map[string]bool{"q": true, "mode": true, "libraryId": true, "directoryId": true, "recursive": true, "kind": true, "modifiedFrom": true, "modifiedBefore": true, "cursor": true, "limit": true}
	for k, entries := range v {
		if !allowed[k] || len(entries) != 1 || entries[0] == "" {
			return search.Request{}, search.ErrInvalid
		}
	}
	r := search.Request{Mode: v.Get("mode")}
	v.Del("mode")
	if id := v.Get("libraryId"); id != "" {
		r.LibraryID, e = parseResourceID(id, "lib_")
		if e != nil {
			return r, search.ErrInvalid
		}
	}
	v.Del("libraryId")
	a, e := parseAssetListQuery(v.Encode())
	if e != nil {
		return r, e
	}
	if a.SearchQuery == nil {
		return r, search.ErrInvalid
	}
	r.Query = *a.SearchQuery
	r.DirectoryID = a.DirectoryID
	r.Recursive = a.Recursive
	r.Kinds = a.Kinds
	r.ModifiedFromNS = a.ModifiedFromNS
	r.ModifiedBeforeNS = a.ModifiedBeforeNS
	r.Cursor = a.Cursor
	r.Limit = a.Limit
	if r.Limit == 0 {
		r.Limit = 50
	}
	return r, nil
}
func writeFusedSearchError(w http.ResponseWriter, r *http.Request, e error) {
	switch {
	case errors.Is(e, search.ErrInvalid), errors.Is(e, catalog.ErrInvalidQuery), errors.Is(e, catalog.ErrLibraryNotFound), errors.Is(e, catalog.ErrDirectoryNotFound):
		writePublicError(w, r, http.StatusBadRequest, codeInvalidRequest, "The search request is invalid.")
	case errors.Is(e, search.ErrCursor), errors.Is(e, catalog.ErrInvalidCursor):
		writePublicError(w, r, http.StatusBadRequest, "invalid_cursor", "The cursor is invalid.")
	case errors.Is(e, search.ErrStale), errors.Is(e, catalog.ErrAssetNotFound):
		writePublicError(w, r, http.StatusConflict, "search_cursor_stale", "The search snapshot changed. Refresh the results.")
	case errors.Is(e, search.ErrUnavailable):
		writePublicError(w, r, http.StatusServiceUnavailable, "model_unavailable", "The semantic source is unavailable.")
	default:
		writeSemanticSearchError(w, r, e)
	}
}
