# SentencePiece C API tokenizer smoke — Linux/arm64

Status: **bounded S1 feasibility subproof passed; ADR-0014 and INT-203 remain
open**.

## Fixed inputs and environment

- Native Linux/arm64 container; Go 1.26.5; cgo enabled.
- Official SentencePiece 0.2.1 source tree archive used for this run:
  `c1a59e9259c9653ad0ade653dadff074cd31f0a6ff2a11316f67bee4189a8f1b`.
- SigLIP `google/siglip-base-patch16-224` revision
  `7fd15f0689c79d79e38b1c2e2e2370a7bf2761ed` `spiece.model`:
  `1e5036bed065526c3c212dfbe288752391797c4bb1a284aa18c9a0b23fcaf8ec`.
- SentencePiece was built as a shared library with tests and tcmalloc disabled;
  the FolioPath harness was linked through its narrow C++ wrapper.

## Result

The tagged Go/cgo test passed. Model metadata matched the frozen candidate
contract (32,000 pieces, unknown ID 2, EOS ID 1). Fixed token-ID fixtures
matched for Chinese, English with removable ASCII punctuation, fullwidth text,
combining accents and emoji. The adapter also enforced 63-piece truncation and
the 64th EOS/pad ID without an unbounded input allocation.

Command result:

```text
ok github.com/HappyQuQu/foliopath/spikes/int001-sentencepiece-capi 0.022s
```

## What this does not prove

- ADR-0014 is still proposed; this code is outside the production import graph.
- Native Linux/amd64 parity, the full Python-reference fixture set, malformed or
  oversized model handling, opened-FD loading, concurrent use, cancellation,
  repeated load/close RSS and final-image ABI/SBOM are not covered.
- The source archive hash identifies the exact bytes used in this run, but
  release provenance, notices, signature policy and vulnerability disposition
  are not yet accepted.
- Image preprocessing, ONNX text inference, fixed text-embedding parity and the
  model-package format-v2 activation path remain missing, so `INT-203` cannot be
  checked off.
