#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
. "${repo_root}/tests/release/http_client.sh"
smoke_root=$(mktemp -d "${TMPDIR:-/tmp}/foliopath-release-smoke.XXXXXX")
image=${FOLIOPATH_RELEASE_IMAGE:-"foliopath:stage5-local-$$"}
dockerfile=${FOLIOPATH_RELEASE_DOCKERFILE:-"${repo_root}/Dockerfile"}
expected_version=${FOLIOPATH_RELEASE_EXPECTED_VERSION:-stage5-local}
build_image=${FOLIOPATH_RELEASE_BUILD_IMAGE:-1}
container="foliopath-stage5-smoke-$$"
http_client="foliopath-stage5-http-client-$$"
proxy_container="foliopath-stage5-proxy-smoke-$$"
proxy_network="foliopath-stage5-proxy-net-$$"
fixture_image="foliopath-release-fixture-generator:local"
fixture_volume="foliopath-stage5-media-fixtures-$$"
data_root="${smoke_root}/data"
media_root="${smoke_root}/library"

cleanup() {
	docker rm --force "${http_client}" >/dev/null 2>&1 || true
	docker rm --force "${container}" >/dev/null 2>&1 || true
	docker rm --force "${proxy_container}" >/dev/null 2>&1 || true
	docker network rm "${proxy_network}" >/dev/null 2>&1 || true
	docker volume rm "${fixture_volume}" >/dev/null 2>&1 || true
	if [ -d "${data_root}" ]; then
		docker run --rm --entrypoint /bin/chmod \
			--mount "type=bind,src=${data_root},dst=/data" \
			"${fixture_image}" -R 0777 /data >/dev/null 2>&1 || true
	fi
	if [ "${build_image}" = "1" ]; then
		docker image rm --force "${image}" >/dev/null 2>&1 || true
	fi
	chmod u+w "${media_root}" "${media_root}/sentinel.txt" >/dev/null 2>&1 || true
	rm -rf -- "${smoke_root}"
}
trap cleanup EXIT HUP INT TERM

mkdir -p "${media_root}"
printf '%s\n' "original media is immutable" >"${media_root}/sentinel.txt"
chmod 0444 "${media_root}/sentinel.txt"
chmod 0555 "${media_root}"
sentinel_before=$(sha256sum "${media_root}/sentinel.txt" | cut -d ' ' -f 1)

case "${build_image}" in
0)
	docker image inspect "${image}" >/dev/null
	;;
1)
	docker build \
		--file "${dockerfile}" \
		--tag "${image}" \
		--build-arg VERSION=stage5-local \
		"${repo_root}"
	;;
*)
	echo "FOLIOPATH_RELEASE_BUILD_IMAGE must be 0 or 1" >&2
	exit 2
	;;
esac

case "${FOLIOPATH_RELEASE_BUILD_HELPERS:-1}" in
0)
	docker image inspect "${fixture_image}" >/dev/null
	;;
1)
	docker build \
		--file "${repo_root}/tests/fixtures/media-matrix/Dockerfile" \
		--tag "${fixture_image}" \
		"${repo_root}/tests/fixtures/media-matrix" >/dev/null
	;;
*)
	echo "FOLIOPATH_RELEASE_BUILD_HELPERS must be 0 or 1" >&2
	exit 2
	;;
esac
docker volume create "${fixture_volume}" >/dev/null
docker run --rm \
	--mount "type=volume,src=${fixture_volume},dst=/fixtures" \
	"${fixture_image}" >/dev/null

docker run --detach \
	--name "${container}" \
	--read-only \
	--cap-drop ALL \
	--security-opt no-new-privileges:true \
	--tmpfs /tmp:rw,noexec,nosuid,size=16m \
	--volume "${data_root}:/app/data" \
	--mount "type=bind,src=${media_root},dst=/library,readonly" \
	--mount "type=volume,src=${fixture_volume},dst=/fixtures" \
	"${image}" >/dev/null

deadline=$(( $(date +%s) + 60 ))
while [ "$(date +%s)" -lt "${deadline}" ]; do
	status=$(docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{end}}' "${container}")
	if [ "${status}" = "healthy" ]; then
		break
	fi
	if [ "${status}" = "unhealthy" ]; then
		docker inspect "${container}"
		docker logs "${container}"
		exit 1
	fi
	sleep 1
done
test "$(docker inspect --format '{{.State.Health.Status}}' "${container}")" = "healthy"

build_release_http_client "${repo_root}"
start_release_http_client "${http_client}" "${container}"
test "$(docker inspect --format '{{.Config.User}}' "${container}")" = "0"
test "$(docker inspect --format '{{.HostConfig.ReadonlyRootfs}}' "${container}")" = "true"
test "$(docker inspect --format '{{.HostConfig.CapDrop}}' "${container}")" = "[ALL]"
test "$(docker inspect --format '{{range .Mounts}}{{if eq .Destination "/library"}}{{.RW}}{{end}}{{end}}' "${container}")" = "false"
test "$(docker exec "${container}" /app/foliopath version)" = \
	"foliopath ${expected_version}"
docker exec "${http_client}" curl --fail --silent --show-error \
	http://127.0.0.1:8080/ | grep -q '<title>FolioPath</title>'
docker exec "${http_client}" curl --fail --silent --show-error \
	http://127.0.0.1:8080/health/ready | grep -q '"status":"ready"'
"${repo_root}/tests/release/image_media_smoke.sh" "${container}" /fixtures

docker stop --time 10 "${container}" >/dev/null
docker rm --force "${http_client}" >/dev/null
test "$(docker inspect --format '{{.State.ExitCode}}' "${container}")" = "0"
docker logs "${container}" | grep -q '"msg":"application.stopped"'

sentinel_after=$(sha256sum "${media_root}/sentinel.txt" | cut -d ' ' -f 1)
test "${sentinel_after}" = "${sentinel_before}"

"${repo_root}/tests/release/compose_smoke.sh" \
	"${image}" "${media_root}" "${data_root}"

docker network create "${proxy_network}" >/dev/null
proxy_gateway=$(docker network inspect \
	--format '{{(index .IPAM.Config 0).Gateway}}' "${proxy_network}")
docker run --detach \
	--name "${proxy_container}" \
	--network "${proxy_network}" \
	--read-only \
	--cap-drop ALL \
	--security-opt no-new-privileges:true \
	--tmpfs /tmp:rw,noexec,nosuid,size=16m \
	--mount "type=bind,src=${data_root},dst=/app/data" \
	--mount "type=bind,src=${media_root},dst=/library,readonly" \
	--env FOLIOPATH_LISTEN=0.0.0.0:8080 \
	--env "FOLIOPATH_TRUSTED_PROXIES=${proxy_gateway}/32" \
	--publish 127.0.0.1::8080 \
	"${image}" >/dev/null

deadline=$(( $(date +%s) + 60 ))
while [ "$(date +%s)" -lt "${deadline}" ]; do
	status=$(docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{end}}' "${proxy_container}")
	test "${status}" != "unhealthy" || break
	test "${status}" != "healthy" || break
	sleep 1
done
test "$(docker inspect --format '{{.State.Health.Status}}' "${proxy_container}")" = "healthy"
published=$(docker port "${proxy_container}" 8080/tcp)
proxy_port=${published##*:}
direct_status=$(curl --silent --output /dev/null --write-out '%{http_code}' \
	"http://127.0.0.1:${proxy_port}/api/v1/auth/status")
test "${direct_status}" = "400"
headers=$(mktemp "${smoke_root}/proxy-headers.XXXXXX")
proxy_status=$(curl --silent --output /dev/null --dump-header "${headers}" \
	--write-out '%{http_code}' \
	--header 'X-Forwarded-Proto: https' \
	--header 'X-Forwarded-Host: photos.example' \
	--header 'X-Forwarded-For: 203.0.113.9' \
	"http://127.0.0.1:${proxy_port}/api/v1/auth/status")
test "${proxy_status}" = "200"
grep -qi '^Strict-Transport-Security:' "${headers}"

"${repo_root}/tests/release/recovery_smoke.sh" "${image}"

docker image inspect "${image}" \
	--format 'candidate platform={{.Os}}/{{.Architecture}} size={{.Size}} digest={{.Id}}'

if [ -n "${FOLIOPATH_RELEASE_EVIDENCE:-}" ]; then
	command -v jq >/dev/null
	test -n "${FOLIOPATH_RELEASE_COMMIT:-}"
	test -n "${FOLIOPATH_RELEASE_EXPECTED_ARCH:-}"
	test -n "${FOLIOPATH_RELEASE_RUN_ID:-}"
	test -n "${FOLIOPATH_RELEASE_RUN_ATTEMPT:-}"

	image_os=$(docker image inspect "${image}" --format '{{.Os}}')
	image_arch=$(docker image inspect "${image}" --format '{{.Architecture}}')
	image_digest=$(docker image inspect "${image}" --format '{{.Id}}')
	image_size=$(docker image inspect "${image}" --format '{{.Size}}')
	test "${image_os}" = "linux"
	test "${image_arch}" = "${FOLIOPATH_RELEASE_EXPECTED_ARCH}"

	evidence_dir=$(dirname -- "${FOLIOPATH_RELEASE_EVIDENCE}")
	mkdir -p "${evidence_dir}"
	evidence_tmp=$(mktemp "${evidence_dir}/release-image-evidence.XXXXXX")
	jq -n \
		--arg release "MVP-2026-07-23" \
		--arg source_commit "${FOLIOPATH_RELEASE_COMMIT}" \
		--arg architecture "${image_arch}" \
		--arg os "${image_os}" \
		--arg image_digest "${image_digest}" \
		--argjson image_size_bytes "${image_size}" \
		--arg run_id "${FOLIOPATH_RELEASE_RUN_ID}" \
		--argjson run_attempt "${FOLIOPATH_RELEASE_RUN_ATTEMPT}" \
		--arg created_at "$(date -u '+%Y-%m-%dT%H:%M:%SZ')" \
		'{
			schemaVersion: 1,
			release: $release,
			sourceCommit: $source_commit,
			architecture: $architecture,
			os: $os,
			imageDigest: $image_digest,
			imageSizeBytes: $image_size_bytes,
			smokeSuite: "tests/release/image_smoke.sh",
			result: "passed",
			workflowRunId: $run_id,
			workflowRunAttempt: $run_attempt,
			createdAt: $created_at
		}' >"${evidence_tmp}"
	mv -- "${evidence_tmp}" "${FOLIOPATH_RELEASE_EVIDENCE}"
fi

if [ -n "${FOLIOPATH_RELEASE_PROVENANCE:-}" ]; then
	test -n "${FOLIOPATH_PROVENANCE_BUILDER_ID:-}"
	test -n "${FOLIOPATH_PROVENANCE_INVOCATION_ID:-}"
	"${repo_root}/scripts/generate-provenance.sh" \
		"${image}" "${FOLIOPATH_RELEASE_PROVENANCE}"
fi

printf '%s\n' "Stage 5 candidate image smoke passed"
