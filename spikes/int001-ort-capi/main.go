//go:build linux && cgo

package main

/*
#include <stdlib.h>
#include "onnxruntime_c_api.h"

static const OrtApi* fp_api(void) {
  return OrtGetApiBase()->GetApi(ORT_API_VERSION);
}
static const char* fp_version(void) { return OrtGetApiBase()->GetVersionString(); }
static OrtStatus* fp_create_env(OrtEnv** out) {
  return fp_api()->CreateEnv(ORT_LOGGING_LEVEL_WARNING, "foliopath-int008", out);
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
static OrtStatus* fp_create_session(OrtEnv* env, const char* path, OrtSessionOptions* options, OrtSession** out) {
  return fp_api()->CreateSession(env, path, options, out);
}
static OrtStatus* fp_create_memory_info(OrtMemoryInfo** out) {
  return fp_api()->CreateCpuMemoryInfo(OrtArenaAllocator, OrtMemTypeDefault, out);
}
static OrtStatus* fp_create_tensor(OrtMemoryInfo* info, void* data, size_t bytes,
                                   int64_t* shape, size_t dimensions, OrtValue** out) {
  return fp_api()->CreateTensorWithDataAsOrtValue(
      info, data, bytes, shape, dimensions, ONNX_TENSOR_ELEMENT_DATA_TYPE_FLOAT, out);
}
static OrtStatus* fp_create_run_options(OrtRunOptions** out) {
  return fp_api()->CreateRunOptions(out);
}
static OrtStatus* fp_run(OrtSession* session, OrtRunOptions* options,
                         const char* input_name, OrtValue* input,
                         const char* output_name, OrtValue** output) {
  const char* inputs[] = {input_name};
  const OrtValue* values[] = {input};
  const char* outputs[] = {output_name};
  return fp_api()->Run(session, options, inputs, values, 1, outputs, 1, output);
}
static OrtStatus* fp_set_terminate(OrtRunOptions* options) {
  return fp_api()->RunOptionsSetTerminate(options);
}
static OrtStatus* fp_unset_terminate(OrtRunOptions* options) {
  return fp_api()->RunOptionsUnsetTerminate(options);
}
static OrtStatus* fp_tensor_data(OrtValue* value, void** out) {
  return fp_api()->GetTensorMutableData(value, out);
}
static OrtErrorCode fp_error_code(OrtStatus* status) { return fp_api()->GetErrorCode(status); }
static const char* fp_error_message(OrtStatus* status) { return fp_api()->GetErrorMessage(status); }
static void fp_release_status(OrtStatus* value) { fp_api()->ReleaseStatus(value); }
static void fp_release_env(OrtEnv* value) { fp_api()->ReleaseEnv(value); }
static void fp_release_options(OrtSessionOptions* value) { fp_api()->ReleaseSessionOptions(value); }
static void fp_release_session(OrtSession* value) { fp_api()->ReleaseSession(value); }
static void fp_release_memory_info(OrtMemoryInfo* value) { fp_api()->ReleaseMemoryInfo(value); }
static void fp_release_value(OrtValue* value) { fp_api()->ReleaseValue(value); }
static void fp_release_run_options(OrtRunOptions* value) { fp_api()->ReleaseRunOptions(value); }
*/
import "C"

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
	"unsafe"
)

type ortError struct {
	code    int
	message string
}

func (err *ortError) Error() string { return fmt.Sprintf("ORT error %d", err.code) }

func takeStatus(status *C.OrtStatus) error {
	if status == nil {
		return nil
	}
	err := &ortError{
		code:    int(C.fp_error_code(status)),
		message: C.GoString(C.fp_error_message(status)),
	}
	C.fp_release_status(status)
	return err
}

type environment struct {
	env     *C.OrtEnv
	options *C.OrtSessionOptions
	memory  *C.OrtMemoryInfo
}

func newEnvironment() (*environment, error) {
	value := &environment{}
	if err := takeStatus(C.fp_create_env(&value.env)); err != nil {
		return nil, err
	}
	if err := takeStatus(C.fp_create_options(&value.options)); err != nil {
		value.close()
		return nil, err
	}
	if err := takeStatus(C.fp_create_memory_info(&value.memory)); err != nil {
		value.close()
		return nil, err
	}
	return value, nil
}

func (value *environment) close() {
	if value.memory != nil {
		C.fp_release_memory_info(value.memory)
	}
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
	err := takeStatus(C.fp_create_session(value.env, cPath, value.options, &session))
	return session, err
}

type tensor struct {
	value *C.OrtValue
	data  unsafe.Pointer
}

func newTensor(memory *C.OrtMemoryInfo, shape []int64) (*tensor, error) {
	elements := int64(1)
	for _, dimension := range shape {
		if dimension <= 0 || elements > math.MaxInt64/dimension {
			return nil, errors.New("invalid tensor shape")
		}
		elements *= dimension
	}
	bytes := uintptr(elements * 4)
	data := C.calloc(1, C.size_t(bytes))
	if data == nil {
		return nil, errors.New("tensor allocation failed")
	}
	value := &tensor{data: data}
	status := C.fp_create_tensor(
		memory,
		data,
		C.size_t(bytes),
		(*C.int64_t)(unsafe.Pointer(&shape[0])),
		C.size_t(len(shape)),
		&value.value,
	)
	if err := takeStatus(status); err != nil {
		C.free(data)
		return nil, err
	}
	return value, nil
}

func (value *tensor) close() {
	if value.value != nil {
		C.fp_release_value(value.value)
	}
	if value.data != nil {
		C.free(value.data)
	}
}

func run(session *C.OrtSession, options *C.OrtRunOptions, input *tensor, inputName, outputName string) (*C.OrtValue, error) {
	cInput := C.CString(inputName)
	cOutput := C.CString(outputName)
	defer C.free(unsafe.Pointer(cInput))
	defer C.free(unsafe.Pointer(cOutput))
	var output *C.OrtValue
	err := takeStatus(C.fp_run(session, options, cInput, input.value, cOutput, &output))
	return output, err
}

func finiteOutput(output *C.OrtValue, elements int) (bool, error) {
	var data unsafe.Pointer
	if err := takeStatus(C.fp_tensor_data(output, &data)); err != nil {
		return false, err
	}
	values := unsafe.Slice((*float32)(data), elements)
	for _, value := range values {
		if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
			return false, nil
		}
	}
	return true, nil
}

func classifyLoad(env *environment, path string) map[string]any {
	session, err := env.load(path)
	if session != nil {
		C.fp_release_session(session)
	}
	result := map[string]any{"rejected": err != nil}
	var ortErr *ortError
	if errors.As(err, &ortErr) {
		result["error_code"] = ortErr.code
		result["message_mentions_path"] = strings.Contains(ortErr.message, path)
	}
	return result
}

func rssKiB() int64 {
	content, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return -1
	}
	for _, line := range strings.Split(string(content), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 3 && fields[0] == "VmRSS:" && fields[2] == "kB" {
			value, parseErr := strconv.ParseInt(fields[1], 10, 64)
			if parseErr == nil {
				return value
			}
		}
	}
	return -1
}

func percentile(values []float64, quantile float64) float64 {
	ordered := append([]float64(nil), values...)
	sort.Float64s(ordered)
	index := int(math.Ceil(quantile*float64(len(ordered)))) - 1
	if index < 0 {
		index = 0
	}
	return ordered[index]
}

func main() {
	var controlPath string
	var invalidPath string
	var imagePath string
	var cycles int
	var maxRSSGrowthKiB int64
	flag.StringVar(&controlPath, "control", "", "embedded control graph")
	flag.StringVar(&invalidPath, "invalid", "", "graph that must be rejected")
	flag.StringVar(&imagePath, "image", "", "fixed SigLIP image graph")
	flag.IntVar(&cycles, "cycles", 30, "bounded cancellation/recovery cycles")
	flag.Int64Var(&maxRSSGrowthKiB, "max-rss-growth-kib", 131072, "retained RSS smoke bound")
	flag.Parse()
	if controlPath == "" || invalidPath == "" || imagePath == "" || cycles < 2 || cycles > 100 {
		fmt.Fprintln(os.Stderr, "control, invalid and image paths are required")
		os.Exit(2)
	}

	env, err := newEnvironment()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer env.close()

	control, err := env.load(controlPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	input, err := newTensor(env.memory, []int64{1, 4})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	output, err := run(control, nil, input, "input", "output")
	controlFinite := false
	if err == nil {
		controlFinite, err = finiteOutput(output, 4)
	}
	if output != nil {
		C.fp_release_value(output)
	}
	input.close()
	C.fp_release_session(control)
	if err != nil || !controlFinite {
		fmt.Fprintln(os.Stderr, "control inference failed")
		os.Exit(1)
	}

	invalid := classifyLoad(env, invalidPath)
	image, err := env.load(imagePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer C.fp_release_session(image)
	imageInput, err := newTensor(env.memory, []int64{1, 3, 224, 224})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer imageInput.close()
	warmOutput, warmErr := run(image, nil, imageInput, "pixel_values", "image_embeds")
	warmFinite := false
	if warmErr == nil {
		warmFinite, warmErr = finiteOutput(warmOutput, 768)
	}
	if warmOutput != nil {
		C.fp_release_value(warmOutput)
	}
	if warmErr != nil || !warmFinite {
		fmt.Fprintln(os.Stderr, "warm inference failed")
		os.Exit(1)
	}
	rssAfterWarmup := rssKiB()

	cancelLatencies := make([]float64, 0, cycles)
	cancelCodes := map[int]int{}
	allCancelled := true
	allRecovered := true
	for range cycles {
		var runOptions *C.OrtRunOptions
		if err := takeStatus(C.fp_create_run_options(&runOptions)); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		started := time.Now()
		result := make(chan error, 1)
		go func() {
			cancelledOutput, runErr := run(image, runOptions, imageInput, "pixel_values", "image_embeds")
			if cancelledOutput != nil {
				C.fp_release_value(cancelledOutput)
			}
			result <- runErr
		}()
		time.Sleep(5 * time.Millisecond)
		terminateErr := takeStatus(C.fp_set_terminate(runOptions))
		var cancelledErr error
		select {
		case cancelledErr = <-result:
		case <-time.After(10 * time.Second):
			C.fp_release_run_options(runOptions)
			fmt.Fprintln(os.Stderr, "cancelled inference timed out")
			os.Exit(1)
		}
		cancelLatencies = append(cancelLatencies, float64(time.Since(started).Microseconds())/1000)
		unsetErr := takeStatus(C.fp_unset_terminate(runOptions))
		C.fp_release_run_options(runOptions)

		var cancelledORT *ortError
		cancelReported := errors.As(cancelledErr, &cancelledORT)
		allCancelled = allCancelled && terminateErr == nil && unsetErr == nil && cancelReported
		if cancelledORT != nil {
			cancelCodes[cancelledORT.code]++
		}

		recoveryOutput, recoveryErr := run(image, nil, imageInput, "pixel_values", "image_embeds")
		recoveryFinite := false
		if recoveryErr == nil {
			recoveryFinite, recoveryErr = finiteOutput(recoveryOutput, 768)
		}
		if recoveryOutput != nil {
			C.fp_release_value(recoveryOutput)
		}
		allRecovered = allRecovered && recoveryErr == nil && recoveryFinite
	}
	runtime.GC()
	rssAfterCycles := rssKiB()
	rssGrowth := int64(-1)
	if rssAfterWarmup >= 0 && rssAfterCycles >= 0 {
		rssGrowth = rssAfterCycles - rssAfterWarmup
	}
	memoryPassed := rssGrowth < 0 || rssGrowth <= maxRSSGrowthKiB
	passed := controlFinite && invalid["rejected"] == true && warmFinite && allCancelled && allRecovered && memoryPassed
	payload := map[string]any{
		"architecture":                          runtime.GOARCH,
		"go_version":                            runtime.Version(),
		"ort_version":                           C.GoString(C.fp_version()),
		"cpu_mem_arena":                         false,
		"control_inference_finite":              controlFinite,
		"invalid_graph":                         invalid,
		"cycles":                                cycles,
		"all_cancellations_reported_error":      allCancelled,
		"cancel_error_code_counts":              cancelCodes,
		"cancel_latency_p50_milliseconds":       percentile(cancelLatencies, 0.50),
		"cancel_latency_p95_milliseconds":       percentile(cancelLatencies, 0.95),
		"all_recovery_inferences_finite":        allRecovered,
		"rss_after_warmup_kib":                  rssAfterWarmup,
		"rss_after_cycles_kib":                  rssAfterCycles,
		"rss_growth_kib":                        rssGrowth,
		"max_rss_growth_kib":                    maxRSSGrowthKiB,
		"retained_rss_within_smoke_bound":       memoryPassed,
		"runtime_messages_exposed_by_harness":   false,
		"runtime_message_path_leakage_observed": invalid["message_mentions_path"],
		"passed":                                passed,
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(payload)
	if !passed {
		os.Exit(1)
	}
}
