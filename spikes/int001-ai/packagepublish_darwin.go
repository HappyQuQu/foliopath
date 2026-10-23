//go:build darwin

package main

import "golang.org/x/sys/unix"

func atomicRenameNoReplace(parentFD int, stagedDirectory, finalDirectory string) error {
	return unix.RenameatxNp(
		parentFD,
		stagedDirectory,
		parentFD,
		finalDirectory,
		unix.RENAME_EXCL|unix.RENAME_NOFOLLOW_ANY,
	)
}
