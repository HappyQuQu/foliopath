//go:build linux

package files

import (
	"errors"
	"io/fs"

	"golang.org/x/sys/unix"
)

func renameNoReplace(oldName, newName string) error {
	err := unix.Renameat2(unix.AT_FDCWD, oldName, unix.AT_FDCWD, newName, unix.RENAME_NOREPLACE)
	if errors.Is(err, unix.EEXIST) {
		return fs.ErrExist
	}
	return err
}
