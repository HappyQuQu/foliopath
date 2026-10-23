package architecture_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

func TestNode24LTSToolchainsStayAligned(t *testing.T) {
	read := func(path string) []byte {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(repositoryRoot(t), path))
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	version := strings.TrimSpace(string(read(".node-version")))
	if !regexp.MustCompile(`^24\.\d+\.\d+$`).MatchString(version) {
		t.Fatal("pin a verified Node 24 LTS patch")
	}
	var manifest struct {
		Engines        map[string]string `json:"engines"`
		PackageManager string            `json:"packageManager"`
	}
	if err := json.Unmarshal(read("web/package.json"), &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Engines["node"] != ">="+version+" <25" {
		t.Fatal("Node engine differs from the canonical pin")
	}
	if !regexp.MustCompile(`^npm@11\.\d+\.\d+$`).MatchString(manifest.PackageManager) {
		t.Fatal("pin the verified npm 11 toolchain")
	}
	npm := strings.TrimPrefix(manifest.PackageManager, "npm@")
	if manifest.Engines["npm"] != ">="+npm+" <12" {
		t.Fatal("npm engine differs from packageManager")
	}
	var lock struct {
		Packages map[string]struct {
			Engines map[string]string `json:"engines"`
		} `json:"packages"`
	}
	if err := json.Unmarshal(read("web/package-lock.json"), &lock); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(manifest.Engines, lock.Packages[""].Engines) {
		t.Fatal("regenerate root lockfile engine metadata")
	}
	image := regexp.MustCompile(`(?m)^FROM node:(\d+\.\d+\.\d+)-bookworm-slim@sha256:([0-9a-f]{64}) AS web$`)
	var digest string
	for _, path := range []string{"Dockerfile", "spikes/automatic-tags/Dockerfile.localized-app"} {
		match := image.FindSubmatch(read(path))
		if len(match) != 3 || string(match[1]) != version {
			t.Fatal("Docker Node pin/version mismatch: " + path)
		}
		if digest != "" && digest != string(match[2]) {
			t.Fatal("Docker build stages must share a verified manifest")
		}
		digest = string(match[2])
	}
	workflow := string(read(".github/workflows/intelligent-media-native.yml"))
	if !strings.Contains(workflow, "node-version-file: .node-version") || regexp.MustCompile(`(?m)^\s+node-version:`).MatchString(workflow) {
		t.Fatal("CI must consume .node-version, not a duplicate version")
	}
	browser := string(read("spikes/int001-remote-ml/browser-test.Dockerfile"))
	if !strings.Contains(browser, "COPY .node-version") || !strings.Contains(browser, `test "$(node --version)"`) {
		t.Fatal("browser image must reject a stale local base toolchain")
	}
}
