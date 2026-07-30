#!/bin/sh
set -eu

image=${1:?usage: collect-release-notices.sh IMAGE OUTPUT_DIRECTORY}
output_dir=${2:?usage: collect-release-notices.sh IMAGE OUTPUT_DIRECTORY}
container=

cleanup() {
  if [ -n "${container}" ]; then
    docker rm "${container}" >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT HUP INT TERM

mkdir -p \
  "${output_dir}/usr/share" \
  "${output_dir}/opt/expat/share" \
  "${output_dir}/opt/ffmpeg/share" \
  "${output_dir}/opt/glib/share" \
  "${output_dir}/opt/vips/share" \
  "${output_dir}/var/lib/dpkg"
test ! -e "${output_dir}/SHA256SUMS"

container=$(docker create "${image}")
docker cp "${container}:/usr/share/doc" "${output_dir}/usr/share/"
docker cp "${container}:/opt/expat/share/licenses" \
  "${output_dir}/opt/expat/share/"
docker cp "${container}:/opt/ffmpeg/share/licenses" \
  "${output_dir}/opt/ffmpeg/share/"
docker cp "${container}:/opt/glib/share/licenses" \
  "${output_dir}/opt/glib/share/"
docker cp "${container}:/opt/vips/share/licenses" \
  "${output_dir}/opt/vips/share/"
docker cp "${container}:/var/lib/dpkg/status.d" \
  "${output_dir}/var/lib/dpkg/"

(
  cd "${output_dir}"
  find . -type f ! -name SHA256SUMS -print0 |
    LC_ALL=C sort -z |
    xargs -0 sha256sum
) >"${output_dir}/SHA256SUMS"

test -s "${output_dir}/opt/expat/share/licenses/expat/COPYING"
test -s "${output_dir}/opt/ffmpeg/share/licenses/ffmpeg/COPYING.LGPLv2.1"
test -s "${output_dir}/opt/glib/share/licenses/glib/COPYING"
test -s "${output_dir}/opt/vips/share/licenses/libvips/LICENSE"
test -s "${output_dir}/var/lib/dpkg/status.d/foliopath-expat"
test -s "${output_dir}/var/lib/dpkg/status.d/foliopath-ffmpeg"
test -s "${output_dir}/var/lib/dpkg/status.d/foliopath-glib"
test -s "${output_dir}/var/lib/dpkg/status.d/foliopath-libvips"
printf '%s\n' "release notices collected for ${image}"
