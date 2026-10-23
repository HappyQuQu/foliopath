//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package files

import (
	"errors"

	"golang.org/x/sys/unix"
)

func filesystemSpace(name string) (available int64, total int64, err error) {
	var status unix.Statfs_t
	if err := unix.Statfs(name, &status); err != nil {
		return 0, 0, err
	}
	return int64(status.Bavail) * int64(status.Bsize), int64(status.Blocks) * int64(status.Bsize), nil
}

func isStorageExhausted(err error) bool {
	return errors.Is(err, unix.ENOSPC) || errors.Is(err, unix.EDQUOT)
}
