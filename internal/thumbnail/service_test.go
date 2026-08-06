package thumbnail

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/media"
)

type repositoryStub struct {
	asset             Asset
	ready             *Ready
	failure           *Failure
	metadataFailure   *MetadataReadyFailure
	storyboardReady   *StoryboardReady
	storyboardFailure *StoryboardFailure
	commitErr         error
}

func (stub *repositoryStub) GetAssetForDerivation(context.Context, int64) (Asset, error) {
	return stub.asset, nil
}

func (stub *repositoryStub) CommitReady(_ context.Context, ready Ready) error {
	stub.ready = &ready
	return stub.commitErr
}

func (stub *repositoryStub) CommitFailure(_ context.Context, failure Failure) error {
	stub.failure = &failure
	return stub.commitErr
}

func (stub *repositoryStub) CommitMetadataReadyFailure(
	_ context.Context,
	failure MetadataReadyFailure,
) error {
	stub.metadataFailure = &failure
	return stub.commitErr
}

func (stub *repositoryStub) CommitStoryboardReady(
	_ context.Context,
	ready StoryboardReady,
) error {
	stub.storyboardReady = &ready
	return stub.commitErr
}

func (stub *repositoryStub) CommitStoryboardFailure(
	_ context.Context,
	failure StoryboardFailure,
) error {
	stub.storyboardFailure = &failure
	return stub.commitErr
}

type sourceStub struct {
	file SourceFile
	err  error
}

func (stub sourceStub) OpenAsset(context.Context, string, string) (SourceFile, error) {
	return stub.file, stub.err
}

type processorStub struct {
	result media.ProcessingResult
	err    error
	calls  int
	run    func(context.Context) (media.ProcessingResult, error)
}

type storyboardProcessorStub struct {
	result   media.StoryboardResult
	request  media.StoryboardRequest
	requests []media.StoryboardRequest
	err      error
	calls    int
	run      func(
		context.Context,
		media.StoryboardRequest,
	) (media.StoryboardResult, error)
}

func (stub *storyboardProcessorStub) Storyboard(
	ctx context.Context,
	_ io.ReadSeeker,
	_ media.Format,
	request media.StoryboardRequest,
) (media.StoryboardResult, error) {
	stub.calls++
	stub.request = request
	stub.requests = append(stub.requests, request)
	if stub.run != nil {
		return stub.run(ctx, request)
	}
	return stub.result, stub.err
}

func (stub *processorStub) Process(
	ctx context.Context,
	_ io.ReadSeeker,
	_ media.Format,
) (media.ProcessingResult, error) {
	stub.calls++
	if stub.run != nil {
		return stub.run(ctx)
	}
	return stub.result, stub.err
}

type publisherStub struct {
	published Published
	calls     int
	err       error
}

type capacityStub struct {
	reserved int64
}

func (stub *capacityStub) Reserve(
	_ context.Context,
	value int64,
) (Reservation, error) {
	stub.reserved = value
	return &cacheReservation{}, nil
}

func (stub *publisherStub) Publish(
	_ context.Context,
	_ Derivation,
	value []byte,
) (Published, error) {
	stub.calls++
	stub.published.ByteSize = int64(len(value))
	return stub.published, stub.err
}

type sourceFileStub struct {
	*bytes.Reader
	info fs.FileInfo
}

func (sourceFileStub) Close() error                    { return nil }
func (stub sourceFileStub) Stat() (fs.FileInfo, error) { return stub.info, nil }

type fileInfoStub struct {
	size  int64
	mtime time.Time
}

func (fileInfoStub) Name() string            { return "asset" }
func (stub fileInfoStub) Size() int64        { return stub.size }
func (fileInfoStub) Mode() fs.FileMode       { return 0o400 }
func (stub fileInfoStub) ModTime() time.Time { return stub.mtime }
func (fileInfoStub) IsDir() bool             { return false }
func (fileInfoStub) Sys() any                { return nil }

func TestServicePublishesThenCommitsFingerprintBoundResult(t *testing.T) {
	mtime := time.Unix(0, 100)
	fingerprint, err := media.NewSourceFingerprint(6, mtime.UnixNano())
	if err != nil {
		t.Fatal(err)
	}
	repository := &repositoryStub{asset: Asset{
		ID: 9, LibraryID: 7, LibraryRoot: "family", RelativePath: "photo.jpg",
		Kind: media.KindImage, Format: media.FormatJPEG,
		SizeBytes: 6, ModifiedAtNS: mtime.UnixNano(), SourceFingerprint: fingerprint,
	}}
	source := sourceStub{file: sourceFileStub{
		Reader: bytes.NewReader([]byte("source")),
		info:   fileInfoStub{size: 6, mtime: mtime},
	}}
	result := media.ProcessingResult{
		Metadata: media.Metadata{
			Width: 96, Height: 64, PlaybackStatus: media.PlaybackNotApplicable,
		},
		Thumbnail: media.Thumbnail{Bytes: []byte("webp"), Width: 48, Height: 32},
	}
	image := &processorStub{result: result}
	video := &processorStub{}
	derivation, err := GridDerivation(7, 9, fingerprint)
	if err != nil {
		t.Fatal(err)
	}
	cachePath, err := derivation.CacheRelativePath()
	if err != nil {
		t.Fatal(err)
	}
	publisher := &publisherStub{published: Published{CacheRelativePath: cachePath}}
	capacity := &capacityStub{}
	service, err := NewService(
		repository, source, publisher, capacity, image, video,
		ServiceOptions{Now: func() time.Time { return time.UnixMilli(1000) }},
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Process(context.Background(), 9); err != nil {
		t.Fatal(err)
	}
	if image.calls != 1 || video.calls != 0 || publisher.calls != 1 ||
		capacity.reserved != int64(len(result.Thumbnail.Bytes)) ||
		repository.ready == nil || repository.ready.CreatedAtMS != 1000 ||
		repository.failure != nil {
		t.Fatalf("calls/result = image %d video %d publisher %d ready %#v failure %#v",
			image.calls, video.calls, publisher.calls, repository.ready, repository.failure)
	}
}

func TestServicePreservesVideoMetadataWhenPosterGenerationFails(t *testing.T) {
	mtime := time.Unix(0, 100)
	fingerprint, err := media.NewSourceFingerprint(6, mtime.UnixNano())
	if err != nil {
		t.Fatal(err)
	}
	repository := &repositoryStub{asset: Asset{
		ID: 9, LibraryID: 7, LibraryRoot: "family", RelativePath: "clip.mp4",
		Kind: media.KindVideo, Format: media.FormatMP4,
		SizeBytes: 6, ModifiedAtNS: mtime.UnixNano(), SourceFingerprint: fingerprint,
	}}
	duration := int64(1_000)
	video := &processorStub{
		result: media.ProcessingResult{Metadata: media.Metadata{
			Width: 1920, Height: 1080, DurationMS: &duration,
			PlaybackStatus: media.PlaybackUnknown,
		}},
		err: media.WithFailureDiagnostic(
			media.ErrUnsupportedMedia,
			media.FailureDiagnostic{
				Stage: media.StagePoster, Reason: media.ReasonDecoderUnavailable,
				Tool: "ffmpeg",
			},
		),
	}
	service, err := NewService(
		repository,
		sourceStub{file: sourceFileStub{
			Reader: bytes.NewReader([]byte("source")),
			info:   fileInfoStub{size: 6, mtime: mtime},
		}},
		&publisherStub{},
		&capacityStub{},
		&processorStub{},
		video,
		ServiceOptions{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Process(context.Background(), 9); !errors.Is(
		err, media.ErrUnsupportedMedia,
	) {
		t.Fatalf("poster failure = %v", err)
	}
	if repository.metadataFailure == nil || repository.failure != nil ||
		repository.metadataFailure.Metadata.Width != 1920 ||
		repository.metadataFailure.Code != media.ErrorUnsupportedMedia {
		t.Fatalf(
			"metadata/failure = %#v / %#v",
			repository.metadataFailure,
			repository.failure,
		)
	}
}

func TestStoryboardServicePublishesAllFramesAndCommitsLayout(t *testing.T) {
	mtime := time.Unix(0, 100)
	fingerprint, err := media.NewSourceFingerprint(6, mtime.UnixNano())
	if err != nil {
		t.Fatal(err)
	}
	duration := int64(10_000)
	repository := &repositoryStub{asset: Asset{
		ID: 9, LibraryID: 7, LibraryRoot: "family",
		RelativePath: "clip.mp4", Kind: media.KindVideo,
		Format: media.FormatMP4, SourceFingerprint: fingerprint,
		Width: 1920, Height: 1080, DurationMS: &duration,
		ProbeStatus: media.ProbeReady, GridReady: true,
	}}
	source := sourceStub{file: sourceFileStub{
		Reader: bytes.NewReader([]byte("source")),
		info:   fileInfoStub{size: 6, mtime: mtime},
	}}
	result := media.StoryboardResult{
		Bytes:      []byte("RIFF\x04\x00\x00\x00WEBP"),
		FrameCount: 10,
		Columns:    5,
		Rows:       2,
		CellWidth:  320,
		CellHeight: 180,
	}
	processor := &storyboardProcessorStub{result: result}
	derivation, err := StoryboardDerivation(7, 9, fingerprint)
	if err != nil {
		t.Fatal(err)
	}
	cachePath, err := derivation.CacheRelativePath()
	if err != nil {
		t.Fatal(err)
	}
	publisher := &publisherStub{
		published: Published{CacheRelativePath: cachePath},
	}
	capacity := &capacityStub{}
	service, err := NewStoryboardService(
		repository,
		source,
		publisher,
		capacity,
		processor,
		ServiceOptions{Now: func() time.Time { return time.UnixMilli(1000) }},
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Process(context.Background(), 9); err != nil {
		t.Fatal(err)
	}
	if processor.calls != 1 ||
		len(processor.request.TimestampsMS) != 10 ||
		publisher.calls != 1 ||
		capacity.reserved != int64(len(result.Bytes)) ||
		repository.storyboardReady == nil ||
		repository.storyboardReady.CreatedAtMS != 1000 ||
		repository.storyboardFailure != nil {
		t.Fatalf(
			"storyboard calls/result = processor %d publisher %d ready %#v failure %#v",
			processor.calls,
			publisher.calls,
			repository.storyboardReady,
			repository.storyboardFailure,
		)
	}
}

func TestStoryboardServiceFallsBackToFourFramesAfterAdaptiveTimeout(t *testing.T) {
	mtime := time.Unix(0, 100)
	fingerprint, _ := media.NewSourceFingerprint(6, mtime.UnixNano())
	duration := int64(68_000)
	repository := &repositoryStub{asset: Asset{
		ID: 9, LibraryID: 7, LibraryRoot: "family",
		RelativePath: "4k60.mp4", Kind: media.KindVideo,
		Format: media.FormatMP4, SourceFingerprint: fingerprint,
		Width: 3840, Height: 2160, DurationMS: &duration,
		ProbeStatus: media.ProbeReady, GridReady: true,
	}}
	source := sourceStub{file: sourceFileStub{
		Reader: bytes.NewReader([]byte("source")),
		info:   fileInfoStub{size: 6, mtime: mtime},
	}}
	processor := &storyboardProcessorStub{run: func(
		_ context.Context,
		request media.StoryboardRequest,
	) (media.StoryboardResult, error) {
		if len(request.TimestampsMS) == StoryboardLongFrameCount {
			return media.StoryboardResult{}, media.WithFailureDiagnostic(
				context.DeadlineExceeded,
				media.FailureDiagnostic{
					Stage:  media.StageFrameExtract,
					Reason: media.ReasonTimedOut,
					Tool:   "ffmpeg",
				},
			)
		}
		return media.StoryboardResult{
			Bytes:      []byte("RIFF\x04\x00\x00\x00WEBP"),
			FrameCount: len(request.TimestampsMS),
			Columns:    request.Columns,
			Rows:       request.Rows,
			CellWidth:  request.CellWidth,
			CellHeight: request.CellHeight,
		}, nil
	}}
	derivation, err := StoryboardDerivation(7, 9, fingerprint)
	if err != nil {
		t.Fatal(err)
	}
	cachePath, err := derivation.CacheRelativePath()
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewStoryboardService(
		repository,
		source,
		&publisherStub{published: Published{CacheRelativePath: cachePath}},
		&capacityStub{},
		processor,
		ServiceOptions{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Process(context.Background(), 9); err != nil {
		t.Fatal(err)
	}
	if processor.calls != 2 || len(processor.requests) != 2 ||
		len(processor.requests[0].TimestampsMS) != StoryboardLongFrameCount ||
		processor.requests[0].Timeout != 3*time.Minute ||
		len(processor.requests[1].TimestampsMS) != StoryboardShortFrameCount ||
		processor.requests[1].Timeout != 90*time.Second ||
		repository.storyboardReady == nil ||
		repository.storyboardReady.Result.FrameCount != StoryboardShortFrameCount ||
		repository.storyboardFailure != nil {
		t.Fatalf(
			"fallback calls/requests/ready/failure = %d/%#v/%#v/%#v",
			processor.calls,
			processor.requests,
			repository.storyboardReady,
			repository.storyboardFailure,
		)
	}
}

func TestStoryboardServiceMarksExhaustedBudgetAsDeterministicFailure(t *testing.T) {
	mtime := time.Unix(0, 100)
	fingerprint, _ := media.NewSourceFingerprint(6, mtime.UnixNano())
	duration := int64(68_000)
	repository := &repositoryStub{asset: Asset{
		ID: 9, LibraryID: 7, LibraryRoot: "family",
		RelativePath: "4k60.mp4", Kind: media.KindVideo,
		Format: media.FormatMP4, SourceFingerprint: fingerprint,
		Width: 3840, Height: 2160, DurationMS: &duration,
		ProbeStatus: media.ProbeReady, GridReady: true,
	}}
	processor := &storyboardProcessorStub{run: func(
		_ context.Context,
		_ media.StoryboardRequest,
	) (media.StoryboardResult, error) {
		return media.StoryboardResult{}, media.WithFailureDiagnostic(
			context.DeadlineExceeded,
			media.FailureDiagnostic{
				Stage:  media.StageFrameExtract,
				Reason: media.ReasonTimedOut,
				Tool:   "ffmpeg",
			},
		)
	}}
	service, err := NewStoryboardService(
		repository,
		sourceStub{file: sourceFileStub{
			Reader: bytes.NewReader([]byte("source")),
			info:   fileInfoStub{size: 6, mtime: mtime},
		}},
		&publisherStub{},
		&capacityStub{},
		processor,
		ServiceOptions{},
	)
	if err != nil {
		t.Fatal(err)
	}
	err = service.Process(context.Background(), 9)
	if !errors.Is(err, ErrStoryboardBudgetExhausted) ||
		!errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("exhausted storyboard error = %v", err)
	}
	if processor.calls != 2 || repository.storyboardFailure == nil ||
		repository.storyboardFailure.Code != media.ErrorProcessingTimed {
		t.Fatalf(
			"exhausted calls/failure = %d/%#v",
			processor.calls,
			repository.storyboardFailure,
		)
	}
}

func TestStoryboardServiceRejectsIneligibleAndPersistsProcessingFailure(t *testing.T) {
	mtime := time.Unix(0, 100)
	fingerprint, _ := media.NewSourceFingerprint(6, mtime.UnixNano())
	duration := int64(10_000)
	repository := &repositoryStub{asset: Asset{
		ID: 9, LibraryID: 7, Kind: media.KindVideo, Format: media.FormatMP4,
		SourceFingerprint: fingerprint, Width: 1920, Height: 1080,
		DurationMS: &duration, ProbeStatus: media.ProbeReady,
	}}
	processor := &storyboardProcessorStub{err: media.ErrInvalidMedia}
	service, err := NewStoryboardService(
		repository,
		sourceStub{file: sourceFileStub{
			Reader: bytes.NewReader([]byte("source")),
			info:   fileInfoStub{size: 6, mtime: mtime},
		}},
		&publisherStub{},
		&capacityStub{},
		processor,
		ServiceOptions{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Process(
		context.Background(),
		9,
	); !errors.Is(err, ErrStoryboardNotEligible) {
		t.Fatalf("ineligible error = %v", err)
	}
	if processor.calls != 0 || repository.storyboardFailure != nil {
		t.Fatalf(
			"ineligible processing = %d/%#v",
			processor.calls,
			repository.storyboardFailure,
		)
	}

	repository.asset.GridReady = true
	if err := service.Process(
		context.Background(),
		9,
	); !errors.Is(err, media.ErrInvalidMedia) {
		t.Fatalf("processing error = %v", err)
	}
	if repository.storyboardFailure == nil ||
		repository.storyboardFailure.Code != media.ErrorInvalidMedia {
		t.Fatalf("storyboard failure = %#v", repository.storyboardFailure)
	}
}

func TestStoryboardServiceFailsClosedAcrossCancellationInvalidOutputAndENOSPC(
	t *testing.T,
) {
	mtime := time.Unix(0, 100)
	fingerprint, _ := media.NewSourceFingerprint(6, mtime.UnixNano())
	duration := int64(10_000)
	asset := Asset{
		ID: 9, LibraryID: 7, LibraryRoot: "family",
		RelativePath: "clip.mp4", Kind: media.KindVideo,
		Format: media.FormatMP4, SourceFingerprint: fingerprint,
		Width: 1920, Height: 1080, DurationMS: &duration,
		ProbeStatus: media.ProbeReady, GridReady: true,
	}
	source := sourceStub{file: sourceFileStub{
		Reader: bytes.NewReader([]byte("source")),
		info:   fileInfoStub{size: 6, mtime: mtime},
	}}
	validResult := func(request media.StoryboardRequest) media.StoryboardResult {
		return media.StoryboardResult{
			Bytes:      []byte("RIFF\x04\x00\x00\x00WEBP"),
			FrameCount: len(request.TimestampsMS),
			Columns:    request.Columns,
			Rows:       request.Rows,
			CellWidth:  request.CellWidth,
			CellHeight: request.CellHeight,
		}
	}

	t.Run("cancellation after native processing", func(t *testing.T) {
		repository := &repositoryStub{asset: asset}
		publisher := &publisherStub{}
		ctx, cancel := context.WithCancel(context.Background())
		processor := &storyboardProcessorStub{run: func(
			_ context.Context,
			request media.StoryboardRequest,
		) (media.StoryboardResult, error) {
			cancel()
			return validResult(request), nil
		}}
		service, err := NewStoryboardService(
			repository,
			source,
			publisher,
			&capacityStub{},
			processor,
			ServiceOptions{},
		)
		if err != nil {
			t.Fatal(err)
		}
		if err := service.Process(ctx, 9); !errors.Is(err, context.Canceled) {
			t.Fatalf("cancelled storyboard error = %v", err)
		}
		if publisher.calls != 0 ||
			repository.storyboardReady != nil ||
			repository.storyboardFailure != nil {
			t.Fatalf(
				"cancelled storyboard published/committed = %d/%#v/%#v",
				publisher.calls,
				repository.storyboardReady,
				repository.storyboardFailure,
			)
		}
	})

	t.Run("invalid sprite", func(t *testing.T) {
		repository := &repositoryStub{asset: asset}
		publisher := &publisherStub{}
		processor := &storyboardProcessorStub{run: func(
			_ context.Context,
			request media.StoryboardRequest,
		) (media.StoryboardResult, error) {
			result := validResult(request)
			result.Bytes = []byte("not-webp")
			return result, nil
		}}
		service, err := NewStoryboardService(
			repository,
			source,
			publisher,
			&capacityStub{},
			processor,
			ServiceOptions{},
		)
		if err != nil {
			t.Fatal(err)
		}
		if err := service.Process(
			context.Background(),
			9,
		); !errors.Is(err, ErrInvalidState) {
			t.Fatalf("invalid storyboard output error = %v", err)
		}
		if publisher.calls != 0 ||
			repository.storyboardReady != nil ||
			repository.storyboardFailure != nil {
			t.Fatal("invalid storyboard output became visible")
		}
	})

	t.Run("cache publication ENOSPC", func(t *testing.T) {
		repository := &repositoryStub{asset: asset}
		derivation, err := StoryboardDerivation(7, 9, fingerprint)
		if err != nil {
			t.Fatal(err)
		}
		cachePath, err := derivation.CacheRelativePath()
		if err != nil {
			t.Fatal(err)
		}
		publisher := &publisherStub{
			published: Published{CacheRelativePath: cachePath},
			err:       errors.New("injected ENOSPC"),
		}
		processor := &storyboardProcessorStub{run: func(
			_ context.Context,
			request media.StoryboardRequest,
		) (media.StoryboardResult, error) {
			return validResult(request), nil
		}}
		service, err := NewStoryboardService(
			repository,
			source,
			publisher,
			&capacityStub{},
			processor,
			ServiceOptions{},
		)
		if err != nil {
			t.Fatal(err)
		}
		if err := service.Process(
			context.Background(),
			9,
		); !errors.Is(err, ErrPublishFailed) {
			t.Fatalf("storyboard ENOSPC error = %v", err)
		}
		if repository.storyboardReady != nil ||
			repository.storyboardFailure != nil {
			t.Fatal("storyboard ENOSPC committed visible state")
		}
	})
}

func TestServicePersistsStableProcessingFailureButNotSourceChange(t *testing.T) {
	mtime := time.Unix(0, 100)
	fingerprint, _ := media.NewSourceFingerprint(6, mtime.UnixNano())
	repository := &repositoryStub{asset: Asset{
		ID: 9, LibraryID: 7, Kind: media.KindImage, Format: media.FormatPNG,
		SourceFingerprint: fingerprint,
	}}
	file := sourceFileStub{
		Reader: bytes.NewReader([]byte("source")),
		info:   fileInfoStub{size: 6, mtime: mtime},
	}
	image := &processorStub{err: media.ErrInvalidMedia}
	service, err := NewService(
		repository, sourceStub{file: file}, &publisherStub{},
		&capacityStub{}, image, &processorStub{}, ServiceOptions{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Process(context.Background(), 9); !errors.Is(err, media.ErrInvalidMedia) {
		t.Fatalf("processing error = %v", err)
	}
	if repository.failure == nil || repository.failure.Code != media.ErrorInvalidMedia {
		t.Fatalf("failure = %#v", repository.failure)
	}

	repository.failure = nil
	file.info = fileInfoStub{size: 7, mtime: mtime}
	service.source = sourceStub{file: file}
	if err := service.Process(context.Background(), 9); !errors.Is(err, ErrSourceChanged) {
		t.Fatalf("source change error = %v", err)
	}
	if repository.failure != nil {
		t.Fatalf("source change persisted failure %#v", repository.failure)
	}
}

func TestServiceRejectsOversizedSourceBeforeNativeProcessor(t *testing.T) {
	mtime := time.Unix(0, 100)
	size := int64(media.MaxImageSourceBytes + 1)
	fingerprint, _ := media.NewSourceFingerprint(size, mtime.UnixNano())
	repository := &repositoryStub{asset: Asset{
		ID: 9, LibraryID: 7, Kind: media.KindImage, Format: media.FormatJPEG,
		SourceFingerprint: fingerprint,
	}}
	image := &processorStub{}
	service, err := NewService(
		repository,
		sourceStub{file: sourceFileStub{
			Reader: bytes.NewReader(nil),
			info:   fileInfoStub{size: size, mtime: mtime},
		}},
		&publisherStub{},
		&capacityStub{},
		image,
		&processorStub{},
		ServiceOptions{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Process(
		context.Background(), 9,
	); !errors.Is(err, media.ErrSourceTooLarge) {
		t.Fatalf("oversized source error = %v", err)
	}
	if image.calls != 0 || repository.failure == nil ||
		repository.failure.Code != media.ErrorProcessingFailed {
		t.Fatalf(
			"native calls/failure = %d, %#v",
			image.calls, repository.failure,
		)
	}
}

func TestServiceObservesCancellationAfterNativeCallBeforePublish(t *testing.T) {
	mtime := time.Unix(0, 100)
	fingerprint, _ := media.NewSourceFingerprint(6, mtime.UnixNano())
	repository := &repositoryStub{asset: Asset{
		ID: 9, LibraryID: 7, Kind: media.KindImage, Format: media.FormatJPEG,
		SourceFingerprint: fingerprint,
	}}
	ctx, cancel := context.WithCancel(context.Background())
	image := &processorStub{run: func(
		context.Context,
	) (media.ProcessingResult, error) {
		cancel()
		return media.ProcessingResult{
			Metadata: media.Metadata{
				Width: 10, Height: 10,
				PlaybackStatus: media.PlaybackNotApplicable,
			},
			Thumbnail: media.Thumbnail{
				Bytes: []byte("webp"), Width: 10, Height: 10,
			},
		}, nil
	}}
	publisher := &publisherStub{}
	service, err := NewService(
		repository,
		sourceStub{file: sourceFileStub{
			Reader: bytes.NewReader([]byte("source")),
			info:   fileInfoStub{size: 6, mtime: mtime},
		}},
		publisher,
		&capacityStub{},
		image,
		&processorStub{},
		ServiceOptions{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Process(ctx, 9); !errors.Is(err, context.Canceled) {
		t.Fatalf("post-native cancellation error = %v", err)
	}
	if publisher.calls != 0 || repository.ready != nil || repository.failure != nil {
		t.Fatalf(
			"cancelled publication = calls %d ready %#v failure %#v",
			publisher.calls, repository.ready, repository.failure,
		)
	}
}

func TestServiceDoesNotCommitReadyAfterCacheWriteFailure(t *testing.T) {
	mtime := time.Unix(0, 100)
	fingerprint, _ := media.NewSourceFingerprint(6, mtime.UnixNano())
	repository := &repositoryStub{asset: Asset{
		ID: 9, LibraryID: 7, Kind: media.KindImage, Format: media.FormatJPEG,
		SourceFingerprint: fingerprint,
	}}
	result := media.ProcessingResult{
		Metadata: media.Metadata{
			Width: 10, Height: 10, PlaybackStatus: media.PlaybackNotApplicable,
		},
		Thumbnail: media.Thumbnail{
			Bytes: []byte("webp"), Width: 10, Height: 10,
		},
	}
	publisher := &publisherStub{
		published: Published{CacheRelativePath: "unused"},
		err:       errors.New("injected ENOSPC"),
	}
	service, err := NewService(
		repository,
		sourceStub{file: sourceFileStub{
			Reader: bytes.NewReader([]byte("source")),
			info:   fileInfoStub{size: 6, mtime: mtime},
		}},
		publisher,
		&capacityStub{},
		&processorStub{result: result},
		&processorStub{},
		ServiceOptions{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Process(
		context.Background(), 9,
	); !errors.Is(err, ErrPublishFailed) {
		t.Fatalf("publish failure error = %v", err)
	}
	if repository.ready != nil || repository.failure != nil {
		t.Fatalf(
			"cache failure committed state ready %#v failure %#v",
			repository.ready, repository.failure,
		)
	}
}
