# INT-S4 browser automation evidence — darwin/arm64

- Date: 2026-09-02
- Source commit: `5c274251c29eb9900be857d1f95d38adcd70261e`
- Host: macOS 26.6.2 (`25G83`), arm64
- Node.js: 22.22.2
- Playwright: 1.61.1
- Engines installed by the locked Playwright package: Chromium, Firefox 151.0 and WebKit 26.5

## Executed checks

After installing the Firefox and WebKit binaries required by the repository lockfile, the following commands were run
serially so that the capacity measurement did not compete with the release E2E workload:

```text
make test-web-release-e2e
make test-browser-capacity
cd web && FOLIOPATH_BROWSER_CAPACITY_ENFORCE=1 \
  FOLIOPATH_BROWSER_CAPACITY_OUTPUT=../docs/evidence/int-001/int-s4-browser-capacity-darwin-arm64-2026-09-02.json \
  npm run test:capacity
```

The release suite passed. Firefox and WebKit each exercised the desktop keyboard/focus/degraded-state matrix, the
200%-equivalent reflow proxy, and bounded/reduced-motion storyboard behavior. The visual Chromium project retained its
Linux-only baseline guard and therefore did not compare screenshots on this Darwin host. Tests that belong only to mobile
Chromium or the separate real-storyboard vertical were also skipped by their declared project/environment guards. The
reported result was 6 passed and 13 skipped; the run did not reinterpret any skip as a pass.

The 100,000-item virtualized collection passed all enforced budgets in Chromium, Firefox and WebKit. The exact second-run
measurements are committed in
[`int-s4-browser-capacity-darwin-arm64-2026-09-02.json`](int-s4-browser-capacity-darwin-arm64-2026-09-02.json).
Its SHA-256 is `a4732915ff780e38d1f9a06d702e4b684450057036ee8d90ea236d54a26d4337`.
All engines mounted 60 items against the 64-item ceiling, remained above 59 FPS, kept frame P95 at or below 21 ms, and
remained below the 1.5 GiB process-tree RSS ceiling.

## Boundary

This is real local browser-engine automation and an inspectable large-collection measurement. Playwright WebKit is not a
claim about retail Safari, viewport emulation is not a physical touch device, and axe/semantic automation is not a
VoiceOver, NVDA or other screen-reader sign-off. Therefore this evidence advances `INT-408` but does not close it.
