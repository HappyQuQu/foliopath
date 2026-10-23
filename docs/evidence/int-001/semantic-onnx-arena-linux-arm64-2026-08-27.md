# ONNX split-session allocator comparison on Linux/arm64

Status: **30-cycle allocator subproof passed; later 100-cycle cgroup capacity
follow-up failed; INT-S0 remains No-Go**.

## Why this follow-up exists

The initial 30-cycle image/text session-switch test retained 213,004 KiB after
cycle 1 and failed the unchanged 131,072 KiB limit. The curve plateaued, but
session close did not restore the first-cycle baseline. This follow-up tests an
ONNX Runtime configuration change instead of weakening the gate.

## Controlled comparison

Both runs used the same immutable split models, prepared public-pilot tensor,
native Linux/arm64 container, 4 CPUs, 4 GiB and no network. Each cycle loaded
the image encoder, ran one inference, closed it, loaded the text encoder, ran
one inference and closed it.

| Configuration | Cycle 1 RSS | Cycle 30 RSS | Growth | Peak RSS | Result |
| --- | ---: | ---: | ---: | ---: | --- |
| CPU arena enabled | 537,752 KiB | 750,756 KiB | 213,004 KiB | 2,305,768 KiB | Fail |
| CPU arena disabled | 509,360 KiB | 509,360 KiB | 0 KiB | 2,046,084 KiB | Pass |

Memory pattern remained enabled. With CPU arena disabled, image load took
0.198～0.237 s, text load 0.479～0.516 s, image inference 0.089～0.103 s and text
inference 0.029～0.042 s. The original threshold was not changed.

The production-input public pilot was then repeated in three fresh processes
with the same setting. Image P95 was 92.8～99.6 ms and single-query text P95
29.8～32.9 ms. All 24 float16 Top-3 lists and quality metrics remained identical
to the arena-enabled runs. Peak RSS was 2,036,696～2,065,276 KiB.

Machine-readable results are in
[`semantic-onnx-arena-linux-arm64-2026-08-27.json`](semantic-onnx-arena-linux-arm64-2026-08-27.json).

## Decision

For the next arm64 spike, `enable_cpu_mem_arena=false` and
`enable_mem_pattern=true` are mandatory. The default arena-enabled lifecycle
remains a recorded failure and must not return by accident. The 30-cycle
retained-process-RSS subcheck is no longer blocked.

A later 100-cycle follow-up preserved the near-zero retained RSS result but
measured a 3,719,651,328-byte container peak, above the unchanged 3.2 GiB
`R-024` threshold. Therefore repeated full-model reload is not an accepted
lifecycle strategy. See
[`semantic-onnx-arena-100-cycle-linux-arm64-2026-08-27.md`](semantic-onnx-arena-100-cycle-linux-arm64-2026-08-27.md).

This does not select the production runtime or pass the broader capacity gate. Native amd64, one frozen runtime
version, Go/C adapter behavior, long-duration switching, hostile graphs and the
complete FolioPath browse/backfill/face workload remain open. Therefore
`INT-006`, `INT-008`, `INT-013` and INT-S0 remain unchecked.
