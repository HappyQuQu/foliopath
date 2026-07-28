#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
. "${repo_root}/tests/release/http_client.sh"
full_capacity=${FOLIOPATH_CAPACITY:-0}
enforce_budget=${FOLIOPATH_CAPACITY_ENFORCE_BUDGET:-0}
scan_budget_ms=${FOLIOPATH_CAPACITY_SCAN_BUDGET_MS:-240000}

case "${full_capacity}:${enforce_budget}" in
0:0 | 0:1 | 1:0 | 1:1)
	;;
*)
	echo "FOLIOPATH_CAPACITY and FOLIOPATH_CAPACITY_ENFORCE_BUDGET must be 0 or 1" >&2
	exit 2
	;;
esac

if [ "${full_capacity}" = "1" ]; then
	directory_count=${FOLIOPATH_CAPACITY_DIRS:-10000}
	asset_count=${FOLIOPATH_CAPACITY_ASSETS:-100000}
	timeout_seconds=${FOLIOPATH_CAPACITY_TIMEOUT_SECONDS:-900}
else
	directory_count=${FOLIOPATH_CAPACITY_DIRS:-100}
	asset_count=${FOLIOPATH_CAPACITY_ASSETS:-1000}
	timeout_seconds=${FOLIOPATH_CAPACITY_TIMEOUT_SECONDS:-180}
fi

case "${directory_count}:${asset_count}:${timeout_seconds}:${scan_budget_ms}" in
*[!0-9:]* | 0:* | *:0:* | *:0)
	echo "capacity counts, timeout, and scan budget must be positive integers" >&2
	exit 2
	;;
esac

if [ -n "${FOLIOPATH_CAPACITY_WORK_ROOT:-}" ]; then
	mkdir -p "${FOLIOPATH_CAPACITY_WORK_ROOT}"
	probe_root="${FOLIOPATH_CAPACITY_WORK_ROOT}/foliopath-release-capacity-$$"
	mkdir "${probe_root}"
else
	probe_root=$(mktemp -d "${TMPDIR:-/tmp}/foliopath-release-capacity.XXXXXX")
fi

image_owned=0
if [ -n "${FOLIOPATH_RELEASE_IMAGE:-}" ]; then
	image=${FOLIOPATH_RELEASE_IMAGE}
	build_image=${FOLIOPATH_CAPACITY_BUILD_IMAGE:-0}
else
	image="foliopath:stage5-capacity-local-$$"
	build_image=1
	image_owned=1
fi

container="foliopath-stage5-capacity-$$"
http_client="foliopath-stage5-capacity-http-client-$$"
data_root="${probe_root}/data"
media_root="${probe_root}/library"
metrics_root="${probe_root}/metrics"

cleanup() {
	docker rm --force "${http_client}" >/dev/null 2>&1 || true
	docker rm --force "${container}" >/dev/null 2>&1 || true
	if [ "${image_owned}" = "1" ]; then
		docker image rm --force "${image}" >/dev/null 2>&1 || true
	fi
	chmod -R u+w "${probe_root}" >/dev/null 2>&1 || true
	rm -rf -- "${probe_root}"
}
trap cleanup EXIT HUP INT TERM

mkdir -p "${data_root}" "${media_root}" "${metrics_root}"
chmod 0777 "${data_root}"

if [ "${build_image}" = "1" ]; then
	docker build \
		--file "${repo_root}/Dockerfile" \
		--tag "${image}" \
		--build-arg VERSION=stage5-capacity \
		"${repo_root}"
else
	docker image inspect "${image}" >/dev/null
fi

build_release_http_client "${repo_root}"
go run "${repo_root}/tests/fixtures/capacitygen" \
	"${media_root}" "${directory_count}" "${asset_count}" \
	>"${metrics_root}/fixture.json"
sentinel="${media_root}/group-000/asset-000000.png"
sentinel_before=$(sha256sum "${sentinel}" | cut -d ' ' -f 1)

docker run --detach \
	--name "${container}" \
	--cpus 4 \
	--memory 4g \
	--read-only \
	--cap-drop ALL \
	--security-opt no-new-privileges:true \
	--tmpfs /tmp:rw,noexec,nosuid,size=256m,uid=65532,gid=65532,mode=0700 \
	--mount "type=bind,src=${data_root},dst=/app/data" \
	--mount "type=bind,src=${media_root},dst=/library,readonly" \
	"${image}" >/dev/null

deadline=$(( $(date +%s) + 60 ))
while [ "$(date +%s)" -lt "${deadline}" ]; do
	status=$(docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{end}}' "${container}")
	test "${status}" != "unhealthy" || break
	test "${status}" != "healthy" || break
	sleep 1
done
if [ "$(docker inspect --format '{{.State.Health.Status}}' "${container}")" != "healthy" ]; then
	docker logs "${container}"
	echo "capacity candidate did not become healthy" >&2
	exit 1
fi

start_release_http_client "${http_client}" "${container}"
setup_response=$(docker exec "${http_client}" curl --fail --silent --show-error \
	--cookie-jar /tmp/cookies \
	--header 'Content-Type: application/json' \
	--header 'Origin: http://127.0.0.1:8080' \
	--data '{"username":"CapacityAdmin","displayName":"Capacity Admin","password":"correct horse battery staple"}' \
	http://127.0.0.1:8080/api/v1/auth/setup)
csrf_token=$(printf '%s' "${setup_response}" | jq -er '.csrfToken')

scan_started=$(date +%s)
create_response=$(docker exec "${http_client}" curl --fail --silent --show-error \
	--cookie /tmp/cookies \
	--header 'Content-Type: application/json' \
	--header "X-CSRF-Token: ${csrf_token}" \
	--header 'Idempotency-Key: stage5-capacity-library' \
	--data '{"name":"Capacity","rootPath":""}' \
	http://127.0.0.1:8080/api/v1/libraries)
library_id=$(printf '%s' "${create_response}" | jq -er '.library.id')
scan_id=$(printf '%s' "${create_response}" | jq -er '.scan.id')

deadline=$(( scan_started + timeout_seconds ))
while [ "$(date +%s)" -lt "${deadline}" ]; do
	library_status=$(docker exec "${http_client}" curl --silent --show-error \
		--output /tmp/library-response.json \
		--write-out '%{http_code}' \
		--cookie /tmp/cookies \
		"http://127.0.0.1:8080/api/v1/libraries/${library_id}")
	library_response=$(docker exec "${http_client}" cat /tmp/library-response.json)
	if [ "${library_status}" != "200" ]; then
		printf '%s\n' "${library_response}" >&2
		docker logs --tail 200 "${container}" >&2
		echo "capacity library probe returned HTTP ${library_status}" >&2
		exit 1
	fi
	status=$(printf '%s' "${library_response}" | jq -er '.status')
	case "${status}" in
	ready)
		break
		;;
	offline | error)
		printf '%s\n' "${library_response}" >&2
		echo "capacity scan entered ${status}" >&2
		exit 1
		;;
	esac
	sleep 1
done
scan_finished=$(date +%s)
if [ "${status}" != "ready" ]; then
	docker logs "${container}"
	echo "capacity scan did not finish within ${timeout_seconds}s" >&2
	exit 1
fi

test "$(printf '%s' "${library_response}" | jq -er '.directoryCount')" -eq $((directory_count + 1))
test "$(printf '%s' "${library_response}" | jq -er '.assetCount')" -eq "${asset_count}"

browse_url="http://127.0.0.1:8080/api/v1/libraries/${library_id}/assets?recursive=true&limit=100"
search_url="http://127.0.0.1:8080/api/v1/libraries/${library_id}/assets?q=asset-000&limit=100"
global_url="http://127.0.0.1:8080/api/v1/assets?q=asset-000&limit=100"

for named_url in "browse:${browse_url}" "search:${search_url}" "global:${global_url}"; do
	name=${named_url%%:*}
	url=${named_url#*:}
	response_status=$(docker exec "${http_client}" curl --silent --show-error \
		--output /tmp/capacity-response.json \
		--write-out '%{http_code}' \
		--cookie /tmp/cookies "${url}")
	response=$(docker exec "${http_client}" cat /tmp/capacity-response.json)
	if [ "${response_status}" != "200" ]; then
		printf '%s\n' "${response}" >&2
		docker logs --tail 200 "${container}" >&2
		echo "${name} capacity probe returned HTTP ${response_status}" >&2
		exit 1
	fi
	test "$(printf '%s' "${response}" | jq -er '.items | length')" -gt 0
	latency_file="${metrics_root}/${name}.seconds"
	if ! docker exec "${http_client}" sh -c '
		url=$1
		samples=$2
		index=0
		while [ "${index}" -lt "${samples}" ]; do
			result=$(curl --silent --show-error --output /tmp/capacity-sample.json \
				--write-out "%{http_code} %{time_total}" \
				--cookie /tmp/cookies "${url}") || exit 1
			status=${result%% *}
			test "${status}" = "200" || {
				cat /tmp/capacity-sample.json >&2
				exit 1
			}
			printf "%s\n" "${result#* }"
			index=$((index + 1))
		done
	' capacity-probe "${url}" 30 >"${latency_file}"; then
		docker inspect --format \
			'state={{.State.Status}} exit={{.State.ExitCode}} oom={{.State.OOMKilled}}' \
			"${container}" >&2 || true
		docker exec "${http_client}" sh -c \
			'printf "memory.current="; cat /proc/1/root/sys/fs/cgroup/memory.current; printf "memory.peak="; cat /proc/1/root/sys/fs/cgroup/memory.peak' \
			>&2 || true
		docker logs --tail 200 "${container}" >&2 || true
		echo "${name} repeated capacity probe failed" >&2
		exit 1
	fi
done

p95_ms() {
	file=$1
	sort -n "${file}" | awk 'NR == 29 { printf "%.3f", $1 * 1000 }'
}

browse_p95_ms=$(p95_ms "${metrics_root}/browse.seconds")
search_p95_ms=$(p95_ms "${metrics_root}/search.seconds")
global_p95_ms=$(p95_ms "${metrics_root}/global.seconds")
memory_peak_bytes=$(docker exec "${http_client}" sh -c \
	'cat /proc/1/root/sys/fs/cgroup/memory.peak 2>/dev/null || cat /proc/1/root/sys/fs/cgroup/memory.current')
database_bytes=$(find "${data_root}" -maxdepth 1 -type f \
	\( -name 'foliopath.db' -o -name 'foliopath.db-shm' -o -name 'foliopath.db-wal' \) \
	-exec du -k {} + | awk '{ total += $1 } END { print total * 1024 }')
cache_bytes=$(du -sk "${data_root}/cache" | awk '{ print $1 * 1024 }')
scan_duration_ms=$(( (scan_finished - scan_started) * 1000 ))
image_platform=$(docker image inspect "${image}" --format '{{.Os}}/{{.Architecture}}')
image_size_bytes=$(docker image inspect "${image}" --format '{{.Size}}')
fixture_duration_ms=$(jq '.generationNanos / 1000000 | floor' "${metrics_root}/fixture.json")
sentinel_after=$(sha256sum "${sentinel}" | cut -d ' ' -f 1)
test "${sentinel_after}" = "${sentinel_before}"

jq -n \
	--arg profile "s5-release-capacity-v1" \
	--arg image "${image}" \
	--arg platform "${image_platform}" \
	--arg libraryId "${library_id}" \
	--arg scanId "${scan_id}" \
	--argjson directories "${directory_count}" \
	--argjson assets "${asset_count}" \
	--argjson fixtureDurationMs "${fixture_duration_ms}" \
	--argjson scanDurationMs "${scan_duration_ms}" \
	--argjson scanBudgetMs "${scan_budget_ms}" \
	--argjson browseP95Ms "${browse_p95_ms}" \
	--argjson searchP95Ms "${search_p95_ms}" \
	--argjson globalP95Ms "${global_p95_ms}" \
	--argjson memoryPeakBytes "${memory_peak_bytes}" \
	--argjson databaseBytes "${database_bytes}" \
	--argjson cacheBytes "${cache_bytes}" \
	--argjson imageSizeBytes "${image_size_bytes}" \
	'{
		profile: $profile,
		image: $image,
		platform: $platform,
		libraryId: $libraryId,
		scanId: $scanId,
		directories: $directories,
		assets: $assets,
		fixtureDurationMs: $fixtureDurationMs,
		scanDurationMs: $scanDurationMs,
		scanBudgetMs: $scanBudgetMs,
		browseP95Ms: $browseP95Ms,
		searchP95Ms: $searchP95Ms,
		globalP95Ms: $globalP95Ms,
		memoryPeakBytes: $memoryPeakBytes,
		databaseBytes: $databaseBytes,
		cacheBytes: $cacheBytes,
		imageSizeBytes: $imageSizeBytes
	}' | tee "${metrics_root}/result.json"

if [ -n "${FOLIOPATH_CAPACITY_METRICS_OUTPUT:-}" ]; then
	mkdir -p "$(dirname -- "${FOLIOPATH_CAPACITY_METRICS_OUTPUT}")"
	cp "${metrics_root}/result.json" "${FOLIOPATH_CAPACITY_METRICS_OUTPUT}"
fi

if [ "${enforce_budget}" = "1" ]; then
	jq -e '
		.scanDurationMs <= .scanBudgetMs and
		.browseP95Ms <= 250 and
		.searchP95Ms <= 250 and
		.globalP95Ms <= 250 and
		.memoryPeakBytes <= 1610612736 and
		.databaseBytes <= 1073741824 and
		.cacheBytes <= 1073741824
	' "${metrics_root}/result.json" >/dev/null
fi

printf '%s\n' "Stage 5 release capacity probe passed"
