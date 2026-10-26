package architecture_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPlannedAnalysisHandoffHasOneFencedOwner(t *testing.T) {
	read := func(path string) string {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(repositoryRoot(t), path))
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	worker := read("internal/jobs/worker.go")
	if !strings.Contains(worker, "<-heartbeatDone") || !strings.Contains(worker, "context.Cause(ctx), ErrPlannedShutdown") || !strings.Contains(worker, "owner.Handoff(handoffCtx, work)") {
		t.Fatal("handoff must be after processor/heartbeat and require the planned cause")
	}
	if strings.Index(worker, "owner.Handoff(handoffCtx, work)") < strings.Index(worker, "<-heartbeatDone") {
		t.Fatal("handoff before heartbeat reap")
	}
	if !strings.Contains(read("internal/app/run.go"), "cancel(jobs.ErrPlannedShutdown)") || !strings.Contains(read("internal/app/ai_models.go"), "context.Cause(service.parent)") {
		t.Fatal("lost signal-to-worker lifecycle cause")
	}
	store := read("internal/store/sqlite/analysis_handoff.go")
	for _, required := range []string{"analysis_job_handoffs", "claim_revision", "claimed_revision", "attempt_count=attempt_count-1", "o.state='running'", "exact &&", "s.withWriteTx"} {
		if !strings.Contains(store, required) {
			t.Fatal("lost fence/audit/snapshot: " + required)
		}
	}
	for _, forbidden := range []string{"UPDATE asset_tags", "UPDATE tags", "UPDATE semantic_embeddings", "UPDATE face_asset_results"} {
		if strings.Contains(store, forbidden) {
			t.Fatal("handoff changed derived/curated output")
		}
	}
	if strings.Contains(read("api/openapi.yaml"), "handoff-stopped") {
		t.Fatal("offline handoff leaked into browser API")
	}
}
