# S2 local capacity refresh — Darwin/arm64

- Date: 2026-09-01
- Host: Darwin/arm64, `GOMAXPROCS=4`
- Status: **local backend capacity evidence passed; not final dual-architecture release evidence**

The repository's enforced 10,000-directory/100,000-asset capacity target completed successfully:

```text
make spike-capacity
TestCapacityBaseline PASS (99.04s)
TestDirectoryRollupDeepChainBaseline PASS (1.10s)
searchKeysetP95Us=130238
concurrentReadP95Us=369
concurrentSearchP95Us=66276
peakGoHeapAllocBytes=51979024
databaseAndWalSizeBytes=157274112
budgetViolations=[]
```

The candidate-dimension face clustering target also completed:

```text
make test-face-capacity
face_count=100000
embedding_dimension=512
paired_cluster_count=50000
singleton_cluster_count=100000
elapsed_ms=7174
memory_sys_bytes=409381208
deterministic_sha256=ed978ca7f471ba742a38f680cebb5f83481b8f622a70b005b906e654f2b706d4
```

These runs prove the local bounded-query and clustering implementation surfaces used by S2. They do not contain a
final reviewed model, native Linux/amd64+arm64 final-image pairing, real inference RSS, governed quality results or
release signatures. Those remain release evidence under `INT-402/403/404/411`.
