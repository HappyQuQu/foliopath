# FolioPath AI candidate image

## Reviewed localized photo policy (ADR-0026)

`Dockerfile.photo-tags` extends an independently verified local multilingual image
without overwriting legacy semantic/face/runtime sources. Its `photo_runtime` starts
the tag-only overlay, sharing the original scheduler and bounded child. The child
reuses `inference_worker` transforms/ORT and substitutes only `photo_policy` output
filtering. Non-tag requests exec the unchanged multilingual/legacy worker.

The single category source is `internal/curation/tag_categories.json`: 153 reviewed
classes, including the original 92, with Chinese/English display and protection
aliases owned by curation. Translation is presentation; output keys remain stable.
Allowed keys bind the new policy digest and overlay execution binds the tag runtime.
Model bytes and cache digest stay unchanged. Ordinary image accuracy is not proven.

Read this profile's expected **tag** identity offline with `python -m photo_policy`.
`image_identity --tag` deliberately still describes the legacy 92-class contract;
it is not the selected photo-overlay identity. Semantic identity stays available
from `python -m multilingual_contract`, face from the unchanged legacy manifest.
Update only the separately pinned `tag.identity` after verification; do not relabel
old results or restart ongoing real-library work as a test. This pairing has local
ARM64 synthetic evidence, not English-512, AMD64, small-budget or release approval.
See [CR-2026-033 evidence](../../docs/changes/CR-2026-033-localized-photo-tags.md).

ADR-0020 selects a thin project-owned service over a fixed Immich ML runtime.
This is a local candidate, not release approval. The default selects
`ViT-B-32__openai`; `--build-arg FOLIOPATH_INCLUDE_FACE=1` additionally selects
`buffalo_l` for the private face-v1 endpoints. Neither implements the old SigLIP contract.
The face endpoints are connected to the optional main-application face jobs;
functional evidence does not establish model quality or release approval.
See the [contract and verification boundary](../../docs/evidence/int-001/face-service-contract-2026-09-15.md).
Changing this selection or wrapper code changes compatibility identity; never reuse
the previous deployment identity. Keep experiments under a separate image tag.

## Bounded inference and automatic tags (ADR-0021)

The optional `--build-arg FOLIOPATH_INCLUDE_TAG=1` adds the fixed JoyTag contract,
not model weights or downloads. Its three exact pinned files must exist under
`/cache/tagging/JoyTag/`. The ordinary image leaves this capability disabled.
The canonical allowlist, model digests, threshold and preprocessing version are
owned by `tag_contract.py`; the 92 allowed object/scene/action/background classes
are classifier output filtering, not user-provided CLIP candidates. Names are
stable English class IDs independent of UI language. Scores are uncalibrated
sigmoid outputs; threshold 0.4 and at most 20 results are an explicit initial policy,
not a claimed photo-quality acceptance. No sensitive identity classes are emitted.

`GET /api/v1/tag/identity` returns version, protocol (`foliopath-tag-v1`), model
(`JoyTag`), model SHA256, policy SHA256, runtime contract SHA256 and a tag-only
cache-manifest SHA256. `POST /api/v1/tag/predict` accepts exactly one multipart
`image`: at most 8 MiB, single-frame JPEG/PNG/WebP, each edge at most 1600 pixels.
It returns `{"labels":[{"name":"flower","confidence":0.57}]}`; empty is valid.
No candidate words, user thresholds or arbitrary model selection are accepted.
Tag responses carry `X-FolioPath-Tag-Model`, `X-FolioPath-Tag-Policy`,
`X-FolioPath-Tag-Cache` and `X-FolioPath-Runtime`. The legacy `X-FolioPath-Cache`
still identifies the CLIP/face manifest and must not be used to verify tag output.

The one HTTP scheduler admits at most one running child and one waiting request.
Every request has a 26-second total deadline including waiting, verified copying,
loading and inference. The third request receives `503 busy`; invalid/unavailable
outputs fail closed. Each child loads only its own model family; completion,
failure, cancellation and timeout kill/reap it before admitting another child.
The parent owns and then deletes that request's private tmpfs directory. Native
ORT memory is released by process exit, not an assumption about Python GC.

Startup verifies all accepted manifest bytes, releasing temporary snapshots after
verification. Before each inference the child re-verifies and copies its complete
fixed model subset to private tmpfs; it never infers directly from mutable source
paths. All subset files preserve the original manifest path/hash; shared CLIP
configuration is retained. Identity stays bound to the complete accepted manifests.
This avoids keeping all three families' private snapshots in RAM at once. The
tradeoff is a cold verification/copy/load on every request, currently around 2–3
seconds in the recorded ARM64 synthetic run, not a throughput SLA.

Build, export identities offline, and reproduce the isolated budget verification:

```sh
docker build --pull=false --build-arg FOLIOPATH_INCLUDE_FACE=1 \
  --build-arg FOLIOPATH_INCLUDE_TAG=1 -f services/inference/Dockerfile \
  -t foliopath-inference:automatic-tags-candidate .
docker run --rm --network none --read-only --entrypoint python \
  foliopath-inference:automatic-tags-candidate /opt/foliopath-inference/image_identity.py
docker run --rm --network none --read-only --entrypoint python \
  foliopath-inference:automatic-tags-candidate /opt/foliopath-inference/image_identity.py --tag
PYTHONDONTWRITEBYTECODE=1 python3 spikes/automatic-tags/run-scheduled.py
```

The host launcher stages a **separate** 52-file cache for synthetic tests; it does
not alter active model caches. See [same-budget evidence](../../spikes/automatic-tags/SCHEDULED-RESULT.md)
for 28 passing unit tests (including real TCP disconnection), three full inference cycles, maximum-edge inputs,
overload admission, exact image/identities and remaining limitations. Original
media, the existing 6010 application and server222 were not used or modified.

Face verification (local Docker, no real media):

```sh
docker build --pull=false --build-arg FOLIOPATH_INCLUDE_FACE=1 -f services/inference/Dockerfile -t foliopath-inference:face-candidate .
docker run --rm --network none --entrypoint python foliopath-inference:face-candidate -m unittest discover -s /opt/foliopath-inference -p 'test*.py' -v
docker run --rm --network none --read-only --tmpfs /tmp --tmpfs /verified-cache:size=1200m --mount type=bind,src=/Volumes/extended/foliopath/model-cache,dst=/cache,readonly --entrypoint python foliopath-inference:face-candidate -m probe_face
```

The embedded manifest selects the copied CLIP cache baseline. Startup verifies
each pinned file through a temporary private snapshot; request-scoped children
re-verify their fixed subsets before inference. Extra source files are not exposed
to inference. Missing, changed and symlinked files fail startup or the subsequent
request if the source changes after startup. The baseline is
local byte evidence, not upstream provenance. The standalone Compose uses the
P4 small-sample AI budget of 2304 MiB including its private snapshot (application
1024 MiB separately). Older probes below use a 4 GiB AI-only limit; neither is
whole-product large-capacity proof.

Build and verify locally:

```sh
docker build -f services/inference/Dockerfile -t foliopath-inference:candidate .
docker run --rm --network none --read-only --tmpfs /tmp:size=134217728 \
  --entrypoint python foliopath-inference:candidate \
  -m unittest discover -s /opt/foliopath-inference -p 'test_*.py' -v
export FOLIOPATH_TEST_MODEL_CACHE=/Volumes/extended/foliopath/model-cache
export FOLIOPATH_TEST_ML_IDENTITY="$(docker run --rm --network none --read-only \
  --entrypoint python foliopath-inference:candidate /opt/foliopath-inference/image_identity.py)"
docker compose -f spikes/int001-remote-ml/compose.yaml \
  -f spikes/int001-remote-ml/compose.foliopath.yaml up -d --wait machine-learning
docker compose -f spikes/int001-remote-ml/compose.yaml \
  -f spikes/int001-remote-ml/compose.foliopath.yaml run --rm encoded-probe
docker compose -f spikes/int001-remote-ml/compose.yaml \
  -f spikes/int001-remote-ml/compose.foliopath.yaml run --rm persisted-probe
```

Build `foliopath-remote-ml-regression:local` from the root Dockerfile first for Go
probes. Expected identity comes from the image, not trust-on-first-use HTTP.
The persisted probe also processes four generated WebP storyboard frames, commits
the full video plan, and checks that an incomplete plan cannot be searched. It does
not test actual video decoding or production activation.
The runtime contract hash covers the base digest and adapter sources; it is NOT
the final OCI image digest. Real deployment must also pin the delivered image.
No model download, database, original media mount, Docker socket or public AI port.
These historical probes do not launch the application. The standalone two-service
deployment entry point is now [compose.ai.yaml](../../compose.ai.yaml); see the
[deployment guide](../../docs/ai-compose.md). Release approval remains separate.
