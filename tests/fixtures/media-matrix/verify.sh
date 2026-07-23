#!/usr/bin/env bash

set -euo pipefail

for required_tool in ffmpeg ffprobe cwebp; do
  if ! command -v "${required_tool}" >/dev/null 2>&1; then
    printf 'missing required fixture tool: %s\n' "${required_tool}" >&2
    exit 2
  fi
done

fixture_dir="$(mktemp -d "${TMPDIR:-/tmp}/foliopath-fs03.XXXXXX")"

cleanup() {
  case "${fixture_dir}" in
    "${TMPDIR:-/tmp}"/foliopath-fs03.*)
      rm -rf -- "${fixture_dir}"
      ;;
    *)
      printf 'refusing to clean unexpected fixture path: %s\n' "${fixture_dir}" >&2
      ;;
  esac
}
trap cleanup EXIT

ffmpeg_common=(-hide_banner -loglevel error)

ffmpeg "${ffmpeg_common[@]}" \
  -f lavfi -i "testsrc2=size=96x64:rate=1" \
  -frames:v 1 -c:v mjpeg -q:v 2 \
  "${fixture_dir}/sample.jpg"

ffmpeg "${ffmpeg_common[@]}" \
  -f lavfi -i "testsrc2=size=96x64:rate=1" \
  -frames:v 1 -c:v png \
  "${fixture_dir}/sample.png"

cwebp -quiet "${fixture_dir}/sample.png" -o "${fixture_dir}/sample.webp"

ffmpeg "${ffmpeg_common[@]}" \
  -f lavfi -i "testsrc2=size=96x64:rate=4" \
  -t 1 -c:v gif \
  "${fixture_dir}/sample.gif"

for extension in mp4 mov mkv; do
  ffmpeg "${ffmpeg_common[@]}" \
    -f lavfi -i "testsrc2=size=96x64:rate=12" \
    -t 1 -an -c:v libx264 -preset ultrafast -pix_fmt yuv420p \
    "${fixture_dir}/sample.${extension}"
done

ffmpeg "${ffmpeg_common[@]}" \
  -f lavfi -i "testsrc2=size=96x64:rate=12" \
  -t 1 -an -c:v ffv1 \
  "${fixture_dir}/nonbrowser-ffv1.mkv"

assert_probe() {
  local path="$1"
  local expected_codec="$2"
  local expected_dimensions="${3:-96x64}"
  local actual_codec
  local dimensions

  actual_codec="$(
    ffprobe -v error -select_streams v:0 \
      -show_entries stream=codec_name \
      -of default=noprint_wrappers=1:nokey=1 \
      "${path}"
  )"
  dimensions="$(
    ffprobe -v error -select_streams v:0 \
      -show_entries stream=width,height \
      -of csv=p=0:s=x \
      "${path}"
  )"

  if [[ "${actual_codec}" != "${expected_codec}" ]]; then
    printf '%s: codec %s, expected %s\n' \
      "$(basename "${path}")" "${actual_codec}" "${expected_codec}" >&2
    return 1
  fi
  if [[ "${dimensions}" != "${expected_dimensions}" ]]; then
    printf '%s: dimensions %s, expected %s\n' \
      "$(basename "${path}")" "${dimensions}" "${expected_dimensions}" >&2
    return 1
  fi

  printf '%s codec=%s dimensions=%s\n' \
    "$(basename "${path}")" "${actual_codec}" "${dimensions}"
}

assert_probe "${fixture_dir}/sample.jpg" "mjpeg"
assert_probe "${fixture_dir}/sample.png" "png"
assert_probe "${fixture_dir}/sample.webp" "webp"
assert_probe "${fixture_dir}/sample.gif" "gif"
assert_probe "${fixture_dir}/sample.mp4" "h264"
assert_probe "${fixture_dir}/sample.mov" "h264"
assert_probe "${fixture_dir}/sample.mkv" "h264"
assert_probe "${fixture_dir}/nonbrowser-ffv1.mkv" "ffv1"

assert_video_metadata() {
  local path="$1"
  local expected_format_fragment="$2"
  local format_name
  local pixel_format
  local duration

  format_name="$(
    ffprobe -v error \
      -show_entries format=format_name \
      -of default=noprint_wrappers=1:nokey=1 \
      "${path}"
  )"
  pixel_format="$(
    ffprobe -v error -select_streams v:0 \
      -show_entries stream=pix_fmt \
      -of default=noprint_wrappers=1:nokey=1 \
      "${path}"
  )"
  duration="$(
    ffprobe -v error \
      -show_entries format=duration \
      -of default=noprint_wrappers=1:nokey=1 \
      "${path}"
  )"

  if [[ "${format_name}" != *"${expected_format_fragment}"* ]]; then
    printf '%s: container %s does not include %s\n' \
      "$(basename "${path}")" "${format_name}" "${expected_format_fragment}" >&2
    return 1
  fi
  if [[ "${pixel_format}" != "yuv420p" ]]; then
    printf '%s: pixel format %s, expected yuv420p\n' \
      "$(basename "${path}")" "${pixel_format}" >&2
    return 1
  fi
  if [[ "${duration}" != "1.000000" ]]; then
    printf '%s: duration %s, expected 1.000000\n' \
      "$(basename "${path}")" "${duration}" >&2
    return 1
  fi

  printf '%s container=%s pixel_format=%s duration=%s\n' \
    "$(basename "${path}")" "${format_name}" "${pixel_format}" "${duration}"
}

assert_video_metadata "${fixture_dir}/sample.mp4" "mp4"
assert_video_metadata "${fixture_dir}/sample.mov" "mov"
assert_video_metadata "${fixture_dir}/sample.mkv" "matroska"
assert_video_metadata "${fixture_dir}/nonbrowser-ffv1.mkv" "matroska"

mp4_brand="$(
  ffprobe -v error \
    -show_entries format_tags=major_brand \
    -of default=noprint_wrappers=1:nokey=1 \
    "${fixture_dir}/sample.mp4"
)"
mov_brand="$(
  ffprobe -v error \
    -show_entries format_tags=major_brand \
    -of default=noprint_wrappers=1:nokey=1 \
    "${fixture_dir}/sample.mov"
)"
if [[ -z "${mp4_brand}" || "${mp4_brand}" == qt* ]]; then
  printf 'sample.mp4: unexpected major_brand %q\n' "${mp4_brand}" >&2
  exit 1
fi
if [[ "${mov_brand}" != qt* ]]; then
  printf 'sample.mov: major_brand %q does not identify QuickTime\n' "${mov_brand}" >&2
  exit 1
fi
printf 'sample.mp4 major_brand=%q\n' "${mp4_brand}"
printf 'sample.mov major_brand=%q\n' "${mov_brand}"

gif_frames="$(
  ffprobe -v error -select_streams v:0 \
    -show_entries stream=nb_frames \
    -of default=noprint_wrappers=1:nokey=1 \
    "${fixture_dir}/sample.gif"
)"
if [[ ! "${gif_frames}" =~ ^[0-9]+$ ]] || ((gif_frames < 2)); then
  printf 'sample.gif: expected multiple frames, got %s\n' "${gif_frames}" >&2
  exit 1
fi
printf 'sample.gif frames=%s\n' "${gif_frames}"

for extension in mp4 mov mkv; do
  ffmpeg "${ffmpeg_common[@]}" \
    -ss 0.25 -i "${fixture_dir}/sample.${extension}" \
    -frames:v 1 -vf "scale=48:-2" -c:v png \
    "${fixture_dir}/poster-${extension}.png"
  assert_probe "${fixture_dir}/poster-${extension}.png" "png" "48x32"
done

ffmpeg "${ffmpeg_common[@]}" \
  -ss 0.25 -i "${fixture_dir}/nonbrowser-ffv1.mkv" \
  -frames:v 1 -vf "scale=48:-2" -c:v png \
  "${fixture_dir}/poster-ffv1-mkv.png"
assert_probe "${fixture_dir}/poster-ffv1-mkv.png" "png" "48x32"

head -c 64 "${fixture_dir}/sample.mp4" >"${fixture_dir}/truncated.mp4"
if ffprobe -v error "${fixture_dir}/truncated.mp4" >/dev/null 2>&1; then
  printf 'truncated.mp4: ffprobe unexpectedly accepted corrupt input\n' >&2
  exit 1
fi
printf 'truncated.mp4 rejected=true\n'

for path in \
  "${fixture_dir}"/sample.{jpg,png,webp,gif,mp4,mov,mkv} \
  "${fixture_dir}"/nonbrowser-ffv1.mkv; do
  byte_count="$(wc -c <"${path}" | tr -d '[:space:]')"
  printf '%s bytes=%s\n' "$(basename "${path}")" "${byte_count}"
done

printf 'FS-03 synthetic FFmpeg/ffprobe matrix passed\n'
