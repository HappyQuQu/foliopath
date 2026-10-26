# AuraFace production-boundary emulated Linux/amd64 preflight

Status: **foreign-architecture compatibility preflight only; not native amd64, model, privacy, quality, capacity, or
release evidence**.

## Bound inputs

- host: macOS `arm64`; Docker Engine 29.7.2 reports Linux `arm64`
- target image: Linux `amd64`; execution therefore used Docker Desktop foreign-architecture emulation
- builder base: `golang:1.26.5-trixie` pinned by the repository Dockerfile
- fixed-source libvips 8.16.1, GLib 2.88.3 plus the repository CVE patch, and Expat 2.8.2
- ONNX Runtime 1.28.0 x64 archive SHA-256
  `a3e1b79d7bb1bf09696ce675f49e4064e6c81f6202b8225624fff0e93f8d6407`, upstream commit
  `da9b5e364c465de65c49d91e696cd6485270757f`
- YuNet candidate: 232,589 bytes, SHA-256
  `8f2383e4dd3cfbb4553ea8718107fc0423210dc964f9f4280604804ed2552fa4`
- AuraFace `glintr100.onnx`: 260,694,151 bytes, SHA-256
  `a7933ea5330113b01c9b60351d8f4c33003f145d8470ac5f0e52ee2effe25c60`
- public JPEG fixture: SHA-256
  `ab8413ad9bb4f53068f4fb63c6747e5989991dd02241c923d5595b614ecf2bf6`

The model directory and JPEG were mounted read-only. Runtime execution used `--network none`, a read-only container
root, a bounded tmpfs, 4 CPUs, 4 GiB memory and 256 PIDs. No model or fixture byte was copied into the repository.

## Result

The current repository built its fixed libvips and application build stage for `linux/amd64`; Docker reported image ID
`sha256:4d78c641cac70f9625734d582d8096557c72db51eb8f7d898f7c8d055f42831f`. A child test image added only the
hash-verified x64 ONNX Runtime archive and compiled the current `internal/inference/faceonnx` package with
`libvips onnxruntime`; its image ID is
`sha256:732cfa35328eb517552d0dbfc84a27018451ff3364b3e13a5fc4c3c2b7aaec7e`.

Under emulation:

- `TestNativeFaceEmbeddingCandidate` passed the exact AuraFace tensor ABI and produced a finite non-zero 512D output;
- `TestNativeFaceDetectorCandidate` passed the exact YuNet 12-output ABI and bounded decode/NMS path;
- `TestNativeFacePipelineCandidate` passed the complete read-only
  `libvips decode -> YuNet -> five-point alignment -> AuraFace` path and produced at least one finite non-zero candidate.

The complete pipeline test reported 1.95 seconds and the two direct tests reported 0.89/0.04 seconds under emulation.
Those timings are recorded only to identify the run and are not performance evidence. An initial direct-test launch used
a non-executable tmpfs and failed before the Go test binary started; the corrected executable tmpfs run above passed.

### Numerical comparison with Linux/arm64 preflight

The exact same model bytes and public JPEG were then passed through the complete pipeline in the Linux/arm64 and
emulated Linux/amd64 images. Both runs produced three candidates with identical boxes. The test records a SHA-256 over
the candidate structure after rounding every scalar and embedding component to 0.001; the arm64 fingerprint was
`8f2edf8487e117dccd5fcd036c5624dc1c2b43a0455b9038c65530192dab0f19` and the emulated amd64 fingerprint was
`db29463994dad49e187fea4dcfffb388b620b710479d7ad71d2952fa0f39c1e2`. The fingerprints differ, so the run does not
claim bitwise or quantized-bin identity.

A one-time local paired comparison aligned candidates by their identical boxes and measured the unrounded results:

- detection and quality maximum absolute difference: `4.0e-8`;
- embedding dimensions: 512 for every candidate;
- per-candidate embedding maximum absolute differences: `3.3e-6`, `1.2406e-4`, and `1.2216e-4`;
- overall embedding maximum/mean absolute difference: `1.2406e-4` / `1.770134448e-5`;
- no embedding component differed by more than `5e-4`; 11 of 1,536 differed by more than `1e-4`;
- paired embedding cosine similarities: `0.999999999999`, `0.999999999449`, and `0.999999999482`.

The different fingerprints are therefore caused by at least one value crossing a 0.001 rounding-bin boundary, not by
a measured 0.001-or-larger component drift. The temporary diagnostic logs contained the public fixture's raw candidate
vectors only long enough to calculate these aggregates and were deleted; the repository test logs only candidate count
and the one-way fingerprint.

## Non-claims

This closes an amd64 compile/ABI/functional preflight gap only. The target did not run on an amd64 host, so it cannot be
fed to `verify-intelligent-media-native-model-evidence` and does not change any checklist numerator or Gate. Native
Linux/amd64 and arm64 must still run the same final package/source SHA and produce strict paired evidence, including
quality/ranking decisions, cancellation, strong-kill recovery, leak/RSS and joint capacity. AuraFace/YuNet provenance,
governed biometric quality, supply chain and owner signatures also remain open; production composition stays empty.
