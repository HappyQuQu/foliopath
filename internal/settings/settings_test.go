package settings

import (
	"context"
	"testing"
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

func TestServiceValidatesAndWakesOnlyForScheduleChanges(t *testing.T) {
	hours := int64(24)
	repository := &repositoryStub{values: Values{
		ScheduledScanIntervalHours: &hours,
		ThumbnailCacheQuotaBytes:   10,
		Language:                   "browser",
		Revision:                   1,
	}}
	waker := &wakerStub{}
	service, err := NewService(repository, waker, FieldValidators{
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
	})
	if err != nil {
		t.Fatal(err)
	}
	language := "zh-CN"
	if _, err := service.Update(context.Background(), 1, Update{Language: &language}); err != nil {
		t.Fatal(err)
	}
	if waker.count != 0 {
		t.Fatal("non-schedule update woke scheduler")
	}
	disabled, err := service.Update(context.Background(), 2, Update{SetSchedule: true})
	if err != nil {
		t.Fatal(err)
	}
	if disabled.ScheduledScanIntervalHours != nil || waker.count != 1 {
		t.Fatalf("disabled settings = %#v, wakes %d", disabled, waker.count)
	}
	invalid := int64(0)
	if _, err := service.Update(context.Background(), 3, Update{
		SetSchedule: true, ScheduledScanIntervalHours: &invalid,
	}); err != ErrInvalid {
		t.Fatalf("invalid interval error = %v", err)
	}
}
