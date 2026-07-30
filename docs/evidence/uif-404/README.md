# UIF-404 browser and accessibility applicability evidence

## Scope

UIF-404 revalidates the automated browser, input and accessibility parts of
`UIF-AC-010` against the production React application. It does not replace the
physical-device and assistive-technology release sign-off owned by
`S5-006B`.

The run used:

- macOS 26.6 on arm64 with Node 22.22.2 and Playwright 1.61.1;
- Playwright Chromium 149.0.7827.55, Firefox 151.0 and WebKit 26.5;
- installed Google Chrome 151.0.7922.71 for the branded stable and
  forced-colors projects;
- a fresh FolioPath container, database and read-only synthetic `/library` for
  the Chromium product chain;
- deterministic API fixtures for the cross-engine degraded-state, input and
  accessibility matrix.

## Automated matrix

| Concern | Evidence |
| --- | --- |
| Chromium production chain | Setup, authentication, account maintenance, library/scan, Browse, Search, preview/Viewer, settings/cache cleanup and logout/login passed against the real backend. |
| Firefox and WebKit | Viewer focus, keyboard information toggle, Escape, unsupported/offline/deleted states, overflow, axe and storyboard hover lifecycle passed in both engines. |
| Keyboard and focus | Initial Viewer focus, `i`/`I`, Escape, Chromium sequence navigation and visible recovery controls passed. |
| Touch | Pixel 5 emulation used real pointer-touch events via Playwright `tap()` for Viewer information and offline recovery controls; storyboard hover remained disabled. |
| axe | The real Chromium chain and the cross-engine offline/touch states had no serious or critical axe violations. |
| forced colors | Installed Chrome ran the desktop matrix with `forcedColors: "active"` and passed. |
| reduced motion | Storyboard hover made zero requests and started no animation after `prefers-reduced-motion: reduce`; the existing Linux visual project also fixes reduced motion. |
| Layout safety | Desktop and emulated mobile checks found no horizontal page overflow in the covered Viewer and recovery states. |

One cross-engine test defect was found during the first WebKit run. The test
required the storyboard request count to remain zero 200 ms after hover and
then immediately required it to become greater than zero. WebKit completed the
intent timer within that fixed delay. The contradictory timing assertion was
removed; the retained contract proves no eager request before hover, eventual
loading after desktop hover, a bounded request count, cleanup on pointer exit,
no touch hover, and zero requests under reduced motion.

## Commands and results

Executed from the repository root:

```sh
FOLIOPATH_E2E_SUITE=chromium tests/e2e/web_auth.sh
FOLIOPATH_E2E_SUITE=release tests/e2e/web_auth.sh
FOLIOPATH_E2E_SUITE=chrome-stable tests/e2e/web_auth.sh
```

Final results:

- Chromium/mobile Chromium: `6 passed`, `3 applicability skips`;
- Firefox/WebKit/macOS visual project: `4 passed`,
  `13 applicability/platform skips`;
- Chrome Stable/forced colors: `4 passed`, `2 applicability skips`.

The exact-pixel project is Linux-owned and therefore skipped on macOS. Its
committed 11-image baseline was already generated and compared without update
flags as `9 passed` by
[`UIF-402`](../uif-402/README.md). A fresh local Linux-container rerun was
attempted, but Docker's local Playwright image cache returned a blob I/O error
after host disk pressure; no new Linux visual result is claimed by UIF-404.
This does not weaken the input/accessibility assertions above and does not
replace the existing UIF-402 Linux evidence.

## Physical applicability boundary

Automation is useful evidence, but it cannot certify the complete physical
accessibility experience:

| Item | UIF-404 disposition |
| --- | --- |
| Safari on a physical Mac | Prior Safari 26.5.2 product-chain evidence remains recorded by `S5-006A`; not rerun here. |
| Real Firefox installation | Still requires the final promised version and operating-system combination. |
| VoiceOver/NVDA or another screen reader | Requires manual reading order, naming, announcement and focus-transition review. axe is not a screen reader. |
| 200%/400% browser zoom | Requires physical-browser reflow, clipping and focus-visibility review. Viewport emulation is not treated as zoom evidence. |
| OS high-contrast/forced-colors | Automated forced-colors passed; representative physical OS settings still require manual review. |
| Physical touch/mobile device | Emulated touch passed; gesture targeting, virtual keyboard, safe areas and media decode still require representative hardware. |

Those items remain blockers under
[`S5-006B`](../../gates/MVP-2026-07-23/s5-browser-quality-candidate.md).
UIF-404 therefore closes the feature's automated applicability revalidation,
not `S5-006`, `S5-009` or Release Candidate approval.
