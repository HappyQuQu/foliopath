package api

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/HappyQuQu/foliopath/internal/auth"
)

type AccountService interface {
	Account(context.Context, string) (auth.Account, error)
	UpdateProfile(context.Context, string, auth.ProfileUpdate) (auth.Account, error)
	ChangeAccountPassword(context.Context, string, auth.PasswordChange) (auth.Account, error)
}

type accountResponse struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
	Revision    int64  `json:"revision"`
	UpdatedAt   string `json:"updatedAt"`
}

type accountUpdateRequest struct {
	DisplayName string `json:"displayName"`
}

type passwordChangeRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

func registerAccountRoutes(mux *http.ServeMux, service AccountService) {
	mux.HandleFunc("GET /api/v1/account", func(writer http.ResponseWriter, request *http.Request) {
		session, ok := authenticatedRequestFromContext(request.Context())
		if !ok {
			writeAuthenticationError(writer, request, authenticationRequiredError())
			return
		}
		account, err := service.Account(request.Context(), session.cookieToken)
		if err != nil {
			writeAccountError(writer, request, err)
			return
		}
		writer.Header().Set("ETag", accountETag(account.Revision))
		writeJSON(writer, http.StatusOK, accountWire(account))
	})

	mux.HandleFunc("PATCH /api/v1/account", func(writer http.ResponseWriter, request *http.Request) {
		session, ok := authenticatedRequestFromContext(request.Context())
		if !ok {
			writeAuthenticationError(writer, request, authenticationRequiredError())
			return
		}
		revision, err := parseAccountIfMatch(request.Header.Get("If-Match"))
		if err != nil {
			writeAccountError(writer, request, err)
			return
		}
		var payload accountUpdateRequest
		if err := decodeAuthenticationJSON(writer, request, &payload); err != nil {
			writeAuthenticationError(writer, request, invalidRequestError())
			return
		}
		account, err := service.UpdateProfile(request.Context(), session.cookieToken, auth.ProfileUpdate{
			DisplayName:      payload.DisplayName,
			ExpectedRevision: revision,
		})
		if err != nil {
			writeAccountError(writer, request, err)
			return
		}
		writer.Header().Set("ETag", accountETag(account.Revision))
		writeJSON(writer, http.StatusOK, accountWire(account))
	})

	mux.HandleFunc("POST /api/v1/account/password", func(writer http.ResponseWriter, request *http.Request) {
		session, ok := authenticatedRequestFromContext(request.Context())
		if !ok {
			writeAuthenticationError(writer, request, authenticationRequiredError())
			return
		}
		revision, err := parseAccountIfMatch(request.Header.Get("If-Match"))
		if err != nil {
			writeAccountError(writer, request, err)
			return
		}
		var payload passwordChangeRequest
		if err := decodeAuthenticationJSON(writer, request, &payload); err != nil {
			writeAuthenticationError(writer, request, invalidRequestError())
			return
		}
		account, err := service.ChangeAccountPassword(
			request.Context(),
			session.cookieToken,
			auth.PasswordChange{
				CurrentPassword:  payload.CurrentPassword,
				NewPassword:      payload.NewPassword,
				ExpectedRevision: revision,
			},
		)
		payload.CurrentPassword = ""
		payload.NewPassword = ""
		if err != nil {
			writeAccountError(writer, request, err)
			return
		}
		writer.Header().Set("ETag", accountETag(account.Revision))
		writeNoContent(writer)
	})
}

func parseAccountIfMatch(value string) (int64, error) {
	if value == "" {
		return 0, errPreconditionRequired
	}
	if !strings.HasPrefix(value, `"account-r`) || !strings.HasSuffix(value, `"`) {
		return 0, auth.ErrPreconditionFailed
	}
	revision, err := strconv.ParseInt(
		strings.TrimSuffix(strings.TrimPrefix(value, `"account-r`), `"`),
		10,
		64,
	)
	if err != nil || revision < 1 {
		return 0, auth.ErrPreconditionFailed
	}
	return revision, nil
}

func accountETag(revision int64) string {
	return `"account-r` + strconv.FormatInt(revision, 10) + `"`
}

func accountWire(account auth.Account) accountResponse {
	return accountResponse{
		ID:          "usr_" + strconv.FormatInt(account.ID, 10),
		Username:    account.Username,
		DisplayName: account.DisplayName,
		Revision:    account.Revision,
		UpdatedAt:   time.UnixMilli(account.UpdatedAtMS).UTC().Format(time.RFC3339Nano),
	}
}

func writeAccountError(writer http.ResponseWriter, request *http.Request, err error) {
	switch {
	case errors.Is(err, errPreconditionRequired):
		writePublicError(
			writer,
			request,
			http.StatusPreconditionRequired,
			"precondition_required",
			"A current resource validator is required.",
		)
	case errors.Is(err, auth.ErrPreconditionFailed):
		writePublicError(
			writer,
			request,
			http.StatusPreconditionFailed,
			"precondition_failed",
			"The account has changed.",
		)
	default:
		writeAuthenticationError(writer, request, mapAuthenticationError(err))
	}
}
