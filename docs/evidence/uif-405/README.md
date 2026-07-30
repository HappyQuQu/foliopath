# UIF-405 capacity and concurrency evidence

## Scope

UIF-405 revalidates `UIF-AC-011` after the production frontend fidelity work:

- the 100,000-item shared media collection remains virtualized and scrollable;
- Chromium, Firefox and WebKit remain inside the frozen FPS, frame-interval,
  mounted-DOM and browser-process RSS budgets;
- the backend still scans 10,000 directories and 100,000 independent files
  while serving bounded browse and search reads;
- durable scan workers still enforce cross-library concurrency and ordering.

This evidence supplements rather than rewrites the accepted
[`S5-005`](../../gates/MVP-2026-07-23/s5-release-capacity-candidate.md)
candidate results. The backend rerun was on physical macOS/arm64 and therefore
does not claim Linux process RSS; the accepted native linux/amd64 and
Docker linux/arm64 resource evidence remains owned by S5-005.

## Environment and budgets

- physical arm64 Mac, macOS 26.6;
- Node 22.22.2, Playwright 1.61.1;
- Go with `GOMAXPROCS=4`;
- browser viewport `1280 × 720`;
- 100,000 frontend media items;
- 10,000 backend directories and 100,000 independent four-byte JPEG fixtures.

The frozen browser budgets are:

| Metric | Budget |
| --- | ---: |
| FPS | at least 45 |
| frame interval P95 | at most 34 ms |
| browser process-tree peak RSS | at most 1.5 GiB |
| mounted media items | at most 64 |

The backend `stage0-comparable-v1` and `s4-search-v1` profiles enforce scan,
query, cancellation, search rebuild, bounded storyboard admission and database
family budgets. The run fails if any metric exceeds its canonical owner in
`tests/performance/capacity_test.go`.

## Browser result

`make test-browser-capacity` rebuilt the component workbench, scrolled
`Patterns/MediaCollection/Capacity100k` continuously for five seconds in each
engine, sampled animation frames and the complete browser process tree, and
enforced every budget.

| Engine | FPS | frame P95 | long frames | mounted | peak RSS |
| --- | ---: | ---: | ---: | ---: | ---: |
| Chromium | 59.996 | 18.4 ms | 0 | 60 | 727,465,984 B |
| Firefox | 58.000 | 17.22 ms | 0 | 60 | 1,350,352,896 B |
| WebKit | 59.952 | 22.0 ms | 0 | 60 | 117,538,816 B |

All three engines passed. The raw report is stored in
[`browser-capacity.json`](browser-capacity.json).

## Backend result

`make spike-capacity` created and scanned the complete 10k/100k fixture while
issuing browse and search reads every 25 ms:

| Metric | Result |
| --- | ---: |
| fixture creation | 8,033 ms |
| full scan | 58,836 ms |
| concurrent browse samples / P95 / max | 2,353 / 0.369 ms / 21.523 ms |
| concurrent search samples / P95 / max | 2,353 / 12.275 ms / 25.939 ms |
| recursive Browse P95 | 0.970 ms |
| FTS / short / global Search P95 | 12.808 / 0.449 / 9.737 ms |
| keyset Search P95 | 140.083 ms |
| storyboard admission count / maximum batch | 10,000 / 128 |
| peak Go heap allocation | 43,211,160 B |
| SQLite database family | 134,041,600 B |
| budget violations | 0 |

The same command also completed the 1,000-level directory rollup in 73 ms.
The raw capacity metrics are stored in
[`backend-capacity.json`](backend-capacity.json).

`TestDurableScanWorkerEnforcesCrossLibraryConcurrencyAndOrder` then passed
independently, confirming that a third library cannot start before the bounded
global worker capacity is released.

## Commands and conclusion

Executed from the repository root:

```sh
make test-browser-capacity
make spike-capacity
go test -count=1 \
  -run '^TestDurableScanWorkerEnforcesCrossLibraryConcurrencyAndOrder$' \
  -v ./tests/integration
```

UIF-405 is complete: the latest shared collection preserves bounded DOM and
smooth three-engine scrolling, and the backend preserves keyset/query and
scan-time concurrency behavior at the full 100k/10k cardinality. This does not
rerun the multi-hour full thumbnail derivation or replace the accepted native
Linux release-capacity evidence.
