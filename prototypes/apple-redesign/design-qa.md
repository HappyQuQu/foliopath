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

## Model acquisition and offline mapped-directory flow — 2026-08-24

### Comparison target

- Existing intelligent settings source:
  `qa/ai-integrated/14-ai-settings-1440x900.png`.
- Revised settings page:
  `qa/ai-integrated/21-model-acquisition-page-1440x900.png`.
- Mapped-directory direct-use state:
  `qa/ai-integrated/22-model-directory-direct-1440x900.png`.
- Mobile mapped-directory dialog:
  `qa/ai-integrated/23-model-directory-mobile-390x844.png`.

The page comparison uses the same light theme and 1440 × 900 CSS viewport. The
new acquisition flow reuses the existing management-center section heading,
button, dialog, segmented control, field, status and package-row language. It
does not add a separate model marketplace or expose a generic filesystem picker.

### Findings and decisions

- Online acquisition lists only sources from a signed release manifest. A
  deployment-provided mirror may appear, but the UI does not accept a temporary
  arbitrary URL.
- `/models` is a fixed read-only container mapping. Scanning displays compatible,
  installed and rejected files before any model can be loaded.
- The safe default copies a verified package into `/app/data/models`. Direct use
  is also represented, but only for a read-only mapping with a pinned SHA-256;
  removal or mutation makes the model unavailable without deleting the existing
  derived index.
- The prototype explicitly surfaces the storage tradeoff instead of presenting
  direct use and managed import as equivalent.
- No actionable P0/P1/P2 visual finding remains. The 390px dialog has no
  document-level horizontal overflow and uses internal vertical scrolling for
  the long package-management flow.

### Interaction evidence

- Online download exposes resumable progress, hash-verification and completion
  feedback.
- The model-source tabs switch between online and mapped-directory states.
- Rescan reports two compatible packages and one rejected arbitrary ONNX file.
- Copy and direct-use choices update the risk/storage explanation; applying a
  direct package pins its version and hash.
- Desktop and mobile tested states produced no browser warning or error logs.

final result: passed

## Intelligent-function management center — 2026-08-24

### Comparison target

- Existing management-center source:
  `qa/ai-integrated/13-ai-settings-source-1440x900.png`.
- Intelligent-function settings implementation:
  `qa/ai-integrated/14-ai-settings-1440x900.png`.
- Expanded model provenance state:
  `qa/ai-integrated/15-ai-settings-model-details-1440x900.png`.
- Responsive implementation:
  `qa/ai-integrated/16-ai-settings-mobile-390x844.png`.

The source and implementation use the same light theme and 1440 × 900 CSS
viewport. The new page extends the management center; it is not a standalone AI
dashboard. It therefore preserves the source header, split navigation, title
hierarchy, neutral cards, separator rhythm, compact controls and semantic status
colors.

### Findings and decisions

- Model management is presented as a verified compatibility bundle, not an
  arbitrary model file picker or remote URL field. This keeps index generations
  reproducible and makes model-license review explicit.
- Per-library controls own the user-visible capabilities. Image/video semantic
  search, tag suggestions and anonymous face grouping can be enabled separately;
  new libraries are described as off by default.
- Runtime status, package version, size, index generation and repair operations
  are visible without exposing internal filesystem paths.
- Index actions distinguish pause, generation-safe rebuild and derived-data
  clearing. Confirmation copy states what is preserved and that original media
  is never modified.
- No actionable P0/P1/P2 visual finding remains. Desktop and mobile have no
  document-level horizontal overflow.

### Interaction evidence

- Disabling the library master switch disables all three capability controls;
  re-enabling restores them.
- Environment check and model validation expose temporary progress and completion
  feedback.
- Pause/continue changes the visible index state. Rebuild and clear operations
  open separate alert dialogs; cancel and Escape close them.
- The provenance disclosure expands to show distribution, licensing and
  generation-safe upgrade rules.
- Desktop and mobile tested states produced no browser warning or error logs.

final result: passed

## Browse sidebar context switch — 2026-08-21

### Comparison target and evidence

- Source browse state before the sidebar extension:
  `qa/curation-pattern-audit/01-browse-quick-access.png`.
- Tag context implementation:
  `qa/ai-integrated/06-sidebar-tags-1440x900.png`.
- People context implementation:
  `qa/ai-integrated/07-sidebar-people-1440x900.png`.
- Mobile people drawer:
  `qa/ai-integrated/08-sidebar-people-mobile-390x844.png`.

The desktop source and implementations use a 1440 × 900 CSS viewport in the
light theme. The mobile implementation uses a 390 × 844 CSS viewport. Source
and implementation screenshots were opened together for comparison; no density
normalization was needed.

### Findings and comparison history

- The earlier implementation navigated the People quick-access item to the
  standalone AI page. That was a P1 interaction mismatch with the requested
  browse pattern.
- The revised implementation keeps Browse mounted and changes only the lower
  sidebar context. Tags replace the directory tree with a tag list; People
  replaces it with named people and anonymous groups requiring review.
- No P0/P1/P2 visual issue remains. Header, toolbar, folder cards and media cards
  retain their original geometry, and the sidebar remains independently
  scrollable when the people list exceeds the viewport.

### Required fidelity surfaces

- Typography: existing system/PingFang sizes and weights are reused.
- Spacing/layout: quick-access rows retain their 34px height; collection rows
  use the same compact sidebar rhythm and do not alter the main content width.
- Colors/tokens: selection, muted text and counts reuse existing accent and
  surface tokens.
- Image quality/assets: people rows use the existing generated face crops and
  Phosphor icons; no placeholder portraits or custom icon drawings were added.
- Copy/content: named people and anonymous groups are visibly separated, and
  anonymous groups expose confidence/review copy without claiming identity.

### Interaction evidence

- Clicking Tags changes the lower sidebar heading from “目录” to “标签” and
  reveals tag rows without navigation.
- Clicking People changes the heading to “人物” and reveals named people plus
  pending anonymous groups without navigation.
- Clicking an item keeps a single selected row with `aria-pressed` semantics.
- The same People switch works inside the 390px mobile directory drawer.

final result: passed

## AI workflow integration into current browse prototype — 2026-08-20

### Comparison target and evidence

- Current React browse source:
  `qa/ai-integrated/00-current-react-baseline-1440x900.png`.
- Browse page with AI extension entries:
  `qa/ai-integrated/01-browse-ai-entry-1440x900.png`.
- Existing non-modal preview with AI inspector:
  `qa/ai-integrated/02-media-ai-preview-1440x900.png`.
- User-confirmed tag state:
  `qa/ai-integrated/03-tag-confirmed-1440x900.png`.
- Anonymous people review destination:
  `qa/ai-integrated/04-people-review-1440x900.png`.
- Mobile directory drawer with AI entries:
  `qa/ai-integrated/05-mobile-ai-drawer-390x844.png`.

The desktop source and browse implementation use a 1440 × 900 CSS viewport,
light theme and device scale factor 1; their captured page content is 1425 × 891
pixels. The people route capture is 1440 × 900. The mobile drawer uses a
390 × 844 CSS viewport and a 375 × 812 page-content capture. No density
normalization was needed.

### Intended extension versus source

The React story is the visual source for unchanged browse structure. The AI
elements are intentional POST-MVP additions placed only in existing extension
surfaces: global-header addon, quick access, library derived-state status and
the right media inspector. Folder hierarchy, toolbars, cards and original-media
failure states remain unchanged.

### Findings and comparison history

- Pass 1 retained the earlier standalone AI dashboard. This was a P1 product
  mismatch because AI appeared to replace normal browsing.
- Pass 2 aligned the standalone search and people routes to the production
  shell, but browse did not expose the proposed workflow.
- Pass 3 added the minimal integration points. No P0/P1/P2 issue remains:
  browse geometry still matches the current React source, and the AI inspector
  scrolls inside the existing non-modal preview rather than expanding or
  obscuring persistent controls.

### Required fidelity surfaces

- Typography: all additions reuse system/PingFang typography and existing small
  control hierarchy.
- Spacing/layout: header addon, sidebar rows and preview inspector use existing
  34px/30px control heights, separators and content insets.
- Colors/tokens: semantic blue, success confirmation and muted copy use the
  existing shared tokens; no second AI palette or theme was introduced.
- Assets: the existing FolioPath mark and Phosphor icons are used. AI does not
  fake thumbnails or replace the source unavailable-media state.
- Copy/content: similarity is explicitly anonymous; tag suggestions require
  confirmation; the prototype states that no name/role is inferred and no
  original is modified.

### Interaction evidence

- Header “智能搜索” opens the semantic search route.
- Sidebar “人物归类” opens the anonymous people review route.
- Clicking a media card opens the existing right preview and exposes detected
  face and controlled-vocabulary tag suggestions.
- Accepting a tag changes it to a confirmed state and displays feedback;
  clicking again reverses only that prototype confirmation.
- Mobile keeps AI entries in the directory drawer and has no document-level
  horizontal overflow.
- Browser console errors: none.

final result: passed

## In-context tag and people management — 2026-08-24

### Comparison target and evidence

- Source browse shell and people sidebar:
  `qa/ai-integrated/09-management-source-people-sidebar-1440x900.png`.
- People management implementation:
  `qa/ai-integrated/10-people-management-1440x900.png`.
- Tag management implementation:
  `qa/ai-integrated/11-tag-management-1440x900.png`.
- Mobile people management implementation:
  `qa/ai-integrated/12-people-management-mobile-390x844.png`.

The desktop captures use a 1440 × 900 CSS viewport in the light theme. The
mobile capture uses a 390 × 844 CSS viewport and produced a 375px content
width. Source and implementation screenshots were opened together in the same
comparison pass; no density normalization was required.

### Findings and comparison history

- Pass 1 kept tag and people lists in Browse but provided no discoverable
  place for rename, merge, delete, representative-face or anonymous-group
  review. This was a P1 workflow gap.
- Pass 2 added a contextual management entry beside each sidebar list heading
  and changed only the main Browse workspace. No separate settings-style shell
  or management-center navigation was introduced.
- No P0/P1/P2 visual finding remains. The management views preserve the source
  header, responsive sidebar width, breadcrumb bar, typography, neutral card
  treatment and blue selection language. Mobile has no document-level
  horizontal overflow.

### Required fidelity surfaces

- Typography: system/PingFang hierarchy and existing small-control weights are
  reused; management headings do not introduce a new dashboard type scale.
- Spacing/layout: the existing shell remains fixed while the toolbar and media
  workspace are replaced by the content-management view. Cards and rows use
  the source surface, separator, radius and shadow rhythm.
- Colors/tokens: accent, muted, success, caution and danger states use existing
  semantic tokens.
- Image quality/assets: person cards reuse the existing fictional face crops;
  icons come from the existing Phosphor set.
- Copy/content: every destructive-looking action states that original media is
  unchanged. AI only proposes similarity groups; naming and merging remain
  explicit user decisions.

### Interaction evidence

- Tags and People expose separate contextual Manage buttons without navigation.
- Complete returns to the previous Browse workspace.
- Tag search filters to one visible matching row; inline rename and new-tag
  creation complete with status feedback.
- People filters switch between named and review groups; rename, merge,
  representative-face, create, assign and inspect actions provide feedback.
- Desktop and mobile core flows produced no browser warning or error logs.

final result: passed

## Intelligent discovery production-alignment correction — 2026-08-20

The 2026-08-18 AI addendum is retained as comparison history but its layout is
superseded. It incorrectly treated the older `apple-redesign` prototype as the
product source and introduced a feature-specific sidebar and dashboard.

### Current-run product evidence

- Current React browse story: `qa/actual-current/01-browse-1440x900.png`.
- Current React non-modal preview story:
  `qa/actual-current/02-media-preview-1440x900.png`.
- Corrected semantic-search route:
  `qa/ai-search-current-ui-1440x900.png`.
- Corrected people-review route:
  `qa/ai-people-current-ui-1440x900.png`.
- Corrected mobile search and preview:
  `qa/ai-search-current-ui-mobile-390x844.png`.

### Corrections

- Removed the separate four-item AI sidebar and prototype-only title from the
  global header.
- Integrated semantic mode into the existing search command surface.
- Kept results in the existing media collection/card rhythm and placed AI tags
  and single-face assignment inside the existing right non-modal preview.
- Kept people review as a standard work page using the same header, content
  density, selection treatment and right inspection pane.
- Kept identity decisions explicit: the model proposes anonymous similarity;
  only the user can name, merge or exclude a face.

Desktop search, people selection, tag confirmation, search-mode switching and
mobile bottom preview were exercised. The tested pages had no console errors or
document-level horizontal overflow. At narrow widths, the existing right preview
becomes a bottom inspection pane instead of inventing a second mobile navigation.

final result: passed; previous AI layout superseded

## Intelligent discovery proposal addendum — 2026-08-18

### Comparison target

- Existing FolioPath visual source:
  `../../docs/evidence/uif-408/visual/implementation/search-1440x900.jpg`.
- New prototype route: `http://127.0.0.1:4173/12-ai-features.html`.
- Same-theme combined comparison: `qa/ai-search-source-comparison.jpg`.
- Desktop captures: `qa/ai-search-dark-1440x900.png`, `qa/ai-people-1440x900.png`,
  `qa/ai-cluster-dialog-1440x900.png`.
- Mobile capture: `qa/ai-search-mobile-390x844.png`.
- Source and desktop implementation: 1440 × 900 CSS pixels, dark theme, device scale factor 1.

The target is an extension of the accepted FolioPath product language, not a pixel clone of the existing
filename-search state. Comparison therefore checks the shared global header, type hierarchy, neutral surfaces,
blue selection, filter rhythm, media-card density, dark/light token behavior and responsive shell. The new
content hierarchy is intentionally different because it must expose index coverage, similarity scores, anonymous
clusters and explicit user confirmation.

### Findings and fixes

- Pass 1 found one P1 responsive issue: the existing mobile sidebar transform moved the new four-item AI
  navigation off-screen while leaving an empty band. The AI route now explicitly returns that navigation to normal
  flow at ≤760px and hides redundant header labels.
- Pass 2 confirmed the four mobile destinations are visible, the 390px viewport has no horizontal overflow,
  and search content/card imagery remains readable. No remaining P0/P1/P2 visual finding was found.
- Generated media consists only of fictional adult subjects and is stored under `assets/ai-media/`; no real
  person name, public identity or character is claimed.
- Desktop and mobile copy consistently distinguishes visual similarity from identity, model suggestions from
  user-confirmed tags, and anonymous clusters from user-created people.

### Interaction evidence

- Semantic query submission updates the result heading without a backend.
- Accepting a tag changes it to an explicit user-confirmed manual tag state; ignoring remains separate.
- Creating a person changes the prototype count from 3 to 4 and unnamed groups from 7 to 6.
- Single-face assignment, AI feature toggles, index rebuild and clear-data boundaries all provide visible feedback.
- Dialog cancel/Escape, theme switching, hash navigation and responsive navigation were exercised.
- Browser warning/error log was empty after the tested core flows.

final result: passed

## Browse prototype current-UI synchronization — 2026-08-20

### Comparison target

- Source visual truth: current React `BrowsePage` Storybook run at
  `qa/browse-current-sync/01-current-react-1440x900.png`.
- Superseded prototype evidence:
  `qa/browse-current-sync/02-old-prototype-1440x900.png`.
- Revised implementation:
  `qa/browse-current-sync/03-synced-prototype-1440x900.png`.
- Current preview state:
  `qa/browse-current-sync/04-preview-open-1440x900.png`.
- Current mobile state:
  `qa/browse-current-sync/05-mobile-390x844.png`.
- Source preview truth:
  `qa/browse-current-sync/06-current-react-preview-1440x900.png`.

The source and revised desktop captures use the same light-theme browser state,
1440 × 900 CSS viewport and device scale factor 1. The accepted PNG content is
1425 × 891 pixels for both source and implementation because the in-app browser
excludes its scrollbar/inset. The mobile implementation uses a 390 × 844 CSS
viewport and produces a 375 × 812 content capture. No density normalization was
required.

### Comparison history

- Pass 1 found P1 structural drift: the previous prototype used an older header,
  omitted current quick access, forced the preview open, used the former folder
  counts and sort model, and did not match current responsive behavior.
- Pass 2 rebuilt the page against the running React story. A P2 remained: the
  desktop sidebar was fixed-width instead of the current `20vw` clamp, the folder
  grid showed seven columns at the wrong breakpoint, and media-card height and
  vertical rhythm differed.
- Pass 3 matched the current 288px sidebar at 1440px (256px at 1280px), 24px
  content inset, seven folder columns at 1440px, 208px media cards, and the
  current preview boundary below the persistent context and toolbar. No
  actionable P0/P1/P2 difference remains.

### Required fidelity surfaces

- Typography: system/PingFang fallback, weights, truncation and small-control
  hierarchy match the production token direction.
- Spacing/layout: header, responsive sidebar, 48px context bar, 49px toolbar,
  content inset, folder grid and media-card geometry were measured from the
  current React story.
- Colors/tokens: the existing FolioPath light/dark tokens remain the source; no
  feature-local palette or theme was added.
- Assets: the existing FolioPath brand mark and Phosphor icon library are used.
  The unavailable-thumbnail state intentionally matches the current story; no
  fake media asset was substituted.
- Copy/content: library state, quick access, breadcrumbs, scan banner, folder
  counts, file names and natural filename sort match the current fixture.

Focused comparison used exact bounding-box checks for sidebar, scan banner,
first folder card and first media card; further crops were unnecessary because
the full-view captures keep labels and control geometry readable.

### Interaction evidence

- Directory filter and media-type selection update visible counts.
- Recursive selection, layout selection, refresh feedback and tree disclosure
  states work.
- Selecting media opens the right non-modal preview; closing it restores the
  full browse workspace without disturbing the header, context bar or toolbar.
- The mobile directory drawer opens over a scrim and the 375px content viewport
  has no document-level horizontal overflow.
- Browser console errors: none.

final result: passed
