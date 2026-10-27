#!/bin/sh
set -eu

output=${1:-/fixtures}
mkdir -p "${output}"

ffmpeg -hide_banner -loglevel error \
	-f lavfi -i 'testsrc2=size=96x64:rate=1' \
	-frames:v 1 -c:v mjpeg -q:v 2 "${output}/sample.jpg"
ffmpeg -hide_banner -loglevel error \
	-f lavfi -i 'testsrc2=size=96x64:rate=1' \
	-frames:v 1 -c:v png "${output}/sample.png"
ffmpeg -hide_banner -loglevel error \
	-f lavfi -i 'testsrc2=size=96x64:rate=1' \
	-frames:v 1 -c:v libwebp "${output}/sample.webp"
ffmpeg -hide_banner -loglevel error \
	-f lavfi -i 'testsrc2=size=96x64:rate=4' \
	-t 1 -c:v gif "${output}/sample.gif"

for extension in mp4 mov mkv; do
	ffmpeg -hide_banner -loglevel error \
		-f lavfi -i 'testsrc2=size=96x64:rate=12' \
		-t 1 -an -c:v libx264 -preset ultrafast -pix_fmt yuv420p \
		"${output}/sample.${extension}"
done
ffmpeg -hide_banner -loglevel error \
	-f lavfi -i 'testsrc2=size=96x64:rate=12' \
	-t 1 -an -c:v ffv1 "${output}/nonbrowser-ffv1.mkv"

head -c 64 "${output}/sample.mp4" >"${output}/truncated.mp4"
chmod 0777 "${output}"
