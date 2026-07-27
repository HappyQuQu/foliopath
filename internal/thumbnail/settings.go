// Package thumbnail owns thumbnail derivation and cache policy.
package thumbnail

import "errors"

const (
	DefaultCacheQuotaBytes = int64(10 << 30)
	CacheHighWaterPercent  = int64(90)
	CacheLowWaterPercent   = int64(80)
	CacheSafeFreeBytes     = int64(512 << 20)
)

var (
	ErrInvalidCacheQuota = errors.New("invalid thumbnail cache quota")
	ErrCacheCapacity     = errors.New("thumbnail cache capacity unavailable")
)

func ValidateCacheQuota(value int64) error {
	if value < 1 {
		return ErrInvalidCacheQuota
	}
	return nil
}
