#!/bin/sh

release_http_client_image="foliopath-release-http-client:local"

build_release_http_client() {
	repository_root=$1
	case "${FOLIOPATH_RELEASE_BUILD_HELPERS:-1}" in
	0)
		docker image inspect "${release_http_client_image}" >/dev/null
		;;
	1)
		docker build \
			--file "${repository_root}/tests/fixtures/http-client/Dockerfile" \
			--tag "${release_http_client_image}" \
			"${repository_root}/tests/fixtures/http-client" >/dev/null
		;;
	*)
		echo "FOLIOPATH_RELEASE_BUILD_HELPERS must be 0 or 1" >&2
		return 2
		;;
	esac
}

start_release_http_client() {
	client_name=$1
	application_container=$2
	docker run --detach \
		--name "${client_name}" \
		--network "container:${application_container}" \
		--pid "container:${application_container}" \
		--cap-add SYS_PTRACE \
		"${release_http_client_image}" >/dev/null
}
