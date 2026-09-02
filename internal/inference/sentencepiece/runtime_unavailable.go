//go:build !linux || !cgo || !sentencepiece

package sentencepiece

import (
	"context"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
)

func (*Runtime) Open(_ context.Context, file aimodel.RuntimeModelFile) (Session, error) {
	if file != nil {
		_ = file.Close()
	}
	return nil, ErrTokenizerUnavailable
}
