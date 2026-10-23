# Candidate and license assessment — 2026-08-25

Status: preliminary source/runtime review, not legal approval and not a frozen
model catalog. YuNet/SFace and both SigLIP smokes have pinned upstream revisions
and verified artifacts. They still need production preprocessing acceptance,
SBOM, notices, legal sign-off, and native dual-architecture evidence.

| Candidate | Preliminary decision | Evidence and unresolved work |
| --- | --- | --- |
| ONNX Runtime CPU | Keep for spike | The upstream runtime is MIT and its official getting-started matrix includes Linux and ARM64. FolioPath has no official Go API to rely on, so a C API/CGO adapter would add a native distroless and dual-architecture closure that must be built, scanned, cancelled, and leak-tested. |
| Google SigLIP 2 base patch16 224 | Keep as candidate A; do not approve | The official model card calls it multilingual/image-text retrieval and Apache-2.0. A pinned PyTorch/F32 package totaled 1.539 GB. On macOS arm64/4 threads, an offline eight-image synthetic smoke observed 1.171 GB RSS after image inference, 58.414 ms image P95, English Recall@1 1.0 and Chinese 0.875. These drawings are not representative quality, and the run does not prove ONNX/Linux, browser concurrency, leakage, backfill, quantization, notices, or complete package integrity. The model is large but has not yet failed the 3.2 GiB process gate. |
| Google SigLIP base patch16 224 | Keep as smaller candidate B; do not approve | The official model metadata is Apache-2.0. The pinned F32 package excluding the duplicate PyTorch bin totaled 815.878 MB and adds a SentencePiece native tokenizer dependency. On the same smoke it observed 724 MB RSS, 60.373 ms image P95, English Recall@1 1.0 and Chinese 0.875. The easy synthetic set cannot distinguish its quality from SigLIP 2; lower RSS is useful, but no ONNX/Linux, representative person set, concurrency, SBOM or redistribution sign-off exists. |
| OFA Chinese-CLIP ViT-B/16 | Do not approve yet; candidate B only if license is clarified | The official repository documents Chinese retrieval, 188M parameters, and ONNX conversion. The official model card reviewed on 2026-08-25 did not state a model license, and the repository did not expose a root license at the expected path. Downloadability is not redistribution permission. |
| OpenCV Zoo YuNet | Keep as face detector candidate | Pinned `2023mar` artifact: 232,589 bytes, SHA-256 `8f2383…52fa4`. macOS ONNX Runtime CPU single-thread random-tensor P95 was 15.780 ms over 1,000 iterations; an OpenCV pipeline smoke also produced boxes/landmarks. These are compatibility results, not recall. Makeup/occlusion and small/large face limits remain unmeasured. |
| OpenCV Zoo SFace | Keep as face embedding candidate | Pinned `2021dec` artifact: 38,696,353 bytes, SHA-256 `0ba9fb…34e79`. macOS ONNX Runtime CPU single-thread random-tensor P95 was 16.261 ms and pipeline alignment produced finite 128-D embeddings. RSS grew about 0.56 MiB during 1,000 iterations and needs native investigation. Pair ROC, cosplay variation, core-cluster precision, and manual constraints remain mandatory. |
| InsightFace public pretrained packs | Reject as default distributable weights | Upstream states that public pretrained model packages are for non-commercial research and directs commercial users to separate licensing. MIT code does not change the weight restriction. Do not bundle or auto-download these weights without a separately reviewed grant. |
| `coder/hnsw` ANN | Do not select yet | Direct code is CC0 and the API is Go with atomic file replacement. At 10k/128-D, default-ish settings reached only 48.5% mean Recall@20; reaching 100% on random vectors required M=32/EfSearch=512 and a 51.7 s build. The importer trusts encoded allocation lengths, exports were not byte deterministic, and transitive/dual-architecture closure is not signed off. Exact scan already meets the local latency budget, so this complexity currently has no benefit. |

The 2026-08-26 [public-license Cosplay pilot](semantic-commons-pilot-2026-08-26.md)
adds ten real photographs and 24 paired Chinese/English queries. SigLIP 2 reached
Chinese/English Recall@1 1.0; SigLIP 1 reached 0.917/1.0, with the Chinese armor
query relevant at rank 2. Broad multi-relevant queries exposed incomplete Top-3
coverage in both models. This is useful discrimination evidence but remains far
below the representative 1,000-image Gate and used an unsafe direct-original
diagnostic input path, so it does not change either candidate's preliminary decision.

The follow-up [512px bounded-input surrogate](semantic-bounded-input-2026-08-26.md)
kept every query's Top-1 and first-relevant rank stable across three fresh
processes. Median RSS after images was about 1.178 GB for SigLIP 2 and 0.728 GB
for SigLIP 1, materially below direct-original decode. This strengthens the
bounded-input design requirement but still does not select a candidate: Pillow
is not the production libvips transform, the quality set remains ten images,
and no complete Linux/ONNX/concurrent-browse process was measured.

## 2026-08-26 ONNX artifact source review

The accepted Google SigLIP 2 revision does not contain an accepted ONNX release
artifact. A 1.5 GB ONNX file is visible only under an unmerged
[`refs/pr/15`](https://huggingface.co/google/siglip2-base-patch16-224/blob/refs%2Fpr%2F15/onnx/model.onnx),
so it is not part of the pinned upstream release.

The separate
[`onnx-community/siglip2-base-patch16-224-ONNX`](https://huggingface.co/onnx-community/siglip2-base-patch16-224-ONNX/tree/main)
repository is an HF Staff/community conversion with one contributor, three
commits and approximately 11.4 GB across variants. It may be comparison input,
but is not a FolioPath trusted origin: the project does not own its conversion
toolchain, source-revision binding or signing/checkpoint operations.

The only admissible production-candidate route is a reproducible project export
from the pinned Google revision using a pinned exporter, opset, shapes and
validation tolerance. Hugging Face's
[`main_export` contract](https://huggingface.co/docs/optimum-onnx/onnx/package_reference/export)
supports explicit revision, opset, shapes and validation. The resulting package
must record source weights, exporter/runtime SBOM, command/config, ONNX digest,
numerical comparison and project catalog signature. Until reproduced on native
arm64/amd64, no ONNX file is an approved downloadable model.

## Primary sources

- ONNX Runtime [license](https://github.com/microsoft/onnxruntime/blob/main/LICENSE) and [platform matrix](https://onnxruntime.ai/getting-started)
- Google [SigLIP 2 model card](https://huggingface.co/google/siglip2-base-patch16-224)
- Google [SigLIP model card](https://huggingface.co/google/siglip-base-patch16-224)
- OFA-Sys [Chinese-CLIP repository](https://github.com/OFA-Sys/Chinese-CLIP), [deployment notes](https://github.com/OFA-Sys/Chinese-CLIP/blob/master/deployment_En.md), and [model card](https://huggingface.co/OFA-Sys/chinese-clip-vit-base-patch16)
- OpenCV Zoo [YuNet README](https://github.com/opencv/opencv_zoo/blob/main/models/face_detection_yunet/README.md) and [SFace README](https://github.com/opencv/opencv_zoo/blob/main/models/face_recognition_sface/README.md)
- InsightFace [license statement](https://github.com/deepinsight/insightface#license) and [model-zoo restriction](https://github.com/deepinsight/insightface/blob/master/model_zoo/README.md)
- `coder/hnsw` pinned [source](https://github.com/coder/hnsw/tree/36cab6028fed4adc9c3edf2323a06f0a95c1f030), [CC0 license](https://github.com/coder/hnsw/blob/36cab6028fed4adc9c3edf2323a06f0a95c1f030/LICENSE), and [persistence implementation](https://github.com/coder/hnsw/blob/36cab6028fed4adc9c3edf2323a06f0a95c1f030/encode.go)

## Immediate consequences

1. There is no approved model catalog yet. The OpenCV entries are explicitly
   `candidate`; the Linux scanner only permits `approved` entries.
2. A semantic candidate must beat the quality/resource gates with Chinese and
   English FolioPath queries. SigLIP 2's synthetic result only keeps it in the
   comparison. SigLIP 1 now supplies a smaller second development candidate, but
   neither result satisfies representative quality or production runtime evidence.
3. The face route is technically plausible with YuNet + SFace, but it remains a
   clustering suggestion system. No evidence currently supports automatic naming
   or a 99.5% core-cluster precision claim.
4. No domestic download mirror exists merely because ModelScope or another page
   hosts a model. A mirror requires an operated origin, provenance, authorization,
   owner, availability evidence, and signed manifest.
5. The current one-file model-catalog/scanner shape cannot safely activate a
   semantic package containing weight, tokenizer, preprocessing and config files.
   S0 must test a signed multi-file package manifest and atomic activation rather
   than treating only `model.safetensors` as the model identity.
