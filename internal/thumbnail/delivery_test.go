package thumbnail

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/media"
)

type deliveryRepositoryStub struct {
	state         DeliveryState
	touches       int
	requeues      int
	repositoryErr error
}

func (stub *deliveryRepositoryStub) GetThumbnailDelivery(
	context.Context,
	int64,
) (DeliveryState, error) {
	return stub.state, stub.repositoryErr
}

func (stub *deliveryRepositoryStub) TouchThumbnail(
	context.Context,
	int64,
	media.SourceFingerprint,
	string,
) error {
	stub.touches++
	return stub.repositoryErr
}

func (stub *deliveryRepositoryStub) RequeueMissingThumbnail(
	context.Context,
	DeliveryState,
) error {
	stub.requeues++
	return stub.repositoryErr
}

type deliveryCacheStub struct {
	content CacheContent
	err     error
}

func (stub deliveryCacheStub) Open(context.Context, string) (CacheContent, error) {
	return stub.content, stub.err
}

type closeableReader struct {
	*bytes.Reader
	closed bool
}

func (reader *closeableReader) Close() error {
	reader.closed = true
	return nil
}

type deliveryWakerStub struct{ calls int }

func (waker *deliveryWakerStub) Wake() { waker.calls++ }

func TestDeliveryServiceReturnsReadyContentAndTouchesLRU(t *testing.T) {
	fingerprint := media.SourceFingerprint("v1:4:5")
	state := DeliveryState{
		AssetID: 7, SourceFingerprint: fingerprint, Status: DeliveryReady,
		CacheRelativePath: "libraries/lib_1/aa/key.webp", ByteSize: 4,
	}
	reader := &closeableReader{Reader: bytes.NewReader([]byte("webp"))}
	repository := &deliveryRepositoryStub{state: state}
	waker := &deliveryWakerStub{}
	service, err := NewDeliveryService(
		repository,
		deliveryCacheStub{content: CacheContent{Reader: reader, ByteSize: 4}},
		waker,
	)
	if err != nil {
		t.Fatal(err)
	}
	got, err := service.Get(context.Background(), 7)
	if err != nil {
		t.Fatal(err)
	}
	value, err := io.ReadAll(got.Content)
	if err != nil {
		t.Fatal(err)
	}
	_ = got.Content.Close()
	if string(value) != "webp" || got.Status != DeliveryReady ||
		got.ContentBytes != 4 || got.ETag == "" ||
		repository.touches != 1 || repository.requeues != 0 ||
		waker.calls != 0 {
		t.Fatalf("delivery = %#v; repository = %#v; wakes = %d", got, repository, waker.calls)
	}
}

func TestDeliveryServiceRepairsMissingCacheWithoutSynchronousProcessing(t *testing.T) {
	repository := &deliveryRepositoryStub{state: DeliveryState{
		AssetID: 7, SourceFingerprint: media.SourceFingerprint("v1:4:5"),
		Status: DeliveryReady, CacheRelativePath: "missing.webp", ByteSize: 4,
	}}
	waker := &deliveryWakerStub{}
	service, err := NewDeliveryService(
		repository, deliveryCacheStub{err: ErrCacheEntryMissing}, waker,
	)
	if err != nil {
		t.Fatal(err)
	}
	got, err := service.Get(context.Background(), 7)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != DeliveryQueued || got.RetryAfterMS != ThumbnailRetryAfterMS ||
		repository.requeues != 1 || repository.touches != 0 || waker.calls != 1 {
		t.Fatalf("delivery = %#v; repository = %#v; wakes = %d", got, repository, waker.calls)
	}
}

func TestDeliveryServicePreservesSafePendingFailedAndOfflineStates(t *testing.T) {
	tests := []struct {
		name  string
		state DeliveryState
	}{
		{name: "queued", state: DeliveryState{Status: DeliveryQueued}},
		{name: "running", state: DeliveryState{Status: DeliveryRunning}},
		{name: "failed", state: DeliveryState{
			Status: DeliveryFailed, ErrorCode: media.ErrorInvalidMedia,
		}},
		{name: "offline", state: DeliveryState{
			Status: DeliveryOffline, ErrorCode: media.ProcessingErrorCode("source_offline"),
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := &deliveryRepositoryStub{state: test.state}
			service, err := NewDeliveryService(
				repository, deliveryCacheStub{err: errors.New("must not open")}, &deliveryWakerStub{},
			)
			if err != nil {
				t.Fatal(err)
			}
			got, err := service.Get(context.Background(), 7)
			if err != nil {
				t.Fatal(err)
			}
			if got.Status != test.state.Status || got.ErrorCode != test.state.ErrorCode {
				t.Fatalf("delivery = %#v", got)
			}
		})
	}
}
