# Candidate runtime/model compliance review

Status: **partial technical record; `INT-014` remains blocked and no candidate
is approved for redistribution**.

This is an engineering evidence record, not legal advice or a legal sign-off.
It distinguishes source-code licensing, a repository's statement about model
files, training-data provenance, notices, SBOM generation and vulnerability
scanning. A permissive label on a model page does not by itself settle every
right or obligation associated with a trained weight.

## Current disposition

| Component | Pinned evidence | Engineering disposition |
| --- | --- | --- |
| ONNX Runtime CPU 1.28.0 Linux/aarch64 | Official archive SHA-256 `e15ff8b5d85afe6c144d97c6fd432254bf76a219daaf17658087d6ecb3e8f0bb`; commit `da9b5e364c465de65c49d91e696cd6485270757f`; MIT `LICENSE` SHA-256 `2f07c72751aed99790b8a4869cf2311df85a860b22ded05fa22803587a48922c`; 6,121-line `ThirdPartyNotices.txt` SHA-256 `0e07b95f3a8d6230037707c5c4a2b554d12c4cb67369669ac255635528ffcee2` | Keep for spike only. License and notices are captured, but final-image SBOM, dependency closure, vulnerability report and legal notice packaging are incomplete. |
| Google SigLIP base patch16 224 | Snapshot `7fd15f0689c79d79e38b1c2e2e2370a7bf2761ed`; source weight SHA-256 `2c63cb7d1f2e95ba501893cbb8faeb4ea9a3af295498d35097126228659c2af8`; model page labels Apache-2.0; FolioPath exports its own fixed graphs | Resource-priority semantic candidate only. Preserve source revision, model-card/license evidence, exporter SBOM, source and derived hashes. Compliance owner must still approve redistribution and notices. |
| OpenCV Zoo YuNet 2023mar | Repository `47534e27c9851bb1128ccc0102f1145e27f23f98`; weight SHA-256 `8f2383e4dd3cfbb4553ea8718107fc0423210dc964f9f4280604804ed2552fa4`; model directory contains an MIT license | Face-detector candidate only. Exact weight/license packaging and compliance sign-off remain required. |
| OpenCV Zoo SFace 2021dec | Same repository revision; weight SHA-256 `0ba9fbfa01b5270c96627c4ef784da859931e02f04419c829e83484087c34e79`; directory README says all files there are Apache-2.0 | **Hold for production.** The exact weight's training-data provenance and commercial/redistribution implications are not documented to the level required by this Gate. A public upstream clarification request for this exact hash remains open with no maintainer answer captured. |
| `coder/hnsw` | Pinned source is CC0-1.0 | Already rejected by the technical capacity spike. It must not be added merely because its code license is permissive. The current vector candidate is SQLite plus application-owned exact search, so there is no additional production vector-engine artifact yet. |

## Vulnerability/SBOM attempt

Docker Scout 1.24.0 was present locally and was invoked against the extracted
official ONNX Runtime directory. It stopped before analysis because the service
requires a Docker login. No credentials were supplied and no report was
produced.

An independent follow-up downloaded the official Grype 0.116.1 Darwin/arm64
release archive and verified SHA-256
`f493f169cbaae48bade169532b20235fc16653d2a044a5bc6fe6f69a3923f975`.
Using vulnerability DB schema v6.1.9 built 2026-08-26, it returned zero matches
but also generated a CycloneDX document with **zero identified components** for
the extracted C/C++ package. The JSON and CycloneDX report SHA-256 values are
`9ebd0067aa568cb73ffb616f5c5e497bcf1bcec2fb5e56650d8f2092c761bc0b`
and `1e24c0bf55425cf2facaa8215f8c6dcdd709fa61b807badab16dff454b1a7086`.
Zero matches with zero inventory is inconclusive, not a clean scan.

Therefore neither attempt is a passed vulnerability scan. The final release
path still needs a reproducible scanner that inventories the final image and
native dependency closure, a machine-readable SBOM, severity policy,
exception/VEX owner and both final architectures.

The archive's own `LICENSE`, `ThirdPartyNotices.txt` and `Privacy.md` were
hash-recorded, but copying those files is not a substitute for reviewing the
third-party notices or generating a final-image SBOM.

A supplemental
[`model-candidates-component.cdx.json`](../../../spikes/int001-ai/model-candidates-component.cdx.json)
now inventories five exact machine-learning components: the retained SigLIP 1
source weight, its deterministic float16 image/text ONNX graphs, YuNet and the
held SFace weight. It binds source revisions and artifact/package hashes and
passed structural validation against the official CycloneDX 1.6 JSON Schema.
Its SHA-256 is
`f77bafc9a08e08a20b98ef450adf7d7d7ed310b8ce49615d024d8d13ac893bd2`.
The BOM intentionally declares an incomplete composition. Its disposition
properties keep SigLIP 1/YuNet at compliance-signoff-pending and SFace at
production-hold; recording an upstream license declaration does not approve
weight redistribution or training provenance.

## Gate consequence

`INT-014` cannot be checked. In particular, SFace must not appear in an
"approved" signed catalog, bundled image or project-operated mirror until the
compliance owner accepts documented provenance and redistribution terms. If
that cannot be obtained, the honest fallback is to select another reviewed
face-embedding model or remove/defer the face slice. Semantic search may only
advance independently if an explicit scope/Gate change approves that split;
the current combined `INT-S0` does not silently become Go.

Primary records reviewed:

- [ONNX Runtime v1.28.0 release](https://github.com/microsoft/onnxruntime/releases/tag/v1.28.0)
- [ONNX Runtime license](https://github.com/microsoft/onnxruntime/blob/v1.28.0/LICENSE)
- [ONNX Runtime third-party notices](https://github.com/microsoft/onnxruntime/blob/v1.28.0/ThirdPartyNotices.txt)
- [Google SigLIP model page](https://huggingface.co/google/siglip-base-patch16-224)
- [YuNet directory license](https://github.com/opencv/opencv_zoo/blob/47534e27c9851bb1128ccc0102f1145e27f23f98/models/face_detection_yunet/LICENSE)
- [SFace directory README](https://github.com/opencv/opencv_zoo/blob/47534e27c9851bb1128ccc0102f1145e27f23f98/models/face_recognition_sface/README.md)
- [Open SFace exact-weight clarification request](https://github.com/opencv/opencv_zoo/issues/313)

Machine-readable conclusions are in
[`candidate-compliance-review-2026-08-27.json`](candidate-compliance-review-2026-08-27.json).
