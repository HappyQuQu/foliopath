//go:build !aix && !darwin && !dragonfly && !freebsd && !linux && !netbsd && !openbsd && !solaris

package files

func filesystemSpace(string) (int64, int64, error) {
	return 0, 0, ErrKernelBoundaryUnavailable
}

func isStorageExhausted(error) bool { return false }
