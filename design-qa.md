# Library Picker Design QA

## Comparison Target

- Source visual truth: `/Users/qu/.codex/generated_images/019fab64-b203-7381-b3ba-b3c5430da350/call_lQ3V0422WWSzlvcanqGTkSkw.png`
- Implementation route: `http://127.0.0.1:18086/libraries/lib_1/browse/dir_5`
- Final implementation screenshot: `/Users/qu/Documents/GitHub/foliopath/.artifacts/library-picker-option-1/implementation-full-open-pass-2.png`
- Dark-mode evidence: `/Users/qu/Documents/GitHub/foliopath/.artifacts/library-picker-option-1/implementation-dark-open.png`
- Final normalized comparison: `/Users/qu/Documents/GitHub/foliopath/.artifacts/library-picker-option-1/comparison-pass-2.png`
- Browser CSS viewport: `1280 × 1024`
- Source pixels: `998 × 1575`
- Implementation pixels: `1265 × 1012`
- Density normalization: the source sidebar was cropped to its left 690px and scaled to 350px wide; the implementation sidebar was cropped to 350px wide. Both were padded to 350 × 1024 and placed side by side in a 700 × 1024 comparison.
- State: Simplified Chinese, light theme, desktop sidebar, library picker open, `桌面测试库` selected.

## Full-view Comparison Evidence

The final implementation preserves the existing FolioPath desktop shell while matching the selected direction's hierarchy: a labeled two-line trigger, current library name, availability/count metadata, anchored open menu, pale-blue selected row, and blue check. The directory tree and bottom navigation retain their existing product positions and visual language.

The source mock contains three illustrative libraries and a `切换媒体库` action. The implementation intentionally renders the one library returned by the real local API and does not invent a management action that is absent from the approved behavior.

## Focused Region Comparison Evidence

Focused comparison was required because the picker typography, border treatment, selected row, menu spacing, and its relationship to the directory tree were too small to judge from the full application screenshot. The normalized side-by-side image confirms:

- Fonts and typography: both use the product's Apple-system/PingFang stack, medium-weight library names, and smaller muted status metadata without wrapping.
- Spacing and layout rhythm: trigger and option padding, radii, and vertical gaps follow the selected target; the menu now pushes the directory section below it instead of exposing content behind the menu.
- Colors and visual tokens: the implementation uses the existing surface, border, accent-soft, accent, and muted-text tokens; selected state remains legible in both light and dark themes.
- Image quality and asset fidelity: no new raster imagery is required. The existing FolioPath brand asset and Phosphor chevron/check icons are retained; no placeholder, CSS-drawn, emoji, or handcrafted SVG asset was introduced.
- Copy and content: current library name, `可用`, and `2498 项` match real API data. The mock-only sample libraries are correctly omitted.

## Findings

No actionable P0, P1, or P2 findings remain.

- [P3] Open-state chevron differs from the static mock
  - Location: library picker trigger.
  - Evidence: the source mock leaves the chevron pointing down while open; the implementation rotates it upward.
  - Impact: minor visual difference only.
  - Disposition: retained because it communicates the expanded state more clearly and remains consistent with the directory disclosure controls.

## Comparison History

### Pass 1

- Evidence: `/Users/qu/Documents/GitHub/foliopath/.artifacts/library-picker-option-1/comparison-pass-1.png`
- [P2] The open trigger used a strong blue outline and halo that was visually heavier than the neutral selected target.
- [P2] The absolutely positioned one-row menu allowed the underlying `目录` label to peek out beside it.
- Fixes:
  - Returned the expanded trigger to the neutral strong-border token and subtle elevation.
  - Changed the menu to participate in sidebar layout so the directory section begins cleanly below it.
  - Kept selected library text neutral while reserving blue for the soft background and check icon.

### Pass 2

- Evidence: `/Users/qu/Documents/GitHub/foliopath/.artifacts/library-picker-option-1/comparison-pass-2.png`
- Post-fix result: trigger emphasis, selected row, spacing, and menu/tree separation match the selected direction with no remaining P0/P1/P2 issue.

## Interaction and Runtime Verification

- Opened and closed the picker.
- Confirmed `aria-expanded`, `listbox`, `option`, and `aria-selected` semantics.
- Confirmed Escape closes the menu and restores focus to the trigger.
- Confirmed the selected library uses both a check icon and selected semantics, not color alone.
- Confirmed the open state in light and dark themes.
- Checked browser console warnings and errors: none.

## Implementation Checklist

- [x] Selected visual direction implemented.
- [x] Real library data retained.
- [x] Keyboard and focus behavior verified.
- [x] Light and dark themes verified.
- [x] Responsive sidebar behavior preserved.
- [x] P0/P1/P2 visual differences resolved.

final result: passed
