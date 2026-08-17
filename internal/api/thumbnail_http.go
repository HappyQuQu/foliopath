package api

import (
	"context"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/HappyQuQu/foliopath/internal/media"
	"github.com/HappyQuQu/foliopath/internal/thumbnail"
)

type ThumbnailService interface {
	Get(context.Context, int64, thumbnail.Variant) (thumbnail.Delivery, error)
}

type thumbnailPendingResponse struct {
	AssetID      string                   `json:"assetId"`
	Variant      string                   `json:"variant"`
	Status       thumbnail.DeliveryStatus `json:"status"`
	RetryAfterMS int                      `json:"retryAfterMs"`
}

func registerThumbnailRoute(mux *http.ServeMux, service ThumbnailService) {
	mux.HandleFunc("GET /api/v1/assets/{assetId}/thumbnail", func(
		writer http.ResponseWriter,
		request *http.Request,
	) {
		assetID, err := parseResourceID(request.PathValue("assetId"), "ast_")
		if err != nil {
			writeThumbnailError(writer, request, thumbnail.ErrAssetNotFound)
			return
		}
		variant, err := parseThumbnailVariant(request.URL.RawQuery)
		if err != nil {
			writePublicError(
				writer, request, http.StatusBadRequest,
				codeInvalidRequest, "The request is invalid.",
			)
			return
		}
		delivery, err := service.Get(request.Context(), assetID, variant)
		if err != nil {
			writeThumbnailError(writer, request, err)
			return
		}
		switch delivery.Status {
		case thumbnail.DeliveryQueued, thumbnail.DeliveryRunning:
			seconds := (delivery.RetryAfterMS + 999) / 1000
			writer.Header().Set("Retry-After", strconv.Itoa(seconds))
			writeJSON(writer, http.StatusAccepted, thumbnailPendingResponse{
				AssetID: assetIDString(assetID), Variant: string(variant),
				Status: delivery.Status, RetryAfterMS: delivery.RetryAfterMS,
			})
		case thumbnail.DeliveryOffline:
			writeThumbnailStateError(
				writer, request, http.StatusConflict, delivery.ErrorCode,
			)
		case thumbnail.DeliveryFailed:
			writeThumbnailStateError(
				writer, request, http.StatusUnprocessableEntity, delivery.ErrorCode,
			)
		case thumbnail.DeliveryReady:
			writeReadyThumbnail(writer, request, delivery)
		default:
			writeInternalError(writer, request)
		}
	})
}

func parseThumbnailVariant(raw string) (thumbnail.Variant, error) {
	values, err := url.ParseQuery(raw)
	if err != nil {
		return "", err
	}
	variant := thumbnail.VariantGrid
	for key, entries := range values {
		if len(entries) != 1 {
			return "", errors.New("invalid thumbnail query")
		}
		switch key {
		case "variant":
			variant = thumbnail.Variant(entries[0])
		case "v":
			if !validThumbnailVersion(entries[0]) {
				return "", errors.New("invalid thumbnail query")
			}
		default:
			return "", errors.New("invalid thumbnail query")
		}
	}
	if variant != thumbnail.VariantGrid &&
		variant != thumbnail.VariantStoryboard {
		return "", errors.New("invalid thumbnail query")
	}
	return variant, nil
}

func validThumbnailVersion(value string) bool {
	if len(value) != 32 {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == 16
}

func writeReadyThumbnail(
	writer http.ResponseWriter,
	request *http.Request,
	delivery thumbnail.Delivery,
) {
	if delivery.Content == nil || delivery.ContentBytes <= 0 || delivery.ETag == "" {
		if delivery.Content != nil {
			_ = delivery.Content.Close()
		}
		writeInternalError(writer, request)
		return
	}
	defer delivery.Content.Close()
	setThumbnailHeaders(writer, delivery)
	if etagMatches(request.Header.Get("If-None-Match"), delivery.ETag) {
		writer.WriteHeader(http.StatusNotModified)
		return
	}
	writer.Header().Set("Content-Type", "image/webp")
	writer.Header().Set("Content-Length", strconv.FormatInt(delivery.ContentBytes, 10))
	writer.WriteHeader(http.StatusOK)
	_, _ = io.CopyN(writer, delivery.Content, delivery.ContentBytes)
}

func setThumbnailHeaders(writer http.ResponseWriter, delivery thumbnail.Delivery) {
	writer.Header().Set("ETag", delivery.ETag)
	writer.Header().Set("Cache-Control", "private, max-age=31536000, immutable")
	writer.Header().Set("X-Content-Type-Options", "nosniff")
	writer.Header().Set("Content-Security-Policy", "default-src 'none'; sandbox")
}

func etagMatches(header, current string) bool {
	for _, candidate := range strings.Split(header, ",") {
		candidate = strings.TrimSpace(candidate)
		if candidate == "*" || candidate == current ||
			strings.TrimPrefix(candidate, "W/") == current {
			return true
		}
	}
	return false
}

func writeThumbnailStateError(
	writer http.ResponseWriter,
	request *http.Request,
	status int,
	code media.ProcessingErrorCode,
) {
	switch string(code) {
	case "source_offline":
		writePublicError(writer, request, status, "source_offline", "The media library is offline.")
	case string(media.ErrorUnsupportedMedia):
		writePublicError(writer, request, status, string(code), "The media format is not supported.")
	case string(media.ErrorInvalidMedia):
		writePublicError(writer, request, status, string(code), "The media file is invalid.")
	case string(media.ErrorProcessingFailed):
		writePublicError(writer, request, status, "thumbnail_failed", "Media processing failed.")
	case string(media.ErrorProcessingTimed):
		writePublicError(writer, request, status, string(code), "Media processing timed out.")
	default:
		writeInternalError(writer, request)
	}
}

func writeThumbnailError(writer http.ResponseWriter, request *http.Request, err error) {
	if errors.Is(err, thumbnail.ErrAssetNotFound) {
		writePublicError(writer, request, http.StatusNotFound, "asset_not_found", "The media item was not found.")
		return
	}
	if errors.Is(err, thumbnail.ErrStoryboardNotEligible) {
		writePublicError(
			writer,
			request,
			http.StatusNotFound,
			string(media.ErrorUnsupportedMedia),
			"The thumbnail variant does not apply to this media item.",
		)
		return
	}
	writeInternalError(writer, request)
}
