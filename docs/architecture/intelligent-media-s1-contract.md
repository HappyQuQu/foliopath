# POST-MVP-5 A+B S1 capability and transaction contract

- Status: **Accepted for S1 review**
- Scope: `POST-MVP-5 revision 1`, Slice A model foundation and Slice B image semantic search only
- Excludes: AI tags, video semantic search, faces and people
- Source of product truth: [Frozen scope](../releases/POST-MVP-5-scope.md)
- Structural decision: [ADR-0013](../adr/0013-local-ai-runtime-and-derived-vector-index.md)

This document freezes ownership and failure semantics before production packages, migrations or handlers exist.
Go names are directional contracts, not permission to create them before `INT-S1` is Go.

## Canonical owners and dependency direction

```text
internal/api
  -> internal/aimodel.Service
  -> internal/semantic.Service

internal/semantic (owns use cases, query binding and ports)
  -> CatalogReader port
  -> ImageEncoder port
  -> EmbeddingRepository port
  -> SearchIndex port
  -> JobScheduler port

internal/aimodel (owns package compatibility and model lifecycle)
  -> CandidateScanner port
  -> PackageStore port
  -> ModelRegistry port
  -> SemanticActivator port

internal/store/sqlite implements repositories
internal/inference/onnx implements semantic.ImageEncoder
internal/files implements anchored media/model-source opens
internal/jobs implements bounded scheduling primitives
internal/app is the only composition and lifecycle owner
```

- `internal/semantic` owns semantic enable/disable, coverage, source-fingerprint invalidation, generation build,
  query normalization, ranking, cursor binding and semantic error classification.
- `internal/aimodel` owns the built-in compatibility catalog, `/models` scan, opaque candidates, package
  verification, managed/direct storage mode, availability, installed model records and activation orchestration.
- `internal/jobs` owns generic admission, lease, retry and cancellation mechanics. It does not decide whether a
  semantic item is stale, complete, retryable or eligible.
- `internal/catalog` owns asset existence, library/directory scope and source fingerprint. Semantic code receives
  opaque IDs and relative facts through a narrow reader; it does not query catalog tables directly.
- `internal/files` is the only filesystem boundary. API, aimodel and semantic services never resolve arbitrary
  paths themselves.
- `internal/api` owns transport validation and maps capability errors to the single OpenAPI error shape. It does
  not manufacture capability state or perform fallback.

Cross-capability calls use public service methods or capability-owned ports. No package copies state machines,
query keys, ETag calculation, transaction rules, model compatibility or error mapping.

## Service contracts

### `aimodel.Service`

| Use case | Input | Result | Required semantics |
| --- | --- | --- | --- |
| `ListModels` | none | installed models and active purpose binding | No paths; stable opaque IDs |
| `ScanCandidates` | admin request ID | scan revision and compatible/incompatible summaries | Fixed `/models` only; bounded; no symlink/mount crossing |
| `InstallCandidate` | candidate ID, scan revision, `managed|direct`, idempotency key | operation ID | Reject stale/unknown candidate; no path or URL input |
| `GetOperation` | operation ID | progress and terminal error code | No raw tool/runtime output |
| `CancelOperation` | operation ID, expected revision | accepted terminal/current state | Cooperative; never rolls back active model |
| `ActivateModel` | installed model ID, expected active revision, idempotency key | activation operation ID | New semantic generation first; compare-and-swap active pointer last |
| `RefreshAvailability` | internal/startup/admin trigger | current model states | Direct source revalidated; unavailable is fail-closed |

### `semantic.Service`

| Use case | Input | Result | Required semantics |
| --- | --- | --- | --- |
| `GetLibrarySettings` | library ID | enabled, revision, active generation, coverage | Offline is status, not empty |
| `UpdateLibrarySettings` | library ID, enabled, expected revision | updated settings | Compare-and-swap; enabling requires compatible available model |
| `RequestBackfill` | library ID, `missing|all`, idempotency key | batch/job ID | Coalesce equivalent active intent; bounded admission |
| `CancelBackfill` | job ID, expected revision | current/terminal state | Stop new admission; committed embeddings stay valid |
| `ClearLibraryData` | library ID, expected settings revision, confirmation token | deletion operation ID | Derived semantic data only; originals/curation/models untouched |
| `Search` | query, scope, filters, cursor, limit | assets, next cursor, coverage snapshot | `score DESC, asset_id ASC`; no vector returned; cursor binds query and generations |

The image encoder receives decoded, bounded input and immutable model-generation metadata. It cannot download,
select or activate a model. The search index receives normalized vectors and opaque asset IDs; SQLite embeddings
remain sufficient to rebuild it, and revision 1 uses SQLite exact search rather than an external ANN truth store.

Revision 1 persists normalized image embeddings as fixed-dimension IEEE-754 binary16 in little-endian order. The
semantic owner uses a float64 squared-norm accumulator before encoding and normalizes the decoded binary16 vector
again before scoring. A non-finite/zero vector or dimension/blob-length mismatch is corruption, not a valid zero-score
result. Text-query embeddings remain bounded transient float32 values and are never persisted.

Embedding writes are bounded, single-generation/single-library batches. Dimension and binary16 validity are checked
before entering the serialized SQLite write gate, then generation identity/state is rechecked inside the short
transaction. Replaying identical source fingerprint/vector bytes is a no-op; changed source identity replaces or
invalidates only that asset-generation row. Asset, library and generation deletion use the declared composite foreign
keys and never cause media filesystem writes.

For a claimed backfill page, embedding rows, completed/failed/stale coverage counters, the job checkpoint and the
public operation progress advance in one write transaction. The commit must match claim revision, progress revision
and prior checkpoint; a stale worker or any row/count failure advances none of them. Native decoding and inference
never occur inside this transaction.

Backfill admission derives its eligible count from the catalog; callers never supply coverage totals. Image and
animated-image candidates use bounded `asset_id ASC` keyset pages. `missing` includes both absent embeddings and rows
whose stored source fingerprint differs from current catalog identity. Idempotency keys are persisted only as a
SHA-256 digest, while a separate fixed request digest detects body reuse. Equivalent active mode intent coalesces;
another active mode for the same library/generation conflicts so a rebuild cannot reset progress beneath a running
missing-only job.

Semantic leases are restart-safe and are not handled by the generic fail-interrupted model-operation recovery.
Expired running work is requeued for at most three claims; each claim receives a new monotonically changing claimed
revision, so a pre-restart worker cannot commit. Expired cancellation becomes cancelled, while already committed
embedding batches remain valid. Claim selects at most one running/cancelling semantic job per library and orders
eligible libraries by their most recent terminal semantic job before job age.

The backfill processor consumes bounded candidate pages sequentially. It receives originals only through a semantic
asset source backed by the existing verified media content service and `internal/files`; the adapter rechecks the
requested library ID, source fingerprint and classified format before exposing an already-open `io.ReadSeekCloser`.
Preprocessing and generation-bound image inference are separate ports. A missing runtime/model is a terminal
fail-closed outcome and must never be replaced with a zero, random or placeholder vector. Individual corrupt inputs
may advance failed/stale counters, but operation success is allowed only after its admitted total is accounted for.
Application lifecycle must not start the production queue until a real encoder with cancellation and resident-session
accounting is composed.

The native ONNX adapter exposes an image-only session rather than loading both graphs for every asset. A session owns
the safely opened model FD for its full lifetime, accepts only the frozen `[1,3,224,224]` float32 tensor and copies
exactly `[1,768]` float32 output before releasing ORT values. Runs are serialized with close, use per-run cancellation
options and return context cancellation without exposing runtime messages. This low-level session is not itself the
generation owner: application composition still must enforce one resident generation, hard execution deadlines and
idle unloading before the backfill queue may start.

Application composition supplies that owner with a strict one-session invariant. Before every Run, it re-resolves the
generation as active with dimension 768 and requires an available reviewed package through `aimodel.Service`; cold
loading additionally revalidates its source. Switching generation closes the previous session before opening the next; load and
each Run have a 30-second deadline, faulted sessions are discarded, five minutes idle unloads the session, and process
shutdown closes it synchronously. The semantic worker has concurrency one and also consumes the existing global
background admission slot; queue recovery remains lease-owned rather than generic operation interruption recovery.
The same lifecycle owner exposes bounded internal resource accounting for current resident sessions, active Runs and
cumulative load/Run/unload transitions. Those counters contain no model or generation identifiers, paths, queries,
vectors or native runtime messages and do not create a second diagnostics owner.

Semantic library settings use compare-and-swap revision. An absent row is represented read-only as disabled revision
1; enabling requires the active generation to reference an available model, while disabling preserves rebuildable
embeddings. Backfill HTTP never accepts generation or eligible counts: the service resolves both from current settings
and catalog state. Generic operation cancellation dispatches semantic kinds to the semantic queue so job and public
operation cannot diverge. Each progress commit updates enabled-library state in the same transaction: unresolved work
is `building`, complete clean coverage is `ready`, and terminal failed/stale coverage is `degraded`.

Revision-1 exact vector search streams eligible SQLite rows and retains only bounded Top-K matches (hard maximum 200).
It normalizes transient query vectors with the same float64 norm rule, re-normalizes decoded binary16 image vectors,
and scores by cosine dot product. Rows are excluded unless their generation is active, library is enabled and online,
and stored source fingerprint still equals catalog identity. Ordering and continuation are exactly
`score DESC, asset_id ASC`; this repository tuple alone is not a public cursor. The semantic owner now obtains one
SQLite read snapshot for the active/available generation, selected scope and enabled/online library set; its encrypted
cursor binds a query hash (never query text), library/directory/recursive scope, catalog revision, generation, the
ordered library/settings/progress fingerprint and the continuation tuple. Directory direct/recursive filtering is
applied before scoring. The HTTP adapter validates the frozen query surface, hydrates at most 200 ordered matches
through one catalog-owned batch projection (not per-result SQLite reads) and maps stable semantic errors, but its route remains absent from production composition until the
accepted production text tokenizer/encoder can produce the query vector.

The retained SigLIP image tensor contract is exactly 224×224 interleaved uint8 RGB after bounded decode, orientation,
alpha removal matching PIL `convert("RGB")`, sRGB conversion and direct bicubic resize. The semantic owner converts
that buffer to planar
`[3,224,224]` float32 by rescaling each uint8 value in float64 by `1/255`, downcasting to float32, then applying
float32 mean/std `0.5/0.5`. These stages must not be algebraically collapsed without regenerating and accepting the
cross-runtime bit fixture. The existing lossy 512px WebP thumbnail is quality-pilot evidence only and is not the
authoritative model input, because its extra encode/decode and resize stages change pixels.

The retained SigLIP 1 candidate requires SentencePiece-compatible tokenization. S1 did not select a production
tokenizer runtime, and the current package format names only one tokenizer-role file. S2 must either accept and supply-
chain-review a deterministic tokenizer adapter or revise the reviewed package contract with the complete tokenizer
artifact set. It may not approximate tokenization, invoke Python, execute package code, or silently add a second native
runtime. Until that decision and a frozen bilingual token fixture pass, activation cannot be considered valid.

## State machines

Model package state:

```text
candidate -> verifying -> installed
              |             |
              v             v
           rejected      available <-> unavailable
```

Activation operation:

```text
queued -> loading -> building -> validating -> succeeded
   |         |          |           |
   +---------+----------+-----------+-> cancelling -> cancelled
   +---------+----------+-----------+-> failed
```

Only `succeeded` may advance the active model/generation pointer. Cancellation or failure leaves the old active
pointer and all reliable embeddings intact.

Semantic generation state:

```text
building -> ready -> active -> retired
    |         |
    +---------+-> failed
```

There is at most one active semantic generation for a model purpose. `retired` remains readable only while a
request/session or recovery reference exists and is otherwise eligible for derived-data cleanup.

## Stable capability errors

| Code | Owner | Meaning / HTTP mapping |
| --- | --- | --- |
| `ai_disabled` | semantic | Library has semantic disabled; `409` |
| `model_unavailable` | aimodel surfaced by semantic | Required local model cannot be loaded; `503` |
| `model_candidate_stale` | aimodel | Candidate scan revision changed or expired; `409` |
| `model_incompatible` | aimodel | Package is not in the built-in compatibility catalog or fails validation; `422` |
| `model_source_unsafe` | files/aimodel | Direct source is writable, symlinked, crossing a mount or otherwise unsafe; `422` |
| `semantic_not_ready` | semantic | No reliable active generation exists; `409` |
| `semantic_cursor_stale` | semantic | Query/model/index/catalog revision no longer matches; `409` |
| `semantic_busy` | semantic/jobs | Bounded interactive capacity exhausted; `429` with retry hint |
| `revision_conflict` | owning service | ETag/expected revision mismatch; `412` |
| `insufficient_space` | aimodel/semantic | Safety margin or quota would be violated; `422` |

Unknown internal failures map to the existing opaque internal error. Raw paths, query text, vectors, SQL,
runtime messages and subprocess output never enter the response.

## Transaction decision table

| Event | Transaction owner | Commit atomically | On partial failure |
| --- | --- | --- | --- |
| Enable/disable one library | semantic repository | setting value + setting revision | No change; return `revision_conflict` or opaque failure |
| Commit one embedding batch | semantic repository | embedding rows + generation counters + job checkpoint | Roll back batch; previously committed rows remain |
| Publish generation | semantic repository | generation `ready/active`, previous active `retired`, active pointer revision | Keep previous active; new rows remain non-active/rebuildable |
| Install managed model | aimodel | Files publish before DB transaction; then model record + operation terminal state | Unreferenced final package is reconciled as orphan, never auto-activated |
| Register direct model | aimodel | Verified source identity + model record + operation state | No record; source untouched |
| Direct source becomes unsafe/missing | aimodel | model `unavailable` + availability revision | Keep active-generation metadata/embeddings; dependent query fails closed |
| Cancel backfill | jobs + semantic checkpoint | cancellation intent/revision; workers commit only already completed atomic batches | No new admission; safe rows remain and coverage stays truthful |
| Clear one library | semantic repository | disable intent/revision + deletion tombstone/operation | Worker deletes bounded child batches; state remains `clearing` until complete; query cannot treat partial data as complete |
| Library removal | library service orchestrates capability cleanup | Existing library-removal transaction records cleanup intent | Semantic rows cascade by library; original files and global models untouched |
| Asset removed/source changes | catalog publishes identity change; semantic repository consumes | Stale semantic rows removed or marked stale with coverage checkpoint | Last full library index remains governed by catalog; no filesystem mutation |
| Concurrent activation | aimodel/semantic | Compare expected active revision and generation | One succeeds; loser gets `revision_conflict`, not a second active pointer |

Filesystem I/O, decoding, inference and model verification never occur while a SQLite write transaction is held.
Long cleanup is restart-safe, uses bounded transactions and exposes an operation rather than pretending to be
atomic across the whole library.

## API and UI visibility boundary

Clients may receive IDs, revisions, stable states, counts, ratios, safe package metadata and stable error codes.
They never receive embeddings, host/container paths, source URLs, query fingerprints, filesystem identities,
model filenames, runtime configuration, raw errors or per-asset background queue internals.
