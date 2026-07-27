# FolioPath complete static UI prototype — design QA

## Review scope

The selected option-1 visual direction is implemented as a clickable static
prototype, not an image mock. The prototype directory exposes 15 routes covering:

- first-admin setup, login/session expiry, and startup failure;
- no-library welcome, library creation, management, rename, and removal;
- current/recursive browse, search results, empty/offline/error search states;
- non-modal preview, explicit full viewer, scan/cancel/offline/failure states;
- theme, language, scan schedule, cache quota, and logout settings.

Every route was browser-rendered at `1440 × 1024` and `390 × 844`. All 15 passed
the no-page-overflow check at both sizes. Representative screens were visually
inspected in light and dark themes. Browser console warnings/errors: none.

## Evidence

- Source visual truth:
  `design-references/preview-docked-inspector-selected.png`
  (`1487 × 1058` px).
- Desktop implementation, light:
  `qa/implementation-preview-docked-light.png` (`1440 × 1024` px).
- Desktop implementation, dark:
  `qa/implementation-preview-docked-dark.png` (`1440 × 1024` px).
- Tablet bottom-workspace fallback:
  `qa/implementation-preview-tablet-dark.png` (`1024 × 768` px).
- Mobile in-flow fallback:
  `qa/implementation-preview-mobile-dark.png` (`375 × 812` browser-content
  pixels from the `390 × 844` viewport override).
- Pinned, playing-video behavior:
  `qa/implementation-preview-pinned-playing-light.png` (`1280 × 720` px).
- Normalized full-view comparison:
  `qa/preview-docked-comparison.png` (`2880 × 1024` px).

The source was scaled and padded to `1440 × 1024` at density 1 before being
placed beside the density-1 desktop implementation. The compared state is the
Kyoto directory, light theme, regular grid, first image selected, and the
non-modal preview open.

## Findings

No actionable P0, P1, or P2 differences remain.

The implementation preserves the source's hierarchy: fixed directory sidebar,
scan banner, breadcrumb/search header, compact browse toolbar, a compressed
three-column media grid, and a persistent inspector beginning below the header.
The selected thumbnail and inspector remain visible together. The production
prototype intentionally renders one dynamic pin-state explanation instead of
the source mock's simultaneous pinned and unpinned examples; this removes
duplicated instructional content without changing the chosen layout.

## Required fidelity surfaces

- Fonts and typography: passed. The implementation uses the existing
  system/Chinese UI stack, `11–15px` hierarchy, restrained weights, truncation,
  and antialiasing consistent with the source.
- Spacing and layout rhythm: passed. The desktop inspector defaults to `406px`,
  has a draggable divider, and leaves the gallery usable. Folder cards remain in
  one horizontally scrollable row when the inspector is open. Tablet uses a
  non-overlapping bottom workspace; mobile places the preview in document flow.
- Colors and visual tokens: passed. Light mode matches the selected concept;
  dark mode reuses the same semantic surfaces and interaction hierarchy. Focus,
  selection, pin, and playback states meet the existing token approach.
- Image quality and asset fidelity: passed for prototype scope. All visible
  thumbnails are local raster fixtures. Image preview uses a source-matched
  cover crop; video uses a real local MP4 fixture with native controls and a
  `16:9` stage. No CSS art, handmade SVG, or placeholder media is used.
- Copy and content: passed. Preview metadata uses Simplified Chinese and
  library-relative locations. No host paths, download/share action, or full EXIF
  inventory is exposed.
- Icons: passed. Controls use the existing Phosphor family with consistent
  weight and accessible names.
- Accessibility and behavior: passed for prototype scope. The inspector is a
  semantic complementary region rather than an `aria-modal` dialog. Close,
  pin, previous/next, fullscreen, media controls, selection, visible focus,
  reduced motion, high contrast, and forced-colors fallbacks are present.

## Comparison history

1. Initial implementation used the former full-screen viewer and blocked all
   gallery interaction. It was replaced with a workspace grid and non-modal
   inspector.
2. First docked capture allowed toolbar labels to collapse into vertical text.
   Preview-state flex bases and nowrap behavior were corrected.
3. The first inspector was `440px` wide and forced folder cards onto a second
   row. The source measured closer to `406px`; the default was corrected and
   the folder strip became one-row horizontal overflow.
4. The first image stage used contain-fit with large black bars. Image preview
   now uses the source-matched cover treatment; video remains contained in a
   native `16:9` player.
5. Tablet and mobile captures confirmed that the responsive fallbacks reduce
   gallery space or enter document flow instead of covering it.
6. The final normalized full-view comparison found no remaining actionable
   P0/P1/P2 mismatch.

## Primary interactions tested

- Clicking an image or video opens the preview without dimming or disabling the
  gallery.
- Unpinned preview follows thumbnail selection.
- Pinned preview keeps the current video source when another thumbnail is
  single-clicked, while that thumbnail still becomes selected.
- Double-clicking a thumbnail explicitly replaces the pinned preview.
- A pinned video remained actively playing while another gallery card was
  clicked (`currentTime` advanced, `paused: false`, duration `18s`).
- Previous/next, close, fullscreen, and draggable width controls render and are
  reachable.
- Theme switching persists across reload and preserves the preview anatomy.
- Desktop, tablet, and mobile responsive states were browser-rendered.
- Browser console warnings/errors: none.

## Full-screen matrix tested

| Group | Routes | Result |
| --- | --- | --- |
| Start/auth | `/setup/admin`, `/login`, `/system/unavailable` | passed |
| Library | `/welcome`, `/settings/libraries/new`, `/settings/libraries` | passed |
| Browse/search | current, recursive, results, empty/offline/error states | passed |
| Media | docked/in-flow preview and `/libraries/family/media/pagoda` | passed |
| Status/settings | scan, offline, partial failure, general settings | passed |

The new-library stepper, approved path disabled reasons, search preview,
scan-cancellation response, rename/remove dialogs, theme switch, and prototype
next/previous navigation were exercised as interactive controls. The prototype
does not simulate successful backend mutations.

## Follow-up polish

- P3: a future production slice could remember the user's preferred inspector
  width per device class.
- P3: real video fixtures could include motion rather than a still-image loop
  when motion-quality testing becomes part of an approved implementation Gate.

final result: passed
