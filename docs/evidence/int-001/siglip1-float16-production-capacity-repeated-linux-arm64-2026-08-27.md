# Repeated SigLIP 1 float16 production-capacity comparison

Status: **memory and ordinary browse subproof passed on arm64; complete capacity
Gate remains open**.

## Repeated paired runs

Three fresh no-AI containers were each followed by a fresh float16-AI
container. Every container used native Linux/arm64, 4 CPUs, 4 GiB, no network,
10,000 directories, 100,000 files and the same production scanner,
`catalog.Service`, SQLite and storyboard-admission capacity test. Each AI run
kept both SigLIP 1 float16-internal sessions resident and continuously inferred
until the production workload completed.

| Metric | Pair 1 | Pair 2 | Pair 3 | Median change |
| --- | ---: | ---: | ---: | ---: |
| AI container peak | 2.906 GB | 2.860 GB | 2.951 GB | — |
| Scan duration | +4.7% | +0.4% | +6.6% | +4.7% |
| Scan-period read P95 | +6.8% | +13.7% | +11.1% | +11.1% |
| Scan-period search P95 | +1.8% | +4.6% | +0.4% | +1.8% |
| Recursive browse P95 | +14.5% | +10.4% | +6.0% | +10.4% |
| Short search P95 | +5.0% | +9.6% | +7.4% | +7.4% |
| Global search P95 | +41.6% | -4.5% | -3.3% | -3.3% |
| Search-keyset P95 | +6.3% | +8.7% | +4.0% | +6.3% |
| Storyboard-period browse P95 | +1.8% | +20.34% | +15.8% | +15.8% |

All three AI peaks passed the 3.2 GiB limit, and all ordinary recursive-browse
comparisons passed the 20% relative limit. The first run's +41.6% global-search
result did not reproduce, indicating high single-run variance rather than a
stable regression.

The complete Gate still does not pass. Pair 2 storyboard-period browse exceeded
20% by 0.335 percentage points. More importantly, search-keyset P95 was
352–363 ms without AI and 377–383 ms with AI, so all six runs failed the
pre-existing 250 ms absolute budget. There is no clean all-budget baseline on
this host.

Machine-readable results are in
[`siglip1-float16-production-capacity-repeated-linux-arm64-2026-08-27.json`](siglip1-float16-production-capacity-repeated-linux-arm64-2026-08-27.json).

## Decision

The arm64 float16 memory and ordinary-browse subproof is repeatable enough to
retain this candidate. `INT-013` remains incomplete because one query path
missed the relative limit, the existing keyset absolute budget failed in every
baseline, and the test still lacks real embedding backfill, authenticated HTTP,
browser/face load and native Linux/amd64. No threshold is relaxed.
