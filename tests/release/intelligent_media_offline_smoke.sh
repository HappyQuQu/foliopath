#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
. "${repo_root}/tests/release/http_client.sh"

work_root=$(mktemp -d "${TMPDIR:-/tmp}/foliopath-ai-offline.XXXXXX")
data_root="${work_root}/data"
media_root="${work_root}/library"
model_root="${work_root}/models"
image=${FOLIOPATH_RELEASE_IMAGE:-"foliopath:intelligent-media-offline-$$"}
build_image=${FOLIOPATH_RELEASE_BUILD_IMAGE:-1}
container="foliopath-ai-offline-$$"
http_client="foliopath-ai-offline-http-client-$$"

cleanup() {
	docker rm --force "${http_client}" "${container}" >/dev/null 2>&1 || true
	if [ "${build_image}" = "1" ]; then
		docker image rm --force "${image}" >/dev/null 2>&1 || true
	fi
	chmod -R u+w "${work_root}" >/dev/null 2>&1 || true
	rm -rf -- "${work_root}"
}
trap cleanup EXIT HUP INT TERM

mkdir -p "${data_root}" "${media_root}" "${model_root}"
chmod 0777 "${data_root}"
printf '%s\n' 'offline media sentinel' >"${media_root}/sentinel.txt"
printf '%s\n' 'offline model sentinel' >"${model_root}/sentinel.txt"
chmod 0444 "${media_root}/sentinel.txt" "${model_root}/sentinel.txt"
chmod 0555 "${media_root}" "${model_root}"
media_before=$(sha256sum "${media_root}/sentinel.txt" | cut -d ' ' -f 1)
model_before=$(sha256sum "${model_root}/sentinel.txt" | cut -d ' ' -f 1)

case "${build_image}" in
0)
	docker image inspect "${image}" >/dev/null
	;;
1)
	docker build \
		--file "${repo_root}/Dockerfile" \
		--tag "${image}" \
		--build-arg VERSION=intelligent-media-offline \
		"${repo_root}" >/dev/null
	;;
*)
	echo "FOLIOPATH_RELEASE_BUILD_IMAGE must be 0 or 1" >&2
	exit 2
	;;
esac

build_release_http_client "${repo_root}"
docker run --detach \
	--name "${container}" \
	--network none \
	--read-only \
	--cap-drop ALL \
	--security-opt no-new-privileges:true \
	--tmpfs /tmp:rw,noexec,nosuid,size=16m \
	--mount "type=bind,src=${data_root},dst=/app/data" \
	--mount "type=bind,src=${media_root},dst=/library,readonly" \
	--mount "type=bind,src=${model_root},dst=/models,readonly" \
	"${image}" >/dev/null

deadline=$(( $(date +%s) + 60 ))
status=
while [ "$(date +%s)" -lt "${deadline}" ]; do
	status=$(docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{end}}' "${container}")
	test "${status}" != "unhealthy" || break
	test "${status}" != "healthy" || break
	sleep 1
done
if [ "${status}" != "healthy" ]; then
	docker inspect "${container}"
	docker logs "${container}"
	exit 1
fi

test "$(docker inspect --format '{{.HostConfig.NetworkMode}}' "${container}")" = "none"
test "$(docker inspect --format '{{.HostConfig.ReadonlyRootfs}}' "${container}")" = "true"
test "$(docker inspect --format '{{.HostConfig.CapDrop}}' "${container}")" = "[ALL]"
test "$(docker inspect --format '{{range .Mounts}}{{if eq .Destination "/library"}}{{.RW}}{{end}}{{end}}' "${container}")" = "false"
test "$(docker inspect --format '{{range .Mounts}}{{if eq .Destination "/models"}}{{.RW}}{{end}}{{end}}' "${container}")" = "false"

start_release_http_client "${http_client}" "${container}"
setup_response=$(docker exec "${http_client}" curl --fail --silent --show-error \
	--cookie-jar /tmp/cookies \
	--header 'Content-Type: application/json' \
	--header 'Origin: http://127.0.0.1:8080' \
	--data '{"username":"OfflineAdmin","displayName":"Offline Admin","password":"correct horse battery staple"}' \
	http://127.0.0.1:8080/api/v1/auth/setup)
csrf_token=$(printf '%s' "${setup_response}" | jq -er '.csrfToken')

docker exec "${http_client}" curl --fail --silent --show-error \
	--cookie /tmp/cookies \
	http://127.0.0.1:8080/api/v1/ai/models |
	jq -e '.items == [] and .activeModelId == null and .activeFaceModelId == null' >/dev/null
docker exec "${http_client}" curl --fail --silent --show-error \
	--cookie /tmp/cookies \
	--header "X-CSRF-Token: ${csrf_token}" \
	--header 'Origin: http://127.0.0.1:8080' \
	--request POST \
	http://127.0.0.1:8080/api/v1/ai/model-candidate-scans |
	jq -e '.candidates == [] and .truncated == false' >/dev/null

docker rm --force "${http_client}" >/dev/null
docker restart "${container}" >/dev/null
deadline=$(( $(date +%s) + 60 ))
status=
while [ "$(date +%s)" -lt "${deadline}" ]; do
	status=$(docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{end}}' "${container}")
	test "${status}" != "unhealthy" || break
	test "${status}" != "healthy" || break
	sleep 1
done
test "${status}" = "healthy"

media_after=$(sha256sum "${media_root}/sentinel.txt" | cut -d ' ' -f 1)
model_after=$(sha256sum "${model_root}/sentinel.txt" | cut -d ' ' -f 1)
test "${media_after}" = "${media_before}"
test "${model_after}" = "${model_before}"

docker image inspect "${image}" --format \
	'offline image platform={{.Os}}/{{.Architecture}} digest={{.Id}}'
printf '%s\n' 'intelligent media no-network /models:ro smoke passed'
