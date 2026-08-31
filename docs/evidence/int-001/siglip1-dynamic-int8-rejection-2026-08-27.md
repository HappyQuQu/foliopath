# SigLIP 1 dynamic-int8 candidate rejection

Status: **rejected before capacity testing**.

## Candidate

The fixed SigLIP 1 split graphs were dynamically quantized twice with ONNX
Runtime, restricting conversion to `MatMul` and `Gemm`, QInt8 weights, no
per-channel quantization and no reduced range. Both quantization runs were
byte-identical. Total graph bytes fell from about 813 MB to about 281 MB:

| Graph | Float32 | Dynamic int8 |
| --- | ---: | ---: |
| Image | 371,682,125 B | 95,795,926 B |
| Text | 441,217,411 B | 184,782,951 B |

## Failure

The same production-govips ten-image/24-query pilot was run on macOS arm64 ORT
1.29 and native Linux/arm64 ORT 1.28. Only 8 of 24 Top-3 lists agreed across the
two runtimes. On native Linux, Chinese Recall@1 fell from the float32
candidate's 0.917 to 0.25 and Recall@3 fell from 1.0 to 0.5. Only 6 of 24 Linux
float32/int8 Top-3 lists were identical.

The resource result was attractive—Linux cgroup peak about 811 MB and image/
text P95 about 35.1/10.7 ms—but it is irrelevant once semantic behavior and
cross-runtime stability fail.

Machine-readable results are in
[`siglip1-dynamic-int8-rejection-2026-08-27.json`](siglip1-dynamic-int8-rejection-2026-08-27.json).

## Decision

This dynamic-QInt8 configuration is rejected and must not proceed to 100k
capacity testing or production selection. Different ONNX Runtime minor versions
are a confounder that reinforces the requirement to freeze one runtime, but the
observed native-target quality collapse is already sufficient to stop. Any
float16, static calibration or smaller-model alternative is a new candidate and
must repeat provenance, cross-architecture quality and capacity gates.
