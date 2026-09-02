# POST-MVP-5 S2 external blocker refresh

Date: 2026-08-31

Status: **blockers unchanged; S2A/S2C release Gates remain No-Go**.

## SFace weight provenance

OpenCV Zoo's current SFace README says every file in the model directory is Apache-2.0 and describes the ONNX files as
MobileFaceNet instances trained with SFace loss. However, the exact-weight clarification request for
`face_recognition_sface_2021dec.onnx` (`0ba9fbfa…34e79`) remains open, with no assignee, maintainer response, milestone,
or linked change. The request still asks which training dataset produced this weight and whether the dataset/model terms
permit commercial inference and redistribution.

Sources:

- [OpenCV Zoo SFace README](https://github.com/opencv/opencv_zoo/blob/main/models/face_recognition_sface/README.md)
- [OpenCV Zoo issue #313](https://github.com/opencv/opencv_zoo/issues/313)
- [OpenCV Zoo SFace weight pull request #35](https://github.com/opencv/opencv_zoo/pull/35)
- [OpenCV Zoo training-details issue #288](https://github.com/opencv/opencv_zoo/issues/288)

Project decision: the directory license statement remains useful license evidence, but it does not answer the frozen
training-data provenance and commercial redistribution approval questions. The exact SFace weight therefore remains
`production hold`; local functional tests do not promote it into the reviewed catalog.

An Apache-2.0 Open Model Zoo ArcFace ResNet100 ONNX alternative was also pinned and tested. The exact graph failed the
frozen ONNX Runtime 1.28 execution contract and is explicitly rejected. A separately identified deterministic
normalization can execute and has narrow single-fixture parity plus local functional evidence, but it creates a 261 MB
derived model. Open Model Zoo identifies that exact source as `LResNet100E-IR,ArcFace@ms1m-refine-v2` and states
Apache-2.0 distribution, while InsightFace's current official pretrained-model terms restrict its public weights to
non-commercial research unless a separate commercial license is obtained. The repository-level Apache statement does not
resolve that weight-level conflict. Its commercial permission, contract, quality, capacity, dual-architecture and
supply-chain approvals are therefore all open. See the
[exact-graph rejection](arcface-resnet100-replacement-rejection-darwin-arm64-2026-08-31.md) and
[normalized candidate](arcface-resnet100-normalized-candidate-darwin-arm64-2026-09-01.md). Neither replaces or unblocks
the held SFace weight; the normalized candidate cannot enter the production catalog without a separate commercial license
or authoritative written clarification for the exact artifact.

Sources:

- [Open Model Zoo ArcFace model README](https://github.com/openvinotoolkit/open_model_zoo/blob/4d4266fbbb7eb5ab80944c2800d7f304868d573d/models/public/face-recognition-resnet100-arcface-onnx/README.md)
- [Open Model Zoo public model index](https://github.com/openvinotoolkit/open_model_zoo/blob/4d4266fbbb7eb5ab80944c2800d7f304868d573d/models/public/index.md)
- [InsightFace Python package license notice](https://github.com/deepinsight/insightface/blob/master/python-package/README.md)
- [InsightFace commercial licensing notice](https://github.com/deepinsight/insightface/blob/master/server/LICENSING.md)

Later same-day recheck still shows issue #313 as open, unassigned, unlabeled, without milestone, relationship,
development link, or maintainer reply. No new evidence changes the production hold.

## AuraFace v1 alternative candidate

The official `fal/AuraFace-v1` repository supplies a materially different replacement path. At pinned revision
`af6d057c9b0ec4071d4c49c80e3539258798b609`, `glintr100.onnx` is 260,694,151 bytes with SHA-256
`a7933ea5330113b01c9b60351d8f4c33003f145d8470ac5f0e52ee2effe25c60`. The provider attaches Apache-2.0,
describes the model as intended for commercial use, and says it was trained on a commercial dataset. The exact graph
also executes directly under frozen ORT 1.28 on Darwin/arm64 and passed the bounded authorized functional smoke.

This candidate avoids the known InsightFace public-weight non-commercial statement attached to the rejected Open Model
Zoo lineage, but the public card does not identify the commercial dataset, licensors, consent basis or deletion chain.
FolioPath therefore records it as `candidate / compliance review required`, not approved. It still needs weight-specific
compliance sign-off, governed quality, production package/runtime, native dual-architecture capacity and final supply-chain
evidence. See [AuraFace candidate evidence](auraface-v1-candidate-darwin-arm64-2026-09-01.md).

Official sources:

- [AuraFace model card](https://huggingface.co/fal/AuraFace-v1/blob/af6d057c9b0ec4071d4c49c80e3539258798b609/README.md)
- [AuraFace Apache-2.0 license](https://huggingface.co/fal/AuraFace-v1/blob/af6d057c9b0ec4071d4c49c80e3539258798b609/LICENSE.md)
- [AuraFace provenance/licensing discussion #8](https://huggingface.co/fal/AuraFace-v1/discussions/8)
- [AuraFace release article and limitations](https://huggingface.co/blog/isidentical/auraface)

## Intel Open Model Zoo alternative review

Intel's official `face-reidentification-retail-0095` is an additional commercially positioned alternative with
hash-bound Open Model Zoo distribution and repository licensing. The exact official files are OpenVINO IR `.xml` +
`.bin`, not an ONNX artifact, so adopting them would add a second production inference runtime or require an ungoverned
reverse conversion. The official records identify LFW evaluation but do not disclose the training-data/consent/deletion
chain. It is therefore held for architecture and provenance review rather than used to bypass the AuraFace hold. See
[the pinned alternative audit](intel-face-reidentification-retail-0095-hold-2026-09-01.md).

The repository-hosted discussion asks the provider for the same exact-weight origin, derivation, dataset terms,
rightsholder authority, redistribution and NOTICE evidence required by this Gate. It was opened on 2025-09-29 and has
follow-ups dated 2026-01-06 and 2026-04-13, but the visible record has no provider response as of 2026-09-01. The release
article separately acknowledges ethnicity-dependent efficacy and training-data limitations. This refresh therefore
strengthens the need for the existing provenance and governed bias checks; it does not approve or reject the candidate.

An additional official OpenCV Zoo 4.9 report says the Zoo collected models with licensing in mind, generally describes
them as usable for commercial purposes, and identifies SFace as an OpenCV Area Chair contribution. That strengthens the
repository-level commercial-use context but still does not identify the training dataset or explicitly bind the statement
to the exact December 2021 ONNX checksum. Because issue #313 remains open with those exact questions unanswered, the
frozen exact-weight provenance requirement is still not met.

The upstream history does not supply the missing binding either. Pull request #35, which introduced the December 2021
weight lineage later replayed as commit `ba91a3b`, has no description and records no training dataset or weight-specific
redistribution statement. Issue #288 separately asked for the official weight's training parameters and explained its
128-dimensional/model-size differences; it was closed without an upstream answer in the public record. These are
negative audit findings, not proof that the weight is unsafe, but they rule out treating the historical PR or that closed
issue as the missing exact-weight provenance evidence.

- [OpenCV Zoo 4.9 report](https://github.com/opencv/opencv_zoo/blob/main/reports/2023-4.9.0/opencv_zoo_report-en-2023-4.9.0.md)

## Debian stable runtime base

Debian's current trixie package page still publishes `libc6 2.41-12+deb13u3` for both amd64 and arm64. This is the same
runtime version already present in the exact image evidence; no newer trixie stable package is available to change the
recorded glibc finding disposition.

Source:

- [Debian trixie libc6 package](https://packages.debian.org/en/trixie/libs/libc6)

Project decision: the final signed SBOM/VEX/security approval remains missing. This refresh does not permit suppressing
the recorded findings or treating reachability probes as an accepted VEX.

Later same-day package-page recheck still reports `libc6 2.41-12+deb13u3` and publishes both amd64 and arm64 downloads;
there is no newer trixie stable revision to rebuild against in this review.
