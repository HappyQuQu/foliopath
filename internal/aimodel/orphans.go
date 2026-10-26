package aimodel

import (
	"context"
	"errors"
	"time"
)

const MaxManagedFinals = 256

type ManagedOrphanSummary struct {
	Scanned      int
	Reviewed     int
	Registered   int
	Unrecognized int
	Invalid      int
	Truncated    bool
}

// ManagedOrphanService reconciles complete filesystem reports with the
// reviewed catalog. It registers exact reviewed packages as available but
// never activates or deletes a package. Unknown or corrupt finals remain
// untouched and invisible to the installed-model list.
type ManagedOrphanService struct {
	models              *Service
	catalog             *Catalog
	source              ManagedActivationSource
	runtimeArchitecture string
}

func NewManagedOrphanService(models *Service, catalog *Catalog, source ManagedActivationSource, runtimeArchitecture string) (*ManagedOrphanService, error) {
	if models == nil || catalog == nil || source == nil ||
		(runtimeArchitecture != "amd64" && runtimeArchitecture != "arm64") {
		return nil, errors.New("managed AI model orphan dependencies are required")
	}
	return &ManagedOrphanService{models: models, catalog: catalog, source: source, runtimeArchitecture: runtimeArchitecture}, nil
}

func (service *ManagedOrphanService) Reconcile(ctx context.Context, contentHashes []string, complete bool) (ManagedOrphanSummary, error) {
	summary := ManagedOrphanSummary{Scanned: len(contentHashes), Truncated: !complete}
	if len(contentHashes) > MaxManagedFinals {
		return ManagedOrphanSummary{}, ErrRepositoryState
	}
	previous := ""
	for _, contentHash := range contentHashes {
		if !hexSHA256.MatchString(contentHash) || contentHash <= previous {
			return ManagedOrphanSummary{}, ErrRepositoryState
		}
		previous = contentHash
	}
	if !complete {
		return summary, nil
	}
	for _, contentHash := range contentHashes {
		if err := ctx.Err(); err != nil {
			return summary, err
		}
		verified, manifest, reviewed := service.catalog.PackageByContentHash(contentHash, service.runtimeArchitecture)
		if !reviewed {
			summary.Unrecognized++
			continue
		}
		summary.Reviewed++
		now := time.Unix(1, 0).UTC()
		probe := Model{
			ID: "aim_orphan_probe", Package: verified, StorageMode: StorageManaged,
			State: StateAvailable, SourceIdentity: "managed:" + contentHash,
			AvailabilityRevision: 1, CreatedAt: now, UpdatedAt: now,
		}
		if err := service.source.ValidateManagedModelPackage(ctx, probe, manifest); err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || ctx.Err() != nil {
				return summary, err
			}
			summary.Invalid++
			continue
		}
		_, created, err := service.models.RegisterInstalled(ctx, verified, StorageManaged, probe.SourceIdentity)
		if err != nil {
			return summary, err
		}
		if created {
			summary.Registered++
		}
	}
	return summary, nil
}
