#!/bin/sh
set -eu
# Synthetic staging only; never accepts a user media directory or overwrites it.
fixture_root=$(mktemp -d /tmp/foliopath-app-fixtures.XXXXXX)
repository=$(CDPATH= cd -- "$(dirname "$0")/../.." && pwd)
cp -p "$repository/tests/fixtures/media/viewer-blue-violet.png" "$fixture_root/"
cp -p "$repository/tests/fixtures/media/range-video.mp4" "$fixture_root/"
ffmpeg -hide_banner -loglevel error -f lavfi -i 'testsrc2=size=160x96:rate=12' \
  -t 3 -an -c:v libx264 -preset ultrafast -pix_fmt yuv420p "$fixture_root/storyboard-test.mp4"
printf '%s\n' "$fixture_root"
