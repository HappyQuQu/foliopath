# SentencePiece + ONNX Runtime distroless emulated amd64 preflight

Status: **linux/amd64 package, SONAME and numeric preflight passed under QEMU;
native amd64 Gate remains open**.

The same parameterized
[`Dockerfile.runtime`](../../../spikes/int001-sentencepiece-capi/Dockerfile.runtime)
used by the native arm64 spike built a linux/amd64 no-SSL distroless image on an
Apple arm64 host. The build consumed both native archives through local named
contexts and checked each digest before extraction:

- SentencePiece 0.2.1 source: 13,485,527 bytes, SHA-256
  `c1a59e9259c9653ad0ade653dadff074cd31f0a6ff2a11316f67bee4189a8f1b`;
- official ORT 1.28.0 x64 archive: 9,125,960 bytes, SHA-256
  `a3e1b79d7bb1bf09696ce675f49e4064e6c81f6202b8225624fff0e93f8d6407`;
- exact amd64 `cc-debian13` child manifest:
  `sha256:1d2e87077bb3b12be8622609c5975fed6a3cba63e68fed53209293be10f7022c`;
- exact amd64 `base-nossl-debian13` child manifest:
  `sha256:cc74a68b2924afee50ab111f14d86b9f4e1c461d02ac8382708343f97f6b6f33`.

The first parameterized build correctly failed before compilation because the
previously pinned distroless digest was an arm64 child manifest; specifying
`--platform linux/amd64` cannot override a single-platform digest. The fixture
now keeps digest pinning but accepts explicit architecture-specific base child
manifests and Debian library directories. An earlier attempt also rejected an
over-broad `/tmp` named context after encountering an unrelated unreadable
directory; the successful run used a dedicated context containing only the
reviewed x64 archive.

The resulting image is 30,805,741 bytes. Its runnable platform manifest is
`sha256:45a9b6c082331769d680e20bbef3c3d253a7a98f70d640d764349d8ffe7aad95`,
config is `sha256:2c91cc767e8c7417b02234828b6cd05a2b9cde448abd6958da29fc6676fa08cf`,
and local provenance-bearing index is
`sha256:3dd3520098658912941a2ac001e55780c58548c511d5b14e8c3c435a45ed6901`.
The x64 SentencePiece shared library is 1,399,912 bytes with SHA-256
`2ab8e0156a79cfe44e076d18e9f920bb01682722835bbe7f35d4eac1f43c63fc`;
the ORT shared library is 24,268,848 bytes with SHA-256
`1461ef7cc3d9e49982591721683cc3e3a55580aeca9a5254e7aac47b75ee4bab`.

Under QEMU with 4 CPUs, 4 GiB, 64 PIDs, uid/gid 65532, no network, a
read-only root filesystem, all capabilities dropped, `no-new-privileges` and
read-only model mounts, all 31 tokenizer-to-text cases passed. Across 23,808
coordinates the maximum absolute difference from the fixed ORT 1.29 reference
was `2.09808349609375e-05`, below the fixed `atol=1e-4, rtol=1e-4` contract.
The emulated run took 4.98 seconds.

Docker Scout 1.24.0 scanned the exact local amd64 image twice and identified
17 base/Go packages, but did not identify the manually copied ONNX Runtime or
SentencePiece libraries. Architecture-specific CycloneDX 1.6 component records
now bind the x64 archive, library, source, license and notices digests. The same
failure-closed merge utility used for arm64 flattened scanner nesting,
consolidated dependencies and added those records. Both fresh scans normalized
to byte-identical incomplete documents containing 1,267 components and 18
dependency rows, UUID `5fc56386-38e8-5f44-9a74-3fd2f50f3065`, and SHA-256
`da490242a8406fa06e8cfc9b6652c6c3cb0ebdda9b84317fa45cb9035a8a13b8`.
This is deterministic package evidence under emulation, not a native amd64
release SBOM or vulnerability result.

This closes the amd64 archive selection, Dockerfile parameterization, package
layout, SONAME resolution and basic numerical preflight. It does **not** close
native linux/amd64 timing, RSS, cancellation, load/close, vulnerability,
SBOM/provenance or full-process evidence. QEMU results must not be promoted to
native evidence. Machine-readable details are in
[`sentencepiece-ort-distroless-emulated-amd64-2026-08-28.json`](sentencepiece-ort-distroless-emulated-amd64-2026-08-28.json).
