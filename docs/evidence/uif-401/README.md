# UIF-401 per-page prototype / production comparison

This evidence freezes the twelve pages in
`web/qa/visual-reference-manifest.json` at one shared comparison input:
Simplified Chinese, dark theme, and a `1280 × 720` CSS viewport.

- Open `index.html` for the combined comparison artifact.
- `source/` contains the latest prototype route or approved page-family state.
- `implementation/` contains the corresponding production route rendered
  against a real first-run administrator, read-only library, completed scan,
  catalog, search, preview, and viewer backend.
- `comparison/` contains the browser-captured combined cards used for the final
  visible review.

The browser excludes its 15px scrollbar gutter from PNG content when a page
scrolls. Raw captures can therefore be `1265 × 712` or `1280 × 720` while still
sharing the same `1280 × 720` CSS viewport. The comparison artifact presents
both sides at the same review scale; raw files remain available for 1:1
inspection.

## State mapping

| Page | Prototype state | Production state |
| --- | --- | --- |
| auth-login | default login | default login |
| auth-setup | shared auth shell | first-run administrator form |
| welcome | first-run welcome | authenticated first-run empty library |
| libraries | configured libraries | scanning library |
| library-new | add-library dialog | independent name step |
| library-status | scanning library card | completed scan detail |
| general | default general settings | persisted default settings |
| storage | ready scan/cache settings | ready cache summary and scan task |
| account | profile/password/session | real administrator profile/password/session |
| browse | Kyoto directory with open preview | Kyoto directory with five child directories, four indexed images, and open preview |
| search | “Kyoto” results | “Kyoto” path results from the same synthetic tree |
| viewer | image viewer | indexed image viewer |

The shared-shell rows for `auth-setup`, `library-new`, and `library-status`
compare an approved page-family prototype with an independently navigable
production child page. Their functional split is deliberate and is governed
by the accepted feature plan; the comparison checks the inherited shell,
hierarchy, spacing, component language, and safety copy rather than asserting
that a dialog and a route are the same DOM.

## History

1. UIF-301 established the shared shell, token, global Header, and management
   navigation comparison.
2. UIF-312 closed the Browse current-directory filter placement and bottom
   blank-space defects.
3. UIF-315 closed Viewer desktop/mobile composition and interaction findings.
4. UIF-316 and UIF-317 closed shared async states, locale, theme, breakpoint,
   contrast, long-name overflow, and duplicate-submit findings.
5. UIF-401 rendered every manifest entry through the same in-app browser
   viewport and a real backend fixture, then reviewed the combined inputs.

No P0, P1, or P2 visual finding remains. There is no deferred P3 item in this
comparison.
