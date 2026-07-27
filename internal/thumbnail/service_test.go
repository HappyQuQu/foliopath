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
	asset     Asset
	ready     *Ready
	failure   *Failure
	commitErr error
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
}

func (stub *processorStub) Process(
	context.Context,
	io.ReadSeeker,
	media.Format,
) (media.ProcessingResult, error) {
	stub.calls++
	return stub.result, stub.err
}

type publisherStub struct {
	published Published
	calls     int
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
	return stub.published, nil
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
