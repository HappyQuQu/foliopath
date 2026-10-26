#!/bin/sh
set -eu

test "$#" -eq 3
image=$1
media_root=$2
data_root=$3
repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
project="foliopath-stage5-$PPID-$$"
. "${repo_root}/tests/release/storage.sh"
configure_release_storage
compose_root=$(mktemp -d "${TMPDIR:-/tmp}/foliopath-compose-smoke.XXXXXX")
volume_name=${FOLIOPATH_TEST_DATA_VOLUME:-"${project}-data"}

compose_release() {
	if [ "${release_data_storage}" = volume ]; then
		set -- --file "${compose_root}/data-volume.yaml" "$@"
	fi
	FOLIOPATH_IMAGE="${image}" \
	FOLIOPATH_LIBRARY_PATH="${media_root}" \
	FOLIOPATH_DATA_PATH="${data_root}" \
	FOLIOPATH_BIND_ADDRESS="127.0.0.1" \
	FOLIOPATH_PORT=0 \
		docker compose --project-name "${project}" \
		--file "${repo_root}/compose.yaml" "$@"
}

cleanup() {
	compose_release down --volumes >/dev/null 2>&1 || true
	if [ "${release_data_storage}" = volume ] && [ -z "${FOLIOPATH_TEST_DATA_VOLUME:-}" ]; then
		docker volume rm "${volume_name}" >/dev/null 2>&1 || true
	fi
	rm -rf -- "${compose_root}"
}
trap cleanup EXIT HUP INT TERM

if [ "${release_data_storage}" = volume ]; then
	if [ -z "${FOLIOPATH_TEST_DATA_VOLUME:-}" ]; then
		docker volume create --label "foliopath.test.run=${FOLIOPATH_TEST_RUN_ID:-unassigned}" \
			"${volume_name}" >/dev/null
	else
		docker volume inspect "${volume_name}" >/dev/null
	fi
	cat >"${compose_root}/data-volume.yaml" <<YAML
services:
  foliopath:
    volumes:
      - type: volume
        source: release-data
        target: /app/data
        volume:
          nocopy: true
volumes:
  release-data:
    external: true
    name: ${volume_name}
YAML
fi

compose_release up --detach
container=$(compose_release ps --quiet foliopath)
test -n "${container}"
test "$(docker inspect --format '{{.Config.User}}' "${container}")" = "0"
if [ "${release_data_storage}" = volume ]; then
	test "$(docker inspect --format '{{range .Mounts}}{{if eq .Destination "/app/data"}}{{.Type}}{{end}}{{end}}' "${container}")" = volume
fi

deadline=$(( $(date +%s) + 60 ))
while [ "$(date +%s)" -lt "${deadline}" ]; do
	status=$(docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{end}}' "${container}")
	test "${status}" != "unhealthy" || break
	test "${status}" != "healthy" || break
	sleep 1
done
test "$(docker inspect --format '{{.State.Health.Status}}' "${container}")" = "healthy"
test "$(docker inspect --format '{{.HostConfig.ReadonlyRootfs}}' "${container}")" = "true"
test "$(docker inspect --format '{{.HostConfig.CapDrop}}' "${container}")" = "[ALL]"
test "$(docker inspect --format '{{range .Mounts}}{{if eq .Destination "/library"}}{{.RW}}{{end}}{{end}}' "${container}")" = "false"

published=$(docker port "${container}" 8080/tcp)
port=${published##*:}
status=$(curl --silent --output /dev/null --write-out '%{http_code}' \
	"http://127.0.0.1:${port}/api/v1/auth/status")
test "${status}" = "200"

printf '%s\n' "candidate Compose authenticated LAN HTTP smoke passed"
