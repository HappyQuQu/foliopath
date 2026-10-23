package imagevips

const (
	NativeConcurrency = 1
	NativeCacheFiles  = 0
	NativeCacheMemory = 64 << 20
	// The source image is normally unique. Do not retain operation graphs and
	// their foreign decoder allocations between independent media requests.
	NativeCacheEntries = 0
	// Serialize/reclaim large native decodes, without imposing a RAM cap.
	NativeLargeImageConcurrency = 1
	largeImageReclaimBytes      = 16 << 20
	largeImageReclaimPixels     = 16_000_000
)

func largeImageDecodeNeedsReclaim(size int64, width, height int) bool {
	return size >= largeImageReclaimBytes || width > 0 && height > 0 && int64(width)*int64(height) >= largeImageReclaimPixels
}
