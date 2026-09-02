// Package sentencepiece provides the narrow, FD-anchored tokenizer runtime
// selected by ADR-0014. Native support is explicit and fails closed otherwise.
package sentencepiece

import (
	"context"
	"errors"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
	"github.com/HappyQuQu/foliopath/internal/semantic"
)

const (
	ModelPieceCount = 32000
	UnknownTokenID  = 2
	EOSTokenID      = semantic.SigLIPPadTokenID
	MaxModelBytes   = 16 << 20
)

var ErrTokenizerUnavailable = errors.New("semantic tokenizer is unavailable")

type Session interface {
	Encode(context.Context, string) ([semantic.TextSequenceLength]int64, error)
	Close() error
}

type Runtime struct{}

func New() *Runtime { return &Runtime{} }

func validModelFile(file aimodel.RuntimeModelFile) bool {
	return file != nil && file.Size() > 0 && file.Size() <= MaxModelBytes && validRuntimePath(file.RuntimePath())
}
