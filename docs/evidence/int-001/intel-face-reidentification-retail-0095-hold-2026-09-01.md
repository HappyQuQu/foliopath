# Intel face-reidentification-retail-0095 candidate hold

Date: 2026-09-01

Status: **officially distributed technical alternative; held for runtime architecture and training-provenance review**

## Official records reviewed

- Open Model Zoo commit `4d4266fbbb7eb5ab80944c2800d7f304868d573d` describes
  [`face-reidentification-retail-0095`](https://github.com/openvinotoolkit/open_model_zoo/blob/4d4266fbbb7eb5ab80944c2800d7f304868d573d/models/intel/face-reidentification-retail-0095/README.md)
  as an Intel MobileNetV2-derived 128×128 BGR face embedding model with a 256-dimensional output and LFW accuracy
  0.9947.
- Its pinned
  [`model.yml`](https://github.com/openvinotoolkit/open_model_zoo/blob/4d4266fbbb7eb5ab80944c2800d7f304868d573d/models/intel/face-reidentification-retail-0095/model.yml)
  points to official Open Model Zoo storage, fixes sizes and SHA-384 checksums, and applies the repository Apache-2.0
  license. The distributed artifacts are OpenVINO IR `.xml` + `.bin` pairs for FP32, FP16 and FP16-INT8; it does not
  identify a source ONNX file.
- Intel's official Hugging Face use-case repository revision
  [`a175eae65ab4eef0f0cb51961f43a553bec33764`](https://huggingface.co/Intel/facial-recognition/tree/a175eae65ab4eef0f0cb51961f43a553bec33764)
  is MIT-tagged, identifies this exact re-identification model, and downloads the same versioned IR files for an
  OpenVINO pipeline. The card also warns that facial recognition handles biometric data and requires applicable
  consent/privacy controls.
- The Open Model Zoo
  [dataset preparation guide](https://github.com/openvinotoolkit/open_model_zoo/blob/4d4266fbbb7eb5ab80944c2800d7f304868d573d/data/datasets.md)
  identifies LFW as the evaluation dataset for this model. It does not identify the training dataset, collection basis,
  consent/deletion chain or demographic composition.

## Decision

This source is materially stronger than an anonymous third-party conversion: Intel distributes the exact weight files,
provides checksums, licenses the repository, documents commercial/edge deployment examples and maintains an official
runtime path. It nevertheless cannot replace the current AuraFace ONNX candidate without a new architecture decision:

1. FolioPath's accepted inference boundary is ONNX Runtime. Consuming the official artifacts would add OpenVINO as a
   second production inference runtime and change the native dependency/supply-chain boundary.
2. Converting OpenVINO IR back to ONNX would create a FolioPath-derived graph without an official conversion recipe or
   source graph. That is not a safe shortcut around the runtime decision.
3. The official records reviewed disclose LFW evaluation, not the model's training-data provenance or biometric
   collection/consent/deletion basis. A repository license alone does not answer the independent S2C privacy review.
4. FolioPath's governed 50×20 quality/bias matrix and native amd64/arm64 final-package evidence would still be required.

The candidate is therefore **held, not rejected on model quality and not selected for implementation**. Reopening it
requires either an accepted ADR for an OpenVINO production runtime plus exact SBOM/VEX evidence, or an official
hash-bound ONNX artifact/conversion contract, together with written training-provenance/compliance review. No artifact
was downloaded into the repository and production composition remains empty.
