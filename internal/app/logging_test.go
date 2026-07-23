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
	for _, forbidden := range []string{"Bearer", "secret-token", "/app/data", "SELECT"} {
		if strings.Contains(logged, forbidden) {
			t.Fatalf("structured log leaked %q: %s", forbidden, logged)
		}
	}
}
