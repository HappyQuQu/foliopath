package app

import (
	"io"
	"log/slog"
	"strings"
)

func newJSONLogger(output io.Writer) *slog.Logger {
	return slog.New(slog.NewJSONHandler(output, &slog.HandlerOptions{
		Level:       slog.LevelInfo,
		AddSource:   false,
		ReplaceAttr: redactLogAttribute,
	}))
}

func redactLogAttribute(_ []string, attribute slog.Attr) slog.Attr {
	key := strings.ToLower(attribute.Key)
	for _, forbidden := range []string{
		"authorization",
		"bbox",
		"cookie",
		"crop",
		"csrf",
		"directory",
		"embedding",
		"face",
		"landmark",
		"name",
		"password",
		"path",
		"person",
		"query",
		"score",
		"secret",
		"similarity",
		"sql",
		"stack",
		"stderr",
		"token",
		"vector",
	} {
		if strings.Contains(key, forbidden) {
			return slog.String(attribute.Key, "[redacted]")
		}
	}
	if key == "err" || key == "error" {
		return slog.String(attribute.Key, "[redacted]")
	}
	return attribute
}
