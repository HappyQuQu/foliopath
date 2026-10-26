package scanner

import (
	"context"
	"errors"
)

// ClaimedProcessor adapts the scanner-owned generation state machine to a
// jobs-owned durable worker. It never allocates or claims queue work itself.
type ClaimedProcessor struct {
	service *Service
	walker  Walker
}

func NewClaimedProcessor(service *Service, walker Walker) (*ClaimedProcessor, error) {
	if service == nil || walker == nil {
		return nil, errors.New("claimed scan processor dependencies are required")
	}
	return &ClaimedProcessor{service: service, walker: walker}, nil
}

func (processor *ClaimedProcessor) Process(ctx context.Context, run ScanRun) error {
	_, err := processor.service.RunClaimedFullScan(ctx, run, processor.walker)
	return err
}

// BackgroundAdmissionCancelled enters the scanner-owned cancellation path
// without opening media. RunClaimedFullScan handles its cancelled entry context
// with the same bounded abort used after walking has begun.
func (processor *ClaimedProcessor) BackgroundAdmissionCancelled(ctx context.Context, run ScanRun) {
	if ctx.Err() != nil {
		_, _ = processor.service.RunClaimedFullScan(ctx, run, processor.walker)
	}
}
