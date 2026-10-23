# Smaller SigLIP 1 split and combined-load evidence on Linux/arm64

Status: **resource candidate passed arm64 proxy; quality and browse gates remain
open; no model selected**.

## Reproducible split export

The previously pinned `google/siglip-base-patch16-224` revision and
812,672,320-byte source weight were exported twice with the same fixed
toolchain, opset 18 and fixed shapes. Both image graphs and both text graphs
were byte-identical. The image graph is 371,682,125 bytes and the text graph is
441,217,411 bytes; both matched PyTorch within `1e-4`.

The same ten production-govips public images and 24 Chinese/English queries
were prepared with the SigLIP 1 processor. macOS arm64 ORT 1.29 and native
Linux/arm64 ORT 1.28 produced identical Top-3 lists, and float16 image storage
preserved all 24 Top-3 lists. Chinese Recall@1 remained 0.917 while English was
1.0. The known Chinese miss prevents treating the smaller model as a quality
winner on even this small pilot.

## Resident and combined capacity

A two-session resident test alternated image/text inference for 100 cycles.
Cgroup memory stabilized by cycle 30 and was unchanged at cycle 100; peak was
2,181,382,144 bytes, below the 3.2 GiB threshold. This contrasts with the
current SigLIP 2 dual-resident peak of 4,008,951,808 bytes.

Three fresh 4 CPU/4 GiB/no-network containers then combined both resident
sessions with the 100,000 × 512 float16 SQLite backfill/search/browse/cancel/
restart proxy:

| Run | Container peak | Image/Text P95 | Search P95 | Browse degradation | Rows cancel → restart |
| --- | ---: | ---: | ---: | ---: | ---: |
| 1 | 2,363,904,000 B | 176.6/56.4 ms | 169.0 ms | 8.70× | 62,720 → 100,000 |
| 2 | 2,363,527,168 B | 176.8/56.0 ms | 171.0 ms | 4.76× | 62,208 → 100,000 |
| 3 | 2,369,597,440 B | 176.1/56.1 ms | 163.0 ms | 7.93× | 62,208 → 100,000 |

All outputs were finite, no run OOMed, exact-search P95 stayed below the
provisional 750 ms limit, and every restart converged to 100,000 rows. The
unchanged 20% relative browse-degradation gate failed in all runs, despite
sub-millisecond absolute proxy latency.

Machine-readable results are in
[`siglip1-split-combined-linux-arm64-2026-08-27.json`](siglip1-split-combined-linux-arm64-2026-08-27.json).

## Decision

SigLIP 1 replaces SigLIP 2 as the resource-priority candidate for the next
quality and native-platform experiments. It is not selected for production:
the pilot is only ten images, the known Chinese miss remains, native amd64 and
representative 1,000-image/100-video evidence are absent, and the browse proxy
still fails. `INT-006`, `INT-008`, `INT-013`, `R-024` and INT-S0 remain open.
