package architecture_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalizedTagsHaveOneCatalogAndSeparateCapabilityIdentity(t *testing.T) {
	read := func(name string) string {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(repositoryRoot(t), name))
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	var categories map[string][]string
	if err := json.Unmarshal([]byte(read("internal/curation/tag_categories.json")), &categories); err != nil {
		t.Fatal(err)
	}
	if len(categories) <= 92 || len(categories) > 256 {
		t.Fatal("unreviewed catalog bounds")
	}
	for _, name := range []string{"services/inference/Dockerfile.photo-tags", "internal/curation/tag_localization.go"} {
		if !strings.Contains(read(name), "tag_categories.json") {
			t.Fatal("lost canonical catalog: " + name)
		}
	}
	policy := read("services/inference/photo_policy.py")
	if !strings.Contains(policy, "SAFE_LABELS = frozenset(categories())") || !strings.Contains(policy, "policy_sha256") {
		t.Fatal("uncoupled or duplicate category policy")
	}
	worker := read("services/inference/photo_worker.py")
	if !strings.Contains(worker, "inference_worker.tag_contract = photo_policy") || !strings.Contains(worker, "os.execv") {
		t.Fatal("tag overlay must reuse transforms and exec unchanged non-tag worker")
	}
	server := read("services/inference/photo_server.py")
	if strings.Contains(server, "InferenceScheduler(") || !strings.Contains(server, "legacy.scheduler.command") {
		t.Fatal("overlay created another scheduler")
	}
}
