// Package app is the application's sole composition and lifecycle boundary.
package app

import "errors"

// ErrCompositionRootUnavailable is returned until the Stage 1 composition root
// is implemented. Returning an error is intentional: the process entry must not
// report a successful start before configuration, storage, and HTTP lifecycle
// ownership exist.
var ErrCompositionRootUnavailable = errors.New("application composition root is unavailable")

// Input contains the process values that the minimal command entry hands to the
// application. Configuration parsing and validation remain owned by this
// package, not cmd/foliopath.
type Input struct {
	Args    []string
	Environ []string
}

// Run is the stable process-to-application boundary. Stage 1 composition,
// lifecycle, and graceful shutdown work will replace the explicit failure.
func Run(Input) error {
	return ErrCompositionRootUnavailable
}
