# INT-S4 retail Safari evidence — darwin/arm64

- Date: 2026-09-02
- Source commit: `9f49c6dd5f07bdb58a86ddb4f663f04a996772a0`
- Host: macOS 26.6.2 (`25G83`), arm64
- Browser: retail Safari 26.6.2
- Candidate image digest: `sha256:dc24df950b1df6100edd9682e7c3eb6d5af80fb0ccb73f8ff60c93903bb658cc`

## Setup and boundary

The app ran from a local Linux/arm64 candidate container behind the repository's trusted-proxy topology. `/library` was a
read-only bind mount containing one temporary copy of the repository-owned `viewer-blue-violet.png` fixture. No developer
or user media was read. The copy and source both ended with SHA-256
`d7205545082793577988361a016cec72bd597af1049b218f0300319c0a0d391b`.

The test created an ephemeral administrator and an `/library/album` library, waited for the first scan, and then exercised
the resulting one-item collection through the actual Safari application. Safari password storage was explicitly declined.
The page was returned to `about:blank`; containers and the candidate image were removed, and the temporary directory was
moved to Trash so cleanup remained recoverable.

## Observed retail Safari behavior

- Library creation, first scan and browse reached the indexed `photo.png` item.
- With the macOS full-keyboard-access alternative (`Option+Tab`), focus followed DOM order from global controls through
  browse filters and sorting to the media preview control.
- `Return` opened the preview. Continued keyboard navigation reached “进入完整查看器”; `Return` opened the full viewer.
- The viewer main container received focus. `I` opened and closed the basic-information panel.
- `Escape` returned to browse and restored focus to “预览 photo.png”.
- Five Safari “Zoom In” steps moved the page from 100% to the browser's 200% level. The actual-size command became enabled,
  the library/sidebar/tool/media controls remained present, and the accessibility tree exposed only a vertical page
  scrollbar, not a horizontal page scrollbar. A direct visual check showed the item and controls remained reachable.
- The test reset Safari to actual size before leaving the page.

## Boundary

This closes the retail-Safari and physical-keyboard portions of `INT-408` on this Mac. It is not a physical touch-device
run and it did not enable or operate VoiceOver, NVDA or another screen reader. Those two owner-controlled checks still
block `INT-408` completion.
