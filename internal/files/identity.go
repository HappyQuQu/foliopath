package files

import (
	"io/fs"
	"os"
)

// Identity is an opaque snapshot of a root directory. It is intended for
// capture-and-verify within a process, such as immediately before and after a
// full scan. It is not a persistent library identifier.
type Identity struct {
	device      uint64
	inode       uint64
	hasPlatform bool
	info        fs.FileInfo
}

func identityFromInfo(info fs.FileInfo) Identity {
	device, inode, ok := platformIdentity(info)
	return Identity{
		device:      device,
		inode:       inode,
		hasPlatform: ok,
		info:        info,
	}
}

func (identity Identity) valid() bool {
	return identity.info != nil
}

// Key returns the platform device and inode numbers when they are available.
// The values identify the captured filesystem object without exposing its path.
// Callers must check ok before persisting or comparing the numbers.
func (identity Identity) Key() (device, inode uint64, ok bool) {
	if !identity.valid() || !identity.hasPlatform {
		return 0, 0, false
	}
	return identity.device, identity.inode, true
}

// Equal reports whether two captures identify the same filesystem object.
func (identity Identity) Equal(other Identity) bool {
	if !identity.valid() || !other.valid() {
		return false
	}
	if identity.hasPlatform && other.hasPlatform {
		return identity.device == other.device && identity.inode == other.inode
	}
	return os.SameFile(identity.info, other.info)
}

func (identity Identity) matches(info fs.FileInfo) bool {
	return identity.Equal(identityFromInfo(info))
}

func (identity Identity) sameFilesystem(info fs.FileInfo) bool {
	device, _, ok := platformIdentity(info)
	if identity.hasPlatform && ok {
		return identity.device == device
	}
	// Non-Unix platforms have no portable device identity. os.Root still
	// provides the path boundary, while callers receive strict device checks on
	// the Linux deployment targets and other Unix development platforms.
	return true
}

func sameFileInfo(left, right fs.FileInfo) bool {
	return identityFromInfo(left).Equal(identityFromInfo(right))
}
