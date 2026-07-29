package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	appsettings "github.com/HappyQuQu/foliopath/internal/settings"
)

type settingsServiceStub struct {
	values appsettings.Values
	update appsettings.Update
	err    error
}

func (stub *settingsServiceStub) Get(context.Context) (appsettings.Values, error) {
	return stub.values, stub.err
}

func (stub *settingsServiceStub) Update(
	_ context.Context,
	revision int64,
	update appsettings.Update,
) (appsettings.Values, error) {
	stub.update = update
	if revision != stub.values.Revision {
		return appsettings.Values{}, appsettings.ErrPreconditionFailed
	}
	stub.values.Revision++
	stub.values.ScheduledScanIntervalHours = update.ScheduledScanIntervalHours
	return stub.values, stub.err
}

func TestSettingsRoutesGetAndDisableSchedule(t *testing.T) {
	hours := int64(24)
	service := &settingsServiceStub{values: appsettings.Values{
		ScheduledScanIntervalHours: &hours,
		AutomaticDiscoveryEnabled:  true,
		ThumbnailCacheQuotaBytes:   10_737_418_240,
		Language:                   "browser",
		Revision:                   1,
		UpdatedAtMS:                1_000,
	}}
	mux := http.NewServeMux()
	registerSettingsRoutes(mux, service)
	get := httptest.NewRecorder()
	mux.ServeHTTP(get, httptest.NewRequest(http.MethodGet, "/api/v1/settings", nil))
	if get.Code != http.StatusOK || get.Header().Get("ETag") != `"settings-r1"` {
		t.Fatalf("get settings = %d %q %s", get.Code, get.Header().Get("ETag"), get.Body.String())
	}
	request := httptest.NewRequest(
		http.MethodPatch,
		"/api/v1/settings",
		strings.NewReader(`{"scheduledScanIntervalHours":null}`),
	)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("If-Match", `"settings-r1"`)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusOK ||
		response.Header().Get("ETag") != `"settings-r2"` ||
		!service.update.SetSchedule ||
		service.update.ScheduledScanIntervalHours != nil {
		t.Fatalf("disable schedule = %d %q %#v %s", response.Code, response.Header().Get("ETag"), service.update, response.Body.String())
	}
}

func TestSettingsRoutesUpdateAutomaticDiscovery(t *testing.T) {
	service := &settingsServiceStub{values: appsettings.Values{
		AutomaticDiscoveryEnabled: true,
		Revision:                  1,
	}}
	mux := http.NewServeMux()
	registerSettingsRoutes(mux, service)
	request := httptest.NewRequest(
		http.MethodPatch,
		"/api/v1/settings",
		strings.NewReader(`{"automaticDiscoveryEnabled":false}`),
	)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("If-Match", `"settings-r1"`)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusOK ||
		service.update.AutomaticDiscoveryEnabled == nil ||
		*service.update.AutomaticDiscoveryEnabled {
		t.Fatalf("automatic discovery update = %d %#v", response.Code, service.update)
	}
}

func TestSettingsRoutesRequireValidatorAndRejectUnknownFields(t *testing.T) {
	service := &settingsServiceStub{values: appsettings.Values{Revision: 1}}
	mux := http.NewServeMux()
	registerSettingsRoutes(mux, service)
	missing := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPatch, "/api/v1/settings", strings.NewReader(`{"language":"en"}`))
	request.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(missing, request)
	if missing.Code != http.StatusPreconditionRequired {
		t.Fatalf("missing validator status = %d", missing.Code)
	}
	unknown := httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPatch, "/api/v1/settings", strings.NewReader(`{"unknown":1}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("If-Match", `"settings-r1"`)
	mux.ServeHTTP(unknown, request)
	if unknown.Code != http.StatusBadRequest {
		t.Fatalf("unknown field status = %d, body %s", unknown.Code, unknown.Body.String())
	}
}
