# Authentication UI design QA

## Result

Passed for the implemented Stage 1 authentication slice. No P1 or P2 visual,
responsive, or interaction defects remain in the reviewed states. Automated browser
E2E and the full release breakpoint/locale/accessibility matrix remain tracked by
`S1-206` and `S1-207`; they are not claimed by this report.

## Scope and sources

- Design source: `prototypes/foliopath-static-ui`, screens 1 and 2 (administrator
  setup and login/session expired).
- Implementation: `web/`, routes `/setup/admin`, `/login`, and
  `/settings/general`.
- Desktop comparison: 1440 × 1024 CSS pixels at device scale 1.
- Mobile comparison: 390 × 844 CSS pixels at device scale 1.
- Browser state: production Vite UI connected to a disposable real FolioPath backend
  with an empty synthetic `/library`; no developer media was read or changed.

## Comparison evidence

### Desktop administrator setup

![Source and implementation at 1440 × 1024](qa/auth-comparison-1440.jpg)

The source is on the left and the implementation is on the right. Card width,
alignment, color, type hierarchy, form rhythm, primary action, and read-only media
footer match the accepted direction. The production setup form is taller because the
accepted API contract requires a display name in addition to username and password.
The prototype-only “continue” action and prototype directory are intentionally absent.

### Mobile login and expired session

![Source and implementation at 390 × 844](qa/auth-comparison-mobile.jpg)

The source is on the left and the implementation is on the right. Both pages preserve
the content hierarchy and full-width form treatment. The implementation has no
page-level horizontal overflow (`scrollWidth === innerWidth === 390`). The production
form starts slightly higher because prototype navigation is absent.

## Interaction and safety verification

- Real backend: created the first administrator, reached general settings, logged out,
  returned to login, logged in again, and restored the authenticated settings page.
- Session state: `/login?reason=session_expired` shows a dismissible safe notice without
  exposing backend details.
- Theme: switching to dark changed the document theme and browser color scheme; the
  toggle then exposed the inverse “switch to light” action and returned to light.
- Responsive: setup/login were checked at 1440 × 1024 and login at 390 × 844; mobile
  produced no horizontal overflow.
- Production-only differences: required display-name field and required markers.
  Prototype-only differences: prototype directory and “continue viewing prototypes”
  controls. These are intentional contract/context differences, not regressions.

## Iteration record

1. Matched the accepted prototype using the central token source, canonical shared
   fields/buttons/status/theme controls, and the Phosphor icon set.
2. Compared desktop source and implementation in one image; retained the
   contract-required display-name field and removed prototype navigation.
3. Compared the expired-session login state at an identical mobile viewport; verified
   overflow and theme behavior.
4. Re-ran the real setup/logout/login journey and left the implementation on the
   authenticated general settings page for inspection.
