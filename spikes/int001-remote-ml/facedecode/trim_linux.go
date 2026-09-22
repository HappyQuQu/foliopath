//go:build linux && cgo

package main

/*
#include <malloc.h>
*/
import "C"

func trimDiagnosticMemory() { C.malloc_trim(0) }
