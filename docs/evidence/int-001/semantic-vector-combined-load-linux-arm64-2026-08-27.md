# Semantic image encoder plus vector concurrency on Linux/arm64

Status: **partial capacity evidence; browse gate failed; INT-S0 remains No-Go**.

## What ran

Three fresh native Linux/arm64 containers ran with 4 CPUs, a 4 GiB memory
limit and no network. Each container continuously executed the pinned split
SigLIP 2 image encoder while the isolated Go/SQLite spike performed a bounded
100,000 × 512 float16 backfill, exact search, keyset-browse proxy, cooperative
cancellation and restart.

The ONNX session used the allocator configuration established by the prior
lifecycle test: CPU memory arena disabled, memory pattern enabled, two image
inference threads and a 20 ms pause between requests. The vector writer used
256-row transactions and a 20 ms synthetic inference delay. Model, prepared
tensor, container and spike-binary identities are recorded in the
machine-readable result.

## Results

| Run | Image P95 | Search P95 | Browse baseline → concurrent | Relative browse degradation | Rows cancel → restart | Container peak |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| 1 | 182.0 ms | 165.4 ms | 0.021 → 0.167 ms | 6.95× | 63,232 → 100,000 | 1,327,558,656 B |
| 2 | 179.2 ms | 186.5 ms | 0.019 → 0.261 ms | 12.74× | 63,232 → 100,000 | 1,292,042,240 B |
| 3 | 182.4 ms | 184.4 ms | 0.028 → 0.177 ms | 5.32× | 63,488 → 100,000 | 1,314,099,200 B |

All image outputs were finite. No process failed or exceeded the 4 GiB
container limit. Exact-search P95 stayed below the provisional 750 ms limit,
all three restarts converged to 100,000 rows, and the maximum container peak
stayed below the `R-024` 3.2 GiB threshold.

The unchanged relative browse-degradation limit is 20%, and all three runs
failed it. Absolute concurrent proxy latency stayed below 0.3 ms, so this
microbenchmark is too small to predict user-visible production browsing; that
does not justify changing or ignoring the gate. A production HTTP/catalog
browse measurement is still required.

Machine-readable results are in
[`semantic-vector-combined-load-linux-arm64-2026-08-27.json`](semantic-vector-combined-load-linux-arm64-2026-08-27.json).

## Decision

The arm64 candidate can continue to production-adapter and representative-data
experiments: this run found no immediate 4 GiB memory or recovery blocker when
one real image encoder shares resources with the synthetic vector workload.
`INT-013` remains incomplete because this was not the complete FolioPath
process, did not encode 100,000 real images, did not run the face runtime or
production HTTP browse path, and did not run on native Linux/amd64. The browse
relative gate also remains failed.
