#!/bin/sh
set -eu

image=${1:?usage: generate-provenance.sh IMAGE OUTPUT}
output=${2:?usage: generate-provenance.sh IMAGE OUTPUT}
builder_id=${FOLIOPATH_PROVENANCE_BUILDER_ID:?FOLIOPATH_PROVENANCE_BUILDER_ID is required}
invocation_id=${FOLIOPATH_PROVENANCE_INVOCATION_ID:?FOLIOPATH_PROVENANCE_INVOCATION_ID is required}

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "${repo_root}"

if [ -n "$(git status --porcelain=v1 --untracked-files=all)" ]; then
  echo "provenance requires a clean source tree" >&2
  exit 1
fi

source_commit=$(git rev-parse HEAD)
source_uri=$(git remote get-url origin)
dockerfile_digest=$(sha256sum Dockerfile | cut -d ' ' -f 1)
image_id=$(docker image inspect "${image}" --format '{{.Id}}')
image_digest=${image_id#sha256:}
image_os=$(docker image inspect "${image}" --format '{{.Os}}')
image_arch=$(docker image inspect "${image}" --format '{{.Architecture}}')
image_size=$(docker image inspect "${image}" --format '{{.Size}}')
created_at=$(date -u '+%Y-%m-%dT%H:%M:%SZ')
output_dir=$(dirname -- "${output}")
mkdir -p "${output_dir}"
temporary=$(mktemp "${output_dir}/provenance.XXXXXX")
trap 'rm -f "${temporary}"' EXIT HUP INT TERM

jq -n \
  --arg image "${image}" \
  --arg image_digest "${image_digest}" \
  --arg source_uri "${source_uri}" \
  --arg source_commit "${source_commit}" \
  --arg dockerfile_digest "${dockerfile_digest}" \
  --arg builder_id "${builder_id}" \
  --arg invocation_id "${invocation_id}" \
  --arg created_at "${created_at}" \
  --arg os "${image_os}" \
  --arg architecture "${image_arch}" \
  --argjson image_size_bytes "${image_size}" \
  '{
    "_type": "https://in-toto.io/Statement/v1",
    "subject": [{
      "name": $image,
      "digest": {"sha256": $image_digest}
    }],
    "predicateType": "https://slsa.dev/provenance/v1",
    "predicate": {
      "buildDefinition": {
        "buildType": "https://github.com/HappyQuQu/foliopath/docker-build@v1",
        "externalParameters": {
          "os": $os,
          "architecture": $architecture
        },
        "internalParameters": {
          "dockerfile": "Dockerfile"
        },
        "resolvedDependencies": [
          {
            "uri": ("git+" + $source_uri),
            "digest": {"gitCommit": $source_commit}
          },
          {
            "uri": "file:./Dockerfile",
            "digest": {"sha256": $dockerfile_digest}
          }
        ]
      },
      "runDetails": {
        "builder": {"id": $builder_id},
        "metadata": {
          "invocationId": $invocation_id,
          "startedOn": $created_at,
          "finishedOn": $created_at
        },
        "byproducts": [{
          "name": "image-size-bytes",
          "content": $image_size_bytes
        }]
      }
    }
  }' >"${temporary}"

jq -e \
  '._type == "https://in-toto.io/Statement/v1" and
   .predicateType == "https://slsa.dev/provenance/v1" and
   (.subject[0].digest.sha256 | test("^[0-9a-f]{64}$"))' \
  "${temporary}" >/dev/null
mv "${temporary}" "${output}"
trap - EXIT HUP INT TERM
sha256sum "${output}"
