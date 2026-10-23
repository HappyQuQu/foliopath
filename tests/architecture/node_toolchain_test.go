package architecture_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestNodeToolchainAgreesAcrossDevelopmentLockfileAndDocker(t *testing.T) {
	root := repositoryRoot(t)
	read := func(name string) []byte {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	version := strings.TrimSpace(string(read(".node-version")))
	if !regexp.MustCompile(`^24\.[0-9]+\.[0-9]+$`).MatchString(version) {
		t.Fatalf("development must pin a concrete Node 24 LTS version: %q", version)
	}
	var manifest struct {
		Engines        map[string]string `json:"engines"`
		PackageManager string            `json:"packageManager"`
	}
	if err := json.Unmarshal(read("web/package.json"), &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Engines["node"] != ">="+version+" <25" ||
		!regexp.MustCompile(`^npm@11\.[0-9]+\.[0-9]+$`).MatchString(manifest.PackageManager) {
		t.Fatal("package engines and pinned npm must match the Node 24 toolchain")
	}
	var lock struct {
		Packages map[string]struct {
			Engines map[string]string `json:"engines"`
		} `json:"packages"`
	}
	if err := json.Unmarshal(read("web/package-lock.json"), &lock); err != nil {
		t.Fatal(err)
	}
	if lock.Packages[""].Engines["node"] != manifest.Engines["node"] {
		t.Fatal("lockfile must retain the declared Node engine")
	}
	image := `(?m)^FROM node:` + regexp.QuoteMeta(version) +
		`-bookworm-slim@sha256:[a-f0-9]{64} AS web$`
	if !regexp.MustCompile(image).Match(read("Dockerfile")) {
		t.Fatal("Docker frontend builder must pin the same Node version and an immutable digest")
	}
}
