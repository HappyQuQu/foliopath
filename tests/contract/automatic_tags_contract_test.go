package contract_test

import (
	"slices"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestAutomaticTagsContractPreservesManualOwnership(t *testing.T) {
	t.Parallel()
	document, err := openapi3.NewLoader().LoadFromFile(openAPIPath(t))
	if err != nil {
		t.Fatal(err)
	}
	path := document.Paths.Find("/api/v1/libraries/{libraryId}/ai/automatic-tags")
	if path == nil || path.Get == nil || path.Put == nil {
		t.Fatal("missing automatic-tag settings contract")
	}
	if path.Put.Security == nil || path.Put.Responses.Status(412) == nil || path.Put.Responses.Status(428) == nil {
		t.Fatal("write must protect CSRF and revisions")
	}
	jobs := document.Paths.Find("/api/v1/libraries/{libraryId}/ai/automatic-tags/jobs")
	if jobs == nil || jobs.Post == nil || jobs.Post.Responses.Status(202) == nil || jobs.Post.Security == nil {
		t.Fatal("missing authenticated async job")
	}
	settings := requiredComponentSchema(t, document, "AutomaticTagLibrarySettings")
	for _, field := range []string{"enabled", "state", "model", "coverage", "activeOperationId", "offline", "revision"} {
		if !slices.Contains(settings.Required, field) {
			t.Errorf("missing %s", field)
		}
	}
	assetTag := requiredComponentSchema(t, document, "AssetTag")
	if !slices.Contains(assetTag.Required, "source") || len(assetTag.Properties["source"].Value.Enum) != 2 {
		t.Fatal("missing per-image tag source")
	}
	globalTag := requiredComponentSchema(t, document, "Tag")
	if globalTag.Properties["source"] != nil {
		t.Fatal("global vocabulary cannot have one association source")
	}
	curation := requiredComponentSchema(t, document, "AssetCuration")
	if curation.Properties["tags"].Value.Items.Ref != "#/components/schemas/AssetTag" {
		t.Fatal("curation must return source")
	}
}
