package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"
)

func TestJSONLoggerProducesStructuredEventsAndRedactsSensitiveAttributes(t *testing.T) {
	var output bytes.Buffer
	logger := newJSONLogger(&output)
	logger.Info(
		"authentication.failed",
		slog.String("request_id", "req_test"),
		slog.String("authorization", "Bearer secret-token"),
		slog.String("face_id", "face-canary-7f64e1"),
		slog.String("person_name", "person-canary-8a7d22"),
		slog.String("embedding_vector", "vector-canary-51ed34"),
		slog.Float64("similarity_score", 0.987654321),
		slog.String("crop_path", "/app/data/private-crop-canary"),
		slog.String("session_token", "secret-token"),
		slog.String("database_path", "/app/data/foliopath.db"),
		slog.String("sql_query", "SELECT password FROM users"),
		slog.Any("error", errors.New("database /app/data/foliopath.db SELECT secret")),
	)

	var event map[string]any
	if err := json.Unmarshal(output.Bytes(), &event); err != nil {
		t.Fatalf("decode structured log: %v", err)
	}
	if event["msg"] != "authentication.failed" || event["request_id"] != "req_test" {
		t.Fatalf("structured event = %#v, want message and request ID", event)
	}
	for _, key := range []string{
		"authorization",
		"face_id",
		"person_name",
		"embedding_vector",
		"similarity_score",
		"crop_path",
		"session_token",
		"database_path",
		"sql_query",
		"error",
	} {
		if event[key] != "[redacted]" {
			t.Fatalf("%s = %#v, want redacted", key, event[key])
		}
	}

	logged := output.String()
	for _, forbidden := range []string{
		"Bearer", "secret-token", "/app/data", "SELECT",
		"face-canary", "person-canary", "vector-canary", "0.987654321",
	} {
		if strings.Contains(logged, forbidden) {
			t.Fatalf("structured log leaked %q: %s", forbidden, logged)
		}
	}
}
