// Package thumbnail owns thumbnail derivation and cache policy.
package thumbnail

import "errors"

var ErrInvalidCacheQuota = errors.New("invalid thumbnail cache quota")

func ValidateCacheQuota(value int64) error {
	if value < 1 {
		return ErrInvalidCacheQuota
	}
	return nil
}
