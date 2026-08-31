# INT-001 evidence index

Status: **local evidence frozen; S0 remains No-Go pending product decisions and external conditions**.

This directory contains machine-readable and reviewable outputs from the
isolated [`spikes/int001-ai`](../../../spikes/int001-ai/) module. No command in
this evidence set read a user media library or modified production contracts.
The local exploration was closed on 2026-08-27. New synthetic permutations are
not an active work queue; see the
[S0 closeout and blocker list](../../gates/POST-MVP-5/int-s0-closeout-and-blockers.md).

## 2026-08-25 local baseline

- Environment: macOS arm64, Go 1.26.4, `GOMAXPROCS=4` for vector runs.
- [Exact-vector raw metrics](vector-exact-darwin-arm64-2026-08-25.json)
- [Vector quantization and filtered-query metrics](vector-quantization-darwin-arm64-2026-08-25.json)
- [YuNet/SFace ONNX Runtime smoke](face-runtime-darwin-arm64-2026-08-25.json)
- [Detection → alignment → embedding pipeline smoke](face-pipeline-opencv-darwin-arm64-2026-08-25.json)
- [Pinned artifact fetch/resume record](model-fetch-2026-08-25.md)
- [Multi-file model package and atomic publish safety spike](model-package-safety-2026-08-25.md)
- [Face ROC/cluster/constraint scorer smoke](face-score-synthetic-2026-08-25.json)
- [HNSW ANN comparison and rejection evidence](ann-coder-hnsw-darwin-arm64-2026-08-25.json)
- [SigLIP 2 multilingual semantic and storyboard-proxy smoke](semantic-siglip2-darwin-arm64-2026-08-25.json)
- [SigLIP 1 smaller-candidate comparison](semantic-siglip1-darwin-arm64-2026-08-25.json)
- [Candidate/runtime and license assessment](candidate-assessment-2026-08-25.md)
- Dataset and model manifest schemas validated using synthetic example records.
- Linux-only model-directory scanner correctly refused to produce safety
  evidence on macOS; native Linux `openat2` runs are still required.

## 2026-08-26 public-license semantic pilot

- [Wikimedia Commons Cosplay pilot provenance, comparison and limitations](semantic-commons-pilot-2026-08-26.md)
- [SigLIP 2 machine result](semantic-commons-pilot-siglip2-darwin-arm64-2026-08-26.json)
- [SigLIP 1 machine result](semantic-commons-pilot-siglip1-darwin-arm64-2026-08-26.json)
- [512px bounded-input surrogate comparison](semantic-bounded-input-2026-08-26.md)
- [SigLIP 2 bounded-input machine result](semantic-bounded-siglip2-darwin-arm64-2026-08-26.json)
- [SigLIP 1 bounded-input machine result](semantic-bounded-siglip1-darwin-arm64-2026-08-26.json)
- [Production govips adapter input comparison under Linux/amd64 QEMU](semantic-vips-input-2026-08-26.md)
- [SigLIP 2 + govips input machine result](semantic-vips-siglip2-darwin-arm64-2026-08-26.json)
- [SigLIP 1 + govips input machine result](semantic-vips-siglip1-darwin-arm64-2026-08-26.json)
- [Native arm64 vs amd64 QEMU govips byte comparison](semantic-vips-cross-arch-2026-08-26.json)
- [Catalog-owned model download failure-matrix spike](model-download-failure-matrix-2026-08-26.md)
- [Production storyboard adapter 4/10-frame extraction evidence](video-storyboard-production-adapter-2026-08-26.md)
- [SQLite vector generation strong-kill recovery](vector-sqlite-recovery-2026-08-26.md)
- [Constrained 100k vector concurrency assessment](vector-concurrency-linux-arm64-2026-08-26.md)
- [Pinned SigLIP 2 ONNX self-export and arm64 runtime evidence](semantic-onnx-export-arm64-2026-08-26.md)
- [ONNX split-session allocator comparison on Linux/arm64](semantic-onnx-arena-linux-arm64-2026-08-27.md)
- [ONNX split-session 100-cycle cgroup follow-up on Linux/arm64](semantic-onnx-arena-100-cycle-linux-arm64-2026-08-27.md)
- [Resident split image/text session capacity on Linux/arm64](semantic-onnx-resident-sessions-linux-arm64-2026-08-27.md)
- [Smaller SigLIP 1 split and combined-load evidence on Linux/arm64](siglip1-split-combined-linux-arm64-2026-08-27.md)
- [SigLIP 1 beside the production 100k catalog capacity path](siglip1-production-catalog-capacity-linux-arm64-2026-08-27.md)
- [Rejected SigLIP 1 dynamic-int8 candidate](siglip1-dynamic-int8-rejection-2026-08-27.md)
- [SigLIP 1 float16-internal production-capacity evidence](siglip1-float16-production-capacity-linux-arm64-2026-08-27.md)
- [Repeated SigLIP 1 float16 production-capacity comparison](siglip1-float16-production-capacity-repeated-linux-arm64-2026-08-27.md)
- [SigLIP 1 float16 split failure-closed smoke](siglip1-float16-failure-closed-arm64-2026-08-27.md)
- [ONNX valid-protobuf hostile graph smoke](onnx-hostile-graph-arm64-2026-08-27.md)
- [ONNX oversized-allocation failure smoke](onnx-oversized-allocation-linux-arm64-2026-08-27.md)
- [ONNX Runtime Go/C API cancellation smoke](onnx-go-capi-linux-arm64-2026-08-27.md)
- [ONNX Runtime distroless closure and vulnerability evidence](onnx-distroless-runtime-linux-arm64-2026-08-27.md)
- [Production catalog search-keyset component diagnosis](catalog-search-keyset-components-linux-arm64-2026-08-27.md)
- [Production catalog search query-plan diagnosis](catalog-search-query-plan-linux-arm64-2026-08-27.md)
- [Catalog search order-first scope/filter/sort/cursor matrix](catalog-search-order-first-matrix-linux-arm64-2026-08-27.md)
- [Catalog search order-first two-library global matrix](catalog-search-order-first-multilibrary-linux-arm64-2026-08-27.md)
- [Candidate runtime/model compliance review](candidate-compliance-review-2026-08-27.md)
- [Split image encoder plus 100k vector concurrency on Linux/arm64](semantic-vector-combined-load-linux-arm64-2026-08-27.md)
- [Anonymous face cluster/manual-person state-machine evidence](face-cluster-state-machine-2026-08-26.md)
- [Dataset governance manifest v2 validator evidence](dataset-governance-manifest-v2-2026-08-27.md)
- [AI diagnostic closed-field privacy contract](ai-diagnostic-privacy-contract-2026-08-27.md)
- [Activation final availability-revision CAS](activation-availability-cas-2026-08-28.md)
- [Managed install worker process-kill recovery](managed-install-worker-sigkill-recovery-2026-08-28.md)
- [Direct model nested-mount boundary on Linux/arm64](direct-model-nested-mount-linux-arm64-2026-08-28.md)
- [Managed model real ENOSPC on Linux/arm64](managed-model-enospc-linux-arm64-2026-08-28.md)
- [Native image inference process-kill recovery on Linux/arm64](native-image-inference-sigkill-linux-arm64-2026-08-28.md)
- [SentencePiece C API tokenizer smoke on native Linux/arm64](sentencepiece-capi-linux-arm64-2026-08-27.md)
- [SentencePiece C API FD/lifecycle evidence on Linux/arm64](sentencepiece-capi-lifecycle-linux-arm64-2026-08-28.md)
- [SentencePiece C API parity smoke on emulated Linux/amd64](sentencepiece-capi-emulated-amd64-2026-08-28.md)
- [SigLIP tokenizer Transformers reference conformance](sentencepiece-transformers-reference-linux-arm64-2026-08-28.md)
- [SigLIP 1 deterministic text embedding reference](siglip1-text-embedding-reference-2026-08-28.md)
- [SigLIP 1 Go/C tokenizer-to-text parity on Linux/arm64](siglip1-go-c-text-parity-linux-arm64-2026-08-28.md)
- [Proposed model package format v2 executable contract](model-package-v2-contract-2026-08-28.md)
- [SentencePiece + ONNX Runtime distroless closure on Linux/arm64](sentencepiece-ort-distroless-runtime-linux-arm64-2026-08-28.md)
  ([machine-readable record](sentencepiece-ort-distroless-runtime-linux-arm64-2026-08-28.json))
- [SentencePiece 0.2.1 source-content attestation](sentencepiece-source-attestation-2026-08-28.md)
- [SentencePiece + ONNX Runtime distroless emulated amd64 preflight](sentencepiece-ort-distroless-emulated-amd64-2026-08-28.md)
- [SentencePiece + ONNX Runtime cross-architecture Grype evidence](sentencepiece-ort-grype-cross-arch-2026-08-28.md)
  ([machine-readable record](sentencepiece-ort-grype-cross-arch-2026-08-28.json))
- [Expanded glibc reachability input across architectures](sentencepiece-ort-glibc-reachability-cross-arch-2026-08-28.md)
  ([machine-readable record](sentencepiece-ort-glibc-reachability-cross-arch-2026-08-28.json))
- [Distroless Debian 13 glibc security status refresh](glibc-security-status-refresh-2026-08-29.md)
- Ten fixed public-license photographs and 24 paired Chinese/English queries
  validate the acquisition/scoring path and expose one SigLIP 1 Chinese Top-1
  miss. This remains a pilot, not the 1,000-image quality set.
- Three fresh-process bounded-input runs kept all Top-1 and first-relevant
  ranks stable while reducing observed model-process RSS. The preprocessor is
  a Pillow surrogate, not production libvips evidence.
- The current production govips adapter then generated the same pilot inputs
  under Linux/amd64 QEMU. All Top-1 and first-relevant ranks remained stable;
  native amd64/arm64 and full-process evidence remain pending.
- A fresh native Linux/arm64 Dockerfile build passed the current production
  imagevips tests. Its ten pilot WebPs were byte-identical to the amd64 QEMU
  outputs; native amd64 performance and ONNX embeddings remain pending.

The same day's isolated download state-machine tests cover pinned ETag resume,
mid-stream cancellation followed by resume, exact-origin redirects, quota
preflight, wrong hash and no-replace publication. macOS/arm64 and native
Linux/arm64 passed, and a size-limited arm64 tmpfs exercised real kernel
`ENOSPC`. A real child-process `SIGKILL` followed by a new-process resume also
passed on both arm64 environments. A transport spike also pins one validated
DNS answer, rejects mixed/special-use answers and disables environment proxies
on both arm64 environments. Resolver error/empty-answer single-attempt failure,
package rename/fsync kill boundaries and complete-package `ENOSPC` were later
covered. A private-CA TLS/SNI test also proved trusted handshake, unknown-CA
rejection and fallback after an explicit first-address failure on both arm64
environments; a blocking dial also proved bounded per-address timeout and
fallback. Public-CA/CDN/DNS TTL, outer retry policy, host power loss and native
Linux/amd64 remain pending. An Ed25519 primitive now authenticates exact catalog
payload/metadata and rejects expiry, rollback and equivocation on both arm64
environments; production key custody/rotation/revocation and durable checkpoint
ownership remain pending. An isolated SQLite registry then proved atomic
checkpoint/active-pointer rollback, idempotence and restart persistence on both
arm64 environments. Native Linux/arm64 anchored reconciliation further proved
orphan registration without activation and missing/corrupt/restore availability
transitions without changing active/checkpoint; production ownership, complete
managed/direct lifecycles and native amd64 remain pending. A real read-only
bind-mount test then verified immutable direct provenance, unavailable-on-
disappearance and exact-remount recovery without copy/delete/active switching;
production direct deployment/API/UI semantics remain pending.

The project-pinned FFmpeg 7.1.5 build also ran the existing production
storyboard adapter on a disposable ten-second H.264 fixture under native
Linux/arm64. Both the four- and ten-frame plans produced canonical valid
results without changing the source hash. This proves the extraction path can
be reused without a second FFmpeg adapter; it does not prove quality on the
required licensed 100-video set.

An isolated SQLite/WAL generation test was also strong-killed with a complete
replacement generation and active-pointer update still uncommitted. Reopen
retained the prior generation, exposed none of the replacement, passed
`integrity_check`, and then rebuilt/switched atomically on macOS/arm64 and
native Linux/arm64. Real embeddings, 100k joint load, amd64 and final storage
budget remain open.

Three native Linux/arm64 4 CPU/4 GiB/no-network runs then combined bounded
float32 backfill, exact search, a keyset-browse proxy, cooperative cancellation
and restart at 100k × 512. Search latency and recovery passed their isolated
checks, but the configured relative browse-degradation gate did not, and the
410.6 MB database leaves insufficient room for the combined video/face scope.
Float32 is not a viable final combined layout; real-embedding float16/dimension
quality and production browse measurements remain required. The same matrix
with float16 reduced the SQLite file to 136.9 MB and search P95 to
138.854–156.912 ms, so it remains the capacity candidate; synthetic vectors do
not approve its retrieval quality.

These numbers are not release claims. Missing evidence includes the 1,000-image/
100-video representative licensed quality set, privacy-reviewed real face
ground truth, ONNX production runtime integration and real-embedding tolerance,
native Linux/amd64, production browse/full-process concurrency, final combined
space, SBOM/compliance and production signing-key/mirror operations.

After the S0 exploration closed, a targeted S1 blocker spike compiled the
official SentencePiece 0.2.1 library and a narrow Go/cgo wrapper on native
Linux/arm64. Fixed Chinese/English/Unicode token IDs and the 64-token boundary
passed. This is implementation-path evidence only: ADR-0014 remains proposed,
the production graph remains unchanged, and amd64/supply-chain/hostile-input/
resource plus end-to-end embedding parity remain open.

An app-level managed-package corruption test now exercises the production
catalog, managed validator, model registry, activation transaction and
availability owner against temporary files. Replacing one reviewed graph marks
the model unavailable while preserving the active generation and an existing
embedding. The unavailable state survives a database-component restart;
restoring the exact bytes afterward returns it to available while preserving
derived state. A read-only media sentinel retains its SHA-256, size and mtime
through this path. This is a synthetic-package ownership/recovery proof, not
approved model or native inference evidence. See
[managed model corruption recovery](managed-model-corruption-recovery-2026-08-28.md).

A file-backed integration test now gives a semantic backfill lease to a real
helper process, kills that process, and reopens the database in a new process
lifecycle. Expired-lease recovery requeues exactly once; the next claim advances
attempt/revision while retaining checkpoint 101 and completed progress 1/2.
This proves semantic queue process-kill recovery, not managed publication or
native inference recovery. See
[semantic backfill process-kill recovery](semantic-backfill-sigkill-recovery-2026-08-28.md).

Production managed-package publication now has process-kill evidence on both
sides of its atomic rename. A kill during real file copy leaves only staging,
which a new store removes; a kill after publish leaves one complete known final,
which reconciliation retains and an idempotent retry verifies. Only the
free-space probe is injected because the development disk is below the frozen
10% reserve. See
[managed package publication process-kill recovery](managed-publication-sigkill-recovery-2026-08-28.md).

Startup reconciliation now consumes the managed store's bounded hash-only final
report. Only a complete report whose package matches the built-in catalog,
current architecture and full production validator is idempotently registered
as available; it remains inactive. Unknown, corrupt and truncated-scan finals
are untouched and unregistered. The app corruption vertical now starts through
this owner chain. See
[managed published-package orphan reconciliation](managed-orphan-reconciliation-2026-08-28.md).

A Linux/arm64 lifecycle follow-up then loaded the fixed model through an open
file descriptor, exercised concurrent callers and closed handles, rejected
empty/truncated/oversized/non-regular inputs, and completed 100 measured
load/close cycles with a 7,602,176-byte retained RSS increase after explicit Go
memory release. Pre-cancelled contexts are rejected before native entry, but
SentencePiece still has no mid-call interruption primitive. Native amd64,
long-duration soak, final-image supply chain and text embedding parity remain
open.

The identical tagged suite also compiled and passed in an emulated
Linux/amd64 userspace, with a 7,655,424-byte retained RSS increase after 100
measured load/close cycles. This is useful ABI/behavior smoke only: QEMU timing
and memory are not native evidence, so the native amd64 gate remains open.

A deterministic 31-case Transformers 4.56.2/SentencePiece 0.2.1 reference
fixture then compared every canonical string and all 64 token IDs. Its first
run exposed contextual Greek final-sigma drift in the Go canonicalizer;
per-code-point lowercase now matches the pinned Transformers behavior and the
Linux/arm64 matrix passes. A follow-up froze literal registered `</s>` and
`<unk>` as invalid user queries and rejects them before inference. The matrix
is still not exhaustive Unicode or native amd64 evidence.

The retained 441,217,411-byte SigLIP 1 text graph then generated raw 768-D
float32 outputs for all 31 tokenizer cases. Two independent byte-identical
exports produced byte-identical 133,700-byte reference fixtures. The isolated
Go/cgo SentencePiece-to-ORT 1.28 chain then matched all 23,808 coordinates with
maximum absolute difference `1.811981201171875e-05`; production FD composition,
native amd64 and final composition remain open. Active-run cancellation and
eight serialized callers passed. After a roughly 356 MB cold-to-stable RSS
expansion, 10 additional load/close cycles retained only 28,672 bytes; this is
plateau evidence, not permission to omit the warm-capacity cost.

The proposed model package v2 is now executable in an isolated Go validator:
it fixes three roles and four pipeline contract IDs, and proves v1/v2 are not
silently interchangeable. Production remains v1-only until ADR-0014 is
accepted.

An isolated no-SSL distroless image now combines the SentencePiece 0.2.1 and
ORT 1.28 native closures without baking either model into the image. Under the
restricted runtime profile, all 31 text cases retained the fixed parity result.
This closes only the arm64 combined-SONAME subproof: the inherited glibc
Critical/High findings, native amd64, complete source attestation, merged SBOM,
signed provenance and production composition remain blockers.

The pinned SigLIP 2 source was also exported three times with the pinned
Optimum ONNX toolchain; the resulting 1,501,208,026-byte ONNX and deterministic
reference were byte-identical. macOS arm64 ORT 1.29.0 and three native
Linux/arm64/no-network ORT 1.28.0 runs matched all PyTorch outputs within
`1e-4`. Linux peak RSS was about 2.19 GiB for this model-only smoke, so this
closes only the arm64 self-export subproof and increases the full-process
capacity concern.

Three fresh native Linux/arm64 containers then kept the split image encoder
active while the 100k float16 SQLite workload performed backfill, exact search,
keyset-browse proxy, cancellation and restart. Container peak stayed at
1.29–1.33 GB, search P95 at 165.443–186.532 ms, and every restart reached
100,000 rows. The unchanged relative browse-degradation gate failed by
5.32–12.74× despite sub-millisecond absolute proxy latency. This is useful
combined-load evidence, but it is not the complete FolioPath process, a real
100k image backfill, production HTTP browse, face concurrency or amd64 proof.

A 100-cycle arena-off image/text load-switch follow-up kept retained process
RSS nearly flat (+28 KiB), but container `memory.peak` reached 3,719,651,328
bytes and failed the unchanged 3.2 GiB `R-024` gate. This rejects repeated
full-model reload as the current lifecycle strategy even though the narrower
allocator check passed.

Keeping both arena-off split sessions resident was worse: cgroup current
stabilized around 3.56 GB and peak reached 4,008,951,808 bytes. At cycle 100,
`memory.stat` showed about 1.90 GB anonymous plus 1.65 GB file memory. The
dual-resident SigLIP 2 strategy is also rejected; the current model has no
accepted 4 GiB lifecycle layout.

The smaller SigLIP 1 candidate was then split twice byte-identically and passed
PyTorch tolerance plus macOS/Linux arm64 ranking parity. Its dual-resident
100-cycle peak was 2.18 GB; three runs combined both encoders with the 100k
float16 vector proxy at 2.364–2.370 GB peak and recovered all rows. It becomes
the resource-priority candidate, not a selected model: the 10-image pilot still
has one Chinese Top-1 miss, and the relative browse gate failed every run.

The same resident runtime was then placed beside the repository's real 10k
directory/100k file production catalog capacity test. Ordinary recursive browse
degraded 11.2%, not the proxy's multi-fold result, but full container peak rose
from 1.604 GB to 3.590 GB and failed the 3.2 GiB gate. Global search and
storyboard-period browse exceeded 20% in the single pair, while the existing
search-keyset absolute budget failed even without AI. This narrows the proxy
problem but keeps `INT-013` No-Go.

A dynamic-QInt8 `MatMul`/`Gemm` experiment reduced the two SigLIP 1 graphs to
about 281 MB and Linux cgroup peak to 811 MB, but was rejected before capacity:
only 8/24 macOS/Linux Top-3 lists agreed and native Linux Chinese Recall@1 fell
from 0.917 to 0.25. Resource savings cannot compensate for this quality and
cross-runtime failure.

The float16-internal alternative retained all 24 pilot Top-3 lists across
float32/float16-model and macOS/Linux arm64. Its 100-cycle resident peak was
1.614 GB and its production 10k/100k capacity peak 2.906 GB, below 3.2 GiB.
Ordinary recursive browse degraded 14.5%, but global search rose 41.6% in the
single pair and the pre-existing search-keyset budget still failed. It becomes
the resource-priority candidate, not a selected production model.

Two additional fresh baseline/AI pairs confirmed float16 AI peaks of
2.860–2.951 GB and ordinary recursive-browse degradation of 6.0–14.5% across
all three pairs. The first global-search outlier did not reproduce. One
storyboard-browse pair was 20.34%, and all six baseline/AI search-keyset runs
still failed the existing 250 ms absolute budget, so the complete Gate remains
open.
