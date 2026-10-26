# Model candidate fetch record — 2026-08-25

Development machine: macOS arm64. Models and the optional image remained under
`/tmp`; none were copied into the repository.

Pinned upstream revision: `opencv/opencv_zoo@47534e27c9851bb1128ccc0102f1145e27f23f98`.

| Artifact | Bytes | SHA-256 | Result |
| --- | ---: | --- | --- |
| `face_detection_yunet_2023mar.onnx` | 232,589 | `8f2383e4dd3cfbb4553ea8718107fc0423210dc964f9f4280604804ed2552fa4` | Downloaded and verified |
| `face_recognition_sface_2021dec.onnx` | 38,696,353 | `0ba9fbfa01b5270c96627c4ef784da859931e02f04419c829e83484087c34e79` | First transfer stopped at byte 5,445,731; `curl -C -` resumed and final hash verified |
| ephemeral YuNet example image | 1,147,146 | `ab8413ad9bb4f53068f4fb63c6747e5989991dd02241c923d5595b614ecf2bf6` | Pipeline smoke only; not committed and not quality ground truth |

The successful client resume is useful evidence but does not complete INT-020:
response headers, ETag/If-Range behavior, changed object, redirect address
policy, cancellation, disk full, temporary quota, atomic activation, and signed
manifest behavior remain untested.

## SigLIP 2 semantic candidate

The official `google/siglip2-base-patch16-224` snapshot was fetched at revision
`75de2d55ec2d0b4efc50b3e9ad70dba96a7b2fa2` into `/tmp`. The nine regular files
totaled 1,539,458,338 bytes. The 1,500,800,904-byte `model.safetensors` SHA-256 was
`612923381c76ec5a9bed335d1c48827e3f2e506ac31b044b63b2031fadee6a0b`;
`tokenizer.json` was 34,363,039 bytes with SHA-256
`cb9140fae3ac5122c972d37adf83e1248471a38147ad76f8215c8872c6fd8322`, and
`tokenizer.model` was 4,241,003 bytes with SHA-256
`61a7b147390c64585d6c3543dd6fc636906c9af3865a5548f27f31aee1d4c8e2`.

The smoke verified the weight before loading and then ran with Hugging Face and
Transformers offline modes enabled. This proves neither a production download
flow nor package integrity: semantic inference needs config, preprocessing and
tokenizer artifacts in addition to the weight, while the current scanner spike
models one file per catalog entry. A signed, atomic multi-file package manifest
is therefore required before SigLIP 2 could enter an approved catalog.

## SigLIP 1 smaller comparison candidate

The official `google/siglip-base-patch16-224` snapshot was fetched at revision
`7fd15f0689c79d79e38b1c2e2e2370a7bf2761ed`, excluding the duplicate
`pytorch_model.bin`. Its nine regular files totaled 815,877,562 bytes. The
812,672,320-byte `model.safetensors` SHA-256 was
`2c63cb7d1f2e95ba501893cbb8faeb4ea9a3af295498d35097126228659c2af8`;
the 798,330-byte `spiece.model` SHA-256 was
`1e5036bed065526c3c212dfbe288752391797c4bb1a284aa18c9a0b23fcaf8ec`.
The same package-integrity limitation applies, and this candidate additionally
requires the SentencePiece runtime dependency.
