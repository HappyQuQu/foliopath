//go:build !linux && !darwin

package main

func atomicRenameNoReplace(parentFD int, stagedDirectory, finalDirectory string) error {
	return errAtomicPackagePublishUnsupported
}
