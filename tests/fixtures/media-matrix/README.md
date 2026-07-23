# FS-03 synthetic media fixtures

`verify.sh` creates the FS-03 media matrix in a new temporary directory, probes it,
extracts video posters, checks an animated GIF and verifies that a truncated MP4 is
rejected. The temporary directory is deleted after the run. No generated media is
committed and the script never reads a media library.

The fixtures are original computer-generated test patterns produced from FFmpeg's
`lavfi` source:

| Fixture | Expected server-side evidence |
| --- | --- |
| `sample.jpg` | JPEG bitstream, 96 × 64 |
| `sample.png` | PNG bitstream, 96 × 64 |
| `sample.webp` | WebP bitstream converted from the synthetic PNG |
| `sample.gif` | GIF bitstream with multiple frames |
| `sample.mp4` | MP4 container with H.264/yuv420p video |
| `sample.mov` | MOV container with H.264/yuv420p video |
| `sample.mkv` | Matroska container with H.264/yuv420p video |
| `nonbrowser-ffv1.mkv` | Matroska/FFV1 candidate for the indexed-but-not-direct-play path; poster extraction is expected to succeed |
| `truncated.mp4` | 64-byte corrupt input that `ffprobe` must reject |

Run from the repository root:

```sh
bash tests/fixtures/media-matrix/verify.sh
```

The harness requires `ffmpeg`, `ffprobe`, `cwebp` and an FFmpeg build with `libx264`
and FFV1 encoders. It verifies the native video toolchain only. The FS-03 report
separately records libvips/govips and browser gaps; passing this script must not be
reported as proof of either.
