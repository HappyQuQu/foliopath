package aimodel

import (
	"context"
	"errors"
)

type DirectActivationSource interface {
	ValidateDirectModelPackage(context.Context, string, Manifest) error
	OpenDirectRuntimeModelFile(context.Context, string, string) (RuntimeModelFile, error)
}

type ManagedActivationSource interface {
	ValidateManagedModelPackage(context.Context, Model, Manifest) error
	OpenManagedRuntimeModelFile(context.Context, Model, string) (RuntimeModelFile, error)
}

type ActivationSourceRouter struct {
	direct  DirectActivationSource
	managed ManagedActivationSource
}

func NewActivationSourceRouter(direct DirectActivationSource, managed ManagedActivationSource) (*ActivationSourceRouter, error) {
	if direct == nil || managed == nil {
		return nil, errors.New("AI activation sources are required")
	}
	return &ActivationSourceRouter{direct: direct, managed: managed}, nil
}

func (router *ActivationSourceRouter) ValidateActivationSource(ctx context.Context, model Model, manifest Manifest) error {
	switch model.StorageMode {
	case StorageDirect:
		return router.direct.ValidateDirectModelPackage(ctx, model.SourceIdentity, manifest)
	case StorageManaged:
		return router.managed.ValidateManagedModelPackage(ctx, model, manifest)
	default:
		return ErrInvalidModel
	}
}

func (router *ActivationSourceRouter) OpenActivationModelFile(ctx context.Context, model Model, name string) (RuntimeModelFile, error) {
	switch model.StorageMode {
	case StorageDirect:
		return router.direct.OpenDirectRuntimeModelFile(ctx, model.SourceIdentity, name)
	case StorageManaged:
		return router.managed.OpenManagedRuntimeModelFile(ctx, model, name)
	default:
		return nil, ErrInvalidModel
	}
}

var _ ActivationPackageSource = (*ActivationSourceRouter)(nil)
