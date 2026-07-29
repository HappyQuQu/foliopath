package architecture_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const requireStoryboardGoEnvironment = "FOLIOPATH_REQUIRE_STORYBOARD_GO"

type storyboardReadiness struct {
	SchemaVersion        int                      `json:"schemaVersion"`
	Release              string                   `json:"release"`
	Feature              string                   `json:"feature"`
	AssessedAt           string                   `json:"assessedAt"`
	Decision             string                   `json:"decision"`
	PrerequisiteGates    []storyboardPrerequisite `json:"prerequisiteGates"`
	AcceptanceCriteria   []storyboardAcceptance   `json:"acceptanceCriteria"`
	ReleaseBlockingRisks []storyboardRisk         `json:"releaseBlockingRisks"`
}

type storyboardPrerequisite struct {
	ID                 string   `json:"id"`
	TaskID             string   `json:"taskId"`
	Status             string   `json:"status"`
	Owner              string   `json:"owner"`
	Evidence           string   `json:"evidence"`
	BlockingConditions []string `json:"blockingConditions"`
}

type storyboardAcceptance struct {
	ID       string   `json:"id"`
	Status   string   `json:"status"`
	Evidence []string `json:"evidence"`
	Anchors  []string `json:"anchors"`
}

type storyboardRisk struct {
	ID                  string `json:"id"`
	Status              string `json:"status"`
	Owner               string `json:"owner"`
	ClosureCondition    string `json:"closureCondition"`
	DispositionEvidence string `json:"dispositionEvidence"`
}

func TestStoryboardReadinessManifestFailsClosed(t *testing.T) {
	root := repositoryRoot(t)
	content, err := os.ReadFile(filepath.Join(
		root,
		"docs",
		"releases",
		"POST-MVP-1-readiness.json",
	))
	if err != nil {
		t.Fatal(err)
	}

	var manifest storyboardReadiness
	if err := json.Unmarshal(content, &manifest); err != nil {
		t.Fatalf("decode storyboard readiness manifest: %v", err)
	}
	if manifest.SchemaVersion != 1 ||
		manifest.Release != "POST-MVP-1" ||
		manifest.Feature != "FTR-VID-001" ||
		manifest.AssessedAt == "" {
		t.Fatalf("invalid storyboard readiness identity: %+v", manifest)
	}
	if manifest.Decision != "go" && manifest.Decision != "no-go" {
		t.Fatalf("invalid storyboard decision %q", manifest.Decision)
	}

	expectedGates := []string{
		"VSP-S0", "VSP-S1", "VSP-S2", "VSP-S3",
		"VSP-301", "VSP-302", "VSP-303", "VSP-304",
	}
	actualGates := make([]string, 0, len(manifest.PrerequisiteGates))
	allGatesPassed := true
	taskList := readArchitectureFile(t, root, "docs/features/video-storyboard-preview-task-list.md")
	for _, gate := range manifest.PrerequisiteGates {
		actualGates = append(actualGates, gate.ID)
		if gate.TaskID == "" || gate.Owner == "" || gate.Evidence == "" {
			t.Errorf("%s requires taskId, owner, and evidence", gate.ID)
			continue
		}
		evidence := readArchitectureFile(t, root, gate.Evidence)
		switch gate.Status {
		case "passed":
			if len(gate.BlockingConditions) != 0 {
				t.Errorf("%s passed with blocking conditions", gate.ID)
			}
			if !strings.Contains(evidence, "**Go") &&
				!strings.Contains(evidence, "**Done") {
				t.Errorf("%s passed but its evidence has no Go/Done conclusion", gate.ID)
			}
		case "blocked":
			allGatesPassed = false
			if len(gate.BlockingConditions) == 0 {
				t.Errorf("%s blocked without closure conditions", gate.ID)
			}
			if !strings.Contains(evidence, "Pending") &&
				!strings.Contains(evidence, "No-Go") {
				t.Errorf("%s blocked but its evidence has no Pending/No-Go conclusion", gate.ID)
			}
		default:
			t.Errorf("%s has invalid status %q", gate.ID, gate.Status)
		}

		checkbox := "- [x]"
		if gate.Status == "blocked" {
			checkbox = "- [ ]"
		}
		if !strings.Contains(taskList, checkbox+" `"+gate.TaskID+"`") {
			t.Errorf(
				"%s status %s disagrees with task %s",
				gate.ID,
				gate.Status,
				gate.TaskID,
			)
		}
	}
	if !slices.Equal(actualGates, expectedGates) {
		t.Fatalf("storyboard prerequisite gates = %v, want %v", actualGates, expectedGates)
	}

	expectedAcceptance := []string{
		"VSP-AC-001", "VSP-AC-002", "VSP-AC-003", "VSP-AC-004",
		"VSP-AC-005", "VSP-AC-006", "VSP-AC-007", "VSP-AC-008",
	}
	actualAcceptance := make([]string, 0, len(manifest.AcceptanceCriteria))
	allAcceptancePassed := true
	featureSpec := readArchitectureFile(t, root, "docs/features/video-storyboard-preview.md")
	integratedGate := readArchitectureFile(
		t,
		root,
		"docs/gates/POST-MVP-1/vsp-s4-integrated-slice-done.md",
	)
	for _, criterion := range manifest.AcceptanceCriteria {
		actualAcceptance = append(actualAcceptance, criterion.ID)
		if len(criterion.Evidence) == 0 {
			t.Errorf("%s has no evidence", criterion.ID)
		}
		if len(criterion.Anchors) == 0 {
			t.Errorf("%s has no evidence anchors", criterion.ID)
		}
		var evidenceContent strings.Builder
		for _, evidence := range criterion.Evidence {
			evidenceContent.WriteString(readArchitectureFile(t, root, evidence))
			evidenceContent.WriteByte('\n')
		}
		for _, anchor := range criterion.Anchors {
			if !strings.Contains(evidenceContent.String(), anchor) {
				t.Errorf("%s evidence is missing anchor %q", criterion.ID, anchor)
			}
		}
		if !strings.Contains(featureSpec, "| `"+criterion.ID+"` |") {
			t.Errorf("%s is absent from the feature acceptance table", criterion.ID)
		}
		switch criterion.Status {
		case "passed":
			if !acceptanceTableContainsStatus(
				integratedGate,
				criterion.ID,
				"已有自动证据",
			) {
				t.Errorf("%s passed but S4 aggregation disagrees", criterion.ID)
			}
		case "blocked":
			allAcceptancePassed = false
			if !acceptanceTableContainsStatus(
				integratedGate,
				criterion.ID,
				"Blocked",
			) {
				t.Errorf("%s blocked but S4 aggregation disagrees", criterion.ID)
			}
		default:
			t.Errorf("%s has invalid status %q", criterion.ID, criterion.Status)
		}
	}
	if !slices.Equal(actualAcceptance, expectedAcceptance) {
		t.Fatalf(
			"storyboard acceptance criteria = %v, want %v",
			actualAcceptance,
			expectedAcceptance,
		)
	}

	allRisksDisposed := true
	if len(manifest.ReleaseBlockingRisks) != 1 ||
		manifest.ReleaseBlockingRisks[0].ID != "R-018" {
		t.Fatalf("storyboard release risks must contain only R-018")
	}
	riskRegister := readArchitectureFile(t, root, "docs/risk-register.md")
	for _, risk := range manifest.ReleaseBlockingRisks {
		if risk.Owner == "" || risk.ClosureCondition == "" {
			t.Errorf("%s requires owner and closure condition", risk.ID)
		}
		switch risk.Status {
		case "closed", "accepted":
			if risk.DispositionEvidence == "" {
				t.Errorf("%s is disposed without evidence", risk.ID)
			} else {
				_ = readArchitectureFile(t, root, risk.DispositionEvidence)
			}
		case "open", "mitigating":
			allRisksDisposed = false
		default:
			t.Errorf("%s has invalid status %q", risk.ID, risk.Status)
		}
		statusText := map[string]string{
			"closed": "已关闭", "accepted": "已接受",
			"open": "开放", "mitigating": "缓解中",
		}[risk.Status]
		if !riskTableContainsStatus(riskRegister, risk.ID, statusText) {
			t.Errorf("%s status %s disagrees with risk register", risk.ID, risk.Status)
		}
	}

	canRelease := allGatesPassed && allAcceptancePassed && allRisksDisposed
	if (manifest.Decision == "go") != canRelease {
		t.Fatalf(
			"storyboard decision %q disagrees with gates=%t acceptance=%t risks=%t",
			manifest.Decision,
			allGatesPassed,
			allAcceptancePassed,
			allRisksDisposed,
		)
	}
	if os.Getenv(requireStoryboardGoEnvironment) == "1" && !canRelease {
		t.Fatal("storyboard feature is No-Go; unresolved gates, acceptance, or risks remain")
	}
}

func readArchitectureFile(t *testing.T, root, relative string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(root, relative))
	if err != nil {
		t.Fatalf("read %s: %v", relative, err)
	}
	return string(content)
}

func riskTableContainsStatus(content, id, status string) bool {
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "| "+id+" |") &&
			strings.HasSuffix(line, "| "+status+" |") {
			return true
		}
	}
	return false
}

func acceptanceTableContainsStatus(content, id, status string) bool {
	for _, line := range strings.Split(content, "\n") {
		normalized := strings.ReplaceAll(line, "**", "")
		if strings.HasPrefix(normalized, "| `"+id+"` ") &&
			strings.Contains(normalized, "| "+status) {
			return true
		}
	}
	return false
}
