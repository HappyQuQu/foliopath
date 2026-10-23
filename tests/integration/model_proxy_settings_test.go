package integration

import (
	"context"
	"errors"
	"github.com/HappyQuQu/foliopath/internal/aimodel"
	"github.com/HappyQuQu/foliopath/internal/aimodel/downloadhttp"
	"github.com/HappyQuQu/foliopath/internal/store/sqlite"
	"io"
	"path/filepath"
	"strings"
	"testing"
)

type proxyProbe struct{ last string }

func (p *proxyProbe) ValidateProxy(v string) error                 { _, e := downloadhttp.ParseProxyURL(v); return e }
func (p *proxyProbe) ProbeProxy(_ context.Context, v string) error { p.last = v; return nil }
func (p *proxyProbe) OpenProxiedDownload(_ context.Context, v, _ string, _ []string, _ int64) (io.ReadCloser, error) {
	p.last = v
	return io.NopCloser(strings.NewReader("fixture")), nil
}
func TestModelProxyPersistenceAndLivePrecedence(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "proxy.db")
	store, err := sqlite.Open(ctx, path, sqlite.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { store.Close() }()
	transport := &proxyProbe{}
	s, err := aimodel.NewProxySettingsService(store, transport, "http://192.168.2.231:7890")
	if err != nil {
		t.Fatal(err)
	}
	v, err := s.Get(ctx)
	if err != nil || v.Mode != "inherit" || !v.DeploymentConfigured || v.EffectiveMode != "proxy" {
		t.Fatal(v, err)
	}
	custom := aimodel.ProxySettings{Mode: "custom", URL: "http://proxy.example:7890", Revision: v.Revision}
	if ok, err := s.Test(ctx, custom); err != nil || !ok {
		t.Fatal(err)
	}
	unchanged, _ := s.Get(ctx)
	if unchanged.Mode != "inherit" {
		t.Fatal("probe saved draft")
	}
	v, err = s.Save(ctx, custom)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Save(ctx, custom); !errors.Is(err, aimodel.ErrPreconditionFailed) {
		t.Fatal("stale update accepted", err)
	}
	r, err := s.OpenDownload(ctx, "https://models.example/file", nil, 7)
	if err != nil {
		t.Fatal(err)
	}
	r.Close()
	if transport.last != custom.URL {
		t.Fatal("custom not effective")
	}
	store.Close()
	store, err = sqlite.Open(ctx, path, sqlite.Options{})
	if err != nil {
		t.Fatal(err)
	}
	s, err = aimodel.NewProxySettingsService(store, transport, "http://deployment.example:7890")
	if err != nil {
		t.Fatal(err)
	}
	v, err = s.Get(ctx)
	if err != nil || v.URL != custom.URL {
		t.Fatal("restart lost settings", v, err)
	}
	v, err = s.Save(ctx, aimodel.ProxySettings{Mode: "direct", Revision: v.Revision})
	if err != nil || v.EffectiveMode != "direct" || v.URL != "" {
		t.Fatal(v, err)
	}
	r, err = s.OpenDownload(ctx, "https://models.example/file", nil, 7)
	if err != nil {
		t.Fatal(err)
	}
	r.Close()
	if transport.last != "" {
		t.Fatal("direct did not override deployment")
	}
	v, err = s.Save(ctx, aimodel.ProxySettings{Mode: "inherit", Revision: v.Revision})
	if err != nil {
		t.Fatal(err)
	}
	r, err = s.OpenDownload(ctx, "https://models.example/file", nil, 7)
	if err != nil {
		t.Fatal(err)
	}
	r.Close()
	if transport.last != "http://deployment.example:7890" {
		t.Fatal("inherit did not restore deployment")
	}
	if _, err = s.Save(ctx, aimodel.ProxySettings{Mode: "custom", URL: "http://u:secret@proxy.example", Revision: v.Revision}); !errors.Is(err, aimodel.ErrInvalidProxySettings) {
		t.Fatal("credentials accepted")
	}
}
