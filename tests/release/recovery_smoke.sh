#!/bin/sh
set -eu

test "$#" -eq 1
image=$1
repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
. "${repo_root}/tests/release/http_client.sh"
smoke_root=$(mktemp -d "${TMPDIR:-/tmp}/foliopath-recovery-smoke.XXXXXX")
container="foliopath-recovery-smoke-$$"
http_client="foliopath-recovery-http-client-$$"
failure_container="foliopath-recovery-failure-$$"
full_volume="foliopath-recovery-full-data-$$"
source_root="${smoke_root}/source"
backup_root="${smoke_root}/backup"
restore_root="${smoke_root}/restore"
corrupt_root="${smoke_root}/corrupt"
media_root="${smoke_root}/library"

cleanup() {
	docker rm --force "${http_client}" >/dev/null 2>&1 || true
	docker rm --force "${container}" "${failure_container}" >/dev/null 2>&1 || true
	docker volume rm "${full_volume}" >/dev/null 2>&1 || true
	chmod -R u+w "${smoke_root}" >/dev/null 2>&1 || true
	rm -rf -- "${smoke_root}"
}
trap cleanup EXIT HUP INT TERM

mkdir -p \
	"${source_root}" "${backup_root}" "${restore_root}" \
	"${corrupt_root}" "${media_root}"
chmod 0777 "${source_root}" "${backup_root}" "${restore_root}" "${corrupt_root}"
printf '%s\n' "recovery media is immutable" >"${media_root}/sentinel.txt"
chmod 0444 "${media_root}/sentinel.txt"
chmod 0555 "${media_root}"
sentinel_before=$(sha256sum "${media_root}/sentinel.txt" | cut -d ' ' -f 1)

wait_healthy() {
	target=$1
	deadline=$(( $(date +%s) + 60 ))
	while [ "$(date +%s)" -lt "${deadline}" ]; do
		status=$(docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{end}}' "${target}")
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
	docker inspect "${target}"
	docker logs "${target}"
	return 1
}

start_application() {
	data_root=$1
	docker run --detach \
		--name "${container}" \
		--read-only \
		--cap-drop ALL \
		--security-opt no-new-privileges:true \
		--tmpfs /tmp:rw,noexec,nosuid,size=16m \
		--mount "type=bind,src=${data_root},dst=/app/data" \
		--mount "type=bind,src=${media_root},dst=/library,readonly" \
	"${image}" >/dev/null
	wait_healthy "${container}"
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

assert_startup_fails() {
	mode=$1
	shift
	docker run --detach \
		--name "${failure_container}" \
		--read-only \
		--cap-drop ALL \
		--security-opt no-new-privileges:true \
		--tmpfs /tmp:rw,noexec,nosuid,size=16m \
		--mount "type=bind,src=${media_root},dst=/library,readonly" \
		"$@" "${image}" >/dev/null
	deadline=$(( $(date +%s) + 20 ))
	while [ "$(date +%s)" -lt "${deadline}" ]; do
		running=$(docker inspect --format '{{.State.Running}}' "${failure_container}")
		test "${running}" = "true" || break
		sleep 1
	done
	if [ "$(docker inspect --format '{{.State.Running}}' "${failure_container}")" = "true" ]; then
		docker logs "${failure_container}"
		echo "${mode} startup unexpectedly remained running" >&2
		return 1
	fi
	test "$(docker inspect --format '{{.State.ExitCode}}' "${failure_container}")" != "0"
	docker logs "${failure_container}" 2>&1 |
		grep -Eq '"msg":"application.failed"|^foliopath: startup failed$'
	docker rm "${failure_container}" >/dev/null
}

# Persist a real administrator so recovery checks application state, not only an empty schema.
build_release_http_client "${repo_root}"
start_application "${source_root}"
setup_status=$(docker exec "${http_client}" curl --silent --show-error \
	--output /tmp/setup-response.json \
	--write-out '%{http_code}' \
	--header 'Content-Type: application/json' \
	--header 'Origin: http://127.0.0.1:8080' \
	--data '{"username":"RecoveryAdmin","displayName":"Recovery Admin","password":"correct horse battery staple"}' \
	http://127.0.0.1:8080/api/v1/auth/setup)
test "${setup_status}" = "201"
assert_initialized
stop_application

# Offline backup deliberately omits reconstructible cache/tmp state and preserves the SQLite family.
docker run --rm --user 0:0 --entrypoint sh \
	--mount "type=bind,src=${source_root},dst=/from,readonly" \
	--mount "type=bind,src=${backup_root},dst=/backup" \
	"${release_http_client_image}" \
	-c "cd /from && tar --exclude='./cache' --exclude='./tmp' -cf /backup/data.tar ."
docker run --rm --user 0:0 --entrypoint sh \
	--mount "type=bind,src=${backup_root},dst=/backup,readonly" \
	--mount "type=bind,src=${restore_root},dst=/restore" \
	"${release_http_client_image}" \
	-c "cd /restore && tar -xf /backup/data.tar"

start_application "${restore_root}"
assert_initialized
test -d "${restore_root}/cache"
test -d "${restore_root}/tmp"
stop_application

# A same-version restart proves migration idempotence; SIGKILL then proves WAL recovery.
start_application "${restore_root}"
assert_initialized
docker kill --signal KILL "${container}" >/dev/null
docker rm --force "${http_client}" >/dev/null
test "$(docker inspect --format '{{.State.ExitCode}}' "${container}")" = "137"
docker rm "${container}" >/dev/null
start_application "${restore_root}"
assert_initialized
stop_application

# Exhausted data storage and a corrupt database must both fail before readiness.
docker volume create \
	--driver local \
	--opt type=tmpfs \
	--opt device=tmpfs \
	--opt o=size=64k,mode=0700 \
	"${full_volume}" >/dev/null
docker run --rm --user 0:0 --entrypoint sh \
	--mount "type=volume,src=${full_volume},dst=/data" \
	"${release_http_client_image}" \
	-c 'chown 65532:65532 /data; dd if=/dev/zero of=/data/fill bs=1024 count=64 >/dev/null 2>&1 || true'
assert_startup_fails "full data" \
	--mount "type=volume,src=${full_volume},dst=/app/data"
docker volume rm "${full_volume}" >/dev/null

printf '%s\n' "not-a-sqlite-database" >"${corrupt_root}/foliopath.db"
chmod 0666 "${corrupt_root}/foliopath.db"
assert_startup_fails "corrupt database" \
	--mount "type=bind,src=${corrupt_root},dst=/app/data"

sentinel_after=$(sha256sum "${media_root}/sentinel.txt" | cut -d ' ' -f 1)
test "${sentinel_after}" = "${sentinel_before}"
printf '%s\n' "candidate recovery smoke passed"
