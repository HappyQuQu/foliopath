# ONNX valid-protobuf hostile graph smoke

Status: **bounded arm64 subproof passed; `INT-008` remains open**.

A deterministic generator built six tiny model fixtures; two independent
output directories and their manifests were byte-identical. This evidence uses
the external-data control plus three structural hostile cases; the oversized
allocation and embedded-initializer control have separate records. The control graph loads a
same-directory external tensor and produces finite output. Three hostile graphs
must be rejected: external tensor location `../sentinel.bin`, an unknown custom
operator and a cyclic dependency. This distinguishes parser-valid hostile
content from the earlier empty/random/truncated-file checks.

macOS/arm64 ONNX Runtime 1.29.0 and native Linux/arm64 ONNX Runtime 1.28.0
produced the same result. Every graph was loaded in a child process with a
15-second bound; there were no timeouts or signal terminations. The Linux run
used 4 CPUs, 4 GiB, no network and a read-only root filesystem.

The runtime rejection of parent traversal is only defense in depth. The first
release slice does not need external tensors because the selected split graphs
fit as independent files with embedded initializers. The S0 recommendation is
therefore to reject all ONNX external-data during release validation and accept
only exact size/hash allowlisted graphs at runtime. The matrix is not
exhaustive and does not cover oversized allocation graphs, native Linux/amd64
or the production C/Go adapter.

Machine-readable results are in
[`onnx-hostile-graph-arm64-2026-08-27.json`](onnx-hostile-graph-arm64-2026-08-27.json).
