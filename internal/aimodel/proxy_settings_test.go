package aimodel

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"
)

type blockingProxyTransport struct{ entered chan struct{} }

func (*blockingProxyTransport) ValidateProxy(string) error { return nil }
func (*blockingProxyTransport) OpenProxiedDownload(context.Context, string, string, []string, int64) (io.ReadCloser, error) {
	return nil, errors.New("unused")
}
func (p *blockingProxyTransport) ProbeProxy(ctx context.Context, _ string) error {
	close(p.entered)
	<-ctx.Done()
	return ctx.Err()
}

type unusedProxyRepository struct{}

func (unusedProxyRepository) GetModelProxy(context.Context) (ProxySettings, error) {
	return ProxySettings{}, errors.New("unused")
}
func (unusedProxyRepository) SaveModelProxy(context.Context, ProxySettings) (ProxySettings, error) {
	return ProxySettings{}, errors.New("unused")
}
func TestProxyProbeIsBoundedAndCancellationReleasesSlot(t *testing.T) {
	p := &blockingProxyTransport{entered: make(chan struct{})}
	s, _ := NewProxySettingsService(unusedProxyRepository{}, p, "")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() {
		defer close(done)
		ok, err := s.Test(ctx, ProxySettings{Mode: "direct", Revision: 1})
		if ok || err != nil {
			t.Error(ok, err)
		}
	}()
	select {
	case <-p.entered:
	case <-time.After(time.Second):
		t.Fatal("probe not started")
	}
	if _, err := s.Test(context.Background(), ProxySettings{Mode: "direct", Revision: 1}); !errors.Is(err, ErrDownloadActive) {
		t.Fatal("parallel probe accepted", err)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("cancel did not release probe")
	}
	if !s.probe.TryLock() {
		t.Fatal("probe slot leaked")
	}
	s.probe.Unlock()
}
