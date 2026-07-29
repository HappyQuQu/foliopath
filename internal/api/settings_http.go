package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"

	appsettings "github.com/HappyQuQu/foliopath/internal/settings"
)

type settingsResponse struct {
	ScheduledScanIntervalHours *int64 `json:"scheduledScanIntervalHours"`
	AutomaticDiscoveryEnabled  bool   `json:"automaticDiscoveryEnabled"`
	ThumbnailCacheQuotaBytes   int64  `json:"thumbnailCacheQuotaBytes"`
	Language                   string `json:"language"`
	UpdatedAt                  string `json:"updatedAt"`
}

var errInvalidSettingsRequest = errors.New("invalid settings request")

func registerSettingsRoutes(mux *http.ServeMux, service SettingsService) {
	mux.HandleFunc("GET /api/v1/settings", func(writer http.ResponseWriter, request *http.Request) {
		values, err := service.Get(request.Context())
		if err != nil {
			writeInternalError(writer, request)
			return
		}
		writer.Header().Set("ETag", settingsETag(values.Revision))
		writeJSON(writer, http.StatusOK, settingsWire(values))
	})
	mux.HandleFunc("PATCH /api/v1/settings", func(writer http.ResponseWriter, request *http.Request) {
		revision, err := parseSettingsIfMatch(request.Header.Get("If-Match"))
		if err != nil {
			writeSettingsError(writer, request, err)
			return
		}
		update, err := decodeSettingsUpdate(writer, request)
		if err != nil {
			writeSettingsError(writer, request, err)
			return
		}
		values, err := service.Update(request.Context(), revision, update)
		if err != nil {
			writeSettingsError(writer, request, err)
			return
		}
		writer.Header().Set("ETag", settingsETag(values.Revision))
		writeJSON(writer, http.StatusOK, settingsWire(values))
	})
}

func decodeSettingsUpdate(writer http.ResponseWriter, request *http.Request) (appsettings.Update, error) {
	contentType, _, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if err != nil || contentType != "application/json" {
		return appsettings.Update{}, errInvalidSettingsRequest
	}
	request.Body = http.MaxBytesReader(writer, request.Body, maxLibraryRequestBodyBytes)
	body, err := io.ReadAll(request.Body)
	if err != nil {
		return appsettings.Update{}, errInvalidSettingsRequest
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil || len(fields) == 0 {
		return appsettings.Update{}, errInvalidSettingsRequest
	}
	var update appsettings.Update
	for name, raw := range fields {
		switch name {
		case "scheduledScanIntervalHours":
			update.SetSchedule = true
			if !bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
				var value int64
				if json.Unmarshal(raw, &value) != nil {
					return appsettings.Update{}, errInvalidSettingsRequest
				}
				update.ScheduledScanIntervalHours = &value
			}
		case "thumbnailCacheQuotaBytes":
			var value int64
			if json.Unmarshal(raw, &value) != nil {
				return appsettings.Update{}, errInvalidSettingsRequest
			}
			update.ThumbnailCacheQuotaBytes = &value
		case "automaticDiscoveryEnabled":
			var value bool
			if json.Unmarshal(raw, &value) != nil {
				return appsettings.Update{}, errInvalidSettingsRequest
			}
			update.AutomaticDiscoveryEnabled = &value
		case "language":
			var value string
			if json.Unmarshal(raw, &value) != nil {
				return appsettings.Update{}, errInvalidSettingsRequest
			}
			update.Language = &value
		default:
			return appsettings.Update{}, errInvalidSettingsRequest
		}
	}
	return update, nil
}

func parseSettingsIfMatch(value string) (int64, error) {
	if value == "" {
		return 0, errPreconditionRequired
	}
	if !strings.HasPrefix(value, `"settings-r`) || !strings.HasSuffix(value, `"`) {
		return 0, appsettings.ErrPreconditionFailed
	}
	revision, err := strconv.ParseInt(strings.TrimSuffix(strings.TrimPrefix(value, `"settings-r`), `"`), 10, 64)
	if err != nil || revision <= 0 {
		return 0, appsettings.ErrPreconditionFailed
	}
	return revision, nil
}

func settingsETag(revision int64) string {
	return `"settings-r` + strconv.FormatInt(revision, 10) + `"`
}

func settingsWire(values appsettings.Values) settingsResponse {
	return settingsResponse{
		ScheduledScanIntervalHours: values.ScheduledScanIntervalHours,
		AutomaticDiscoveryEnabled:  values.AutomaticDiscoveryEnabled,
		ThumbnailCacheQuotaBytes:   values.ThumbnailCacheQuotaBytes,
		Language:                   values.Language,
		UpdatedAt:                  timestamp(values.UpdatedAtMS),
	}
}

func writeSettingsError(writer http.ResponseWriter, request *http.Request, err error) {
	switch {
	case errors.Is(err, errPreconditionRequired):
		writePublicError(writer, request, http.StatusPreconditionRequired, "precondition_required", "A current resource validator is required.")
	case errors.Is(err, appsettings.ErrPreconditionFailed):
		writePublicError(writer, request, http.StatusPreconditionFailed, "precondition_failed", "The settings have changed.")
	case errors.Is(err, appsettings.ErrInvalid):
		writePublicError(writer, request, http.StatusUnprocessableEntity, "settings_invalid", "One or more settings are invalid.")
	default:
		if errors.Is(err, errInvalidSettingsRequest) {
			writePublicError(writer, request, http.StatusBadRequest, codeInvalidRequest, "The request is invalid.")
			return
		}
		writeInternalError(writer, request)
	}
}
