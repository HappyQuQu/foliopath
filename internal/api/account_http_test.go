package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/auth"
)

type accountServiceStub struct {
	account  auth.Account
	update   auth.ProfileUpdate
	password auth.PasswordChange
	err      error
}

func (stub *accountServiceStub) Account(context.Context, string) (auth.Account, error) {
	return stub.account, stub.err
}

func (stub *accountServiceStub) UpdateProfile(
	_ context.Context,
	_ string,
	update auth.ProfileUpdate,
) (auth.Account, error) {
	stub.update = update
	stub.account.DisplayName = update.DisplayName
	stub.account.Revision++
	return stub.account, stub.err
}

func (stub *accountServiceStub) ChangeAccountPassword(
	_ context.Context,
	_ string,
	change auth.PasswordChange,
) (auth.Account, error) {
	stub.password = change
	stub.account.Revision++
	return stub.account, stub.err
}

func TestAccountRoutesExposeETagAndMutateWithValidator(t *testing.T) {
	service := &accountServiceStub{account: auth.Account{
		ID:          1,
		Username:    "administrator",
		DisplayName: "Admin",
		Revision:    1,
		UpdatedAtMS: 1_700_000_000_000,
	}}
	mux := http.NewServeMux()
	registerAccountRoutes(mux, service)

	get := serveAuthenticatedAccountRequest(
		mux,
		httptest.NewRequest(http.MethodGet, "/api/v1/account", nil),
	)
	if get.Code != http.StatusOK || get.Header().Get("ETag") != `"account-r1"` {
		t.Fatalf("GET account = %d %q %s", get.Code, get.Header().Get("ETag"), get.Body.String())
	}
	if get.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("GET account Cache-Control = %q", get.Header().Get("Cache-Control"))
	}

	patchRequest := httptest.NewRequest(
		http.MethodPatch,
		"/api/v1/account",
		strings.NewReader(`{"displayName":"Home Admin"}`),
	)
	patchRequest.Header.Set("Content-Type", "application/json")
	patchRequest.Header.Set("If-Match", `"account-r1"`)
	patch := serveAuthenticatedAccountRequest(mux, patchRequest)
	if patch.Code != http.StatusOK ||
		patch.Header().Get("ETag") != `"account-r2"` ||
		service.update.ExpectedRevision != 1 ||
		service.update.DisplayName != "Home Admin" {
		t.Fatalf("PATCH account = %d %q %#v %s", patch.Code, patch.Header().Get("ETag"), service.update, patch.Body.String())
	}

	passwordRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/account/password",
		strings.NewReader(`{"currentPassword":"current pass","newPassword":"replacement pass"}`),
	)
	passwordRequest.Header.Set("Content-Type", "application/json")
	passwordRequest.Header.Set("If-Match", `"account-r2"`)
	password := serveAuthenticatedAccountRequest(mux, passwordRequest)
	if password.Code != http.StatusNoContent ||
		password.Header().Get("ETag") != `"account-r3"` ||
		service.password.ExpectedRevision != 2 {
		t.Fatalf("POST password = %d %q %#v %s", password.Code, password.Header().Get("ETag"), service.password, password.Body.String())
	}
}

func TestAccountRoutesRequireStrongValidator(t *testing.T) {
	service := &accountServiceStub{}
	mux := http.NewServeMux()
	registerAccountRoutes(mux, service)
	request := httptest.NewRequest(
		http.MethodPatch,
		"/api/v1/account",
		strings.NewReader(`{"displayName":"Admin"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	response := serveAuthenticatedAccountRequest(mux, request)
	if response.Code != http.StatusPreconditionRequired {
		t.Fatalf("missing If-Match status = %d, body %s", response.Code, response.Body.String())
	}
}

func serveAuthenticatedAccountRequest(
	handler http.Handler,
	request *http.Request,
) *httptest.ResponseRecorder {
	ctx := context.WithValue(request.Context(), authenticatedRequestContextKey{}, authenticatedRequest{
		current:     auth.CurrentSession{Administrator: auth.Administrator{ID: 1}},
		cookieToken: "cookie-token",
	})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request.WithContext(ctx))
	return response
}
