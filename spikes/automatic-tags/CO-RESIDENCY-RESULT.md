# Five-model co-residency: failed at the current Compose budget

Date: 2026-10-22. **Result: confirmed OOM; not compatible as five permanently
resident CPU sessions at the tested 2304 MiB budget.** This is isolated feasibility
evidence, not a production change or a justification to silently raise limits.

## Exact scope

`python3 spikes/automatic-tags/run-co-residency.py` creates one disposable,
network-disabled, read-only container from the same existing local image used in
the [single-model spike](README.md). It mounts only the old model cache, new
experiment cache, spike scripts and existing synthetic PNG directory, all read-only.
No `/library`, database, socket, published port, running-service API or server222
is involved.

Limits match the checked `compose.ai.yaml`: CPU 3, memory/swap ceiling 2304 MiB,
PID ceiling 256, `/verified-cache` tmpfs 1200 MiB, `/tmp` tmpfs 128 MiB. No resource
limit was raised and no model was unloaded to produce a passing result. The inner
deadline is 140 seconds; the launcher has a 150-second attach timeout plus bounded
create/stop/inspect/remove calls (under 3 minutes).

The source model files were resolved before the test:

| Existing model file | Bytes |
| --- | ---: |
| `clip/ViT-B-32__openai/visual/model.onnx` | 351,613,724 |
| `clip/ViT-B-32__openai/textual/model.onnx` | 254,193,396 |
| `facial-recognition/buffalo_l/detection/model.onnx` | 16,923,827 |
| `facial-recognition/buffalo_l/recognition/model.onnx` | 174,383,866 |

The existing image's own `cache_snapshot.snapshot` verifies and privately copies
all 49 pinned legacy files (802,152,765 bytes), then separately verifies/copies the
3 pinned JoyTag files (366,193,236 bytes) into the **same** tmpfs budget. Combined
bytes are 1,168,346,001, below the actual 1,258,291,200-byte tmpfs capacity. The
snapshot stage succeeded; the failure is **resident memory**, not cache capacity.

All sessions use the image's canonical `immich_ml.sessions.ort.OrtSession`, not a
feature-local reimplementation of its options. Verified options: CPU provider,
`arena_extend_strategy=kSameAsRequested`, arena enabled, intra-op 2, inter-op 1,
`ORT_SEQUENTIAL`, `ORT_ENABLE_ALL`. Each session remains referenced for the entire
test. The order was CLIP image, CLIP text, face detection, face recognition, JoyTag.

## Observations

The effective run was container `fp-tag-residency-spike-07373901b90c`:
start `2026-10-22T09:58:58.887693805Z`, finish `2026-10-22T09:59:02.774997709Z`.

| Last completed stage | Current cgroup bytes | Peak cgroup bytes | Process peak RSS KiB |
| --- | ---: | ---: | ---: |
| Runtime imports / preflight | 463,171,584 | 463,171,584 | 249,668 |
| Legacy verified snapshot | 1,376,346,112 | 1,390,764,032 | 252,228 |
| Both verified snapshots | 1,705,349,120 | 1,712,291,840 | 252,228 |
| CLIP image session | 1,959,469,056 | 1,959,469,056 | 638,152 |
| CLIP text session | 2,150,383,616 | 2,237,575,168 | 953,688 |
| Face detection session | 2,166,386,688 | 2,237,575,168 | 953,688 |
| Face recognition session | 2,276,503,552 | 2,336,874,496 | 1,113,272 |
| JoyTag session construction | Process killed; no post-load measurement | — | — |

Docker inspection after exit independently reported:

```json
{"Status":"exited","Running":false,"OOMKilled":true,"ExitCode":137,"Error":""}
```

The process was killed while constructing the fifth session. **No JoyTag inference
ran in this co-resident configuration**; its previously successful isolated run is
not substituted for this failed attempt. The table's last peak is the last emitted
sample, not a claimed exact peak at the OOM event. Cgroup memory includes private
tmpfs snapshots and other container memory, not only the process RSS.

Two preliminary harness attempts exited with ordinary Python import errors
(`/usr/src` not on the script path, then upstream package import order). They were
fixed before the effective measurement; neither was OOM evidence. Their containers
were also removed. The upstream import emitted a read-only Matplotlib config
warning and fell back to `/tmp`, which is isolated tmpfs; no host config was written.

## Consequence

The current Compose cannot simply add this tagger as another permanently resident
session alongside both search encoders and both face models. Keep the production
integration gate open. A shared bounded admission policy with demonstrated
on-demand session loading/unloading or a separately justified resource-budget
change is required, followed by the same-budget regression. Do not assume Python
object deletion alone returns all ORT/native allocator memory; measure reclamation.

Passing a future residency run would still not prove simultaneous inference
workspace peaks, repeated-load stability, throughput, AMD64 behavior or label
quality. This run did not reach any inference with all five models resident.

## Verification and cleanup

- Canonical snapshot verification: passed for legacy and JoyTag data.
- Four existing model sessions: loaded successfully with observed runtime options.
- Fifth session: OOM confirmed by Docker, exit 137; correct recorded failure.
- No increase in memory/tmpfs/CPU limits, no production activation.
- Launcher regression tests **3/3 passed**, covering OOM propagation, timeout
  stop/remove and cleanup after a Docker-create timeout.
- The isolated measurement container and its tmpfs were removed after collecting
  Docker state. No original media, source model data or database was removed.
- Existing inference service remained `Up 9 days (healthy)` after the experiment.

Run the host-only launcher tests with:

```sh
PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover \
  -s spikes/automatic-tags -p test_residency_launcher.py -v
```

They mock Docker and do not rerun the OOM experiment. The production-wide tests
and any model scheduling implementation are outside this experiment.
