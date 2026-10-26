//go:build linux && cgo && sentencepiece && onnxruntime

package sentencepiececapi

/*
#cgo CFLAGS: -I/opt/onnxruntime/include
#cgo LDFLAGS: -L/opt/onnxruntime/lib -Wl,-rpath,/opt/onnxruntime/lib -lonnxruntime
#include <stdlib.h>
#include <stdatomic.h>
#include "onnxruntime_c_api.h"

static const OrtApi* fp_text_api(void) { return OrtGetApiBase()->GetApi(ORT_API_VERSION); }
static const char* fp_text_version(void) { return OrtGetApiBase()->GetVersionString(); }
static OrtStatus* fp_text_env(OrtEnv** out) { return fp_text_api()->CreateEnv(ORT_LOGGING_LEVEL_WARNING, "foliopath-text-spike", out); }
static OrtStatus* fp_text_options(OrtSessionOptions** out) {
  OrtStatus* status = fp_text_api()->CreateSessionOptions(out);
  if (status) return status;
  if ((status = fp_text_api()->SetIntraOpNumThreads(*out, 2))) return status;
  if ((status = fp_text_api()->SetInterOpNumThreads(*out, 1))) return status;
  return fp_text_api()->DisableCpuMemArena(*out);
}
static OrtStatus* fp_text_session(OrtEnv* env, const char* path, OrtSessionOptions* options, OrtSession** out) {
  return fp_text_api()->CreateSession(env, path, options, out);
}
static OrtStatus* fp_text_memory(OrtMemoryInfo** out) { return fp_text_api()->CreateCpuMemoryInfo(OrtArenaAllocator, OrtMemTypeDefault, out); }
static OrtStatus* fp_text_tensor(OrtMemoryInfo* info, int64_t* data, size_t bytes, OrtValue** out) {
  int64_t shape[] = {1, 64};
  return fp_text_api()->CreateTensorWithDataAsOrtValue(info, data, bytes, shape, 2, ONNX_TENSOR_ELEMENT_DATA_TYPE_INT64, out);
}
static OrtStatus* fp_text_run_options(OrtRunOptions** out) { return fp_text_api()->CreateRunOptions(out); }
static OrtStatus* fp_text_terminate(OrtRunOptions* value) { return fp_text_api()->RunOptionsSetTerminate(value); }
static _Atomic int fp_text_run_active = 0;
static int fp_text_running(void) { return atomic_load(&fp_text_run_active); }
static OrtStatus* fp_text_run(OrtSession* session, OrtRunOptions* options, OrtValue* input, OrtValue** output) {
  const char* input_names[] = {"input_ids"};
  const char* output_names[] = {"text_embeds"};
  const OrtValue* inputs[] = {input};
  atomic_store(&fp_text_run_active, 1);
  OrtStatus* status = fp_text_api()->Run(session, options, input_names, inputs, 1, output_names, 1, output);
  atomic_store(&fp_text_run_active, 0);
  return status;
}
static OrtStatus* fp_text_data(OrtValue* value, float** out) { return fp_text_api()->GetTensorMutableData(value, (void**)out); }
static OrtErrorCode fp_text_error(OrtStatus* status) { return fp_text_api()->GetErrorCode(status); }
static void fp_text_release_status(OrtStatus* value) { fp_text_api()->ReleaseStatus(value); }
static void fp_text_release_env(OrtEnv* value) { fp_text_api()->ReleaseEnv(value); }
static void fp_text_release_options(OrtSessionOptions* value) { fp_text_api()->ReleaseSessionOptions(value); }
static void fp_text_release_session(OrtSession* value) { fp_text_api()->ReleaseSession(value); }
static void fp_text_release_memory(OrtMemoryInfo* value) { fp_text_api()->ReleaseMemoryInfo(value); }
static void fp_text_release_value(OrtValue* value) { fp_text_api()->ReleaseValue(value); }
static void fp_text_release_run_options(OrtRunOptions* value) { fp_text_api()->ReleaseRunOptions(value); }
*/
import "C"

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"unsafe"
)

const textEmbeddingDimension = 768

var errTextRuntime = errors.New("text encoder rejected model or input")

type textSession struct {
	mu      sync.Mutex
	env     *C.OrtEnv
	options *C.OrtSessionOptions
	session *C.OrtSession
}

func openTextSession(ctx context.Context, path string) (*textSession, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if C.GoString(C.fp_text_version()) != "1.28.0" {
		return nil, errTextRuntime
	}
	value := &textSession{}
	if err := takeTextStatus(C.fp_text_env(&value.env)); err != nil {
		return nil, err
	}
	if err := takeTextStatus(C.fp_text_options(&value.options)); err != nil {
		value.close()
		return nil, err
	}
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))
	if err := takeTextStatus(C.fp_text_session(value.env, cPath, value.options, &value.session)); err != nil {
		value.close()
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		value.close()
		return nil, err
	}
	return value, nil
}

func (value *textSession) encode(ctx context.Context, ids [64]int64) ([textEmbeddingDimension]float32, error) {
	var result [textEmbeddingDimension]float32
	if err := ctx.Err(); err != nil {
		return result, err
	}
	value.mu.Lock()
	defer value.mu.Unlock()
	if value.session == nil {
		return result, errTextRuntime
	}
	var memory *C.OrtMemoryInfo
	if err := takeTextStatus(C.fp_text_memory(&memory)); err != nil {
		return result, err
	}
	defer C.fp_text_release_memory(memory)
	var input *C.OrtValue
	if err := takeTextStatus(C.fp_text_tensor(memory, (*C.int64_t)(unsafe.Pointer(&ids[0])), C.size_t(len(ids)*8), &input)); err != nil {
		return result, err
	}
	defer C.fp_text_release_value(input)
	var options *C.OrtRunOptions
	if err := takeTextStatus(C.fp_text_run_options(&options)); err != nil {
		return result, err
	}
	defer C.fp_text_release_run_options(options)
	done := make(chan struct{})
	watched := make(chan struct{})
	go func() {
		defer close(watched)
		select {
		case <-ctx.Done():
			if status := C.fp_text_terminate(options); status != nil {
				C.fp_text_release_status(status)
			}
		case <-done:
		}
	}()
	var output *C.OrtValue
	runErr := takeTextStatus(C.fp_text_run(value.session, options, input, &output))
	runtime.KeepAlive(ids)
	close(done)
	<-watched
	if output != nil {
		defer C.fp_text_release_value(output)
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	if runErr != nil || output == nil {
		return result, errTextRuntime
	}
	var data *C.float
	if err := takeTextStatus(C.fp_text_data(output, &data)); err != nil || data == nil {
		return result, errTextRuntime
	}
	copy(result[:], unsafe.Slice((*float32)(unsafe.Pointer(data)), len(result)))
	return result, nil
}

func (value *textSession) close() {
	value.mu.Lock()
	defer value.mu.Unlock()
	if value.session != nil {
		C.fp_text_release_session(value.session)
		value.session = nil
	}
	if value.options != nil {
		C.fp_text_release_options(value.options)
		value.options = nil
	}
	if value.env != nil {
		C.fp_text_release_env(value.env)
		value.env = nil
	}
}

func (*textSession) running() bool { return C.fp_text_running() != 0 }

func takeTextStatus(status *C.OrtStatus) error {
	if status == nil {
		return nil
	}
	_ = int(C.fp_text_error(status))
	C.fp_text_release_status(status)
	return errTextRuntime
}
