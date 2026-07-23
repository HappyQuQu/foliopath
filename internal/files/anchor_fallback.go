//go:build !linux

package files

import (
	"io/fs"
	"os"
)

// anchoredRoot is the explicit non-Linux fallback. os.Root plus the checks in
// Root preserve the existing no-symlink and same-device behavior, but these
// platforms do not claim Linux's atomic no-mount-crossing guarantee.
type anchoredRoot struct {
	root     *os.Root
	boundary *os.Root
	relative string
}

func openAnchoredRoot(name string) (*anchoredRoot, error) {
	root, err := os.OpenRoot(name)
	if err != nil {
		return nil, err
	}
	return &anchoredRoot{
		root:     root,
		boundary: root,
	}, nil
}

func (root *anchoredRoot) OpenRoot(name string) (*anchoredRoot, error) {
	relative := root.resolve(name)
	child, err := root.boundary.OpenRoot(relative)
	if err != nil {
		return nil, err
	}
	if relative == "." {
		relative = ""
	}
	return &anchoredRoot{
		root:     child,
		boundary: root.boundary,
		relative: relative,
	}, nil
}

func (root *anchoredRoot) OpenFile(name string, flag int, mode fs.FileMode) (*os.File, error) {
	return root.boundary.OpenFile(root.resolve(name), flag, mode)
}

func (root *anchoredRoot) Open(name string) (*os.File, error) {
	return root.boundary.Open(root.resolve(name))
}

func (root *anchoredRoot) Stat(name string) (fs.FileInfo, error) {
	if name == "" || name == "." {
		return root.root.Stat(".")
	}
	return root.boundary.Stat(root.resolve(name))
}

func (root *anchoredRoot) Lstat(name string) (fs.FileInfo, error) {
	return root.boundary.Lstat(root.resolve(name))
}

func (root *anchoredRoot) Close() error {
	return root.root.Close()
}

func (root *anchoredRoot) resolve(name string) string {
	if name == "" || name == "." {
		if root.relative == "" {
			return "."
		}
		return root.relative
	}
	if root.relative == "" {
		return name
	}
	return root.relative + "/" + name
}
