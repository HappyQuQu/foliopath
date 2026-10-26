//go:build !linux

package files

func anchoredRootReadOnly(_ *anchoredRoot) (bool, error) {
	return false, ErrKernelBoundaryUnavailable
}
