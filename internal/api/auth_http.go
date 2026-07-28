package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strconv"
	"time"

	"github.com/HappyQuQu/foliopath/internal/auth"
)

const (
	maxAuthenticationBodyBytes = 4 * 1024
	csrfTokenHeader            = "X-CSRF-Token"

	codeAuthenticationRequired = "authentication_required"
	codeCSRFInvalid            = "csrf_invalid"
	codeInvalidCredentials     = "invalid_credentials"
	codeInvalidRequest         = "invalid_request"
	codeOriginInvalid          = "origin_invalid"
	codeSessionExpired         = "session_expired"
	codeSetupClosed            = "setup_closed"
	codeSetupInProgress        = "setup_in_progress"
	codeValidationFailed       = "validation_failed"
)

type AuthenticationService interface {
	SetupState(context.Context) (auth.SetupState, error)
	Initialize(context.Context, auth.InitializeParams) (auth.EstablishedSession, error)
	Login(context.Context, auth.LoginParams) (auth.EstablishedSession, error)
	Session(context.Context, string) (auth.CurrentSession, error)
	Logout(context.Context, string) error
}

type authenticationStatusResponse struct {
	SetupRequired          bool `json:"setupRequired"`
	AuthenticationRequired bool `json:"authenticationRequired"`
}

type setupRequest struct {
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
	Password    string `json:"password"`
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type administratorResponse struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
}

type sessionResponse struct {
	Administrator administratorResponse `json:"administrator"`
	ExpiresAt     string                `json:"expiresAt"`
	CSRFToken     string                `json:"csrfToken"`
}

func registerAuthenticationRoutes(mux *http.ServeMux, service AuthenticationService) {
	mux.HandleFunc("GET /api/v1/auth/status", func(
		writer http.ResponseWriter,
		request *http.Request,
	) {
		state, err := service.SetupState(request.Context())
		if err != nil {
			writeInternalError(writer, request)
			return
		}
		writeJSON(writer, http.StatusOK, authenticationStatusResponse{
			SetupRequired:          state == auth.SetupRequired,
			AuthenticationRequired: true,
		})
	})

	mux.HandleFunc("POST /api/v1/auth/setup", func(
		writer http.ResponseWriter,
		request *http.Request,
	) {
		if !requestHasSameOrigin(request) {
			writeAuthenticationError(writer, request, authHTTPError{
				status:  http.StatusForbidden,
				code:    codeOriginInvalid,
				message: "The request origin is not allowed.",
			})
			return
		}
		var payload setupRequest
		if err := decodeAuthenticationJSON(writer, request, &payload); err != nil {
			writeAuthenticationError(writer, request, invalidRequestError())
			return
		}
		established, err := service.Initialize(request.Context(), auth.InitializeParams{
			Username:    payload.Username,
			DisplayName: payload.DisplayName,
			Password:    payload.Password,
		})
		payload.Password = ""
		if err != nil {
			writeAuthenticationError(writer, request, mapAuthenticationError(err))
			return
		}
		writeEstablishedSession(writer, request, http.StatusCreated, established)
	})

	mux.HandleFunc("POST /api/v1/auth/login", func(
		writer http.ResponseWriter,
		request *http.Request,
	) {
		if !requestHasSameOrigin(request) {
			writeAuthenticationError(writer, request, authHTTPError{
				status:  http.StatusForbidden,
				code:    codeOriginInvalid,
				message: "The request origin is not allowed.",
			})
			return
		}
		var payload loginRequest
		if err := decodeAuthenticationJSON(writer, request, &payload); err != nil {
			writeAuthenticationError(writer, request, invalidRequestError())
			return
		}
		established, err := service.Login(request.Context(), auth.LoginParams{
			Username: payload.Username,
			Password: payload.Password,
		})
		payload.Password = ""
		if err != nil {
			writeAuthenticationError(writer, request, mapAuthenticationError(err))
			return
		}
		writeEstablishedSession(writer, request, http.StatusOK, established)
	})

	mux.HandleFunc("GET /api/v1/auth/session", func(
		writer http.ResponseWriter,
		request *http.Request,
	) {
		session, ok := authenticatedRequestFromContext(request.Context())
		if !ok {
			writeAuthenticationError(writer, request, authenticationRequiredError())
			return
		}
		writeJSON(writer, http.StatusOK, currentSessionResponse(session.current))
	})

	mux.HandleFunc("POST /api/v1/auth/logout", func(
		writer http.ResponseWriter,
		request *http.Request,
	) {
		session, ok := authenticatedRequestFromContext(request.Context())
		if !ok {
			writeAuthenticationError(writer, request, authenticationRequiredError())
			return
		}
		if err := service.Logout(request.Context(), session.cookieToken); err != nil {
			writeAuthenticationError(writer, request, mapAuthenticationError(err))
			return
		}
		expired := ExpiredSessionCookie(requestTransportFrom(request).secure)
		http.SetCookie(writer, &expired)
		writeNoContent(writer)
	})
}

func writeEstablishedSession(
	writer http.ResponseWriter,
	request *http.Request,
	status int,
	established auth.EstablishedSession,
) {
	expiresAt := time.UnixMilli(established.ExpiresAtMS).UTC()
	cookie, err := NewSessionCookie(
		established.CookieToken,
		time.Now().UTC(),
		expiresAt,
		requestTransportFrom(request).secure,
	)
	if err != nil {
		writeInternalError(writer, request)
		return
	}
	http.SetCookie(writer, &cookie)
	writeJSON(writer, status, sessionResponse{
		Administrator: administratorWire(established.Administrator),
		ExpiresAt:     expiresAt.Format(time.RFC3339Nano),
		CSRFToken:     established.CSRFToken,
	})
}

func currentSessionResponse(current auth.CurrentSession) sessionResponse {
	return sessionResponse{
		Administrator: administratorWire(current.Administrator),
		ExpiresAt: time.UnixMilli(current.ExpiresAtMS).
			UTC().
			Format(time.RFC3339Nano),
		CSRFToken: current.CSRFToken,
	}
}

func administratorWire(administrator auth.Administrator) administratorResponse {
	return administratorResponse{
		ID:          "usr_" + strconv.FormatInt(administrator.ID, 10),
		Username:    administrator.Username,
		DisplayName: administrator.DisplayName,
	}
}

func decodeAuthenticationJSON(
	writer http.ResponseWriter,
	request *http.Request,
	target any,
) error {
	contentType, _, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if err != nil || contentType != "application/json" {
		return errors.New("authentication request must be JSON")
	}

	request.Body = http.MaxBytesReader(writer, request.Body, maxAuthenticationBodyBytes)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode authentication request: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("authentication request contains multiple JSON values")
	}
	return nil
}

type authHTTPError struct {
	status  int
	code    string
	message string
}

func mapAuthenticationError(err error) authHTTPError {
	switch {
	case errors.Is(err, auth.ErrInvalidUsername),
		errors.Is(err, auth.ErrInvalidDisplayName),
		errors.Is(err, auth.ErrInvalidPassword):
		return authHTTPError{
			status:  http.StatusUnprocessableEntity,
			code:    codeValidationFailed,
			message: "One or more authentication fields are invalid.",
		}
	case errors.Is(err, auth.ErrInvalidCredentials):
		return authHTTPError{
			status:  http.StatusUnauthorized,
			code:    codeInvalidCredentials,
			message: "The username or password is invalid.",
		}
	case errors.Is(err, auth.ErrSetupClosed):
		return authHTTPError{
			status:  http.StatusConflict,
			code:    codeSetupClosed,
			message: "Administrator setup is closed.",
		}
	case errors.Is(err, auth.ErrSetupInProgress):
		return authHTTPError{
			status:  http.StatusConflict,
			code:    codeSetupInProgress,
			message: "Administrator setup is already in progress.",
		}
	case errors.Is(err, auth.ErrAuthenticationRequired):
		return authenticationRequiredError()
	case errors.Is(err, auth.ErrSessionExpired):
		return authHTTPError{
			status:  http.StatusUnauthorized,
			code:    codeSessionExpired,
			message: "The administrator session has expired.",
		}
	default:
		return authHTTPError{
			status:  http.StatusInternalServerError,
			code:    codeInternalError,
			message: "An unexpected error occurred.",
		}
	}
}

func authenticationRequiredError() authHTTPError {
	return authHTTPError{
		status:  http.StatusUnauthorized,
		code:    codeAuthenticationRequired,
		message: "An authenticated administrator session is required.",
	}
}

func invalidRequestError() authHTTPError {
	return authHTTPError{
		status:  http.StatusBadRequest,
		code:    codeInvalidRequest,
		message: "The request body is invalid.",
	}
}

func writeAuthenticationError(
	writer http.ResponseWriter,
	request *http.Request,
	public authHTTPError,
) {
	writePublicError(writer, request, public.status, public.code, public.message)
}

func writeNoContent(writer http.ResponseWriter) {
	writer.Header().Set("Cache-Control", "no-store")
	writer.Header().Set("X-Content-Type-Options", "nosniff")
	writer.WriteHeader(http.StatusNoContent)
}
