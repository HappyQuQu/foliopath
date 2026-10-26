//go:build !libvips

package imagevips

import (
	"context"
	"errors"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/media"
	"github.com/HappyQuQu/foliopath/internal/ocr"
)

func TestOCRPreprocessingRequiresNativeRuntime(t *testing.T) {
	result, err := New().PrepareOCRImage(context.Background(), nil, media.FormatPNG)
	if !errors.Is(err, ocr.ErrPreprocessUnavailable) || result.PNG != nil || result.ContractSHA256 != "" {
		t.Fatal(result, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := New().PrepareOCRImage(ctx, nil, media.FormatPNG); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
