# Resident split image/text sessions on Linux/arm64

Status: **capacity gate failed; dual-resident SigLIP 2 strategy rejected**.

## What ran

One fresh native Linux/arm64 container with 4 CPUs, 4 GiB and no network loaded
the pinned split SigLIP 2 image and text graphs into two simultaneous ONNX
Runtime sessions. CPU memory arenas were disabled, memory patterns remained
enabled, and each session used two inference threads. The test alternated one
image and one text inference for 100 cycles and sampled process RSS plus cgroup
`memory.current`, `memory.peak` and `memory.stat`.

## Results

- Both sessions loaded and all 200 outputs were finite.
- Immediately after both loads, cgroup current was 3,551,387,648 bytes and
  cgroup peak was 4,008,951,808 bytes.
- At cycle 100, cgroup current was 3,563,499,520 bytes; it had stabilized, but
  remained above the 3.2 GiB `R-024` threshold.
- At cycle 100, `memory.stat` attributed about 1.90 GB to anonymous memory and
  1.65 GB to file memory. Process RSS was 1,887,404 KiB and process HWM
  2,319,760 KiB, demonstrating why process RSS alone understates the container
  budget.
- Image/text inference P95 was 169.8/54.3 ms with two threads.

Machine-readable results are in
[`semantic-onnx-resident-sessions-linux-arm64-2026-08-27.json`](semantic-onnx-resident-sessions-linux-arm64-2026-08-27.json).

## Decision

Keeping both current SigLIP 2 split sessions resident is rejected. It avoids
reload churn but consumes almost the whole 4 GiB hard limit and exceeds the
existing 3.2 GiB gate before SQLite, the HTTP process, thumbnails or face
inference are added. Together with the repeated-reload failure, this means the
current SigLIP 2 lifecycle has no accepted 4 GiB layout. The next experiment
must use a materially smaller model or remove the need for both large encoders
to coexist. `INT-008`, `INT-013`, `R-024` and INT-S0 remain open.
