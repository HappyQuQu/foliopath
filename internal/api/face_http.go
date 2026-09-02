package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
	"github.com/HappyQuQu/foliopath/internal/catalog"
	"github.com/HappyQuQu/foliopath/internal/face"
)

type FaceSettingsService interface {
	Get(context.Context, int64) (face.LibrarySettings, error)
	Update(context.Context, int64, bool, int64) (face.LibrarySettings, error)
}

type FaceJobAdmissionService interface {
	RequestFaceJob(context.Context, int64, face.JobMode, string) (aimodel.Operation, bool, error)
}

type FaceClearAdmissionService interface {
	RequestDerivedFaceClear(context.Context, int64, int64, string) (aimodel.Operation, bool, error)
	RequestManualFaceClear(context.Context, int64, int64, string, face.ManualClearCounts) (aimodel.Operation, bool, error)
}

type FaceClusterService interface {
	List(context.Context, face.FaceClusterListRequest) (face.FaceClusterPage, error)
}

type PeopleListService interface {
	List(context.Context, face.PeopleListRequest) (face.PeoplePage, error)
}

type PersonQueryService interface {
	GetPerson(context.Context, string) (face.Person, error)
}

type AssetFacesQueryService interface {
	List(context.Context, int64) ([]face.AssetFaceView, error)
}

type PersonAssetsQueryService interface {
	List(context.Context, face.PersonAssetRequest) (face.PersonAssetPage, error)
}

type FaceClusterDetailQueryService interface {
	List(context.Context, face.FaceClusterDetailRequest) (face.FaceClusterDetailPage, error)
}

type FaceReviewMutationService interface {
	Review(context.Context, string, face.ReviewRequest) (face.ReviewResult, error)
	Undo(context.Context, string, string, int64) (face.ReviewResult, error)
}

type FacePersonMutationService interface {
	Create(context.Context, string, string, string, *int64) (face.Person, bool, error)
	Rename(context.Context, string, string, int64) (face.Person, error)
	Delete(context.Context, string, int64) error
}

type faceSettingsWireResponse struct {
	LibraryID          string           `json:"libraryId"`
	Enabled            bool             `json:"enabled"`
	State              string           `json:"state"`
	Revision           int64            `json:"revision"`
	ActiveGenerationID *string          `json:"activeGenerationId"`
	Coverage           faceCoverageJSON `json:"coverage"`
}

type faceCoverageJSON struct {
	Eligible  int64 `json:"eligible"`
	Completed int64 `json:"completed"`
	Degraded  int64 `json:"degraded"`
	Failed    int64 `json:"failed"`
	Stale     int64 `json:"stale"`
	Complete  bool  `json:"complete"`
	Revision  int64 `json:"revision"`
}

type faceClusterJSON struct {
	ID              string   `json:"id"`
	LibraryID       string   `json:"libraryId"`
	Kind            string   `json:"kind"`
	MemberCount     int64    `json:"memberCount"`
	PreviewAssetIDs []string `json:"previewAssetIds"`
	Revision        int64    `json:"revision"`
}

type faceClusterPageJSON struct {
	Items                  []faceClusterJSON `json:"items"`
	NextCursor             *string           `json:"nextCursor"`
	Coverage               faceCoverageJSON  `json:"coverage"`
	GroupAssignmentAllowed bool              `json:"groupAssignmentAllowed"`
}

type personJSON struct {
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	ConfirmedFaceCount int64     `json:"confirmedFaceCount"`
	AssetCount         int64     `json:"assetCount"`
	Revision           int64     `json:"revision"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

type peoplePageJSON struct {
	Items      []personJSON `json:"items"`
	NextCursor *string      `json:"nextCursor"`
}

type faceRegionJSON struct {
	XPercent      int `json:"xPercent"`
	YPercent      int `json:"yPercent"`
	WidthPercent  int `json:"widthPercent"`
	HeightPercent int `json:"heightPercent"`
}

type assetFaceJSON struct {
	FaceID   string         `json:"faceId"`
	AssetID  string         `json:"assetId"`
	Ordinal  int            `json:"ordinal"`
	Region   faceRegionJSON `json:"region"`
	State    string         `json:"state"`
	PersonID *string        `json:"personId"`
	Revision int64          `json:"revision"`
}

type assetFacePageJSON struct {
	Items []assetFaceJSON `json:"items"`
}

type faceClusterMemberJSON struct {
	FaceID   string         `json:"faceId"`
	AssetID  string         `json:"assetId"`
	Kind     string         `json:"kind"`
	Region   faceRegionJSON `json:"region"`
	Revision int64          `json:"revision"`
}

type faceClusterDetailPageJSON struct {
	Cluster    faceClusterJSON         `json:"cluster"`
	Items      []faceClusterMemberJSON `json:"items"`
	NextCursor *string                 `json:"nextCursor"`
}

type personAssetJSON struct {
	Asset   assetResponse `json:"asset"`
	FaceIDs []string      `json:"faceIds"`
}

type personAssetPageJSON struct {
	Items      []personAssetJSON `json:"items"`
	NextCursor *string           `json:"nextCursor"`
}

type faceReviewRequestJSON struct {
	Action                  string `json:"action"`
	FaceID                  string `json:"faceId"`
	PersonID                string `json:"personId"`
	ClusterID               string `json:"clusterId"`
	LeftFaceID              string `json:"leftFaceId"`
	RightFaceID             string `json:"rightFaceId"`
	SourcePersonID          string `json:"sourcePersonId"`
	TargetPersonID          string `json:"targetPersonId"`
	ExpectedFaceRevision    int64  `json:"expectedFaceRevision"`
	ExpectedPersonRevision  int64  `json:"expectedPersonRevision"`
	ExpectedClusterRevision int64  `json:"expectedClusterRevision"`
	ExpectedLeftRevision    int64  `json:"expectedLeftRevision"`
	ExpectedRightRevision   int64  `json:"expectedRightRevision"`
	ExpectedSourceRevision  int64  `json:"expectedSourceRevision"`
	ExpectedTargetRevision  int64  `json:"expectedTargetRevision"`
	ConflictsAcknowledged   bool   `json:"conflictsAcknowledged"`
}

type faceReviewResultJSON struct {
	ReviewID          string   `json:"reviewId"`
	Action            string   `json:"action"`
	AffectedPersonIDs []string `json:"affectedPersonIds"`
	Revision          int64    `json:"revision"`
	Undoable          bool     `json:"undoable"`
}

type createPersonRequestJSON struct {
	Name                    string `json:"name"`
	SourceClusterID         string `json:"sourceClusterId"`
	ExpectedClusterRevision *int64 `json:"expectedClusterRevision"`
}

type renamePersonRequestJSON struct {
	Name string `json:"name"`
}

type faceSettingsUpdateJSON struct {
	Enabled *bool `json:"enabled"`
}

type faceJobRequestJSON struct {
	Mode string `json:"mode"`
}

type faceClearJSON struct {
	Confirmation            string `json:"confirmation"`
	ExpectedPersonCount     *int64 `json:"expectedPersonCount"`
	ExpectedAssignmentCount *int64 `json:"expectedAssignmentCount"`
	ExpectedConstraintCount *int64 `json:"expectedConstraintCount"`
}

// registerFaceReadRoutes is intentionally kept separate from production
// composition while the release gate remains No-Go. Adapter tests can exercise
// the accepted wire contract without making face data reachable at runtime.
func registerFaceReadRoutes(mux *http.ServeMux, settings FaceSettingsService, clusters FaceClusterService, people PeopleListService, persons PersonQueryService, assetFaces AssetFacesQueryService, personAssets PersonAssetsQueryService, clusterDetails FaceClusterDetailQueryService, catalogService CatalogService) {
	mux.HandleFunc("GET /api/v1/libraries/{libraryId}/ai/faces", func(w http.ResponseWriter, r *http.Request) {
		id, err := parseResourceID(r.PathValue("libraryId"), "lib_")
		if err != nil {
			writeFaceError(w, r, face.ErrFaceLibraryNotFound)
			return
		}
		value, err := settings.Get(r.Context(), id)
		if err != nil {
			writeFaceError(w, r, err)
			return
		}
		w.Header().Set("ETag", faceSettingsETag(id, value.Revision))
		writeJSON(w, http.StatusOK, faceSettingsResponse(value))
	})
	mux.HandleFunc("GET /api/v1/libraries/{libraryId}/ai/face-clusters", func(w http.ResponseWriter, r *http.Request) {
		id, err := parseResourceID(r.PathValue("libraryId"), "lib_")
		role, cursor, limit, queryErr := parseFaceClusterQuery(r.URL.Query())
		if err != nil || queryErr != nil {
			writeFaceError(w, r, face.ErrInvalidClusterRecord)
			return
		}
		page, err := clusters.List(r.Context(), face.FaceClusterListRequest{LibraryID: id, Role: role, Cursor: cursor, Limit: limit})
		if err != nil {
			writeFaceError(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, faceClusterPageResponse(page))
	})
	mux.HandleFunc("GET /api/v1/people", func(w http.ResponseWriter, r *http.Request) {
		query, cursor, limit, err := parsePeopleQuery(r.URL.Query())
		if err != nil {
			writeFaceError(w, r, err)
			return
		}
		page, err := people.List(r.Context(), face.PeopleListRequest{Search: query, Cursor: cursor, Limit: limit})
		if err != nil {
			writeFaceError(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, peoplePageResponse(page))
	})
	mux.HandleFunc("GET /api/v1/people/{personId}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("personId")
		value, err := persons.GetPerson(r.Context(), id)
		if err != nil {
			writeFaceError(w, r, err)
			return
		}
		w.Header().Set("ETag", fmt.Sprintf(`"person-%s-r%d"`, value.ID, value.Revision))
		writeJSON(w, http.StatusOK, personResponse(value))
	})
	mux.HandleFunc("GET /api/v1/assets/{assetId}/faces", func(w http.ResponseWriter, r *http.Request) {
		id, err := parseResourceID(r.PathValue("assetId"), "ast_")
		if err != nil {
			writeFaceError(w, r, face.ErrInvalidFaceProjection)
			return
		}
		items, err := assetFaces.List(r.Context(), id)
		if err != nil {
			writeFaceError(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, assetFacePageResponse(items))
	})
	mux.HandleFunc("GET /api/v1/people/{personId}/assets", func(w http.ResponseWriter, r *http.Request) {
		cursor, limit, err := parseFacePageQuery(r.URL.Query(), face.MaxPersonAssetPageSize)
		if err != nil {
			writeFaceError(w, r, err)
			return
		}
		page, err := personAssets.List(r.Context(), face.PersonAssetRequest{PersonID: r.PathValue("personId"), Cursor: cursor, Limit: limit})
		if err != nil {
			writeFaceError(w, r, err)
			return
		}
		assets, err := hydratePersonAssets(r.Context(), catalogService, page.Items)
		if err != nil {
			writeFaceError(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, personAssetPageResponse(page, assets))
	})
	mux.HandleFunc("GET /api/v1/libraries/{libraryId}/ai/face-clusters/{clusterId}", func(w http.ResponseWriter, r *http.Request) {
		id, idErr := parseResourceID(r.PathValue("libraryId"), "lib_")
		cursor, limit, queryErr := parseFacePageQuery(r.URL.Query(), face.MaxFaceClusterMemberPageSize)
		if idErr != nil || queryErr != nil {
			writeFaceError(w, r, face.ErrInvalidFaceProjection)
			return
		}
		page, err := clusterDetails.List(r.Context(), face.FaceClusterDetailRequest{LibraryID: id, ClusterID: r.PathValue("clusterId"), Cursor: cursor, Limit: limit})
		if err != nil {
			writeFaceError(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, faceClusterDetailPageResponse(page))
	})
}

func registerFaceControlRoutes(mux *http.ServeMux, settings FaceSettingsService, jobs FaceJobAdmissionService, clears FaceClearAdmissionService) {
	mux.HandleFunc("PATCH /api/v1/libraries/{libraryId}/ai/faces", func(w http.ResponseWriter, r *http.Request) {
		libraryID, err := parseResourceID(r.PathValue("libraryId"), "lib_")
		if err != nil {
			writeFaceError(w, r, face.ErrFaceLibraryNotFound)
			return
		}
		revision, err := parseAIRevisionETag(r.Header.Get("If-Match"), fmt.Sprintf("face-library-%d", libraryID))
		if err != nil {
			writeAIModelError(w, r, err)
			return
		}
		var payload faceSettingsUpdateJSON
		if decodeAIModelJSON(r, &payload) != nil || payload.Enabled == nil {
			writeFaceError(w, r, face.ErrInvalidFaceJob)
			return
		}
		value, err := settings.Update(r.Context(), libraryID, *payload.Enabled, revision)
		if err != nil {
			writeFaceError(w, r, err)
			return
		}
		w.Header().Set("ETag", faceSettingsETag(value.LibraryID, value.Revision))
		writeJSON(w, http.StatusOK, faceSettingsResponse(value))
	})
	mux.HandleFunc("POST /api/v1/libraries/{libraryId}/ai/faces/jobs", func(w http.ResponseWriter, r *http.Request) {
		libraryID, err := parseResourceID(r.PathValue("libraryId"), "lib_")
		key := r.Header.Get("Idempotency-Key")
		var payload faceJobRequestJSON
		if err != nil || !idempotencyKeyPattern.MatchString(key) || decodeAIModelJSON(r, &payload) != nil ||
			(payload.Mode != "missing" && payload.Mode != "all") {
			writeFaceError(w, r, face.ErrInvalidFaceJob)
			return
		}
		operation, replayed, err := jobs.RequestFaceJob(r.Context(), libraryID, face.JobMode(payload.Mode), key)
		if err != nil {
			writeFaceError(w, r, err)
			return
		}
		writeFaceOperation(w, operation, replayed)
	})
	mux.HandleFunc("POST /api/v1/libraries/{libraryId}/ai/faces/derived-clear", func(w http.ResponseWriter, r *http.Request) {
		libraryID, revision, key, err := parseFaceClearHeaders(r)
		var payload faceClearJSON
		if err != nil || decodeAIModelJSON(r, &payload) != nil || payload.Confirmation != "clear_derived_face_data" ||
			payload.ExpectedPersonCount != nil || payload.ExpectedAssignmentCount != nil || payload.ExpectedConstraintCount != nil {
			writeFaceControlInputError(w, r, err)
			return
		}
		operation, replayed, err := clears.RequestDerivedFaceClear(r.Context(), libraryID, revision, key)
		if err != nil {
			writeFaceError(w, r, err)
			return
		}
		writeFaceOperation(w, operation, replayed)
	})
	mux.HandleFunc("POST /api/v1/libraries/{libraryId}/ai/faces/manual-clear", func(w http.ResponseWriter, r *http.Request) {
		libraryID, revision, key, err := parseFaceClearHeaders(r)
		var payload faceClearJSON
		if err != nil || decodeAIModelJSON(r, &payload) != nil || payload.Confirmation != "clear_manual_face_relationships" ||
			payload.ExpectedPersonCount == nil || payload.ExpectedAssignmentCount == nil || payload.ExpectedConstraintCount == nil ||
			*payload.ExpectedPersonCount < 0 || *payload.ExpectedAssignmentCount < 0 || *payload.ExpectedConstraintCount < 0 {
			writeFaceControlInputError(w, r, err)
			return
		}
		counts := face.ManualClearCounts{People: *payload.ExpectedPersonCount, Assignments: *payload.ExpectedAssignmentCount, Constraints: *payload.ExpectedConstraintCount}
		operation, replayed, err := clears.RequestManualFaceClear(r.Context(), libraryID, revision, key, counts)
		if err != nil {
			writeFaceError(w, r, err)
			return
		}
		writeFaceOperation(w, operation, replayed)
	})
}

func parseFaceClearHeaders(r *http.Request) (int64, int64, string, error) {
	libraryID, err := parseResourceID(r.PathValue("libraryId"), "lib_")
	if err != nil {
		return 0, 0, "", face.ErrFaceLibraryNotFound
	}
	key := r.Header.Get("Idempotency-Key")
	if !idempotencyKeyPattern.MatchString(key) {
		return 0, 0, "", face.ErrInvalidFaceClear
	}
	revision, err := parseAIRevisionETag(r.Header.Get("If-Match"), fmt.Sprintf("face-library-%d", libraryID))
	return libraryID, revision, key, err
}

func writeFaceControlInputError(w http.ResponseWriter, r *http.Request, err error) {
	if err != nil {
		if errors.Is(err, face.ErrFaceLibraryNotFound) || errors.Is(err, face.ErrInvalidFaceClear) {
			writeFaceError(w, r, err)
		} else {
			writeAIModelError(w, r, err)
		}
		return
	}
	writeFaceError(w, r, face.ErrInvalidFaceClear)
}

func writeFaceOperation(w http.ResponseWriter, operation aimodel.Operation, replayed bool) {
	w.Header().Set("Location", "/api/v1/ai/operations/"+operation.ID)
	w.Header().Set("Idempotency-Replayed", strconv.FormatBool(replayed))
	w.Header().Set("ETag", aiOperationETag(operation))
	writeJSON(w, http.StatusAccepted, aiOperationWire(operation))
}

// registerFaceMutationRoutes remains outside production composition until the
// S2C release gate accepts the final model, quality, privacy, and native evidence.
func registerFaceMutationRoutes(mux *http.ServeMux, reviews FaceReviewMutationService) {
	mux.HandleFunc("POST /api/v1/ai/face-reviews", func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("Idempotency-Key")
		var payload faceReviewRequestJSON
		if !idempotencyKeyPattern.MatchString(key) || decodeAIModelJSON(r, &payload) != nil {
			writeFaceError(w, r, face.ErrInvalidReview)
			return
		}
		result, err := reviews.Review(r.Context(), key, face.ReviewRequest{
			Action: payload.Action, FaceID: payload.FaceID, PersonID: payload.PersonID, ClusterID: payload.ClusterID,
			LeftFaceID: payload.LeftFaceID, RightFaceID: payload.RightFaceID, SourcePersonID: payload.SourcePersonID,
			TargetPersonID: payload.TargetPersonID, ExpectedFaceRevision: payload.ExpectedFaceRevision,
			ExpectedPersonRevision: payload.ExpectedPersonRevision, ExpectedClusterRevision: payload.ExpectedClusterRevision,
			ExpectedLeftRevision: payload.ExpectedLeftRevision, ExpectedRightRevision: payload.ExpectedRightRevision,
			ExpectedSourceRevision: payload.ExpectedSourceRevision, ExpectedTargetRevision: payload.ExpectedTargetRevision,
			ConflictsAcknowledged: payload.ConflictsAcknowledged,
		})
		if err != nil {
			writeFaceError(w, r, err)
			return
		}
		w.Header().Set("Idempotency-Replayed", strconv.FormatBool(result.Replayed))
		w.Header().Set("ETag", fmt.Sprintf(`"face-review-%s-r%d"`, result.EventID, result.Revision))
		writeJSON(w, http.StatusOK, faceReviewResponse(result))
	})
	mux.HandleFunc("POST /api/v1/ai/face-reviews/{reviewId}/undo", func(w http.ResponseWriter, r *http.Request) {
		key, reviewID := r.Header.Get("Idempotency-Key"), r.PathValue("reviewId")
		revision, err := parseAIRevisionETag(r.Header.Get("If-Match"), "face-review-"+reviewID)
		if err != nil || !idempotencyKeyPattern.MatchString(key) {
			writeFaceError(w, r, face.ErrInvalidReview)
			return
		}
		result, err := reviews.Undo(r.Context(), key, reviewID, revision)
		if err != nil {
			writeFaceError(w, r, err)
			return
		}
		w.Header().Set("Idempotency-Replayed", strconv.FormatBool(result.Replayed))
		w.Header().Set("ETag", fmt.Sprintf(`"face-review-%s-r%d"`, result.EventID, result.Revision))
		writeJSON(w, http.StatusOK, faceReviewResponse(result))
	})
}

func registerFacePersonMutationRoutes(mux *http.ServeMux, persons FacePersonMutationService) {
	mux.HandleFunc("POST /api/v1/people", func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("Idempotency-Key")
		var payload createPersonRequestJSON
		if !idempotencyKeyPattern.MatchString(key) || decodeAIModelJSON(r, &payload) != nil {
			writeFaceError(w, r, face.ErrInvalidPerson)
			return
		}
		person, replayed, err := persons.Create(r.Context(), key, payload.Name, payload.SourceClusterID, payload.ExpectedClusterRevision)
		if err != nil {
			writeFaceError(w, r, err)
			return
		}
		w.Header().Set("Location", "/api/v1/people/"+person.ID)
		w.Header().Set("Idempotency-Replayed", strconv.FormatBool(replayed))
		w.Header().Set("ETag", fmt.Sprintf(`"person-%s-r%d"`, person.ID, person.Revision))
		writeJSON(w, http.StatusCreated, personResponse(person))
	})
	mux.HandleFunc("PATCH /api/v1/people/{personId}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("personId")
		revision, err := parseAIRevisionETag(r.Header.Get("If-Match"), "person-"+id)
		var payload renamePersonRequestJSON
		if err != nil || decodeAIModelJSON(r, &payload) != nil {
			writeFaceError(w, r, face.ErrInvalidPerson)
			return
		}
		person, err := persons.Rename(r.Context(), id, payload.Name, revision)
		if err != nil {
			writeFaceError(w, r, err)
			return
		}
		w.Header().Set("ETag", fmt.Sprintf(`"person-%s-r%d"`, person.ID, person.Revision))
		writeJSON(w, http.StatusOK, personResponse(person))
	})
	mux.HandleFunc("DELETE /api/v1/people/{personId}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("personId")
		revision, err := parseAIRevisionETag(r.Header.Get("If-Match"), "person-"+id)
		if err != nil {
			writeFaceError(w, r, face.ErrInvalidPerson)
			return
		}
		if err := persons.Delete(r.Context(), id, revision); err != nil {
			writeFaceError(w, r, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

func parseFacePageQuery(values url.Values, maximum int) (string, int, error) {
	for key, entries := range values {
		if (key != "cursor" && key != "limit") || len(entries) != 1 {
			return "", 0, face.ErrInvalidFaceProjection
		}
	}
	cursor, limit := values.Get("cursor"), 50
	if raw := values.Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			return "", 0, face.ErrInvalidFaceProjection
		}
		limit = parsed
	}
	if limit < 1 || limit > maximum || len(cursor) > 4096 {
		return "", 0, face.ErrInvalidFaceProjection
	}
	return cursor, limit, nil
}

func parseFaceClusterQuery(values url.Values) (string, string, int, error) {
	for key, entries := range values {
		if (key != "kind" && key != "cursor" && key != "limit") || len(entries) != 1 {
			return "", "", 0, face.ErrInvalidClusterRecord
		}
	}
	role, cursor, limit := values.Get("kind"), values.Get("cursor"), 50
	if role != "" && role != "core" && role != "edge" {
		return "", "", 0, face.ErrInvalidClusterRecord
	}
	if raw := values.Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			return "", "", 0, err
		}
		limit = parsed
	}
	if limit < 1 || limit > face.MaxFaceClusterPageSize || len(cursor) > 4096 {
		return "", "", 0, face.ErrInvalidClusterRecord
	}
	return role, cursor, limit, nil
}

func parsePeopleQuery(values url.Values) (string, string, int, error) {
	for key, entries := range values {
		if (key != "query" && key != "cursor" && key != "limit") || len(entries) != 1 {
			return "", "", 0, face.ErrInvalidPerson
		}
	}
	query, cursor, limit := values.Get("query"), values.Get("cursor"), 50
	if raw := values.Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			return "", "", 0, err
		}
		limit = parsed
	}
	if limit < 1 || limit > face.MaxPeoplePageSize || len(cursor) > 4096 {
		return "", "", 0, face.ErrInvalidPerson
	}
	return query, cursor, limit, nil
}

func faceSettingsResponse(value face.LibrarySettings) faceSettingsWireResponse {
	var generation *string
	if value.ActiveGenerationID != "" {
		generation = &value.ActiveGenerationID
	}
	return faceSettingsWireResponse{LibraryID: libraryID(value.LibraryID), Enabled: value.Enabled, State: value.State, Revision: value.Revision, ActiveGenerationID: generation, Coverage: faceCoverageResponse(value.Coverage)}
}
func faceCoverageResponse(value face.FaceCoverage) faceCoverageJSON {
	return faceCoverageJSON{Eligible: value.Eligible, Completed: value.Completed, Degraded: 0, Failed: value.Failed, Stale: value.Stale, Complete: value.Complete(), Revision: value.Revision}
}
func faceClusterPageResponse(page face.FaceClusterPage) faceClusterPageJSON {
	items := make([]faceClusterJSON, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, faceClusterResponse(item))
	}
	var next *string
	if page.NextCursor != "" {
		next = &page.NextCursor
	}
	return faceClusterPageJSON{Items: items, NextCursor: next, Coverage: faceCoverageResponse(page.Coverage), GroupAssignmentAllowed: page.GroupAssignmentAllowed}
}
func faceClusterResponse(item face.FaceClusterView) faceClusterJSON {
	ids := make([]string, len(item.PreviewAssetIDs))
	for i, id := range item.PreviewAssetIDs {
		ids[i] = assetID(id)
	}
	return faceClusterJSON{ID: item.ID, LibraryID: libraryID(item.LibraryID), Kind: item.Role, MemberCount: item.MemberCount, PreviewAssetIDs: ids, Revision: item.Revision}
}
func personResponse(value face.Person) personJSON {
	return personJSON{ID: value.ID, Name: value.Name, ConfirmedFaceCount: value.ConfirmedFaceCount, AssetCount: value.AssetCount, Revision: value.Revision, CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt}
}
func peoplePageResponse(page face.PeoplePage) peoplePageJSON {
	items := make([]personJSON, 0, len(page.Items))
	for _, value := range page.Items {
		items = append(items, personResponse(value))
	}
	var next *string
	if page.NextCursor != "" {
		next = &page.NextCursor
	}
	return peoplePageJSON{Items: items, NextCursor: next}
}

func faceRegionResponse(value face.CoarseRegion) faceRegionJSON {
	return faceRegionJSON{XPercent: value.XPercent, YPercent: value.YPercent, WidthPercent: value.WidthPercent, HeightPercent: value.HeightPercent}
}

func assetFacePageResponse(values []face.AssetFaceView) assetFacePageJSON {
	items := make([]assetFaceJSON, 0, len(values))
	for _, value := range values {
		var person *string
		if value.PersonID != "" {
			id := value.PersonID
			person = &id
		}
		items = append(items, assetFaceJSON{FaceID: value.FaceID, AssetID: assetID(value.AssetID), Ordinal: value.Ordinal, Region: faceRegionResponse(value.Region), State: value.State, PersonID: person, Revision: value.Revision})
	}
	return assetFacePageJSON{Items: items}
}

func faceClusterDetailPageResponse(page face.FaceClusterDetailPage) faceClusterDetailPageJSON {
	items := make([]faceClusterMemberJSON, 0, len(page.Items))
	for _, value := range page.Items {
		items = append(items, faceClusterMemberJSON{FaceID: value.FaceID, AssetID: assetID(value.AssetID), Kind: value.Role, Region: faceRegionResponse(value.Region), Revision: value.Revision})
	}
	var next *string
	if page.NextCursor != "" {
		next = &page.NextCursor
	}
	return faceClusterDetailPageJSON{Cluster: faceClusterResponse(page.Cluster), Items: items, NextCursor: next}
}

func hydratePersonAssets(ctx context.Context, service CatalogService, values []face.PersonAssetView) ([]catalog.Asset, error) {
	ids := make([]int64, len(values))
	for index, value := range values {
		ids[index] = value.AssetID
	}
	if len(ids) == 0 {
		return []catalog.Asset{}, nil
	}
	assets, err := service.GetAssetsByIDs(ctx, ids)
	if err != nil {
		if errors.Is(err, catalog.ErrAssetNotFound) {
			return nil, face.ErrFaceProjectionStale
		}
		return nil, err
	}
	if len(assets) != len(values) {
		return nil, face.ErrFaceProjectionStale
	}
	ordered := make([]catalog.Asset, len(values))
	for index, value := range values {
		item := assets[index]
		if item.ID != value.AssetID || item.LibraryID != value.LibraryID {
			return nil, face.ErrFaceProjectionStale
		}
		if item.Availability != catalog.SourceAvailable {
			return nil, face.ErrFaceNotReady
		}
		if item.Kind != catalog.KindImage && item.Kind != catalog.KindAnimated {
			return nil, face.ErrInvalidFaceProjection
		}
		ordered[index] = item
	}
	return ordered, nil
}

func personAssetPageResponse(page face.PersonAssetPage, assets []catalog.Asset) personAssetPageJSON {
	items := make([]personAssetJSON, 0, len(page.Items))
	for index, value := range page.Items {
		items = append(items, personAssetJSON{Asset: assetWire(assets[index]), FaceIDs: append([]string(nil), value.FaceIDs...)})
	}
	var next *string
	if page.NextCursor != "" {
		next = &page.NextCursor
	}
	return personAssetPageJSON{Items: items, NextCursor: next}
}

func faceReviewResponse(value face.ReviewResult) faceReviewResultJSON {
	return faceReviewResultJSON{ReviewID: value.EventID, Action: value.Action, AffectedPersonIDs: append([]string(nil), value.AffectedPersonIDs...), Revision: value.Revision, Undoable: value.Undoable}
}

func writeFaceError(w http.ResponseWriter, r *http.Request, err error) {
	status, code, message := http.StatusInternalServerError, codeInternalError, "The request could not be completed."
	switch {
	case errors.Is(err, face.ErrInvalidPerson), errors.Is(err, face.ErrInvalidReview), errors.Is(err, face.ErrInvalidClusterRecord), errors.Is(err, face.ErrInvalidFaceProjection), errors.Is(err, face.ErrInvalidFaceClusterCursor), errors.Is(err, face.ErrInvalidPeopleCursor), errors.Is(err, face.ErrInvalidFaceJob), errors.Is(err, face.ErrInvalidFaceClear):
		status, code, message = http.StatusBadRequest, codeInvalidRequest, "The request is invalid."
	case errors.Is(err, face.ErrFaceLibraryNotFound):
		status, code, message = http.StatusNotFound, "library_not_found", "The library was not found."
	case errors.Is(err, face.ErrFaceLibraryOffline):
		status, code, message = http.StatusConflict, "library_offline", "The library is offline."
	case errors.Is(err, face.ErrPersonNotFound):
		status, code, message = http.StatusNotFound, "person_not_found", "The person was not found."
	case errors.Is(err, face.ErrPersonConflict), errors.Is(err, face.ErrFaceSettingsConflict):
		status, code, message = http.StatusPreconditionFailed, "precondition_failed", "The resource changed before this request was applied."
	case errors.Is(err, face.ErrReviewConflict):
		status, code, message = http.StatusConflict, "face_constraint_conflict", "The face review conflicts with current state."
	case errors.Is(err, face.ErrFaceJobConflict), errors.Is(err, face.ErrFaceClearConflict):
		status, code, message = http.StatusConflict, "idempotency_conflict", "The request conflicts with an existing operation."
	case errors.Is(err, face.ErrFaceClearCountConflict):
		status, code, message = http.StatusUnprocessableEntity, "validation_failed", "The confirmed impact counts changed."
	case errors.Is(err, face.ErrFaceModelUnavailable):
		status, code, message = http.StatusConflict, "face_model_unavailable", "The face model is unavailable."
	case errors.Is(err, face.ErrFaceDisabled):
		status, code, message = http.StatusConflict, "face_disabled", "Face analysis is disabled."
	case errors.Is(err, face.ErrFaceNotReady), errors.Is(err, face.ErrFaceProjectionStale), errors.Is(err, face.ErrFaceClusterCursorStale), errors.Is(err, face.ErrPeopleCursorStale):
		status, code, message = http.StatusConflict, "face_not_ready", "Face analysis is not ready."
	}
	writePublicError(w, r, status, code, message)
}

func faceSettingsETag(libraryID, revision int64) string {
	return fmt.Sprintf(`"face-library-%d-r%d"`, libraryID, revision)
}
