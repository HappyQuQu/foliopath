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

## Stage 2 implemented-slice check

- Compared the confirmed `/welcome` source and production empty-library state at
  1440×1024 in the same browser. Sidebar width, top bar, dashed panel, icon treatment,
  typography, spacing, and primary/secondary actions align after the token correction.
- Checked the production welcome and three-step creation flow at 390×844 with no
  page-level horizontal overflow. The mobile navigation opens with a semantic button,
  closes with Escape, and restores focus.
- Exercised the real authenticated backend from the UI: name → approved `/library`
  root → safety review → CSRF/idempotent create → initial full-scan admission → library
  list. No host path or raw backend diagnostic was displayed.
- The production shell intentionally replaces the prototype's fictional online-library
  footer with the truthful read-only-media statement and keeps the accepted theme
  control in the top bar.

final result: passed
