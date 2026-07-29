//go:build !linux

package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "WCH-001 requires a native Linux runtime")
	os.Exit(2)
}
