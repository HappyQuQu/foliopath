package api

import (
	"github.com/HappyQuQu/foliopath/internal/face"
	"time"
)

type faceDiagnosticTaskJSON struct {
	OperationID string    `json:"operationId"`
	State       string    `json:"state"`
	ErrorCode   *string   `json:"errorCode"`
	UpdatedAt   time.Time `json:"updatedAt"`
}
type faceFailureReasonJSON struct {
	Code  string `json:"code"`
	Count int64  `json:"count"`
}
type faceFailureSampleJSON struct {
	AssetID   string    `json:"assetId"`
	Name      string    `json:"name"`
	SizeBytes int64     `json:"sizeBytes"`
	Code      string    `json:"code"`
	UpdatedAt time.Time `json:"updatedAt"`
}
type faceDiagnosticsJSON struct {
	LatestTask         *faceDiagnosticTaskJSON `json:"latestTask"`
	RecordedFailures   int64                   `json:"recordedFailures"`
	UnrecordedFailures int64                   `json:"unrecordedFailures"`
	Reasons            []faceFailureReasonJSON `json:"reasons"`
	Samples            []faceFailureSampleJSON `json:"samples"`
}

func faceDiagnosticsResponse(value face.FailureDiagnostics) faceDiagnosticsJSON {
	result := faceDiagnosticsJSON{RecordedFailures: value.RecordedFailures, UnrecordedFailures: value.UnrecordedFailures, Reasons: []faceFailureReasonJSON{}, Samples: []faceFailureSampleJSON{}}
	if task := value.LatestTask; task != nil {
		var code *string
		if task.ErrorCode != "" {
			c := task.ErrorCode
			code = &c
		}
		result.LatestTask = &faceDiagnosticTaskJSON{OperationID: task.OperationID, State: task.State, ErrorCode: code, UpdatedAt: task.UpdatedAt}
	}
	for _, reason := range value.Reasons {
		result.Reasons = append(result.Reasons, faceFailureReasonJSON{Code: reason.Code, Count: reason.Count})
	}
	for _, sample := range value.Samples {
		result.Samples = append(result.Samples, faceFailureSampleJSON{AssetID: assetID(sample.AssetID), Name: sample.Name, SizeBytes: sample.SizeBytes, Code: sample.Code, UpdatedAt: sample.UpdatedAt})
	}
	return result
}
