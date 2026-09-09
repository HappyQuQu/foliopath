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
	return installer.InstallWithProgress(ctx, candidate, storageMode, nil)
}

func (installer *Installer) InstallWithProgress(ctx context.Context, candidate Candidate, storageMode StorageMode, progress func(int64, int64) error) (Model, bool, error) {
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
		var completed int64
		if progress != nil {
			if err := progress(0, candidate.Package.PackageSizeByte); err != nil {
				return Model{}, false, err
			}
		}
		managedIdentity, err := installer.publisher.PublishModelPackage(
			ctx,
			candidate.Package,
			candidate.Manifest,
			func(openContext context.Context, name string) (io.ReadCloser, int64, error) {
				reader, size, err := installer.source.OpenModelPackageFile(openContext, sourceIdentity, name)
				if err != nil || progress == nil {
					return reader, size, err
				}
				return &installProgressReader{ReadCloser: reader, advance: func(n int) error {
					completed += int64(n)
					return progress(min(completed, candidate.Package.PackageSizeByte), candidate.Package.PackageSizeByte)
				}}, size, nil
			},
		)
		if err != nil {
			return Model{}, false, err
		}
		sourceIdentity = managedIdentity
		if progress != nil {
			if err := progress(candidate.Package.PackageSizeByte, candidate.Package.PackageSizeByte); err != nil {
				return Model{}, false, err
			}
		}
	default:
		return Model{}, false, ErrInvalidModel
	}
	return installer.service.RegisterInstalled(ctx, candidate.Package, storageMode, sourceIdentity)
}

type installProgressReader struct {
	io.ReadCloser
	advance func(int) error
}

func (reader *installProgressReader) Read(buffer []byte) (int, error) {
	n, err := reader.ReadCloser.Read(buffer)
	if n > 0 {
		if progressErr := reader.advance(n); progressErr != nil {
			return n, progressErr
		}
	}
	return n, err
}
