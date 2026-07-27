package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/library"
)

type libraryPathServiceStub struct {
	list  func(context.Context, library.ListPathParams) (library.PathPage, error)
	calls int
}

func (stub *libraryPathServiceStub) ListPaths(
	ctx context.Context,
	params library.ListPathParams,
) (library.PathPage, error) {
	stub.calls++
	if stub.list == nil {
		return library.PathPage{}, errors.New("unexpected library path request")
	}
	return stub.list(ctx, params)
}

func TestLibraryPathsRequireAuthenticationBeforeFilesystemService(t *testing.T) {
	service := &libraryPathServiceStub{}
	handler := testRoutes(t, RouteDependencies{
		Readiness:      func() Readiness { return Readiness{Ready: true} },
		Authentication: rejectingAuthentication(),
		SystemStatus:   unusedStatus,
		LibraryPaths:   service,
	})
	response := httptest.NewRecorder()
	handler.ServeHTTP(
		response,
		httptest.NewRequest(http.MethodGet, "/api/v1/library-paths", nil),
	)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
	assertSafeErrorResponse(t, response, codeAuthenticationRequired)
	if service.calls != 0 {
		t.Fatalf("unauthorized request performed %d directory listings", service.calls)
	}
}

func TestLibraryPathsReturnContractPage(t *testing.T) {
	wantParams := library.ListPathParams{
		Parent: "家庭",
		Cursor: "opaque-cursor",
		Limit:  2,
	}
	service := &libraryPathServiceStub{
		list: func(
			_ context.Context,
			params library.ListPathParams,
		) (library.PathPage, error) {
			if !reflect.DeepEqual(params, wantParams) {
				t.Fatalf("params = %#v, want %#v", params, wantParams)
			}
			return library.PathPage{
				Location: library.PathLocation{Name: "家庭", RelativePath: "家庭"},
				Breadcrumbs: []library.PathLocation{
					{Name: "Media root", RelativePath: ""},
					{Name: "家庭", RelativePath: "家庭"},
				},
				Items: []library.PathEntry{
					{
						Name:         "2026",
						RelativePath: "家庭/2026",
						HasChildren:  true,
						Selectable:   true,
					},
					{
						Name:          "external",
						RelativePath:  "家庭/external",
						Selectable:    false,
						BlockedReason: library.SelectionBlockedMountBoundary,
					},
				},
				NextCursor: "next-opaque-cursor",
			}, nil
		},
	}
	handler := libraryPathTestRoutes(t, service)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/library-paths?parent=%E5%AE%B6%E5%BA%AD&cursor=opaque-cursor&limit=2",
		nil,
	)
	request.AddCookie(&http.Cookie{Name: SessionCookieName, Value: testCookieToken})
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body)
	}
	assertNoStore(t, response)
	assertJSONEquals(t, response, map[string]any{
		"location": map[string]any{
			"name":         "家庭",
			"relativePath": "家庭",
		},
		"breadcrumbs": []any{
			map[string]any{"name": "Media root", "relativePath": ""},
			map[string]any{"name": "家庭", "relativePath": "家庭"},
		},
		"items": []any{
			map[string]any{
				"name":                   "2026",
				"relativePath":           "家庭/2026",
				"hasChildren":            true,
				"selectable":             true,
				"selectionBlockedReason": nil,
				"conflictingLibraryId":   nil,
				"conflictingLibraryName": nil,
			},
			map[string]any{
				"name":                   "external",
				"relativePath":           "家庭/external",
				"hasChildren":            false,
				"selectable":             false,
				"selectionBlockedReason": "mount_boundary",
				"conflictingLibraryId":   nil,
				"conflictingLibraryName": nil,
			},
		},
		"nextCursor": "next-opaque-cursor",
	})
}

func TestLibraryPathQueryRejectsMalformedValuesBeforeService(t *testing.T) {
	service := &libraryPathServiceStub{}
	handler := libraryPathTestRoutes(t, service)
	for _, query := range []string{
		"?unknown=value",
		"?parent=a&parent=b",
		"?limit=0",
		"?limit=201",
		"?limit=1.5",
		"?limit=",
		"?parent=a;b",
	} {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(
			http.MethodGet,
			"/api/v1/library-paths"+query,
			nil,
		)
		request.AddCookie(&http.Cookie{Name: SessionCookieName, Value: testCookieToken})
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("query %q status = %d, want %d", query, response.Code, http.StatusBadRequest)
		}
		assertSafeErrorResponse(t, response, codeInvalidRequest)
	}
	if service.calls != 0 {
		t.Fatalf("invalid query performed %d service calls", service.calls)
	}

	response := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/library-paths?cursor=",
		nil,
	)
	request.AddCookie(&http.Cookie{Name: SessionCookieName, Value: testCookieToken})
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("empty cursor status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	assertSafeErrorResponse(t, response, codeInvalidCursor)
	if service.calls != 0 {
		t.Fatalf("empty cursor performed %d service calls", service.calls)
	}
}

func TestLibraryPathErrorsUseStableSafeMappings(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{
			name:   "invalid cursor",
			err:    library.ErrInvalidCursor,
			status: http.StatusBadRequest,
			code:   codeInvalidCursor,
		},
		{
			name:   "invalid parent",
			err:    library.ErrInvalidParent,
			status: http.StatusBadRequest,
			code:   codeInvalidRequest,
		},
		{
			name:   "unavailable",
			err:    library.ErrParentUnavailable,
			status: http.StatusConflict,
			code:   codeLibraryRootUnavailable,
		},
		{
			name:   "symlink",
			err:    library.ErrParentSymlink,
			status: http.StatusConflict,
			code:   codeLibraryRootSymlink,
		},
		{
			name:   "mount",
			err:    library.ErrParentMountBoundary,
			status: http.StatusConflict,
			code:   codeLibraryRootMountBoundary,
		},
		{
			name:   "internal masked",
			err:    errors.New("open /host/private: permission denied"),
			status: http.StatusInternalServerError,
			code:   codeInternalError,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := &libraryPathServiceStub{
				list: func(
					context.Context,
					library.ListPathParams,
				) (library.PathPage, error) {
					return library.PathPage{}, test.err
				},
			}
			handler := libraryPathTestRoutes(t, service)
			response := httptest.NewRecorder()
			request := httptest.NewRequest(
				http.MethodGet,
				"/api/v1/library-paths",
				nil,
			)
			request.AddCookie(&http.Cookie{Name: SessionCookieName, Value: testCookieToken})
			handler.ServeHTTP(response, request)
			if response.Code != test.status {
				t.Fatalf("status = %d, want %d", response.Code, test.status)
			}
			assertSafeErrorResponse(t, response, test.code)
			if test.name == "internal masked" && len(response.Body.String()) > 0 {
				for _, leaked := range []string{"/host/private", "permission denied"} {
					if contains := stringContains(response.Body.String(), leaked); contains {
						t.Fatalf("response leaked %q: %s", leaked, response.Body)
					}
				}
			}
		})
	}
}

func libraryPathTestRoutes(
	t *testing.T,
	service LibraryPathService,
) http.Handler {
	t.Helper()
	return testRoutes(t, RouteDependencies{
		Readiness:      func() Readiness { return Readiness{Ready: true} },
		Authentication: acceptingAuthentication(),
		SystemStatus:   unusedStatus,
		LibraryPaths:   service,
	})
}

func stringContains(value, fragment string) bool {
	for index := 0; index+len(fragment) <= len(value); index++ {
		if value[index:index+len(fragment)] == fragment {
			return true
		}
	}
	return false
}
