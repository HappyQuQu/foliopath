#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
work_root=$(mktemp -d "${TMPDIR:-/tmp}/foliopath-storyboard-ffmpeg.XXXXXX")
image_prefix="foliopath-storyboard-ffmpeg-$$"

cleanup() {
	for platform in arm64 amd64; do
		docker image rm --force "${image_prefix}-${platform}" >/dev/null 2>&1 || true
		docker image rm --force "${image_prefix}-${platform}-runtime" >/dev/null 2>&1 || true
	done
	rm -rf -- "${work_root}"
}
trap cleanup EXIT HUP INT TERM

command -v docker >/dev/null
command -v ffmpeg >/dev/null
command -v ffprobe >/dev/null

av1_encoder=""
for candidate in libsvtav1 libaom-av1 librav1e; do
	if ffmpeg -hide_banner -encoders 2>&1 | grep -F "${candidate}" >/dev/null; then
		av1_encoder=${candidate}
		break
	fi
done
test -n "${av1_encoder}"

ffmpeg -hide_banner -loglevel error \
	-f lavfi -i "testsrc2=size=320x180:rate=24:duration=10" \
	-c:v mpeg4 -pix_fmt yuv420p -g 24 -y "${work_root}/input.mp4"
source_before=$(sha256sum "${work_root}/input.mp4" | cut -d ' ' -f 1)
ffmpeg -hide_banner -loglevel error \
	-f lavfi -i "testsrc2=size=64x64:rate=1:duration=1" \
	-c:v "${av1_encoder}" -pix_fmt yuv420p \
	-y "${work_root}/input-av1.mp4"
av1_source_before=$(sha256sum "${work_root}/input-av1.mp4" | cut -d ' ' -f 1)
ffmpeg -hide_banner -loglevel error \
	-f lavfi -i "testsrc2=size=128x72:rate=24:duration=2" \
	-f lavfi -i "sine=frequency=1000:duration=2" \
	-c:v mpeg2video -c:a mp2 -shortest -f mpegts \
	-y "${work_root}/input-mpegts-as-mp4.mp4"
mpegts_source_before=$(sha256sum "${work_root}/input-mpegts-as-mp4.mp4" | cut -d ' ' -f 1)

for platform in arm64 amd64; do
	image="${image_prefix}-${platform}"
	docker buildx build \
		--platform "linux/${platform}" \
		--target ffmpeg \
		--tag "${image}" \
		--load \
		"${repo_root}"
	runtime_image="${image}-runtime"
	docker buildx build \
		--platform "linux/${platform}" \
		--tag "${runtime_image}" \
		--load \
		"${repo_root}"

	for capability in \
		"encoders: png " \
		"encoders: libwebp " \
		"decoders: libdav1d " \
		"decoders: png " \
		"demuxers: image2 " \
		"demuxers: mpegts " \
		"filters: scale " \
		"filters: setsar " \
		"filters: xstack "
	do
		group=${capability%%:*}
		name=${capability#*: }
		docker run --rm --platform "linux/${platform}" "${image}" \
			/opt/ffmpeg-build/bin/ffmpeg -hide_banner "-${group}" 2>&1 |
			grep -F "${name}" >/dev/null
	done

	docker run --rm --platform "linux/${platform}" \
		-v "${work_root}:/work" \
		"${image}" /bin/sh -ec '
			ffmpeg=/opt/ffmpeg-build/bin/ffmpeg
			index=0
			for timestamp in 0.5 1.5 2.5 3.5 4.5 5.5 6.5 7.5 8.5 9.5; do
				frame=$(printf "/work/'"${platform}"'-frame-%02d.png" "${index}")
				"${ffmpeg}" -nostdin -v error -threads 1 -filter_threads 1 \
					-ss "${timestamp}" -i /work/input.mp4 -map 0:v:0 -frames:v 1 \
					-vf scale=160:90:flags=lanczos,setsar=1 -an \
					-f image2pipe -vcodec png "${frame}"
				index=$((index + 1))
			done
			"${ffmpeg}" -nostdin -v error -threads 1 -filter_threads 1 \
				-i /work/'"${platform}"'-frame-00.png \
				-i /work/'"${platform}"'-frame-01.png \
				-i /work/'"${platform}"'-frame-02.png \
				-i /work/'"${platform}"'-frame-03.png \
				-i /work/'"${platform}"'-frame-04.png \
				-i /work/'"${platform}"'-frame-05.png \
				-i /work/'"${platform}"'-frame-06.png \
				-i /work/'"${platform}"'-frame-07.png \
				-i /work/'"${platform}"'-frame-08.png \
				-i /work/'"${platform}"'-frame-09.png \
				-filter_complex \
				"[0:v][1:v][2:v][3:v][4:v][5:v][6:v][7:v][8:v][9:v]xstack=inputs=10:layout=0_0|160_0|320_0|480_0|640_0|0_90|160_90|320_90|480_90|640_90:fill=black[v]" \
				-map "[v]" -frames:v 1 -an -f image2pipe \
				-vcodec libwebp -q:v 75 /work/storyboard-'"${platform}"'.webp
		'
	docker run --rm --platform "linux/${platform}" \
		-v "${work_root}:/work" \
		--entrypoint /opt/ffmpeg/bin/ffmpeg \
		"${runtime_image}" \
			-nostdin -v error -threads 1 -filter_threads 1 \
			-i /work/input-av1.mp4 -map 0:v:0 -frames:v 1 \
			-vf scale=64:64:flags=lanczos,setsar=1 -an \
			-f image2pipe -vcodec png "/work/av1-frame-${platform}.png"
	actual_container=$(docker run --rm --platform "linux/${platform}" \
		-v "${work_root}:/work" \
		--entrypoint /opt/ffmpeg/bin/ffprobe \
		"${runtime_image}" \
			-v error -show_entries format=format_name \
			-of default=noprint_wrappers=1:nokey=1 \
			/work/input-mpegts-as-mp4.mp4)
	test "${actual_container}" = "mpegts"
	docker run --rm --platform "linux/${platform}" \
		-v "${work_root}:/work" \
		--entrypoint /opt/ffmpeg/bin/ffmpeg \
		"${runtime_image}" \
			-nostdin -v error -threads 1 -filter_threads 1 \
			-i /work/input-mpegts-as-mp4.mp4 -map 0:v:0 -frames:v 1 \
			-vf scale=64:64:flags=lanczos,setsar=1 -an \
			-f image2pipe -vcodec png "/work/mpegts-frame-${platform}.png"

	dimensions=$(ffprobe -v error -select_streams v:0 \
		-show_entries stream=width,height \
		-of csv=p=0:s=x "${work_root}/storyboard-${platform}.webp")
	test "${dimensions}" = "800x180"
	av1_dimensions=$(ffprobe -v error -select_streams v:0 \
		-show_entries stream=width,height \
		-of csv=p=0:s=x "${work_root}/av1-frame-${platform}.png")
	test "${av1_dimensions}" = "64x64"
	mpegts_dimensions=$(ffprobe -v error -select_streams v:0 \
		-show_entries stream=width,height \
		-of csv=p=0:s=x "${work_root}/mpegts-frame-${platform}.png")
	test "${mpegts_dimensions}" = "64x64"
	test "$(wc -c <"${work_root}/storyboard-${platform}.webp")" -le 8388608
done

source_after=$(sha256sum "${work_root}/input.mp4" | cut -d ' ' -f 1)
test "${source_before}" = "${source_after}"
av1_source_after=$(sha256sum "${work_root}/input-av1.mp4" | cut -d ' ' -f 1)
test "${av1_source_before}" = "${av1_source_after}"
mpegts_source_after=$(sha256sum "${work_root}/input-mpegts-as-mp4.mp4" | cut -d ' ' -f 1)
test "${mpegts_source_before}" = "${mpegts_source_after}"

printf '%s\n' \
	"storyboard FFmpeg runtime smoke passed for linux/arm64 and linux/amd64"
