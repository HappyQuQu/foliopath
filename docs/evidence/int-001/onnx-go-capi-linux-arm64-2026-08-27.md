# ONNX Runtime Go/C API cancellation smoke

Status: **isolated native Linux/arm64 subproof passed; `INT-008` remains open**.

The official ONNX Runtime 1.28.0 Linux/aarch64 C/C++ package was downloaded to
a temporary directory, hash-bound and kept outside the repository. The archive
is 8,116,278 bytes with SHA-256
`e15ff8b5d85afe6c144d97c6fd432254bf76a219daaf17658087d6ecb3e8f0bb`.
Its declared commit is `da9b5e364c465de65c49d91e696cd6485270757f`.

An isolated nested Go module links the C API without changing FolioPath's
production dependency graph. Tensor memory is C-allocated, CPU memory arena is
disabled and runtime messages are reduced to stable numeric errors. In a native
Linux/arm64, 4 CPU, 4 GiB, no-network, read-only-root container it completed:

- an embedded-initializer control inference with finite output;
- invalid-operator rejection;
- 100 consecutive SigLIP image inference cancellations and 100 finite recovery
  inferences, cancellation P95 6.56 ms;
- retained RSS growth 17,404 KiB against a 131,072 KiB smoke limit;
- a race-enabled 30-cycle repetition with no Go race report, cancellation P95
  6.691 ms and RSS growth 80,364 KiB.

`go vet` passed in the same native container. Dynamic inspection shows the
harness needs `libonnxruntime.so.1`, glibc, libstdc++, libgcc, libdl, librt,
libpthread and libm; the final distroless closure and vulnerability scan remain
release work rather than an assumed property.

The raw ORT invalid-model message contained the model path. The harness detected
that fact but emitted only the numeric error code; production error mapping must
preserve that boundary.

This does not select the production adapter. Native Linux/amd64, final image
ABI/linkage, distroless loading, supply-chain scanning, context ownership,
admission and wider concurrency remain open.

Machine-readable results are in
[`onnx-go-capi-linux-arm64-2026-08-27.json`](onnx-go-capi-linux-arm64-2026-08-27.json).
