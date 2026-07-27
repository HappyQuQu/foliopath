//go:build !darwin && !dragonfly && !freebsd && !linux && !netbsd && !openbsd

package videoffmpeg

import "os/exec"

func configureCommandCancellation(*exec.Cmd) {}
