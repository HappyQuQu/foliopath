#!/bin/sh
set -eu

image=${1:?usage: verify.sh IMAGE}
helper_image="debian:bookworm-slim@sha256:7b140f374b289a7c2befc338f42ebe6441b7ea838a042bbd5acbfca6ec875818"
run_id="fs05-$$"
container="${run_id}-server"
source_volume="${run_id}-source"
backup_volume="${run_id}-backup"
restore_volume="${run_id}-restore"
library_dir=$(mktemp -d)

cleanup() {
  docker rm -f "$container" >/dev/null 2>&1 || true
  docker volume rm "$source_volume" "$backup_volume" "$restore_volume" >/dev/null 2>&1 || true
  rm -rf "$library_dir"
}
trap cleanup EXIT INT TERM

mkdir -p "$library_dir/recovery"
printf 'synthetic read-only media\n' >"$library_dir/recovery/example.jpg"
docker volume create "$source_volume" >/dev/null
docker volume create "$backup_volume" >/dev/null
docker volume create "$restore_volume" >/dev/null

docker run --rm \
  --mount "type=volume,src=$source_volume,dst=/app/data" \
  "$image" seed "Recovery Library"

docker run -d --name "$container" \
  --read-only --tmpfs /tmp:rw,noexec,nosuid,size=16m \
  --cap-drop ALL --security-opt no-new-privileges:true \
  --mount "type=bind,src=$library_dir,dst=/library,readonly" \
  --mount "type=volume,src=$source_volume,dst=/app/data" \
  "$image" >/dev/null

attempt=0
until [ "$(docker inspect -f '{{.State.Health.Status}}' "$container")" = "healthy" ]; do
  attempt=$((attempt + 1))
  if [ "$attempt" -ge 20 ]; then
    docker logs "$container"
    exit 1
  fi
  sleep 1
done

test "$(docker exec "$container" id -u)" = "65532"
test "$(docker exec "$container" id -g)" = "65532"
test "$(docker exec "$container" sh -c "awk '/CapEff/ {print \$2}' /proc/1/status")" = "0000000000000000"
if docker exec "$container" sh -c 'touch /library/must-not-write' 2>/dev/null; then
  echo "read-only library write unexpectedly succeeded" >&2
  exit 1
fi
docker exec "$container" sh -c 'touch /app/data/write-probe && rm /app/data/write-probe'
docker stop --time 8 "$container" >/dev/null
docker logs "$container" 2>&1 | grep 'graceful shutdown complete' >/dev/null
docker rm "$container" >/dev/null

# Offline backup copies the complete stopped SQLite family and permissions.
docker run --rm \
  --mount "type=volume,src=$source_volume,dst=/from,readonly" \
  --mount "type=volume,src=$backup_volume,dst=/backup" \
  "$helper_image" \
  sh -c 'cd /from && tar cf /backup/data.tar .'
docker run --rm \
  --mount "type=volume,src=$backup_volume,dst=/backup,readonly" \
  --mount "type=volume,src=$restore_volume,dst=/restore" \
  "$helper_image" \
  sh -c 'cd /restore && tar xf /backup/data.tar'
docker run --rm \
  --mount "type=volume,src=$restore_volume,dst=/app/data" \
  "$image" verify "Recovery Library"
# A repeated startup represents the current no-op upgrade path and migration idempotence.
docker run --rm \
  --mount "type=volume,src=$restore_volume,dst=/app/data" \
  "$image" verify "Recovery Library"

# Read-only data, a full tiny tmpfs, and corrupt SQLite must all fail closed.
if docker run --rm --read-only \
  --mount "type=volume,src=$restore_volume,dst=/app/data,readonly" \
  "$image" verify "Recovery Library"; then
  echo "read-only data directory unexpectedly accepted writes" >&2
  exit 1
fi
if docker run --rm --tmpfs /app/data:rw,noexec,nosuid,size=64k \
  --entrypoint sh "$image" -c \
  'dd if=/dev/zero of=/app/data/fill bs=1024 count=64 2>/dev/null || true; exec /app/foliopath-runtime-spike verify'; then
  echo "full data directory unexpectedly became ready" >&2
  exit 1
fi
docker volume rm "$restore_volume" >/dev/null
docker volume create "$restore_volume" >/dev/null
docker run --rm \
  --mount "type=volume,src=$restore_volume,dst=/app/data" \
  --entrypoint sh "$image" -c \
  'printf not-a-sqlite-database > /app/data/foliopath.db'
if docker run --rm \
  --mount "type=volume,src=$restore_volume,dst=/app/data" \
  "$image" verify "Recovery Library"; then
  echo "corrupt database unexpectedly passed verification" >&2
  exit 1
fi

echo "FS05_RUNTIME_OK image=$image"
