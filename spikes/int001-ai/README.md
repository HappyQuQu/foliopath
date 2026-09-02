# INT-001 AI feasibility spike

This is an isolated feasibility module. It is not imported by the FolioPath
application and does not define production API, database, or UI behavior. It
must only use synthetic, openly licensed, or specifically authorized fixtures;
never point it at production `/library` or an unreviewed media root. The bounded
functional smoke below additionally requires an explicit operator authorization
reference and emits no paths, crops, or embeddings.

The first slice provides three reproducible tools:

```sh
# Validate dataset and model manifests.
go run . validate -dataset testdata/dataset-manifest.example.json
go run . validate -dataset testdata/dataset-manifest.authorized-face-template.json
go run . validate -models testdata/model-catalog.example.json
go run . validate -models testdata/model-catalog.package.example.json
go run . validate -models testdata/model-catalog.siglip-candidates.json

# Compare in-memory and SQLite-blob exact vector scans.
go run . vector -backend memory -items 10000 -dims 256 -queries 10
go run . vector -backend sqlite -items 10000 -dims 256 -queries 10
go run . vector -backend sqlite -items 100000 -dims 512 -filter-modulo 10
go run . vector-concurrency -items 100000 -dims 512 -batch 256 -format float16
go run . vector-quant -items 100000 -dims 512 -queries 20
go run . ann -items 10000 -dims 128 -queries 20 -m 16 -ef-search 64
go run . face-score -input testdata/face-score-synthetic.json

# Linux only: fail-closed discovery from a read-only /models directory.
go run . scan-models -root /models -catalog model-catalog.json

# Synthetic-only ONNX load/inference smoke (models are never committed).
python3 -m venv /tmp/foliopath-int001-venv
/tmp/foliopath-int001-venv/bin/pip install -r requirements-face-smoke.txt
/tmp/foliopath-int001-venv/bin/python face_smoke.py \
  --catalog testdata/model-catalog.opencv-zoo-candidates.json \
  --model-root /tmp/operator-supplied-models

# Optional ephemeral pipeline fixture; never commit the referenced image.
/tmp/foliopath-int001-venv/bin/python face_pipeline_smoke.py \
  --catalog testdata/model-catalog.opencv-zoo-candidates.json \
  --fixture-manifest testdata/face-pipeline-fixture.json \
  --model-root /tmp/operator-supplied-models

# Optional bounded functional smoke on an explicitly operator-authorized local
# dataset. The command reads ordinary image files without following symlinks,
# emits aggregate JSON only, and never persists crops or embeddings. It is not
# identity ground truth, quality evidence, legal approval, or release evidence.
/tmp/foliopath-int001-venv/bin/python face_functional_smoke.py \
  --catalog testdata/model-catalog.opencv-zoo-candidates.json \
  --model-root /tmp/operator-supplied-models \
  --media-root "${AUTHORIZED_FACE_ROOT}" \
  --dataset-id operator-local-functional-v1 \
  --authorization-ref operator-local-functional-2026-08-31 \
  --max-per-group 100 \
  --pair-limit 100000

When the media root has top-level groups, the report also contains bounded
same-group/cross-group threshold metrics. Group labels and embeddings remain only
in memory; the report deliberately omits names, paths, crops, and vectors. These
metrics are functional error-merge evidence, not governed identity ground truth.

# Rejected-upstream/derived ArcFace replacement experiment. The normalizer is
# pinned to one exact source graph and ONNX 1.22.0, refuses overwrite/symlink
# input, and emits one frozen derived digest. Neither graph is production
# approved; this path exists only to reproduce the recorded rejection/candidate.
/tmp/foliopath-int001-venv/bin/python arcface_normalize_onnx.py \
  --source /tmp/operator-supplied-models/arcfaceresnet100-8.onnx \
  --output /tmp/operator-supplied-models/arcfaceresnet100-8-normalized-v1.onnx
/tmp/foliopath-int001-venv/bin/python face_arcface_functional_smoke.py \
  --catalog testdata/model-catalog.arcface-alternative.json \
  --model-root /tmp/operator-supplied-models \
  --normalized-embedder /tmp/operator-supplied-models/arcfaceresnet100-8-normalized-v1.onnx \
  --normalized-embedder-sha256 345e28fd93dc48fd7bfb3552c58434ca7e279f85ee2132c810b26945d4550844 \
  --media-root "${AUTHORIZED_FACE_ROOT}" \
  --dataset-id operator-local-arcface-functional-v1 \
  --authorization-ref operator-local-functional-2026-08-31

# AuraFace v1 candidate uses its exact upstream graph directly. The pinned
# catalog fixes SHA-256, ORT tensor names/shapes and InsightFace preprocessing;
# the weight remains outside Git and the result is not production approval.
/tmp/foliopath-int001-venv/bin/python face_arcface_functional_smoke.py \
  --catalog testdata/model-catalog.auraface-candidate.json \
  --model-root /tmp/operator-supplied-models \
  --media-root "${AUTHORIZED_FACE_ROOT}" \
  --dataset-id operator-local-auraface-functional-v2 \
  --authorization-ref operator-user-authorized-functional-2026-09-01

# Prepare private review material from an explicitly authorized local tree.
# The output must remain outside Git. It contains derived face thumbnails,
# embeddings, relative paths and a pending review CSV; candidate clusters are
# not identity ground truth until a human reviewer completes the CSV.
/tmp/foliopath-int001-venv/bin/python face_ground_truth_prepare.py \
  --catalog testdata/model-catalog.auraface-candidate.json \
  --model-root /tmp/operator-supplied-models \
  --media-root "${AUTHORIZED_FACE_ROOT}" \
  --output /tmp/foliopath-private-face-review \
  --authorization-ref operator-private-review-2026-09-01

# Standard-library safety contract; no OpenCV model files required.
python3 -m unittest face_functional_smoke_test.py face_arcface_functional_smoke_test.py

# Synthetic multilingual semantic smoke. The pinned model stays outside Git.
/tmp/foliopath-int001-venv/bin/python semantic_fixture_generate.py \
  --output /tmp/foliopath-int001-semantic-fixtures
HF_HUB_OFFLINE=1 TRANSFORMERS_OFFLINE=1 \
  /tmp/foliopath-int001-venv/bin/python semantic_smoke.py \
  --model /tmp/operator-supplied-siglip2 \
  --fixture testdata/semantic-score-synthetic.json \
  --images /tmp/foliopath-int001-semantic-fixtures

# Public-license pilot: downloads only to an explicit temporary directory,
# rechecks Wikimedia Commons metadata, and verifies every digest.
/tmp/foliopath-int001-venv/bin/python semantic_public_fixture_fetch.py \
  --manifest testdata/semantic-score-commons-pilot.json \
  --output /tmp/foliopath-int001-commons-pilot
/tmp/foliopath-int001-venv/bin/python semantic_input_prepare.py \
  --source-manifest testdata/semantic-score-commons-pilot.json \
  --source-images /tmp/foliopath-int001-commons-pilot \
  --output-manifest /tmp/foliopath-int001-commons-prepared.json \
  --output-images /tmp/foliopath-int001-commons-prepared
HF_HUB_OFFLINE=1 TRANSFORMERS_OFFLINE=1 \
  /tmp/foliopath-int001-venv/bin/python semantic_smoke.py \
  --model /tmp/operator-supplied-siglip2 \
  --fixture /tmp/foliopath-int001-commons-prepared.json \
  --images /tmp/foliopath-int001-commons-prepared

# Reproducible SigLIP 2 ONNX export and deterministic reference comparison.
# Both large model outputs and the reference remain outside Git.
python3 -m venv /tmp/foliopath-int001-onnx-export
/tmp/foliopath-int001-onnx-export/bin/pip install \
  -r requirements-semantic-onnx-export.txt
/tmp/foliopath-int001-onnx-export/bin/python semantic_onnx_export.py \
  --cache /tmp/foliopath-int001-hf \
  --output /tmp/foliopath-int001-siglip2-onnx \
  --reference-output /tmp/foliopath-int001-siglip2-reference.npz
/tmp/foliopath-int001-onnx-export/bin/python semantic_onnx_split_export.py \
  --source /tmp/foliopath-int001-hf/models--google--siglip2-base-patch16-224/snapshots/75de2d55ec2d0b4efc50b3e9ad70dba96a7b2fa2 \
  --output /tmp/foliopath-int001-siglip2-split-onnx
/tmp/foliopath-int001-onnx-export/bin/python onnx_compare.py \
  --model /tmp/foliopath-int001-siglip2-onnx/model.onnx \
  --reference /tmp/foliopath-int001-siglip2-reference.npz
/tmp/foliopath-int001-onnx-export/bin/python onnx_runtime_stress.py \
  --model /tmp/foliopath-int001-siglip2-onnx/model.onnx \
  --reference /tmp/foliopath-int001-siglip2-reference.npz
/tmp/foliopath-int001-onnx-export/bin/python semantic_onnx_smoke.py \
  --model /tmp/foliopath-int001-siglip2-onnx/model.onnx \
  --processor /tmp/foliopath-int001-siglip2-onnx \
  --fixture /tmp/foliopath-int001-commons-prepared.json \
  --images /tmp/foliopath-int001-commons-prepared \
  --expected-model-bytes 1501208026 \
  --expected-model-sha256 18a16d73759d3760a664596660e5bb8f4800635bbec39775bafe88a85cf57226

# Run the same prepared pilot through separate fixed-shape image/text graphs.
/tmp/foliopath-int001-onnx-export/bin/python semantic_onnx_split_tensor_smoke.py \
  --image-model /tmp/foliopath-int001-siglip2-split-onnx/image_encoder.onnx \
  --image-model-bytes 371682125 \
  --image-model-sha256 7fc85e0e8a0f4e5fce7be45c7830e7473c81f010633b7d31bc1613dafc734ab0 \
  --text-model /tmp/foliopath-int001-siglip2-split-onnx/text_encoder.onnx \
  --text-model-bytes 1129345413 \
  --text-model-sha256 ef9451f51568152758e53ebca85cec4d84d2462c45e1611b42f49a42bd1be953 \
  --tensors /tmp/foliopath-int001-siglip2-prepared.npz \
  --expected-tensors-sha256 REVIEWED_SHA256 \
  --fixture /tmp/foliopath-int001-commons-prepared.json
/tmp/foliopath-int001-onnx-export/bin/python onnx_split_switch_stress.py \
  --image-model /tmp/foliopath-int001-siglip2-split-onnx/image_encoder.onnx \
  --image-model-bytes 371682125 \
  --image-model-sha256 7fc85e0e8a0f4e5fce7be45c7830e7473c81f010633b7d31bc1613dafc734ab0 \
  --text-model /tmp/foliopath-int001-siglip2-split-onnx/text_encoder.onnx \
  --text-model-bytes 1129345413 \
  --text-model-sha256 ef9451f51568152758e53ebca85cec4d84d2462c45e1611b42f49a42bd1be953 \
  --tensors /tmp/foliopath-int001-siglip2-prepared.npz \
  --expected-tensors-sha256 REVIEWED_SHA256

# Keep both split sessions resident and sample process/cgroup memory while
# alternating bounded inference.
/tmp/foliopath-int001-onnx-export/bin/python onnx_split_resident_stress.py \
  --image-model /tmp/foliopath-int001-siglip2-split-onnx/image_encoder.onnx \
  --image-model-bytes 371682125 \
  --image-model-sha256 7fc85e0e8a0f4e5fce7be45c7830e7473c81f010633b7d31bc1613dafc734ab0 \
  --text-model /tmp/foliopath-int001-siglip2-split-onnx/text_encoder.onnx \
  --text-model-bytes 1129345413 \
  --text-model-sha256 ef9451f51568152758e53ebca85cec4d84d2462c45e1611b42f49a42bd1be953 \
  --tensors /tmp/foliopath-int001-siglip2-prepared.npz \
  --expected-tensors-sha256 REVIEWED_SHA256 \
  --cycles 100 \
  --threads 2 \
  --stop-file /tmp/foliopath-int001-resident-stop

# Create a hash-bound dynamic-int8 weight candidate. This does not approve it;
# runtime compatibility and retrieval quality must be rerun afterward.
python onnx_dynamic_quantize.py \
  --input /tmp/image_encoder.onnx \
  --expected-input-bytes IMAGE_BYTES \
  --expected-input-sha256 IMAGE_SHA256 \
  --output /tmp/image_encoder.int8.onnx

# Create a float16-internal candidate while preserving float32 API I/O.
python onnx_float16_convert.py \
  --input /tmp/image_encoder.onnx \
  --expected-input-bytes IMAGE_BYTES \
  --expected-input-sha256 IMAGE_SHA256 \
  --output /tmp/image_encoder.float16.onnx

# Reject bounded corrupt graphs and fixed-shape/dtype violations, then prove
# that both sessions still produce finite output. Run in a disposable process;
# this smoke is not a substitute for the production adapter's admission policy.
python onnx_split_failure_smoke.py \
  --image-model /tmp/image_encoder.float16.onnx \
  --image-model-bytes IMAGE_BYTES \
  --image-model-sha256 IMAGE_SHA256 \
  --text-model /tmp/text_encoder.float16.onnx \
  --text-model-bytes TEXT_BYTES \
  --text-model-sha256 TEXT_SHA256 \
  --tensors /tmp/prepared.npz \
  --expected-tensors-sha256 TENSOR_SHA256 \
  --threads 2

# Generate tiny parser-valid hostile graphs outside the repository, then load
# each in a bounded child process. The first release recommendation is still to
# reject ONNX external-data during release validation and use exact graph hashes.
python onnx_hostile_fixture_generate.py --output /tmp/hostile-fixtures
python onnx_hostile_model_smoke.py --fixtures /tmp/hostile-fixtures

# Linux-only bounded child proof for a graph requesting a 6 GB output. A
# passing result does not make process-wide RLIMIT_AS a production design.
python onnx_oversized_allocation_smoke.py \
  --fixtures /tmp/hostile-fixtures \
  --address-space-bytes 1073741824 \
  --timeout-seconds 15

# Bounded image-encoder load for a separate, constrained joint-load harness.
/tmp/foliopath-int001-onnx-export/bin/python onnx_image_backfill_load.py \
  --model /tmp/foliopath-int001-siglip2-split-onnx/image_encoder.onnx \
  --model-bytes 371682125 \
  --model-sha256 7fc85e0e8a0f4e5fce7be45c7830e7473c81f010633b7d31bc1613dafc734ab0 \
  --tensors /tmp/foliopath-int001-siglip2-prepared.npz \
  --expected-tensors-sha256 REVIEWED_SHA256 \
  --stop-file /tmp/foliopath-int001-stop
```

The semantic command defaults to the pinned SigLIP 2 candidate. Candidate
comparisons must also pass explicit `--model-id`, `--revision`,
`--expected-weight-bytes`, and `--expected-weight-sha256` values; the script
refuses to load a weight whose size or digest differs.

`semantic_onnx_export.py` downloads only the exact pinned source revision,
verifies the reviewed source weight before export, fixes exporter/runtime
versions and opset 18, validates the ONNX graph, and emits a deterministic
PyTorch reference fixture. `onnx_compare.py` checks all four exported outputs
against that fixture. Neither script makes the resulting 1.5 GB artifacts
eligible for Git or production distribution; project signing, SBOM, compliance,
native amd64 and full runtime failure testing remain separate gates.

The public pilot manifest records source revision, author, license, source hash,
download hash, dimensions and byte size. Images remain outside Git. Public
licensing does not remove personality-rights or privacy obligations, and this
pilot is not the representative 1,000-image acceptance dataset.

Dataset manifest schema v1 is now restricted to synthetic fixtures. Schema v2 is
the mandatory intake shape for every non-synthetic dataset, including face ground
truth. It records an allowlisted evaluation purpose and access role,
retention date, deterministic deletion procedure, opaque authorization/privacy
review references, and prohibited redistribution. The validator rejects treating
a public media license as biometric authority and requires opaque identity IDs for
verification/clustering data. The checked-in authorized-face file is a schema
template with placeholders only: it is neither real data nor proof of consent,
legal authority, privacy approval, or dataset quality. Restricted source files and
consent records must never be committed or emitted as CI artifacts.

`semantic_input_prepare.py` aligns its 512px WebP/quality-82 limits with the
current FolioPath grid-thumbnail contract and uses JPEG shrink-on-load, but its
Pillow implementation is only a macOS-development surrogate. It does not prove
the production libvips transform, memory envelope, or numerical equivalence.

`scan-models` accepts only catalogued regular files whose size and SHA-256
match. Schema v2 additionally requires an exact multi-file package directory:
missing, extra, changed or symlinked artifacts reject the whole package. It
rejects a symlink root, mount crossings, writable roots (by default), and entries
not present in the allowlist. Discovery is not an installer and does not make
arbitrary URLs or paths part of a public contract.

`go test -run TestFetchTrustedArtifactFailureMatrix ./...` exercises an
isolated, catalog-owned download state machine using only `httptest` and
temporary files. It covers pinned ETag resume, exact-origin redirect policy,
quota preflight, mid-stream cancellation followed by resume, wrong hash and
no-replace publication. Separate tests cover real partial-write and complete-
package `ENOSPC`, download restart, package rename/fsync strong-kill boundaries,
resolver error/empty-answer failure, signed-catalog verification and isolated
checkpoint/activation transactions. They do not cover production signing-key/
checkpoint operations, host power loss, real TLS/CDN rotation, outer retry
policy or native Linux/amd64. Set
`INT001_ENOSPC_DIR` only to a deliberately size-limited test filesystem to run
the real kernel-ENOSPC case.
`TestFetchTrustedArtifactStrongKillRecovery` starts and kills a real helper
process after partial persistence, then requires a new process to resume and
publish the exact digest; it is not a substitute for every container-kill and
database-activation fault point.
`newCatalogHTTPClient` resolves a reviewed hostname once, rejects an entire DNS
answer containing private or selected special-use addresses, pins dials to the
accepted addresses, preserves the TLS server name and disables environment
proxies. Each address gets at most five seconds, bounded further by the outer
deadline. Private-CA tests exercise TLS/SNI, unknown-CA rejection and fallback
inside one pinned address set. Test-only loopback and CA injection must never
be used by a production catalog.
`model-candidates-component.cdx.json` is a supplemental CycloneDX 1.6 inventory
for exact retained/held model artifacts that package scanners do not identify.
It marks its composition incomplete and SFace as a production hold. It is not
a redistribution approval, final model catalog, vulnerability report or signed
release SBOM.
`verifySignedCatalog` authenticates a bounded exact JSON payload and signed
metadata with Ed25519, then rejects invalid validity windows, rollback and
same-sequence equivocation. Ephemeral test keys do not prove production key
custody, rotation, revocation or durable checkpoint ownership.
`activationStore` is an isolated SQLite registry for already-published immutable
generations. It atomically advances the catalog checkpoint and active pointer;
an injected active-update failure rolls both back. This is not a production
migration and does not by itself solve the full filesystem/database lifecycle,
backup, quota or unavailable-state ownership. On Linux, `reconcileModelScan`
accepts only a completed kernel-anchored scan report: exact catalogued orphans become
available without activation; missing/corrupt generations become unavailable;
exact restoration becomes available again. It never deletes an unknown path or
changes the active pointer.
`TestDirectModelLifecycleRequiresReadOnlyMount` uses a real read-only bind mount
inside a disposable capability-gated test container. Direct provenance cannot
be relabelled as managed; disappearance/restoration changes availability only.
The temporary mount capability is test infrastructure, not a release-container
requirement.

Local macOS results are development evidence only. S0 still requires native
linux/amd64 and linux/arm64 runs under the documented 4 CPU/4 GiB envelope.

The separate root-module command `spikes/int001-vips-input` has a `libvips`
build-tagged implementation that calls FolioPath's production image adapter to
prepare the same public pilot. It is intended for controlled Linux development
containers only; the default non-libvips build does not process media. QEMU
results do not satisfy native dual-architecture evidence. The tool requires an
explicit `--execution-mode native|qemu`; it records the compiled Go architecture
and refuses an omitted or unknown mode so evidence cannot silently mislabel an
emulated run as native.
