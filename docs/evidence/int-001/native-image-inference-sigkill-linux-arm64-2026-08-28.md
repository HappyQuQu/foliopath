# Native image inference process-kill recovery on Linux/arm64

Date: 2026-08-28
Scope: `INT-215`, production Go/cgo ONNX Runtime image-session boundary

## Inputs and environment

- Docker Desktop native Linux/arm64
- Project-pinned Go builder digest:
  `sha256:4ee9ffa999b4583ce281939cdff828763083610292f252279a0cee77473bd9a7`
- Project-pinned ONNX Runtime 1.28.0 native arm64 closure
- Existing SigLIP 1 FP16 candidate image encoder: 177 MiB,
  SHA-256 `4d4260477eaf57ce263d8f65656b030384fd6854513896254b9cae9566432b3d`
- Model bind-mounted read-only; no media library was mounted

## Executed verification

```text
go test -tags 'onnxruntime inferencekill' ./internal/inference/onnx \
  -run '^TestNativeImageInferenceRecoversAfterProcessKill$' -count=1 -v

--- PASS: TestNativeImageInferenceRecoversAfterProcessKill
PASS
ok github.com/HappyQuQu/foliopath/internal/inference/onnx
```

The child opens the model only through an already-open file descriptor and production `/proc/self/fd/<fd>` runtime
path, creates the production arena-disabled ORT image session, and calls `Encode` with the fixed
`1×3×224×224` input. A separate goroutine emits `RUN_ACTIVE` only when the C API call has not returned after 20 ms.
The parent waits for that signal and sends an operating-system process kill rather than Go cancellation.

A fresh child process then loads the same read-only model through a new descriptor/session and completes inference.
It verifies exactly 768 outputs and rejects every NaN or infinity. This proves a killed native runtime does not make the
model artifact or a new runtime process unusable.

## Remaining limits

The encoder is the resource-priority candidate, not an approved production catalog entry. This closes the native
image-runtime kill/reload subcase on Linux/arm64, but not native Linux/amd64 or the complete app backfill + SQLite
composition. It does not approve model quality, licensing, tokenizer ADR-0014, or production semantic search.
