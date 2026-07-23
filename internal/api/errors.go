package api

import (
	"encoding/json"
	"net/http"
)

const (
	codeInternalError    = "internal_error"
	codeResourceNotFound = "resource_not_found"
)

type errorResponse struct {
	Error publicError `json:"error"`
}

type publicError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"requestId"`
}

func writePublicError(
	writer http.ResponseWriter,
	request *http.Request,
	status int,
	code string,
	message string,
) {
	requestID, ok := RequestIDFromContext(request.Context())
	if !ok {
		requestID = fallbackRequestID()
		writer.Header().Set(RequestIDHeader, requestID)
	}

	writeJSON(writer, status, errorResponse{
		Error: publicError{
			Code:      code,
			Message:   message,
			RequestID: requestID,
		},
	})
}

func writeInternalError(writer http.ResponseWriter, request *http.Request) {
	writePublicError(
		writer,
		request,
		http.StatusInternalServerError,
		codeInternalError,
		"An unexpected error occurred.",
	)
}

func writeJSON(writer http.ResponseWriter, status int, payload any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.Header().Set("Cache-Control", "no-store")
	writer.Header().Set("X-Content-Type-Options", "nosniff")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(payload)
}

func notFoundHandler() http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writePublicError(
			writer,
			request,
			http.StatusNotFound,
			codeResourceNotFound,
			"The requested resource was not found.",
		)
	})
}
