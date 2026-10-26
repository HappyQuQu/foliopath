package jobs

import "errors"

var ErrHandoffConflict = errors.New("analysis handoff snapshot conflict")

// HandoffClaim is a bounded, offline operator snapshot, not a browser contract.
// The application owner must be stopped before the maintenance adapter uses it.
type HandoffClaim struct {
	Kind           string `json:"kind"`
	JobID          string `json:"jobId"`
	OperationID    string `json:"operationId"`
	ClaimRevision  int64  `json:"claimRevision"`
	AttemptCount   int64  `json:"attemptCount"`
	CheckpointID   int64  `json:"checkpointId"`
	CompletedItems int64  `json:"completedItems"`
}
