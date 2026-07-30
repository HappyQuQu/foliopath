# UIF-316 Design QA

## Comparison target

- Latest prototype behavior and operational-state source:
  `prototypes/apple-redesign/11-task-detail.html`
- Prototype capture:
  `prototypes/apple-redesign/qa/operations-task-detail-final.png`
- Existing canonical component visual sources:
  `docs/evidence/uif-316/source-async-offline.png` and
  `docs/evidence/uif-316/source-inline-warning.png`
- Desktop implementation:
  `docs/evidence/uif-316/implementation-state-matrix-1440.png`
- Mobile implementation:
  `docs/evidence/uif-316/implementation-state-matrix-390.png`
- Combined comparison evidence:
  `docs/evidence/uif-316/comparison-state-matrix-1440.png` and
  `docs/evidence/uif-316/comparison-state-matrix-390.png`

The prototype defines the restrained operational hierarchy and the product
documents define failure/recovery behavior. The component workbench sources are
the exact visual target for shared offline and persistent-status treatments;
the matrix intentionally documents components rather than cloning the complete
task-detail page.

## Normalization

- Desktop implementation used a `1440 × 1000` CSS viewport and
  `devicePixelRatio: 1`, producing a `1440 × 1000` capture.
- The mobile browser override requested `390 × 844`; the in-app browser exposed
  a `375px` content viewport after its scrollbar reservation. The full-page
  capture is `375 × 1372`, and measured body width equals viewport width, so no
  horizontal overflow is hidden.
- Both canonical source captures are `720 × 520` at density 1. The desktop
  comparison stacks those two sources and scales only the full source and
  implementation boards to a common height. The mobile comparison scales and
  pads the same source board beside the unscaled mobile implementation.
- State: Simplified Chinese, light theme, eight-state workbench matrix. The
  matrix has no server state of its own; every production consumer continues
  to use its real query, mutation, scan, cache, thumbnail, or viewer state.

## Findings

- No actionable P0, P1, or P2 differences remain.
- Fonts and typography: state headings, strong titles, supporting copy and
  control labels use the existing FolioPath hierarchy and retain readable
  wrapping at mobile width.
- Spacing and layout rhythm: desktop uses a balanced two-column workbench;
  mobile collapses to one column. Empty/offline/error regions retain their
  canonical height and internal rhythm; compact conflict/cancel/success notices
  do not inflate into page-level states.
- Colors and visual tokens: offline/warning, danger/conflict, accent/pending
  and success all use central semantic tokens. Icons accompany every semantic
  color, so meaning does not depend on color alone.
- Image quality and asset fidelity: these state surfaces use the approved
  Phosphor icon library and do not require raster product imagery. No prototype
  logo, illustration, or media asset was replaced or approximated.
- Copy and content: every named UIF-316 state is explicit. Offline and cancel
  copy says the reliable index is preserved; conflict says to refresh instead
  of overwriting; error says the current interface and originals are unchanged;
  pending blocks duplicate submission; success confirms the reliable result.
- Accessibility and interaction: blocking error and conflict use
  `role="alert"`; loading, offline, cancel and persistent success use
  `role="status"`. Pending is a disabled loading button. Recovery controls
  remain semantic buttons, and the mobile board has no horizontal overflow.

## Focused comparison

- The source offline component and the matrix offline cell use the same border,
  warning surface, icon, title hierarchy and button treatment; only the
  approved recovery copy differs by context.
- The source warning notice establishes the compact persistent-status geometry.
  Conflict, cancel and success preserve that geometry while switching only the
  canonical semantic token and icon.
- A separate crop was unnecessary because the combined artifacts render these
  controls at readable size and the DOM/semantic checks cover the nonvisual
  distinctions.

## Comparison history

1. The first combined desktop and mobile comparison found no P0/P1/P2 visual
   mismatch. No visual fix was made after that pass.
2. The mobile capture was repeated after the initial reload observation
   occurred before Storybook had rendered. The settled capture contains all
   eight states, reports two alerts and four status regions, and has no
   horizontal overflow. This was an evidence-capture correction, not a design
   change.

## Browser verification

- Rendered Storybook route:
  `http://127.0.0.1:6006/iframe.html?id=ui-statematrix--complete&viewMode=story`
- Primary checks: all eight named sections rendered; alert/status semantics;
  pending control disabled; responsive one-column collapse; full-page mobile
  scroll; no page-level horizontal overflow.
- Console errors and warnings checked on the settled mobile implementation:
  none.
- Component tests cover loading/offline announcements, blocking error retry,
  empty recovery action, danger/success semantics and dismiss behavior.

## Follow-up polish

- The exact `390px` content-width capture, dark theme and English locale belong
  to the already-planned UIF-317 four-width/theme/locale matrix.

final result: passed
