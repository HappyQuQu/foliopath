# AuraFace v1 face-embedding replacement candidate

Date: 2026-09-01

Status: **technical candidate only; production, privacy, quality and release Gates remain No-Go**

## Frozen artifact and provider statements

The official `fal/AuraFace-v1` repository revision
`af6d057c9b0ec4071d4c49c80e3539258798b609` publishes `glintr100.onnx` as 260,694,151 bytes with SHA-256
`a7933ea5330113b01c9b60351d8f4c33003f145d8470ac5f0e52ee2effe25c60`. The repository labels the model
Apache-2.0, contains the full Apache-2.0 text, describes AuraFace as intended for commercial settings, and says the
weight was trained on a commercial dataset. Those are materially clearer provider statements than the held SFace and
InsightFace-derived candidates, but they are inputs to compliance review rather than a FolioPath approval.

Official records:

- [AuraFace model card](https://huggingface.co/fal/AuraFace-v1/blob/af6d057c9b0ec4071d4c49c80e3539258798b609/README.md)
- [AuraFace license](https://huggingface.co/fal/AuraFace-v1/blob/af6d057c9b0ec4071d4c49c80e3539258798b609/LICENSE.md)
- [Pinned model artifact](https://huggingface.co/fal/AuraFace-v1/blob/af6d057c9b0ec4071d4c49c80e3539258798b609/glintr100.onnx)
- [Repository provenance/licensing discussion #8](https://huggingface.co/fal/AuraFace-v1/discussions/8)

The public card does not identify the commercial dataset, its licensors, collection dates, consent basis, deletion
process, or a weight-specific copyright/provenance attestation. Consequently the candidate still needs written
compliance review and cannot enter the production catalog on this evidence alone.

A repository-hosted due-diligence discussion opened on 2025-09-29 asks fal to confirm the exact `glintr100.onnx`
origin, derivation, training datasets and terms, weight-specific rights, redistribution/sublicensing grant, NOTICE and
artifact identity. The visible thread contains follow-ups on 2026-01-06 and 2026-04-13 but no provider answer as of this
2026-09-01 review. The AuraFace announcement also expressly notes ethnicity-dependent efficacy and training-data
limitations. These are authoritative reasons to retain the governed bias matrix and written provenance sign-off; the
Apache file and commercial-positioning statement do not answer those separate questions.

## ORT 1.28 compatibility

The exact graph loaded directly under ONNX Runtime 1.28.0 on Darwin/arm64 without the compatibility rewrite required by
the rejected Open Model Zoo graph. Its frozen interface is dynamic batch `data [N,3,112,112] float32` to
`1333 [1,512] float32`; an all-zero tensor produced a finite non-zero vector. The candidate harness fixes the provider's
InsightFace preprocessing to RGB `(pixel - 127.5) / 127.5` and rejects graph/name/shape/type drift.

## Authorized local functional result

The corrected schema-v2 bounded, read-only harness sampled 135 files across nine top-level directory groups under the
operator-authorized media root. It decoded all 135 files, YuNet found 79 candidates, and AuraFace produced 79 finite,
non-zero 512-dimensional embeddings with no invalid outputs. Median/P95 embedding latency was 48.234/51.919 ms on
Darwin/arm64. No original was copied or modified, and no crop, embedding, path, directory name or person name was
persisted.

The root contains multiple people, and directory grouping is not face-level identity ground truth. At threshold 0.5,
the directory-only aggregate accepted 46 of 2,692 cross-group pairs and 85.09% of 389 within-group pairs; at 0.6 it
accepted zero cross-group pairs and 60.93% of within-group pairs. These are deliberately named group accept rates, not
verification recall/FPR. They only show that the pipeline and threshold measurement run; they are not demographic bias
or the required 99.5% anonymous-core precision. The earlier schema-v1 JSON is retained as a superseded historical
record because its `same`/`different` field names could imply unsupported identity labels.

## Disposition

AuraFace is retained as a **candidate for formal compliance and native evaluation**, not selected or activated. Remaining
requirements include owner approval of exact model provenance/redistribution, a reviewed detector+embedder package
contract, production adapter, governed 50×20 ground truth and five bias slices, native Linux amd64/arm64 parity and RSS,
100k joint capacity, final SBOM/VEX/notices/provenance, and privacy/compliance/security/release signatures. Until those
exist, production face composition and catalog remain empty/fail-closed.

Current structured aggregate evidence:
[schema-v2 JSON](auraface-v1-candidate-darwin-arm64-2026-09-01-v2.json).
