//go:build linux

package performance_test

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func processPeakRSS() (uint64, string, error) {
	status, err := os.Open("/proc/self/status")
	if err != nil {
		return 0, "", err
	}
	defer status.Close()

	scanner := bufio.NewScanner(status)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) != 3 || fields[0] != "VmHWM:" || fields[2] != "kB" {
			continue
		}
		kibibytes, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			return 0, "", fmt.Errorf("parse VmHWM: %w", err)
		}
		return kibibytes * 1024, "procfs:VmHWM", nil
	}
	if err := scanner.Err(); err != nil {
		return 0, "", fmt.Errorf("scan /proc/self/status: %w", err)
	}
	return 0, "", fmt.Errorf("VmHWM is absent from /proc/self/status")
}
