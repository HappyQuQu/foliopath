//go:build linux

package files

import "golang.org/x/sys/unix"

func anchoredRootReadOnly(root *anchoredRoot) (bool, error) {
	var status unix.Statfs_t
	if err := unix.Fstatfs(int(root.file.Fd()), &status); err != nil {
		return false, err
	}
	return status.Flags&unix.ST_RDONLY != 0, nil
}
