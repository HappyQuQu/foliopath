# Face clustering 100k bounded-candidate evidence — darwin/arm64

Status: superseded for capacity dimension and LSH boundary behavior by the
[100k × 512 Linux/arm64 follow-up](face-clustering-100k-512-linux-arm64-2026-09-01.md). This historical 32-dimensional
run remains retained as the original baseline.

Date: 2026-09-01
Host: Apple M4 Max, macOS 26.6.2 (25G83), native arm64
Scope: synthetic capacity/algorithm evidence only

Command executed from the repository root:

```sh
/usr/bin/time -l env FOLIOPATH_RUN_CAPACITY_TEST=1 \
  go test ./internal/face -run '^TestClusterFaces100KCapacity$' -count=1 -v
```

Observed result:

```text
--- PASS: TestClusterFaces100KCapacity (45.47s)
PASS
ok github.com/HappyQuQu/foliopath/internal/face 45.951s
46.14 real, 45.50 user, 0.74 sys
96,190,464 maximum resident set size
0 swaps
```

The fixture contains 100,000 normalized 32-dimensional observations arranged as
50,000 exact synthetic pairs. It verifies that every input face is emitted once
while the large-input path uses deterministic bounded LSH neighborhoods followed
by exact cosine and cannot-link checks. The prior all-pairs candidate construction
is retained only below 4,096 faces.

This closes neither `INT-250` nor the S2C Gate. It does not contain real-face
ground truth, does not prove the frozen 99.5% anonymous-core precision threshold,
does not exercise SQLite/HTTP/runtime concurrency, and is not the required native
Linux amd64/arm64 4-CPU/4-GiB joint 100k-media evidence.
