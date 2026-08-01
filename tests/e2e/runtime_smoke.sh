#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
smoke_root=$(mktemp -d "${TMPDIR:-/tmp}/foliopath-smoke.XXXXXX")
image=${FOLIOPATH_SMOKE_IMAGE:-"foliopath-application-smoke:local-$$"}
container="foliopath-application-smoke-$$"
data_root="${smoke_root}/data"
media_root="${smoke_root}/library"

cleanup() {
	docker rm --force "${container}" >/dev/null 2>&1 || true
	docker image rm --force "${image}" >/dev/null 2>&1 || true
	chmod u+w "${media_root}" "${media_root}/sentinel.txt" >/dev/null 2>&1 || true
	rm -rf -- "${smoke_root}"
}
trap cleanup EXIT HUP INT TERM

mkdir -p "${data_root}" "${media_root}"
# Match Docker's root-owned bind directory created by a short Compose mount.
chmod 0755 "${data_root}"
printf '%s\n' "media must remain unchanged" >"${smoke_root}/media-sentinel"
cp "${smoke_root}/media-sentinel" "${media_root}/sentinel.txt"
chmod 0444 "${media_root}/sentinel.txt"
chmod 0555 "${media_root}"
sentinel_before=$(sha256sum "${media_root}/sentinel.txt" | cut -d ' ' -f 1)

docker build \
	--file "${repo_root}/tests/e2e/Dockerfile" \
	--tag "${image}" \
	--build-arg VERSION=s1-smoke \
	"${repo_root}"

wait_healthy() {
	deadline=$(( $(date +%s) + 45 ))
	while [ "$(date +%s)" -lt "${deadline}" ]; do
		status=$(docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{end}}' "${container}")
		case "${status}" in
			healthy)
				return 0
				;;
			unhealthy)
				break
				;;
		esac
		sleep 1
	done
	docker inspect "${container}"
	docker logs "${container}"
	return 1
}

start_application() {
	docker run --detach \
		--name "${container}" \
		--mount "type=bind,src=${data_root},dst=/app/data" \
		--mount "type=bind,src=${media_root},dst=/library,readonly" \
		"${image}" >/dev/null
	wait_healthy
}

stop_application() {
	docker stop --time 10 "${container}" >/dev/null
	exit_code=$(docker inspect --format '{{.State.ExitCode}}' "${container}")
	test "${exit_code}" = "0"
	docker logs "${container}" | grep -q '"msg":"application.stopped"'
	docker rm "${container}" >/dev/null
}

start_application
test -z "$(docker inspect --format '{{.Config.User}}' "${container}")"
test "$(docker inspect --format '{{range .Mounts}}{{if eq .Destination "/library"}}{{.RW}}{{end}}{{end}}' "${container}")" = "false"
docker exec "${container}" curl --fail --silent --show-error \
	http://127.0.0.1:8080/health/live | grep -q '"status":"live"'
docker exec "${container}" curl --fail --silent --show-error \
	http://127.0.0.1:8080/health/ready | grep -q '"status":"ready"'
status_code=$(docker exec "${container}" curl --silent --output /dev/null \
	--write-out '%{http_code}' http://127.0.0.1:8080/api/v1/status)
test "${status_code}" = "401"
test -f "${data_root}/foliopath.db"
test -d "${data_root}/cache"
test -d "${data_root}/tmp"
stop_application

# A second real process against the same data volume proves that embedded
# migrations and application-data preparation are restart-safe.
start_application
stop_application

sentinel_after=$(sha256sum "${media_root}/sentinel.txt" | cut -d ' ' -f 1)
test "${sentinel_after}" = "${sentinel_before}"
printf '%s\n' "application container smoke passed"
