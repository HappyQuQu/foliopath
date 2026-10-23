# ArcFace ResNet100 replacement-candidate rejection

Date: 2026-08-31

Status: **rejected before quality or production selection**

Scope: local compatibility audit only; not face quality, privacy, compliance approval, or release evidence.

## Candidate binding

- Open Model Zoo revision: `4d4266fbbb7eb5ab80944c2800d7f304868d573d`
- Model: `face-recognition-resnet100-arcface-onnx` / `arcfaceresnet100-8.onnx`
- Bytes: `261,036,388`
- SHA-256: `f0a2e278b430372d308fef67c1aea308c2baf37f32e8908d9bfce035c26a3fb4`
- Upstream SHA-384: `4bceb34ee96c7c0b544012be53870d757f332d9d2972962065618de6942cf60f63b94f702f85fcc31844b21de326ec72`
- Declared graph contract: float32 `data [1,3,112,112]` to `fc1 [1,512]`
- Runtime under test: ONNX Runtime `1.28.0`, darwin/arm64 CPU

The downloaded bytes matched both the pinned size and Open Model Zoo SHA-384; the independently calculated SHA-256 is
recorded in the strict candidate catalog. Open Model Zoo identifies the original model as Apache-2.0 and reports LFW
accuracy `99.68%`. These upstream claims were recorded only to audit the candidate, not accepted as FolioPath quality or
compliance evidence.

## Result

The graph loaded and exposed the declared input/output metadata, but the first real aligned-face inference failed inside
its `stage1_unit1_bn1` BatchNormalization node because the stored scale tensor shape is incompatible with ONNX Runtime
`1.28.0`. No embedding was accepted or persisted. The candidate is therefore marked `rejected` before running pair,
cluster, bias, capacity, dual-architecture, or release gates.

Adapting the graph, switching to OpenVINO, or adding a compatibility rewrite introduces a new derived model/runtime
contract and repeats provenance, deterministic conversion, numeric parity, quality, capacity, SBOM and architecture
review. FolioPath did not silently promote this graph. A later isolated
[deterministic-normalization candidate](arcface-resnet100-normalized-candidate-darwin-arm64-2026-09-01.md) is tracked as a
separate unapproved artifact; it does not change this exact upstream graph's rejection or unblock the quality/release
work in `INT-250/251`. `INT-241` was later closed by the fail-closed combination-package implementation using a different
candidate boundary, not by approving this rejected graph.

## Sources

- [Pinned Open Model Zoo model metadata](https://github.com/openvinotoolkit/open_model_zoo/blob/4d4266fbbb7eb5ab80944c2800d7f304868d573d/models/public/face-recognition-resnet100-arcface-onnx/model.yml)
- [Open Model Zoo model documentation](https://github.com/openvinotoolkit/open_model_zoo/blob/4d4266fbbb7eb5ab80944c2800d7f304868d573d/models/public/face-recognition-resnet100-arcface-onnx/README.md)
- [Open Model Zoo Apache-2.0 license](https://github.com/openvinotoolkit/open_model_zoo/blob/4d4266fbbb7eb5ab80944c2800d7f304868d573d/LICENSE)
