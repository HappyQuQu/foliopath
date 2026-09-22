# FolioPath AI candidate image

ADR-0020 selects a thin project-owned service over a fixed Immich ML runtime.
This is a local candidate, not release approval. The default selects
`ViT-B-32__openai`; `--build-arg FOLIOPATH_INCLUDE_FACE=1` additionally selects
`buffalo_l` for the private face-v1 endpoints. Neither implements the old SigLIP contract.
The face endpoints are connected to the optional main-application face jobs;
functional evidence does not establish model quality or release approval.
See the [contract and verification boundary](../../docs/evidence/int-001/face-service-contract-2026-09-15.md).
Changing this selection or wrapper code changes compatibility identity; never reuse
the previous deployment identity. Keep experiments under a separate image tag.

Face verification (local Docker, no real media):

```sh
docker build --pull=false --build-arg FOLIOPATH_INCLUDE_FACE=1 -f services/inference/Dockerfile -t foliopath-inference:face-candidate .
docker run --rm --network none --entrypoint python foliopath-inference:face-candidate -m unittest discover -s /opt/foliopath-inference -p 'test*.py' -v
docker run --rm --network none --read-only --tmpfs /tmp --tmpfs /verified-cache:size=1200m --mount type=bind,src=/Volumes/extended/foliopath/model-cache,dst=/cache,readonly --entrypoint python foliopath-inference:face-candidate -m probe_face
```

The embedded manifest selects the copied CLIP cache baseline. Startup verifies
each pinned file and copies it to private tmpfs; extra source files are not exposed
to inference. Missing, changed and symlinked files fail startup. The baseline is
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
