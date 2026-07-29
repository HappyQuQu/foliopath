package contract_test

import (
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestAutomaticDiscoveryOpenAPIContract(t *testing.T) {
	t.Parallel()

	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = false
	document, err := loader.LoadFromFile(openAPIPath(t))
	if err != nil {
		t.Fatalf("load OpenAPI: %v", err)
	}

	settings := requiredComponentSchema(t, document, "Settings")
	settingsUpdate := requiredComponentSchema(t, document, "SettingsUpdate")
	for name, schema := range map[string]*openapi3.Schema{
		"Settings":       settings,
		"SettingsUpdate": settingsUpdate,
	} {
		field, ok := schema.Properties["automaticDiscoveryEnabled"]
		if !ok || field == nil || field.Value == nil || field.Value.Type == nil ||
			!field.Value.Type.Is("boolean") {
			t.Errorf("%s.automaticDiscoveryEnabled must be a boolean", name)
		}
	}
	if !contains(settings.Required, "automaticDiscoveryEnabled") {
		t.Error("Settings.automaticDiscoveryEnabled must be required")
	}

	library := requiredComponentSchema(t, document, "Library")
	for _, field := range []string{
		"automaticDiscoveryStatus",
		"automaticDiscoveryErrorCode",
		"lastAutomaticDiscoveryAt",
		"contentRevision",
	} {
		if _, ok := library.Properties[field]; !ok {
			t.Errorf("Library is missing %q", field)
		}
		if !contains(library.Required, field) {
			t.Errorf("Library.%s must be required", field)
		}
	}
	assertEnum(
		t,
		requiredComponentSchema(t, document, "AutomaticDiscoveryStatus"),
		"active",
		"degraded",
		"unsupported",
		"disabled",
	)
	assertEnum(
		t,
		requiredComponentSchema(t, document, "AutomaticDiscoveryErrorCode"),
		"watch_unavailable",
		"watch_resource_limit",
		"watch_overflow",
		"source_unavailable",
		"internal_error",
	)

	state := requiredComponentSchema(t, document, "CatalogState")
	if !contains(state.Required, "contentRevision") {
		t.Error("CatalogState.contentRevision must be required")
	}
	operation := document.Paths.Find("/api/v1/catalog/state")
	if operation == nil || operation.Get == nil {
		t.Fatal("GET /api/v1/catalog/state is missing")
	}
	if _, ok := operation.Get.Responses.Map()["304"]; !ok {
		t.Error("GET /api/v1/catalog/state must support 304")
	}
}
