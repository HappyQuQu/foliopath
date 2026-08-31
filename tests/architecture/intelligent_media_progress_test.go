package architecture_test

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

type intelligentMediaStageProgress struct {
	heading string
	label   string
	done    int
	total   int
}

var intelligentMediaMainTask = regexp.MustCompile("^- \\[[ x]\\] `INT-[0-9]+`")

func TestIntelligentMediaProgressSummaryMatchesMainTaskCheckboxes(t *testing.T) {
	content, err := os.ReadFile(filepath.Join(
		repositoryRoot(t),
		"docs",
		"features",
		"intelligent-media-discovery-task-list.md",
	))
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	lines := strings.Split(text, "\n")
	stages := []intelligentMediaStageProgress{
		{heading: "## S0 审计台账（本地探索已收口）", label: "S0 可行性审计"},
		{heading: "## S1：A+B 权威合同（完成）", label: "S1 A+B 权威合同"},
		{heading: "## S1R2：C+D+E 权威合同扩展（当前阶段）", label: "S1R2 C+D+E 合同扩展"},
		{heading: "## S2A：模型管理与图片语义搜索后端", label: "S2A 模型管理与图片语义搜索后端"},
		{heading: "## S2B：标签建议与视频代表帧搜索后端（revision 2，当前阶段）", label: "S2B 标签与视频搜索"},
		{heading: "## S2C：人脸聚类与人物库后端（revision 2，等待隐私准入）", label: "S2C 人脸与人物库"},
		{heading: "## S3：消费者与 UI（每个页面等待对应 S2）", label: "S3 消费者与 UI"},
		{heading: "## S4：纵向、容量与发布", label: "S4 纵向、容量与发布"},
	}

	for index := range stages {
		stages[index].done, stages[index].total = countIntelligentMediaMainTasks(
			t,
			lines,
			stages[index].heading,
		)
		percentage := roundedProgress(stages[index].done, stages[index].total)
		expectedRow := fmt.Sprintf(
			"| %s | %d / %d | %d%% |",
			stages[index].label,
			stages[index].done,
			stages[index].total,
			percentage,
		)
		if !strings.Contains(text, expectedRow) {
			t.Errorf("progress table is missing %q", expectedRow)
		}
	}

	revisionDone, revisionTotal := sumIntelligentMediaProgress(stages[:6])
	assertIntelligentMediaSummary(
		t,
		text,
		"revision 2 all-S2 main progress",
		revisionDone,
		revisionTotal,
	)
	allDone, allTotal := sumIntelligentMediaProgress(stages)
	assertIntelligentMediaSummary(
		t,
		text,
		"all-roadmap progress",
		allDone,
		allTotal,
	)
}

func countIntelligentMediaMainTasks(t *testing.T, lines []string, heading string) (int, int) {
	t.Helper()
	inside := false
	done, total := 0, 0
	for _, line := range lines {
		if line == heading {
			inside = true
			continue
		}
		if inside && strings.HasPrefix(line, "## ") {
			break
		}
		if !inside || !intelligentMediaMainTask.MatchString(line) {
			continue
		}
		total++
		if strings.HasPrefix(line, "- [x]") {
			done++
		}
	}
	if !inside || total == 0 {
		t.Fatalf("stage %q has no main tasks", heading)
	}
	return done, total
}

func roundedProgress(done, total int) int {
	return int(math.Round(float64(done) * 100 / float64(total)))
}

func sumIntelligentMediaProgress(stages []intelligentMediaStageProgress) (int, int) {
	done, total := 0, 0
	for _, stage := range stages {
		done += stage.done
		total += stage.total
	}
	return done, total
}

func assertIntelligentMediaSummary(t *testing.T, text, name string, done, total int) {
	t.Helper()
	expected := fmt.Sprintf("**%d / %d（%d%%）**", done, total, roundedProgress(done, total))
	if !strings.Contains(text, expected) {
		t.Errorf("%s is missing %q", name, expected)
	}
}
