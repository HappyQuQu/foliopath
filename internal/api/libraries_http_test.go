package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/library"
	"github.com/HappyQuQu/foliopath/internal/scanner"
)

type libraryLifecycleStub struct {
	create func(context.Context, string, string, string) (library.CreateResult, error)
	get    func(context.Context, int64) (library.Details, error)
	rename func(context.Context, int64, int64, string) (library.Details, error)
	remove func(context.Context, int64, int64, string) (library.RemoveResult, error)
}

func (stub libraryLifecycleStub) Create(
	ctx context.Context,
	name, root, key string,
) (library.CreateResult, error) {
	return stub.create(ctx, name, root, key)
}

func (libraryLifecycleStub) List(context.Context, string, int) (library.Page, error) {
	return library.Page{}, errors.New("unexpected List")
}

func (stub libraryLifecycleStub) Get(ctx context.Context, id int64) (library.Details, error) {
	return stub.get(ctx, id)
}

func (stub libraryLifecycleStub) Rename(
	ctx context.Context,
	id, revision int64,
	name string,
) (library.Details, error) {
	return stub.rename(ctx, id, revision, name)
}

func (stub libraryLifecycleStub) Remove(
	ctx context.Context,
	id, revision int64,
	key string,
) (library.RemoveResult, error) {
	return stub.remove(ctx, id, revision, key)
}

func (libraryLifecycleStub) GetRemoval(context.Context, int64) (library.Removal, error) {
	return library.Removal{}, errors.New("unexpected GetRemoval")
}

func TestCreateLibraryReturnsContractHeadersAndSafePaths(t *testing.T) {
	service := libraryLifecycleStub{
		create: func(_ context.Context, name, root, key string) (library.CreateResult, error) {
			if name != "Family" || root != "family" || key != "create-key-1" {
				t.Fatalf("create arguments = %q, %q, %q", name, root, key)
			}
			return library.CreateResult{
				Library: library.Details{
					Library: library.Library{
						ID:               7,
						Name:             "Family",
						RootRelativePath: "family",
						Status:           library.StatusPending,
						Revision:         1,
						CreatedAtMS:      1_000,
						UpdatedAtMS:      1_000,
					},
					LatestScanID: pointerTo(int64(9)),
				},
				Scan: library.Scan{
					ID:          9,
					LibraryID:   7,
					Generation:  1,
					Trigger:     "library_created",
					Status:      "queued",
					Phase:       "queued",
					Revision:    1,
					CreatedAtMS: 1_000,
				},
			}, nil
		},
	}
	mux := http.NewServeMux()
	registerLibraryRoutes(mux, service)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/libraries",
		strings.NewReader(`{"name":"Family","rootPath":"family"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "create-key-1")
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d; body = %s", response.Code, response.Body)
	}
	if got := response.Header().Get("Location"); got != "/api/v1/libraries/lib_7" {
		t.Fatalf("Location = %q", got)
	}
	if got := response.Header().Get("ETag"); got != `"lib_7-r1"` {
		t.Fatalf("ETag = %q", got)
	}
	if strings.Contains(response.Body.String(), "/mnt/") ||
		!strings.Contains(response.Body.String(), `"displayPath":"/library/family"`) {
		t.Fatalf("unsafe or missing display path: %s", response.Body)
	}
}

func TestLibraryMutationRequiresExactStrongValidator(t *testing.T) {
	calls := 0
	service := libraryLifecycleStub{
		rename: func(
			_ context.Context,
			id, revision int64,
			name string,
		) (library.Details, error) {
			calls++
			if id != 7 || revision != 2 || name != "Home" {
				t.Fatalf("rename args = %d, %d, %q", id, revision, name)
			}
			return library.Details{Library: library.Library{
				ID: 7, Name: "Home", Status: library.StatusReady, Revision: 3,
			}}, nil
		},
	}
	mux := http.NewServeMux()
	registerLibraryRoutes(mux, service)

	for _, validator := range []struct {
		value string
		want  int
	}{
		{value: "", want: http.StatusPreconditionRequired},
		{value: `W/"lib_7-r2"`, want: http.StatusPreconditionFailed},
		{value: `"lib_8-r2"`, want: http.StatusPreconditionFailed},
		{value: `"lib_7-r2"`, want: http.StatusOK},
	} {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(
			http.MethodPatch,
			"/api/v1/libraries/lib_7",
			strings.NewReader(`{"name":"Home"}`),
		)
		request.Header.Set("Content-Type", "application/json")
		if validator.value != "" {
			request.Header.Set("If-Match", validator.value)
		}
		mux.ServeHTTP(response, request)
		if response.Code != validator.want {
			t.Fatalf("If-Match %q status = %d, want %d; body = %s",
				validator.value, response.Code, validator.want, response.Body)
		}
	}
	if calls != 1 {
		t.Fatalf("rename service calls = %d, want 1", calls)
	}
}

type scanAdmissionStub struct {
	result scanner.AdmissionResult
	err    error
}

func (stub scanAdmissionStub) RequestManual(
	context.Context,
	int64,
) (scanner.AdmissionResult, error) {
	return stub.result, stub.err
}

func TestManualScanAdmissionDistinguishesNewAndCoalescedWork(t *testing.T) {
	for _, test := range []struct {
		name      string
		coalesced bool
		want      int
	}{
		{name: "new", want: http.StatusAccepted},
		{name: "coalesced", coalesced: true, want: http.StatusOK},
	} {
		t.Run(test.name, func(t *testing.T) {
			mux := http.NewServeMux()
			registerScanAdmissionRoute(mux, scanAdmissionStub{
				result: scanner.AdmissionResult{
					Run: scanner.ScanRun{
						ID: 4, LibraryID: 7, Generation: 2,
						Trigger: scanner.TriggerManual,
						Status:  scanner.RunStatusQueued,
						Phase:   "queued", Revision: 3, CreatedAtMS: 1000,
					},
					Coalesced: test.coalesced,
				},
			})
			response := httptest.NewRecorder()
			mux.ServeHTTP(
				response,
				httptest.NewRequest(
					http.MethodPost,
					"/api/v1/libraries/lib_7/scans",
					nil,
				),
			)
			if response.Code != test.want {
				t.Fatalf("status = %d, want %d; body = %s",
					response.Code, test.want, response.Body)
			}
			if response.Header().Get("Location") != "/api/v1/scans/scan_4" ||
				response.Header().Get("ETag") != `"scan_4-r3"` {
				t.Fatalf("headers = %#v", response.Header())
			}
		})
	}
}

func pointerTo[T any](value T) *T { return &value }
