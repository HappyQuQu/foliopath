# POST-MVP-5 revision 1 offline model package contract and proposed v2

- Status: **Accepted for S1 review**
- Scope: local model discovery, managed copy, strict direct read and activation
- Network: none; revision 1 has no download/source/mirror/proxy contract

## On-disk shape

`/models` contains zero or more direct-child package directories. A package directory name ends in
`.foliomodel`, but its name is never an identity and is never returned by the API.

```text
/models/
  any-local-name.foliomodel/
    manifest.json
    image_encoder.onnx
    text_encoder.onnx
    tokenizer.json
```

The directory may contain only regular files named by the built-in catalog entry. Nested directories, archives,
symlinks, hardlinks, devices, sockets, FIFOs, executable bits and ONNX external-data references are rejected.
No file from a rejected package is passed to ONNX Runtime or a tokenizer parser.

`manifest.json` is UTF-8 JSON with duplicate keys rejected, maximum 64 KiB, no unknown fields and this v1 shape:

```json
{
  "formatVersion": 1,
  "packageId": "semantic-siglip1-base-v1",
  "purpose": "semantic_image_text",
  "version": "1.0.0",
  "architecture": "portable-onnx",
  "licenseId": "Apache-2.0",
  "files": [
    {"name": "image_encoder.onnx", "size": 1, "sha256": "64 lowercase hex", "role": "image_encoder"},
    {"name": "text_encoder.onnx", "size": 1, "sha256": "64 lowercase hex", "role": "text_encoder"},
    {"name": "tokenizer.json", "size": 1, "sha256": "64 lowercase hex", "role": "tokenizer"}
  ]
}
```

The illustrative sizes/hashes above are not release catalog values. The release binary embeds the exact accepted
manifest projection and hashes; a self-declared package manifest cannot authorize itself. `packageId` is an internal
catalog key and is not accepted from public API requests.

## Proposed format v2 for ADR-0014

Format v1 remains the accepted S1 contract and is not silently reinterpreted. It cannot represent the selected
SentencePiece runtime because its third role is `tokenizer`. Until ADR-0014 is accepted, production continues to
accept only v1 and the built-in catalog remains empty.

The proposed v2 changes the third role to `sentencepiece_model` and adds explicit contract IDs:

```json
{
  "formatVersion": 2,
  "packageId": "semantic-siglip1-base-v2",
  "purpose": "semantic_image_text",
  "version": "1.0.0",
  "architecture": "portable-onnx",
  "licenseId": "Apache-2.0",
  "contracts": {
    "imagePreprocess": "siglip-rgb224-bicubic-v1",
    "textCanonicalization": "siglip-transformers-4.56.2-v1",
    "tokenizer": "sentencepiece-32k-unk2-eos1-pad1-seq64-v1",
    "embeddingAndStorage": "siglip-768-l2-f16le-v1"
  },
  "files": [
    {"name": "image_encoder.onnx", "size": 1, "sha256": "64 lowercase hex", "role": "image_encoder"},
    {"name": "text_encoder.onnx", "size": 1, "sha256": "64 lowercase hex", "role": "text_encoder"},
    {"name": "spiece.model", "size": 1, "sha256": "64 lowercase hex", "role": "sentencepiece_model"}
  ]
}
```

These are illustrative bytes, not an approved release catalog entry. V2 requires exactly the three roles above and
exactly the four frozen contract IDs. A v1 `tokenizer` role, a v2 `sentencepiece_model` role presented as v1, missing
or unknown contracts, additional files, nested paths, duplicate names/roles/JSON keys, unknown fields, trailing JSON
or bound violations fail closed. Scanner and activation must branch on an accepted exact format version; they never
guess a format from filenames.

The executable proposal under `spikes/int001-model-package-v2` verifies these shape and non-confusion rules without
changing the production catalog parser. Moving it into `internal/aimodel` requires ADR-0014 acceptance, an updated
built-in catalog entry and the production FD/tokenizer activation evidence.

## Bounds and enumeration

- Scan at most 64 direct child entries and report `truncated=true` when more exist.
- One package has at most 16 files, one directory level, 255-byte file names, 4 GiB total declared/actual bytes and
  64 KiB manifest bytes. Release catalog entries may impose smaller limits.
- Scan concurrency and install concurrency are each 1. Hashing uses bounded buffers and observes cancellation and
  operation timeouts; it never loads a model file fully into memory.
- The anchored source is opened first and each file is verified through that descriptor. A path pre-check, `realpath`
  or later reopen by string is not containment evidence.
- Compatible candidate IDs bind scan revision, package catalog key and verified source identity through the server's
  authenticated opaque-token codec. A new scan or source identity change makes them stale.

## Managed and direct storage

Managed mode copies verified files into same-filesystem staging under `/app/data/models`, rechecks size/hash, fsyncs
files, publishes a content-addressed generation with no-replace rename, then fsyncs the parent. Only after publication
does a short SQLite transaction record the installed model. Startup reconciliation accepts only a complete bounded
hash-only final report; an exact current-architecture package must match the built-in catalog and pass the full managed
validator before it is idempotently registered as available but inactive. Unknown, corrupt and truncated-scan finals
remain unregistered and are never auto-activated or deleted.

Direct mode records the anchored source identity and hashes but does not copy or modify bytes. `/models` must be a
read-only mount, and the package must remain a safe regular-file tree. Startup, session load and manual refresh repeat
the checks. Missing, writable, replaced or mismatched sources become `unavailable`; recovery requires the exact same
catalog package and hashes. FolioPath never deletes or rewrites `/models`.

Production session loading is Linux-only and passes already validated, still-open file handles to the runtime through
`/proc/self/fd/<fd>`. The runtime port never receives a container or host model path and does not copy complete ONNX
graphs into Go memory or a second temporary package. The activation attempt owns each handle until session creation
finishes; non-Linux implementations fail closed at this boundary.

The production adapter is compiled only with the explicit `onnxruntime` build tag and must link exactly ONNX Runtime
1.28.0. A build without that tag remains usable for non-AI FolioPath features but returns a stable runtime-unavailable
result for activation. The adapter fixes intra-op threads to 2, inter-op threads to 1 and disables the CPU memory
arena. It accepts only the reviewed split graph ABI: `pixel_values` float32 `[1,3,224,224]` to `image_embeds`
float32 `[1,768]`, and `input_ids` int64 `[1,64]` to `text_embeds` float32 `[1,768]`.

ONNX Runtime 1.28 does not expose cancellation for `CreateSession`. Activation therefore checks cooperative
cancellation before and after each graph load, but cannot promise immediate cancellation while one graph is being
parsed and initialized. A hard session-load timeout would require an accepted process-isolation change; it must not be
simulated by abandoning a goroutine that still owns native resources.

The default managed-model quota is 8 GiB and may later be configured from 1–64 GiB through the existing settings
owner; revision 1 reserves the greater of 1 GiB or 10% of the `/app/data` filesystem before starting a copy/build.
Staging bytes, installed managed packages and concurrently retained activation generations all count. Direct bytes do
not count, but derived embeddings and indexes do. Quota failure returns `insufficient_space` before destructive cleanup.

## Activation and removal

Install does not activate. Under accepted v1, activation opens both graphs and the declared tokenizer through the same controlled handle boundary,
loads and validates both graph ABIs, then validates tokenizer/preprocess behavior and the frozen query fixture before
compare-and-swap updates the active pointer. Merely opening the tokenizer and checking its manifest size is not
tokenizer validation; that deterministic contract belongs to `INT-203`. Failure or cancellation keeps the previous
active package/generation. Revision 1 exposes no model-delete endpoint; automatic cleanup may remove only unreferenced
staging/retired managed generations after recovery references expire.

For the selected SigLIP/SentencePiece path, that activation behavior is blocked until proposed v2 and ADR-0014 are
accepted. V1 must not be made to load `spiece.model` under its generic `tokenizer` role.

## Distribution boundary

FolioPath does not bundle model bytes and does not define where users obtain them in revision 1. Documentation may
publish exact package IDs/hashes/license notices only after Release Gate approval. It must not advertise an online
download, domestic mirror or arbitrary import. Adding any network source requires a new scope revision covering signed
catalog envelopes, key/checkpoint rotation and revocation, redirects, DNS/proxy behavior, credentials, retry, cost and
an operational owner.
