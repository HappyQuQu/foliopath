//go:build libvips && (!linux || !cgo)

package imagevips

// Development-only fallback; it is not Linux native-memory acceptance evidence.
func trimNativeImageMemory() {}
