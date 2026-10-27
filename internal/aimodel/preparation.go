package aimodel

import "context"

// Preparation is informational and never grants model admission or identity trust.
type Preparation struct {
	State          string `json:"state"`
	CompletedFiles int64  `json:"completedFiles"`
	TotalFiles     int64  `json:"totalFiles"`
	CompletedBytes int64  `json:"completedBytes"`
	TotalBytes     int64  `json:"totalBytes"`
}

type PreparationReader interface {
	Preparation(context.Context) (Preparation, error)
}
