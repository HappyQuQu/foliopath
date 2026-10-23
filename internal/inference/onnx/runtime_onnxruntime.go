//go:build linux && cgo && onnxruntime

package onnx

/*
#include <stdlib.h>
#include "onnxruntime_c_api.h"

static const OrtApi* fp_api(void) {
  return OrtGetApiBase()->GetApi(ORT_API_VERSION);
}
static const char* fp_version(void) { return OrtGetApiBase()->GetVersionString(); }
static OrtStatus* fp_create_env(OrtEnv** out) {
  return fp_api()->CreateEnv(ORT_LOGGING_LEVEL_WARNING, "foliopath", out);
}
static OrtStatus* fp_create_options(OrtSessionOptions** out) {
  OrtStatus* status = fp_api()->CreateSessionOptions(out);
  if (status) return status;
  status = fp_api()->SetIntraOpNumThreads(*out, 2);
  if (status) return status;
  status = fp_api()->SetInterOpNumThreads(*out, 1);
  if (status) return status;
  return fp_api()->DisableCpuMemArena(*out);
}
static OrtStatus* fp_create_session(OrtEnv* env, const char* path,
                                    OrtSessionOptions* options, OrtSession** out) {
  return fp_api()->CreateSession(env, path, options, out);
}
static OrtStatus* fp_input_count(const OrtSession* session, size_t* out) {
  return fp_api()->SessionGetInputCount(session, out);
}
static OrtStatus* fp_output_count(const OrtSession* session, size_t* out) {
  return fp_api()->SessionGetOutputCount(session, out);
}
static OrtStatus* fp_default_allocator(OrtAllocator** out) {
  return fp_api()->GetAllocatorWithDefaultOptions(out);
}
static OrtStatus* fp_input_name(const OrtSession* session, size_t index,
                                OrtAllocator* allocator, char** out) {
  return fp_api()->SessionGetInputName(session, index, allocator, out);
}
static OrtStatus* fp_output_name(const OrtSession* session, size_t index,
                                 OrtAllocator* allocator, char** out) {
  return fp_api()->SessionGetOutputName(session, index, allocator, out);
}
static OrtStatus* fp_input_type(const OrtSession* session, size_t index, OrtTypeInfo** out) {
  return fp_api()->SessionGetInputTypeInfo(session, index, out);
}
static OrtStatus* fp_output_type(const OrtSession* session, size_t index, OrtTypeInfo** out) {
  return fp_api()->SessionGetOutputTypeInfo(session, index, out);
}
static OrtStatus* fp_tensor_info(const OrtTypeInfo* value,
                                 const OrtTensorTypeAndShapeInfo** out) {
  return fp_api()->CastTypeInfoToTensorInfo(value, out);
}
static OrtStatus* fp_element_type(const OrtTensorTypeAndShapeInfo* value,
                                  ONNXTensorElementDataType* out) {
  return fp_api()->GetTensorElementType(value, out);
}
static OrtStatus* fp_dimensions_count(const OrtTensorTypeAndShapeInfo* value, size_t* out) {
  return fp_api()->GetDimensionsCount(value, out);
}
static OrtStatus* fp_dimensions(const OrtTensorTypeAndShapeInfo* value,
                                int64_t* out, size_t count) {
  return fp_api()->GetDimensions(value, out, count);
}
static void fp_free_name(OrtAllocator* allocator, void* value) {
  if (value) allocator->Free(allocator, value);
}
static OrtErrorCode fp_error_code(OrtStatus* status) {
  return fp_api()->GetErrorCode(status);
}
static void fp_release_status(OrtStatus* value) { fp_api()->ReleaseStatus(value); }
static void fp_release_env(OrtEnv* value) { fp_api()->ReleaseEnv(value); }
static void fp_release_options(OrtSessionOptions* value) {
  fp_api()->ReleaseSessionOptions(value);
}
static void fp_release_session(OrtSession* value) { fp_api()->ReleaseSession(value); }
static void fp_release_type_info(OrtTypeInfo* value) { fp_api()->ReleaseTypeInfo(value); }
static OrtStatus* fp_create_memory_info(OrtMemoryInfo** out) {
  return fp_api()->CreateCpuMemoryInfo(OrtArenaAllocator, OrtMemTypeDefault, out);
}
static OrtStatus* fp_create_tensor(OrtMemoryInfo* info, float* data, size_t bytes,
                                   const int64_t* shape, size_t shape_len, OrtValue** out) {
  return fp_api()->CreateTensorWithDataAsOrtValue(info, data, bytes, shape, shape_len,
                                                  ONNX_TENSOR_ELEMENT_DATA_TYPE_FLOAT, out);
}
static OrtStatus* fp_create_int64_tensor(OrtMemoryInfo* info, int64_t* data, size_t bytes,
                                         const int64_t* shape, size_t shape_len, OrtValue** out) {
  return fp_api()->CreateTensorWithDataAsOrtValue(info, data, bytes, shape, shape_len,
                                                  ONNX_TENSOR_ELEMENT_DATA_TYPE_INT64, out);
}
static OrtStatus* fp_create_run_options(OrtRunOptions** out) {
  return fp_api()->CreateRunOptions(out);
}
static OrtStatus* fp_terminate_run(OrtRunOptions* options) {
  return fp_api()->RunOptionsSetTerminate(options);
}
static OrtStatus* fp_run_float(OrtSession* session, OrtRunOptions* options,
                               const char* input_name, const char* output_name,
                               OrtValue* input, OrtValue** output) {
  const char* input_names[] = {input_name};
  const char* output_names[] = {output_name};
  const OrtValue* inputs[] = {input};
  return fp_api()->Run(session, options, input_names, inputs, 1,
                       output_names, 1, output);
}
static OrtStatus* fp_run_yunet(OrtSession* session, OrtRunOptions* options,
                               OrtValue* input, OrtValue** outputs) {
  const char* input_names[] = {"input"};
  const char* output_names[] = {
    "cls_8", "cls_16", "cls_32",
    "obj_8", "obj_16", "obj_32",
    "bbox_8", "bbox_16", "bbox_32",
    "kps_8", "kps_16", "kps_32"
  };
  const OrtValue* inputs[] = {input};
  return fp_api()->Run(session, options, input_names, inputs, 1,
                       output_names, 12, outputs);
}
static OrtStatus* fp_tensor_data(OrtValue* value, float** out) {
  return fp_api()->GetTensorMutableData(value, (void**)out);
}
static void fp_release_memory_info(OrtMemoryInfo* value) { fp_api()->ReleaseMemoryInfo(value); }
static void fp_release_value(OrtValue* value) { fp_api()->ReleaseValue(value); }
static void fp_release_run_options(OrtRunOptions* value) { fp_api()->ReleaseRunOptions(value); }
*/
import "C"

import (
	"context"
	"math"
	"runtime"
	"strconv"
	"sync"
	"unsafe"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
	"github.com/HappyQuQu/foliopath/internal/inference/sentencepiece"
)

type environment struct {
	env     *C.OrtEnv
	options *C.OrtSessionOptions
}

type tensorContract struct {
	name        string
	elementType C.ONNXTensorElementDataType
	dimensions  []int64
}

const imageTensorElements = 1 * 3 * 224 * 224
const textTensorElements = 64
const faceEmbeddingTensorElements = 1 * 3 * 112 * 112
const faceDetectorTensorElements = 1 * 3 * FaceDetectorHeight * FaceDetectorWidth

type imageSession struct {
	mu      sync.Mutex
	env     *environment
	session *C.OrtSession
	file    aimodel.RuntimeModelFile
	closed  bool
}

type textSession struct {
	mu      sync.Mutex
	env     *environment
	session *C.OrtSession
	file    aimodel.RuntimeModelFile
	closed  bool
}

type faceEmbeddingSession struct {
	mu      sync.Mutex
	env     *environment
	session *C.OrtSession
	file    aimodel.RuntimeModelFile
	closed  bool
}

type faceDetectorSession struct {
	mu      sync.Mutex
	env     *environment
	session *C.OrtSession
	file    aimodel.RuntimeModelFile
	closed  bool
}

func (*Runtime) OpenImageSession(ctx context.Context, manifest aimodel.Manifest, open aimodel.RuntimeFileOpener) (ImageSession, error) {
	if C.GoString(C.fp_version()) != RuntimeVersion {
		return nil, aimodel.ErrInferenceRuntimeUnavailable
	}
	file, err := openModelRole(ctx, manifest, "image_encoder", open)
	if err != nil {
		return nil, err
	}
	env, err := newEnvironment()
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	session, err := env.load(file.RuntimePath())
	if err != nil {
		env.close()
		_ = file.Close()
		return nil, err
	}
	if err := validateSession(session,
		tensorContract{name: "pixel_values", elementType: C.ONNX_TENSOR_ELEMENT_DATA_TYPE_FLOAT, dimensions: []int64{1, 3, 224, 224}},
		tensorContract{name: "image_embeds", elementType: C.ONNX_TENSOR_ELEMENT_DATA_TYPE_FLOAT, dimensions: []int64{1, EmbeddingDimension}},
	); err != nil {
		C.fp_release_session(session)
		env.close()
		_ = file.Close()
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		C.fp_release_session(session)
		env.close()
		_ = file.Close()
		return nil, err
	}
	return &imageSession{env: env, session: session, file: file}, nil
}

func (*Runtime) OpenTextSession(ctx context.Context, manifest aimodel.Manifest, open aimodel.RuntimeFileOpener) (TextSession, error) {
	if C.GoString(C.fp_version()) != RuntimeVersion {
		return nil, aimodel.ErrInferenceRuntimeUnavailable
	}
	file, err := openModelRole(ctx, manifest, "text_encoder", open)
	if err != nil {
		return nil, err
	}
	env, err := newEnvironment()
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	session, err := env.load(file.RuntimePath())
	if err != nil {
		env.close()
		_ = file.Close()
		return nil, err
	}
	if err := validateSession(session,
		tensorContract{name: "input_ids", elementType: C.ONNX_TENSOR_ELEMENT_DATA_TYPE_INT64, dimensions: []int64{1, textTensorElements}},
		tensorContract{name: "text_embeds", elementType: C.ONNX_TENSOR_ELEMENT_DATA_TYPE_FLOAT, dimensions: []int64{1, EmbeddingDimension}},
	); err != nil {
		C.fp_release_session(session)
		env.close()
		_ = file.Close()
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		C.fp_release_session(session)
		env.close()
		_ = file.Close()
		return nil, err
	}
	return &textSession{env: env, session: session, file: file}, nil
}

func (*Runtime) OpenFaceEmbeddingSession(ctx context.Context, manifest aimodel.Manifest, open aimodel.RuntimeFileOpener) (FaceEmbeddingSession, error) {
	if C.GoString(C.fp_version()) != RuntimeVersion {
		return nil, aimodel.ErrInferenceRuntimeUnavailable
	}
	file, err := openModelRole(ctx, manifest, "face_embedder", open)
	if err != nil {
		return nil, err
	}
	env, err := newEnvironment()
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	session, err := env.load(file.RuntimePath())
	if err != nil {
		env.close()
		_ = file.Close()
		return nil, err
	}
	if err := validateSession(session,
		tensorContract{name: "data", elementType: C.ONNX_TENSOR_ELEMENT_DATA_TYPE_FLOAT, dimensions: []int64{-1, 3, 112, 112}},
		tensorContract{name: "1333", elementType: C.ONNX_TENSOR_ELEMENT_DATA_TYPE_FLOAT, dimensions: []int64{1, FaceEmbeddingDimension}},
	); err != nil {
		C.fp_release_session(session)
		env.close()
		_ = file.Close()
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		C.fp_release_session(session)
		env.close()
		_ = file.Close()
		return nil, err
	}
	return &faceEmbeddingSession{env: env, session: session, file: file}, nil
}

func (*Runtime) OpenFaceDetectorSession(ctx context.Context, manifest aimodel.Manifest, open aimodel.RuntimeFileOpener) (FaceDetectorSession, error) {
	if C.GoString(C.fp_version()) != RuntimeVersion {
		return nil, aimodel.ErrInferenceRuntimeUnavailable
	}
	file, err := openModelRole(ctx, manifest, "face_detector", open)
	if err != nil {
		return nil, err
	}
	env, err := newEnvironment()
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	session, err := env.load(file.RuntimePath())
	if err != nil {
		env.close()
		_ = file.Close()
		return nil, err
	}
	if err := validateFaceDetectorSession(session); err != nil {
		C.fp_release_session(session)
		env.close()
		_ = file.Close()
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		C.fp_release_session(session)
		env.close()
		_ = file.Close()
		return nil, err
	}
	return &faceDetectorSession{env: env, session: session, file: file}, nil
}

func (session *imageSession) Encode(ctx context.Context, input []float32) ([]float32, error) {
	if len(input) != imageTensorElements {
		return nil, aimodel.ErrModelIncompatible
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	if session.closed || session.session == nil {
		return nil, aimodel.ErrInferenceRuntimeUnavailable
	}
	var memoryInfo *C.OrtMemoryInfo
	if err := takeStatus("create tensor memory info", C.fp_create_memory_info(&memoryInfo)); err != nil {
		return nil, err
	}
	defer C.fp_release_memory_info(memoryInfo)
	shape := [...]C.int64_t{1, 3, 224, 224}
	var inputValue *C.OrtValue
	if err := takeStatus("create image input tensor", C.fp_create_tensor(memoryInfo,
		(*C.float)(unsafe.Pointer(&input[0])), C.size_t(len(input)*4), &shape[0], C.size_t(len(shape)), &inputValue)); err != nil {
		return nil, err
	}
	defer C.fp_release_value(inputValue)
	var options *C.OrtRunOptions
	if err := takeStatus("create image run options", C.fp_create_run_options(&options)); err != nil {
		return nil, err
	}
	defer C.fp_release_run_options(options)

	watchDone := make(chan struct{})
	watched := make(chan struct{})
	go func() {
		defer close(watched)
		select {
		case <-ctx.Done():
			status := C.fp_terminate_run(options)
			if status != nil {
				C.fp_release_status(status)
			}
		case <-watchDone:
		}
	}()
	var outputValue *C.OrtValue
	inputName := C.CString("pixel_values")
	outputName := C.CString("image_embeds")
	defer C.free(unsafe.Pointer(inputName))
	defer C.free(unsafe.Pointer(outputName))
	runErr := takeStatus("run image encoder", C.fp_run_float(session.session, options, inputName, outputName, inputValue, &outputValue))
	runtime.KeepAlive(input)
	close(watchDone)
	<-watched
	if outputValue != nil {
		defer C.fp_release_value(outputValue)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if runErr != nil {
		return nil, runErr
	}
	if outputValue == nil {
		return nil, incompatible("read image encoder output")
	}
	var outputData *C.float
	if err := takeStatus("read image encoder output", C.fp_tensor_data(outputValue, &outputData)); err != nil {
		return nil, err
	}
	if outputData == nil {
		return nil, incompatible("read image encoder output data")
	}
	output := make([]float32, int(EmbeddingDimension))
	copy(output, unsafe.Slice((*float32)(unsafe.Pointer(outputData)), int(EmbeddingDimension)))
	return output, nil
}

func (session *textSession) EncodeText(ctx context.Context, input []int64) ([]float32, error) {
	if len(input) != textTensorElements {
		return nil, aimodel.ErrModelIncompatible
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	if session.closed || session.session == nil {
		return nil, aimodel.ErrInferenceRuntimeUnavailable
	}
	var memoryInfo *C.OrtMemoryInfo
	if err := takeStatus("create text tensor memory info", C.fp_create_memory_info(&memoryInfo)); err != nil {
		return nil, err
	}
	defer C.fp_release_memory_info(memoryInfo)
	shape := [...]C.int64_t{1, textTensorElements}
	var inputValue *C.OrtValue
	if err := takeStatus("create text input tensor", C.fp_create_int64_tensor(memoryInfo,
		(*C.int64_t)(unsafe.Pointer(&input[0])), C.size_t(len(input)*8), &shape[0], C.size_t(len(shape)), &inputValue)); err != nil {
		return nil, err
	}
	defer C.fp_release_value(inputValue)
	var options *C.OrtRunOptions
	if err := takeStatus("create text run options", C.fp_create_run_options(&options)); err != nil {
		return nil, err
	}
	defer C.fp_release_run_options(options)

	watchDone := make(chan struct{})
	watched := make(chan struct{})
	go func() {
		defer close(watched)
		select {
		case <-ctx.Done():
			status := C.fp_terminate_run(options)
			if status != nil {
				C.fp_release_status(status)
			}
		case <-watchDone:
		}
	}()
	var outputValue *C.OrtValue
	inputName := C.CString("input_ids")
	outputName := C.CString("text_embeds")
	defer C.free(unsafe.Pointer(inputName))
	defer C.free(unsafe.Pointer(outputName))
	runErr := takeStatus("run text encoder", C.fp_run_float(session.session, options, inputName, outputName, inputValue, &outputValue))
	runtime.KeepAlive(input)
	close(watchDone)
	<-watched
	if outputValue != nil {
		defer C.fp_release_value(outputValue)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if runErr != nil || outputValue == nil {
		return nil, incompatible("run text encoder")
	}
	var output *C.float
	if err := takeStatus("read text encoder output", C.fp_tensor_data(outputValue, &output)); err != nil || output == nil {
		return nil, incompatible("read text encoder output")
	}
	result := append([]float32(nil), unsafe.Slice((*float32)(unsafe.Pointer(output)), int(EmbeddingDimension))...)
	for _, value := range result {
		if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
			return nil, aimodel.ErrModelIncompatible
		}
	}
	return result, nil
}

func (session *faceEmbeddingSession) EmbedFace(ctx context.Context, input []float32) ([]float32, error) {
	if len(input) != faceEmbeddingTensorElements {
		return nil, aimodel.ErrModelIncompatible
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	if session.closed || session.session == nil {
		return nil, aimodel.ErrInferenceRuntimeUnavailable
	}
	var memoryInfo *C.OrtMemoryInfo
	if err := takeStatus("create face tensor memory info", C.fp_create_memory_info(&memoryInfo)); err != nil {
		return nil, err
	}
	defer C.fp_release_memory_info(memoryInfo)
	shape := [...]C.int64_t{1, 3, 112, 112}
	var inputValue *C.OrtValue
	if err := takeStatus("create face input tensor", C.fp_create_tensor(memoryInfo,
		(*C.float)(unsafe.Pointer(&input[0])), C.size_t(len(input)*4), &shape[0], C.size_t(len(shape)), &inputValue)); err != nil {
		return nil, err
	}
	defer C.fp_release_value(inputValue)
	var options *C.OrtRunOptions
	if err := takeStatus("create face run options", C.fp_create_run_options(&options)); err != nil {
		return nil, err
	}
	defer C.fp_release_run_options(options)

	watchDone := make(chan struct{})
	watched := make(chan struct{})
	go func() {
		defer close(watched)
		select {
		case <-ctx.Done():
			status := C.fp_terminate_run(options)
			if status != nil {
				C.fp_release_status(status)
			}
		case <-watchDone:
		}
	}()
	inputName := C.CString("data")
	outputName := C.CString("1333")
	defer C.free(unsafe.Pointer(inputName))
	defer C.free(unsafe.Pointer(outputName))
	var outputValue *C.OrtValue
	runErr := takeStatus("run face embedder", C.fp_run_float(session.session, options, inputName, outputName, inputValue, &outputValue))
	runtime.KeepAlive(input)
	close(watchDone)
	<-watched
	if outputValue != nil {
		defer C.fp_release_value(outputValue)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if runErr != nil {
		return nil, runErr
	}
	if outputValue == nil {
		return nil, incompatible("read face embedder output")
	}
	var outputData *C.float
	if err := takeStatus("read face embedder output", C.fp_tensor_data(outputValue, &outputData)); err != nil {
		return nil, err
	}
	if outputData == nil {
		return nil, incompatible("read face embedder output data")
	}
	output := make([]float32, int(FaceEmbeddingDimension))
	copy(output, unsafe.Slice((*float32)(unsafe.Pointer(outputData)), int(FaceEmbeddingDimension)))
	return output, nil
}

func (session *faceDetectorSession) DetectFaces(ctx context.Context, input []float32) ([]FaceDetection, error) {
	if len(input) != faceDetectorTensorElements {
		return nil, aimodel.ErrModelIncompatible
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	if session.closed || session.session == nil {
		return nil, aimodel.ErrInferenceRuntimeUnavailable
	}
	var memoryInfo *C.OrtMemoryInfo
	if err := takeStatus("create face detector tensor memory info", C.fp_create_memory_info(&memoryInfo)); err != nil {
		return nil, err
	}
	defer C.fp_release_memory_info(memoryInfo)
	shape := [...]C.int64_t{1, 3, FaceDetectorHeight, FaceDetectorWidth}
	var inputValue *C.OrtValue
	if err := takeStatus("create face detector input tensor", C.fp_create_tensor(memoryInfo,
		(*C.float)(unsafe.Pointer(&input[0])), C.size_t(len(input)*4), &shape[0], C.size_t(len(shape)), &inputValue)); err != nil {
		return nil, err
	}
	defer C.fp_release_value(inputValue)
	var options *C.OrtRunOptions
	if err := takeStatus("create face detector run options", C.fp_create_run_options(&options)); err != nil {
		return nil, err
	}
	defer C.fp_release_run_options(options)

	watchDone := make(chan struct{})
	watched := make(chan struct{})
	go func() {
		defer close(watched)
		select {
		case <-ctx.Done():
			status := C.fp_terminate_run(options)
			if status != nil {
				C.fp_release_status(status)
			}
		case <-watchDone:
		}
	}()
	var outputValues [12]*C.OrtValue
	runErr := takeStatus("run face detector", C.fp_run_yunet(session.session, options, inputValue, &outputValues[0]))
	runtime.KeepAlive(input)
	close(watchDone)
	<-watched
	for _, output := range outputValues {
		if output != nil {
			defer C.fp_release_value(output)
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if runErr != nil {
		return nil, runErr
	}
	strides := [...]int{8, 16, 32}
	heads := make([]faceDetectorHead, 0, len(strides))
	for strideIndex, stride := range strides {
		cells := FaceDetectorWidth / stride * (FaceDetectorHeight / stride)
		cls, err := copyFloatTensor(outputValues[strideIndex], cells)
		if err != nil {
			return nil, err
		}
		obj, err := copyFloatTensor(outputValues[3+strideIndex], cells)
		if err != nil {
			return nil, err
		}
		bbox, err := copyFloatTensor(outputValues[6+strideIndex], cells*4)
		if err != nil {
			return nil, err
		}
		kps, err := copyFloatTensor(outputValues[9+strideIndex], cells*10)
		if err != nil {
			return nil, err
		}
		heads = append(heads, faceDetectorHead{stride: stride, cls: cls, obj: obj, bbox: bbox, kps: kps})
	}
	return decodeFaceDetectorHeads(heads)
}

func copyFloatTensor(value *C.OrtValue, elements int) ([]float32, error) {
	if value == nil || elements < 1 {
		return nil, incompatible("read face detector output")
	}
	var data *C.float
	if err := takeStatus("read face detector output", C.fp_tensor_data(value, &data)); err != nil {
		return nil, err
	}
	if data == nil {
		return nil, incompatible("read face detector output data")
	}
	result := make([]float32, elements)
	copy(result, unsafe.Slice((*float32)(unsafe.Pointer(data)), elements))
	return result, nil
}

func (session *imageSession) Close() error {
	session.mu.Lock()
	defer session.mu.Unlock()
	if session.closed {
		return nil
	}
	session.closed = true
	if session.session != nil {
		C.fp_release_session(session.session)
		session.session = nil
	}
	if session.env != nil {
		session.env.close()
		session.env = nil
	}
	if session.file != nil {
		err := session.file.Close()
		session.file = nil
		return err
	}
	return nil
}

func (session *textSession) Close() error {
	session.mu.Lock()
	defer session.mu.Unlock()
	if session.closed {
		return nil
	}
	session.closed = true
	if session.session != nil {
		C.fp_release_session(session.session)
		session.session = nil
	}
	if session.env != nil {
		session.env.close()
		session.env = nil
	}
	if session.file != nil {
		err := session.file.Close()
		session.file = nil
		return err
	}
	return nil
}

func (session *faceEmbeddingSession) Close() error {
	session.mu.Lock()
	defer session.mu.Unlock()
	if session.closed {
		return nil
	}
	session.closed = true
	if session.session != nil {
		C.fp_release_session(session.session)
		session.session = nil
	}
	if session.env != nil {
		session.env.close()
		session.env = nil
	}
	if session.file != nil {
		err := session.file.Close()
		session.file = nil
		return err
	}
	return nil
}

func (session *faceDetectorSession) Close() error {
	session.mu.Lock()
	defer session.mu.Unlock()
	if session.closed {
		return nil
	}
	session.closed = true
	if session.session != nil {
		C.fp_release_session(session.session)
		session.session = nil
	}
	if session.env != nil {
		session.env.close()
		session.env = nil
	}
	if session.file != nil {
		err := session.file.Close()
		session.file = nil
		return err
	}
	return nil
}

var _ ImageSession = (*imageSession)(nil)
var _ TextSession = (*textSession)(nil)
var _ FaceEmbeddingSession = (*faceEmbeddingSession)(nil)
var _ FaceDetectorSession = (*faceDetectorSession)(nil)

func (*Runtime) LoadAndValidate(
	ctx context.Context,
	_ aimodel.Model,
	manifest aimodel.Manifest,
	open aimodel.RuntimeFileOpener,
) (aimodel.RuntimeMetadata, error) {
	if C.GoString(C.fp_version()) != RuntimeVersion {
		return aimodel.RuntimeMetadata{}, aimodel.ErrInferenceRuntimeUnavailable
	}
	files, err := openModelFiles(ctx, manifest, open)
	if err != nil {
		return aimodel.RuntimeMetadata{}, err
	}
	defer files.close()

	env, err := newEnvironment()
	if err != nil {
		return aimodel.RuntimeMetadata{}, err
	}
	defer env.close()

	image, err := env.load(files.image.RuntimePath())
	if err != nil {
		return aimodel.RuntimeMetadata{}, err
	}
	defer C.fp_release_session(image)
	if err := ctx.Err(); err != nil {
		return aimodel.RuntimeMetadata{}, err
	}
	text, err := env.load(files.text.RuntimePath())
	if err != nil {
		return aimodel.RuntimeMetadata{}, err
	}
	defer C.fp_release_session(text)

	if err := validateSession(image,
		tensorContract{name: "pixel_values", elementType: C.ONNX_TENSOR_ELEMENT_DATA_TYPE_FLOAT, dimensions: []int64{1, 3, 224, 224}},
		tensorContract{name: "image_embeds", elementType: C.ONNX_TENSOR_ELEMENT_DATA_TYPE_FLOAT, dimensions: []int64{1, EmbeddingDimension}},
	); err != nil {
		return aimodel.RuntimeMetadata{}, err
	}
	if err := validateSession(text,
		tensorContract{name: "input_ids", elementType: C.ONNX_TENSOR_ELEMENT_DATA_TYPE_INT64, dimensions: []int64{1, 64}},
		tensorContract{name: "text_embeds", elementType: C.ONNX_TENSOR_ELEMENT_DATA_TYPE_FLOAT, dimensions: []int64{1, EmbeddingDimension}},
	); err != nil {
		return aimodel.RuntimeMetadata{}, err
	}
	if manifest.FormatVersion == aimodel.SemanticFormatVersion {
		tokenizerFile := files.tokenizer
		files.tokenizer = nil
		tokenizer, err := sentencepiece.New().Open(ctx, tokenizerFile)
		if err != nil {
			return aimodel.RuntimeMetadata{}, err
		}
		defer tokenizer.Close()
		ids, err := tokenizer.Encode(ctx, "red armor portrait")
		if err != nil || ids[0] != 1226 || ids[1] != 14431 || ids[2] != 9391 {
			return aimodel.RuntimeMetadata{}, aimodel.ErrModelIncompatible
		}
		for _, id := range ids[3:] {
			if id != sentencepiece.EOSTokenID {
				return aimodel.RuntimeMetadata{}, aimodel.ErrModelIncompatible
			}
		}
		output, err := (&textSession{session: text}).EncodeText(ctx, ids[:])
		if err != nil || len(output) != int(EmbeddingDimension) {
			return aimodel.RuntimeMetadata{}, aimodel.ErrModelIncompatible
		}
		var norm float64
		for _, value := range output {
			norm += float64(value) * float64(value)
		}
		if norm == 0 || math.IsNaN(norm) || math.IsInf(norm, 0) {
			return aimodel.RuntimeMetadata{}, aimodel.ErrModelIncompatible
		}
	}
	if err := ctx.Err(); err != nil {
		return aimodel.RuntimeMetadata{}, err
	}
	return aimodel.RuntimeMetadata{EmbeddingDimension: EmbeddingDimension}, nil
}

func newEnvironment() (*environment, error) {
	value := &environment{}
	if err := takeStatus("create environment", C.fp_create_env(&value.env)); err != nil {
		return nil, err
	}
	if err := takeStatus("create session options", C.fp_create_options(&value.options)); err != nil {
		value.close()
		return nil, err
	}
	return value, nil
}

func (value *environment) close() {
	if value.options != nil {
		C.fp_release_options(value.options)
	}
	if value.env != nil {
		C.fp_release_env(value.env)
	}
}

func (value *environment) load(path string) (*C.OrtSession, error) {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))
	var session *C.OrtSession
	if err := takeStatus("load model", C.fp_create_session(value.env, cPath, value.options, &session)); err != nil {
		return nil, err
	}
	return session, nil
}

func validateSession(session *C.OrtSession, input, output tensorContract) error {
	var inputCount C.size_t
	if err := takeStatus("read input count", C.fp_input_count(session, &inputCount)); err != nil {
		return err
	}
	var outputCount C.size_t
	if err := takeStatus("read output count", C.fp_output_count(session, &outputCount)); err != nil {
		return err
	}
	if inputCount != 1 || outputCount != 1 {
		return incompatible("validate graph arity")
	}
	if err := validateTensor(session, true, input); err != nil {
		return err
	}
	return validateTensor(session, false, output)
}

func validateFaceDetectorSession(session *C.OrtSession) error {
	var inputCount C.size_t
	if err := takeStatus("read face detector input count", C.fp_input_count(session, &inputCount)); err != nil {
		return err
	}
	var outputCount C.size_t
	if err := takeStatus("read face detector output count", C.fp_output_count(session, &outputCount)); err != nil {
		return err
	}
	if inputCount != 1 || outputCount != 12 {
		return incompatible("validate face detector graph arity")
	}
	if err := validateTensorAt(session, true, 0, tensorContract{name: "input", elementType: C.ONNX_TENSOR_ELEMENT_DATA_TYPE_FLOAT,
		dimensions: []int64{1, 3, FaceDetectorHeight, FaceDetectorWidth}}); err != nil {
		return err
	}
	index := 0
	for _, group := range []struct {
		prefix string
		width  int64
	}{{"cls", 1}, {"obj", 1}, {"bbox", 4}, {"kps", 10}} {
		for _, stride := range []int64{8, 16, 32} {
			cells := int64(FaceDetectorWidth) * int64(FaceDetectorHeight) / (stride * stride)
			if err := validateTensorAt(session, false, index, tensorContract{name: group.prefix + "_" + strconv.FormatInt(stride, 10),
				elementType: C.ONNX_TENSOR_ELEMENT_DATA_TYPE_FLOAT, dimensions: []int64{1, cells, group.width}}); err != nil {
				return err
			}
			index++
		}
	}
	return nil
}

func validateTensor(session *C.OrtSession, input bool, expected tensorContract) error {
	return validateTensorAt(session, input, 0, expected)
}

func validateTensorAt(session *C.OrtSession, input bool, index int, expected tensorContract) error {
	var allocator *C.OrtAllocator
	if err := takeStatus("get allocator", C.fp_default_allocator(&allocator)); err != nil {
		return err
	}
	var name *C.char
	var status *C.OrtStatus
	if input {
		status = C.fp_input_name(session, C.size_t(index), allocator, &name)
	} else {
		status = C.fp_output_name(session, C.size_t(index), allocator, &name)
	}
	if err := takeStatus("read tensor name", status); err != nil {
		return err
	}
	defer C.fp_free_name(allocator, unsafe.Pointer(name))
	if C.GoString(name) != expected.name {
		return incompatible("validate tensor name")
	}

	var typeInfo *C.OrtTypeInfo
	if input {
		status = C.fp_input_type(session, C.size_t(index), &typeInfo)
	} else {
		status = C.fp_output_type(session, C.size_t(index), &typeInfo)
	}
	if err := takeStatus("read tensor type", status); err != nil {
		return err
	}
	defer C.fp_release_type_info(typeInfo)
	var tensorInfo *C.OrtTensorTypeAndShapeInfo
	if err := takeStatus("read tensor shape", C.fp_tensor_info(typeInfo, &tensorInfo)); err != nil {
		return err
	}
	if tensorInfo == nil {
		return incompatible("validate tensor type")
	}
	var elementType C.ONNXTensorElementDataType
	if err := takeStatus("read element type", C.fp_element_type(tensorInfo, &elementType)); err != nil {
		return err
	}
	if elementType != expected.elementType {
		return incompatible("validate element type")
	}
	var dimensionCount C.size_t
	if err := takeStatus("read dimension count", C.fp_dimensions_count(tensorInfo, &dimensionCount)); err != nil {
		return err
	}
	if int(dimensionCount) != len(expected.dimensions) {
		return incompatible("validate tensor rank")
	}
	dimensions := make([]C.int64_t, int(dimensionCount))
	if err := takeStatus("read dimensions", C.fp_dimensions(tensorInfo, &dimensions[0], dimensionCount)); err != nil {
		return err
	}
	for index, actual := range dimensions {
		if int64(actual) != expected.dimensions[index] {
			return incompatible("validate tensor shape")
		}
	}
	return nil
}

func takeStatus(operation string, status *C.OrtStatus) error {
	if status == nil {
		return nil
	}
	code := int(C.fp_error_code(status))
	C.fp_release_status(status)
	return mapRuntimeError(operation, code)
}

var _ aimodel.InferenceRuntime = (*Runtime)(nil)
