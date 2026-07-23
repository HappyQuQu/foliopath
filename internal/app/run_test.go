package app

import (
	"errors"
	"testing"
)

func TestRunFailsUntilCompositionRootExists(t *testing.T) {
	err := Run(Input{})
	if !errors.Is(err, ErrCompositionRootUnavailable) {
		t.Fatalf("Run() error = %v, want ErrCompositionRootUnavailable", err)
	}
}
