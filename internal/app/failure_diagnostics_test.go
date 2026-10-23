package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/jobs"
)

type diagnosticStorageError int

func (err diagnosticStorageError) Error() string { return "SQL secret /private/media/secret.jpg" }
func (err diagnosticStorageError) Code() int     { return int(err) }

func TestRuntimeFailureDiagnosticsRemainPrivateAndClassified(t *testing.T) {
	for _, test := range []struct {
		name, code string
		err        error
	}{
		{"busy", "storage_busy", diagnosticStorageError(5)},
		{"snapshot", "storage_busy", diagnosticStorageError(517)},
		{"interrupted", "storage_interrupted", diagnosticStorageError(9)},
		{"constraint", "storage_constraint", diagnosticStorageError(19)},
		{"unknown", "component_failure", errors.New("SQL secret /private/media/secret.jpg")},
		{"deadline", "deadline_exceeded", context.DeadlineExceeded},
	} {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			cause := fmt.Errorf("untrusted SQL /private/media: %w", test.err)
			err := componentStopError{name: "scan-worker", err: scanComponentFailure{source: "worker", err: jobs.WorkerFailure{Stage: "claim", Err: cause}}}
			newJSONLogger(&output).Error("application.component_stopped", runtimeFailureAttributes(err)...)
			var event map[string]any
			if err := json.Unmarshal(output.Bytes(), &event); err != nil {
				t.Fatal(err)
			}
			if event["failure_code"] != test.code || event["worker_stage"] != "claim" || event["component_source"] != "worker" {
				t.Fatalf("unexpected event: %#v", event)
			}
			for _, secret := range []string{"secret", "SQL", "/private/media"} {
				if strings.Contains(output.String(), secret) {
					t.Fatalf("leaked %s", secret)
				}
			}
			if !errors.Is(err, test.err) {
				t.Fatal("cause lost")
			}
		})
	}
}
