#!/bin/sh

configure_release_storage() {
	release_data_storage=${FOLIOPATH_TEST_STORAGE:-volume}
	case "${release_data_storage}" in
	volume) ;;
	bind)
		if [ "$(uname -s)" != Linux ] || docker info --format '{{.OperatingSystem}}' | grep -qi 'Docker Desktop'; then
			echo 'bind test data requires native Linux; use FOLIOPATH_TEST_STORAGE=volume on Docker Desktop' >&2
			return 2
		fi
		;;
	*)
		echo 'FOLIOPATH_TEST_STORAGE must be volume or bind' >&2
		return 2
		;;
	esac
}

release_data_volume() {
	printf '%s-%s-data' "${container}" "$(basename "$1")"
}

create_release_data() {
	if [ "${release_data_storage}" = volume ]; then
		docker volume create --label "foliopath.test.run=${FOLIOPATH_TEST_RUN_ID:-unassigned}" \
			"$(release_data_volume "$1")" >/dev/null
	else
		mkdir -p "$1"
		chmod 0777 "$1"
	fi
}

remove_release_data() {
	if [ "${release_data_storage}" = volume ]; then
		docker volume rm "$(release_data_volume "$1")" >/dev/null 2>&1 || true
	fi
}

release_data_mount() {
	if [ "${release_data_storage}" = volume ]; then
		printf 'type=volume,src=%s,dst=%s,volume-nocopy%s' \
			"$(release_data_volume "$1")" "$2" "${3:-}"
	else
		printf 'type=bind,src=%s,dst=%s%s' "$1" "$2" "${3:-}"
	fi
}

backup_release_data() {
	docker run --rm --network none --user 0:0 --entrypoint sh \
		--mount "$(release_data_mount "$1" /from ,readonly)" \
		--mount "type=bind,src=${backup_root},dst=/backup" \
		"${release_http_client_image}" -ec \
		'cd /from; find . -type f -exec sha256sum {} + | LC_ALL=C sort >/backup/source.sha256; tar -cf /backup/data.tar .'
}

restore_release_data() {
	docker run --rm --network none --user 0:0 --entrypoint sh \
		--mount "type=bind,src=${backup_root},dst=/backup,readonly" \
		--mount "$(release_data_mount "$1" /restore)" \
		"${release_http_client_image}" -ec \
		'cd /restore; tar -xf /backup/data.tar; find . -type f -exec sha256sum {} + | LC_ALL=C sort >/tmp/restored.sha256; cmp /backup/source.sha256 /tmp/restored.sha256; sha256sum --check /backup/source.sha256'
}

release_database_hash() {
	docker run --rm --network none --user 0:0 --entrypoint sh \
		--mount "$(release_data_mount "$1" /data ,readonly)" \
		"${release_http_client_image}" -ec 'sha256sum /data/foliopath.db' | cut -d ' ' -f 1
}
