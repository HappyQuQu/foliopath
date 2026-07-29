package thumbnail

import (
	"context"
	"errors"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/media"
)

func TestJobResultClassificationSeparatesPermanentAndRetryableFailures(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		outcome JobOutcome
		code    JobErrorCode
	}{
		{name: "success", outcome: JobSucceeded},
		{
			name: "invalid", err: media.ErrInvalidMedia,
			outcome: JobPermanent, code: JobErrorInvalidMedia,
		},
		{
			name: "unsupported", err: media.ErrUnsupportedMedia,
			outcome: JobPermanent, code: JobErrorUnsupportedMedia,
		},
		{
			name: "timeout", err: context.DeadlineExceeded,
			outcome: JobRetry, code: JobErrorTimeout,
		},
		{
			name: "source", err: ErrSourceUnavailable,
			outcome: JobRetry, code: JobErrorSource,
		},
		{
			name: "capacity", err: ErrCacheCapacity,
			outcome: JobRetry, code: JobErrorCache,
		},
		{
			name: "stale", err: ErrSourceChanged,
			outcome: JobStale,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := classifyJobResult(test.err)
			if result.Outcome != test.outcome || result.Code != test.code {
				t.Fatalf("result = %#v", result)
			}
			if result.Outcome == JobRetry && result.RetryDelay <= 0 {
				t.Fatal("retry result has no backoff")
			}
		})
	}
}

type jobCompletionStub struct {
	result JobResult
	err    error
}

func (stub *jobCompletionStub) FinishMediaJob(
	context.Context,
	Job,
	JobResult,
) error {
	stub.result = JobResult{Outcome: JobSucceeded}
	return stub.err
}

func TestClaimedProcessorDoesNotCommitCancellationAsTerminal(t *testing.T) {
	repository := &jobCompletionStub{}
	service, err := NewService(
		&repositoryStub{},
		sourceStub{},
		&publisherStub{},
		&capacityStub{},
		&processorStub{},
		&processorStub{},
		ServiceOptions{},
	)
	if err != nil {
		t.Fatal(err)
	}
	storyboard, err := NewStoryboardService(
		&repositoryStub{},
		sourceStub{},
		&publisherStub{},
		&capacityStub{},
		&storyboardProcessorStub{},
		ServiceOptions{},
	)
	if err != nil {
		t.Fatal(err)
	}
	processor, err := NewClaimedProcessor(service, storyboard, repository)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := processor.Process(ctx, Job{
		AssetID: 1, Variant: VariantGrid, TransformVersion: GridTransformVersion,
	}); !errors.Is(
		err, context.Canceled,
	) {
		t.Fatalf("cancellation error = %v", err)
	}
	if repository.result.Outcome != "" {
		t.Fatal("cancelled job was committed terminal")
	}
}
