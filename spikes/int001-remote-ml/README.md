# Independent ML service probe

## Persisted real-service probe

The opt-in `persisted-probe` runs the existing backfill processor with native
image preparation and real HTTP inference, then closes/reopens a temporary SQLite
database and queries the stored vectors using a real text embedding. Only generated
JPEG bytes are used; the catalog and generation identity are synthetic fixtures.
This does not exercise deployment admission, activation, HTTP API or production UI.
The test container has no host mounts; its database lives in bounded `/tmp` tmpfs.

```sh
export FOLIOPATH_TEST_MODEL_CACHE=/Volumes/extended/foliopath/model-cache
docker build --target remote-ml-regression -t foliopath-remote-ml-regression:local .
docker compose -f spikes/int001-remote-ml/compose.yaml up -d machine-learning
docker compose -f spikes/int001-remote-ml/compose.yaml run --rm persisted-probe
docker compose -f spikes/int001-remote-ml/compose.yaml stop machine-learning
```

Observed 2026-09-15: PASS with race detection, 3 synthetic images committed,
3 matches after reopening. This proves plumbing, not retrieval relevance.

## Candidate main-process chain

Build `docker build -t foliopath-app:candidate .` and layer `compose.app.yaml`
over `compose.yaml` and `compose.foliopath.yaml`, supplying the independently
read image identity through `FOLIOPATH_AI_DEPLOYMENT` (see deployment docs).
This uses the actual distroless runtime including FFmpeg, not the regression
image. Only synthetic `tests/fixtures/media` is mounted, read-only; state uses
the isolated `candidate-app-data` volume and the app publishes loopback 6091.

Initialize the isolated instance with administrator username `candidate-test`.
Prepare the three-second storyboard fixture with
`export FOLIOPATH_TEST_SYNTHETIC_MEDIA=$(sh spikes/int001-remote-ml/prepare-app-fixtures.sh)`
before creating the app container. This creates a fresh temporary synthetic tree;
never set this variable to a real media library. Fixture generation uses local
FFmpeg; application and inference verification run in Docker.
Set its password in `FOLIOPATH_TEST_PASSWORD`, then run:

```sh
node spikes/int001-remote-ml/app-chain.mjs
```

Set `FOLIOPATH_TEST_INVALIDATION=1` to additionally change the generated image's
timestamp and verify scan-driven stale coverage followed by missing backfill.
This requires the same `FOLIOPATH_TEST_SYNTHETIC_MEDIA` temporary directory used
by Compose; it rejects other root names and symlinks. Never use real media.

Set `FOLIOPATH_TEST_VIDEO_REMOVAL=1` to temporarily move the generated 3-second
video outside the test root, scan, verify pending coverage, and reclassify the
remaining short video. The video is restored in `finally` and a scan requested;
rerun the base script to rebuild its video index. The temporary holding directory
is retained, and no file is permanently deleted. Each run uses fresh job keys.

The bounded script logs in, creates/reuses the synthetic library, enables AI,
requests image and video backfill, and checks coverage and real text searches.
It may retry failed derived media via the public repair API, never edit originals.
It is only for this isolated Compose instance, not another server on port 6091.
Stop both services after testing; preserve the test volume for restart checks.

## Local Docker regression suite

2026-09-15 observed result: semantic/imagevips passed; the entire SQLite race
suite initially exceeded its 5-minute budget. The unchanged full suite subsequently
passed with an explicit 15-minute budget (exit 0). The initial timeout remains in
the evidence record. The Dockerfile now gives the full SQLite suite 15 minutes
and verbose output, without removing tests or race detection. The same full command
can be run against an already-built image:

```sh
docker compose -f spikes/int001-remote-ml/compose.yaml run --rm regression \
  '/out/sqlite.test -test.v -test.timeout=15m'
```

```sh
docker compose -f spikes/int001-remote-ml/compose.yaml run --rm regression \
  '/out/sqlite.test -test.v -test.run "TestGenerationSpace|TestBoundEncoderUsesPersistedGeneration|TestStreamBackfillPersistedLifecycle|TestSemanticBackfillProcessor" -test.timeout=3m'
```

Set `FOLIOPATH_TEST_MODEL_CACHE` as shown below before Compose commands.

The user-requested local Docker workflow (2026-09-15) runs all semantic, imagevips
and SQLite package tests with the Go race detector. The regression container has
no network, a read-only root and a bounded temporary filesystem for synthetic test
databases. No host media/database volume is mounted. The encoded probe separately
uses only the private inference network.

```sh
export FOLIOPATH_TEST_MODEL_CACHE=/Volumes/extended/foliopath/model-cache
docker compose -f spikes/int001-remote-ml/compose.yaml build regression
docker compose -f spikes/int001-remote-ml/compose.yaml run --rm regression
node spikes/int001-remote-ml/cache-audit.mjs
docker compose -f spikes/int001-remote-ml/compose.yaml up -d machine-learning
docker compose -f spikes/int001-remote-ml/compose.yaml run --rm encoded-probe
docker compose -f spikes/int001-remote-ml/compose.yaml stop machine-learning
node spikes/int001-remote-ml/cache-audit.mjs
```

Stop machine-learning even if the probe fails. Package regressions and the real
ML probe are separate evidence: this is not the complete application UI/search
deployment. The regression image includes build dependencies and is not a release
image; never push it as the FolioPath application.

## Native image preparation to real service probe

This opt-in target is excluded from the production runtime image. It generates
PNG/JPEG/GIF/WebP in memory, checks preparation to 1024x512 PNG, then submits each
to the private ML service. It also checks a text vector. It neither opens media
nor writes an index, and does not attest model identity or retrieval quality.

```sh
docker build --target remote-ml-image-probe -t foliopath-remote-image-probe:local .
export FOLIOPATH_TEST_MODEL_CACHE=/Volumes/extended/foliopath/model-cache
node spikes/int001-remote-ml/cache-audit.mjs
docker compose -f spikes/int001-remote-ml/compose.yaml up -d machine-learning
docker run --rm --network foliopath-ml-probe_inference \
  --cpus=1 --memory=512m --pids-limit=64 --cap-drop=ALL \
  --security-opt=no-new-privileges --read-only \
  -e FOLIOPATH_TEST_ML_ORIGIN=http://machine-learning:3003 \
  foliopath-remote-image-probe:local
docker compose -f spikes/int001-remote-ml/compose.yaml stop machine-learning
node spikes/int001-remote-ml/cache-audit.mjs
```

The probe has no volume mounts and no published ports. A cold service may need
startup time before the test; a failed run must still be followed by the stop
command. The successful local run is Linux/arm64 only.

## Local cache baseline check (2026-09-14)

`cache-baseline.json` records the complete copied local fixture (including hidden
metadata), not an upstream-signed release manifest. Never regenerate it to make a
failed verification pass without reviewing the source change. This baseline is
diagnostic evidence only; it must not authorize production activation.

```sh
node --test spikes/int001-remote-ml/cache-audit.test.mjs
FOLIOPATH_TEST_MODEL_CACHE=/Volumes/extended/foliopath/model-cache \
  node spikes/int001-remote-ml/cache-audit.mjs
```

The scanner is bounded to 1,000 files, 2 GiB and depth 16, includes hidden files,
rejects symlinks/nonregular files and checks metadata changes during hashing.
Run only against a trusted, quiescent local copied model cache. It is not a
kernel-anchored, race-safe filesystem boundary; parent-path replacement, mount
crossing and post-verification host writes are not prevented. The ML container's
read-only mount does not prevent host writes. Do not use this script on media.

Current baseline: 49 files / 802,152,765 bytes, manifest SHA-256
`c8be0544ab43ffd0c3f81e550218c833410d4e593a3b282fc7844f811606211b`.

POST-MVP-5 / CR-2026-030 / ADR-0019. This is an isolated protocol probe, not the
production FolioPath integration. The main application is not wired to this service yet.

The Compose pins the official Immich ML v3.1.0 image digest. It starts only an ML
server plus a disposable HTTP test client on an internal Docker network, with no
published ports, no original-media mount and no application database.
The model cache is read-only and Hugging Face offline mode is enabled. Missing
models must fail, never download silently during this test.

```sh
export FOLIOPATH_TEST_MODEL_CACHE=/Volumes/extended/foliopath/model-cache
docker compose -f spikes/int001-remote-ml/compose.yaml up -d machine-learning
docker compose -f spikes/int001-remote-ml/compose.yaml run --rm probe
docker compose -f spikes/int001-remote-ml/compose.yaml down
```

The client creates a solid red image in memory and submits it and a fixed English
text. It checks finite nonzero 512-dimensional CLIP outputs, repeated-text stability,
and an empty face result for the solid-color image. It does not prove recognition
quality, Chinese retrieval quality, face embeddings on real faces, automatic
download, failure recovery, a complete memory budget, or application integration.

Fixtures copied from server222: ViT-B-32__openai and buffalo_l. Keep models/media
outside Git. Cache source changes require rechecking all file hashes. The current
Go SigLIP 768-dimensional and YuNet/AuraFace contracts cannot be reused unchanged.

Protocol references:
- [v3.1.0 request schema](https://github.com/immich-app/immich/blob/v3.1.0/machine-learning/immich_ml/schemas.py)
- [v3.1.0 configuration](https://github.com/immich-app/immich/blob/v3.1.0/machine-learning/immich_ml/config.py)

## Go adapter probe

The bounded Go transport now lives in `internal/inference/remoteml`, governed by
`docs/architecture/independent-ml-transport-contract.md`. It is not wired to the
main application. Ordinary tests use only httptest, with no ML service required:

```sh
go test -race ./internal/inference/remoteml ./internal/semantic
```

To check it against the isolated service on an arm64 Docker host, start the ML
service above, then build/run the opt-in synthetic-image test from the repo root:

```sh
probe_dir=$(mktemp -d /tmp/foliopath-go-ml-probe.XXXXXX)
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go test -c -tags=remotemlprobe \
  -o "$probe_dir/probe.test" ./internal/inference/remoteml
docker run --rm --network foliopath-ml-probe_inference \
  --cpus=1 --memory=256m --pids-limit=64 --cap-drop=ALL \
  --security-opt=no-new-privileges \
  --mount "type=bind,source=$probe_dir,target=/probe,readonly" \
  -e FOLIOPATH_TEST_ML_ORIGIN=http://machine-learning:3003 \
  --entrypoint /probe/probe.test \
  ghcr.io/immich-app/immich-machine-learning:v3.1.0@sha256:5a0839dc5303cd7215bcd2180a26aed3af41675aefb3e75e5157e9f10ad16e6e \
  -test.run '^TestOfflineServiceProbe$' -test.v -test.timeout=2m
docker compose -f spikes/int001-remote-ml/compose.yaml stop machine-learning
```

Set GOARCH=amd64 on an amd64 host; that variant has not been verified in this
slice. This test checks neither model identity attestation nor application search.
