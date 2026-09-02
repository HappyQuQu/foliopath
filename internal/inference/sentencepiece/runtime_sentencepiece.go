//go:build linux && cgo && sentencepiece

package sentencepiece

/*
#cgo CXXFLAGS: -std=c++17
#cgo LDFLAGS: -lsentencepiece -lstdc++
#include <stdlib.h>
#include "tokenizer.h"
*/
import "C"

import (
	"context"
	"sync"
	"unsafe"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
	"github.com/HappyQuQu/foliopath/internal/semantic"
)

type nativeSession struct {
	mu     sync.Mutex
	value  *C.fp_sentencepiece
	file   aimodel.RuntimeModelFile
	closed bool
}

func (*Runtime) Open(ctx context.Context, file aimodel.RuntimeModelFile) (Session, error) {
	if err := ctx.Err(); err != nil {
		if file != nil {
			_ = file.Close()
		}
		return nil, err
	}
	if !validModelFile(file) {
		if file != nil {
			_ = file.Close()
		}
		return nil, aimodel.ErrModelIncompatible
	}
	path := C.CString(file.RuntimePath())
	defer C.free(unsafe.Pointer(path))
	value := C.fp_sentencepiece_open(path)
	if value == nil || int(C.fp_sentencepiece_piece_size(value)) != ModelPieceCount ||
		int(C.fp_sentencepiece_unk_id(value)) != UnknownTokenID || int64(C.fp_sentencepiece_eos_id(value)) != EOSTokenID {
		if value != nil {
			C.fp_sentencepiece_close(value)
		}
		_ = file.Close()
		return nil, aimodel.ErrModelIncompatible
	}
	if err := ctx.Err(); err != nil {
		C.fp_sentencepiece_close(value)
		_ = file.Close()
		return nil, err
	}
	return &nativeSession{value: value, file: file}, nil
}

func (session *nativeSession) Encode(ctx context.Context, query string) ([semantic.TextSequenceLength]int64, error) {
	var result [semantic.TextSequenceLength]int64
	if err := ctx.Err(); err != nil {
		return result, err
	}
	canonical, err := semantic.CanonicalizeQuery(query)
	if err != nil {
		return result, err
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	if session.closed || session.value == nil {
		return result, ErrTokenizerUnavailable
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	input := C.CBytes([]byte(canonical))
	defer C.free(input)
	buffer := make([]C.int32_t, semantic.MaxQueryRunes*4+1)
	count := int(C.fp_sentencepiece_encode(session.value, (*C.char)(input), C.size_t(len(canonical)),
		&buffer[0], C.size_t(len(buffer))))
	if count < 0 {
		return result, aimodel.ErrModelIncompatible
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
		result[index] = EOSTokenID
	}
	return result, nil
}

func (session *nativeSession) Close() error {
	session.mu.Lock()
	defer session.mu.Unlock()
	if session.closed {
		return nil
	}
	session.closed = true
	if session.value != nil {
		C.fp_sentencepiece_close(session.value)
		session.value = nil
	}
	if session.file != nil {
		err := session.file.Close()
		session.file = nil
		return err
	}
	return nil
}

var _ Session = (*nativeSession)(nil)
