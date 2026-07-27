package settings

import (
	"context"
	"errors"
)

var (
	ErrInvalid            = errors.New("settings are invalid")
	ErrPreconditionFailed = errors.New("settings precondition failed")
)

type Values struct {
	ScheduledScanIntervalHours *int64
	ThumbnailCacheQuotaBytes   int64
	Language                   string
	Revision                   int64
	UpdatedAtMS                int64
}

type Update struct {
	ScheduledScanIntervalHours *int64
	ThumbnailCacheQuotaBytes   *int64
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

type Service struct {
	repository Repository
	waker      WakeNotifier
	validators FieldValidators
}

type FieldValidators struct {
	Schedule   func(*int64) error
	CacheQuota func(int64) error
	Language   func(string) error
}

func NewService(
	repository Repository,
	waker WakeNotifier,
	validators FieldValidators,
) (*Service, error) {
	if repository == nil || waker == nil ||
		validators.Schedule == nil ||
		validators.CacheQuota == nil ||
		validators.Language == nil {
		return nil, errors.New("settings dependencies are required")
	}
	return &Service{repository: repository, waker: waker, validators: validators}, nil
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
		(!update.SetSchedule && update.ThumbnailCacheQuotaBytes == nil && update.Language == nil) {
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
	if update.ThumbnailCacheQuotaBytes != nil {
		next.ThumbnailCacheQuotaBytes = *update.ThumbnailCacheQuotaBytes
	}
	if update.Language != nil {
		next.Language = *update.Language
	}
	if service.validators.Schedule(next.ScheduledScanIntervalHours) != nil ||
		service.validators.CacheQuota(next.ThumbnailCacheQuotaBytes) != nil ||
		service.validators.Language(next.Language) != nil {
		return Values{}, ErrInvalid
	}
	updated, err := service.repository.UpdateSettings(ctx, expectedRevision, next)
	if err != nil {
		return Values{}, err
	}
	if update.SetSchedule {
		service.waker.Wake()
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
