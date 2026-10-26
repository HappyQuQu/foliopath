# OCR producer / shared-admission local probe

CR-2026-048 / ADR-0034. This is an opt-in candidate AI image and an isolated
producer/consumer verification, not application OCR jobs, indexing or search.
It never reads a real media library, connects to server222 or updates 6010.

```sh
go run ./spikes/ocr-service/contractgen
go run ./spikes/ocr-service/contractgen -check
docker build -f services/inference/Dockerfile.ocr \
  -t foliopath-inference:ocr-candidate-20261025 .
docker run --rm --network none --read-only --tmpfs /tmp \
  -e MPLCONFIGDIR=/tmp/matplotlib --entrypoint python \
  foliopath-inference:ocr-candidate-20261025 \
  -m unittest ocr_tests test_scheduler test_runtime_manifest
docker build --target ocr-service-probe -t foliopath:ocr-service-probe-20261025 .
python3 spikes/ocr-service/run_probe.py \
  --image foliopath-inference:ocr-candidate-20261025 \
  --probe foliopath:ocr-service-probe-20261025 \
  --output "$PWD/.artifacts/ocr-service/actual"
```

Resolve immutable digests before verification/promotion. The final evidence uses
fixed digests, not the mutable convenience tags above. The model caches must
already exist: the default complete legacy cache is
`/Volumes/extended/foliopath/model-experiments/multilingual-integration-cache`,
and the independent OCR cache is `/Volumes/extended/foliopath/ocr-feasibility-cache`.
`--cache`/`--ocr-cache` may select other explicit verified copies. There are no
runtime downloads; preparation belongs to the earlier reviewed OCR spike.

The host runner checks every legacy model byte against the pinned image's
manifests and the synthetic fixture hashes. It starts a new internal network with
no published ports, mounts model sources read-only, and uses dedicated disposable
disk scratch (no media/DB/application data). Service RAM/swap are 4 GiB / no extra
swap, 2 CPUs; the separate native Go/race client uses 1 GiB / 2 CPUs.

It sequentially runs the frozen baseline image, the OCR image, then the OCR image
without its OCR cache. A preconfigured identity is read **offline from trusted
image artifacts**, never fetched from HTTP and trusted on first use. The Go
probe uses production imagevips/HTTP adapters. It compares old text/image vectors,
JoyTag labels and face results byte for byte; then checks actual Chinese/English,
EXIF, wide, blank and invalid PNG cases. Missing OCR cache must leave old outputs
unchanged and report OCR unavailable. Startup/predict temporary snapshots and
named containers/network are removed; ignored logs and synthetic observations remain.

To run the additional real-weight cancellation/cache-fault tests, mount the
separate OCR cache read-only at `/ocr-cache`, an empty disposable directory at
`/verified-cache`, use network none/root read-only/4 GiB RAM with no extra swap,
and run the same immutable AI image with `--entrypoint python -m ocr_native_tests`.
These tests observe child RSS before cancellation, require reaping/empty scratch
and anonymous-memory recovery, then check both OCR and legacy dispatch recovery.
They use only generated white pixels and private fake damaged model caches.

The routing adapter uses the frozen core's **visual admission category** solely
as a compatibility shim, with a request-local explicit OCR route. HTTP and worker
kind remain `ocr`; old model paths, commands and outputs are unchanged. No copied
admission, timeout, reaper or second queue is permitted. `scripts/check_ocr_ownership.py`
is part of `make arch-check`; generated policies are checked by `make generate-check`.

Known open work: source generations/fenced persistent jobs/clearing/access,
application settings and OCR search/UI; no-EXIF 90-degree/layout/general accuracy;
native amd64, combined application+AI capacity, long-running and release gates.
See `docs/changes/CR-2026-048-ocr-producer.md` for exact executed evidence and first failures.
