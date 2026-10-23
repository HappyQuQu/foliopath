# Independent ML service probe

`p4-small-probe.mjs` now records the fixed anonymous manifest digest and public
search results by anonymous sample name (not database IDs). Compare paired runs
with `node spikes/int001-remote-ml/p4-rank-compare.mjs LEFT.json RIGHT.json`.
The comparison reports rank equality and public video score/frame differences;
it does not invent a tolerance, export embeddings, or approve model quality.

## Semantic preprocessing upgrade and paired rollback

`p4-semantic-upgrade-probe.mjs` uses synthetic media only. Set
`FOLIOPATH_P3_APP_IMAGE` to the old candidate, `FOLIOPATH_P4_UPGRADE_IMAGE` to the
new candidate, `FOLIOPATH_TEST_RESOURCE_PROFILE=p4-large`, and
`FOLIOPATH_P4_UPGRADE_REPORT` to an absolute ignored report path. It creates an
isolated old database, manual tag/person relation, and stopped-volume backup;
then verifies changed semantic generation, no inherited completed-vector count,
rebuild in both libraries, restart persistence and old-image/preupgrade-backup
rollback. The existing short-video fixture remains degraded before and after;
the probe requires that exact baseline rather than declaring all videos complete.
It never opens an existing user database or deploys an image.

### Large-image memory diagnostic

`facedecode` is opt-in (`FOLIOPATH_FACE_SAMPLE_DIAGNOSTIC=1`), reads only anonymous sample copies through `internal/files` on a read-only `/library`, and processes at most 100 inputs. Its trim/thread-cleanup switches are diagnostic experiments, not production behavior. Never run it against an original media mount. `p4-large` gives the app 1024 MiB and inference 2304 MiB with 1+3 CPUs and no swap; `p4-native` retains 768/2560 MiB. Record the profile with every result. See [memory evidence](../../docs/evidence/int-001/p4-face-memory-2026-09-16.md); semantic size-limit failures remain failures.

## Opt-in 100-sample diagnostic

`sampleprep` must run inside Linux Docker with the explicitly authorized source
mounted read-only at `/library`, a new output directory at `/output`, and
`FOLIOPATH_ALLOW_SAMPLE_READ=100-readonly`. It uses `internal/files`, bounded
candidate selection and anonymous copies, never source mutations. Do not add
real media to CI fixtures. `FOLIOPATH_SAMPLE_AUDIT=1` only reads headers from
an already-isolated sample mount and outputs aggregate pixel counts.

`p4-small-probe.mjs` uses `FOLIOPATH_SMALL_SAMPLE_ROOT` (a generated
`/tmp/foliopath-p4-small.*` directory), `FOLIOPATH_SMALL_MANIFEST` and
`FOLIOPATH_SMALL_REPORT`. It reuses the P3 synthetic bootstrap, creates a separate
100-sample library and checks actual coverage, search and copy hashes. Resource
profile `p4-native` remains explicit. It removes its test environment, retaining
the anonymous sample and report for review. This is not a quality benchmark.
Run `node --test spikes/int001-remote-ml/p4-small-result.test.mjs` for result-gate
regression. See `docs/evidence/int-001/p4-small-sample-2026-09-16.md` for retained
remote OOM failures; do not assume the current remote sample passes.

## P4 explicitly authorized native amd64 maintenance

The user authorized temporary service shutdown on server222 for isolated testing,
followed by restoration of the same four original containers and image IDs.
`tests/release/p4-server222-maintenance.sh` is opt-in, never a CI/default target.
It builds native images using an explicit build-only proxy and an isolated,
memory-limited BuildKit, then stops the builder before measurements. Only synthetic
media and temporary test volumes are used; the existing model cache is read-only.
The P3 environment accepts `FOLIOPATH_TEST_RESOURCE_PROFILE=p4-native` (app
1 CPU/768 MiB; AI 3 CPU/2560 MiB; no swap), `FOLIOPATH_TEST_MODEL_CACHE`, and
`FOLIOPATH_P3_AI_IMAGE` for this test. Exact-run labels permit failure cleanup
without targeting original services or volumes. This is not deployment permission.
See `docs/evidence/int-001/p4-server222-native-2026-09-16.md` for actual results;
the older local-only statements below describe their original P2/P3 runs.

## P3 recovery and current UI acceptance

Build `foliopath-app:p3` from the repository Dockerfile, the face inference image,
and `foliopath-p3-browser:local` from `browser-test.Dockerfile`. The browser image
installs Chromium, Firefox and WebKit; it is a test runner, not a deployment unit.
Build `foliopath-release-http-client:local` from `tests/fixtures/http-client/Dockerfile`
for stopped-volume copies. Then run:

```sh
node spikes/int001-remote-ml/p3-business-probe.mjs
node spikes/int001-remote-ml/p3-recovery-probe.mjs
node spikes/int001-remote-ml/p3-fault-probe.mjs
node spikes/int001-remote-ml/p3-browser-probe.mjs
```

These use loopback ports 6098, 6096, 6097, 6095 respectively, isolated temporary
synthetic fixtures/volumes/networks, and the existing model cache read-only.
Host FFmpeg is used only to prepare a synthetic three-second video; all application,
inference and browser assertions execute in local Docker. Real lease expiry can
take several minutes. Keep the final image tag stable during acceptance; each run
also holds a private image tag until cleanup. No server222 or real-media tests.
Use `FOLIOPATH_P3_APP_IMAGE=foliopath-app:p23-final` to select the final candidate.
Add `FOLIOPATH_P3_D04=1 FOLIOPATH_D04_VISUAL=1` to the browser command for
the real people/tag keyboard, mobile-touch, locale/theme/viewport matrix and
32 Chromium screenshot comparisons. Only generate references intentionally with
`FOLIOPATH_D04_UPDATE=1`; inspect them, then run again without UPDATE in a new
environment. Never count reference generation as a regression pass. Wait for the
previous browser probe to exit and clean up before reusing port 6095.
See `docs/evidence/int-001/p2-p3-final-recheck-2026-09-16.md` for final results,
retained failures, conditional skips and boundaries.

## Non-empty face business acceptance (P2)

`node spikes/int001-remote-ml/face-business-probe.mjs` creates and removes its own
temporary synthetic media tree, Docker network, database volume and containers.
It never uses server222 or a real media directory. The fixture's provenance and
full generation prompt are in `tests/fixtures/ai-faces/README.md`.
Only the model cache `/Volumes/extended/foliopath/model-cache` is reused read-only.
The app is loopback-only on 6094; the inference service has no published port.

Prerequisites: build the inference face candidate as below, then:

```sh
docker build -t foliopath-app:face-worker .
docker build --target web -t foliopath-web-check:local .
docker build -f spikes/int001-remote-ml/browser-test.Dockerfile -t foliopath-face-browser:local .
node spikes/int001-remote-ml/face-business-probe.mjs
```

The harness pins the app image ID for the whole run, including forced restart.
It drives the authenticated API and real Chromium UI, including manual review,
pagination, clearing confirmations, generation switching, cancellation and source
reconciliation. Browser screenshots go to ignored `test-results/face-real`.
Two repeated synthetic faces prove non-empty plumbing, not accuracy or calibration.
The test-only threshold profile deliberately does not enable whole-group assignment.

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
