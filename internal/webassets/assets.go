// Package webassets owns the embedded production SPA and its HTTP delivery.
package webassets

import (
	"embed"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"
)

// dist is populated by the Vite production build before the release Go build.
// The tracked .gitkeep keeps ordinary Go builds valid without committing
// generated frontend output.
//
//go:embed all:dist
var dist embed.FS

// NewHandler serves embedded frontend files and falls through to next for API,
// health, missing asset, and development builds without a generated index.
func NewHandler(next http.Handler) http.Handler {
	site, err := fs.Sub(dist, "dist")
	if err != nil {
		return fallback(next)
	}
	return newHandler(site, next)
}

func newHandler(site fs.FS, next http.Handler) http.Handler {
	next = fallback(next)
	files := http.FileServer(http.FS(site))

	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet && request.Method != http.MethodHead {
			next.ServeHTTP(writer, request)
			return
		}
		if reservedPath(request.URL.Path) {
			next.ServeHTTP(writer, request)
			return
		}

		name := strings.TrimPrefix(path.Clean(request.URL.Path), "/")
		if name == "." || name == "" {
			name = "index.html"
		}
		if !fs.ValidPath(name) {
			next.ServeHTTP(writer, request)
			return
		}

		if info, err := fs.Stat(site, name); err == nil && !info.IsDir() {
			setStaticHeaders(writer, name)
			files.ServeHTTP(writer, request)
			return
		}
		if path.Ext(name) != "" {
			next.ServeHTTP(writer, request)
			return
		}
		if _, err := fs.Stat(site, "index.html"); err != nil {
			next.ServeHTTP(writer, request)
			return
		}

		clone := request.Clone(request.Context())
		clone.URL.Path = "/"
		setStaticHeaders(writer, "index.html")
		files.ServeHTTP(writer, clone)
	})
}

func reservedPath(requestPath string) bool {
	return requestPath == "/api" ||
		strings.HasPrefix(requestPath, "/api/") ||
		requestPath == "/health" ||
		strings.HasPrefix(requestPath, "/health/")
}

func setStaticHeaders(writer http.ResponseWriter, name string) {
	writer.Header().Set("X-Content-Type-Options", "nosniff")
	if mediaType := mime.TypeByExtension(path.Ext(name)); mediaType != "" {
		writer.Header().Set("Content-Type", mediaType)
	}
	if name == "index.html" {
		writer.Header().Set("Cache-Control", "no-cache")
		return
	}
	writer.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
}

func fallback(next http.Handler) http.Handler {
	if next == nil {
		return http.NotFoundHandler()
	}
	return next
}
