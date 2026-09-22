package architecture_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/oasdiff/yaml"
)

func TestAIComposeDeploymentBoundaries(t *testing.T) {
	content, err := os.ReadFile(filepath.Join(repositoryRoot(t), "compose.ai.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		Services map[string]struct {
			Image      string
			ReadOnly   bool     `json:"read_only"`
			PullPolicy string   `json:"pull_policy"`
			CapDrop    []string `json:"cap_drop"`
			Networks   []string
			Ports      []string
			DependsOn  any `json:"depends_on"`
			Volumes    []struct {
				Target   string
				ReadOnly bool `json:"read_only"`
				Bind     struct {
					CreateHostPath bool `json:"create_host_path"`
				}
			}
		}
		Networks map[string]struct{ Internal bool }
	}
	if _, err := yaml.Unmarshal(content, &config, yaml.DecodeOpts{}); err != nil {
		t.Fatal(err)
	}
	app, appOK := config.Services["foliopath"]
	ai, aiOK := config.Services["machine-learning"]
	if len(config.Services) != 2 || !appOK || !aiOK {
		t.Fatal("deployment must contain exactly the application and inference service")
	}
	if app.DependsOn != nil || len(ai.Ports) != 0 || len(ai.Networks) != 1 || ai.Networks[0] != "inference" || !config.Networks["inference"].Internal {
		t.Fatal("AI must remain internal and must not gate core startup")
	}
	if len(ai.Volumes) != 1 || ai.Volumes[0].Target != "/cache" || !ai.Volumes[0].ReadOnly {
		t.Fatal("AI may only mount the read-only cache")
	}
	if len(app.Volumes) != 2 || app.Volumes[0].Target != "/library" || !app.Volumes[0].ReadOnly || app.Volumes[1].Target != "/app/data" {
		t.Fatal("application must use one read-only media root and dedicated state")
	}
	for name, service := range config.Services {
		if !service.ReadOnly || service.PullPolicy != "never" || len(service.CapDrop) != 1 || service.CapDrop[0] != "ALL" {
			t.Errorf("%s lost its read-only, capability, or offline boundary", name)
		}
		for _, volume := range service.Volumes {
			if volume.Bind.CreateHostPath {
				t.Errorf("%s must not silently create missing mount sources", name)
			}
		}
	}
}
