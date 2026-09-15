//go:build !libvips

package imagevips

import (
	"context"
	"errors"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/media"
	"github.com/HappyQuQu/foliopath/internal/semantic"
)

func TestRemoteSemanticUnavailableWithoutNativeRuntime(t *testing.T) {
	got, err := New().PrepareRemoteSemanticImage(context.Background(), nil, media.FormatPNG)
	if got != nil || !errors.Is(err, semantic.ErrImagePreprocessUnavailable) {
		t.Fatalf("stub did not fail closed: %v", err)
	}
}
