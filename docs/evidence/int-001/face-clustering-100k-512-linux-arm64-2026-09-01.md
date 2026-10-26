# Face clustering 100k × 512 bounded evidence — Linux/arm64

Date: 2026-09-01
Environment: Docker Desktop Linux VM on Apple M4 Max, native arm64 container
Resource tier: 4 CPUs, 4 GiB memory, 256 PIDs, read-only rootfs, no network

The capacity test runs two deterministic 512-dimensional, 100,000-observation extremes: 50,000 exact pairs that must
all become core clusters, then 100,000 distinct observations that must remain edge-only singletons. It contains no
photographs, identities, demographic labels, paths or model output.

The first upgraded run on Darwin exposed an LSH bucket-boundary defect and timed out after 10 minutes in the edge
assignment scan. After correcting the bounded neighbor iteration and adding a bucket-boundary regression, the same
workload completed locally and in the constrained Linux container.

Linux/arm64 command shape:

```sh
docker run --rm --network none --read-only \
  --tmpfs /tmp:rw,exec,nosuid,nodev,size=256m \
  --cpus 4 --memory 4g --pids-limit 256 \
  --env FOLIOPATH_RUN_CAPACITY_TEST=1 --env GOMAXPROCS=4 \
  --entrypoint /out/face-capacity.test IMAGE \
  -test.v -test.timeout=10m -test.run '^TestClusterFaces100KCapacity$'
```

Observed result:

```text
face_count=100000
embedding_dimension=512
paired_cluster_count=50000
paired_member_count=100000
singleton_cluster_count=100000
singleton_member_count=100000
goos=linux
goarch=arm64
deterministic_sha256=ed978ca7f471ba742a38f680cebb5f83481b8f622a70b005b906e654f2b706d4
elapsed_ms=7157
memory_sys_bytes=425300328
PASS
```

The native workflow now reproduces this test on both architecture runners and its strict verifier requires the same
deterministic result. That workflow has not yet run from a registered source commit, so no paired native claim is made.
This synthetic test verifies bounded core and worst-case singleton/edge clustering behavior only; it does not prove
real-face recall, precision, bias, runtime RSS, database/browser concurrency, or the final joint 100k release workload.
