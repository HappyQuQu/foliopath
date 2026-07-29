# Apple redesign implementation QA

## Source and implementation

- Visual source of truth: `prototypes/apple-redesign/03-browse.html`
- Source capture: `web/qa/apple-redesign-browse-source-current.jpg`
- Functional implementation: `http://127.0.0.1:5176/libraries/lib_1/browse/dir_16`
- Implementation capture: `web/qa/apple-redesign-browse-implementation-current.jpg`
- Both captures use the same 1265 × 712 desktop viewport and light theme.

## What was copied

- 272px fixed library sidebar, nested directory tree, bottom navigation, and compact library status.
- 52px breadcrumb/search/action bar and 52px browse toolbar.
- Apple-style semantic light/dark tokens, type scale, radii, borders, and restrained shadows.
- Three-column folder-card layout when the docked preview is open.
- Docked preview hierarchy: title and fixed-preview control, 296px media stage, previous/next controls, viewer action, filename/type, details, and pin state.
- The production page now uses the prototype directory hierarchy in QA: `家庭影像 / 旅行 / 日本 / 京都`, seven child folders, and twelve media items.

## Functional behavior retained

- Real libraries, directory queries, cursor-backed media queries, URL state, and indexed thumbnails.
- Search submission, filters navigation, recursive toggle, layout selection, sorting, preview open/close, previous/next, pinning, resizing, viewer navigation, theme switching, and keyboard focus behavior.
- Real ready/scanning/offline/error states. The scan banner is shown only while the backend reports `scanning`; the reference capture happens to show that state.
- Natural directory ordering and real indexed counts remain authoritative, so sample counts and the prototype's hand-arranged folder order are not hard-coded into product behavior.

## Visual comparison result

The final implementation capture matches the source structure and proportions. The remaining visible differences are data-driven: real media replaces the prototype gradients, real counts replace sample counts, and the scan banner is absent after the QA scan completes.

## Verification

- TypeScript checks: passed.
- Frontend unit/component tests: 79 passed.
- Production Vite build: passed.
- Browser QA: nested directory restoration, real thumbnail rendering, preview opening, light theme, and docked desktop layout verified.

final result: passed
