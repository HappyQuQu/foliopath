# UIF-315 Design QA

## Comparison target

- Source visual truth: `prototypes/apple-redesign/05-viewer.html`
- Source desktop capture:
  `docs/evidence/uif-315/source-viewer-desktop-final.png`
- Source mobile capture: `docs/evidence/uif-315/source-viewer-mobile.png`
- Final desktop implementation:
  `docs/evidence/uif-315/implementation-viewer-desktop-v3.png`
- Final mobile implementation:
  `docs/evidence/uif-315/implementation-viewer-mobile-final.png`
- Desktop combined comparison:
  `docs/evidence/uif-315/comparison-viewer-desktop-final.png`
- Mobile combined comparison:
  `docs/evidence/uif-315/comparison-viewer-mobile-final.png`
- Focused information-panel evidence:
  `docs/evidence/uif-315/implementation-viewer-info-open-desktop-v3.png` and
  `docs/evidence/uif-315/implementation-viewer-info-open-mobile-final.png`

## Normalization

- Desktop source and implementation used a `1440 × 900` CSS viewport,
  `devicePixelRatio: 1`, and produced `1440 × 900` captures.
- Mobile source and implementation used a `390 × 844` CSS viewport,
  `devicePixelRatio: 1`, and produced `390 × 844` captures.
- Combined artifacts place equal-size captures side by side without rescaling.
- State: image viewer, fitted media, information panel initially closed. The
  focused-region captures additionally show the settled information-open state.

## Findings

- No actionable P0, P1, or P2 differences remain.
- Fonts and typography: filename size, weight, truncation, toolbar density, and
  information hierarchy match the prototype at both tested widths.
- Spacing and layout rhythm: desktop media occupies the prototype's centered
  `75vw × 75vh` frame; the toolbar, navigation positions, and 340px desktop
  information panel align. Mobile keeps the toolbar on one row and uses a
  bottom information panel without horizontal overflow.
- Colors and tokens: the near-black viewer chrome and blue-violet raster asset
  match the source direction. The implementation uses the canonical viewer
  tokens rather than feature-local theme values.
- Image quality and asset fidelity: the Storybook state uses the committed
  `viewer-blue-violet.png` raster. The mobile implementation intentionally
  preserves the media's intrinsic aspect ratio instead of stretching the
  prototype placeholder to `75vw × 75vh`; this is required for a real media
  viewer and is not an actionable fidelity defect.
- Copy and content: filename, Chinese control names, indexed detail labels,
  information summary, and close affordances are complete. The implementation
  retains the approved zoom controls that the static visual does not exercise.
- Accessibility and interaction: DOM order remains close → filename → actions;
  the viewer root receives route focus, close is the first Tab stop, pressed
  state is exposed, information has a labelled complementary landmark, and
  desktop/mobile close controls remain visible.

## Comparison history

1. Initial desktop comparison
   (`comparison-viewer-desktop-v1.png`) found P2 drift: a textured, over-saturated
   placeholder asset; a visible initial focus ring; centered filename; persistent
   pressed styling on fit; and overly visible circular navigation backgrounds.
   The asset was regenerated as a smooth raster, route focus moved to the viewer
   root, filename alignment and local pressed styling were corrected, and
   navigation backgrounds were quieted.
2. The first mobile implementation wrapped the action group onto a second row
   (P2) and reduced useful media space. The 390px layout now uses compact
   single-row controls while preserving every viewer action.
3. The final same-size desktop and mobile combined comparisons were reviewed
   after those fixes. No actionable P0/P1/P2 issue remained.

## Browser verification

- Primary interactions tested: open/close indexed information, desktop
  slide-in panel, mobile bottom panel, responsive toolbar, and initial route
  focus. Component tests cover fit, 1:1, zoom, wheel zoom, keyboard navigation,
  Escape, `i`, fullscreen API, video controls, unavailable media, and retry
  chrome.
- Console errors checked on the rendered Storybook implementation: none.
- The non-modal preview remains the single shared implementation and controller
  previously captured in `docs/evidence/uif-312`; UIF-315 did not fork or copy
  that controller.

## Follow-up polish

- P3: real media naturally differs in aspect ratio from the prototype's
  rectangular placeholder; the viewer correctly uses `object-fit: contain`.
- P3: zoom-in/zoom-out controls add two icons beyond the static prototype but
  are retained because the accepted viewer contract requires explicit zoom.

final result: passed
