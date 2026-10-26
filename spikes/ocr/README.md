# OCR S0 isolated experiment

CR-2026-045 / POST-MVP-5 follow-up, candidate FR-OCR-001–003. This directory
is outside the production import graph. It provides no HTTP endpoint, database,
index, capability registration, UI setting or 6010 deployment.

The immutable base is the reviewed local photo-tags image pinned in `Dockerfile`.
It retains Immich 3.1.0 / RapidOCR 3.8.1. Only the two fixed PP-OCRv5_server
ONNX files in `model-manifest.json` are used. Existing CLIP/face/tag caches and
identities are untouched. `visual` is solely the existing scheduler's injected
test-command transport; it does not declare OCR to be a semantic embedding.

## Reproduce locally

Run from the repository root. Preparation explicitly selects a proxy or direct
mode, never inherits ambient proxy settings. No direct fallback occurs.

```sh
python3 spikes/ocr/fetch_models.py /absolute/separate/ocr-cache \
  --proxy http://192.168.2.231:7890
docker build -f spikes/ocr/Dockerfile -t foliopath-inference:ocr-feasibility-20261025 .
```

The checked-in `tests/fixtures/ocr` contains synthetic images used in the record.
To regenerate, provide a locally available CJK font **read-only**, without committing or
redistributing it. This example assumes `.artifacts/ocr/font.ttc` was supplied
locally. Fixture text is synthetic; no library file is read.

```sh
docker run --rm --network none --read-only --tmpfs /tmp \
  --mount "type=bind,src=$PWD/.artifacts/ocr,dst=/output" \
  --entrypoint python foliopath-inference:ocr-feasibility-20261025 \
  /opt/ocr-spike/create_fixtures.py /output/font.ttc /output/fixtures
python3 -m unittest discover -s spikes/ocr -p 'test_contract.py'
python3 -m unittest discover -s spikes/ocr -p 'test_fetch_models.py'
docker run --rm --network none --read-only --tmpfs /tmp --entrypoint python \
  foliopath-inference:ocr-feasibility-20261025 \
  -m unittest discover -s /opt/ocr-spike -p 'test_*.py'
docker run --rm --network none --read-only --tmpfs /tmp --entrypoint python \
  foliopath-inference:ocr-feasibility-20261025 \
  -m unittest test_scheduler test_runtime_manifest
python3 spikes/ocr/run_probe.py --cache /absolute/separate/ocr-cache \
  --fixtures "$PWD/tests/fixtures/ocr" --output "$PWD/.artifacts/ocr/probe"
```

The runner resolves the immutable candidate digest, checks Docker network mode,
ports and read-only mount inventory, and removes its exact test container. It
sets an isolated **2 CPU / 4 GiB test budget**, not a production memory change.
Private verified model snapshots reside only in disposable `/work` tmpfs; no
media or database mount, public port or network access is permitted. Logs and
results contain only synthetic text. The downloaded cache is preparation state,
not an activated or redistribution-approved model package.

## Candidate bounds and interpretation

- Input: JPEG/PNG/WebP/GIF first frame, 8 MiB / 4 Mi pixels, minimum edge 32;
  EXIF transpose, RGB and longest edge 2048. This is an experiment with PIL, not
  the production `internal/files` / `imagevips` adapter or its identity.
- Detector: upstream resolution parameter 736 (short-edge behavior), score .5;
  reject more than 128 raw contours **before** truncation and more than 128 boxes.
  Reject total perspective crop area above 16 Mi pixels before allocating crops.
- Recognition: one crop per batch, score .8. Normalized convex quadrilateral,
  finite scores, NFC plaintext, 512 characters per line / 8192 total. Reject the
  whole malformed output; empty text results are distinct from input failures.
- Reading order: deterministic horizontal rows then left-to-right; box scores
  follow their original detections. No vertical/multicolumn/table layout claim.
- Existing scheduler: one child, at most two admitted requests, 26-second total
  deadline, reap and remove private snapshot before releasing admission. Probe
  cancels after observing native memory and verifies recovery; .25-second fault
  checks startup/cache deadline, not a mid-inference timing benchmark.
- Fixed-model faults produce `model_unavailable`; bad source `invalid_media`;
  input budget `input_limit`; malformed output `output_invalid`; other errors
  `inference_failed`. No raw exceptions become a public contract.

The rotated observation deliberately has no accuracy assertion. The recorded
90-degree image produced zero matching lines, so rotation recovery and quality
calibration remain open. Horizontal synthetic success is not general OCR
accuracy, joint CLIP/face/tag capacity, native amd64, or release approval.

See [change record](../../docs/changes/CR-2026-045-ocr-feasibility.md) and its
machine evidence for the executed digest, timings and limitations. A production
slice still requires accepted architecture/contracts, generation/source-binding,
job/retry/clear semantics, persistence/search and product acceptance.

Sources: fixed [Immich detector](https://github.com/immich-app/immich/blob/v3.1.0/machine-learning/immich_ml/models/ocr/detection.py),
[recognizer](https://github.com/immich-app/immich/blob/v3.1.0/machine-learning/immich_ml/models/ocr/recognition.py),
[RapidOCR metadata](https://github.com/RapidAI/RapidOCR/blob/v3.8.1/python/rapidocr/default_models.yaml).
Upstream license references alone do not approve model redistribution.
