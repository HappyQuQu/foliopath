# SigLIP 1 Go/C tokenizer-to-text parity — Linux/arm64

Status: **isolated Linux/arm64 end-to-end parity passed; production integration
and INT-203 remain open**.

## Fixed chain

The tagged isolation harness combined:

1. the FolioPath canonical query owner;
2. official SentencePiece 0.2.1 through the narrow Go/C++ wrapper;
3. the fixed 64-element `int64` token sequence;
4. ONNX Runtime 1.28.0 C API with CPU arena disabled, intra-op 2 and inter-op 1;
5. the retained 441,217,411-byte SigLIP 1 `text_encoder.onnx`, SHA-256
   `16eef12730b862a0c4f75926213d86749d9c6a5ec79b37b6feebc20f826fd664`;
6. the deterministic ORT 1.29.0 reference fixture, SHA-256
   `943c05755587be5092570063c8dcadf910fc6ba06dd6e917f285b38e68f40225`.

The harness is guarded by `linux && cgo && sentencepiece && onnxruntime` and
lives under `spikes/`; it is not imported by a production package.

## Result

All 31 tokenizer cases ran through the text graph. Every one of the 23,808
float32 coordinates was finite and within `atol=1e-4, rtol=1e-4` of the fixed
reference. The maximum absolute difference was
`1.811981201171875e-05`.

```text
=== RUN   TestPinnedSigLIPTokenizerToTextEncoderParity
    text_onnxruntime_test.go:115: cases=31 dimensions=768 max_abs_difference=1.811981201171875e-05
--- PASS: TestPinnedSigLIPTokenizerToTextEncoderParity (2.23s)
PASS
ok github.com/HappyQuQu/foliopath/spikes/int001-sentencepiece-capi 4.209s
```

The same run verified model and fixture hashes, pre-cancel rejection before
native inference, idempotent close and rejection after close. A follow-up
observed the C wrapper inside `OrtRun`, cancelled the context, received
`context.Canceled`, then successfully reused the same session across eight
concurrent Go callers serialized at the native handle.

A bounded lifecycle check measured 10 warm-up plus 10 observed text-session
load/close cycles. RSS grew from a 14,938,112-byte cold process to a
370,806,784-byte warmed plateau. The following 10 cycles ended at 370,835,456
bytes, a measured retained increase of only 28,672 bytes. This does not show a
continuing per-cycle leak, but the 355,897,344-byte cold-to-stable expansion is
real capacity cost and must remain in full-process budgeting. ORT emitted a
CPU-vendor warning under the containerized arm64 environment; inference still
completed and matched the reference.

## Limits

- This is isolated Linux/arm64 evidence, not native Linux/amd64 proof.
- Cancellation and serialized concurrency are bounded spike evidence, not a
  hard-timeout guarantee for every hostile graph or scheduler state.
- The lifecycle sample covers 20 cycles after one cold measurement, not a
  long-duration soak. Its roughly 356 MB cold-to-stable RSS expansion must not
  be discarded merely because the subsequent 10-cycle slope was flat.
- The spike opens an explicit test path. The proposed production adapter must
  use the reviewed model owner and `/proc/self/fd` boundary.
- Package format v2, final image ABI/SBOM/provenance, ADR-0014 acceptance and
  production composition remain open. The semantic HTTP route must stay
  disabled until those contracts are satisfied.
