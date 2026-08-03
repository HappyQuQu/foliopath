package architecture_test

import (
	"encoding/json"
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
		"uses: googleapis/release-please-action@v4",
		"config-file: release-please-config.json",
		"manifest-file: .release-please-manifest.json",
		"Automatically merge release pull request",
		`gh pr merge "${pr_number}"`,
		"--squash",
		"Finalize automatically merged release",
		"skip-github-pull-request: true",
		"uses: docker/setup-qemu-action@v3",
		"platforms: linux/amd64,linux/arm64",
		"VERSION=${{ steps.version.outputs.value }}",
		"sbom: true",
		"provenance: true",
		"type=raw,value=latest,enable=${{ github.ref == 'refs/heads/main' || startsWith(github.ref, 'refs/tags/v') }}",
		"type=raw,value=${{ needs.release.outputs.version }},enable=${{ needs.release.outputs.release_created == 'true' }}",
		"type=raw,value=sha-${{ steps.version.outputs.short_sha }}",
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
	if strings.Contains(workflow, "192.168.2.222") ||
		strings.Contains(workflow, "appleboy/ssh-action") {
		t.Error("release publication must not deploy to a live instance")
	}
}

func TestFriendlyReleaseAutomationContract(t *testing.T) {
	root := repositoryRoot(t)
	config, err := os.ReadFile(filepath.Join(root, "release-please-config.json"))
	if err != nil {
		t.Fatalf("read release-please config: %v", err)
	}
	manifest, err := os.ReadFile(filepath.Join(root, ".release-please-manifest.json"))
	if err != nil {
		t.Fatalf("read release-please manifest: %v", err)
	}
	changelog, err := os.ReadFile(filepath.Join(root, "CHANGELOG.md"))
	if err != nil {
		t.Fatalf("read changelog: %v", err)
	}

	requireFragments(t, "release-please-config.json", string(config), []string{
		`"release-type": "simple"`,
		`"include-v-in-tag": true`,
		`"section": "✨ 新功能"`,
		`"section": "🚀 改进"`,
		`"section": "🐛 修复"`,
		`"section": "⚠️ 注意事项"`,
		`"type": "chore"`,
		`"hidden": true`,
	})
	var versions map[string]string
	if err := json.Unmarshal(manifest, &versions); err != nil {
		t.Fatalf("parse release-please manifest: %v", err)
	}
	if !regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`).MatchString(versions["."]) {
		t.Errorf("release-please manifest root version is not semantic: %q", versions["."])
	}
	requireFragments(t, "CHANGELOG.md", string(changelog), []string{
		"# 更新日志",
		"用户可见变化",
	})
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

	for _, advancedComposeSetting := range []string{
		"security_opt:",
		"cap_drop:",
		"stop_grace_period:",
		"    user:",
		"read_only: true",
		"tmpfs:",
	} {
		if strings.Contains(string(content), advancedComposeSetting) {
			t.Errorf("Docker Hub quick start must omit advanced Compose setting %q", advancedComposeSetting)
		}
	}
}
