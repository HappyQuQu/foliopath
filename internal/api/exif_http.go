package api

import (
	"context"
	"errors"
	"net/http"

	"github.com/HappyQuQu/foliopath/internal/media"
)

type exifFieldResponse struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type exifResponse struct {
	Fields []exifFieldResponse `json:"fields"`
}

type ExifService interface {
	Read(context.Context, int64) (media.ExifResult, error)
}

func registerExifRoutes(mux *http.ServeMux, service ExifService, admission ContentAdmission) {
	gate := &contentHandler{slots: make(chan struct{}, contentReadConcurrency), admission: admission}
	mux.HandleFunc("GET /api/v1/assets/{assetId}/exif", func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.RawQuery != "" {
			writePublicError(writer, request, http.StatusBadRequest, "invalid_request", "The request is invalid.")
			return
		}
		id, err := parseResourceID(request.PathValue("assetId"), "ast_")
		if err != nil {
			writeContentOpenError(writer, request, media.ErrContentAssetNotFound)
			return
		}
		release, ok := gate.tryAcquire()
		if !ok {
			writer.Header().Set("Retry-After", "1")
			writePublicError(writer, request, http.StatusTooManyRequests, "rate_limited", "Too many media requests are active.")
			return
		}
		defer release()
		writer.Header().Set("Cache-Control", "no-store")
		result, err := service.Read(request.Context(), id)
		if err != nil {
			writeExifError(writer, request, err)
			return
		}
		fields := make([]exifFieldResponse, 0, len(result.Fields))
		for _, field := range result.Fields {
			fields = append(fields, exifFieldResponse{Key: field.Key, Value: field.Value})
		}
		writeJSON(writer, http.StatusOK, exifResponse{Fields: fields})
	})
}

func writeExifError(writer http.ResponseWriter, request *http.Request, err error) {
	switch {
	case errors.Is(err, context.Canceled):
		return
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, media.ErrProcessingTimedOut):
		writePublicError(writer, request, http.StatusUnprocessableEntity, "media_processing_timeout", "The media information could not be read in time.")
	case errors.Is(err, media.ErrInvalidMedia):
		writePublicError(writer, request, http.StatusUnprocessableEntity, "invalid_media", "The media information could not be read.")
	case errors.Is(err, media.ErrUnsupportedMedia), errors.Is(err, media.ErrSourceTooLarge):
		writePublicError(writer, request, http.StatusUnprocessableEntity, "unsupported_media", "The media information is unavailable for this file.")
	default:
		writeContentOpenError(writer, request, err)
	}
}
