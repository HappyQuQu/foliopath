//go:build linux

package app

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/library"
)

func TestMediaRootServiceMapsExistingNestedMountBoundary(t *testing.T) {
	if _, err := os.Stat("/proc"); err != nil {
		t.Skipf("/proc is unavailable: %v", err)
	}
	service, component, err := newMediaRootService("/")
	if err != nil {
		t.Fatal(err)
	}
	if err := component.start(context.Background()); err != nil {
		t.Fatalf("start media-root service: %v", err)
	}
	t.Cleanup(func() {
		if err := component.stop(context.Background()); err != nil {
			t.Errorf("stop media-root service: %v", err)
		}
	})

	if err := service.ValidateLibraryRoot(
		context.Background(),
		"proc",
	); !errors.Is(err, library.ErrRootMountBoundary) {
		t.Fatalf("ValidateLibraryRoot(proc) error = %v, want ErrRootMountBoundary", err)
	}
}
