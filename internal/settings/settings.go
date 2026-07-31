package settings

import (
	"context"
	"errors"

	"github.com/HappyQuQu/foliopath/internal/resourcecontrol"
)

var (
	ErrInvalid            = errors.New("settings are invalid")
	ErrPreconditionFailed = errors.New("settings precondition failed")
)

type Values struct {
	ScheduledScanIntervalHours *int64
	AutomaticDiscoveryEnabled  bool
	ThumbnailCacheQuotaBytes   int64
	ResourceProfile            resourcecontrol.Profile
	Language                   string
	Revision                   int64
	UpdatedAtMS                int64
}

type Update struct {
	ScheduledScanIntervalHours *int64
	AutomaticDiscoveryEnabled  *bool
	ThumbnailCacheQuotaBytes   *int64
	ResourceProfile            *resourcecontrol.Profile
	Language                   *string
	SetSchedule                bool
}

type Repository interface {
	GetSettings(context.Context) (Values, error)
	UpdateSettings(context.Context, int64, Values) (Values, error)
}

type WakeNotifier interface {
	Wake()
}

type ResourceProfileApplier interface {
	ApplyResourceProfile(resourcecontrol.Profile) error
}

type Service struct {
	repository      Repository
	scheduleWaker   WakeNotifier
	discoveryWaker  WakeNotifier
	cacheWaker      WakeNotifier
	resourceApplier ResourceProfileApplier
	validators      FieldValidators
}

type FieldValidators struct {
	Schedule   func(*int64) error
	CacheQuota func(int64) error
	Language   func(string) error
}

func NewService(
	repository Repository,
	scheduleWaker WakeNotifier,
	discoveryWaker WakeNotifier,
	cacheWaker WakeNotifier,
	resourceApplier ResourceProfileApplier,
	validators FieldValidators,
) (*Service, error) {
	if repository == nil || scheduleWaker == nil || discoveryWaker == nil || cacheWaker == nil ||
		resourceApplier == nil ||
		validators.Schedule == nil ||
		validators.CacheQuota == nil ||
		validators.Language == nil {
		return nil, errors.New("settings dependencies are required")
	}
	return &Service{
		repository:      repository,
		scheduleWaker:   scheduleWaker,
		discoveryWaker:  discoveryWaker,
		cacheWaker:      cacheWaker,
		resourceApplier: resourceApplier,
		validators:      validators,
	}, nil
}

func (service *Service) Get(ctx context.Context) (Values, error) {
	return service.repository.GetSettings(ctx)
}

func (service *Service) Update(
	ctx context.Context,
	expectedRevision int64,
	update Update,
) (Values, error) {
	if expectedRevision <= 0 ||
		(!update.SetSchedule && update.AutomaticDiscoveryEnabled == nil &&
			update.ThumbnailCacheQuotaBytes == nil && update.ResourceProfile == nil &&
			update.Language == nil) {
		return Values{}, ErrInvalid
	}
	current, err := service.repository.GetSettings(ctx)
	if err != nil {
		return Values{}, err
	}
	next := current
	if update.SetSchedule {
		next.ScheduledScanIntervalHours = update.ScheduledScanIntervalHours
	}
	if update.AutomaticDiscoveryEnabled != nil {
		next.AutomaticDiscoveryEnabled = *update.AutomaticDiscoveryEnabled
	}
	if update.ThumbnailCacheQuotaBytes != nil {
		next.ThumbnailCacheQuotaBytes = *update.ThumbnailCacheQuotaBytes
	}
	if update.ResourceProfile != nil {
		next.ResourceProfile = *update.ResourceProfile
	}
	if update.Language != nil {
		next.Language = *update.Language
	}
	if service.validators.Schedule(next.ScheduledScanIntervalHours) != nil ||
		service.validators.CacheQuota(next.ThumbnailCacheQuotaBytes) != nil ||
		resourcecontrol.ValidateProfile(next.ResourceProfile) != nil ||
		service.validators.Language(next.Language) != nil {
		return Values{}, ErrInvalid
	}
	updated, err := service.repository.UpdateSettings(ctx, expectedRevision, next)
	if err != nil {
		return Values{}, err
	}
	if update.SetSchedule {
		service.scheduleWaker.Wake()
	}
	if update.AutomaticDiscoveryEnabled != nil {
		service.discoveryWaker.Wake()
	}
	if update.ThumbnailCacheQuotaBytes != nil {
		service.cacheWaker.Wake()
	}
	if update.ResourceProfile != nil {
		if err := service.resourceApplier.ApplyResourceProfile(updated.ResourceProfile); err != nil {
			return Values{}, err
		}
	}
	return updated, nil
}

func ValidateLanguage(value string) error {
	switch value {
	case "browser", "zh-CN", "en":
		return nil
	default:
		return ErrInvalid
	}
}
