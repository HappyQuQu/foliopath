package settings

import (
	"context"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/resourcecontrol"
)

type repositoryStub struct{ values Values }

func (stub *repositoryStub) GetSettings(context.Context) (Values, error) { return stub.values, nil }
func (stub *repositoryStub) UpdateSettings(_ context.Context, revision int64, values Values) (Values, error) {
	if revision != stub.values.Revision {
		return Values{}, ErrPreconditionFailed
	}
	values.Revision++
	stub.values = values
	return values, nil
}

type wakerStub struct{ count int }

func (stub *wakerStub) Wake() { stub.count++ }

type resourceApplierStub struct {
	count  int
	limits resourcecontrol.Limits
}

func (stub *resourceApplierStub) ApplyLimits(limits resourcecontrol.Limits) error {
	stub.count++
	stub.limits = limits
	return nil
}

func TestServiceWakesTheOwnerOfEachChangedSetting(t *testing.T) {
	hours := int64(24)
	repository := &repositoryStub{values: Values{
		ScheduledScanIntervalHours: &hours,
		ThumbnailCacheQuotaBytes:   10,
		BackgroundConcurrency:      2,
		ContentReadConcurrency:     8,
		Language:                   "browser",
		Revision:                   1,
	}}
	scheduleWaker := &wakerStub{}
	discoveryWaker := &wakerStub{}
	cacheWaker := &wakerStub{}
	resourceApplier := &resourceApplierStub{}
	service, err := NewService(
		repository,
		scheduleWaker,
		discoveryWaker,
		cacheWaker,
		resourceApplier,
		FieldValidators{
			Schedule: func(value *int64) error {
				if value != nil && *value < 1 {
					return ErrInvalid
				}
				return nil
			},
			CacheQuota: func(value int64) error {
				if value < 1 {
					return ErrInvalid
				}
				return nil
			},
			Language: ValidateLanguage,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	language := "zh-CN"
	if _, err := service.Update(context.Background(), 1, Update{Language: &language}); err != nil {
		t.Fatal(err)
	}
	if scheduleWaker.count != 0 || discoveryWaker.count != 0 || cacheWaker.count != 0 {
		t.Fatal("non-schedule update woke scheduler")
	}
	quota := int64(20)
	if _, err := service.Update(context.Background(), 2, Update{
		ThumbnailCacheQuotaBytes: &quota,
	}); err != nil {
		t.Fatal(err)
	}
	if scheduleWaker.count != 0 || cacheWaker.count != 1 {
		t.Fatalf("cache update wakes = schedule %d cache %d",
			scheduleWaker.count, cacheWaker.count)
	}
	enabled := false
	if _, err := service.Update(context.Background(), 3, Update{
		AutomaticDiscoveryEnabled: &enabled,
	}); err != nil {
		t.Fatal(err)
	}
	if discoveryWaker.count != 1 {
		t.Fatalf("discovery wakes = %d, want 1", discoveryWaker.count)
	}
	background := int64(3)
	content := int64(12)
	if _, err := service.Update(context.Background(), 4, Update{
		BackgroundConcurrency:  &background,
		ContentReadConcurrency: &content,
	}); err != nil {
		t.Fatal(err)
	}
	if resourceApplier.count != 1 || resourceApplier.limits != (resourcecontrol.Limits{Background: 3, Content: 12}) {
		t.Fatalf("resource apply = count %d limits %#v", resourceApplier.count, resourceApplier.limits)
	}
	disabled, err := service.Update(context.Background(), 5, Update{SetSchedule: true})
	if err != nil {
		t.Fatal(err)
	}
	if disabled.ScheduledScanIntervalHours != nil || scheduleWaker.count != 1 {
		t.Fatalf("disabled settings = %#v, wakes %d", disabled, scheduleWaker.count)
	}
	invalid := int64(0)
	if _, err := service.Update(context.Background(), 6, Update{
		SetSchedule: true, ScheduledScanIntervalHours: &invalid,
	}); err != ErrInvalid {
		t.Fatalf("invalid interval error = %v", err)
	}
}
