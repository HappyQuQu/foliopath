package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
	"github.com/HappyQuQu/foliopath/internal/catalog"
	"github.com/HappyQuQu/foliopath/internal/semantic"
)

type aiTagVocabularyHTTPStub struct {
	value            semantic.TagVocabulary
	expectedRevision int64
	tagIDs           []int64
}

type aiTagMutationHTTPStub struct {
	operation         aimodel.Operation
	library, revision int64
	mode              semantic.JobMode
	key               string
}

func (stub *aiTagMutationHTTPStub) RequestTagJob(_ context.Context, library int64, mode semantic.JobMode, key string) (aimodel.Operation, bool, error) {
	stub.library, stub.mode, stub.key = library, mode, key
	return stub.operation, true, nil
}
func (stub *aiTagMutationHTTPStub) RequestTagReviewClear(_ context.Context, library, revision int64, key string) (aimodel.Operation, bool, error) {
	stub.library, stub.revision, stub.key = library, revision, key
	return stub.operation, false, nil
}

func TestAITagJobAndReviewClearHTTPContracts(t *testing.T) {
	now := time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)
	total := int64(2)
	stub := &aiTagMutationHTTPStub{operation: aimodel.Operation{ID: "aio_tag_mutation", Kind: aimodel.OperationTagSuggestionMissing, State: aimodel.OperationQueued, Phase: aimodel.PhaseQueued, TotalItems: &total, Revision: 1, CreatedAt: now, UpdatedAt: now}}
	mux := http.NewServeMux()
	registerAITagJobRoute(mux, stub)
	registerAITagReviewClearRoute(mux, stub)
	jobRequest := httptest.NewRequest(http.MethodPost, "/api/v1/libraries/lib_7/ai/tag-suggestions/jobs", strings.NewReader(`{"mode":"missing"}`))
	jobRequest.Header.Set("Content-Type", "application/json")
	jobRequest.Header.Set("Idempotency-Key", "tag-job-key-001")
	job := httptest.NewRecorder()
	mux.ServeHTTP(job, jobRequest)
	if job.Code != http.StatusAccepted || stub.library != 7 || stub.mode != semantic.JobMissing || stub.key != "tag-job-key-001" || job.Header().Get("Idempotency-Replayed") != "true" {
		t.Fatalf("job=%d stub=%#v headers=%v body=%s", job.Code, stub, job.Header(), job.Body.String())
	}
	stub.operation.Kind = aimodel.OperationTagReviewClear
	clearRequest := httptest.NewRequest(http.MethodPost, "/api/v1/libraries/lib_7/ai/tag-suggestion-reviews/clear", strings.NewReader(`{"confirmation":"clear_tag_suggestion_reviews"}`))
	clearRequest.Header.Set("Content-Type", "application/json")
	clearRequest.Header.Set("Idempotency-Key", "tag-clear-key-001")
	clearRequest.Header.Set("If-Match", `"ai-tag-reviews-7-r4"`)
	clear := httptest.NewRecorder()
	mux.ServeHTTP(clear, clearRequest)
	if clear.Code != http.StatusAccepted || stub.library != 7 || stub.revision != 4 || stub.key != "tag-clear-key-001" {
		t.Fatalf("clear=%d stub=%#v headers=%v body=%s", clear.Code, stub, clear.Header(), clear.Body.String())
	}
}

func (stub *aiTagVocabularyHTTPStub) Get(context.Context) (semantic.TagVocabulary, error) {
	return stub.value, nil
}
func (stub *aiTagVocabularyHTTPStub) Publish(_ context.Context, revision int64, ids []int64) (semantic.TagVocabulary, error) {
	stub.expectedRevision, stub.tagIDs = revision, ids
	stub.value.Revision = revision + 1
	return stub.value, nil
}

func TestAITagVocabularyHTTPUsesOpaqueTagsAndStrongRevision(t *testing.T) {
	now := time.Date(2026, 8, 30, 10, 0, 0, 0, time.UTC)
	stub := &aiTagVocabularyHTTPStub{value: semantic.TagVocabulary{ID: "aivocab_initial", Revision: 1,
		Entries: []semantic.TagVocabularyEntry{{TagID: 7, Name: "Family"}}, PublishedAt: now}}
	mux := http.NewServeMux()
	registerAITagVocabularyRoutes(mux, stub)
	get := httptest.NewRecorder()
	mux.ServeHTTP(get, httptest.NewRequest(http.MethodGet, "/api/v1/ai/tag-vocabulary", nil))
	if get.Code != http.StatusOK || get.Header().Get("ETag") != `"ai-tag-vocabulary-r1"` {
		t.Fatalf("get=%d headers=%v body=%s", get.Code, get.Header(), get.Body.String())
	}
	var body aiTagVocabularyResponse
	if err := json.NewDecoder(get.Body).Decode(&body); err != nil || len(body.Entries) != 1 || body.Entries[0].TagID != "tag_7" {
		t.Fatalf("body=%#v err=%v", body, err)
	}
	putRequest := httptest.NewRequest(http.MethodPut, "/api/v1/ai/tag-vocabulary", strings.NewReader(`{"tagIds":["tag_7","tag_9"]}`))
	putRequest.Header.Set("Content-Type", "application/json")
	putRequest.Header.Set("If-Match", `"ai-tag-vocabulary-r1"`)
	put := httptest.NewRecorder()
	mux.ServeHTTP(put, putRequest)
	if put.Code != http.StatusOK || stub.expectedRevision != 1 || len(stub.tagIDs) != 2 || stub.tagIDs[1] != 9 ||
		put.Header().Get("ETag") != `"ai-tag-vocabulary-r2"` {
		t.Fatalf("put=%d revision=%d tags=%v headers=%v body=%s", put.Code, stub.expectedRevision, stub.tagIDs, put.Header(), put.Body.String())
	}
}

func TestAITagVocabularyHTTPRejectsMissingPreconditionAndDuplicateTags(t *testing.T) {
	stub := &aiTagVocabularyHTTPStub{}
	mux := http.NewServeMux()
	registerAITagVocabularyRoutes(mux, stub)
	missing := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/api/v1/ai/tag-vocabulary", strings.NewReader(`{"tagIds":[]}`))
	request.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(missing, request)
	if missing.Code != http.StatusPreconditionRequired {
		t.Fatalf("missing=%d body=%s", missing.Code, missing.Body.String())
	}
	duplicate := httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPut, "/api/v1/ai/tag-vocabulary", strings.NewReader(`{"tagIds":["tag_7","tag_7"]}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("If-Match", `"ai-tag-vocabulary-r1"`)
	mux.ServeHTTP(duplicate, request)
	if duplicate.Code != http.StatusUnprocessableEntity {
		t.Fatalf("duplicate=%d body=%s", duplicate.Code, duplicate.Body.String())
	}
}

type aiTagSuggestionListHTTPStub struct {
	request semantic.TagSuggestionListRequest
	page    semantic.TagSuggestionPage
	err     error
}

func (stub *aiTagSuggestionListHTTPStub) List(_ context.Context, request semantic.TagSuggestionListRequest) (semantic.TagSuggestionPage, error) {
	stub.request = request
	return stub.page, stub.err
}

func TestAITagSuggestionListHTTPHydratesAssetsAndPreservesReviewLineage(t *testing.T) {
	reviewed := time.Date(2026, 8, 30, 11, 0, 0, 0, time.UTC)
	stub := &aiTagSuggestionListHTTPStub{page: semantic.TagSuggestionPage{
		Items: []semantic.TagSuggestionView{{ID: "ais_suggestion_123", LibraryID: 7, AssetID: 9, TagID: 4,
			TagName: "Family", Confidence: .85, Status: semantic.TagSuggestionAccepted,
			GenerationID: "aig_generation123", VocabularyRevision: 3, Revision: 1, ReviewedAt: &reviewed}},
		NextCursor: "opaque-tag-cursor", Coverage: semantic.TagSuggestionCoverage{Eligible: 3, Completed: 1, Revision: 4}, ReviewRevision: 3,
	}}
	catalogStub := catalogServiceStub{getAsset: func(_ context.Context, assetID int64) (catalog.Asset, error) {
		return catalog.Asset{ID: assetID, LibraryID: 7, LibraryName: "Main", DirectoryID: 5,
			Name: "portrait.jpg", RelativePath: "set/portrait.jpg", Kind: catalog.KindImage,
			MIMEType: "image/jpeg", SizeBytes: 10, Availability: catalog.SourceAvailable}, nil
	}}
	mux := http.NewServeMux()
	registerAITagSuggestionListRoute(mux, stub, catalogStub)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet,
		"/api/v1/libraries/lib_7/ai/tag-suggestions?status=accepted&tagId=tag_4&limit=20", nil))
	if response.Code != http.StatusOK || stub.request.LibraryID != 7 || stub.request.Status != semantic.TagSuggestionAccepted ||
		stub.request.TagID != 4 || stub.request.Limit != 20 || response.Header().Get("ETag") != `"ai-tag-reviews-7-r3"` {
		t.Fatalf("response=%d request=%#v body=%s", response.Code, stub.request, response.Body.String())
	}
	var body aiTagSuggestionPageResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil || len(body.Items) != 1 ||
		body.Items[0].Asset.ID != "ast_9" || body.Items[0].Tag.TagID != "tag_4" || body.Items[0].ReviewedAt == nil ||
		body.NextCursor == nil || *body.NextCursor != "opaque-tag-cursor" {
		t.Fatalf("body=%#v err=%v", body, err)
	}
}

func TestAITagSuggestionListHTTPRejectsUnknownQueryAndProjectionDrift(t *testing.T) {
	stub := &aiTagSuggestionListHTTPStub{page: semantic.TagSuggestionPage{Items: []semantic.TagSuggestionView{{
		ID: "ais_suggestion_123", LibraryID: 7, AssetID: 9, TagID: 4, Status: semantic.TagSuggestionPending,
	}}}}
	mux := http.NewServeMux()
	registerAITagSuggestionListRoute(mux, stub, catalogServiceStub{})
	unknown := httptest.NewRecorder()
	mux.ServeHTTP(unknown, httptest.NewRequest(http.MethodGet, "/api/v1/libraries/lib_7/ai/tag-suggestions?score=1", nil))
	if unknown.Code != http.StatusBadRequest {
		t.Fatalf("unknown=%d body=%s", unknown.Code, unknown.Body.String())
	}
	drift := httptest.NewRecorder()
	mux.ServeHTTP(drift, httptest.NewRequest(http.MethodGet, "/api/v1/libraries/lib_7/ai/tag-suggestions", nil))
	if drift.Code != http.StatusConflict || !strings.Contains(drift.Body.String(), `"code":"suggestion_cursor_stale"`) {
		t.Fatalf("drift=%d body=%s", drift.Code, drift.Body.String())
	}
}

type aiTagReviewHTTPStub struct {
	key    string
	items  []semantic.TagReviewItem
	result semantic.IdempotentTagReviewResult
	err    error
}

func (stub *aiTagReviewHTTPStub) Review(_ context.Context, key string, items []semantic.TagReviewItem) (semantic.IdempotentTagReviewResult, error) {
	stub.key, stub.items = key, items
	return stub.result, stub.err
}

func TestAITagReviewHTTPMapsWireActionsAndReplay(t *testing.T) {
	stub := &aiTagReviewHTTPStub{result: semantic.IdempotentTagReviewResult{Replayed: true, Items: []semantic.TagReviewOutcome{
		{SuggestionID: "ais_suggestion_123", Outcome: semantic.TagReviewAccept, Revision: 1},
		{SuggestionID: "ais_suggestion_456", Conflict: true, Revision: 2},
	}}}
	mux := http.NewServeMux()
	registerAITagReviewRoute(mux, stub)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/ai/tag-suggestion-reviews", strings.NewReader(`{"items":[
        {"suggestionId":"ais_suggestion_123","action":"accept","expectedSuggestionRevision":1,"expectedCurationRevision":4},
        {"suggestionId":"ais_suggestion_456","action":"dismiss","expectedSuggestionRevision":2,"expectedCurationRevision":1}]}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "tag-review-key-001")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusOK || response.Header().Get("Idempotency-Replayed") != "true" ||
		stub.key != "tag-review-key-001" || len(stub.items) != 2 || stub.items[0].Action != semantic.TagReviewAccept ||
		stub.items[1].Action != semantic.TagReviewDismiss {
		t.Fatalf("response=%d headers=%v key=%s items=%#v body=%s", response.Code, response.Header(), stub.key, stub.items, response.Body.String())
	}
	var body aiTagReviewResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil || len(body.Items) != 2 ||
		body.Items[0].Outcome != "accepted" || body.Items[1].Outcome != "conflict" || body.Items[1].Revision != 2 {
		t.Fatalf("body=%#v err=%v", body, err)
	}
}

func TestAITagReviewHTTPRequiresIdempotencyKeyAndRejectsUnknownAction(t *testing.T) {
	stub := &aiTagReviewHTTPStub{err: semantic.ErrInvalidTagSuggestion}
	mux := http.NewServeMux()
	registerAITagReviewRoute(mux, stub)
	missing := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/ai/tag-suggestion-reviews", strings.NewReader(`{"items":[]}`))
	request.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(missing, request)
	if missing.Code != http.StatusBadRequest {
		t.Fatalf("missing=%d body=%s", missing.Code, missing.Body.String())
	}
	invalid := httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, "/api/v1/ai/tag-suggestion-reviews", strings.NewReader(`{"items":[
        {"suggestionId":"ais_suggestion_123","action":"invent","expectedSuggestionRevision":1,"expectedCurationRevision":1}]}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "tag-review-key-002")
	mux.ServeHTTP(invalid, request)
	if invalid.Code != http.StatusUnprocessableEntity {
		t.Fatalf("invalid=%d body=%s", invalid.Code, invalid.Body.String())
	}
}
