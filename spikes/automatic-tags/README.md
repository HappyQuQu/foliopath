# Automatic image tags — isolated JoyTag feasibility spike

The later localized photo-tag candidate uses `Dockerfile.localized-app` here only
to reuse a pinned local native builder/runtime while compiling the current complete
application and production frontend. It is not the canonical release/supply-chain
build. The deployed inference overlay source is under `services/inference`, not
this spike. See [CR-2026-033](../../docs/changes/CR-2026-033-localized-photo-tags.md)
for immutable image IDs, regression results and the separate 6010 deployment status.

This is the **historical standalone spike**. The later ADR-0021 production adapter
and its isolated same-budget validation are recorded in
[SCHEDULED-RESULT.md](SCHEDULED-RESULT.md). The failure of permanent co-residency
below remains valid; the new implementation uses request-scoped processes and
private snapshots instead, without raising the budget. Neither report is a
general photo-quality or release approval.

Status: **real inference smoke passed; no production activation, accuracy gate, or
release approval**. Recorded 2026-10-22. Owner: AI tagging research slice requested
by the user; parent change record owns production scope/contracts. This directory
is outside the production import graph and is not a new deployment unit.

Follow-up: the [same-budget five-model co-residency test](CO-RESIDENCY-RESULT.md)
**failed with a Docker-confirmed OOM** while loading JoyTag. Do not infer that the
single-model success means the current Compose can host all models together.

## What this establishes

The current application has CLIP candidate-label scoring, not an image tagger.
This experiment instead runs the author's trained JoyTag multi-label head on the
image. Its 5813 internal classes do not require a user to supply vocabulary or
review suggestions. The spike's allowlist only filters classifier output; it does
not generate text embeddings or rank CLIP candidate phrases.

The experiment does **not** add jobs, automatic tag persistence, UI, translations,
manual-removal suppression, model activation, or a production inference endpoint.

## Provenance and files

- Author repository: <https://github.com/fpgaminer/joytag>.
- Model: <https://huggingface.co/fancyfeast/joytag>.
- Immutable revision: `6b7f16331a6ccf0fdce37d5a9564715f6e772b22`.
- Author model card labels it Apache-2.0. License/notice packaging and the chosen
  distribution policy still need review before shipping weights.
- ONNX: 366,116,154 bytes; SHA256
  `f85b7130e6e549b5b0822537007b7482e8c4c8e754c8d9a5bee08e27050e1097`.
- `config.json`: 330 bytes; SHA256
  `b53ee29a8f0d6353331f455fd640c70ef45e5bd979b656f7b9983cffa3724562`.
- `top_tags.txt`: 76,752 bytes; SHA256
  `32b1963a234af848643b2bbf47d8eff1f1c7889406810c57b980f41b2b9e01d0`.

`model-manifest.json` also records the small files' Git blob identities returned
by the author's fixed-revision repository. All three sizes and SHA256 hashes, and
both small-file Git hashes, were checked after download and on the idempotent
second fetch. The inference script independently rechecks sizes and SHA256 hashes.
102 GiB was available on the destination volume before the 350 MiB download.

Weights exist only at
`/Volumes/extended/foliopath/model-experiments/joytag`; do not commit them. The
existing `/Volumes/extended/foliopath/model-cache` was not changed. No server222
connection, original-media access, production API call, or activation occurred.

## Reproduction and isolation

From repository root:

```sh
bash spikes/automatic-tags/fetch-model.sh
bash -n spikes/automatic-tags/fetch-model.sh spikes/automatic-tags/run-probe.sh
bash spikes/automatic-tags/run-probe.sh
```

The fetch script explicitly downloads from the fixed author revision over HTTPS,
ignores ambient proxies, verifies the complete bytes, and refuses to overwrite an
existing mismatched file. Partial/untrusted downloads are retained with temporary
names, never installed or executed. It never fetches or executes remote Python.

The run script reuses this exact already-present local inference image:
`sha256:31484348fd97a6cfefd01a83dfe3104f22ba9e4d23e92c1e1000d62774c30689`.
It uses a separate disposable `--network none --read-only` container with no
ports, dropped capabilities, 2 CPU, 2304 MiB RAM/swap ceiling, and 128 PID limit.
Only the isolated model, spike script, and three synthetic fixture directories
are mounted, all read-only. Neither a `/library` nor a database/cache mount exists.
The original running AI container remains unchanged.

The model is **not** imported through Hugging Face `trust_remote_code`. The only
inference dependencies are the existing image's NumPy, Pillow, and ONNX Runtime.
The image is Linux ARM64 with Python 3.11.15 and ONNX Runtime 1.26.0; the execution
provider was explicitly `CPUExecutionProvider`. AMD64 and GPU have not been tested.

The pinned ONNX graph was separately inspected with the image's existing `onnx`
module: producer `pytorch 2.2.1`, default-domain opset 17, one float input
`[batch, 3, 448, 448]`, and one output `[batch, 5813]`. Its final operation is
`Gemm`, not `Sigmoid`. The probe applies sigmoid, matching the author's example.
Preprocessing follows the author's white square padding, bicubic resize, RGB
float32 normalization and NCHW layout.

## Observed run

11 pre-existing repository synthetic PNGs, three measured runs each, plus one
warm-up inference. The 9 `ai-media` images are documented as fictional adults in
`prototypes/apple-redesign/design-qa.md`; the two-person fixture has a source
statement under `tests/fixtures/ai-faces/README.md`. No real personal library was
used. The 11th image is the viewer's synthetic blue/violet fixture.

| Measurement | Observed |
| --- | ---: |
| Cold ONNX session load | 326.774 ms |
| First inference | 846.496 ms |
| Warm inference p50, 33 runs | 827.247 ms |
| Warm inference p95, 33 runs | 859.053 ms |
| Process peak RSS | 662,784 KiB (647.25 MiB) |
| Container cgroup memory peak | 986,329,088 bytes (about 940.6 MiB) |
| Container memory limit | 2,415,919,104 bytes (2304 MiB) |
| Intra-op / inter-op threads | 2 / 1 |

RSS and cgroup peak are different metrics; the latter includes container-accounted
memory such as file cache, not just process resident anonymous memory. These are
one warm-cache local Docker run, not a capacity or latency SLA.

Output threshold was the author's example value `0.4`, **not calibrated for
FolioPath**. The probe prints at most 10 labels from 92 explicitly allowed common
objects/scenes/actions/background colors. Sensitive/sexual body terms, inferred
traits and character identities are not emitted or saved. No label is forced if
all eligible scores are below the threshold. Model scores are not percentages of
real-world correctness.

| Synthetic fixture | Observed allowed output (descending score) |
| --- | --- |
| azure-01 | flower 0.577398; dress 0.478499; window 0.440335 |
| azure-02 | flower 0.588887; dress 0.571935 |
| azure-03 | flower 0.657968 |
| crimson-01 | dress 0.573314; indoors 0.517760 |
| crimson-02 | dress 0.634324; sitting 0.538439; indoors 0.469729 |
| crimson-03 | dress 0.757660; indoors 0.600609; blurry_background 0.487092; depth_of_field 0.428410; standing 0.413172; flower 0.409087 |
| forest-01 | standing 0.409561 |
| forest-02 | blurry_background 0.599450; tree 0.582960; outdoors 0.533351; depth_of_field 0.458156; dress 0.431053; standing 0.400249 |
| forest-03 | sitting 0.553433; outdoors 0.542107; depth_of_field 0.466228; day 0.450447; forest 0.450132; tree 0.449198 |
| synthetic-pair | shirt 0.814641; grey_background 0.599610 |
| viewer-blue-violet | monochrome 0.533975; purple_background 0.533004; blue_background 0.469650 |

The script emits each fixture's SHA256 and every measured duration on stdout for
reproduction. These outputs are model observations, **not verified ground truth**.
For example, an image with two colors receiving `monochrome` illustrates why
successful tensor output is not an accuracy claim.

## What remains unproven

1. Broad photo quality: the existing fixtures are portrait-heavy; there is no
   representative vehicle/animal/landscape/interior accuracy set. Do not call
   the user's “car” problem solved by this experiment.
2. The model's Danbooru-derived taxonomy contains many non-general-purpose labels.
   A production output policy, audited class mapping, localized labels and suitable
   calibration need explicit ownership; this 92-label spike filter is not that
   policy or a replacement for quality evaluation.
3. No CLIP or face session is loaded concurrently, and no production tmpfs snapshot
   is copied in this single-model run. The follow-up co-residency probe under the
   current 2304 MiB AI ceiling and 1200 MiB snapshot ceiling failed with OOM. A
   bounded loading/unloading policy or justified budget change remains required;
   do not simply sum only RSS or claim existing Compose memory is sufficient.
4. Automatic source/version-aware persistence, user removal suppression, restart/
   cancel recovery, bounded queue ownership, per-library coverage and backend/API
   contracts remain implementation work outside this spike.
5. The current production identity hashes the complete wrapper and cache manifest;
   adding a tagger needs deliberate compatibility/migration handling, not reuse of
   old embedding identity or an unannounced full index rebuild.

## Executed checks

- `bash -n` for both shell scripts: passed.
- Idempotent download verification: passed for all three model data files.
- Independent pinned graph inspection: input/output/opset/last operation confirmed.
- Actual CPU inference: all 34 calls completed; each fixture's final output was
  checked for the expected finite tensor shape.
- `test_probe.py`: **5/5 passed** in an isolated image container, covering output
  filtering, no forced below-threshold label, invalid output rejection, corrupt/
  symlink model rejection, and image tensor preprocessing.
- `git diff --no-index --check /dev/null <file>` for all six new files: passed.

Unit-test command (the temporary directory is memory-only):

```sh
docker run --rm --pull never --network none --read-only --cap-drop ALL \
  --security-opt no-new-privileges --memory 512m --memory-swap 512m \
  --cpus 1 --pids-limit 64 --tmpfs /tmp:rw,noexec,nosuid,size=16m \
  --env PYTHONDONTWRITEBYTECODE=1 \
  --mount type=bind,src="$PWD/spikes/automatic-tags",dst=/probe,readonly \
  --workdir /probe --entrypoint python \
  sha256:31484348fd97a6cfefd01a83dfe3104f22ba9e4d23e92c1e1000d62774c30689 \
  -m unittest -v test_probe
```

No repository-wide production test, browser test, model accuracy evaluation,
concurrent-model pressure test, or release gate is claimed by this report.
