package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/HappyQuQu/foliopath/internal/settings"
	"github.com/HappyQuQu/foliopath/internal/store/sqlite/dbgen"
)

func (s *Store) GetSettings(ctx context.Context) (settings.Values, error) {
	row, err := s.queries.GetSettings(ctx)
	if err != nil {
		return settings.Values{}, fmt.Errorf("get settings: %w", err)
	}
	return settingsValues(row), nil
}

func (s *Store) UpdateSettings(
	ctx context.Context,
	expectedRevision int64,
	values settings.Values,
) (settings.Values, error) {
	var interval sql.NullInt64
	if values.ScheduledScanIntervalHours != nil {
		interval = sql.NullInt64{Int64: *values.ScheduledScanIntervalHours, Valid: true}
	}
	row, err := s.queries.UpdateSettings(ctx, dbgen.UpdateSettingsParams{
		ScheduledScanIntervalHours: interval,
		AutomaticDiscoveryEnabled:  boolToInt64(values.AutomaticDiscoveryEnabled),
		ThumbnailCacheQuotaBytes:   values.ThumbnailCacheQuotaBytes,
		BackgroundConcurrency:      values.BackgroundConcurrency,
		ContentReadConcurrency:     values.ContentReadConcurrency,
		Language:                   values.Language,
		UpdatedAtMs:                s.nowMS(),
		ExpectedRevision:           expectedRevision,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return settings.Values{}, settings.ErrPreconditionFailed
	}
	if err != nil {
		return settings.Values{}, fmt.Errorf("update settings: %w", err)
	}
	return settingsValues(row), nil
}

func settingsValues(row dbgen.Setting) settings.Values {
	values := settings.Values{
		AutomaticDiscoveryEnabled: row.AutomaticDiscoveryEnabled != 0,
		ThumbnailCacheQuotaBytes:  row.ThumbnailCacheQuotaBytes,
		BackgroundConcurrency:     row.BackgroundConcurrency,
		ContentReadConcurrency:    row.ContentReadConcurrency,
		Language:                  row.Language, Revision: row.Revision, UpdatedAtMS: row.UpdatedAtMs,
	}
	if row.ScheduledScanIntervalHours.Valid {
		value := row.ScheduledScanIntervalHours.Int64
		values.ScheduledScanIntervalHours = &value
	}
	return values
}

func boolToInt64(value bool) int64 {
	if value {
		return 1
	}
	return 0
}
