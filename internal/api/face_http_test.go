package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
	"github.com/HappyQuQu/foliopath/internal/catalog"
	"github.com/HappyQuQu/foliopath/internal/face"
)

type faceHTTPSettingsStub struct{ value face.LibrarySettings }

func (s faceHTTPSettingsStub) Get(context.Context, int64) (face.LibrarySettings, error) {
	return s.value, nil
}

func (s faceHTTPSettingsStub) Update(_ context.Context, libraryID int64, enabled bool, revision int64) (face.LibrarySettings, error) {
	value := s.value
	value.LibraryID, value.Enabled, value.Revision = libraryID, enabled, revision+1
	return value, nil
}

type faceHTTPControlStub struct {
	mode          face.JobMode
	clearKind     face.ClearKind
	clearCounts   face.ManualClearCounts
	clearRevision int64
}

func (stub *faceHTTPControlStub) RequestFaceJob(_ context.Context, libraryID int64, mode face.JobMode, _ string) (aimodel.Operation, bool, error) {
	stub.mode = mode
	total := int64(3)
	now := time.Unix(1, 0).UTC()
	return aimodel.Operation{ID: "aio_face_http_job", Kind: aimodel.OperationFaceMissing, State: aimodel.OperationQueued, Phase: aimodel.PhaseQueued, LibraryID: libraryID, TotalItems: &total, Revision: 1, CreatedAt: now, UpdatedAt: now}, false, nil
}

func (stub *faceHTTPControlStub) RequestDerivedFaceClear(_ context.Context, libraryID, revision int64, _ string) (aimodel.Operation, bool, error) {
	stub.clearKind, stub.clearRevision = face.ClearDerived, revision
	return faceHTTPClearOperation(libraryID, aimodel.OperationFaceDerivedClear), true, nil
}

func (stub *faceHTTPControlStub) RequestManualFaceClear(_ context.Context, libraryID, revision int64, _ string, counts face.ManualClearCounts) (aimodel.Operation, bool, error) {
	stub.clearKind, stub.clearRevision, stub.clearCounts = face.ClearManual, revision, counts
	return faceHTTPClearOperation(libraryID, aimodel.OperationFaceManualClear), false, nil
}

func faceHTTPClearOperation(libraryID int64, kind aimodel.OperationKind) aimodel.Operation {
	now := time.Unix(1, 0).UTC()
	return aimodel.Operation{ID: "aio_face_http_clear", Kind: kind, State: aimodel.OperationQueued, Phase: aimodel.PhaseQueued, LibraryID: libraryID, Revision: 1, CreatedAt: now, UpdatedAt: now}
}

type faceHTTPClustersStub struct{ value face.FaceClusterPage }

func (s faceHTTPClustersStub) List(context.Context, face.FaceClusterListRequest) (face.FaceClusterPage, error) {
	return s.value, nil
}

type faceHTTPPeopleStub struct{ value face.PeoplePage }

func (s faceHTTPPeopleStub) List(context.Context, face.PeopleListRequest) (face.PeoplePage, error) {
	return s.value, nil
}

type faceHTTPPersonStub struct{ value face.Person }

func (s faceHTTPPersonStub) GetPerson(context.Context, string) (face.Person, error) {
	return s.value, nil
}

type faceHTTPAssetFacesStub struct{ value []face.AssetFaceView }

func (s faceHTTPAssetFacesStub) List(context.Context, int64) ([]face.AssetFaceView, error) {
	return s.value, nil
}

type faceHTTPPersonAssetsStub struct{ value face.PersonAssetPage }

func (s faceHTTPPersonAssetsStub) List(context.Context, face.PersonAssetRequest) (face.PersonAssetPage, error) {
	return s.value, nil
}

type faceHTTPPersonAssetsErrorStub struct{ err error }

func (s faceHTTPPersonAssetsErrorStub) List(context.Context, face.PersonAssetRequest) (face.PersonAssetPage, error) {
	return face.PersonAssetPage{}, s.err
}

type faceHTTPClusterDetailStub struct{ value face.FaceClusterDetailPage }

func (s faceHTTPClusterDetailStub) List(context.Context, face.FaceClusterDetailRequest) (face.FaceClusterDetailPage, error) {
	return s.value, nil
}

type faceHTTPReviewStub struct {
	reviewRequest face.ReviewRequest
	undoID        string
	undoRevision  int64
}

type faceHTTPPersonMutationStub struct {
	createdName, createdCluster string
	createdRevision             *int64
	renamedName                 string
	renamedRevision             int64
	deletedID                   string
	deletedRevision             int64
}

func (stub *faceHTTPPersonMutationStub) Create(_ context.Context, _ string, name, cluster string, revision *int64) (face.Person, bool, error) {
	stub.createdName, stub.createdCluster, stub.createdRevision = name, cluster, revision
	return face.Person{ID: "person_http_created", Name: name, Revision: 1, CreatedAt: time.Unix(1, 0).UTC(), UpdatedAt: time.Unix(1, 0).UTC()}, false, nil
}
func (stub *faceHTTPPersonMutationStub) Rename(_ context.Context, id, name string, revision int64) (face.Person, error) {
	stub.renamedName, stub.renamedRevision = name, revision
	return face.Person{ID: id, Name: name, Revision: revision + 1, CreatedAt: time.Unix(1, 0).UTC(), UpdatedAt: time.Unix(2, 0).UTC()}, nil
}
func (stub *faceHTTPPersonMutationStub) Delete(_ context.Context, id string, revision int64) error {
	stub.deletedID, stub.deletedRevision = id, revision
	return nil
}

func (stub *faceHTTPReviewStub) Review(_ context.Context, _ string, request face.ReviewRequest) (face.ReviewResult, error) {
	stub.reviewRequest = request
	return face.ReviewResult{EventID: "freview_http_0001", Action: request.Action, AffectedPersonIDs: []string{request.PersonID}, Revision: 2, Undoable: true}, nil
}

func (stub *faceHTTPReviewStub) Undo(_ context.Context, _ string, reviewID string, revision int64) (face.ReviewResult, error) {
	stub.undoID, stub.undoRevision = reviewID, revision
	return face.ReviewResult{EventID: "freview_http_undo1", Action: "undo", Revision: 3}, nil
}

func TestFaceReadRoutesUsePrivacySafeWireDTOs(t *testing.T) {
	now := time.Date(2026, 9, 1, 7, 0, 0, 0, time.UTC)
	person := face.Person{ID: "person_http_01", Name: "同名", ConfirmedFaceCount: 2, AssetCount: 1, Revision: 3, CreatedAt: now, UpdatedAt: now}
	mux := http.NewServeMux()
	registerFaceReadRoutes(mux,
		faceHTTPSettingsStub{face.LibrarySettings{LibraryID: 1, Enabled: true, State: "ready", Revision: 2, ActiveGenerationID: "face_generation_http", Coverage: face.FaceCoverage{Eligible: 2, Completed: 2, Revision: 4}}},
		faceHTTPClustersStub{face.FaceClusterPage{Items: []face.FaceClusterView{{ID: "fcluster_http_01", LibraryID: 1, Role: "core", MemberCount: 2, PreviewAssetIDs: []int64{7, 8}, Revision: 1}}, Coverage: face.FaceCoverage{Eligible: 2, Completed: 2, Revision: 4}, GroupAssignmentAllowed: false}},
		faceHTTPPeopleStub{face.PeoplePage{Items: []face.Person{person}}}, faceHTTPPersonStub{person},
		faceHTTPAssetFacesStub{[]face.AssetFaceView{{FaceID: "face_http_0001", AssetID: 7, Ordinal: 1, Region: face.CoarseRegion{XPercent: 10, YPercent: 20, WidthPercent: 30, HeightPercent: 40}, State: "assigned", PersonID: person.ID, Revision: 2}}},
		faceHTTPPersonAssetsStub{face.PersonAssetPage{Items: []face.PersonAssetView{{LibraryID: 1, AssetID: 7, FaceIDs: []string{"face_http_0001"}}}}},
		faceHTTPClusterDetailStub{face.FaceClusterDetailPage{Cluster: face.FaceClusterView{ID: "fcluster_http_01", LibraryID: 1, Role: "core", MemberCount: 1, PreviewAssetIDs: []int64{7}, Revision: 1}, Items: []face.FaceClusterMemberView{{FaceID: "face_http_0001", AssetID: 7, Role: "core", Region: face.CoarseRegion{XPercent: 10, YPercent: 20, WidthPercent: 30, HeightPercent: 40}, Revision: 2}}}},
		catalogServiceStub{getAssetsByIDs: func(context.Context, []int64) ([]catalog.Asset, error) {
			return []catalog.Asset{{ID: 7, LibraryID: 1, DirectoryID: 2, Name: "多人.jpg", RelativePath: "多人.jpg", Kind: catalog.KindImage, ModifiedAtNS: now.UnixNano(), Availability: catalog.SourceAvailable}}, nil
		}})
	for _, path := range []string{"/api/v1/libraries/lib_1/ai/faces", "/api/v1/libraries/lib_1/ai/face-clusters?kind=core&limit=20", "/api/v1/people?query=%E5%90%8C&limit=20", "/api/v1/people/person_http_01", "/api/v1/assets/ast_7/faces", "/api/v1/people/person_http_01/assets?limit=20", "/api/v1/libraries/lib_1/ai/face-clusters/fcluster_http_01?limit=20"} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("path=%s status=%d body=%s", path, response.Code, response.Body.String())
		}
		body := response.Body.String()
		for _, forbidden := range []string{"embedding", "vector", "cropPath", "box_x", "sourceFingerprint", "detectionScore", "qualityScore"} {
			if strings.Contains(body, forbidden) {
				t.Fatalf("path=%s leaked %q: %s", path, forbidden, body)
			}
		}
	}
}

func TestFaceReadRoutesRejectDuplicateOrUnknownQueryKeys(t *testing.T) {
	mux := http.NewServeMux()
	registerFaceReadRoutes(mux, faceHTTPSettingsStub{}, faceHTTPClustersStub{}, faceHTTPPeopleStub{}, faceHTTPPersonStub{}, faceHTTPAssetFacesStub{}, faceHTTPPersonAssetsStub{}, faceHTTPClusterDetailStub{}, catalogServiceStub{})
	for _, path := range []string{"/api/v1/people?limit=1&limit=2", "/api/v1/libraries/lib_1/ai/face-clusters?offset=1", "/api/v1/people/person_http_01/assets?limit=1&limit=2", "/api/v1/libraries/lib_1/ai/face-clusters/fcluster_http_01?offset=1"} {
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusBadRequest {
			t.Fatalf("path=%s status=%d", path, response.Code)
		}
	}
}

func TestPersonAssetsRouteDoesNotPresentOfflineSourcesAsEmpty(t *testing.T) {
	mux := http.NewServeMux()
	registerFaceReadRoutes(mux, faceHTTPSettingsStub{}, faceHTTPClustersStub{}, faceHTTPPeopleStub{}, faceHTTPPersonStub{}, faceHTTPAssetFacesStub{}, faceHTTPPersonAssetsErrorStub{err: face.ErrFaceNotReady}, faceHTTPClusterDetailStub{}, catalogServiceStub{})
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/people/person_http_01/assets?limit=20", nil))
	if response.Code != http.StatusConflict || !strings.Contains(response.Body.String(), `"code":"face_not_ready"`) {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestPersonAssetHydrationFailsClosedOnCatalogRaces(t *testing.T) {
	values := []face.PersonAssetView{{LibraryID: 1, AssetID: 7, FaceIDs: []string{"face_http_0001"}}}
	tests := []struct {
		name  string
		items []catalog.Asset
		err   error
		want  error
	}{
		{name: "asset disappeared", err: catalog.ErrAssetNotFound, want: face.ErrFaceProjectionStale},
		{name: "short page", items: []catalog.Asset{}, want: face.ErrFaceProjectionStale},
		{name: "cross library", items: []catalog.Asset{{ID: 7, LibraryID: 2, Kind: catalog.KindImage, Availability: catalog.SourceAvailable}}, want: face.ErrFaceProjectionStale},
		{name: "source offline", items: []catalog.Asset{{ID: 7, LibraryID: 1, Kind: catalog.KindImage, Availability: catalog.SourceOffline}}, want: face.ErrFaceNotReady},
		{name: "wrong media kind", items: []catalog.Asset{{ID: 7, LibraryID: 1, Kind: catalog.KindVideo, Availability: catalog.SourceAvailable}}, want: face.ErrInvalidFaceProjection},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := catalogServiceStub{getAssetsByIDs: func(context.Context, []int64) ([]catalog.Asset, error) {
				return test.items, test.err
			}}
			if _, err := hydratePersonAssets(context.Background(), service, values); !errors.Is(err, test.want) {
				t.Fatalf("expected %v, got %v", test.want, err)
			}
		})
	}
}

func TestFaceControlRoutesUseSettingsRevisionIdempotencyAndExactClearCounts(t *testing.T) {
	settings := faceHTTPSettingsStub{value: face.LibrarySettings{State: "disabled", Revision: 1, Coverage: face.FaceCoverage{Revision: 1}}}
	control := &faceHTTPControlStub{}
	mux := http.NewServeMux()
	registerFaceControlRoutes(mux, settings, control, control)

	update := httptest.NewRequest(http.MethodPatch, "/api/v1/libraries/lib_7/ai/faces", strings.NewReader(`{"enabled":true}`))
	update.Header.Set("Content-Type", "application/json")
	update.Header.Set("If-Match", `"face-library-7-r1"`)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, update)
	if response.Code != http.StatusOK || response.Header().Get("ETag") != `"face-library-7-r2"` {
		t.Fatalf("settings status=%d headers=%v body=%s", response.Code, response.Header(), response.Body.String())
	}

	job := httptest.NewRequest(http.MethodPost, "/api/v1/libraries/lib_7/ai/faces/jobs", strings.NewReader(`{"mode":"missing"}`))
	job.Header.Set("Content-Type", "application/json")
	job.Header.Set("Idempotency-Key", "face-http-job-key-001")
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, job)
	if response.Code != http.StatusAccepted || control.mode != face.JobMissing || response.Header().Get("Location") != "/api/v1/ai/operations/aio_face_http_job" {
		t.Fatalf("job status=%d mode=%q headers=%v body=%s", response.Code, control.mode, response.Header(), response.Body.String())
	}

	manual := httptest.NewRequest(http.MethodPost, "/api/v1/libraries/lib_7/ai/faces/manual-clear", strings.NewReader(`{"confirmation":"clear_manual_face_relationships","expectedPersonCount":2,"expectedAssignmentCount":3,"expectedConstraintCount":4}`))
	manual.Header.Set("Content-Type", "application/json")
	manual.Header.Set("Idempotency-Key", "face-http-clear-key-001")
	manual.Header.Set("If-Match", `"face-library-7-r2"`)
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, manual)
	if response.Code != http.StatusAccepted || control.clearKind != face.ClearManual || control.clearRevision != 2 || control.clearCounts != (face.ManualClearCounts{People: 2, Assignments: 3, Constraints: 4}) {
		t.Fatalf("manual status=%d stub=%+v body=%s", response.Code, control, response.Body.String())
	}

	derived := httptest.NewRequest(http.MethodPost, "/api/v1/libraries/lib_7/ai/faces/derived-clear", strings.NewReader(`{"confirmation":"clear_derived_face_data"}`))
	derived.Header.Set("Content-Type", "application/json")
	derived.Header.Set("Idempotency-Key", "face-http-clear-key-002")
	derived.Header.Set("If-Match", `"face-library-7-r3"`)
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, derived)
	if response.Code != http.StatusAccepted || control.clearKind != face.ClearDerived || response.Header().Get("Idempotency-Replayed") != "true" {
		t.Fatalf("derived status=%d stub=%+v headers=%v body=%s", response.Code, control, response.Header(), response.Body.String())
	}
}

func TestFaceControlRoutesFailClosedBeforeServicesForMalformedInputs(t *testing.T) {
	settings := faceHTTPSettingsStub{}
	control := &faceHTTPControlStub{}
	mux := http.NewServeMux()
	registerFaceControlRoutes(mux, settings, control, control)
	requests := []*http.Request{
		httptest.NewRequest(http.MethodPatch, "/api/v1/libraries/lib_7/ai/faces", strings.NewReader(`{"enabled":true,"path":"/private"}`)),
		httptest.NewRequest(http.MethodPost, "/api/v1/libraries/lib_7/ai/faces/jobs", strings.NewReader(`{"mode":"unknown"}`)),
		httptest.NewRequest(http.MethodPost, "/api/v1/libraries/lib_7/ai/faces/manual-clear", strings.NewReader(`{"confirmation":"clear_manual_face_relationships","expectedPersonCount":-1,"expectedAssignmentCount":0,"expectedConstraintCount":0}`)),
	}
	for _, request := range requests {
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Idempotency-Key", "face-http-invalid-key-001")
		request.Header.Set("If-Match", `"face-library-7-r1"`)
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("path=%s status=%d body=%s", request.URL.Path, response.Code, response.Body.String())
		}
	}
}

func TestFaceErrorMapsOfflineSeparatelyFromModelUnavailable(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/libraries/lib_7/ai/faces/jobs", nil)
	response := httptest.NewRecorder()
	writeFaceError(response, request, face.ErrFaceLibraryOffline)
	if response.Code != http.StatusConflict || !strings.Contains(response.Body.String(), `"code":"library_offline"`) {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestFaceMutationRoutesDecodeTypedReviewAndStrongUndoPrecondition(t *testing.T) {
	stub := &faceHTTPReviewStub{}
	mux := http.NewServeMux()
	registerFaceMutationRoutes(mux, stub)

	review := httptest.NewRequest(http.MethodPost, "/api/v1/ai/face-reviews", strings.NewReader(`{"action":"assign_face","faceId":"face_http_0001","personId":"person_http_01","expectedFaceRevision":1,"expectedPersonRevision":4}`))
	review.Header.Set("Content-Type", "application/json")
	review.Header.Set("Idempotency-Key", "face-review-http-key-001")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, review)
	if response.Code != http.StatusOK || stub.reviewRequest.Action != "assign_face" || stub.reviewRequest.FaceID != "face_http_0001" || stub.reviewRequest.ExpectedPersonRevision != 4 || response.Header().Get("ETag") != `"face-review-freview_http_0001-r2"` {
		t.Fatalf("status=%d request=%+v headers=%v body=%s", response.Code, stub.reviewRequest, response.Header(), response.Body.String())
	}

	undo := httptest.NewRequest(http.MethodPost, "/api/v1/ai/face-reviews/freview_http_0001/undo", nil)
	undo.Header.Set("Idempotency-Key", "face-review-http-key-002")
	undo.Header.Set("If-Match", `"face-review-freview_http_0001-r2"`)
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, undo)
	if response.Code != http.StatusOK || stub.undoID != "freview_http_0001" || stub.undoRevision != 2 {
		t.Fatalf("status=%d undo=(%q,%d) body=%s", response.Code, stub.undoID, stub.undoRevision, response.Body.String())
	}
}

func TestFaceMutationRoutesRejectUnknownJSONAndMissingPrecondition(t *testing.T) {
	mux := http.NewServeMux()
	registerFaceMutationRoutes(mux, &faceHTTPReviewStub{})
	for _, request := range []*http.Request{
		httptest.NewRequest(http.MethodPost, "/api/v1/ai/face-reviews", strings.NewReader(`{"action":"assign_face","path":"/private"}`)),
		httptest.NewRequest(http.MethodPost, "/api/v1/ai/face-reviews/freview_http_0001/undo", nil),
	} {
		request.Header.Set("Idempotency-Key", "face-review-http-key-003")
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("path=%s status=%d body=%s", request.URL.Path, response.Code, response.Body.String())
		}
	}
}

func TestFacePersonMutationRoutesUseIdempotencyAndRevisionPreconditions(t *testing.T) {
	stub := &faceHTTPPersonMutationStub{}
	mux := http.NewServeMux()
	registerFacePersonMutationRoutes(mux, stub)

	create := httptest.NewRequest(http.MethodPost, "/api/v1/people", strings.NewReader(`{"name":"人物","sourceClusterId":"face_cluster_http","expectedClusterRevision":3}`))
	create.Header.Set("Content-Type", "application/json")
	create.Header.Set("Idempotency-Key", "face-person-http-key-001")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, create)
	if response.Code != http.StatusCreated || stub.createdName != "人物" || stub.createdCluster != "face_cluster_http" || stub.createdRevision == nil || *stub.createdRevision != 3 || response.Header().Get("Location") != "/api/v1/people/person_http_created" {
		t.Fatalf("status=%d stub=%+v headers=%v body=%s", response.Code, stub, response.Header(), response.Body.String())
	}

	rename := httptest.NewRequest(http.MethodPatch, "/api/v1/people/person_http_created", strings.NewReader(`{"name":"改名"}`))
	rename.Header.Set("Content-Type", "application/json")
	rename.Header.Set("If-Match", `"person-person_http_created-r1"`)
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, rename)
	if response.Code != http.StatusOK || stub.renamedName != "改名" || stub.renamedRevision != 1 || response.Header().Get("ETag") != `"person-person_http_created-r2"` {
		t.Fatalf("status=%d stub=%+v headers=%v body=%s", response.Code, stub, response.Header(), response.Body.String())
	}

	remove := httptest.NewRequest(http.MethodDelete, "/api/v1/people/person_http_created", nil)
	remove.Header.Set("If-Match", `"person-person_http_created-r2"`)
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, remove)
	if response.Code != http.StatusNoContent || stub.deletedID != "person_http_created" || stub.deletedRevision != 2 {
		t.Fatalf("status=%d stub=%+v body=%s", response.Code, stub, response.Body.String())
	}
}
