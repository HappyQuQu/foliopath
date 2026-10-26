# Catalog search two-library global matrix

Status: **cross-library ordering subproof passed; production fix still not selected**.

Two non-overlapping libraries were created through the real library service and
scanned through the production scanner. Each contained 5,000 directories and
50,000 assets. On native Linux/arm64 with 4 CPU/4 GiB, no network and a
read-only root filesystem, the benchmark-only candidate was compared with the
production repository for global name/modified/size ordering in both directions.
The derived catalog was then split into different per-library image/video/
animated proportions and exercised with video-only, animated-only,
image-plus-video, a middle-50-percent modification window and a roughly
two-percent filename band.

All 11 first pages and all 11 second pages returned exactly the same ordered
asset IDs, including the global library-ID tie-break. Candidate one-shot page
latency was at most 82.253 ms, below the unchanged isolated 250 ms
page budget. The diagnostic binary and full timings are pinned in the adjacent
JSON.

This closes the synthetic cross-library mixed-media/date/sparse correctness
subproof, not the production Gate. Real-world distributions and selectivity,
complete hydration, repeated P95 measurements and native amd64 remain missing.
Production SQL, schema, cursor, API, threshold and budget were not changed.
