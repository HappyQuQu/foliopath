# UIF-317 locale/theme/responsive/input matrix

## Visual truth and scope

- Latest prototype references:
  - `prototypes/apple-redesign/qa/adaptive-width-mobile.png`
  - `prototypes/apple-redesign/qa/adaptive-width-tablet.png`
  - `prototypes/apple-redesign/qa/adaptive-width-1265.png`
  - `prototypes/apple-redesign/qa/adaptive-width-wide.png`
- Shared-state visual baseline:
  `docs/evidence/uif-316/implementation-state-matrix-1440.png` and
  `docs/evidence/uif-316/implementation-state-matrix-390.png`.
- Rendered target:
  `UI/StateMatrix/Complete` in Storybook. It consumes the real locale provider,
  central theme tokens and canonical state primitives; it does not own business
  state.

## Captured matrix

Every cell rendered all eight UIF-316 states with two alerts and four
non-blocking status regions. The reported document width equalled its scroll
width in every capture.

| Locale | Theme | 390 × 844 | 768 × 1024 | 1265 × 800 | 1440 × 900 |
| --- | --- | --- | --- | --- | --- |
| `zh-CN` | light | `state-matrix-zh-CN-light-390.png` | `state-matrix-zh-CN-light-768.png` | `state-matrix-zh-CN-light-1265.png` | `state-matrix-zh-CN-light-1440.png` |
| `zh-CN` | dark | `state-matrix-zh-CN-dark-390.png` | `state-matrix-zh-CN-dark-768.png` | `state-matrix-zh-CN-dark-1265.png` | `state-matrix-zh-CN-dark-1440.png` |
| `en` | light | `state-matrix-en-light-390.png` | `state-matrix-en-light-768.png` | `state-matrix-en-light-1265.png` | `state-matrix-en-light-1440.png` |
| `en` | dark | `state-matrix-en-dark-390.png` | `state-matrix-en-dark-768.png` | `state-matrix-en-dark-1265.png` | `state-matrix-en-dark-1440.png` |

The in-app browser reserves a 15px scrollbar gutter for the requested 390px
override, so its screenshots expose a 375px content viewport. The production
Playwright matrix uses the exact `390 × 844` viewport and separately asserts no
page overflow.

## Same-state comparisons

- `comparison-zh-CN-light-1440x1000.png`: UIF-316 source on the left, localized
  UIF-317 implementation on the right, both `1440 × 1000`.
- `comparison-zh-CN-light-390.png`: UIF-316 source on the left and the settled
  mobile implementation on the right. The one-pixel pad normalizes the two
  full-page heights without scaling either state surface.

The new localized copy intentionally replaces workbench-only prose with the
authoritative production messages. Geometry, typography, icons, radii and
semantic color treatments remain owned by the same shared components.

`implementation-zh-CN-dark-final-1265x737.png` is the post-axe-fix browser
capture. It records the final central dark accent after the active management
navigation contrast was raised above the WCAG AA threshold.

## Nonvisual matrix

- `web/tests/e2e/auth.spec.ts` runs the real general-settings screen through
  `zh-CN/en × light/dark × 390/768/1265/1440`, checks `lang`, resolved theme,
  horizontal overflow and serious/critical axe results, and verifies the
  reduced-motion token resolves to `1ms`.
- The same vertical E2E exercises keyboard submit, dialog Escape/focus return,
  preview/viewer focus return, layout toggles, language/theme controls and
  duplicate-submission prevention.
- `web/tests/e2e/media-matrix.spec.ts` retains the real Pixel 5 touch path,
  desktop keyboard viewer path, Firefox/WebKit/Chromium degraded states,
  forced colors and reduced-motion storyboard behavior.
- Shared component tests retain focus, pressed-state, motion and semantic
  announcement coverage. UIF-317 adds no second controller or feature-local
  input policy.
- The real long-library-name Browse path also asserts that the canonical tools
  row wraps inside its available pane at 1024px, and the storage settings path
  verifies that a double click produces only one PATCH request.
