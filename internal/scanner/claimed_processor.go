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
