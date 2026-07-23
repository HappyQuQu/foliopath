#!/bin/sh
set -eu

image=${1:?usage: generate-sbom.sh IMAGE OUTPUT_DIRECTORY}
output_dir=${2:?usage: generate-sbom.sh IMAGE OUTPUT_DIRECTORY}
syft_image="anchore/syft:v1.44.0@sha256:86fde6445b483d902fe011dd9f68c4987dd94e07da1e9edc004e3c2422650de6"

mkdir -p "$output_dir"

docker run --rm \
  --mount "type=bind,src=$PWD,dst=/src,readonly" \
  "$syft_image" dir:/src -o spdx-json >"$output_dir/source.spdx.json"

(
  cd web
  npm sbom --package-lock-only --sbom-format spdx
) >"$output_dir/npm.spdx.json"

docker run --rm \
  --mount type=bind,src=/var/run/docker.sock,dst=/var/run/docker.sock \
  "$syft_image" "$image" -o spdx-json >"$output_dir/image.spdx.json"

for sbom in "$output_dir"/*.spdx.json; do
  jq -e '.spdxVersion == "SPDX-2.3"' "$sbom" >/dev/null
done

sha256sum "$output_dir"/*.spdx.json
