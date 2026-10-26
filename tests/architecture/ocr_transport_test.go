package architecture_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-0032/0033/0035 accept transport, input and dormant storage, not runtime composition.
// Broaden this allowlist only with the later OCR backend gate and its evidence.
func TestOCRHasOnlyApprovedConsumers(t *testing.T) {
	command := exec.Command("go", "list", "-json", "./internal/...")
	command.Dir = repositoryRoot(t)
	output, err := command.Output()
	if err != nil {
		t.Fatal(err)
	}
	decoder := json.NewDecoder(strings.NewReader(string(output)))
	found := false
	for decoder.More() {
		var current listedPackage
		if err := decoder.Decode(&current); err != nil {
			t.Fatal(err)
		}
		for _, imported := range current.Imports {
			if imported != modulePath+"/internal/ocr" {
				continue
			}
			if current.ImportPath != modulePath+"/internal/inference/remoteml" && current.ImportPath != modulePath+"/internal/media/imagevips" && current.ImportPath != modulePath+"/internal/store/sqlite" && current.ImportPath != modulePath+"/internal/app" && current.ImportPath != modulePath+"/internal/api" {
				t.Errorf("OCR production consumer requires a later backend gate: %s", current.ImportPath)
			}
			found = true
		}
	}
	if !found {
		t.Fatal("approved OCR transport consumer missing")
	}
}

// Storage may only mutate its own derived tables. Runtime composition is allowed only in app by ADR-0036; API calls services.
func TestOCRPersistenceWritesOnlyItsOwnTables(t *testing.T) {
	root := repositoryRoot(t)
	for _, name := range []string{"ocr_settings.go", "ocr_jobs.go", "ocr_results.go", "ocr_runtime.go", "ocr_search.go"} {
		source, err := os.ReadFile(filepath.Join(root, "internal", "store", "sqlite", name))
		if err != nil {
			t.Fatal(err)
		}
		for _, table := range []string{"assets", "libraries", "asset_tags", "asset_tag_suppressions", "people", "person_face_anchors", "semantic_embeddings", "face_observations", "ai_model_operations", "autotag_jobs"} {
			for _, verb := range []string{"INSERT INTO ", "UPDATE ", "DELETE FROM "} {
				if strings.Contains(strings.ToLower(string(source)), strings.ToLower(verb+table+" ")) {
					t.Errorf("OCR store may not mutate %s: %s", table, name)
				}
			}
		}
	}
}
