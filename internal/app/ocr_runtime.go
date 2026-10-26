package app

import (
	"context"
	"errors"

	"github.com/HappyQuQu/foliopath/internal/inference/remoteml"
	"github.com/HappyQuQu/foliopath/internal/ocr"
)

type ocrRuntime struct{ client *remoteml.OCRClient }

func (r ocrRuntime) Identity() ocr.Identity { return r.client.Identity() }
func (r ocrRuntime) Recognize(ctx context.Context, png []byte) (ocr.Result, error) {
	v, err := r.client.Recognize(ctx, png)
	if ctx.Err() != nil {
		return ocr.Result{}, ctx.Err()
	}
	switch {
	case err == nil:
		return v, nil
	case errors.Is(err, remoteml.ErrBusy):
		return ocr.Result{}, ocr.ErrRuntimeBusy
	case errors.Is(err, remoteml.ErrIdentityMismatch):
		return ocr.Result{}, ocr.ErrRuntimeIdentity
	case errors.Is(err, remoteml.ErrUnavailable):
		return ocr.Result{}, ocr.ErrRuntimeUnavailable
	// The transport consumes already-prepared PNG. A producer rejecting those
	// bytes or its region/output budget is not proof the original is corrupt.
	case errors.Is(err, ocr.ErrInputLimit):
		return ocr.Result{}, ocr.ErrRuntimeOutputLimit
	default:
		return ocr.Result{}, ocr.ErrRuntimeOutput
	}
}
