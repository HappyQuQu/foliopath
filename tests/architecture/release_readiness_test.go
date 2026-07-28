package architecture_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const requireReleaseGoEnvironment = "FOLIOPATH_REQUIRE_RELEASE_GO"

type releaseReadiness struct {
	SchemaVersion        int                   `json:"schemaVersion"`
	Release              string                `json:"release"`
	AssessedAt           string                `json:"assessedAt"`
	Decision             string                `json:"decision"`
	PrerequisiteGates    []releasePrerequisite `json:"prerequisiteGates"`
	ReleaseBlockingRisks []releaseRisk         `json:"releaseBlockingRisks"`
}

type releasePrerequisite struct {
	ID                 string   `json:"id"`
	Status             string   `json:"status"`
	Owner              string   `json:"owner"`
	Evidence           string   `json:"evidence"`
	BlockingConditions []string `json:"blockingConditions"`
}

type releaseRisk struct {
	ID                  string          `json:"id"`
	Status              string          `json:"status"`
	Owner               string          `json:"owner"`
	ClosureCondition    string          `json:"closureCondition"`
	DispositionEvidence string          `json:"dispositionEvidence"`
	Acceptance          *riskAcceptance `json:"acceptance"`
}

type riskAcceptance struct {
	Approver  string `json:"approver"`
	Scope     string `json:"scope"`
	ExpiresAt string `json:"expiresAt"`
	Evidence  string `json:"evidence"`
}

func TestReleaseReadinessManifestFailsClosed(t *testing.T) {
	root := repositoryRoot(t)
	manifestPath := filepath.Join(
		root,
		"docs",
		"releases",
		"MVP-2026-07-23-rc-readiness.json",
	)
	content, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}

	var manifest releaseReadiness
	if err := json.Unmarshal(content, &manifest); err != nil {
		t.Fatalf("decode release readiness manifest: %v", err)
	}
	if manifest.SchemaVersion != 1 ||
		manifest.Release != "MVP-2026-07-23" ||
		manifest.AssessedAt == "" {
		t.Fatalf("invalid release readiness identity: %+v", manifest)
	}
	if manifest.Decision != "go" && manifest.Decision != "no-go" {
		t.Fatalf("invalid release decision %q", manifest.Decision)
	}

	expectedGates := []string{
		"S5-001", "S5-002", "S5-003", "S5-004",
		"S5-005", "S5-006", "S5-007", "S5-008",
	}
	actualGates := make([]string, 0, len(manifest.PrerequisiteGates))
	allGatesPassed := true
	for _, gate := range manifest.PrerequisiteGates {
		actualGates = append(actualGates, gate.ID)
		if gate.Owner == "" || gate.Evidence == "" {
			t.Errorf("%s requires an owner and evidence", gate.ID)
		}
		evidence, err := os.ReadFile(filepath.Join(root, gate.Evidence))
		if err != nil {
			t.Errorf("%s evidence: %v", gate.ID, err)
		}
		switch gate.Status {
		case "passed":
			if len(gate.BlockingConditions) != 0 {
				t.Errorf("%s passed with blocking conditions", gate.ID)
			}
			if !strings.Contains(string(evidence), "**Go") {
				t.Errorf("%s is passed but its evidence is not a Go Gate", gate.ID)
			}
		case "blocked":
			allGatesPassed = false
			if len(gate.BlockingConditions) == 0 {
				t.Errorf("%s is blocked without a closure condition", gate.ID)
			}
			if !strings.Contains(string(evidence), "**No-Go") &&
				!strings.Contains(string(evidence), "**Conditional") &&
				!strings.Contains(string(evidence), "**In progress") {
				t.Errorf("%s is blocked but its evidence has no blocking conclusion", gate.ID)
			}
		default:
			t.Errorf("%s has invalid status %q", gate.ID, gate.Status)
		}
	}
	if !slices.Equal(actualGates, expectedGates) {
		t.Fatalf("release prerequisite gates = %v, want %v", actualGates, expectedGates)
	}
	taskList, err := os.ReadFile(filepath.Join(root, "docs", "task-list.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, gate := range manifest.PrerequisiteGates {
		checkbox := "- [ ]"
		if gate.Status == "passed" {
			checkbox = "- [x]"
		}
		expected := checkbox + " `"
		found := false
		for _, line := range strings.Split(string(taskList), "\n") {
			if strings.HasPrefix(line, expected) &&
				strings.Contains(line, "`"+gate.ID+"`") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%s status %s disagrees with docs/task-list.md", gate.ID, gate.Status)
		}
	}

	expectedRisks := []string{
		"R-002", "R-003", "R-006", "R-008",
		"R-010", "R-011", "R-014", "R-017",
	}
	actualRisks := make([]string, 0, len(manifest.ReleaseBlockingRisks))
	allRisksDisposed := true
	for _, risk := range manifest.ReleaseBlockingRisks {
		actualRisks = append(actualRisks, risk.ID)
		if risk.Owner == "" || risk.ClosureCondition == "" {
			t.Errorf("%s requires an owner and closure condition", risk.ID)
		}
		switch risk.Status {
		case "closed":
			if risk.DispositionEvidence == "" {
				t.Errorf("%s is closed without disposition evidence", risk.ID)
			} else if _, err := os.Stat(
				filepath.Join(root, risk.DispositionEvidence),
			); err != nil {
				t.Errorf("%s disposition evidence: %v", risk.ID, err)
			}
		case "accepted":
			if risk.Acceptance == nil ||
				risk.Acceptance.Approver == "" ||
				risk.Acceptance.Scope == "" ||
				risk.Acceptance.ExpiresAt == "" ||
				risk.Acceptance.Evidence == "" {
				t.Errorf("%s acceptance is missing approver, scope, expiry, or evidence", risk.ID)
			} else if _, err := os.Stat(
				filepath.Join(root, risk.Acceptance.Evidence),
			); err != nil {
				t.Errorf("%s acceptance evidence: %v", risk.ID, err)
			}
		case "open", "mitigating":
			allRisksDisposed = false
		default:
			t.Errorf("%s has invalid status %q", risk.ID, risk.Status)
		}
	}
	if !slices.Equal(actualRisks, expectedRisks) {
		t.Fatalf("release-blocking risks = %v, want %v", actualRisks, expectedRisks)
	}
	riskRegister, err := os.ReadFile(filepath.Join(root, "docs", "risk-register.md"))
	if err != nil {
		t.Fatal(err)
	}
	riskStatusText := map[string]string{
		"closed":     "已关闭",
		"accepted":   "已接受",
		"open":       "开放",
		"mitigating": "缓解中",
	}
	for _, risk := range manifest.ReleaseBlockingRisks {
		found := false
		for _, line := range strings.Split(string(riskRegister), "\n") {
			if strings.HasPrefix(line, "| "+risk.ID+" |") &&
				strings.HasSuffix(line, "| "+riskStatusText[risk.Status]+" |") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%s status %s disagrees with docs/risk-register.md", risk.ID, risk.Status)
		}
	}

	canRelease := allGatesPassed && allRisksDisposed
	if (manifest.Decision == "go") != canRelease {
		t.Fatalf(
			"release decision %q disagrees with gatesPassed=%t risksDisposed=%t",
			manifest.Decision,
			allGatesPassed,
			allRisksDisposed,
		)
	}
	if os.Getenv(requireReleaseGoEnvironment) == "1" && !canRelease {
		t.Fatal("release candidate is No-Go; unresolved gates and risks remain")
	}
}
