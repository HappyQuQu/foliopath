# Production catalog search query-plan diagnosis

Status: **root execution shape proven; a broad-query order-first candidate passes
isolated ID selection, but production search still fails**.

The production catalog capacity test was extended with benchmark-only
`EXPLAIN QUERY PLAN` capture. It ran without an AI runtime in native
Linux/arm64 with 4 CPUs, 4 GiB, no network, 10,000 directories and 100,000
assets. The existing `s4-search-v1` 250 ms keyset limit and production SQL were
not changed.

For the broad name-ascending `asset` search, the count plan is:

- `SCAN asset_search VIRTUAL TABLE INDEX 0:M2`;
- `SEARCH a USING INTEGER PRIMARY KEY (rowid=?)`.

The first- and second-page list plans both start with the same FTS virtual-table
scan and asset rowid lookup, then end with `USE TEMP B-TREE FOR ORDER BY`.
Both plans also perform indexed thumbnail lookups and the correlated favorite
lookup. The second-page keyset predicate does not change the plan shape.

This proves the main structural issue: the broad trigram FTS match drives the
candidate stream, and SQLite materializes a temporary sort for the derived
directory-path/name order. The existing
`assets_browse_folder_name_v2` expression index is not used to produce this
search order, so keyset page two cannot stop after walking the next 101 ordered
matches. Count separately scans all FTS candidates and returns to `assets` for
kind totals.

In this diagnostic run, repository count/list P95 values were 61.370 ms,
348.923 ms and 377.920 ms; service first/second pages were 409.111 ms and
443.540 ms, and the combined keyset operation was 847.735 ms. These absolute
numbers are not compared with the earlier image because this run used a new
statically linked diagnostic binary and a no-SSL distroless harness. The query
plans—not cross-run latency deltas—are the deciding evidence.

Adding another ordinary browse index alone cannot fix a plan that is driven by
the FTS rowid stream and explicitly sorts into a temporary B-tree. The approved
next work belongs to a separate maintenance slice.

One benchmark-only comparison now forces the existing
`assets_browse_folder_name_v2` order and checks FTS membership per asset. On a
fresh run with the same platform and fixture, its first and second 101-ID pages
exactly matched the production repository result. Across two extended runs,
ID-selection P95 was 16.843–33.990 ms and 16.037–32.451 ms; complete first-page
asset, thumbnail, storyboard and favorite hydration was 16.988–33.668 ms. A
sparse `asset-099` first page was 53.767–110.061 ms and also returned the same
101 ordered IDs. Candidate active cancellation converged in 2.332 ms in the
latest 100k run. Production service first-plus-second P95 varied from 849.441
ms to 1.605 s in those runs. Candidate plans use the browse expression index
and an FTS `EXISTS` lookup; none contains a temporary order B-tree.

This does not select the production fix. Only library-scope name-ascending was
tested, using one broad and one sparse synthetic term. The existing service already computes an
exact count before listing, but its repository contract does not pass that
result as a planner hint. A maintenance Gate must therefore compare a
count-driven hybrid threshold across broad and sparse terms, every supported
scope/filter/sort/order, full row hydration, cancellation and both native
architectures. Bounded/materialized candidates and count revision caching or
first-page-only count semantics remain alternatives. No production query,
schema, API or threshold changed.

Machine-readable results are in
[`catalog-search-query-plan-linux-arm64-2026-08-27.json`](catalog-search-query-plan-linux-arm64-2026-08-27.json).

Follow-up: the later
[scope/filter/sort/cursor matrix](catalog-search-order-first-matrix-linux-arm64-2026-08-27.md)
closed the initial single-query correctness gap for 24 first-page and 19
second-page combinations on arm64. It did not close mixed-media,
multi-library, repeated-distribution, filtered-plan, full-hydration or amd64
evidence, and it did not authorize a production change.
