#!/usr/bin/env bash
set -euo pipefail

output_dir=${1:-build/intelligent-media-native}
source_commit=${SOURCE_COMMIT:-$(git rev-parse HEAD)}
ort_commit=da9b5e364c465de65c49d91e696cd6485270757f
detector_url=https://github.com/opencv/opencv_zoo/raw/47534e27c9851bb1128ccc0102f1145e27f23f98/models/face_detection_yunet/face_detection_yunet_2023mar.onnx
detector_sha=8f2383e4dd3cfbb4553ea8718107fc0423210dc964f9f4280604804ed2552fa4
embedder_url=https://huggingface.co/fal/AuraFace-v1/resolve/af6d057c9b0ec4071d4c49c80e3539258798b609/glintr100.onnx
embedder_sha=a7933ea5330113b01c9b60351d8f4c33003f145d8470ac5f0e52ee2effe25c60
fixture_url=https://github.com/opencv/opencv_zoo/raw/47534e27c9851bb1128ccc0102f1145e27f23f98/models/face_detection_yunet/example_outputs/largest_selfie.jpg
fixture_sha=ab8413ad9bb4f53068f4fb63c6747e5989991dd02241c923d5595b614ecf2bf6

test "$(uname -s)" = Linux
goarch=$(go env GOARCH)
machine=$(uname -m)
docker_arch=$(docker info --format '{{.Architecture}}')
case "${goarch}:${machine}:${docker_arch}" in
  amd64:x86_64:x86_64)
    ort_url=https://github.com/microsoft/onnxruntime/releases/download/v1.28.0/onnxruntime-linux-x64-1.28.0.tgz
    ort_sha=a3e1b79d7bb1bf09696ce675f49e4064e6c81f6202b8225624fff0e93f8d6407
    ;;
  arm64:aarch64:aarch64)
    ort_url=https://github.com/microsoft/onnxruntime/releases/download/v1.28.0/onnxruntime-linux-aarch64-1.28.0.tgz
    ort_sha=e15ff8b5d85afe6c144d97c6fd432254bf76a219daaf17658087d6ecb3e8f0bb
    ;;
  *)
    echo "native Linux architecture mismatch: go=${goarch} machine=${machine} docker=${docker_arch}" >&2
    exit 1
    ;;
esac

temporary=$(mktemp -d)
cleanup() {
  docker image rm "foliopath-face-candidate:${source_commit}-${goarch}" \
    "foliopath-build:${source_commit}-${goarch}" >/dev/null 2>&1 || true
  rm -rf "${temporary}"
}
trap cleanup EXIT

download() {
  local url=$1
  local output=$2
  local expected_sha=$3
  curl --fail --location --show-error --silent \
    --retry 5 --retry-all-errors --connect-timeout 30 --max-time 1200 \
    "${url}" --output "${output}"
  echo "${expected_sha}  ${output}" | sha256sum --check --strict -
}

mkdir -p "${output_dir}" "${temporary}/models" "${temporary}/fixture"
download "${detector_url}" "${temporary}/models/face_detection_yunet_2023mar.onnx" "${detector_sha}"
download "${embedder_url}" "${temporary}/models/glintr100.onnx" "${embedder_sha}"
download "${fixture_url}" "${temporary}/fixture/largest-selfie.jpg" "${fixture_sha}"

docker build --target build --progress plain \
  --tag "foliopath-build:${source_commit}-${goarch}" .
docker build --progress plain \
  --file tests/release/face_candidate_native.Dockerfile \
  --build-arg "BASE_IMAGE=foliopath-build:${source_commit}-${goarch}" \
  --build-arg "ORT_ARCHIVE_URL=${ort_url}" \
  --build-arg "ORT_ARCHIVE_SHA256=${ort_sha}" \
  --build-arg "ORT_COMMIT=${ort_commit}" \
  --tag "foliopath-face-candidate:${source_commit}-${goarch}" .

image_id=$(docker image inspect \
  "foliopath-face-candidate:${source_commit}-${goarch}" --format '{{.Id}}')
docker run --rm --network none --read-only \
  --tmpfs /tmp:rw,exec,nosuid,nodev,size=256m \
  --cpus 4 --memory 4g --pids-limit 256 \
  --env FOLIOPATH_ORT_FACE_DETECTOR=/models/face_detection_yunet_2023mar.onnx \
  --env FOLIOPATH_ORT_FACE_MODEL=/models/glintr100.onnx \
  --env FOLIOPATH_FACE_IMAGE=/fixture/largest-selfie.jpg \
  --volume "${temporary}/models:/models:ro" \
  --volume "${temporary}/fixture/largest-selfie.jpg:/fixture/largest-selfie.jpg:ro" \
  "foliopath-face-candidate:${source_commit}-${goarch}" \
  -test.v -test.run '^TestNativeFacePipelineCandidate$' \
  | tee "${output_dir}/face-candidate.log"

docker run --rm --network none --read-only \
  --tmpfs /tmp:rw,exec,nosuid,nodev,size=256m \
  --cpus 4 --memory 4g --pids-limit 256 \
  --env FOLIOPATH_RUN_CAPACITY_TEST=1 \
  --env GOMAXPROCS=4 \
  --entrypoint /out/face-capacity.test \
  "foliopath-face-candidate:${source_commit}-${goarch}" \
  -test.v -test.timeout=10m -test.run '^TestClusterFaces100KCapacity$' \
  | tee "${output_dir}/face-capacity.log"

candidate_count=$(sed -n 's/.*candidate_count=\([0-9][0-9]*\).*/\1/p' \
  "${output_dir}/face-candidate.log" | tail -n 1)
fingerprint=$(sed -n 's/.*quantized_1e3_sha256=\([0-9a-f]\{64\}\).*/\1/p' \
  "${output_dir}/face-candidate.log" | tail -n 1)
test -n "${candidate_count}"
test -n "${fingerprint}"

capacity_metric() {
  local name=$1
  sed -n "s/.*${name}=\\([^ ]*\\).*/\\1/p" \
    "${output_dir}/face-capacity.log" | tail -n 1
}
face_count=$(capacity_metric face_count)
embedding_dimension=$(capacity_metric embedding_dimension)
paired_cluster_count=$(capacity_metric paired_cluster_count)
paired_member_count=$(capacity_metric paired_member_count)
singleton_cluster_count=$(capacity_metric singleton_cluster_count)
singleton_member_count=$(capacity_metric singleton_member_count)
capacity_goos=$(capacity_metric goos)
capacity_goarch=$(capacity_metric goarch)
capacity_fingerprint=$(capacity_metric deterministic_sha256)
elapsed_ms=$(capacity_metric elapsed_ms)
memory_sys_bytes=$(capacity_metric memory_sys_bytes)
test "${face_count}" = 100000
test "${embedding_dimension}" = 512
test "${paired_cluster_count}" = 50000
test "${paired_member_count}" = 100000
test "${singleton_cluster_count}" = 100000
test "${singleton_member_count}" = 100000
test "${capacity_goos}" = linux
test "${capacity_goarch}" = "${goarch}"
test -n "${capacity_fingerprint}"
test "${elapsed_ms}" -gt 0
test "${memory_sys_bytes}" -gt 0
test "${memory_sys_bytes}" -le 3435973836

jq -n \
  --arg sourceCommit "${source_commit}" \
  --arg architecture "${goarch}" \
  --arg machine "${machine}" \
  --arg dockerArchitecture "${docker_arch}" \
  --arg imageId "${image_id}" \
  --arg faceCount "${face_count}" \
  --arg embeddingDimension "${embedding_dimension}" \
  --arg pairedClusterCount "${paired_cluster_count}" \
  --arg pairedMemberCount "${paired_member_count}" \
  --arg singletonClusterCount "${singleton_cluster_count}" \
  --arg singletonMemberCount "${singleton_member_count}" \
  --arg fingerprint "${capacity_fingerprint}" \
  --arg elapsedMillis "${elapsed_ms}" \
  --arg memorySysBytes "${memory_sys_bytes}" \
  '{
    schemaVersion: 1,
    evidenceClass: "synthetic-native-capacity-only",
    sourceCommit: $sourceCommit,
    architecture: $architecture,
    machine: $machine,
    dockerArchitecture: $dockerArchitecture,
    qemuAllowed: false,
    imageId: $imageId,
    containerCPUs: 4,
    containerMemoryBytes: 4294967296,
    faceCount: ($faceCount | tonumber),
    embeddingDimension: ($embeddingDimension | tonumber),
    pairedClusterCount: ($pairedClusterCount | tonumber),
    pairedMemberCount: ($pairedMemberCount | tonumber),
    singletonClusterCount: ($singletonClusterCount | tonumber),
    singletonMemberCount: ($singletonMemberCount | tonumber),
    deterministicSHA256: $fingerprint,
    elapsedMillis: ($elapsedMillis | tonumber),
    memorySysBytes: ($memorySysBytes | tonumber),
    identityGroundTruth: false,
    qualityGate: false,
    result: "passed"
  }' >"${output_dir}/face-capacity.json"

jq -n \
  --arg sourceCommit "${source_commit}" \
  --arg architecture "${goarch}" \
  --arg machine "${machine}" \
  --arg dockerArchitecture "${docker_arch}" \
  --arg imageId "${image_id}" \
  --arg ortCommit "${ort_commit}" \
  --arg ortArchiveSHA256 "${ort_sha}" \
  --arg detectorSHA256 "${detector_sha}" \
  --arg embedderSHA256 "${embedder_sha}" \
  --arg fixtureSHA256 "${fixture_sha}" \
  --arg fingerprint "${fingerprint}" \
  --arg candidateCount "${candidate_count}" \
  '{
    schemaVersion: 1,
    evidenceClass: "candidate-native-functional-preflight-only",
    sourceCommit: $sourceCommit,
    architecture: $architecture,
    machine: $machine,
    dockerArchitecture: $dockerArchitecture,
    qemuAllowed: false,
    imageId: $imageId,
    onnxRuntimeCommit: $ortCommit,
    onnxRuntimeArchiveSHA256: $ortArchiveSHA256,
    detectorSHA256: $detectorSHA256,
    embedderSHA256: $embedderSHA256,
    fixtureSHA256: $fixtureSHA256,
    candidateCount: ($candidateCount | tonumber),
    quantized1e3SHA256: $fingerprint,
    productionApproved: false,
    qualityGate: false,
    complianceGate: false,
    result: "passed"
  }' >"${output_dir}/face-candidate.json"
