# Apple Redesign Shell Restoration — Design QA

## Comparison Target

- Source visual truth: `/Users/qu/Documents/GitHub/foliopath/prototypes/apple-redesign/qa/browse-after-desktop.png`
- Implementation route: `http://127.0.0.1:18086/libraries/lib_1/browse/dir_5`
- Before screenshot: `/Users/qu/Documents/GitHub/foliopath/web/qa/redesign-regression-2026-07-29/04-before-full-redesign.png`
- Final implementation screenshot: `/Users/qu/Documents/GitHub/foliopath/web/qa/redesign-regression-2026-07-29/12-final-implementation.png`
- Final combined comparison: `/Users/qu/Documents/GitHub/foliopath/web/qa/redesign-regression-2026-07-29/13-final-comparison.png`
- Browser CSS viewport: `1265 × 712`
- Source pixels: `1265 × 712`
- Implementation pixels: `1265 × 712`
- Density normalization: none; source and implementation were captured at the same pixel dimensions.
- State: Simplified Chinese, light theme, desktop browse route, one media item selected with the non-modal preview open.

## Full-view Comparison Evidence

The final combined comparison restores the source design's major composition: 268px fixed sidebar, plain
FolioPath wordmark, compact library picker with status below it, directory tree, bottom search/settings
navigation, breadcrumb/search top bar, two-row sticky browse chrome, and a right-side non-modal preview.

Dynamic content differs by design. The source uses a synthetic `家庭影像/京都` hierarchy with child-directory
cards and a placeholder preview; the implementation shows the real `lib/4K风景` index, real thumbnails, and
the selected original. These differences do not change the shell hierarchy or component styling being judged.

## Focused Comparison Evidence

The full-view combined image is sufficient for the shell, navigation, toolbar, grid, and preview proportions
because both captures share the same 1265 × 712 viewport and all important controls are readable.

- Fonts and typography: both use the product Apple-system/PingFang stack, compact UI sizes, medium navigation
  labels, and the same heading hierarchy. Real long filenames correctly truncate.
- Spacing and layout rhythm: fixed sidebar width, 52px top bars, compact picker, toolbar spacing, grid gutters,
  preview split, borders, radii, and muted surfaces follow the source.
- Colors and visual tokens: canvas, surface, border, muted text, accent-soft selection, and blue focus/selection
  all reuse the central redesign tokens.
- Image quality and asset fidelity: the implementation uses real indexed thumbnails and the authenticated
  original preview. No placeholder, CSS drawing, emoji, or handcrafted SVG replaced a source asset.
- Copy and content: shell labels match the Chinese redesign. Library, directory, counts, filenames, dates, and
  preview details correctly come from the real API.
- Icons: Phosphor icons retain the source's weight and size. The manual refresh is intentionally a compact icon
  at the end of the toolbar.

## Findings

No actionable P0, P1, or P2 findings remain.

- [P3] Product-required account control remains in the top bar.
  - Evidence: the static source shows theme and filter controls only; production also exposes the authenticated
    account/logout menu.
  - Disposition: accepted because logout is a stable-release requirement and the control is visually compact.
- [P3] Refresh icon is an intentional addition.
  - Evidence: the static source predates WCH-S3 manual refresh.
  - Disposition: accepted as a compact toolbar icon that does not alter the redesign hierarchy.

## Comparison History

### Pass 1 — blocked

- Evidence: `/Users/qu/Documents/GitHub/foliopath/web/qa/redesign-regression-2026-07-29/05-before-comparison.png`
- [P1] Resizable oversized sidebar and branded mark changed the primary frame.
- [P1] Search/settings navigation was removed from the sidebar and moved into unrelated top-right controls.
- [P1] Browse toolbar grouped recursive/layout/type controls into a denser, non-source structure.
- [P2] Library picker combined status into an oversized card instead of the compact source hierarchy.

### Pass 2 — blocked

- Evidence: `/Users/qu/Documents/GitHub/foliopath/web/qa/redesign-regression-2026-07-29/08-after-comparison-pass1.png`
- Fixed the frame, wordmark, picker hierarchy, bottom navigation, top bar, and toolbar.
- Remaining [P2]: the active `返回浏览` self-link appeared in the browse sidebar although the source only shows
  search and settings there.

### Pass 3 — passed

- Evidence: `/Users/qu/Documents/GitHub/foliopath/web/qa/redesign-regression-2026-07-29/13-final-comparison.png`
- Removed the active browse self-link and aligned the preview width to the source state.
- No actionable P0/P1/P2 visual mismatch remains after accounting for real-data and required-auth differences.

## Interaction and Runtime Verification

- Opened the production browse route against the preserved real database and desktop read-only media mount.
- Opened a media preview and verified the docked layout.
- Verified media type selection through the top-bar filter in component tests.
- Verified manual refresh, directory navigation refresh, account logout semantics, mobile navigation focus, and
  URL-bound browse state in component tests.
- Production image rebuilt and the running application reports healthy.

## Implementation Checklist

- [x] Fixed redesign sidebar and bottom navigation restored.
- [x] Breadcrumb/search top bar and compact browse toolbar restored.
- [x] Media type filter retained in the source-style filter control.
- [x] Manual refresh retained as a compact icon.
- [x] Real data, preview, logout, themes, and URL state preserved.
- [x] Same-viewport source/implementation comparison completed.

final result: passed
