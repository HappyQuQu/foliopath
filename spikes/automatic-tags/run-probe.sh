#!/usr/bin/env bash
set -euo pipefail
repo=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
model=/Volumes/extended/foliopath/model-experiments/joytag
image=sha256:31484348fd97a6cfefd01a83dfe3104f22ba9e4d23e92c1e1000d62774c30689
[[ -d "$model" && ! -L "$model" ]] || { echo "Download the pinned experiment first." >&2; exit 2; }
# Exact existing local image only. No pulls, new deployment, ports, production cache,
# media mount or database. Only read-only model/script/synthetic fixture mounts.
exec docker run --rm --pull never --network none --read-only --cap-drop ALL \
  --security-opt no-new-privileges --memory 2304m --memory-swap 2304m \
  --cpus 2 --pids-limit 128 --env PYTHONDONTWRITEBYTECODE=1 \
  --env HF_HUB_OFFLINE=1 --env HF_HUB_DISABLE_TELEMETRY=1 \
  --mount "type=bind,src=$model,dst=/model,readonly" \
  --mount "type=bind,src=$repo/spikes/automatic-tags,dst=/probe,readonly" \
  --mount "type=bind,src=$repo/prototypes/apple-redesign/assets/ai-media,dst=/repo/prototypes/apple-redesign/assets/ai-media,readonly" \
  --mount "type=bind,src=$repo/tests/fixtures/ai-faces,dst=/repo/tests/fixtures/ai-faces,readonly" \
  --mount "type=bind,src=$repo/tests/fixtures/media,dst=/repo/tests/fixtures/media,readonly" \
  --entrypoint python "$image" /probe/probe.py "$@"
