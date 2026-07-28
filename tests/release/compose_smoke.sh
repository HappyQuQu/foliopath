#!/bin/sh
set -eu

test "$#" -eq 3
image=$1
media_root=$2
data_root=$3
repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
project="foliopath-stage5-$PPID-$$"

cleanup() {
	FOLIOPATH_IMAGE="${image}" \
	FOLIOPATH_LIBRARY_PATH="${media_root}" \
	FOLIOPATH_DATA_PATH="${data_root}" \
	FOLIOPATH_TRUSTED_PROXIES="192.0.2.1/32" \
	FOLIOPATH_PORT=0 \
		docker compose --project-name "${project}" \
		--file "${repo_root}/compose.yaml" down --volumes >/dev/null 2>&1 || true
}
trap cleanup EXIT HUP INT TERM

FOLIOPATH_IMAGE="${image}" \
FOLIOPATH_LIBRARY_PATH="${media_root}" \
FOLIOPATH_DATA_PATH="${data_root}" \
FOLIOPATH_TRUSTED_PROXIES="192.0.2.1/32" \
FOLIOPATH_PORT=0 \
	docker compose --project-name "${project}" \
	--file "${repo_root}/compose.yaml" up --detach

container=$(FOLIOPATH_IMAGE="${image}" \
	FOLIOPATH_LIBRARY_PATH="${media_root}" \
	FOLIOPATH_DATA_PATH="${data_root}" \
	FOLIOPATH_TRUSTED_PROXIES="192.0.2.1/32" \
	FOLIOPATH_PORT=0 \
	docker compose --project-name "${project}" \
	--file "${repo_root}/compose.yaml" ps --quiet foliopath)
test -n "${container}"

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
test "${status}" = "400"

printf '%s\n' "candidate Compose security smoke passed"
