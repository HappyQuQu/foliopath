//go:build unix

package files

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

type unixFileInfo struct {
	name string
	stat syscall.Stat_t
}

func (info *unixFileInfo) Name() string       { return info.name }
func (info *unixFileInfo) Size() int64        { return 0 }
func (info *unixFileInfo) Mode() fs.FileMode  { return fs.ModeDir | 0o755 }
func (info *unixFileInfo) ModTime() time.Time { return time.Time{} }
func (info *unixFileInfo) IsDir() bool        { return true }
func (info *unixFileInfo) Sys() any           { return &info.stat }

func TestIdentityDetectsCrossDevice(t *testing.T) {
	t.Parallel()

	rootInfo := &unixFileInfo{name: "root", stat: syscall.Stat_t{Dev: 11, Ino: 101}}
	sameDevice := &unixFileInfo{name: "same", stat: syscall.Stat_t{Dev: 11, Ino: 202}}
	otherDevice := &unixFileInfo{name: "other", stat: syscall.Stat_t{Dev: 12, Ino: 303}}
	identity := identityFromInfo(rootInfo)
	if !identity.sameFilesystem(sameDevice) {
		t.Fatal("same device was rejected")
	}
	if identity.sameFilesystem(otherDevice) {
		t.Fatal("cross-device entry was accepted")
	}
	device, inode, ok := identity.Key()
	if !ok || device != 11 || inode != 101 {
		t.Fatalf("Identity.Key = (%d, %d, %v); want (11, 101, true)", device, inode, ok)
	}
}

func TestWalkReportsFIFOAsSpecialWithoutOpeningIt(t *testing.T) {
	t.Parallel()

	rootPath, root := newTestRoot(t)
	fifo := filepath.Join(rootPath, "pipe")
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		t.Skipf("mkfifo unavailable: %v", err)
	}
	var reported error
	err := root.Walk(context.Background(), "", func(relative string, _ fs.DirEntry, walkErr error) error {
		if relative == "pipe" {
			reported = walkErr
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	if !errors.Is(reported, ErrSpecialFile) {
		t.Fatalf("FIFO error = %v; want ErrSpecialFile", reported)
	}
	if file, openErr := root.Open("pipe"); file != nil || !errors.Is(openErr, ErrNotRegular) {
		if file != nil {
			_ = file.Close()
		}
		t.Fatalf("Open(FIFO) = (%v, %v); want nil, ErrNotRegular", file, openErr)
	}
}

func TestUnreadableRootBecomesOffline(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can read mode-000 directories")
	}
	rootPath, root := newTestRoot(t)
	identity, err := root.Capture()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(rootPath, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(rootPath, 0o755) })
	if err := root.Verify(identity); !errors.Is(err, ErrOffline) || !errors.Is(err, fs.ErrPermission) {
		t.Fatalf("Verify unreadable root error = %v; want ErrOffline and fs.ErrPermission", err)
	}
}
