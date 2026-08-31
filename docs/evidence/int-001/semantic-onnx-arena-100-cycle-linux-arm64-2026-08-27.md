# ONNX split-session 100-cycle follow-up on Linux/arm64

Status: **retained-RSS check passed; cgroup capacity gate failed; lifecycle
strategy rejected**.

## Why the 30-cycle result was insufficient

Disabling the ONNX Runtime CPU arena reduced the prior 30-cycle retained RSS
growth to zero. That measurement observed the Python process, but the 4 GiB
acceptance environment must also account for file-backed page cache charged to
the container while it repeatedly reads the 371.7 MB image graph and 1.129 GB
text graph.

The follow-up therefore ran 100 image load/infer/close → text
load/infer/close cycles in one fresh native Linux/arm64 container with 4 CPUs,
4 GiB, no network, CPU arena disabled and memory pattern enabled. The hashes
and runtime were unchanged.

## Results

- All 200 inference outputs were finite.
- Process RSS after a full cycle was 500,156 KiB at cycle 1 and 500,184 KiB at
  cycle 100: only 28 KiB retained growth, below the unchanged 128 MiB limit.
- Process `VmHWM` was 2,037,064 KiB.
- Container `memory.peak` was 3,719,651,328 bytes, above the `R-024` 3.2 GiB
  threshold of 3,435,973,836 bytes, although it stayed below the hard 4 GiB
  container limit.
- Image load/inference ranges were 0.194–0.228 s / 0.089–0.095 s; text ranges
  were 0.461–0.495 s / 0.029–0.043 s.

The gap between process peak RSS and cgroup peak is consistent with substantial
container-charged file cache from repeatedly loading both large graphs. The run
did not sample `memory.stat` per cycle, so that explanation is an inference,
not a proven byte-level attribution.

Machine-readable results are in
[`semantic-onnx-arena-100-cycle-linux-arm64-2026-08-27.json`](semantic-onnx-arena-100-cycle-linux-arm64-2026-08-27.json).

## Decision

The earlier allocator subproof remains valid: arena-off avoids retained process
RSS growth in this test. The broader repeated full-model reload strategy is
rejected because the 100-cycle cgroup peak exceeds the existing 3.2 GiB gate.
The next candidate must compare bounded long-lived sessions or a deliberately
recycled isolated inference worker and must record cgroup anonymous/file memory
over time. The gate is not weakened. `INT-008`, `INT-013`, `R-024` and INT-S0
remain open.
