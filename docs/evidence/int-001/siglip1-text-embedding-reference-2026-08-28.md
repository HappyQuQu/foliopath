# SigLIP 1 text embedding reference fixture

Status: **deterministic Python/ORT reference generated and consumed by the
isolated Go/C parity harness**.

The retained fixed SigLIP 1 split text graph was recovered locally, so no model
download or re-export was required:

- `text_encoder.onnx`: 441,217,411 bytes, SHA-256
  `16eef12730b862a0c4f75926213d86749d9c6a5ec79b37b6feebc20f826fd664`;
- input ABI: `input_ids`, `tensor(int64)`, `[1,64]`;
- output ABI: `text_embeds`, `tensor(float)`, `[1,768]`;
- reference runtime: Python 3.12.13, ONNX Runtime 1.29.0, NumPy 2.5.2,
  CPU provider, intra-op 2, inter-op 1, CPU arena disabled.

The deterministic generator consumed all 31 cases from the pinned tokenizer
fixture. Every output was finite. Raw little-endian float32 outputs, per-case
hashes and float64 L2 norms are committed in
`spikes/int001-sentencepiece-capi/testdata/siglip-text-embedding-reference-v1.json`.
The fixture is 133,700 bytes with SHA-256
`943c05755587be5092570063c8dcadf910fc6ba06dd6e917f285b38e68f40225`.

Two independently retained split exports had the same graph SHA-256. Running
the generator against each produced byte-identical reference JSON.

The follow-up isolated Go/cgo SentencePiece → Linux ORT 1.28 harness consumed
this fixture and passed all 23,808 coordinates with maximum absolute difference
`1.811981201171875e-05`. Production FD integration, native amd64, active-run
cancellation/lifecycle evidence and normalized search behavior remain blocked.
