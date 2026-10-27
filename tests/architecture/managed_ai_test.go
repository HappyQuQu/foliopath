package architecture_test

import (
	"github.com/oasdiff/yaml"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestManagedAIComposeKeepsTwoServicesAndMountBoundaries(t *testing.T) {
	root := repositoryRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "compose.ai.simple.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		Services map[string]struct {
			Image       string
			Ports       []string
			Volumes     []string
			Environment map[string]string
			DependsOn   any `json:"depends_on"`
		}
	}
	if _, err := yaml.Unmarshal(data, &config, yaml.DecodeOpts{}); err != nil {
		t.Fatal(err)
	}
	app, ai := config.Services["foliopath"], config.Services["machine-learning"]
	if len(config.Services) != 2 || app.Image == "" || !strings.HasSuffix(ai.Image, "-ai") || len(ai.Ports) != 0 || app.DependsOn != nil {
		t.Fatal("two independent services; only app publishes HTTP")
	}
	if app.Environment["FOLIOPATH_AI_PROFILE"] != "standard-v1" || len(app.Volumes) != 2 || !strings.HasSuffix(app.Volumes[0], ":/library:ro") || !strings.HasSuffix(app.Volumes[1], ":/app/data") {
		t.Fatal("managed profile, read-only media and dedicated data required")
	}
	targets := map[string]bool{}
	for _, volume := range ai.Volumes {
		parts := strings.Split(volume, ":")
		if len(parts) != 2 || !strings.HasPrefix(parts[0], "/root/foliopath/models/") {
			t.Fatal("AI only owns dedicated model directories")
		}
		targets[parts[1]] = true
	}
	if len(targets) != 3 || !targets["/cache"] || !targets["/ocr-cache"] || !targets["/verified-cache"] {
		t.Fatal("cache and scratch boundaries")
	}
	command := exec.Command("python3", "scripts/generate_managed_ai.py", "--check")
	command.Dir = root
	if out, err := command.CombinedOutput(); err != nil {
		t.Fatalf("generated profile stale: %v %s", err, out)
	}
}
