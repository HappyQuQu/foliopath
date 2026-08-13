package thumbnail

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

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
			name: "source too large", err: media.ErrSourceTooLarge,
			outcome: JobPermanent, code: JobErrorProcessing,
		},
		{
			name: "frame unavailable", err: media.ErrFrameUnavailable,
			outcome: JobPermanent, code: JobErrorProcessing,
		},
		{
			name: "timeout", err: context.DeadlineExceeded,
			outcome: JobRetry, code: JobErrorTimeout,
		},
		{
			name: "storyboard budget exhausted",
			err: errors.Join(
				ErrStoryboardBudgetExhausted,
				context.DeadlineExceeded,
			),
			outcome: JobPermanent, code: JobErrorTimeout,
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
	job    Job
	calls  int
	err    error
}

func (stub *jobCompletionStub) FinishMediaJob(
	_ context.Context,
	job Job,
	result JobResult,
) error {
	stub.job = job
	stub.result = result
	stub.calls++
	return stub.err
}

func TestClaimedProcessorRecordsRecoveredJPEGWarningAsSucceeded(t *testing.T) {
	mtime := time.Unix(0, 100)
	fingerprint, err := media.NewSourceFingerprint(6, mtime.UnixNano())
	if err != nil {
		t.Fatal(err)
	}
	assetRepository := &repositoryStub{asset: Asset{
		ID: 9, LibraryID: 7, LibraryRoot: "family", RelativePath: "photo.jpg",
		Kind: media.KindImage, Format: media.FormatJPEG,
		SourceFingerprint: fingerprint,
	}}
	warning := media.FailureDiagnostic{
		Stage: media.StageValidation, Reason: media.ReasonDecodeRecovered,
		Tool: "libvips",
	}
	result := media.ProcessingResult{
		Metadata: media.Metadata{
			Width: 96, Height: 64, PlaybackStatus: media.PlaybackNotApplicable,
		},
		Thumbnail: media.Thumbnail{Bytes: []byte("webp"), Width: 48, Height: 32},
		Warning:   &warning,
	}
	derivation, err := GridDerivation(7, 9, fingerprint)
	if err != nil {
		t.Fatal(err)
	}
	cachePath, err := derivation.CacheRelativePath()
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewService(
		assetRepository,
		sourceStub{file: sourceFileStub{
			Reader: bytes.NewReader([]byte("source")),
			info:   fileInfoStub{size: 6, mtime: mtime},
		}},
		&publisherStub{published: Published{CacheRelativePath: cachePath}},
		&capacityStub{},
		&processorStub{result: result},
		&processorStub{},
		ServiceOptions{},
	)
	if err != nil {
		t.Fatal(err)
	}
	storyboard, err := NewStoryboardService(
		assetRepository,
		sourceStub{},
		&publisherStub{},
		&capacityStub{},
		&storyboardProcessorStub{},
		ServiceOptions{},
	)
	if err != nil {
		t.Fatal(err)
	}
	completion := &jobCompletionStub{}
	processor, err := NewClaimedProcessor(service, storyboard, completion)
	if err != nil {
		t.Fatal(err)
	}
	job := Job{
		ID: 3, LibraryID: 7, AssetID: 9, Variant: VariantGrid,
		TransformVersion:  GridTransformVersion,
		SourceFingerprint: fingerprint, Attempt: 1,
	}
	if err := processor.Process(context.Background(), job); err != nil {
		t.Fatalf("process recovered JPEG: %v", err)
	}
	if assetRepository.ready == nil || completion.calls != 1 ||
		completion.job != job || completion.result.Outcome != JobSucceeded ||
		completion.result.Code != "" || completion.result.Diagnostic != warning {
		t.Fatalf(
			"ready/completion = %#v calls %d job %#v result %#v",
			assetRepository.ready, completion.calls, completion.job, completion.result,
		)
	}
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
