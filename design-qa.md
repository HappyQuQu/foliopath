# UIF-401 Design QA

## Comparison target

- Canonical page list: `web/qa/visual-reference-manifest.json`.
- Latest prototype sources: the twelve mapped pages under
  `prototypes/apple-redesign`.
- Production implementation: the corresponding React routes rendered against
  a real administrator session, a temporary read-only synthetic media library,
  completed scans, catalog/search responses, previews, and original media.
- Combined artifact: `docs/evidence/uif-401/index.html`.
- Raw and combined evidence:
  `docs/evidence/uif-401/source`,
  `docs/evidence/uif-401/implementation`, and
  `docs/evidence/uif-401/comparison`.

## Normalization

- Every raw capture used the same in-app-browser `1280 × 720` CSS viewport,
  Simplified Chinese locale, and dark theme.
- A scrolling document reserves a 15px browser scrollbar gutter, so the raw PNG
  may be `1265 × 712`; non-scrolling captures are `1280 × 720`. This is a
  browser raster boundary, not a different CSS viewport.
- The combined artifact applies one equal review scale to both raw images. It
  does not crop either page; the original PNGs remain available for 1:1 review.
- Browse was normalized to the Kyoto directory with five child directories,
  four direct images, and an open preview. Search was normalized to the same
  “Kyoto” indexed-path result set.
- `auth-setup`, `library-new`, and `library-status` inherit an approved
  page-family prototype but are independent production routes by accepted
  requirement. Their review compares the inherited shell, hierarchy, spacing,
  components, and safety language rather than pretending a dialog and a
  navigable route are the same DOM.

## Findings and fixes

- No actionable P0, P1, or P2 difference remains inside the frozen MVP scope.
- The login comparison exposed a real geometry drift: production used a 420px
  card and 56px mark with left-aligned hierarchy while the prototype used a
  460px card, 64px mark, centered heading hierarchy, and tighter vertical
  padding. The central auth tokens and canonical Auth page styles now match the
  prototype values; login and first-administrator setup consume the same fix.
- Browse initially compared different folders and media counts. The production
  evidence was regenerated from a read-only synthetic Kyoto tree with the same
  folder/media/preview state, preventing content differences from being
  misreported as layout findings.
- General settings, the management shell, global Header, Browse, Search, and
  Viewer retain the accepted shared token, radius, typography, icon, and
  responsive patterns. No placeholder logo, emoji, CSS drawing, or handwritten
  SVG replaces a visible source asset.
- The latest Storage prototype also shows task center, missing-cache backfill,
  full rebuild, and maintenance concepts. Those controls are explicitly outside
  `FTR-UIF-001` and remain governed by the Post-MVP task-center/system-
  maintenance records. Production correctly exposes only the real scan,
  settings, cache-summary, and bounded cleanup contracts; no mock or fake
  success was added for visual similarity.

## Comparison history

1. UIF-301 established the central visual foundation, shared Header, management
   shell, and navigation ownership.
2. UIF-312 closed Browse current-directory filter placement and the top/middle/
   bottom blank-space defect.
3. UIF-315 closed Viewer desktop/mobile composition and interaction findings.
4. UIF-316 and UIF-317 closed shared async-state, locale, theme, breakpoint,
   contrast, long-name overflow, and duplicate-submit findings.
5. UIF-401 captured all twelve manifest pages in one browser viewport. The
   first pass found the auth geometry drift and a non-equivalent Browse fixture;
   both were corrected and the affected combined images were regenerated.

## Browser verification

- Comparison artifact:
  `http://127.0.0.1:4175/docs/evidence/uif-401/index.html`.
- Production routes: `http://127.0.0.1:4174`.
- Prototype routes: `http://127.0.0.1:4173`.
- Verified flows: first administrator setup, first-run empty library, three-step
  library creation, real scan completion/status, four management routes,
  directory traversal, preview, indexed path search, full Viewer, logout, and
  login.
- Original media remained on the temporary read-only mount. No production
  screenshot depends on a static replacement page or localStorage business
  success.

## Follow-up polish

- No P3 item is deferred from UIF-401.
- UIF-402 owns the Linux screenshot baselines for Header, four management
  pages, Browse top/bottom, Search, preview, and Viewer. UIF-401 does not
  pre-accept those future baseline files.

final result: passed
