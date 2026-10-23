//go:build libvips && linux && cgo

package imagevips

/*
#include <malloc.h>
*/
import "C"

func trimNativeImageMemory() { C.malloc_trim(0) }
