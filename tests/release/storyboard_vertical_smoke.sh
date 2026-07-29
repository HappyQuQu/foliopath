#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
smoke_root=$(mktemp -d "${TMPDIR:-/tmp}/foliopath-storyboard-vertical.XXXXXX")
image=${FOLIOPATH_RELEASE_IMAGE:-"foliopath:storyboard-vertical-$$"}
build_image=${FOLIOPATH_RELEASE_BUILD_IMAGE:-1}
skip_browser=${FOLIOPATH_STORYBOARD_SKIP_BROWSER:-0}
container="foliopath-storyboard-vertical-$$"
data_root="${smoke_root}/data"
media_root="${smoke_root}/library"
cookie_jar="${smoke_root}/cookies"
headers="${smoke_root}/headers"

cleanup() {
	docker rm --force "${container}" >/dev/null 2>&1 || true
	if [ "${build_image}" = "1" ]; then
		docker image rm --force "${image}" >/dev/null 2>&1 || true
	fi
	chmod -R u+w "${smoke_root}" >/dev/null 2>&1 || true
	rm -rf -- "${smoke_root}"
}
trap cleanup EXIT HUP INT TERM

command -v curl >/dev/null
command -v docker >/dev/null
command -v ffmpeg >/dev/null
command -v ffprobe >/dev/null
command -v jq >/dev/null

mkdir -p "${data_root}" "${media_root}"
chmod 0777 "${data_root}"
ffmpeg -hide_banner -loglevel error \
	-f lavfi -i "testsrc2=size=320x180:rate=24:duration=10" \
	-c:v mpeg4 -pix_fmt yuv420p -g 24 -y "${media_root}/clip.mp4"
chmod 0444 "${media_root}/clip.mp4"
source_hash=$(sha256sum "${media_root}/clip.mp4" | cut -d ' ' -f 1)
source_mtime=$(stat -f %m "${media_root}/clip.mp4" 2>/dev/null ||
	stat -c %Y "${media_root}/clip.mp4")

case "${build_image}" in
0)
	docker image inspect "${image}" >/dev/null
	;;
1)
	docker build \
		--tag "${image}" \
		--build-arg VERSION=storyboard-vertical \
		"${repo_root}"
	;;
*)
	echo "FOLIOPATH_RELEASE_BUILD_IMAGE must be 0 or 1" >&2
	exit 2
	;;
esac

image_os=$(docker image inspect "${image}" --format '{{.Os}}')
image_arch=$(docker image inspect "${image}" --format '{{.Architecture}}')
test "${image_os}" = "linux"
if [ -n "${FOLIOPATH_STORYBOARD_EXPECTED_ARCH:-}" ]; then
	test "${image_arch}" = "${FOLIOPATH_STORYBOARD_EXPECTED_ARCH}"
fi
ffmpeg_version=$(docker run --rm \
	--entrypoint /opt/ffmpeg/bin/ffmpeg \
	"${image}" -hide_banner -version | sed -n '1p')

docker run --detach \
	--name "${container}" \
	--cpus 4 \
	--memory 4g \
	--read-only \
	--cap-drop ALL \
	--security-opt no-new-privileges:true \
	--tmpfs /tmp:rw,noexec,nosuid,size=64m \
	--mount "type=bind,src=${data_root},dst=/app/data" \
	--mount "type=bind,src=${media_root},dst=/library,readonly" \
	--env FOLIOPATH_LISTEN=0.0.0.0:8080 \
	--publish 127.0.0.1::8080 \
	"${image}" >/dev/null

deadline=$(( $(date +%s) + 90 ))
while [ "$(date +%s)" -lt "${deadline}" ]; do
	status=$(docker inspect \
		--format '{{if .State.Health}}{{.State.Health.Status}}{{end}}' \
		"${container}")
	test "${status}" != "unhealthy" || break
	test "${status}" != "healthy" || break
	sleep 1
done
if [ "$(docker inspect --format '{{.State.Health.Status}}' "${container}")" != "healthy" ]; then
	docker logs "${container}"
	echo "storyboard vertical candidate did not become healthy" >&2
	exit 1
fi

port=$(docker port "${container}" 8080/tcp | sed -n '1s/.*://p')
origin="http://127.0.0.1:${port}"
setup=$(curl --fail --silent --show-error \
	--cookie-jar "${cookie_jar}" \
	--header "Origin: ${origin}" \
	--header "Content-Type: application/json" \
	--data '{"username":"StoryboardAdmin","displayName":"Storyboard Admin","password":"correct horse battery staple"}' \
	"${origin}/api/v1/auth/setup")
csrf_token=$(printf '%s' "${setup}" | jq -er '.csrfToken')

create=$(curl --fail --silent --show-error \
	--cookie "${cookie_jar}" \
	--header "Origin: ${origin}" \
	--header "Content-Type: application/json" \
	--header "X-CSRF-Token: ${csrf_token}" \
	--header "Idempotency-Key: storyboard-vertical-library" \
	--data '{"name":"Storyboard","rootPath":""}' \
	"${origin}/api/v1/libraries")
library_id=$(printf '%s' "${create}" | jq -er '.library.id')

deadline=$(( $(date +%s) + 120 ))
asset_id=
storyboard_status=
while [ "$(date +%s)" -lt "${deadline}" ]; do
	page=$(curl --fail --silent --show-error \
		--cookie "${cookie_jar}" \
		"${origin}/api/v1/libraries/${library_id}/assets?recursive=true&limit=20")
	asset_id=$(printf '%s' "${page}" | jq -r '.items[0].id // empty')
	storyboard_status=$(printf '%s' "${page}" |
		jq -r '.items[0].storyboard.status // empty')
	test "${storyboard_status}" != "ready" || break
	sleep 1
done
test -n "${asset_id}"
test "${storyboard_status}" = "ready"

target="${origin}/api/v1/assets/${asset_id}/thumbnail?variant=storyboard"
unauthorized_status=$(curl --silent --output "${smoke_root}/unauthorized.json" \
	--write-out '%{http_code}' "${target}")
test "${unauthorized_status}" = "401"
test "$(jq -r '.error.code' "${smoke_root}/unauthorized.json")" = \
	"authentication_required"

curl --fail --silent --show-error \
	--cookie "${cookie_jar}" \
	--dump-header "${headers}" \
	--output "${smoke_root}/storyboard.webp" \
	"${target}"
grep -i '^Content-Type: image/webp' "${headers}" >/dev/null
grep -i '^Cache-Control: private, max-age=31536000, immutable' \
	"${headers}" >/dev/null
grep -i '^X-Content-Type-Options: nosniff' "${headers}" >/dev/null
etag=$(sed -n 's/^[Ee][Tt][Aa][Gg]:[[:space:]]*//p' "${headers}" |
	tr -d '\r')
test -n "${etag}"
test "$(ffprobe -v error -select_streams v:0 \
	-show_entries stream=width,height -of csv=p=0:s=x \
	"${smoke_root}/storyboard.webp")" = "1600x360"
storyboard_pixel_hash=$(ffmpeg -hide_banner -loglevel error \
	-i "${smoke_root}/storyboard.webp" \
	-f rawvideo -pix_fmt rgb24 - | sha256sum | cut -d ' ' -f 1)

conditional_status=$(curl --silent --output /dev/null \
	--write-out '%{http_code}' \
	--cookie "${cookie_jar}" \
	--header "If-None-Match: ${etag}" \
	"${target}")
test "${conditional_status}" = "304"

case "${skip_browser}" in
0)
	FOLIOPATH_WEB_E2E_URL="${origin}" \
		FOLIOPATH_STORYBOARD_REAL_E2E=1 \
		FOLIOPATH_STORYBOARD_LIBRARY_ID="${library_id}" \
		npm --prefix "${repo_root}/web" run test:e2e -- \
		--project=chromium tests/e2e/storyboard-real.spec.ts
	;;
1) ;;
*)
	echo "FOLIOPATH_STORYBOARD_SKIP_BROWSER must be 0 or 1" >&2
	exit 2
	;;
esac

find "${data_root}/cache" -type f -print >"${smoke_root}/cache-files"
storyboard_cache=
storyboard_cache_count=0
while IFS= read -r candidate; do
	dimensions=$(ffprobe -v error -select_streams v:0 \
		-show_entries stream=width,height -of csv=p=0:s=x \
		"${candidate}" 2>/dev/null || true)
	if [ "${dimensions}" = "1600x360" ]; then
		storyboard_cache="${candidate}"
		storyboard_cache_count=$((storyboard_cache_count + 1))
	fi
done <"${smoke_root}/cache-files"
test "${storyboard_cache_count}" = "1"
case "${storyboard_cache}" in
"${data_root}/cache/"*) ;;
*)
	echo "resolved storyboard cache escaped the temporary cache root" >&2
	exit 1
	;;
esac
rm "${storyboard_cache}"
repair_status=$(curl --silent --output "${smoke_root}/repair.json" \
	--write-out '%{http_code}' \
	--cookie "${cookie_jar}" \
	"${target}")
test "${repair_status}" = "202"

deadline=$(( $(date +%s) + 120 ))
rebuilt_status=
while [ "$(date +%s)" -lt "${deadline}" ]; do
	rebuilt_status=$(curl --silent --output "${smoke_root}/rebuilt.webp" \
		--write-out '%{http_code}' \
		--cookie "${cookie_jar}" \
		"${target}")
	test "${rebuilt_status}" != "200" || break
	sleep 1
done
test "${rebuilt_status}" = "200"
test "$(ffprobe -v error -select_streams v:0 \
	-show_entries stream=width,height -of csv=p=0:s=x \
	"${smoke_root}/rebuilt.webp")" = "1600x360"
rebuilt_pixel_hash=$(ffmpeg -hide_banner -loglevel error \
	-i "${smoke_root}/rebuilt.webp" \
	-f rawvideo -pix_fmt rgb24 - | sha256sum | cut -d ' ' -f 1)
test "${rebuilt_pixel_hash}" = "${storyboard_pixel_hash}"

source_hash_after=$(sha256sum "${media_root}/clip.mp4" | cut -d ' ' -f 1)
source_mtime_after=$(stat -f %m "${media_root}/clip.mp4" 2>/dev/null ||
	stat -c %Y "${media_root}/clip.mp4")
test "${source_hash_after}" = "${source_hash}"
test "${source_mtime_after}" = "${source_mtime}"

if [ -n "${FOLIOPATH_STORYBOARD_EVIDENCE:-}" ]; then
	test -n "${FOLIOPATH_STORYBOARD_SOURCE_COMMIT:-}"
	test -n "${FOLIOPATH_STORYBOARD_EXPECTED_ARCH:-}"
	test -n "${FOLIOPATH_STORYBOARD_RUN_ID:-}"
	test -n "${FOLIOPATH_STORYBOARD_RUN_ATTEMPT:-}"
	image_digest=$(docker image inspect "${image}" --format '{{.Id}}')
	image_size=$(docker image inspect "${image}" --format '{{.Size}}')
	evidence_dir=$(dirname -- "${FOLIOPATH_STORYBOARD_EVIDENCE}")
	mkdir -p "${evidence_dir}"
	evidence_tmp=$(mktemp "${evidence_dir}/storyboard-evidence.XXXXXX")
	jq -n \
		--arg feature "FTR-VID-001" \
		--arg source_commit "${FOLIOPATH_STORYBOARD_SOURCE_COMMIT}" \
		--arg architecture "${image_arch}" \
		--arg os "${image_os}" \
		--arg image_digest "${image_digest}" \
		--argjson image_size_bytes "${image_size}" \
		--arg ffmpeg_version "${ffmpeg_version}" \
		--arg source_sha256 "${source_hash}" \
		--arg storyboard_pixel_sha256 "${storyboard_pixel_hash}" \
		--arg run_id "${FOLIOPATH_STORYBOARD_RUN_ID}" \
		--argjson run_attempt "${FOLIOPATH_STORYBOARD_RUN_ATTEMPT}" \
		--arg created_at "$(date -u '+%Y-%m-%dT%H:%M:%SZ')" \
		'{
			schemaVersion: 1,
			feature: $feature,
			sourceCommit: $source_commit,
			architecture: $architecture,
			os: $os,
			imageDigest: $image_digest,
			imageSizeBytes: $image_size_bytes,
			ffmpegVersion: $ffmpeg_version,
			fixture: {
				sourceSHA256: $source_sha256,
				frameCount: 10,
				columns: 5,
				rows: 2,
				width: 1600,
				height: 360,
				decodedPixelSHA256: $storyboard_pixel_sha256
			},
			cacheRepair: {
				initialStatus: 200,
				missingStatus: 202,
				rebuiltStatus: 200,
				decodedPixelsMatch: true
			},
			originalMediaUnchanged: true,
			resourceLimit: {
				cpus: 4,
				memoryBytes: 4294967296
			},
			smokeSuite: "tests/release/storyboard_vertical_smoke.sh",
			result: "passed",
			workflowRunId: $run_id,
			workflowRunAttempt: $run_attempt,
			createdAt: $created_at
		}' >"${evidence_tmp}"
	mv -- "${evidence_tmp}" "${FOLIOPATH_STORYBOARD_EVIDENCE}"
fi

printf '%s\n' \
	"authenticated storyboard vertical smoke passed with cache repair and product hover"
