//go:build !linux

package main

import "fmt"

type ModelScanReport struct{}

func ScanModels(root string, catalog ModelCatalog, requireReadOnly bool) (ModelScanReport, error) {
	return ModelScanReport{}, fmt.Errorf("model directory safety evidence requires native Linux openat2; current platform is unsupported")
}
