package aimodel

import (
	"context"
	"errors"
	"io"
	"sync"
	"time"
)

var ErrInvalidProxySettings = errors.New("invalid model proxy settings")

type ProxySettings struct {
	Mode, URL string
	Revision  int64
}
type ProxySnapshot struct {
	ProxySettings
	DeploymentConfigured bool
	EffectiveMode        string
}
type ProxySettingsRepository interface {
	GetModelProxy(context.Context) (ProxySettings, error)
	SaveModelProxy(context.Context, ProxySettings) (ProxySettings, error)
}
type ProxyTransport interface {
	ValidateProxy(string) error
	OpenProxiedDownload(context.Context, string, string, []string, int64) (io.ReadCloser, error)
	ProbeProxy(context.Context, string) error
}
type ProxySettingsService struct {
	repository ProxySettingsRepository
	transport  ProxyTransport
	deployment string
	probe      sync.Mutex
}

func NewProxySettingsService(repository ProxySettingsRepository, transport ProxyTransport, deployment string) (*ProxySettingsService, error) {
	if repository == nil || transport == nil || transport.ValidateProxy(deployment) != nil {
		return nil, ErrInvalidProxySettings
	}
	return &ProxySettingsService{repository: repository, transport: transport, deployment: deployment}, nil
}
func (s *ProxySettingsService) validate(v ProxySettings) error {
	if v.Mode != "inherit" && v.Mode != "direct" && v.Mode != "custom" {
		return ErrInvalidProxySettings
	}
	if v.Mode == "custom" {
		if v.URL == "" || s.transport.ValidateProxy(v.URL) != nil {
			return ErrInvalidProxySettings
		}
	} else if v.URL != "" {
		return ErrInvalidProxySettings
	}
	return nil
}
func (s *ProxySettingsService) effective(v ProxySettings) string {
	if v.Mode == "inherit" {
		return s.deployment
	}
	if v.Mode == "custom" {
		return v.URL
	}
	return ""
}
func (s *ProxySettingsService) snapshot(v ProxySettings) ProxySnapshot {
	mode := "direct"
	if s.effective(v) != "" {
		mode = "proxy"
	}
	return ProxySnapshot{ProxySettings: v, DeploymentConfigured: s.deployment != "", EffectiveMode: mode}
}
func (s *ProxySettingsService) Get(ctx context.Context) (ProxySnapshot, error) {
	v, err := s.repository.GetModelProxy(ctx)
	if err != nil {
		return ProxySnapshot{}, err
	}
	return s.snapshot(v), nil
}
func (s *ProxySettingsService) Save(ctx context.Context, v ProxySettings) (ProxySnapshot, error) {
	if s.validate(v) != nil || v.Revision < 1 {
		return ProxySnapshot{}, ErrInvalidProxySettings
	}
	v, err := s.repository.SaveModelProxy(ctx, v)
	if err != nil {
		return ProxySnapshot{}, err
	}
	return s.snapshot(v), nil
}
func (s *ProxySettingsService) Test(ctx context.Context, v ProxySettings) (bool, error) {
	if err := s.validate(v); err != nil {
		return false, err
	}
	if !s.probe.TryLock() {
		return false, ErrDownloadActive
	}
	defer s.probe.Unlock()
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	return s.transport.ProbeProxy(ctx, s.effective(v)) == nil, nil
}
func (s *ProxySettingsService) OpenDownload(ctx context.Context, location string, hosts []string, size int64) (io.ReadCloser, error) {
	v, err := s.repository.GetModelProxy(ctx)
	if err != nil {
		return nil, err
	}
	if err = s.validate(v); err != nil {
		return nil, err
	}
	return s.transport.OpenProxiedDownload(ctx, s.effective(v), location, hosts, size)
}
