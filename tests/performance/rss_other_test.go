//go:build !linux

package performance_test

func processPeakRSS() (uint64, string, error) {
	return 0, "", nil
}
