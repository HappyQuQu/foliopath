#!/bin/sh
set -eu

image=${1:?usage: generate-supply-chain-evidence.sh IMAGE SBOM_DIRECTORY NOTICES_DIRECTORY OUTPUT}
sbom_dir=${2:?usage: generate-supply-chain-evidence.sh IMAGE SBOM_DIRECTORY NOTICES_DIRECTORY OUTPUT}
notices_dir=${3:?usage: generate-supply-chain-evidence.sh IMAGE SBOM_DIRECTORY NOTICES_DIRECTORY OUTPUT}
output=${4:?usage: generate-supply-chain-evidence.sh IMAGE SBOM_DIRECTORY NOTICES_DIRECTORY OUTPUT}

source_commit=${FOLIOPATH_SUPPLY_CHAIN_SOURCE_COMMIT:?FOLIOPATH_SUPPLY_CHAIN_SOURCE_COMMIT is required}
expected_arch=${FOLIOPATH_SUPPLY_CHAIN_EXPECTED_ARCH:?FOLIOPATH_SUPPLY_CHAIN_EXPECTED_ARCH is required}
run_id=${FOLIOPATH_SUPPLY_CHAIN_RUN_ID:?FOLIOPATH_SUPPLY_CHAIN_RUN_ID is required}
run_attempt=${FOLIOPATH_SUPPLY_CHAIN_RUN_ATTEMPT:?FOLIOPATH_SUPPLY_CHAIN_RUN_ATTEMPT is required}

image_spdx="${sbom_dir}/image.spdx.json"
npm_spdx="${sbom_dir}/npm.spdx.json"
source_spdx="${sbom_dir}/source.spdx.json"
vulnerability_report="${sbom_dir}/vulnerabilities.json"
vulnerability_summary="${sbom_dir}/vulnerability-summary.json"
notices_sums="${notices_dir}/SHA256SUMS"

for required in \
  "${image_spdx}" \
  "${npm_spdx}" \
  "${source_spdx}" \
  "${vulnerability_report}" \
  "${vulnerability_summary}" \
  "${notices_sums}"; do
  test -s "${required}"
done

image_os=$(docker image inspect "${image}" --format '{{.Os}}')
image_arch=$(docker image inspect "${image}" --format '{{.Architecture}}')
image_digest=$(docker image inspect "${image}" --format '{{.Id}}')
image_size=$(docker image inspect "${image}" --format '{{.Size}}')
test "${image_os}" = "linux"
test "${image_arch}" = "${expected_arch}"

jq -e '
  .total == 0 and
  .critical == 0 and
  .high == 0 and
  .fixedAvailable == 0
' "${vulnerability_summary}" >/dev/null

glib_version=$(jq -er '
  [.packages[]? | select(.name == "foliopath-glib") | .versionInfo] as $versions |
  if ($versions | length) == 1 then $versions[0] else empty end
' "${image_spdx}")
test "${glib_version}" = "2.88.3-1"

banned_package_count=$(jq '
  [
    .packages[]? |
    select(
      .name == "libblkid1" or
      .name == "libglib2.0-0t64" or
      .name == "libmount1" or
      .name == "libselinux1"
    )
  ] |
  length
' "${image_spdx}")
test "${banned_package_count}" -eq 0

hash_file() {
  sha256sum "$1" | cut -d ' ' -f 1
}

output_dir=$(dirname -- "${output}")
mkdir -p "${output_dir}"
output_tmp=$(mktemp "${output_dir}/supply-chain-evidence.XXXXXX")

jq -n \
  --arg release "MVP-2026-07-23" \
  --arg source_commit "${source_commit}" \
  --arg architecture "${image_arch}" \
  --arg os "${image_os}" \
  --arg image_digest "${image_digest}" \
  --argjson image_size_bytes "${image_size}" \
  --arg image_spdx_sha256 "$(hash_file "${image_spdx}")" \
  --arg npm_spdx_sha256 "$(hash_file "${npm_spdx}")" \
  --arg source_spdx_sha256 "$(hash_file "${source_spdx}")" \
  --arg vulnerability_report_sha256 "$(hash_file "${vulnerability_report}")" \
  --arg vulnerability_summary_sha256 "$(hash_file "${vulnerability_summary}")" \
  --arg notices_sha256 "$(hash_file "${notices_sums}")" \
  --arg glib_package_version "${glib_version}" \
  --argjson banned_package_count "${banned_package_count}" \
  --arg run_id "${run_id}" \
  --argjson run_attempt "${run_attempt}" \
  --arg created_at "$(date -u '+%Y-%m-%dT%H:%M:%SZ')" \
  '{
    schemaVersion: 1,
    release: $release,
    sourceCommit: $source_commit,
    architecture: $architecture,
    os: $os,
    imageDigest: $image_digest,
    imageSizeBytes: $image_size_bytes,
    sbom: {
      imageSHA256: $image_spdx_sha256,
      npmSHA256: $npm_spdx_sha256,
      sourceSHA256: $source_spdx_sha256
    },
    vulnerabilityScan: {
      policy: "all",
      total: 0,
      critical: 0,
      high: 0,
      reportSHA256: $vulnerability_report_sha256,
      summarySHA256: $vulnerability_summary_sha256
    },
    noticesSHA256: $notices_sha256,
    glibPackageVersion: $glib_package_version,
    bannedPackageCount: $banned_package_count,
    workflowRunId: $run_id,
    workflowRunAttempt: $run_attempt,
    createdAt: $created_at,
    result: "passed"
  }' >"${output_tmp}"

mv -- "${output_tmp}" "${output}"
jq -e '.result == "passed"' "${output}" >/dev/null
printf '%s\n' "supply-chain evidence generated for ${image_arch}"
