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
		"cookie",
		"csrf",
		"directory",
		"password",
		"path",
		"query",
		"secret",
		"sql",
		"stack",
		"stderr",
		"token",
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
