# SentencePiece C API parity smoke — emulated Linux/amd64

Status: **emulated architecture parity subproof passed; native amd64 gate
remains open**.

## Environment and fixed inputs

- Docker `--platform linux/amd64` under QEMU emulation on the development host;
  this was not a native amd64 machine.
- Debian bookworm-slim base digest
  `sha256:88200866dfff7ea7f5cbcb6ec7c8a701889efe6fe859fe64d6990e4b07ea4171`.
- Go 1.26.5 Linux/amd64 and cgo enabled.
- Official SentencePiece 0.2.1 source archive, SHA-256
  `c1a59e9259c9653ad0ade653dadff074cd31f0a6ff2a11316f67bee4189a8f1b`.
- Fixed SigLIP `spiece.model`, SHA-256
  `1e5036bed065526c3c212dfbe288752391797c4bb1a284aa18c9a0b23fcaf8ec`.

## Result

The same tagged token fixture, FD loading, concurrency/close, malformed input,
pre-cancellation, fixed truncation and 100-cycle lifecycle suite used for the
Linux/arm64 run passed under the amd64 userspace/ABI. After explicit Go memory
release, measured process RSS rose from 78,024,704 to 85,680,128 bytes, a
retained increase of 7,655,424 bytes, below the 64 MiB spike bound.

```text
resident bytes before=78024704 after=85680128 retained_increase=7655424
PASS
ok github.com/HappyQuQu/foliopath/spikes/int001-sentencepiece-capi 2.651s
```

## Interpretation and limits

- The wrapper and SentencePiece source compile, link and produce the same
  asserted behavior for an amd64 userspace/ABI under emulation.
- QEMU can hide or alter timing, memory and CPU-specific behavior. This result
  does not close the native Linux/amd64 parity, performance or soak gate.
- The runtime was assembled for a disposable test. Final-image ABI, Go archive
  provenance, SBOM, vulnerability disposition and redistribution review were
  not completed.
- ADR-0014 remains proposed, the production import graph remains unchanged and
  `INT-203` remains open.
