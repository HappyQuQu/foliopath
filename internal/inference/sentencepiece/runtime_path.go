package sentencepiece

import (
	"strconv"
	"strings"
)

func validRuntimePath(value string) bool {
	fdText, found := strings.CutPrefix(value, "/proc/self/fd/")
	if !found || fdText == "" || strings.Contains(fdText, "/") {
		return false
	}
	fd, err := strconv.ParseUint(fdText, 10, 31)
	return err == nil && fd > 2
}
