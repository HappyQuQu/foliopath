# ArcFace ResNet100 normalized replacement candidate

Date: 2026-09-01

Status: **technical candidate only; production and quality Gates remain No-Go**

The exact upstream Open Model Zoo graph remains rejected because ONNX Runtime 1.28 cannot execute its legacy
`BatchNormalization(spatial=0)` declarations. A separate deterministic transform removed exactly those 154 attributes,
after first binding the source to 261,036,388 bytes and SHA-256
`f0a2e278b430372d308fef67c1aea308c2baf37f32e8908d9bfce035c26a3fb4`. It changed no tensor, initializer, node,
input/output name or other attribute. ONNX 1.22 deterministic serialization produced 261,033,924 bytes with SHA-256
`345e28fd93dc48fd7bfb3552c58434ca7e279f85ee2132c810b26945d4550844`.

## Numeric and functional result

Against OpenCV 5.0's execution of the original graph, a deterministic 512-element output comparison had maximum absolute
difference `6.736e-6`, mean absolute difference `1.675e-6`, and cosine `1.0`. This supports technical equivalence for the
single fixture; it is not cross-architecture or quality parity.

On 135 operator-authorized, read-only local images spanning nine directory groups, YuNet decoded all 135 files, detected
79 faces, and the normalized graph produced 79 finite 512-dimensional embeddings. Median/P95 embedding latency was
27.841/29.125 ms on darwin/arm64. The directory-only functional grouping produced 389 within-group and 2,692 cross-group
pairs: threshold 0.5 retained 90.49% of within-group pairs with 54 cross-group candidates; threshold 0.6 retained 77.38%
with four cross-group candidates; threshold 0.7 removed those candidates while retaining 44.47%. Paths, names, crops and
embeddings were not persisted.

Directory groups are not face-level identity labels, so these rates are not recall, FPR, ROC, bias or anonymous-core
precision. The result only proves that a deterministic compatibility transform is technically viable.

## Disposition

The source model is 261 MB versus the held 39 MB SFace candidate. Open Model Zoo identifies the exact public model as
`LResNet100E-IR,ArcFace@ms1m-refine-v2` and states that it is distributed under Apache-2.0. That repository-level statement
does not override InsightFace's current official terms, which say that its public pretrained models are available for
non-commercial research and require a separate license for commercial use. The exact MS1M-derived weight therefore remains
on compliance hold unless a separate commercial license or authoritative written clarification binds permission to this
exact artifact. Before it could replace SFace, the project would also need an accepted derived-model contract,
deterministic rebuild on both native architectures, final-model numeric/quality evidence, 100k joint capacity, SBOM/VEX/
notices/provenance and privacy/compliance/security/release signatures. None is inferred from Open Model Zoo's Apache-2.0
statement or LFW result. The production catalog therefore remains empty and `INT-250/251` remain incomplete. `INT-241`
was later closed as a fail-closed backend implementation item without admitting this candidate.

Official source records:

- [Open Model Zoo ArcFace model README](https://github.com/openvinotoolkit/open_model_zoo/blob/4d4266fbbb7eb5ab80944c2800d7f304868d573d/models/public/face-recognition-resnet100-arcface-onnx/README.md)
- [Open Model Zoo public model index](https://github.com/openvinotoolkit/open_model_zoo/blob/4d4266fbbb7eb5ab80944c2800d7f304868d573d/models/public/index.md)
- [InsightFace Python package license notice](https://github.com/deepinsight/insightface/blob/master/python-package/README.md)
- [InsightFace commercial licensing notice](https://github.com/deepinsight/insightface/blob/master/server/LICENSING.md)

Structured evidence: [JSON](arcface-resnet100-normalized-candidate-darwin-arm64-2026-09-01.json).
