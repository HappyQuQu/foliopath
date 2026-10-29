#!/bin/sh
set -eu

test "$#" -eq 2
container=$1
fixture_root=$2

docker exec "${container}" /opt/ffmpeg/bin/ffmpeg \
	-hide_banner -encoders 2>&1 | grep -q 'libwebp'

for extension in jpg png webp gif; do
	docker exec "${container}" /opt/vips/bin/vipsheader \
		"${fixture_root}/sample.${extension}" | grep -q '96x64'
done

for extension in mp4 mov mkv; do
	docker exec "${container}" /opt/ffmpeg/bin/ffmpeg \
		-hide_banner -loglevel error -y \
		-ss 0.25 -i "${fixture_root}/sample.${extension}" \
		-frames:v 1 -vf 'scale=48:-2' -f image2pipe \
		-c:v libwebp "${fixture_root}/poster-${extension}.webp"
	test "$(docker exec "${container}" /opt/ffmpeg/bin/ffprobe \
		-v error -select_streams v:0 \
		-show_entries stream=codec_name -of default=nw=1:nk=1 \
		"${fixture_root}/sample.${extension}")" = h264
	docker exec "${container}" /opt/vips/bin/vipsheader \
		"${fixture_root}/poster-${extension}.webp" | grep -q '48x32'
done

test "$(docker exec "${container}" /opt/ffmpeg/bin/ffprobe \
	-v error -select_streams v:0 \
	-show_entries stream=codec_name -of default=nw=1:nk=1 \
	"${fixture_root}/nonbrowser-ffv1.mkv")" = ffv1
docker exec "${container}" /opt/ffmpeg/bin/ffmpeg \
	-hide_banner -loglevel error -y \
	-ss 0.25 -i "${fixture_root}/nonbrowser-ffv1.mkv" \
	-frames:v 1 -vf 'scale=48:-2' -f image2pipe \
	-c:v libwebp "${fixture_root}/poster-ffv1-mkv.webp"
docker exec "${container}" /opt/vips/bin/vipsheader \
	"${fixture_root}/poster-ffv1-mkv.webp" | grep -q '48x32'

if docker exec "${container}" /opt/ffmpeg/bin/ffprobe \
	-v error "${fixture_root}/truncated.mp4" >/dev/null 2>&1; then
	echo 'ffprobe accepted truncated MP4' >&2
	exit 1
fi

docker exec "${container}" /usr/bin/prlimit --as=268435456 --cpu=60 -- \
    /usr/lib/7zip/7z a -t7z -ms=on -mmt=1 -bd \
    "${fixture_root}/images.7z" "${fixture_root}/sample.jpg" "${fixture_root}/sample.png" >/dev/null
docker exec "${container}" /usr/bin/prlimit --as=268435456 --cpu=60 -- \
    /usr/lib/7zip/7z l -slt -ba -t7z "${fixture_root}/images.7z" | grep -q '^Path = sample.jpg$'
source_hash=$(docker cp "${container}:${fixture_root}/sample.jpg" - | tar -xOf - | sha256sum | cut -d ' ' -f 1)
member_hash=$(docker exec "${container}" /usr/bin/prlimit --as=268435456 --cpu=60 -- \
    /usr/lib/7zip/7z x -so -spd -t7z -- "${fixture_root}/images.7z" sample.jpg | sha256sum | cut -d ' ' -f 1)
test "${source_hash}" = "${member_hash}"

printf '%s\n' "candidate image media matrix passed"
