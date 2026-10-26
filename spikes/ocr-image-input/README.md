# OCR native input probe

CR-2026-047 / ADR-0033. Synthetic-only validation of the production imagevips OCR
input adapter; not an OCR HTTP producer, background job, index or UI.

From the repository root:

```sh
docker build --target ocr-image-probe -t foliopath:ocr-input-probe-20261025 .
python3 spikes/ocr-image-input/run_probe.py \
  --image foliopath:ocr-input-probe-20261025 \
  --output "$PWD/.artifacts/ocr-input/model-bridge"
```

The runner resolves an immutable native image, verifies the three fixed fixture
hashes against the S0 record, and checks the separate prepared OCR model cache.
It requires the previously built, pinned S0 OCR experiment image and the two verified
weights in `/Volumes/extended/foliopath/ocr-feasibility-cache` (`--cache` may select
another explicitly prepared copy). It downloads nothing and uses no real media.
Preparation details are in `spikes/ocr/README.md`.

The native container gets only the fixed synthetic fixture tree read-only and its
own artifact output directory. Each model container gets only the separate OCR
cache read-only and receives the prepared bytes on stdin. All use network none,
root read-only, no public port, bounded tmpfs and 2 CPUs. Native tests/probe use
1 GiB; the separate S0 model bridge uses its explicit 4 GiB experiment budget.
Containers are removed even on errors. Prepared PNGs/logs are ignored artifacts.

The bridge checks actual recognized Chinese/English strings, not just exit codes.
It checks EXIF-normalized coordinates/dimensions and preserved fixture/model hashes.
It deliberately calls the **existing experimental S0 worker**, not a production
OCR endpoint. No Go transport/service/DB/job/search integration is claimed.
90-degree images without EXIF correction and general layout/quality remain open.

To run the native race regression, resolve the image digest first and use:

```sh
docker run --rm --network none --read-only --memory 1g --cpus 2 \
  --tmpfs /tmp:size=256m --entrypoint /out/ocr-image.test \
  foliopath:ocr-input-probe-20261025 -test.v -test.timeout=8m
```

Two existing progressive-JPEG regressions require a generated synthetic large
fixture. Generate it in a **separate disposable container**, never a real library:

```sh
mkdir -p .artifacts/ocr-input/progressive
docker run --rm --network none --read-only --memory 2g --cpus 2 \
  --tmpfs /tmp:size=128m \
  --mount "type=bind,src=$PWD/.artifacts/ocr-input/progressive,dst=/generated" \
  -e FOLIOPATH_SYNTHETIC_PROGRESSIVE_JPEG=/generated/large.jpg \
  -e FOLIOPATH_GENERATE_PROGRESSIVE=1 --entrypoint /out/ocr-image.test \
  foliopath:ocr-input-probe-20261025 \
  -test.run '^TestDecodeFaceProgressiveJPEGWorkingMemory$' -test.v
```

Then rerun the full native regression with that directory mounted read-only at
`/synthetic` and `FOLIOPATH_SYNTHETIC_PROGRESSIVE_JPEG=/synthetic/large.jpg`.
Fixture generation is not an acceptance result. The final executed digest,
counts, errors and skipped/remaining scope are recorded in
`docs/changes/CR-2026-047-ocr-image-input.md` and its machine evidence.
