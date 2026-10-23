# ONNX oversized-allocation failure smoke

Status: **bounded Linux/arm64 subproof passed; `INT-008` remains open**.

A deterministic 112-byte protobuf-valid graph uses `ConstantOfShape` to request
1.5 billion float32 values, or 6 GB. Native Linux/arm64 ONNX Runtime 1.28.0 ran
the hostile graph and a valid tiny external-data control graph in separate
child processes with a 15-second timeout.

At 1, 1.5 and 2 GiB child address-space limits, the control produced finite
output while the hostile inference returned `RuntimeException`. No valid run
timed out or terminated by signal. A 512 MiB experiment also broke the control
process and is explicitly excluded because it cannot distinguish hostile input
from an unusable runtime configuration.

This does not authorize process-wide `RLIMIT_AS` in the Go monolith. The product
boundary must prevent the graph from being admitted: only release-validated,
fixed-shape, exact-size/hash graphs may load, and the first release rejects ONNX
external-data. Runtime allocation failure is defense in depth, not the resource
policy. Native Linux/amd64 and production C/Go adapter behavior remain open.

Machine-readable results are in
[`onnx-oversized-allocation-linux-arm64-2026-08-27.json`](onnx-oversized-allocation-linux-arm64-2026-08-27.json).
