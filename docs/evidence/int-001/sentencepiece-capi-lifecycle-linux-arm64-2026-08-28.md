# SentencePiece C API lifecycle evidence — Linux/arm64

Status: **bounded S1 feasibility subproof passed; ADR-0014 and INT-203 remain
open**.

## Fixed inputs and environment

- Disposable Linux/arm64 container with cgo enabled.
- Official SentencePiece 0.2.1 source archive, SHA-256
  `c1a59e9259c9653ad0ade653dadff074cd31f0a6ff2a11316f67bee4189a8f1b`.
- SigLIP `google/siglip-base-patch16-224` revision
  `7fd15f0689c79d79e38b1c2e2e2370a7bf2761ed` `spiece.model`, 798,330
  bytes, SHA-256
  `1e5036bed065526c3c212dfbe288752391797c4bb1a284aa18c9a0b23fcaf8ec`.
- SentencePiece was compiled as a shared library with tests and tcmalloc
  disabled. The tagged spike linked it through the narrow FolioPath C++/cgo
  wrapper; production packages did not import it.

## Executed evidence

The tagged test suite passed in 2.081 seconds. It covered:

- fixed token-ID fixtures for Chinese, English, fullwidth text, combining
  accents and emoji, plus the fixed 63-piece truncation and 64-token EOS/pad
  contract;
- loading through an already-open regular-file descriptor under `/proc/self/fd`
  and encoding after the caller closed that descriptor, demonstrating that
  model loading completed synchronously;
- 32 concurrent callers through the serialized handle, idempotent close and
  rejection of use after close;
- rejection of empty, truncated, oversized and non-regular model inputs;
- 100 pre-cancelled calls rejected before native encode; and
- 10 warm-up plus 100 measured model load/close cycles. After an explicit Go
  memory release, process RSS rose from 26,030,080 to 33,632,256 bytes, a
  retained increase of 7,602,176 bytes, below the spike's 64 MiB bound.

Observed result:

```text
resident bytes before=26030080 after=33632256 retained_increase=7602176
PASS
ok github.com/HappyQuQu/foliopath/spikes/int001-sentencepiece-capi 2.081s
```

## Limits and remaining blockers

- This is one Linux/arm64 container run, not native Linux/amd64 parity or a
  final release-image ABI check.
- Cancellation is proven before native entry and observed after native return;
  the SentencePiece API does not provide mid-call interruption. A production
  timeout policy must not claim hard interruption of an in-flight encode.
- The 100-cycle RSS sample is a bounded leak signal, not a long-duration soak
  or proof under the complete FolioPath process.
- The full Python-reference fixture set, text ONNX embedding parity, package
  format-v2 activation, final SBOM/provenance and vulnerability disposition are
  still missing.
- ADR-0014 remains proposed. This evidence does not authorize a production
  SentencePiece dependency or complete `INT-203`.
