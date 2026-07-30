# UIF-317 Design QA

## Comparison target

- Latest responsive prototype sources:
  `prototypes/apple-redesign/qa/adaptive-width-mobile.png`,
  `prototypes/apple-redesign/qa/adaptive-width-tablet.png`,
  `prototypes/apple-redesign/qa/adaptive-width-1265.png`, and
  `prototypes/apple-redesign/qa/adaptive-width-wide.png`.
- Accepted shared-state visual source:
  `docs/evidence/uif-316/implementation-state-matrix-1440.png` and
  `docs/evidence/uif-316/implementation-state-matrix-390.png`.
- Final implementation evidence:
  `docs/evidence/uif-317/implementation-zh-CN-light-1440x1000.png`,
  `docs/evidence/uif-317/state-matrix-zh-CN-light-390.png`, and
  `docs/evidence/uif-317/implementation-zh-CN-dark-final-1265x737.png`.
- Combined same-state evidence:
  `docs/evidence/uif-317/comparison-zh-CN-light-1440x1000.png` and
  `docs/evidence/uif-317/comparison-zh-CN-light-390.png`.

The prototype remains the visual source of truth. Production locale messages,
theme tokens, component semantics and backend contracts remain the behavioral
source of truth; the workbench does not introduce its own feature state.

## Normalization

- Desktop source and implementation are both `1440 × 1000` pixels at a
  `1440 × 1000` CSS viewport and density 1. Their combined comparison is
  `2880 × 1000`; neither side is scaled.
- The in-app browser reserved a 15px scrollbar gutter for the requested
  `390 × 844` mobile override. The accepted UIF-316 source is `375 × 1372` and
  the localized UIF-317 implementation is `375 × 1411`, both density 1. The
  `752 × 1412` comparison pads the shorter side and adds a one-pixel divider;
  it does not scale either surface.
- The post-fix dark capture is `1265 × 737` at density 1. It verifies the final
  central dark-accent token after the automated contrast finding.
- Production Playwright separately uses exact `390 × 844`, `768 × 1024`,
  `1265 × 800`, and `1440 × 900` CSS viewports.
- Compared state: all eight shared loading, empty, offline, error, conflict,
  cancel, pending and success states; Simplified Chinese and English; light and
  dark themes.

## Findings

- No actionable P0, P1 or P2 difference remains.
- Fonts and typography: the production locale provider supplies all visible
  copy. English and Chinese headings, body copy, buttons and status text retain
  the approved hierarchy, line height and optical weight; long English copy
  wraps without clipping at the exact 390px production viewport.
- Spacing and layout rhythm: the 1440/1265 boards retain the approved two-column
  rhythm; 768/390 collapse without horizontal overflow. The real Browse
  toolbar now wraps when its available pane is narrower than the viewport
  because of the fixed directory sidebar or unusually long library names.
- Colors and visual tokens: light and dark states use the central theme source.
  The final dark accent is `#2997ff`, which resolves the measured active-menu
  contrast failure while preserving the prototype's restrained blue emphasis.
  Semantic warning, danger, information and success colors remain icon-backed.
- Image quality and asset fidelity: the matrix uses the approved Phosphor icon
  package and central FolioPath primitives. No logo, product image,
  illustration or non-standard icon from the source was replaced with a
  placeholder, CSS drawing or handwritten SVG.
- Copy and content: Storybook no longer contains a parallel hard-coded Chinese
  state vocabulary. Both locales consume the production keys, including
  reliable-index preservation and non-destructive recovery language.
- Accessibility and interaction: the exact matrix verifies document language,
  resolved theme, no horizontal overflow, serious/critical axe results and the
  `1ms` reduced-motion token. Keyboard focus enters the Viewer root and returns
  to the invoker; touch controls, Escape, dialog focus return, forced colors
  and duplicate-submit protection are covered by browser evidence.

## Focused comparison

- The combined desktop comparison keeps every state readable at 1:1 density,
  so typography, padding, border, radius, icon and semantic-color details can
  be judged without a separate crop.
- The combined mobile comparison focuses on the responsive stack and long-copy
  wrapping. The final dark capture separately focuses the changed accent,
  pending control and semantic notices after the contrast fix.

## Comparison history

1. The initial full desktop/mobile comparison found no layout drift in the
   state workbench, but the production matrix exposed a serious contrast
   failure in the light global-search submit label (`4.37:1`). The submit label
   was moved to the central primary text token; the Chromium axe rerun passed.
2. The next production dark-theme pass exposed the active management-menu
   accent at `4.36:1` against its resolved background. The central dark accent
   changed from `#0a84ff` to `#2997ff`; the post-fix dark capture above and the
   complete Chromium axe matrix passed.
3. The 1024px real Browse path with a 128-character library name exposed an
   `88px` page overflow. The canonical toolbar now wraps at its available
   width; the same E2E path reports document scroll width equal to client width.
4. The same real path found that a double click on “Save scan and cache
   settings” could issue two PATCH requests. A synchronous in-flight guard was
   added to the owning settings page; the rerun observed exactly one request.

## Browser verification

- Rendered Storybook route:
  `http://127.0.0.1:6006/iframe.html?id=ui-statematrix--complete&viewMode=story`
- Primary interactions: locale/theme selection, four responsive widths,
  keyboard search submit, Escape and focus return, Viewer information toggle,
  touch recovery controls, settings save, duplicate submit, and reduced motion.
- Browser engines: Chromium and Pixel 5 emulation; Firefox; WebKit; Chrome
  Stable; Chrome forced-colors.
- Console errors and warnings checked on the settled state-matrix capture:
  none.
- Automated results: Chromium/mobile suite `6 passed, 3 skipped`; release suite
  `4 passed, 5 skipped`; Chrome Stable/forced-colors suite
  `4 passed, 2 skipped`.

## Follow-up polish

- No P3 item is required for UIF-317. UIF-318 may consume this matrix as the
  final Consumer/UI Ready gate evidence.

final result: passed
