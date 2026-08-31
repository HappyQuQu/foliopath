//go:build linux

package files

import (
	"fmt"
	"os"
)

func runtimeFilePath(file *os.File) (string, error) {
	if file == nil {
		return "", ErrChanged
	}
	return fmt.Sprintf("/proc/self/fd/%d", file.Fd()), nil
}
