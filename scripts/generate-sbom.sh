#!/bin/sh
set -eu

image=${1:?usage: generate-sbom.sh IMAGE OUTPUT_DIRECTORY}
output_dir=${2:?usage: generate-sbom.sh IMAGE OUTPUT_DIRECTORY}
syft_image="anchore/syft:v1.44.0@sha256:86fde6445b483d902fe011dd9f68c4987dd94e07da1e9edc004e3c2422650de6"

mkdir -p "$output_dir"

docker run --rm \
  --mount "type=bind,src=$PWD,dst=/src,readonly" \
  "$syft_image" dir:/src \
  --exclude './.git/**' \
  --exclude './.github/**' \
  --exclude './.artifacts/**' \
  --exclude './build/**' \
  --exclude './data/**' \
  --exclude './foliopath-data/**' \
  --exclude './internal/webassets/dist/**' \
  --exclude './playwright-report/**' \
  --exclude './prototypes/**' \
  --exclude './test-results/**' \
  --exclude './web/coverage/**' \
  --exclude './web/node_modules/**' \
  --exclude './web/qa/**' \
  --exclude './web/storybook-static/**' \
  --exclude '**/*.db' \
  --exclude '**/*.db-shm' \
  --exclude '**/*.db-wal' \
  --exclude '**/*.log' \
  -o spdx-json >"$output_dir/source.spdx.json"

(
  cd web
  npm sbom --package-lock-only --sbom-format spdx
) >"$output_dir/npm.spdx.json"

docker run --rm \
  --mount type=bind,src=/var/run/docker.sock,dst=/var/run/docker.sock \
  "$syft_image" "$image" -o spdx-json >"$output_dir/image.spdx.json"

normalize_spdx() {
  sbom=$1
  kind=$2
  normalized="${sbom}.normalized"
  content="${sbom}.content"
  jq 'del(.creationInfo.created, .documentNamespace)' "$sbom" >"$content"
  content_digest=$(sha256sum "$content" | cut -d ' ' -f 1)
  jq \
    --arg namespace \
      "https://foliopath.local/spdx/${kind}/${content_digest}" \
    '.creationInfo.created = "1970-01-01T00:00:00Z" |
     .documentNamespace = $namespace' \
    "$content" >"$normalized"
  mv "$normalized" "$sbom"
  rm "$content"
}

normalize_spdx "$output_dir/source.spdx.json" source
normalize_spdx "$output_dir/npm.spdx.json" npm
normalize_spdx "$output_dir/image.spdx.json" image

for sbom in "$output_dir"/*.spdx.json; do
  jq -e '.spdxVersion == "SPDX-2.3"' "$sbom" >/dev/null
done

sha256sum "$output_dir"/*.spdx.json
