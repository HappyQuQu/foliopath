# Pinned SigLIP 2 ONNX self-export and arm64 runtime evidence

Status: **preliminary development evidence; INT-S0 remains No-Go**.

## Question and boundary

The source review rejected the unmerged ONNX artifact in the Google repository
and the third-party ONNX Community conversion as production download origins.
This run tests the remaining route: export from the reviewed Google source
revision using a fully pinned toolchain, then compare every ONNX output with the
pinned PyTorch model. It does not approve distribution, a production adapter or
semantic quality.

The model, ONNX output, tokenizer and deterministic reference fixture lived
under `/private/tmp/foliopath-int006-onnx.6JK9QB`. None was added to Git and no
user media was read.

## Fixed source and toolchain

- Source: `google/siglip2-base-patch16-224`
- Revision: `75de2d55ec2d0b4efc50b3e9ad70dba96a7b2fa2`
- Source `model.safetensors`: `1,500,800,904` bytes,
  SHA-256 `612923381c76ec5a9bed335d1c48827e3f2e506ac31b044b63b2031fadee6a0b`
- Export: Python 3.12, PyTorch 2.13.0, Transformers 4.57.6,
  Optimum 2.1.0, Optimum ONNX 0.1.0, Accelerate 1.14.0, ONNX 1.20.1,
  ONNX Runtime 1.29.0, opset 18, CPU, validation enabled, `atol=1e-4`
- Fixed contract: 224 × 224 RGB input and maximum text length 64

The reproducible entry points are
[`semantic_onnx_export.py`](../../../spikes/int001-ai/semantic_onnx_export.py),
[`onnx_compare.py`](../../../spikes/int001-ai/onnx_compare.py) and the pinned
[`requirements`](../../../spikes/int001-ai/requirements-semantic-onnx-export.txt).

## Results

Three exports—including a final run with Accelerate's weight-deduplication
check available—were byte-identical:

| Artifact | Result |
| --- | ---: |
| ONNX | 1,501,208,026 bytes |
| ONNX SHA-256 | `18a16d73759d3760a664596660e5bb8f4800635bbec39775bafe88a85cf57226` |
| Reference NPZ SHA-256 | `020887ad3ef9397e9e3a9c0520600c2564abcf8e3b5fca7bf17a5a8bf160a137` |
| Graph | IR 8, opset 18, 2,168 nodes, 408 initializers |
| Outputs | image/text logits plus 768-dimension image/text embeddings |

ONNX structure validation passed. On macOS arm64, ORT 1.29.0 produced finite
outputs and all four outputs matched PyTorch at `atol=rtol=1e-4`; the largest
absolute error was `9.536743e-6`.

The same immutable ONNX and reference fixture then ran three times in native
Linux/arm64 with 4 CPUs, a 4 GiB limit and `--network none`. The existing base
image digest was
`sha256:3686d08675dc8f5a20a34635d30ad90cbe81e8a51a1836dae98526752fd73502`.
ORT 1.28.0 was installed from locally hashed wheels before the offline runs.

| Run | Load | Inference | Peak RSS | Max absolute error |
| ---: | ---: | ---: | ---: | ---: |
| 1 | 0.925 s | 0.144 s | 2,295,264 KiB | `1.049042e-5` |
| 2 | 0.858 s | 0.144 s | 2,295,720 KiB | `1.049042e-5` |
| 3 | 0.847 s | 0.144 s | 2,295,292 KiB | `1.049042e-5` |

Every output was finite and within `atol=rtol=1e-4` in all runs. Machine-readable
details are in
[`semantic-onnx-export-arm64-2026-08-26.json`](semantic-onnx-export-arm64-2026-08-26.json).

The isolated Python runtime harness then ran another 100 fixed-shape inferences
per fresh process, rejected missing input, wrong input dtype, four-channel image
and length-65 text, terminated an in-flight larger-batch request, and confirmed
that a normal inference still worked afterward. macOS arm64 passed once. Three
Linux/arm64/no-network runs passed with RSS growth of 9,740～10,508 KiB after
warm-up, below the provisional 65,536 KiB smoke threshold. This is bounded
allocator behavior over 100 calls, not a long-duration leak proof.

The exported graph was also evaluated on the ten-image/24-query public pilot,
using WebPs produced by FolioPath's production govips adapter under native
Linux/arm64. The tensor bundle was bound to the fixture and held outside Git.
macOS arm64 ORT 1.29.0 and native Linux/arm64 ORT 1.28.0 produced byte-identical
Top-3 result records for both float32 and float16-stored image embeddings.
Chinese/English Recall@1 stayed 1.0; mean relevant Recall@3 stayed 0.9722/0.9306,
and float16 preserved all 24 Top-3 lists. macOS/Linux image P95 was 38.0/128.1 ms;
the 24-text batch was 192.1/763.1 ms.

This is real model output on licensed pilot images, but ten easy images cannot
approve float16 Recall. It also exposed a design defect in the first export:
the monolithic graph executes both encoders on every image and text call. A
split image/text export must be compared before selecting a production layout.

## Fixed-shape split export

A second isolated exporter retained only `vision_model` or `text_model` and
froze the actual product shapes instead of advertising untested dynamic axes.
Two fresh exports were byte-identical:

| Graph | Bytes | SHA-256 | Max error vs PyTorch |
| --- | ---: | --- | ---: |
| image encoder | 371,682,125 | `7fc85e0e8a0f4e5fce7be45c7830e7473c81f010633b7d31bc1613dafc734ab0` | `5.275011e-6` |
| text encoder | 1,129,345,413 | `ef9451f51568152758e53ebca85cec4d84d2462c45e1611b42f49a42bd1be953` | `3.814697e-6` |

The same production-input pilot ran three times on Linux/arm64. Image P95 was
90.2～97.5 ms, single-text P95 28.9～37.0 ms, and every float32/float16 Top-3
record remained identical. After the image session, RSS was 584～611 MiB; after
closing it, about 527～530 MiB remained. The subsequent text session reached
1,421～1,433 MiB current RSS and a 2,027～2,031 MiB process high-water mark.

Against one same-shape monolithic run, split image P95 was about 20.7% lower and
split text P95 about 78.1% lower, with the same rank-result digest. This is enough
to reject the monolithic graph as the final layout and retain the fixed split
graph for the next capacity experiment. It does not decide whether one process
can unload/schedule the two sessions safely during concurrent search/backfill.

That next unload/load experiment did **not** pass. In a fresh constrained
Linux/arm64 process, 30 cycles each loaded the image session, inferred, closed
it, loaded the text session, inferred and closed it. Outputs stayed finite, but
RSS after a complete cycle rose from 537,752 KiB in cycle 1 to 758,944 KiB in
cycle 10 and 750,756 KiB in cycle 30. The 213,004 KiB final increase exceeded
the provisional 131,072 KiB limit. Peak RSS reached 2,305,768 KiB.

The series jumped after the early cycles and then fluctuated around a higher
plateau, so this run does not prove an unbounded leak. It does prove that
closing ORT sessions does not restore the first-cycle memory baseline. A
production scheduler may not budget image and text sessions as if their memory
were perfectly released, and the current split lifecycle fails its S0 smoke
threshold.

Follow-up on 2026-08-27 preserved that failure and changed the runtime setting,
not the threshold. Disabling the ORT CPU memory arena made all 30 post-cycle RSS
samples remain at 509,360 KiB and reduced peak RSS to 2,046,084 KiB, while the
three-run public pilot kept rankings and latency. See the
[allocator comparison](semantic-onnx-arena-linux-arm64-2026-08-27.md). This
setting is mandatory for subsequent arm64 spikes but is not yet a production
runtime decision.

## Decision and remaining blockers

Self-export from the pinned Google revision is technically viable on arm64 and
removes the need to trust a third-party converted weight. It is the only SigLIP
2 ONNX provenance route retained for the next experiment.

This is not a model/runtime selection:

- Linux used ORT 1.28.0 because the queried index did not provide an ORT 1.29.0
  CPython 3.13 arm64 wheel. A single production runtime version is not frozen.
- The exporter emitted trace warnings around image-size interpolation and text
  sequence branching. Only the fixed 224 × 224 / length-64 contract is approved;
  advertised dynamic dimensions must not be treated as validated.
- Roughly 2.19 GiB peak RSS for one semantic session leaves too little evidence
  for the complete application, face runtime, backfill, browse and cache load.
- The monolithic graph performs unnecessary opposite-branch inference and is
  rejected as the production candidate. The split graph is only the next spike
  candidate, not an accepted runtime contract.
- The default CPU-arena lifecycle failed its retained-RSS limit. Disabling the
  arena passed the same limit, but the configuration still needs production
  adapter enforcement and full-process remeasurement.
- Random tensors prove numerical compatibility, not Recall. Float16 storage
  still requires real exported embeddings and the representative quality set.
- Native Linux/amd64, C/Go adapter cancellation, hostile model files,
  repeated-load and long-duration leak tests, SBOM, vulnerability review,
  notices, redistribution approval and project signing remain mandatory.

Therefore `INT-006E1B` may be recorded as an arm64 self-export subproof, while
`INT-006`, `INT-006E2`, `INT-008`, `INT-013`, `INT-014` and INT-S0 remain open.
