# Stage 1 authentication UI design QA

## Findings

No actionable P0, P1, or P2 differences remain in the reviewed Stage 1 states.

- Intentional contract difference: production administrator setup includes the
  required display-name field and required markers.
- Intentional safety difference: the production unavailable state replaces the
  prototype's internal path and SQLite diagnostics with a safe status summary.
- Intentional prototype-context difference: production omits the prototype directory,
  “continue viewing prototypes,” “return to prototype start,” and other prototype-only
  navigation.

## Source and implementation

- Source visual truth: `prototypes/foliopath-static-ui`, screens 1, 2, and 15.
- Implementation: `web/`, routes `/setup/admin`, `/login`,
  `/settings/general`, and `/system/unavailable`.
- Desktop captures: 1440 × 1024 CSS pixels, 1440 × 1024 source and implementation
  images, device scale factor 1.
- Mobile captures: 390 × 844 CSS pixels, 390 × 844 source and implementation images,
  device scale factor 1.
- Browser state: production Vite UI connected to a disposable real FolioPath backend
  with an empty synthetic `/library`; no developer media was read or changed.

## Full-view comparison evidence

### Desktop administrator setup

![Source and implementation at 1440 × 1024](qa/auth-comparison-1440.jpg)

The source is on the left and implementation on the right. Card width, alignment,
palette, type hierarchy, form rhythm, primary action, and read-only footer follow the
accepted direction. The implementation is taller because the API contract requires a
display name and confirmation field.

### Mobile login and expired session

![Source and implementation at 390 × 844](qa/auth-comparison-mobile.jpg)

The implementation preserves the source hierarchy and full-width form treatment. Both
captures use the same viewport and density. The implementation has no page-level
horizontal overflow.

### Desktop startup unavailable

![Source and implementation at 1440 × 1024](qa/unavailable-comparison-1440.jpg)

The source is on the left and implementation on the right. The production state keeps
the centered card, icon, hierarchy, diagnostic panel rhythm, and full-width retry
action while replacing unsafe internal diagnostics with a safe status summary.

Focused region comparisons were not needed: the 1440-pixel comparisons keep the form,
status panel, icon, type, and controls readable at their captured size, and the mobile
comparison is already a focused single-card view.

## Required fidelity surfaces

- Fonts and typography: the accepted system-font stack, weight hierarchy, line height,
  letter spacing, wrapping, and Chinese/English optical hierarchy are consistent.
- Spacing and layout rhythm: card widths, centering, padding, section gaps, control
  heights, borders, radii, and elevation are consistent across reviewed states.
- Colors and tokens: all reviewed states use the central semantic tokens; light/dark
  contrast passed axe at serious/critical severity.
- Image and asset fidelity: these states contain no raster product imagery. Visible
  icons use the shared Phosphor icon source and retain consistent size and weight.
- Copy and content: production copy follows the accepted flow while preserving the
  API-required display name and removing path, SQLite, and container diagnostics.

## Interaction and accessibility evidence

- Real backend journey: first-administrator setup, authenticated settings, logout,
  protected-route session-expired return, and login all passed.
- Readiness: contracted `application_data_unavailable` produces the safe blocking
  state without host paths, `/app/data`, SQLite text, stack traces, or raw responses.
- Theme: light/dark switching updates the document theme and exposes the inverse
  accessible action.
- Locale: English is selected from an English browser default; Simplified Chinese and
  English switching updates the whole Stage 1 interface and `html[lang]`.
- Responsive: 390, 768, 1024, and 1440 widths have no page-level horizontal overflow.
- Accessibility: automated Chromium axe checks report no serious or critical
  violations for setup, authenticated dark settings, expired login, English locale,
  and the startup unavailable state.

## Comparison history

1. Authentication comparisons passed with only the intentional API/prototype
   differences listed above.
2. The first automated axe pass found a P1 semantic defect: the named Toast viewport
   lacked a permitted role. The shared Toast owner now uses a named `region`; the
   follow-up axe pass is clean.
3. The first dark-settings axe pass found a P1 contrast defect caused by broad
   descendant selectors recoloring Button text. Descriptions and identity text now use
   explicit classes; the follow-up dark-state axe pass is clean.
4. The first unavailable-page comparison found a P2 visual drift: left-aligned content
   and a content-width retry button differed from the centered source. The
   implementation now centers the state and uses a full-width primary action; the
   post-fix comparison is `qa/unavailable-comparison-1440.jpg`.

## Residual release work

Stage 5 still owns the full target-browser matrix, final deployment image, trusted
proxy/network configuration, and release-candidate visual regression matrix.

## Stage 2 media-library and scan design QA

### Findings

No actionable P0, P1, or P2 differences remain in the reviewed Stage 2 states.

- Intentional data difference: the source library screen uses two fictional libraries
  to demonstrate scanning and offline rows; the implementation capture uses the one
  real library available in the disposable backend.
- Intentional contract difference: production exposes “view status” and “scan again”
  as separate actions and presents scan counters on a dedicated route.
- Intentional shell difference: production keeps the accepted theme control in the
  top bar and replaces the prototype's fictional online-library footer with the
  truthful read-only-media statement.

### Source and implementation

- Source visual truth: `prototypes/foliopath-static-ui`, screens 10–14.
- Implementation: `web/`, routes `/settings/libraries`,
  `/settings/libraries/:libraryId/status`, and `/settings/general`.
- Library full-view comparison:
  `qa/stage2-library-source-1440.png`,
  `qa/stage2-library-implementation-1440.png`, and
  `qa/stage2-library-comparison-1440.jpg`.
- Settings full-view comparison:
  `qa/stage2-settings-source-1440.png`,
  `qa/stage2-settings-implementation-1440.png`, and
  `qa/stage2-settings-comparison-1440.jpg`.
- Desktop library viewport and pixels: 1440 × 1024 CSS pixels, 1440 × 1024 source and
  implementation pixels, device scale factor 1.
- Desktop settings viewport: 1440 × 1024 CSS target; browser-rendered content captures
  are both 1425 × 1013 pixels after the same scrollbar/chrome exclusion, device scale
  factor 1. No cross-density scaling was used.
- Mobile implementation captures: 390 × 844 CSS pixels, device scale factor 1:
  `qa/stage2-library-implementation-mobile.png`,
  `qa/stage2-rename-dialog-mobile.png`,
  `qa/stage2-remove-dialog-mobile.png`, and
  `qa/stage2-scan-status-mobile.png`.
- Browser state: production Vite UI connected to a disposable real FolioPath backend
  and synthetic empty `/library`; no developer media was read or modified.

### Full-view and focused comparison evidence

![Library source and implementation at 1440 × 1024](qa/stage2-library-comparison-1440.jpg)

The source is on the left and production on the right. Fixed navigation, top bar,
content width, heading hierarchy, primary action, card treatment, semantic status,
and row-action alignment follow the accepted screen 13 direction.

![Settings source and implementation](qa/stage2-settings-comparison-1440.jpg)

The source is on the left and production on the right. After the comparison-driven
fix, appearance and language share one panel, the content width matches the library
screen, and scanning/cache controls remain grouped in one save transaction.

The mobile dialog and scan captures are the focused comparisons: they keep the
destructive consequences, read-only guarantee, close control, status icon, counters,
timestamps, and primary action readable at the 390-pixel breakpoint. No additional
desktop crop was needed because controls and copy are readable in the full views.

### Required fidelity surfaces

- Fonts and typography: both source and implementation use the accepted system-font
  stack, Chinese/English fallback, weight hierarchy, line height, wrapping, and
  compact control text. Long library names wrap without hiding actions.
- Spacing and layout rhythm: sidebar and header proportions, content insets, card
  radii, borders, row gaps, dialog sections, and mobile one-column rhythm align.
- Colors and tokens: all visible colors come from the central semantic tokens.
  Success, warning, danger, focus, light, and dark states remain text/icon backed and
  do not rely on color alone.
- Image quality and asset fidelity: reviewed Stage 2 screens contain no raster product
  imagery. All visible interface icons use the shared Phosphor source; no emoji,
  text-glyph close icon, handcrafted SVG, or CSS-drawn substitute remains.
- Copy and content: production copy preserves the approved read-only promise and adds
  contract-specific ETag, asynchronous removal, reliable-index, validation, and
  retry behavior without exposing host paths or raw diagnostics.

### Interaction and accessibility evidence

- Real backend journey: create → rename → manual scan → status → save schedule/cache
  settings → logout/login → asynchronous remove passed in Chromium.
- Removal dialog enumerates configuration, index/directories, jobs, and reconstructible
  cache, then explicitly states that originals are not deleted, moved, or modified.
- Scan states implement queued, running, cancelling, succeeded, failed, cancelled,
  offline, and interrupted copy. Failed/offline/interrupted/cancelled states retain a
  visible reliable-index preservation message.
- Mobile at 390 × 844 keeps four library actions reachable, renders dialogs within the
  viewport, exposes semantic close controls, and keeps the scan page in one column.
- Theme/locale route behavior and automated axe serious/critical checks remain covered
  by the real-backend browser test.
- Browser console inspection after the final library/status/settings journey found no
  warnings or errors beyond Vite connection and React development informational logs.
- Success and error toasts now auto-dismiss after six seconds and remain manually
  dismissible, so transient feedback cannot indefinitely cover later actions.

### Comparison history

1. The first Stage 2 settings comparison found a P2 density/layout mismatch:
   appearance and language were split into separate narrow cards, pushing scanning,
   cache, and account actions below the source's first-view rhythm. Production now
   combines appearance/language and uses the shared content width. The post-fix
   evidence is `qa/stage2-settings-comparison-1440.jpg`.
2. The first completed-scan mobile capture found a P2 state mismatch: a succeeded scan
   with a null backend ratio rendered an indeterminate progress bar. Succeeded scans
   now force the semantic and visual progress value to 100%; terminal failure states
   retain their last reported ratio.
3. The first real-backend E2E pass found a P1 interaction obstruction: a success toast
   persisted over the logout action. The canonical Toast owner now auto-dismisses
   after six seconds, uses the shared Phosphor close icon, and has a regression test;
   the follow-up E2E passed.

### Residual Stage 2 work

`S2-207～208` still own the exhaustive long-name/path, repeated-submit, offline,
partially unreadable, cancellation timing, keyboard matrix, and final Integrated Done
Gate. These are coverage gaps, not remaining visual mismatches in the reviewed states.

final result: passed
