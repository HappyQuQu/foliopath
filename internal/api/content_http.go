package api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/HappyQuQu/foliopath/internal/media"
)

const (
	contentReadConcurrency = 16
	maxRangeHeaderBytes    = 128
	maxDateHeaderBytes     = 128
	maxETagHeaderBytes     = 512
)

type ContentService interface {
	Open(context.Context, int64) (media.Content, error)
}

type contentHandler struct {
	service ContentService
	slots   chan struct{}
}

type byteRange struct {
	start int64
	end   int64
}

func registerContentRoutes(mux *http.ServeMux, service ContentService) {
	handler := &contentHandler{
		service: service,
		slots:   make(chan struct{}, contentReadConcurrency),
	}
	mux.HandleFunc("GET /api/v1/assets/{assetId}/content", handler.handle)
	mux.HandleFunc("HEAD /api/v1/assets/{assetId}/content", handler.handle)
}

func (handler *contentHandler) handle(writer http.ResponseWriter, request *http.Request) {
	if request.URL.RawQuery != "" {
		writePublicError(writer, request, http.StatusBadRequest, "invalid_request", "The request is invalid.")
		return
	}
	if !validContentRequestHeaders(request) {
		writePublicError(writer, request, http.StatusBadRequest, "invalid_request", "The request is invalid.")
		return
	}
	assetID, err := parseResourceID(request.PathValue("assetId"), "ast_")
	if err != nil {
		writePublicError(writer, request, http.StatusNotFound, "asset_not_found", "The media item was not found.")
		return
	}
	select {
	case handler.slots <- struct{}{}:
		defer func() { <-handler.slots }()
	default:
		writer.Header().Set("Retry-After", "1")
		writePublicError(writer, request, http.StatusTooManyRequests, "rate_limited", "Too many media requests are active.")
		return
	}

	content, err := handler.service.Open(request.Context(), assetID)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return
		}
		writeContentOpenError(writer, request, err)
		return
	}
	defer content.File.Close()

	setContentHeaders(writer.Header(), content)
	var requestedRange byteRange
	var hasRange bool
	if request.Method == http.MethodGet {
		requestedRange, hasRange, err = parseContentRange(
			request.Header.Values("Range"),
			content.SizeBytes,
		)
		if err != nil {
			writer.Header().Set("Content-Range", "bytes */"+strconv.FormatInt(content.SizeBytes, 10))
			writePublicError(writer, request, http.StatusRequestedRangeNotSatisfiable, "range_not_satisfiable", "The requested byte range cannot be served.")
			return
		}
	}
	if contentNotModified(request, content) {
		writer.WriteHeader(http.StatusNotModified)
		return
	}
	if request.Method == http.MethodHead {
		writer.Header().Set("Content-Length", strconv.FormatInt(content.SizeBytes, 10))
		writer.WriteHeader(http.StatusOK)
		return
	}

	if hasRange && !ifRangeMatches(request.Header.Get("If-Range"), content) {
		hasRange = false
	}
	if !hasRange {
		writer.Header().Set("Content-Length", strconv.FormatInt(content.SizeBytes, 10))
		writer.WriteHeader(http.StatusOK)
		_ = copyContent(request.Context(), writer, content.File, content.SizeBytes)
		return
	}
	if _, err := content.File.Seek(requestedRange.start, io.SeekStart); err != nil {
		writeInternalError(writer, request)
		return
	}
	length := requestedRange.end - requestedRange.start + 1
	writer.Header().Set("Content-Range", fmt.Sprintf(
		"bytes %d-%d/%d",
		requestedRange.start, requestedRange.end, content.SizeBytes,
	))
	writer.Header().Set("Content-Length", strconv.FormatInt(length, 10))
	writer.WriteHeader(http.StatusPartialContent)
	_ = copyContent(request.Context(), writer, content.File, length)
}

func validContentRequestHeaders(request *http.Request) bool {
	return validSingleHeader(request.Header.Values("If-None-Match"), maxETagHeaderBytes) &&
		validSingleHeader(request.Header.Values("If-Modified-Since"), maxDateHeaderBytes) &&
		validSingleHeader(request.Header.Values("If-Range"), maxETagHeaderBytes)
}

func validSingleHeader(values []string, maximum int) bool {
	return len(values) <= 1 && (len(values) == 0 || len(values[0]) <= maximum)
}

func setContentHeaders(header http.Header, content media.Content) {
	header.Set("Accept-Ranges", "bytes")
	header.Set("Content-Type", content.MIMEType)
	header.Set("Content-Disposition", safeInlineDisposition(content.Name))
	header.Set("ETag", content.ETag)
	header.Set("Last-Modified", content.ModifiedAt.UTC().Format(http.TimeFormat))
	header.Set("X-Content-Type-Options", "nosniff")
	header.Set("Cache-Control", "private, no-cache")
}

func safeInlineDisposition(name string) string {
	filename := path.Base(name)
	value := mime.FormatMediaType("inline", map[string]string{"filename": filename})
	if value == "" {
		return "inline"
	}
	return value
}

func contentNotModified(request *http.Request, content media.Content) bool {
	if values := request.Header.Values("If-None-Match"); len(values) > 0 {
		return ifNoneMatch(strings.Join(values, ","), content.ETag)
	}
	value := request.Header.Get("If-Modified-Since")
	if value == "" {
		return false
	}
	since, err := http.ParseTime(value)
	if err != nil {
		return false
	}
	return !content.ModifiedAt.Truncate(time.Second).After(since)
}

func ifNoneMatch(value, current string) bool {
	for _, candidate := range strings.Split(value, ",") {
		candidate = strings.TrimSpace(candidate)
		if candidate == "*" || strings.TrimPrefix(candidate, "W/") == current {
			return true
		}
	}
	return false
}

func ifRangeMatches(value string, content media.Content) bool {
	if value == "" {
		return true
	}
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, `"`) {
		return value == content.ETag
	}
	if strings.HasPrefix(value, "W/") {
		return false
	}
	date, err := http.ParseTime(value)
	if err != nil {
		return false
	}
	return !content.ModifiedAt.Truncate(time.Second).After(date)
}

func parseContentRange(values []string, size int64) (byteRange, bool, error) {
	if len(values) == 0 {
		return byteRange{}, false, nil
	}
	if len(values) != 1 || len(values[0]) == 0 || len(values[0]) > maxRangeHeaderBytes {
		return byteRange{}, true, errors.New("invalid range")
	}
	value := values[0]
	if !strings.HasPrefix(value, "bytes=") || strings.Contains(value, ",") {
		return byteRange{}, true, errors.New("invalid range")
	}
	spec := strings.TrimPrefix(value, "bytes=")
	parts := strings.Split(spec, "-")
	if len(parts) != 2 || (parts[0] == "" && parts[1] == "") || size <= 0 {
		return byteRange{}, true, errors.New("invalid range")
	}
	if parts[0] == "" {
		suffix, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil || suffix <= 0 {
			return byteRange{}, true, errors.New("invalid range")
		}
		if suffix > size {
			suffix = size
		}
		return byteRange{start: size - suffix, end: size - 1}, true, nil
	}
	start, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || start < 0 || start >= size {
		return byteRange{}, true, errors.New("invalid range")
	}
	end := size - 1
	if parts[1] != "" {
		end, err = strconv.ParseInt(parts[1], 10, 64)
		if err != nil || end < start {
			return byteRange{}, true, errors.New("invalid range")
		}
		if end >= size {
			end = size - 1
		}
	}
	return byteRange{start: start, end: end}, true, nil
}

func copyContent(ctx context.Context, writer io.Writer, reader io.Reader, remaining int64) error {
	buffer := make([]byte, 64*1024)
	for remaining > 0 {
		if err := ctx.Err(); err != nil {
			return err
		}
		readSize := int64(len(buffer))
		if remaining < readSize {
			readSize = remaining
		}
		count, readErr := reader.Read(buffer[:readSize])
		if count > 0 {
			written, writeErr := writer.Write(buffer[:count])
			if writeErr != nil {
				return writeErr
			}
			if written != count {
				return io.ErrShortWrite
			}
			remaining -= int64(count)
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) && remaining == 0 {
				return nil
			}
			return readErr
		}
		if count == 0 {
			return io.ErrNoProgress
		}
	}
	return nil
}

func writeContentOpenError(writer http.ResponseWriter, request *http.Request, err error) {
	switch {
	case errors.Is(err, media.ErrContentAssetNotFound):
		writePublicError(writer, request, http.StatusNotFound, "asset_not_found", "The media item was not found.")
	case errors.Is(err, media.ErrContentSourceOffline):
		writePublicError(writer, request, http.StatusConflict, "source_offline", "The media library is offline.")
	case errors.Is(err, media.ErrContentSourceChanged),
		errors.Is(err, media.ErrContentUnavailable):
		code := "source_missing"
		if errors.Is(err, media.ErrContentUnavailable) {
			code = "source_unreadable"
		}
		writePublicError(writer, request, http.StatusConflict, code, "The media source is unavailable.")
	default:
		writeInternalError(writer, request)
	}
}
