//go:build linux && cgo && sentencepiece

package sentencepiececapi

/*
#cgo CXXFLAGS: -std=c++17
#cgo LDFLAGS: -lsentencepiece -lstdc++
#include <stdlib.h>
#include "tokenizer.h"
*/
import "C"

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"unsafe"

	"github.com/HappyQuQu/foliopath/internal/semantic"
)

var errTokenizer = errors.New("tokenizer rejected input or model")

const maximumSentencePieceModelBytes = 16 << 20

type tokenizer struct {
	mu    sync.Mutex
	value *C.fp_sentencepiece
}

func open(path string) (*tokenizer, error) {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))
	value := C.fp_sentencepiece_open(cPath)
	if value == nil || int(C.fp_sentencepiece_piece_size(value)) != 32000 ||
		int(C.fp_sentencepiece_unk_id(value)) != 2 || int(C.fp_sentencepiece_eos_id(value)) != 1 {
		if value != nil {
			C.fp_sentencepiece_close(value)
		}
		return nil, errTokenizer
	}
	return &tokenizer{value: value}, nil
}

func openFile(file *os.File) (*tokenizer, error) {
	if file == nil {
		return nil, errTokenizer
	}
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() < 1 || info.Size() > maximumSentencePieceModelBytes {
		return nil, errTokenizer
	}
	return open(fmt.Sprintf("/proc/self/fd/%d", file.Fd()))
}

func (value *tokenizer) close() {
	value.mu.Lock()
	defer value.mu.Unlock()
	if value.value != nil {
		C.fp_sentencepiece_close(value.value)
		value.value = nil
	}
}

func (value *tokenizer) encode(query string) ([semantic.TextSequenceLength]int64, error) {
	return value.encodeContext(context.Background(), query)
}

func (value *tokenizer) encodeContext(ctx context.Context, query string) ([semantic.TextSequenceLength]int64, error) {
	var result [semantic.TextSequenceLength]int64
	if err := ctx.Err(); err != nil {
		return result, err
	}
	canonical, err := semantic.CanonicalizeQuery(query)
	if err != nil {
		return result, err
	}
	value.mu.Lock()
	defer value.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return result, err
	}
	if value.value == nil {
		return result, errTokenizer
	}
	input := C.CBytes([]byte(canonical))
	defer C.free(input)
	// A Unicode code point occupies at most four UTF-8 bytes. Keep one extra
	// slot for SentencePiece's leading metaspace token in the byte-fallback
	// worst case; the public query limit still owns the allocation bound.
	buffer := make([]C.int32_t, semantic.MaxQueryRunes*4+1)
	count := int(C.fp_sentencepiece_encode(
		value.value, (*C.char)(input), C.size_t(len(canonical)), &buffer[0], C.size_t(len(buffer)),
	))
	if count < 0 {
		return result, errTokenizer
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	if count > semantic.TextSequenceLength-1 {
		count = semantic.TextSequenceLength - 1
	}
	for index := 0; index < count; index++ {
		result[index] = int64(buffer[index])
	}
	for index := count; index < len(result); index++ {
		result[index] = semantic.SigLIPPadTokenID
	}
	return result, nil
}
