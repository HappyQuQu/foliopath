package aimodel

import (
	"context"
	"io"
)

type PackageOpener func(context.Context, string) (io.ReadCloser, int64, error)

type PackageSource interface {
	OpenModelPackageFile(context.Context, string, string) (io.ReadCloser, int64, error)
	ValidateDirectModelSource(context.Context, string) error
}

type ManagedPublisher interface {
	PublishModelPackage(context.Context, VerifiedPackage, Manifest, PackageOpener) (string, error)
}

type Installer struct {
	service   *Service
	source    PackageSource
	publisher ManagedPublisher
}

func NewInstaller(service *Service, source PackageSource, publisher ManagedPublisher) (*Installer, error) {
	if service == nil || source == nil || publisher == nil {
		return nil, ErrInvalidModel
	}
	return &Installer{service: service, source: source, publisher: publisher}, nil
}

func (installer *Installer) Install(
	ctx context.Context,
	candidate Candidate,
	storageMode StorageMode,
) (Model, bool, error) {
	if candidate.Compatibility != "compatible" || ValidatePackage(candidate.Package) != nil ||
		validateManifest(candidate.Manifest) != nil || candidate.SourceIdentity == "" {
		return Model{}, false, ErrModelIncompatible
	}
	sourceIdentity := candidate.SourceIdentity
	switch storageMode {
	case StorageDirect:
		if err := installer.source.ValidateDirectModelSource(ctx, sourceIdentity); err != nil {
			return Model{}, false, err
		}
	case StorageManaged:
		managedIdentity, err := installer.publisher.PublishModelPackage(
			ctx,
			candidate.Package,
			candidate.Manifest,
			func(openContext context.Context, name string) (io.ReadCloser, int64, error) {
				return installer.source.OpenModelPackageFile(openContext, sourceIdentity, name)
			},
		)
		if err != nil {
			return Model{}, false, err
		}
		sourceIdentity = managedIdentity
	default:
		return Model{}, false, ErrInvalidModel
	}
	return installer.service.RegisterInstalled(ctx, candidate.Package, storageMode, sourceIdentity)
}
