// Package api owns FolioPath's HTTP transport, public error mapping, and
// request-scoped transport metadata.
package api

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"
)

const RequestIDHeader = "X-Request-ID"

type requestIDContextKey struct{}

type requestIDGenerator func() (string, error)

var fallbackRequestSequence atomic.Uint64

// RequestIDFromContext returns the server-generated correlation ID for a
// request. Client-supplied request IDs are never stored in this context.
func RequestIDFromContext(ctx context.Context) (string, bool) {
	requestID, ok := ctx.Value(requestIDContextKey{}).(string)
	return requestID, ok && requestID != ""
}

func withRequestID(next http.Handler, generate requestIDGenerator) http.Handler {
	if next == nil {
		next = http.NotFoundHandler()
	}
	if generate == nil {
		generate = generateRequestID
	}

	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestID, err := generate()
		if err != nil || !validRequestID(requestID) {
			requestID = fallbackRequestID()
		}

		writer.Header().Set(RequestIDHeader, requestID)
		ctx := context.WithValue(request.Context(), requestIDContextKey{}, requestID)
		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}

func generateRequestID() (string, error) {
	var entropy [18]byte
	if _, err := rand.Read(entropy[:]); err != nil {
		return "", fmt.Errorf("generate request ID: %w", err)
	}
	return "req_" + base64.RawURLEncoding.EncodeToString(entropy[:]), nil
}

func fallbackRequestID() string {
	return fmt.Sprintf(
		"req_fallback_%x_%x",
		time.Now().UnixNano(),
		fallbackRequestSequence.Add(1),
	)
}

func validRequestID(value string) bool {
	if len(value) < 5 || len(value) > 128 || value[:4] != "req_" {
		return false
	}
	for _, character := range value[4:] {
		switch {
		case character >= 'A' && character <= 'Z':
		case character >= 'a' && character <= 'z':
		case character >= '0' && character <= '9':
		case character == '_', character == '-':
		default:
			return false
		}
	}
	return true
}
