//go:build unix

package files

import (
	"io/fs"
	"syscall"
)

func platformIdentity(info fs.FileInfo) (device, inode uint64, ok bool) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, 0, false
	}
	return uint64(stat.Dev), uint64(stat.Ino), true
}

func platformReadOnlyFlags() int {
	return syscall.O_NONBLOCK | syscall.O_NOFOLLOW
}
