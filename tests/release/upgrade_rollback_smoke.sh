#!/bin/sh
set -eu

test "$#" -eq 2
previous_image=$1
current_image=$2
repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
. "${repo_root}/tests/release/http_client.sh"

previous_id=$(docker image inspect --format '{{.Id}}' "${previous_image}")
current_id=$(docker image inspect --format '{{.Id}}' "${current_image}")
test "${previous_id}" != "${current_id}"

smoke_root=$(mktemp -d "${TMPDIR:-/tmp}/foliopath-upgrade-smoke.XXXXXX")
source_root="${smoke_root}/source"
backup_root="${smoke_root}/backup"
rollback_root="${smoke_root}/rollback"
media_root="${smoke_root}/library"
container="foliopath-upgrade-smoke-$$"
http_client="foliopath-upgrade-http-client-$$"

cleanup() {
	docker rm --force "${http_client}" "${container}" >/dev/null 2>&1 || true
	chmod -R u+w "${smoke_root}" >/dev/null 2>&1 || true
	rm -rf -- "${smoke_root}"
}
trap cleanup EXIT HUP INT TERM

mkdir -p "${source_root}" "${backup_root}" "${rollback_root}" "${media_root}"
chmod 0777 "${source_root}" "${backup_root}" "${rollback_root}"
printf '%s\n' "upgrade media is immutable" >"${media_root}/sentinel.txt"
chmod 0444 "${media_root}/sentinel.txt"
chmod 0555 "${media_root}"
sentinel_before=$(sha256sum "${media_root}/sentinel.txt" | cut -d ' ' -f 1)

wait_healthy() {
	deadline=$(( $(date +%s) + 60 ))
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
	image=$1
	data_root=$2
	docker run --detach \
		--name "${container}" \
		--read-only \
		--cap-drop ALL \
		--security-opt no-new-privileges:true \
		--tmpfs /tmp:rw,noexec,nosuid,size=16m \
		--mount "type=bind,src=${data_root},dst=/app/data" \
		--mount "type=bind,src=${media_root},dst=/library,readonly" \
		"${image}" >/dev/null
	wait_healthy
	start_release_http_client "${http_client}" "${container}"
}

stop_application() {
	docker rm --force "${http_client}" >/dev/null
	docker stop --time 10 "${container}" >/dev/null
	test "$(docker inspect --format '{{.State.ExitCode}}' "${container}")" = "0"
	docker logs "${container}" | grep -q '"msg":"application.stopped"'
	docker rm "${container}" >/dev/null
}

assert_initialized() {
	docker exec "${http_client}" curl --fail --silent --show-error \
		http://127.0.0.1:8080/api/v1/auth/status |
		grep -q '"setupRequired":false'
}

build_release_http_client "${repo_root}"

# Seed persistent state with the immutable previous candidate.
start_application "${previous_id}" "${source_root}"
setup_status=$(docker exec "${http_client}" curl --silent --show-error \
	--output /tmp/setup-response.json \
	--write-out '%{http_code}' \
	--header 'Content-Type: application/json' \
	--header 'Origin: http://127.0.0.1:8080' \
	--data '{"username":"UpgradeAdmin","displayName":"Upgrade Admin","password":"correct horse battery staple"}' \
	http://127.0.0.1:8080/api/v1/auth/setup)
test "${setup_status}" = "201"
assert_initialized
stop_application

# Stop-before-copy backup is the rollback boundary.
docker run --rm --user 0:0 --entrypoint sh \
	--mount "type=bind,src=${source_root},dst=/from,readonly" \
	--mount "type=bind,src=${backup_root},dst=/backup" \
	"${release_http_client_image}" \
	-c "cd /from && tar --exclude='./cache' --exclude='./tmp' -cf /backup/data.tar ."
backup_hash=$(sha256sum "${backup_root}/data.tar" | cut -d ' ' -f 1)

# Forward upgrade reuses the persistent data and must preserve initialized state.
start_application "${current_id}" "${source_root}"
assert_initialized
stop_application

# Rollback is only valid when the previous image is paired with the pre-upgrade backup.
docker run --rm --user 0:0 --entrypoint sh \
	--mount "type=bind,src=${backup_root},dst=/backup,readonly" \
	--mount "type=bind,src=${rollback_root},dst=/restore" \
	"${release_http_client_image}" \
	-c "cd /restore && tar -xf /backup/data.tar"
start_application "${previous_id}" "${rollback_root}"
assert_initialized
stop_application

sentinel_after=$(sha256sum "${media_root}/sentinel.txt" | cut -d ' ' -f 1)
test "${sentinel_after}" = "${sentinel_before}"
printf '%s\n' \
	"previous_image_id=${previous_id}" \
	"current_image_id=${current_id}" \
	"pre_upgrade_backup_sha256=${backup_hash}" \
	"candidate upgrade and paired rollback smoke passed"
