# AuraFace production-boundary Linux/arm64 smoke

Status: **candidate runtime boundary evidence; not model, privacy, quality, or release approval**.

## Bound inputs

- platform: Docker Desktop Linux `arm64`
- Go build image: `golang:1.26.5-trixie@sha256:4ee9ffa999b4583ce281939cdff828763083610292f252279a0cee77473bd9a7`
- ONNX Runtime: `1.28.0`, upstream commit
  `da9b5e364c465de65c49d91e696cd6485270757f`, archive SHA-256
  `e15ff8b5d85afe6c144d97c6fd432254bf76a219daaf17658087d6ecb3e8f0bb`
- candidate: `fal/AuraFace-v1` revision
  `af6d057c9b0ec4071d4c49c80e3539258798b609`, `glintr100.onnx`, 260,694,151 bytes,
  SHA-256 `a7933ea5330113b01c9b60351d8f4c33003f145d8470ac5f0e52ee2effe25c60`
- detector: OpenCV Zoo YuNet `2023mar`, 232,589 bytes, SHA-256
  `8f2383e4dd3cfbb4553ea8718107fc0423210dc964f9f4280604804ed2552fa4`
- model mount: read-only `/models`; no model byte was copied into the repository or image
- functional image: OpenCV Zoo YuNet public example JPEG, SHA-256
  `ab8413ad9bb4f53068f4fb63c6747e5989991dd02241c923d5595b614ecf2bf6`, mounted read-only and not committed

## Result

The current `internal/inference/onnx` package compiled with CGO and the `onnxruntime` build tag against the pinned
Linux/arm64 C API. Native candidate tests opened both models through `/proc/self/fd`. YuNet validated and executed
`input float32[1,3,640,640]` plus its exact 12 stride outputs, then passed the bounded decode/NMS contract. AuraFace
validated `data float32[-1,3,112,112] -> 1333 float32[1,512]`, executed one zero-input inference, and returned a finite,
non-zero 512-dimensional output. The package's kernel-handle and manifest-role tests passed in the same process.

The current production adapter boundary then ran the public JPEG through the pinned libvips build, bounded RGB decode,
YuNet BGR tensor construction, exact 12-output execution and NMS, source-coordinate remapping, five-landmark similarity
alignment, AuraFace preprocessing, and embedding. It produced at least one candidate with a finite, non-zero
512-dimensional embedding. The source and both model mounts were read-only; no crop, path, embedding, or identity label
was persisted.

The build produced a local ephemeral test image manifest list
`sha256:90334bf1b7350690c07bcd1802757d5faeb1a04d94a8d589f3b20149c5cb1ec6`; it is not a release image or retained
artifact.

The later full-pipeline test image manifest list was
`sha256:63faaa04a45033e4dc6bdb09b9169a94715e0bc7f53051ccdd6d610f90773e7f`; it is likewise ephemeral evidence, not a
release image.

A subsequent privacy-safe comparison reran this complete arm64 pipeline and the emulated amd64 pipeline against the
same immutable inputs. Both produced three candidates and identical boxes. The arm64 0.001-quantized result fingerprint
was `8f2edf8487e117dccd5fcd036c5624dc1c2b43a0455b9038c65530192dab0f19`; it did not equal the emulated amd64
fingerprint. Direct local comparison found an overall embedding maximum absolute difference of `1.2406e-4`, no
component above `5e-4`, and per-candidate cosine similarities of at least `0.999999999449`. Full methodology and
non-claims are recorded in the
[emulated Linux/amd64 preflight](auraface-production-boundary-emulated-amd64-2026-09-01.md). Raw diagnostic vectors
were deleted after aggregate calculation and were not committed.

## Non-claims

This does not approve AuraFace/YuNet training-data provenance, license/compliance, biometric use, decode/align quality,
native amd64 behavior, real-face quality/bias, joint 100k load, production model catalog, production composition, or
S2C release. The candidate remains fail-closed and unreachable from the application.
