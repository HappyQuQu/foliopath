package imagevips

const (
	NativeConcurrency = 1
	NativeCacheFiles  = 0
	NativeCacheMemory = 64 << 20
	// The source image is normally unique. Do not retain operation graphs and
	// their foreign decoder allocations between independent media requests.
	NativeCacheEntries = 0
)
