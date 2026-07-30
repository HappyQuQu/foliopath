# Apple redesign prototype — Design QA

## Operations and maintenance addendum — 2026-07-30

- Task center: `qa/operations-task-center-final.png`
- Task detail: `qa/operations-task-detail-final.png`
- System maintenance: `qa/operations-maintenance-final.png`
- Routes:
  - `http://127.0.0.1:4173/08-settings-storage.html`
  - `http://127.0.0.1:4173/11-task-detail.html?id=scan-work`
  - `http://127.0.0.1:4173/10-settings-maintenance.html`

The scan and cache page now presents one operational overview rather than a
collection of unrelated settings. Active, historical and attention-required
filters change the visible task set; cache completion and full rebuild are
distinct actions, and full rebuild requires confirmation. Task detail preserves
the global Header and management navigation while exposing progress, execution
stages, cancellation/retry and reliable-state safety copy.

The maintenance page is an independent destination. Health, integrity reports,
application-data backup, retention and sanitized diagnostics are separated from
ordinary preferences and library configuration. Integrity checks and backup
creation transition through running states and settle into persistent results.
All prototype mutations remain browser-local.

Browser verification covered semantic headings and navigation, all three task
filters, the offline task detail, a completed integrity-check cycle, the backup
confirmation and completion cycle, and a 1280px viewport with no document-level
horizontal overflow. `node --check`, `git diff --check`, prototype link checks
and a fresh-page DOM inspection completed without reported errors.

## Brand alignment addendum — 2026-07-30

- Authoritative visual source:
  `../../web/public/foliopath-mark-tree.svg`
- Prototype asset:
  `assets/foliopath-mark-tree.svg`
- Combined source/implementation comparison:
  `qa/logo-brand-comparison.jpg`
- Desktop header:
  `qa/logo-header-desktop.png` — `1440 × 900`
- Mobile header:
  `qa/logo-header-mobile.png` — 390px CSS viewport
- Dark theme:
  `qa/logo-header-dark.png` — `1440 × 900`
- Authentication:
  `qa/logo-auth-desktop.png` — `900 × 760`
- First-run welcome:
  `qa/logo-welcome-desktop.png` — `900 × 760`

The prototype now uses the confirmed directory-path “F” asset without changing
its three paths, colors or proportions. The desktop global Header uses the
specified 28px mark with real `FolioPath` text; at 390px the text is hidden and
the 28px mark remains visible. Authentication and first-run screens use the
specified 64px standalone mark without an added tile. The same SVG is used as
the favicon on every prototype route.

Runtime checks confirmed that the SVG loaded with non-zero natural dimensions,
the desktop and mobile slots measured 28 × 28px, standalone slots measured
64 × 64px, and the tested 1440px and 390px states had no document-level
horizontal overflow. The welcome page contains one global Header after removing
the legacy duplicate shell. A fresh management-page load produced no browser
console warnings or errors.

No actionable brand-fidelity, responsive, theme or asset-loading findings
remain.

## Management center independent pages

## Comparison target

- Source visual truth: the accepted management-navigation reference preserved in
  `qa/management-navigation-source-comparison.png`.
- Source pixels: `271 × 446`.
- Implementation routes:
  - `http://127.0.0.1:4173/06-settings-libraries.html`
  - `http://127.0.0.1:4173/07-settings-general.html`
  - `http://127.0.0.1:4173/08-settings-storage.html`
  - `http://127.0.0.1:4173/09-settings-account.html`
- Final implementation screenshots:
  - `qa/management-libraries-final.png` — `1440 × 900`
  - `qa/management-general-final.png` — `1440 × 900`
  - `qa/management-storage-final.png` — `1425 × 891`
  - `qa/management-account-final.png` — `1425 × 891`
- Full four-page evidence:
  `qa/management-pages-final-contact-sheet.png`
- Focused navigation comparison:
  `qa/management-navigation-source-comparison.png`
- Browser CSS viewport: `1440 × 900`, device scale factor 1. The in-app browser
  reserved 15 × 9 pixels on pages with a vertical scrollbar.
- Focus normalization: the implementation navigation/header crop was resized to
  the source `271 × 446` pixels before side-by-side comparison.
- State: Simplified Chinese, light/system theme, default management data.

## Full-view and focused comparison evidence

The source establishes a compact FolioPath management navigation with one active
row and four peer destinations. The implementation preserves that hierarchy and
selected-state treatment while honoring the previously accepted full-width
global Header. Each destination now opens a standalone page with page-specific
content instead of a same-document anchor.

- Fonts and typography: the source and implementation use the same Apple/system
  and PingFang-style hierarchy. Sidebar label weight, 44px row rhythm, section
  headings, helper copy and form labels remain readable without awkward wrapping.
- Spacing and layout rhythm: desktop retains the 216–280px adaptive sidebar and
  full remaining content width. At 720px and below, navigation becomes a
  horizontal category strip; page content remains single-column.
- Colors and tokens: active blue, neutral fills, separators, status colors,
  disabled controls, focus rings and danger actions reuse the existing FolioPath
  token system.
- Image and icon fidelity: the source contains no raster imagery. Existing
  FolioPath navigation icons are reused consistently; no placeholder imagery,
  decorative art or generated assets were introduced.
- Copy and content: each page has a distinct purpose and standalone explanation.
  Read-only media, reliable-index preservation and cache safety language remain
  explicit at the point of action.
- Accessibility: navigation uses links and `aria-current`; forms use labels,
  field errors and live status; dialogs use dialog semantics and Escape; status
  is expressed with text as well as color.

## Findings

No actionable P0, P1 or P2 visual or interaction findings remain.

## Comparison history

### Pass 1 — issues identified

- P1: “扫描与缓存”和“账户” were same-page anchors inside General rather
  than independent destinations.
- P1: library action buttons, settings controls and account actions were static
  chrome, so the prototype could not define validation, success, failure or
  destructive flows for implementation.
- P2: the existing page descriptions mixed responsibilities and did not expose
  saved/dirty state.

### Pass 2 — fixes applied

- Added four independent URLs with a shared, single-level navigation.
- Added persisted prototype state and complete create, rename, scan, cancel,
  retry, remove, save, reset, quota, cache-clear, profile, password and logout
  flows.
- Added field-level errors, disabled save states, progress, persistent faults,
  confirmation dialogs and transient success messages.

### Pass 3 — issue identified and fixed

- P2: the first removal dialog required typing the library name, conflicting with
  the accepted single explicit confirmation flow. The extra text gate was
  removed while retaining clear consequences and original-media safety copy.
- P2: the new management requirement initially reused `FR-UI-008`, already owned
  by the Post-MVP video storyboard feature. It was corrected to `FR-UI-009`, with
  scope revision 3 and traceability updated.

### Pass 4 — passed

- Focused navigation evidence:
  `qa/management-navigation-source-comparison.png`
- Full page evidence:
  `qa/management-pages-final-contact-sheet.png`
- Mobile evidence:
  `qa/management-libraries-mobile.png`,
  `qa/management-storage-mobile.png`,
  `qa/management-library-dialog-mobile.png`

## Interaction and runtime verification

- Navigation: clicked between all four categories; each produced a distinct URL,
  page heading and active navigation item.
- General: previewed dark theme, saved light/theme, masonry and preview changes,
  reloaded, and confirmed persisted values and cleared dirty state.
- Libraries: created a library from an allowed path; duplicate-name validation
  blocked rename; valid rename succeeded; scan state progressed; cancel and
  offline retry were available; safe removal removed only prototype-derived
  state.
- Scan and cache: rejected interval `0`, saved `48`, rejected quota `0`, saved
  `20`, cleared cache after confirmation, and started/cancelled a library scan.
- Account: saved a display name, rejected a short/mismatched password, accepted a
  valid 8+ character Unicode password, cleared password inputs after success, and
  opened/cancelled logout confirmation.
- Dialogs: Escape closed the add-library dialog; focus returned to a button.
- Responsive matrix: all four pages at `1280`, `1024`, `768`, `720` and `390`
  pixels; 20 combinations had no document-level horizontal overflow.
- Mobile dialog: width matched the 390px viewport and remained fully contained.
- Browser console warnings and errors: none.

## Implementation checklist

- [x] Four independent, directly addressable management pages.
- [x] Working navigation and active-page semantics.
- [x] Complete general, library, scan/cache and account prototype flows.
- [x] Validation, dirty, disabled, success, persistent failure and danger states.
- [x] Desktop, tablet and mobile responsive verification.
- [x] PRD, UI spec, user flow, scope, change record and traceability updates.

final result: passed
