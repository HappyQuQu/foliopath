package contract_test

import (
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestStoryboardOpenAPIContract(t *testing.T) {
	t.Parallel()

	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = false
	document, err := loader.LoadFromFile(openAPIPath(t))
	if err != nil {
		t.Fatalf("load OpenAPI: %v", err)
	}

	variant := requiredParameterSchema(t, document, "ThumbnailVariantParameter")
	assertEnum(t, variant, "grid", "storyboard")

	storyboard := requiredComponentSchema(t, document, "StoryboardReference")
	for _, field := range []string{
		"status",
		"url",
		"frameCount",
		"columns",
		"rows",
		"cellWidth",
		"cellHeight",
		"errorCode",
	} {
		if _, ok := storyboard.Properties[field]; !ok {
			t.Errorf("StoryboardReference is missing %q", field)
		}
	}
	assertEnum(
		t,
		storyboard.Properties["status"].Value,
		"pending",
		"ready",
		"failed",
		"unavailable",
		"not_applicable",
	)
	assertIntegerBounds(t, storyboard.Properties["frameCount"].Value, 4, 10)
	assertIntegerBounds(t, storyboard.Properties["columns"].Value, 1, 5)
	assertIntegerBounds(t, storyboard.Properties["rows"].Value, 1, 2)
	assertIntegerBounds(t, storyboard.Properties["cellWidth"].Value, 1, 320)
	assertIntegerBounds(t, storyboard.Properties["cellHeight"].Value, 1, 320)

	asset := requiredComponentSchema(t, document, "Asset")
	if _, ok := asset.Properties["storyboard"]; !ok {
		t.Fatal("Asset is missing storyboard")
	}
	if !contains(asset.Required, "storyboard") {
		t.Fatal("Asset.storyboard must be required so consumers always handle its state")
	}
}

func requiredParameterSchema(
	t *testing.T,
	document *openapi3.T,
	name string,
) *openapi3.Schema {
	t.Helper()

	ref, ok := document.Components.Parameters[name]
	if !ok || ref == nil || ref.Value == nil || ref.Value.Schema == nil || ref.Value.Schema.Value == nil {
		t.Fatalf("parameter %s has no schema", name)
	}
	return ref.Value.Schema.Value
}

func requiredComponentSchema(
	t *testing.T,
	document *openapi3.T,
	name string,
) *openapi3.Schema {
	t.Helper()

	ref, ok := document.Components.Schemas[name]
	if !ok || ref == nil || ref.Value == nil {
		t.Fatalf("schema %s is missing", name)
	}
	return ref.Value
}

func assertEnum(t *testing.T, schema *openapi3.Schema, want ...string) {
	t.Helper()

	got := make(map[string]bool, len(schema.Enum))
	for _, value := range schema.Enum {
		text, ok := value.(string)
		if !ok {
			t.Fatalf("enum contains non-string value %#v", value)
		}
		got[text] = true
	}
	if len(got) != len(want) {
		t.Fatalf("enum = %#v, want %v", schema.Enum, want)
	}
	for _, value := range want {
		if !got[value] {
			t.Errorf("enum is missing %q", value)
		}
	}
}

func assertIntegerBounds(t *testing.T, schema *openapi3.Schema, minimum, maximum int64) {
	t.Helper()

	if schema.Min == nil || int64(*schema.Min) != minimum {
		t.Errorf("minimum = %v, want %d", schema.Min, minimum)
	}
	if schema.Max == nil || int64(*schema.Max) != maximum {
		t.Errorf("maximum = %v, want %d", schema.Max, maximum)
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
