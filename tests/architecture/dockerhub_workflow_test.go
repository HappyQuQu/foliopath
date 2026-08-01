package architecture_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestDockerHubPublicationKeepsTagAndPlatformBoundaries(t *testing.T) {
	root := repositoryRoot(t)
	content, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "dockerhub.yml"))
	if err != nil {
		t.Fatalf("read Docker Hub workflow: %v", err)
	}
	workflow := string(content)

	requireFragments(t, ".github/workflows/dockerhub.yml", workflow, []string{
		"- main",
		`- "v*.*.*"`,
		"workflow_dispatch:",
		"IMAGE_NAME: evanqu/foliopath",
		"username: ${{ secrets.DOCKERHUB_USERNAME }}",
		"password: ${{ secrets.DOCKERHUB_TOKEN }}",
		"uses: docker/setup-qemu-action@v3",
		"platforms: linux/amd64,linux/arm64",
		"VERSION=${{ github.ref_name }}",
		"sbom: true",
		"provenance: true",
		"type=raw,value=latest,enable=${{ github.ref == 'refs/heads/main' || startsWith(github.ref, 'refs/tags/v') }}",
		"DOCKERHUB_DESCRIPTION_TOKEN",
		"secrets.DOCKERHUB_DESCRIPTION_TOKEN || secrets.DOCKERHUB_TOKEN",
		"readme-filepath: README.dockerhub.md",
	})

	if strings.Contains(workflow, "pull_request:") {
		t.Error("Docker Hub publication must not expose registry credentials to pull requests")
	}
	if strings.Contains(workflow, "Skip Docker Hub overview sync") {
		t.Error("Docker Hub overview publication must not be silently skipped")
	}
}

func TestDockerHubReadmeUsesAbsoluteReferences(t *testing.T) {
	root := repositoryRoot(t)
	content, err := os.ReadFile(filepath.Join(root, "README.dockerhub.md"))
	if err != nil {
		t.Fatalf("read Docker Hub README: %v", err)
	}

	patterns := []*regexp.Regexp{
		regexp.MustCompile(`!?\[[^]]*\]\(([^)]+)\)`),
		regexp.MustCompile(`(?:href|src)="([^"]+)"`),
	}
	for _, pattern := range patterns {
		for _, match := range pattern.FindAllStringSubmatch(string(content), -1) {
			if !strings.HasPrefix(match[1], "https://") && !strings.HasPrefix(match[1], "#") {
				t.Errorf("Docker Hub README reference must be an absolute HTTPS URL or an internal fragment: %q", match[1])
			}
		}
	}
}
