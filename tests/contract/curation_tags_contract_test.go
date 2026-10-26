package contract_test

import (
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestTagListOpenAPIHasBackwardCompatibleLibraryAndKindScope(t *testing.T) {
	t.Parallel()
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = false
	document, err := loader.LoadFromFile(openAPIPath(t))
	if err != nil {
		t.Fatal(err)
	}
	path := document.Paths.Find("/api/v1/tags")
	if path == nil || path.Get == nil {
		t.Fatal("GET /api/v1/tags is missing")
	}
	operation := path.Get
	want := map[string]bool{"q": false, "libraryId": false, "kind": false, "cursor": false, "limit": false}
	for _, reference := range operation.Parameters {
		parameter := reference.Value
		if parameter == nil || parameter.In != "query" || parameter.Required {
			t.Fatalf("tag-list parameter must be optional query scope: %#v", parameter)
		}
		if seen, ok := want[parameter.Name]; !ok || seen {
			t.Fatalf("unexpected or duplicate parameter %q", parameter.Name)
		}
		want[parameter.Name] = true
		if parameter.Name == "kind" && (parameter.Style != "form" || parameter.Explode == nil || *parameter.Explode || parameter.Schema == nil || parameter.Schema.Value == nil || parameter.Schema.Value.Items == nil) {
			t.Fatal("kind must reuse comma-separated media-kind array semantics")
		}
	}
	for name, present := range want {
		if !present {
			t.Errorf("optional tag scope %q is missing", name)
		}
	}
	if operation.Responses.Status(404) == nil {
		t.Error("unknown library must have an explicit 404 response")
	}
	for _, semantics := range []string{"including", "no associated assets", "distinct asset", "kind=image,animated", "curation", "never fall back globally"} {
		if !strings.Contains(operation.Description, semantics) {
			t.Errorf("tag-list contract does not document %q", semantics)
		}
	}
	page := requiredComponentSchema(t, document, "TagPage")
	if page.Properties["items"].Value.MaxItems == nil || *page.Properties["items"].Value.MaxItems != 200 {
		t.Error("tag lists must remain bounded to 200 items")
	}
}

func TestTagLocalizationContractIsAdditiveAndKeepsManualNames(t *testing.T) {
	loader := openapi3.NewLoader()
	document, err := loader.LoadFromFile(openAPIPath(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Tag", "AssetTag"} {
		schema := requiredComponentSchema(t, document, name)
		if schema.Properties["localization"] == nil {
			t.Fatal(name + " missing reviewed projection")
		}
		for _, required := range schema.Required {
			if required == "localization" {
				t.Fatal("legacy clients require optional metadata")
			}
		}
	}
	metadata := requiredComponentSchema(t, document, "TagLocalization")
	for _, field := range []string{"key", "zhCN", "en"} {
		if metadata.Properties[field] == nil {
			t.Fatal(field)
		}
	}
	if !strings.Contains(metadata.Description, "manual") || !strings.Contains(metadata.Description, "browser preference") {
		t.Fatal("scope/protection missing")
	}
}
