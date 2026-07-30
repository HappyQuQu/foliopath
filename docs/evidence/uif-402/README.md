# UIF-402 Linux visual regression evidence

## Scope

UIF-402 converts the approved MVP regions from the latest prototype comparison
into deterministic, Linux-owned Playwright baselines. It extends
`web/tests/e2e/visual-regression.spec.ts` without adding product behavior or
mock-only production paths.

The suite fixes one desktop environment:

- Playwright `visual-chromium` project;
- Linux Chromium from `mcr.microsoft.com/playwright:v1.61.1-noble`;
- Node `22.22.0`, `1280 × 800`, English, dark theme, reduced motion;
- a fixed authenticated administrator, library, directory tree, cache/account
  documents, 24 indexed image records, and the committed synthetic
  `tests/fixtures/media/viewer-blue-violet.png`;
- fixed dates, counts, names, sizes, ETags, and readiness/session documents.

No screenshot uses a dynamic-region mask. Media decode and two animation frames
are awaited before capture; the preview state additionally waits for its short
focus/selection transition to settle.

## Baseline inventory

The Linux snapshots live beside the test in
`web/tests/e2e/visual-regression.spec.ts-snapshots`:

1. `global-header-dark-visual-chromium-linux.png`
2. `management-general-dark-visual-chromium-linux.png`
3. `management-libraries-dark-visual-chromium-linux.png`
4. `management-storage-dark-visual-chromium-linux.png`
5. `management-account-dark-visual-chromium-linux.png`
6. `browse-top-dark-visual-chromium-linux.png`
7. `browse-bottom-dark-visual-chromium-linux.png`
8. `browse-preview-dark-visual-chromium-linux.png`
9. `search-results-dark-visual-chromium-linux.png`
10. `viewer-image-dark-visual-chromium-linux.png`
11. `offline-viewer-dark-visual-chromium-linux.png`

The first ten cover UIF-402. The existing offline Viewer baseline remains in
the same suite and was re-recorded against the shared deterministic fixture.

## Verification

The baselines were generated in the Linux image and immediately compared in a
second run without snapshot updates:

```sh
FOLIOPATH_WEB_E2E_URL=http://127.0.0.1:4174 \
  npx playwright test tests/e2e/visual-regression.spec.ts \
  --project=visual-chromium --update-snapshots

FOLIOPATH_WEB_E2E_URL=http://127.0.0.1:4174 \
  npx playwright test tests/e2e/visual-regression.spec.ts \
  --project=visual-chromium
```

Result: `9 passed`. The second command proves the committed PNGs compare
without an update flag. Source/prototype comparison remains owned by
[`UIF-401`](../uif-401/README.md); UIF-402 guards subsequent production drift
and does not redefine prototype scope.

