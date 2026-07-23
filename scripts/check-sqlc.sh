#!/bin/sh
set -eu

repository_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
temporary_root=$(mktemp -d "${TMPDIR:-/tmp}/foliopath-sqlc.XXXXXX")
trap 'rm -rf "$temporary_root"' EXIT HUP INT TERM

mkdir -p "$temporary_root/internal/store/sqlite"
cp -R "$repository_root/migrations" "$temporary_root/migrations"
cp "$repository_root/internal/store/sqlite/sqlc.yaml" "$temporary_root/internal/store/sqlite/sqlc.yaml"
cp -R "$repository_root/internal/store/sqlite/queries" "$temporary_root/internal/store/sqlite/queries"

(
	cd "$temporary_root"
	"${GO:-go}" run "github.com/sqlc-dev/sqlc/cmd/sqlc@${SQLC_VERSION:-v1.31.1}" generate \
		-f internal/store/sqlite/sqlc.yaml
)

diff -ru \
	"$repository_root/internal/store/sqlite/dbgen" \
	"$temporary_root/internal/store/sqlite/dbgen"
