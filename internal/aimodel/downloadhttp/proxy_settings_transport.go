package downloadhttp

import (
	"context"
	"crypto/sha256"
	"fmt"
	"github.com/HappyQuQu/foliopath/internal/aimodel"
	"io"
	"net/http"
)

type ProxySettingsTransport struct{}

func (ProxySettingsTransport) ValidateProxy(v string) error { _, err := ParseProxyURL(v); return err }

type ownedProxyBody struct {
	io.ReadCloser
	transport *http.Transport
}

func (b *ownedProxyBody) Close() error {
	err := b.ReadCloser.Close()
	b.transport.CloseIdleConnections()
	return err
}
func (ProxySettingsTransport) OpenProxiedDownload(ctx context.Context, proxy, location string, hosts []string, size int64) (io.ReadCloser, error) {
	source, err := NewWithProxy(proxy)
	if err != nil {
		return nil, err
	}
	transport := source.transport.(*http.Transport)
	reader, err := source.OpenDownload(ctx, location, hosts, size)
	if err != nil {
		transport.CloseIdleConnections()
		return nil, err
	}
	return &ownedProxyBody{ReadCloser: reader, transport: transport}, nil
}
func (t ProxySettingsTransport) ProbeProxy(ctx context.Context, proxy string) error {
	r, err := t.OpenProxiedDownload(ctx, proxy, "https://huggingface.co/google/siglip-base-patch16-224/resolve/7fd15f0689c79d79e38b1c2e2e2370a7bf2761ed/spiece.model", []string{"us.aws.cdn.hf.co"}, 798330)
	if err != nil {
		return err
	}
	defer r.Close()
	h := sha256.New()
	n, err := io.Copy(h, r)
	if err != nil || n != 798330 || fmt.Sprintf("%x", h.Sum(nil)) != "1e5036bed065526c3c212dfbe288752391797c4bb1a284aa18c9a0b23fcaf8ec" {
		return aimodel.ErrDownloadUnavailable
	}
	return nil
}
