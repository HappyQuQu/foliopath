package aimodel

import (
	"context"
	"errors"
)

type AvailabilitySummary struct {
	Checked     int
	Changed     int
	Unavailable int
}

// AvailabilityService is the canonical owner for revalidating installed model
// sources. Validation failures make a model unavailable; they never remove the
// model, change the active generation, or discard derived embeddings.
type AvailabilityService struct {
	models  *Service
	catalog *Catalog
	source  ActivationPackageSource
}

func NewAvailabilityService(models *Service, catalog *Catalog, source ActivationPackageSource) (*AvailabilityService, error) {
	if models == nil || catalog == nil || source == nil {
		return nil, errors.New("AI model availability dependencies are required")
	}
	return &AvailabilityService{models: models, catalog: catalog, source: source}, nil
}

func (service *AvailabilityService) Refresh(ctx context.Context) (AvailabilitySummary, error) {
	snapshot, err := service.models.List(ctx)
	if err != nil {
		return AvailabilitySummary{}, err
	}
	summary := AvailabilitySummary{}
	for _, model := range snapshot.Items {
		if err := ctx.Err(); err != nil {
			return summary, err
		}
		summary.Checked++
		desired := StateUnavailable
		if manifest, reviewed := service.catalog.Manifest(model.Package.PackageID); reviewed {
			err := service.source.ValidateActivationSource(ctx, model, manifest)
			if err == nil {
				desired = StateAvailable
			} else if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return summary, err
			}
		}
		if desired == StateUnavailable {
			summary.Unavailable++
		}
		if model.State == desired {
			continue
		}
		if _, err := service.models.SetAvailability(ctx, model.ID, model.AvailabilityRevision, desired == StateAvailable); err != nil {
			if !errors.Is(err, ErrPreconditionFailed) {
				return summary, err
			}
			current, getErr := service.models.Get(ctx, model.ID)
			if getErr != nil {
				return summary, getErr
			}
			if current.State == desired {
				continue
			}
			if _, err = service.models.SetAvailability(ctx, current.ID, current.AvailabilityRevision, desired == StateAvailable); err != nil {
				return summary, err
			}
		}
		summary.Changed++
	}
	return summary, nil
}

func (service *AvailabilityService) MarkUnavailable(ctx context.Context, modelID string, expectedRevision int64) error {
	if modelID == "" || expectedRevision < 1 {
		return ErrInvalidModel
	}
	if _, err := service.models.SetAvailability(ctx, modelID, expectedRevision, false); err == nil {
		return nil
	} else if !errors.Is(err, ErrPreconditionFailed) {
		return err
	}
	current, err := service.models.Get(ctx, modelID)
	if err != nil {
		return err
	}
	if current.State == StateUnavailable {
		return nil
	}
	_, err = service.models.SetAvailability(ctx, current.ID, current.AvailabilityRevision, false)
	return err
}
