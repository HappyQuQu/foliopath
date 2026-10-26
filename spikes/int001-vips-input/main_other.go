//go:build !libvips

package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "int001-vips-input requires the libvips build tag")
	os.Exit(2)
}
