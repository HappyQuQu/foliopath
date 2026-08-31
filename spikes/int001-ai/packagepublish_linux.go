//go:build linux

package main

import "golang.org/x/sys/unix"

func atomicRenameNoReplace(parentFD int, stagedDirectory, finalDirectory string) error {
	return unix.Renameat2(parentFD, stagedDirectory, parentFD, finalDirectory, unix.RENAME_NOREPLACE)
}
