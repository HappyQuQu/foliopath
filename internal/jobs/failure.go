package jobs

import "fmt"

// WorkerFailure identifies a fixed worker boundary without exposing its cause
// in operational logs. The cause remains available through errors.Is/As.
type WorkerFailure struct {
	Stage string
	Err   error
}

func (failure WorkerFailure) Error() string {
	label := "worker failure"
	switch failure.Stage {
	case "recovery":
		label = "recover expired work"
	case "claim":
		label = "claim work"
	case "heartbeat":
		label = "refresh work lease"
	case "handoff":
		label = "handoff planned work"
	}
	return fmt.Sprintf("%s: %v", label, failure.Err)
}
func (failure WorkerFailure) Unwrap() error { return failure.Err }

func workerFailure(stage string, err error) error {
	return WorkerFailure{Stage: stage, Err: err}
}
