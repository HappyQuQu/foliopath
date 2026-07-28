#!/bin/sh
set -eu

image=${1:?usage: scan-release-image.sh IMAGE OUTPUT_DIRECTORY}
output_dir=${2:?usage: scan-release-image.sh IMAGE OUTPUT_DIRECTORY}
policy=${FOLIOPATH_VULNERABILITY_POLICY:-fixed}
trivy_image="aquasec/trivy:0.70.0@sha256:be1190afcb28352bfddc4ddeb71470835d16462af68d310f9f4bca710961a41e"

case "${policy}" in
all | fixed | report)
	;;
*)
	echo "FOLIOPATH_VULNERABILITY_POLICY must be all, fixed, or report" >&2
	exit 2
	;;
esac

mkdir -p "${output_dir}" "${output_dir}/.trivy-cache"
output_dir=$(CDPATH= cd -- "${output_dir}" && pwd)
report="${output_dir}/vulnerabilities.json"
summary="${output_dir}/vulnerability-summary.json"

docker run --rm \
	--mount type=bind,src=/var/run/docker.sock,dst=/var/run/docker.sock \
	--mount "type=bind,src=${output_dir},dst=/reports" \
	--mount "type=bind,src=${output_dir}/.trivy-cache,dst=/cache" \
	"${trivy_image}" image \
	--cache-dir /cache \
	--skip-version-check \
	--scanners vuln \
	--severity HIGH,CRITICAL \
	--format json \
	--output /reports/vulnerabilities.json \
	"${image}"

jq '
	[.Results[]?.Vulnerabilities[]?] as $findings |
	{
		scannedAt: .CreatedAt,
		schemaVersion: .SchemaVersion,
		total: ($findings | length),
		uniqueVulnerabilities: ($findings | map(.VulnerabilityID) | unique | length),
		critical: ($findings | map(select(.Severity == "CRITICAL")) | length),
		high: ($findings | map(select(.Severity == "HIGH")) | length),
		fixedAvailable: (
			$findings |
			map(select((.FixedVersion // "") != "")) |
			length
		),
		byPackage: (
			$findings |
			map(.PkgName) |
			group_by(.) |
			map({package: .[0], count: length}) |
			sort_by(-.count)
		)
	}
' "${report}" >"${summary}"

jq . "${summary}"
sha256sum "${report}" "${summary}"

case "${policy}" in
all)
	blocking=$(jq '.total' "${summary}")
	;;
fixed)
	blocking=$(jq '.fixedAvailable' "${summary}")
	;;
report)
	blocking=0
	;;
esac

if [ "${blocking}" -ne 0 ]; then
	echo "vulnerability policy ${policy} rejected ${blocking} finding(s)" >&2
	jq -r '
		.Results[]?.Vulnerabilities[]? |
		select(
			("'"${policy}"'" == "all") or
			((.FixedVersion // "") != "")
		) |
		[
			.Severity,
			.VulnerabilityID,
			.PkgName,
			.InstalledVersion,
			(.FixedVersion // "unavailable")
		] |
		@tsv
	' "${report}" >&2
	exit 1
fi
