# SigLIP 1 beside the production 100k catalog capacity path

Status: **micro-proxy concern narrowed; full-process memory gate failed;
INT-S0 remains No-Go**.

## Method

The current `tests/performance` capacity binary was cross-compiled without
changing production code and run in fresh native Linux/arm64 containers with
4 CPUs, 4 GiB and no network. Both runs created 10,000 real directories and
100,000 real fixture files, performed a production full scan, exercised the
SQLite store and `catalog.Service` browse/search operations, and admitted the
existing 10,000-video storyboard capacity workload.

The paired AI run kept both arena-off SigLIP 1 split sessions resident and
continuously alternated image/text inference until the capacity test ended. It
completed 356 cycles with finite outputs. Budgets were collected but not used
to terminate the paired runs, because an earlier enforced baseline had already
shown this environment failing the existing search-keyset budget.

## Results

| Metric | Baseline | With AI | Change |
| --- | ---: | ---: | ---: |
| Scan duration | 69.472 s | 69.847 s | +0.5% |
| Scan-period catalog read P95 | 0.657 ms | 0.704 ms | +7.2% |
| Scan-period catalog search P95 | 38.526 ms | 39.133 ms | +1.6% |
| Recursive browse P95 | 28.267 ms | 31.431 ms | +11.2% |
| FTS search P95 | 20.769 ms | 23.925 ms | +15.2% |
| Short search P95 | 35.961 ms | 38.683 ms | +7.6% |
| Global search P95 | 18.811 ms | 23.574 ms | +25.3% |
| Search keyset P95 | 355.609 ms | 380.979 ms | +7.1% |
| Storyboard-period browse P95 | 39.371 ms | 47.358 ms | +20.3% |
| Container peak | 1,604,067,328 B | 3,589,988,352 B | +1,985,921,024 B |

The production recursive browse result does not reproduce the isolated
micro-proxy's 4.76–8.70× relative degradation. That proxy is therefore not a
useful predictor of ordinary browsing and must not be the final Gate.

The candidate still fails the combined acceptance envelope. Container peak
exceeded the existing 3.2 GiB threshold. Global search and storyboard-period
browse exceeded the provisional 20% relative allowance in this single paired
comparison. Both baseline and AI runs also failed the pre-existing absolute
`searchKeysetP95Us <= 250000` budget (355.6/381.0 ms), so the environment does
not provide a clean all-budget baseline and the relative numbers need repeated
runs before attribution.

Machine-readable results are in
[`siglip1-production-catalog-capacity-linux-arm64-2026-08-27.json`](siglip1-production-catalog-capacity-linux-arm64-2026-08-27.json).

## Decision

The smaller model remains the resource-priority candidate, but it does not pass
`INT-013`. Its model-only/vector proxy fit under 3.2 GiB; the real 100k
filesystem/catalog/storyboard process did not. The next resource experiment
must materially reduce model memory—model quantization or a smaller approved
encoder—before repeating this production-path comparison. The existing
search-keyset baseline failure must also be resolved or measured on the
accepted native target; no threshold is changed here.
