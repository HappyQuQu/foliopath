package main

import (
	"errors"
	"fmt"
	"os"
)

type packagePublishPhase string

const (
	packagePublishBeforeRename packagePublishPhase = "before_rename"
	packagePublishAfterRename  packagePublishPhase = "after_rename_before_parent_sync"
	packagePublishAfterSync    packagePublishPhase = "after_parent_sync"
)

var errAtomicPackagePublishUnsupported = errors.New("atomic no-replace package publish is unsupported on this platform")

// PublishPackageDirectory makes an already-verified staged directory visible
// under its final generation name. Both names are direct children of parent, so
// the rename cannot cross filesystems. Existing generations are never replaced.
func PublishPackageDirectory(parent, stagedDirectory, finalDirectory string) error {
	return publishPackageDirectoryWithHook(parent, stagedDirectory, finalDirectory, nil)
}

func publishPackageDirectoryWithHook(
	parent, stagedDirectory, finalDirectory string,
	hook func(packagePublishPhase) error,
) error {
	if !packageSegmentPattern.MatchString(stagedDirectory) || !packageSegmentPattern.MatchString(finalDirectory) {
		return errors.New("staged and final package names must be safe path segments")
	}
	parentDirectory, err := os.Open(parent)
	if err != nil {
		return fmt.Errorf("open package parent: %w", err)
	}
	defer parentDirectory.Close()
	info, err := parentDirectory.Stat()
	if err != nil || !info.IsDir() {
		return errors.New("package parent is not a directory")
	}
	if hook != nil {
		if err := hook(packagePublishBeforeRename); err != nil {
			return fmt.Errorf("before package publish: %w", err)
		}
	}
	if err := atomicRenameNoReplace(int(parentDirectory.Fd()), stagedDirectory, finalDirectory); err != nil {
		return fmt.Errorf("atomically publish package: %w", err)
	}
	if hook != nil {
		if err := hook(packagePublishAfterRename); err != nil {
			return fmt.Errorf("after package publish: %w", err)
		}
	}
	if err := parentDirectory.Sync(); err != nil {
		return fmt.Errorf("sync package parent after publish: %w", err)
	}
	if hook != nil {
		if err := hook(packagePublishAfterSync); err != nil {
			return fmt.Errorf("after package parent sync: %w", err)
		}
	}
	return nil
}
