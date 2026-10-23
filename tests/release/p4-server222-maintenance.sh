#!/usr/bin/env bash
# Explicitly authorized, single-host maintenance runner. Never called by make/CI.
set -euo pipefail
test "${FOLIOPATH_ALLOW_P4_MAINTENANCE:-}" = server222
test "$(uname -m)" = x86_64
run_root=${1:?isolated run root required}
case "$run_root" in /tmp/foliopath-p4-native.*) ;; *) exit 2;; esac
test -d "$run_root/source"
run_id="p4-native-${run_root##*.}"
builder="$run_id"
build_proxy=${FOLIOPATH_P4_BUILD_PROXY:?explicit build-only proxy required}
cd "$run_root/source"
names=(foliopath-foliopath-1 immich_server immich_postgres immich_redis)
for name in "${names[@]}"; do
  test "$(docker inspect --format '{{.State.Running}}' "$name")" = true
done
test "$(docker ps -q | wc -l)" -eq 4
docker inspect --format '{{.Name}} {{.Image}} {{.State.Running}}' "${names[@]}" > "$run_root/services-before.txt"
mkdir -p "$run_root/results" "$run_root/tmp"
restore() {
  result=$?
  trap - EXIT INT TERM HUP
  set +e
  docker buildx rm "$builder" 2>/dev/null
  # Only containers created for this exact test run; never existing application data.
  for id in $(docker ps -aq --filter "label=foliopath.test.run=$run_id"); do
    docker rm -f "$id"
  done
  docker start immich_postgres immich_redis
  docker start immich_server foliopath-foliopath-1
  restored=0
  for attempt in $(seq 1 120); do
    restored=1
    for name in "${names[@]}"; do
      state=$(docker inspect --format '{{.State.Running}} {{if .State.Health}}{{.State.Health.Status}}{{else}}healthy{{end}}' "$name")
      test "$state" = 'true healthy' || restored=0
    done
    test "$restored" = 1 && break
    sleep 1
  done
  docker inspect --format '{{.Name}} {{.Image}} {{.State.Running}}' "${names[@]}" > "$run_root/services-after.txt"
  cmp "$run_root/services-before.txt" "$run_root/services-after.txt" || restored=0
  printf 'test_exit=%s\nservices_restored=%s\n' "$result" "$restored" | tee "$run_root/result.txt"
  test "$restored" = 1 || exit 90
  exit "$result"
}
trap restore EXIT
trap 'exit 130' INT
trap 'exit 143' TERM HUP
docker stop --time 60 foliopath-foliopath-1 immich_server
docker stop --time 60 immich_postgres immich_redis
free -h
if [ "${FOLIOPATH_P4_REUSE_TOOLS_ID:-}" != '' ]; then
  test "$(docker image inspect --format '{{.Id}}' foliopath-p4-native:app)" = "${FOLIOPATH_P4_REUSE_APP_ID:?app ID required}"
  test "$(docker image inspect --format '{{.Id}}' foliopath-p4-native:ai)" = "${FOLIOPATH_P4_REUSE_AI_ID:?AI ID required}"
  test "$(docker image inspect --format '{{.Id}}' foliopath-p4-native:tools)" = "$FOLIOPATH_P4_REUSE_TOOLS_ID"
else
HTTP_PROXY="$build_proxy" HTTPS_PROXY="$build_proxy" docker buildx create --name "$builder" --driver docker-container \
  --buildkitd-config tests/release/p4-buildkit.toml \
  --driver-opt image=foliopath-p4-buildkit:local \
  --driver-opt "env.HTTP_PROXY=$build_proxy" --driver-opt "env.HTTPS_PROXY=$build_proxy" \
  --driver-opt env.NO_PROXY=localhost --driver-opt memory=3g --bootstrap
build() {
  HTTP_PROXY="$build_proxy" HTTPS_PROXY="$build_proxy" docker buildx build --builder "$builder" --load \
    --build-arg "HTTP_PROXY=$build_proxy" --build-arg "HTTPS_PROXY=$build_proxy" \
    --build-arg BUILDKIT_SYNTAX=docker/dockerfile:1.7@sha256:a57df69d0ea827fb7266491f2813635de6f17269be881f696fbfdf2d83dda33e "$@"
}
if [ "${FOLIOPATH_P4_REUSE_APP_ID:-}" != '' ]; then
  # Explicit IDs permit a tools-only repair without silently testing old code.
  test "$(docker image inspect --format '{{.Id}}' foliopath-p4-native:app)" = "$FOLIOPATH_P4_REUSE_APP_ID"
  test "$(docker image inspect --format '{{.Id}}' foliopath-p4-native:ai)" = "${FOLIOPATH_P4_REUSE_AI_ID:?paired AI image ID required}"
else
  build -t foliopath-p4-native:app .
  build -f services/inference/Dockerfile --build-arg FOLIOPATH_INCLUDE_FACE=1 -t foliopath-p4-native:ai .
fi
build -f tests/release/p4-native-tools.Dockerfile -t foliopath-p4-native:tools .
build -f tests/fixtures/http-client/Dockerfile -t foliopath-release-http-client:local tests/fixtures/http-client
docker buildx stop "$builder"
fi
docker image inspect --format '{{.Id}} {{.Os}}/{{.Architecture}}' foliopath-p4-native:app foliopath-p4-native:ai > "$run_root/results/images.txt"
sample_args=()
if [ "${FOLIOPATH_P4_SMALL_ROOT:-}" != '' ]; then
  case "$FOLIOPATH_P4_SMALL_ROOT" in /tmp/foliopath-p4-small.*) ;; *) exit 2;; esac
  test -f "$FOLIOPATH_P4_SMALL_ROOT/manifest.json"
  sample_args=(-v "$FOLIOPATH_P4_SMALL_ROOT:$FOLIOPATH_P4_SMALL_ROOT:ro"
    -e "FOLIOPATH_SMALL_SAMPLE_ROOT=$FOLIOPATH_P4_SMALL_ROOT"
    -e "FOLIOPATH_SMALL_MANIFEST=$FOLIOPATH_P4_SMALL_ROOT/manifest.json"
    -e "FOLIOPATH_SMALL_REPORT=$run_root/results/small-report.json")
fi
driver() {
  docker run --rm --name foliopath-p4-native-driver --network host --memory 512m \
    --label "foliopath.test.run=$run_id" -e "FOLIOPATH_TEST_RUN_ID=$run_id" \
    -v /var/run/docker.sock:/var/run/docker.sock \
    -v "$run_root:$run_root" \
    -v /root/immich/model-cache:/root/immich/model-cache:ro \
    -w "$run_root/source" -e "TMPDIR=$run_root/tmp" \
    -e FOLIOPATH_RELEASE_BUILD_HELPERS=0 \
    -e FOLIOPATH_RELEASE_IMAGE=foliopath-p4-native:app \
    -e FOLIOPATH_CAPACITY=1 -e FOLIOPATH_CAPACITY_ENFORCE_BUDGET=1 \
    -e "FOLIOPATH_CAPACITY_METRICS_OUTPUT=$run_root/results/capacity.json" \
    -e FOLIOPATH_P3_APP_IMAGE=foliopath-p4-native:app \
    -e FOLIOPATH_P3_AI_IMAGE=foliopath-p4-native:ai \
    -e "FOLIOPATH_TEST_RESOURCE_PROFILE=${FOLIOPATH_P4_RESOURCE_PROFILE:-p4-native}" \
    -e FOLIOPATH_TEST_MODEL_CACHE=/root/immich/model-cache \
    "${sample_args[@]}" foliopath-p4-native:tools "$@"
}
driver node spikes/int001-remote-ml/cache-audit.mjs | tee "$run_root/results/cache.json"
if [ "${FOLIOPATH_P4_SMALL_ROOT:-}" != '' ]; then
  driver node spikes/int001-remote-ml/p4-small-probe.mjs | tee "$run_root/results/small.log"
else
driver sh tests/release/capacity_smoke.sh | tee "$run_root/results/capacity.log"
driver node spikes/int001-remote-ml/p3-business-probe.mjs | tee "$run_root/results/business.log"
driver node spikes/int001-remote-ml/p3-recovery-probe.mjs | tee "$run_root/results/recovery.log"
fi
