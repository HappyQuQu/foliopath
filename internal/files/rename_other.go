//go:build !linux

package files

import (
	"errors"
	"io/fs"
	"os"
)

func renameNoReplace(oldName, newName string) error {
	if _, err := os.Lstat(newName); err == nil {
		return fs.ErrExist
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return os.Rename(oldName, newName)
}
