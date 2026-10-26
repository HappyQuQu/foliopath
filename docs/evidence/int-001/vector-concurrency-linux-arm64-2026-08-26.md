# INT-009/013 constrained vector concurrency evidence

Status: **measurement completed; combined Gate not met**.

The isolated spike ran three fresh native Linux/arm64 processes with four CPUs,
4 GiB memory and no network. Each process seeded 100,000 asset rows, backfilled
512-dimensional float32 and float16 vectors in bounded 256-row transactions, ran exact
search and keyset-browse proxies while the writer continued, cancelled the
writer after at least half the rows were committed, then restarted it to
100,000 rows.

Machine results: [JSON](vector-concurrency-linux-arm64-2026-08-26.json).

## Result

- Float32 exact search P95 was 229.843–258.371 ms; float16 was
  138.854–156.912 ms. Both were below the provisional 750 ms budget.
- Backfill cancellation preserved 65,024–68,352 committed rows and every run
  restarted to exactly 100,000 rows.
- Peak process RSS was 24.9–27.9 MiB for the vector/SQLite process alone.
- The checkpointed float32 database was 410,619,904 bytes; float16 was
  136,880,128 bytes. SQLite page layout makes this reduction larger than the
  raw two-to-one element-width ratio.
- Browse P95 remained below 0.5 ms in absolute terms, but relative degradation
  was 16.25–23.22 times the 0.018–0.024 ms synthetic baseline. That fails the
  configured 20% relative gate; the threshold was not relaxed.

The browse proxy is too small to substitute for the production HTTP/catalog
path, so this failure does not prove user-visible browse is slow. It does prove
that this microbenchmark cannot close the browse gate.

More importantly, 100k × 512 float32 already consumes about 78% of the entire
500 MiB initial derived-index budget before video-frame vectors, face vectors,
WAL headroom or backup overhead. Float32 exact storage is therefore rejected as
the final combined-scope layout. Real embeddings must show acceptable float16
or lower-dimensional recall and cross-architecture tolerance; otherwise the
video or face/vector scope must be reduced.

Float16 is now the only exact-SQLite capacity path worth continuing: it keeps
substantial headroom under 500 MiB and was faster in this I/O-bound synthetic
scan. It is not selected yet. Synthetic random-vector recall and arm64 timing
cannot establish real SigLIP retrieval quality or amd64 numerical tolerance.
