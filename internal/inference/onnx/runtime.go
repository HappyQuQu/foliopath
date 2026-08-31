// Package onnx adapts the pinned ONNX Runtime C API to FolioPath inference ports.
package onnx

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
)

const (
	RuntimeVersion     = "1.28.0"
	EmbeddingDimension = int64(768)
)

type Runtime struct{}

func New() *Runtime { return &Runtime{} }

type ImageSession interface {
	Encode(context.Context, []float32) ([]float32, error)
	Close() error
}

type modelFiles struct {
	image     aimodel.RuntimeModelFile
	text      aimodel.RuntimeModelFile
	tokenizer aimodel.RuntimeModelFile
}

func (files *modelFiles) close() {
	if files.tokenizer != nil {
		_ = files.tokenizer.Close()
	}
	if files.text != nil {
		_ = files.text.Close()
	}
	if files.image != nil {
		_ = files.image.Close()
	}
}

func openModelFiles(ctx context.Context, manifest aimodel.Manifest, open aimodel.RuntimeFileOpener) (*modelFiles, error) {
	if open == nil {
		return nil, aimodel.ErrModelIncompatible
	}
	roles := make(map[string]aimodel.ManifestFile, len(manifest.Files))
	for _, file := range manifest.Files {
		roles[file.Role] = file
	}
	files := &modelFiles{}
	for _, target := range []struct {
		role string
		set  func(aimodel.RuntimeModelFile)
	}{
		{role: "image_encoder", set: func(value aimodel.RuntimeModelFile) { files.image = value }},
		{role: "text_encoder", set: func(value aimodel.RuntimeModelFile) { files.text = value }},
		{role: "tokenizer", set: func(value aimodel.RuntimeModelFile) { files.tokenizer = value }},
	} {
		if err := ctx.Err(); err != nil {
			files.close()
			return nil, err
		}
		expected, exists := roles[target.role]
		if !exists {
			files.close()
			return nil, aimodel.ErrModelIncompatible
		}
		value, err := open(ctx, expected.Name)
		if err != nil {
			files.close()
			return nil, err
		}
		if value == nil || value.Size() != expected.Size || !validRuntimePath(value.RuntimePath()) {
			if value != nil {
				_ = value.Close()
			}
			files.close()
			return nil, aimodel.ErrModelIncompatible
		}
		target.set(value)
	}
	return files, nil
}

func openModelRole(ctx context.Context, manifest aimodel.Manifest, role string, open aimodel.RuntimeFileOpener) (aimodel.RuntimeModelFile, error) {
	if open == nil {
		return nil, aimodel.ErrModelIncompatible
	}
	var expected *aimodel.ManifestFile
	for index := range manifest.Files {
		if manifest.Files[index].Role == role {
			if expected != nil {
				return nil, aimodel.ErrModelIncompatible
			}
			expected = &manifest.Files[index]
		}
	}
	if expected == nil {
		return nil, aimodel.ErrModelIncompatible
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	value, err := open(ctx, expected.Name)
	if err != nil {
		return nil, err
	}
	if value == nil || value.Size() != expected.Size || !validRuntimePath(value.RuntimePath()) {
		if value != nil {
			_ = value.Close()
		}
		return nil, aimodel.ErrModelIncompatible
	}
	return value, nil
}

func validRuntimePath(value string) bool {
	fdText, found := strings.CutPrefix(value, "/proc/self/fd/")
	if !found || fdText == "" || strings.Contains(fdText, "/") {
		return false
	}
	fd, err := strconv.ParseUint(fdText, 10, 31)
	return err == nil && fd > 2
}

type runtimeError struct {
	operation string
	code      int
}

func (err *runtimeError) Error() string {
	return fmt.Sprintf("ONNX Runtime %s failed (code %d)", err.operation, err.code)
}

func (err *runtimeError) Unwrap() error { return aimodel.ErrModelIncompatible }

func incompatible(operation string) error {
	return &runtimeError{operation: operation, code: -1}
}

func mapRuntimeError(operation string, code int) error {
	if code < 0 {
		return incompatible(operation)
	}
	return &runtimeError{operation: operation, code: code}
}

var _ error = (*runtimeError)(nil)
