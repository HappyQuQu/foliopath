//go:build !linux

package files

import "os"

func runtimeFilePath(*os.File) (string, error) { return "", ErrKernelBoundaryUnavailable }
