package thumbnail

import "testing"

func TestCachePolicyDefaultsAndQuotaValidation(t *testing.T) {
	if DefaultCacheQuotaBytes != 10_737_418_240 ||
		CacheHighWaterPercent != 90 ||
		CacheLowWaterPercent != 80 ||
		CacheSafeFreeBytes != 536_870_912 {
		t.Fatalf(
			"cache policy = quota %d high %d low %d free %d",
			DefaultCacheQuotaBytes,
			CacheHighWaterPercent,
			CacheLowWaterPercent,
			CacheSafeFreeBytes,
		)
	}
	if ValidateCacheQuota(DefaultCacheQuotaBytes) != nil {
		t.Fatal("default cache quota was rejected")
	}
	if ValidateCacheQuota(0) == nil {
		t.Fatal("zero cache quota unexpectedly accepted")
	}
}
