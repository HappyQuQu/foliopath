//go:build !unix

package files

import "io/fs"

func platformIdentity(fs.FileInfo) (device, inode uint64, ok bool) {
	return 0, 0, false
}

func platformReadOnlyFlags() int {
	return 0
}
