package aimodel

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"strings"
	"sync"
)

const MaxDownloadCatalogBytes = 1 << 20

var ErrDownloadUnavailable = errors.New("model download unavailable")
var ErrDownloadActive = errors.New("model download already active")

// DownloadDefinition is deployment-owned input, never accepted from the API.
type DownloadDefinition struct {
	ID                   string            `json:"id"`
	Name                 string            `json:"name"`
	SourceLabel          string            `json:"sourceLabel"`
	Manifest             Manifest          `json:"manifest"`
	ContentHash          string            `json:"contentHash"`
	RuntimeArchitectures []string          `json:"runtimeArchitectures"`
	Files                map[string]string `json:"files"`
	RedirectHosts        []string          `json:"redirectHosts"`
}

type DownloadCatalog struct {
	entries []DownloadDefinition
	catalog *Catalog
	arch    string
}

func ParseDownloadCatalog(data []byte, architecture string) (*DownloadCatalog, error) {
	var entries []DownloadDefinition
	if len(data) > 0 {
		if len(data) > MaxDownloadCatalogBytes || rejectDuplicateJSONKeys(data) != nil {
			return nil, ErrInvalidModel
		}
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.DisallowUnknownFields()
		if decoder.Decode(&entries) != nil || entries == nil || ensureJSONEOF(decoder) != nil {
			return nil, ErrInvalidModel
		}
	}
	if len(entries) > 16 || (architecture != "amd64" && architecture != "arm64") {
		return nil, ErrInvalidModel
	}
	seen := map[string]bool{}
	packages := make([]CatalogEntry, 0, len(entries))
	for _, entry := range entries {
		if !facePackageIDPattern.MatchString(entry.ID) || len(entry.ID) < 8 || strings.ContainsAny(entry.ID, ".+") ||
			seen[entry.ID] || strings.TrimSpace(entry.Name) == "" || len(entry.Name) > 128 ||
			strings.TrimSpace(entry.SourceLabel) == "" || len(entry.SourceLabel) > 128 ||
			len(entry.Files) != len(entry.Manifest.Files) || len(entry.RedirectHosts) > 8 {
			return nil, ErrInvalidModel
		}
		seen[entry.ID] = true
		for _, file := range entry.Manifest.Files {
			if !ValidDownloadURL(entry.Files[file.Name]) {
				return nil, ErrInvalidModel
			}
		}
		for _, host := range entry.RedirectHosts {
			parsed, err := url.Parse("https://" + host)
			if err != nil || parsed.Hostname() != host || !ValidDownloadURL(parsed.String()) {
				return nil, ErrInvalidModel
			}
		}
		packages = append(packages, CatalogEntry{Manifest: entry.Manifest, ContentHash: entry.ContentHash, RuntimeArchitectures: entry.RuntimeArchitectures})
	}
	catalog, err := NewCatalog(packages)
	if err != nil {
		return nil, err
	}
	return &DownloadCatalog{entries: entries, catalog: catalog, arch: architecture}, nil
}

func ValidDownloadURL(value string) bool {
	u, err := url.Parse(value)
	return err == nil && u.Scheme == "https" && u.Hostname() != "" && u.User == nil &&
		u.Fragment == "" && (u.Port() == "" || u.Port() == "443") && len(value) <= 4096
}

func (catalog *DownloadCatalog) Packages() *Catalog { return catalog.catalog }

func (catalog *DownloadCatalog) candidate(id string) (Candidate, DownloadDefinition, error) {
	for _, entry := range catalog.entries {
		if entry.ID != id {
			continue
		}
		model, manifest, ok := catalog.catalog.PackageByContentHash(entry.ContentHash, catalog.arch)
		if !ok {
			return Candidate{}, DownloadDefinition{}, ErrModelIncompatible
		}
		return Candidate{ID: entry.ID, Package: model, Manifest: manifest, Compatibility: "compatible", SourceIdentity: "download:" + entry.ContentHash}, entry, nil
	}
	return Candidate{}, DownloadDefinition{}, ErrDownloadUnavailable
}

type DownloadFileSource interface {
	OpenDownload(context.Context, string, []string, int64) (io.ReadCloser, error)
}

// DownloadSource shares the installer's existing verified, atomic publisher.
type DownloadSource struct {
	Catalog *DownloadCatalog
	Remote  DownloadFileSource
	Offline PackageSource
}

func (source *DownloadSource) OpenModelPackageFile(ctx context.Context, identity, name string) (io.ReadCloser, int64, error) {
	if !strings.HasPrefix(identity, "download:") {
		return source.Offline.OpenModelPackageFile(ctx, identity, name)
	}
	for _, entry := range source.Catalog.entries {
		if identity != "download:"+entry.ContentHash {
			continue
		}
		for _, file := range entry.Manifest.Files {
			if file.Name == name {
				reader, err := source.Remote.OpenDownload(ctx, entry.Files[name], entry.RedirectHosts, file.Size)
				return reader, file.Size, err
			}
		}
	}
	return nil, 0, ErrDownloadUnavailable
}

func (source *DownloadSource) ValidateDirectModelSource(ctx context.Context, identity string) error {
	if strings.HasPrefix(identity, "download:") {
		return ErrModelIncompatible
	}
	return source.Offline.ValidateDirectModelSource(ctx, identity)
}

type DownloadHistory interface {
	LatestAIModelInstall(context.Context, string) (Operation, bool, error)
}

type DownloadOffer struct {
	ID, Name, SourceLabel, Compatibility string
	Purpose, Version, LicenseID          string
	PackageSizeBytes                     int64
	InstalledModelID                     string
	Operation                            *Operation
}

type DownloadPage struct {
	Items            []DownloadOffer
	Presets          []ModelPreset
	RuntimeAvailable bool
}

type DownloadService struct {
	mu      sync.Mutex
	catalog *DownloadCatalog
	models  *Service
	install InstallAdmission
	history DownloadHistory
	runtime bool
}

func NewDownloadService(catalog *DownloadCatalog, models *Service, admission InstallAdmission, history DownloadHistory, runtime bool) (*DownloadService, error) {
	if catalog == nil || models == nil || admission == nil || history == nil {
		return nil, ErrInvalidModel
	}
	return &DownloadService{catalog: catalog, models: models, install: admission, history: history, runtime: runtime}, nil
}

func (service *DownloadService) ListDownloads(ctx context.Context) (DownloadPage, error) {
	models, err := service.models.List(ctx)
	if err != nil {
		return DownloadPage{}, err
	}
	page := DownloadPage{Items: make([]DownloadOffer, 0, len(service.catalog.entries)), Presets: BuiltinModelPresets(), RuntimeAvailable: service.runtime}
	for _, entry := range service.catalog.entries {
		offer := DownloadOffer{ID: entry.ID, Name: entry.Name, SourceLabel: entry.SourceLabel,
			Purpose: entry.Manifest.Purpose, Version: entry.Manifest.Version, LicenseID: manifestLicenseSummary(entry.Manifest), Compatibility: "compatible"}
		if _, _, err := service.catalog.candidate(entry.ID); err != nil {
			offer.Compatibility = "architecture_mismatch"
		}
		for _, file := range entry.Manifest.Files {
			offer.PackageSizeBytes += file.Size
		}
		for _, model := range models.Items {
			if model.Package.ContentHash == entry.ContentHash && model.State != StateUnavailable {
				offer.InstalledModelID = model.ID
			}
		}
		operation, found, err := service.history.LatestAIModelInstall(ctx, entry.ID)
		if err != nil {
			return DownloadPage{}, err
		}
		if found {
			offer.Operation = &operation
		}
		page.Items = append(page.Items, offer)
	}
	return page, nil
}

func (service *DownloadService) StartDownload(ctx context.Context, id, key string) (InstallResult, error) {
	if len(key) < 8 || len(key) > 128 {
		return InstallResult{}, ErrInvalidModel
	}
	service.mu.Lock()
	defer service.mu.Unlock()
	if result, found, err := service.install.ReplayInstall(ctx, id, StorageManaged, key); err != nil || found {
		return result, err
	}
	candidate, _, err := service.catalog.candidate(id)
	if err != nil {
		return InstallResult{}, err
	}
	latest, found, err := service.history.LatestAIModelInstall(ctx, id)
	if err != nil {
		return InstallResult{}, err
	}
	if found && !isTerminalOperationState(latest.State) {
		return InstallResult{}, ErrDownloadActive
	}
	return service.install.StartInstall(ctx, candidate, StorageManaged, key)
}
