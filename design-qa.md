# UIF-402 Design QA

## Comparison target

- Prototype/source authority:
  `web/qa/visual-reference-manifest.json` and the twelve latest pages under
  `prototypes/apple-redesign`.
- Same-state source/production comparison:
  `docs/evidence/uif-401/index.html`.
- Regression implementation:
  `web/tests/e2e/visual-regression.spec.ts`.
- Linux baseline inventory and reproduction record:
  `docs/evidence/uif-402/README.md`.

UIF-401 remains the visual source comparison. UIF-402 does not reinterpret the
prototype or accept new functionality; it turns the approved current MVP
regions into stable regression assets.

## Captured states

- One canonical global Header.
- Management Center: General, Libraries, Scanning and cache, Account.
- Browse: top, document bottom, and open right-side preview.
- Search: populated current-library result state.
- Viewer: available image and preserved offline-source state.

Every new capture uses a fixed Linux Chromium environment at `1280 × 800`,
English, dark theme, and reduced motion. The authenticated account, library,
directory tree, settings, cache, media metadata, ETags, dates, counts, and
synthetic media bytes are deterministic.

## Combined visual review

The UIF-401 source captures and UIF-402 baselines were reviewed together at one
equal scale for General, Libraries, Storage, Account, Browse, Search, and
Viewer. The expected language and viewport-height differences do not change the
shared composition:

- the same global Header remains the only top-level Header;
- management navigation remains one non-duplicated left rail with four
  independent routes;
- Browse keeps the filter on the right, the directory hierarchy on the left,
  a single vertical document scroll, and the preview in the right workspace;
- Search remains a dedicated result page entered from the global search;
- Viewer keeps the approved full-screen toolbar, media stage, close,
  previous/next, fit/zoom/1:1, fullscreen, and information behavior.

No visible P0, P1, or P2 regression was found. No placeholder logo, emoji, CSS
drawing, handwritten SVG, or mock-only production control was introduced.

## Stability controls

- All API responses are intercepted at the real generated-client HTTP boundary;
  feature code is not altered to consume test-only state.
- The committed `tests/fixtures/media/viewer-blue-violet.png` is used for
  thumbnails and original-image responses.
- Image completion, decode, and paint frames are awaited before capture.
- The preview selection/focus transition is allowed to settle before capture.
- No screenshot mask is used.
- Baselines were generated in Linux, then the full visual file was run again
  without `--update-snapshots`: `9 passed`.

## Scope guard

The latest prototype Storage page still contains Post-MVP task-center,
missing-cache backfill, full rebuild, and maintenance concepts. UIF-402 does
not add those unsupported controls to production. Their contracts and delivery
remain governed by the separate Post-MVP feature records, so visual fidelity
does not create false backend capability.

## Follow-up

- UIF-403 owns the complete real backend vertical chain.
- UIF-404 owns the release browser/accessibility matrix.
- UIF-405 owns candidate 100k/10k capacity evidence.
- UIF-406～408 own full repository verification, documentation convergence, and
  Integrated Slice sign-off.

final result: passed
