package architecture_test

import (
	"os"
	"path/filepath"
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
	})

	if strings.Contains(workflow, "pull_request:") {
		t.Error("Docker Hub publication must not expose registry credentials to pull requests")
	}
}
