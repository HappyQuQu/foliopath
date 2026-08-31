# SigLIP 1 float16-internal candidate on production capacity

Status: **arm64 memory and ordinary browse proxy passed; repeatability and all-query Gate remain open**.

## Conversion and quality

The float32 SigLIP 1 split graphs were converted twice using the pinned ONNX
Runtime float16 transformer while preserving float32 inputs/outputs. The image
converter initially produced an invalid topological order after inserting I/O
casts; the checker rejected it. Applying the runtime's deterministic
topological sort before validation produced two byte-identical, checker-valid
graphs:

| Graph | Float32 | Float16-internal |
| --- | ---: | ---: |
| Image | 371,682,125 B | 185,966,808 B |
| Text | 441,217,411 B | 220,719,548 B |

The ten-image/24-query public pilot retained all 24 Top-3 lists across
float32/float16-model and macOS/Linux arm64. Linux image/text P95 rose from
about 90.0/29.4 ms to 113.0/45.3 ms, confirming a CPU latency cost. A
dual-session 100-cycle run stabilized by cycle 30 and peaked at 1,613,815,808
bytes, versus 2,181,382,144 bytes for the float32 model.

## Production 100k capacity

The float16 sessions then ran beside the same production 10,000-directory/
100,000-file scanner/catalog/SQLite/storyboard test used for the float32
comparison. Container peak was 2,905,825,280 bytes, below the existing 3.2 GiB
limit. The AI process completed 295 image/text cycles with finite outputs.

| Metric | No-AI baseline | Float16 AI | Change |
| --- | ---: | ---: | ---: |
| Scan duration | 69.472 s | 72.746 s | +4.7% |
| Scan-period read/search P95 | 0.657/38.526 ms | 0.702/39.216 ms | +6.8%/+1.8% |
| Recursive browse P95 | 28.267 ms | 32.369 ms | +14.5% |
| FTS/short search P95 | 20.769/35.961 ms | 23.365/37.772 ms | +12.5%/+5.0% |
| Global search P95 | 18.811 ms | 26.631 ms | +41.6% |
| Storyboard-period browse P95 | 39.371 ms | 40.065 ms | +1.8% |

Memory and ordinary recursive browsing passed the provisional limits. Global
search did not pass the 20% relative allowance in this single comparison, and
the microsecond-scale directory-list P95 varied from 51 to 121 µs. Both runs
also failed the pre-existing absolute search-keyset budget. These facts require
repeated paired measurements on the accepted runtime/target; they do not
justify changing a threshold.

Machine-readable results are in
[`siglip1-float16-production-capacity-linux-arm64-2026-08-27.json`](siglip1-float16-production-capacity-linux-arm64-2026-08-27.json).

## Decision

The float16-internal SigLIP 1 graph becomes the resource-priority candidate for
the next S0 measurements. It is not selected: representative 1,000-image
quality, repeated production comparisons, a single frozen runtime version,
native Linux/amd64, real embedding backfill and full HTTP/browser/face load are
still absent. `INT-006`, `INT-008`, `INT-013`, `R-024` and INT-S0 remain open.
