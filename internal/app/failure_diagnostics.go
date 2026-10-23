package app

import (
	"context"
	"errors"
	"log/slog"

	"github.com/HappyQuQu/foliopath/internal/jobs"
)

// Only fixed classifications and numeric storage codes cross the log boundary.
// Never use err.Error(), SQL, filesystem paths, or stack traces as attributes.
func runtimeFailureAttributes(err error) []any {
	code := "component_failure"
	var storage interface{ Code() int }
	attributes := []any{}
	switch {
	case errors.Is(err, context.Canceled):
		code = "context_cancelled"
	case errors.Is(err, context.DeadlineExceeded):
		code = "deadline_exceeded"
	case errors.As(err, &storage):
		attributes = append(attributes, slog.Int("storage_code", storage.Code()))
		switch storage.Code() & 255 {
		case 5:
			code = "storage_busy"
		case 6:
			code = "storage_locked"
		case 9:
			code = "storage_interrupted"
		case 10:
			code = "storage_io"
		case 11:
			code = "storage_corrupt"
		case 13:
			code = "storage_full"
		case 19:
			code = "storage_constraint"
		default:
			code = "storage_failure"
		}
	case errors.Is(err, errComponentStopped):
		code = "component_stopped"
	}
	attributes = append(attributes, slog.String("failure_code", code))
	var scan scanComponentFailure
	if errors.As(err, &scan) {
		switch scan.source {
		case "worker", "startup_admission", "scheduler":
			attributes = append(attributes, slog.String("component_source", scan.source))
		}
	}
	var worker jobs.WorkerFailure
	if errors.As(err, &worker) {
		switch worker.Stage {
		case "recovery", "claim", "heartbeat", "handoff":
			attributes = append(attributes, slog.String("worker_stage", worker.Stage))
		}
	}
	return attributes
}
