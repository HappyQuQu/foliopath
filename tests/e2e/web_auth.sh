#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
e2e_root=$(mktemp -d "${TMPDIR:-/tmp}/foliopath-web-e2e.XXXXXX")
image=${FOLIOPATH_BROWSER_E2E_IMAGE:-"foliopath-browser-e2e:local-$$"}
application_container="foliopath-browser-e2e-application-$$"
proxy_container="foliopath-browser-e2e-proxy-$$"
proxy_image=${FOLIOPATH_SOCAT_IMAGE:-"alpine/socat@sha256:f134cb7ebb983f971f5deb44e92bc62c1385b0a3b525393f32dd0722acc30315"}
backend_port=${FOLIOPATH_BROWSER_BACKEND_PORT:-18080}
web_port=${FOLIOPATH_BROWSER_WEB_PORT:-4174}
data_root="${e2e_root}/data"
media_root="${e2e_root}/library"
long_path_one="family-archives-with-a-deliberately-long-directory-name"
long_path_two="2026-travel-and-celebration-originals-with-more-context"
vite_log="${e2e_root}/vite.log"
vite_pid=""

cleanup() {
	if [ -n "${vite_pid}" ]; then
		kill "${vite_pid}" >/dev/null 2>&1 || true
		wait "${vite_pid}" >/dev/null 2>&1 || true
	fi
	docker rm --force "${proxy_container}" "${application_container}" >/dev/null 2>&1 || true
	docker image rm --force "${image}" >/dev/null 2>&1 || true
	chmod -R u+w "${media_root}" >/dev/null 2>&1 || true
	rm -rf -- "${e2e_root}"
}
trap cleanup EXIT HUP INT TERM

mkdir -p "${data_root}" "${media_root}"
mkdir -p \
	"${media_root}/${long_path_one}/${long_path_two}/visible-child"
ln -s "visible-child" \
	"${media_root}/${long_path_one}/${long_path_two}/linked-child"
chmod 0777 "${data_root}"
chmod 0555 "${media_root}"

docker build \
	--file "${repo_root}/tests/e2e/Dockerfile" \
	--tag "${image}" \
	--build-arg VERSION=s1-browser-e2e \
	"${repo_root}"

docker run --detach \
	--name "${application_container}" \
	--publish "127.0.0.1:${backend_port}:8081" \
	--mount "type=bind,src=${data_root},dst=/app/data" \
	--mount "type=bind,src=${media_root},dst=/library,readonly" \
	"${image}" >/dev/null

docker run --detach \
	--name "${proxy_container}" \
	--network "container:${application_container}" \
	"${proxy_image}" \
	"TCP-LISTEN:8081,fork,reuseaddr" \
	"TCP:127.0.0.1:8080" >/dev/null

wait_for_url() {
	url=$1
	deadline=$(( $(date +%s) + 60 ))
	while [ "$(date +%s)" -lt "${deadline}" ]; do
		if curl --fail --silent "${url}" >/dev/null 2>&1; then
			return 0
		fi
		sleep 1
	done
	return 1
}

if ! wait_for_url "http://127.0.0.1:${backend_port}/health/ready"; then
	docker logs "${application_container}"
	docker logs "${proxy_container}"
	exit 1
fi

FOLIOPATH_API_ORIGIN="http://127.0.0.1:${backend_port}" \
	npm --prefix "${repo_root}/web" run dev -- \
	--host 127.0.0.1 \
	--port "${web_port}" \
	--strictPort >"${vite_log}" 2>&1 &
vite_pid=$!

if ! wait_for_url "http://127.0.0.1:${web_port}/"; then
	cat "${vite_log}"
	exit 1
fi

FOLIOPATH_WEB_E2E_URL="http://127.0.0.1:${web_port}" \
	FOLIOPATH_E2E_LONG_PATH_ONE="${long_path_one}" \
	FOLIOPATH_E2E_LONG_PATH_TWO="${long_path_two}" \
	npm --prefix "${repo_root}/web" run test:e2e

printf '%s\n' "authentication and library browser e2e passed"
