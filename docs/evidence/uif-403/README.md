# UIF-403 real vertical browser chain

## Scope

UIF-403 extends the existing production browser acceptance chain instead of
creating a parallel mock application. The test starts a fresh FolioPath
container, writable application-data directory, and one read-only `/library`
mount, then drives the React application through the real generated client and
HTTP backend.

The primary Chromium chain now covers:

1. first-administrator setup;
2. library creation from a server-approved directory and initial scan;
3. rename, manual rescan, status, cancellation and preserved reliable-index
   states;
4. Browse direct/recursive media, current-directory `q` filtering, sort,
   layout persistence, preview pinning/resizing and full Viewer;
5. library and global Search, result preview and Viewer return focus;
6. General preferences, administrator display-name update and password change;
7. scheduled-scan/cache quota update and confirmed reconstructible-cache
   cleanup;
8. logout, protected-route redirect, login with the changed password, locale
   round trip, and safe library removal.

The error, offline, thumbnail-pending/failure and pagination fixtures in the
same file remain supplemental failure-state checks. They do not replace the
real setup, account, library, scan, catalog, search, media-content, settings,
cache-cleanup, session, or removal operations above.

## Original-media invariant

`tests/e2e/web_auth.sh` records two manifests before the application starts:

- every media path under the synthetic `/library` tree;
- the SHA-256 digest of every original media file.

After the complete browser suite, it records both manifests again and requires
byte-for-byte equality with `cmp`. The application container sees the same
tree only through a read-only bind mount. The E2E run therefore fails if a path
or original-media byte changes.

## Verification

Executed from the repository root:

```sh
FOLIOPATH_E2E_SUITE=chromium tests/e2e/web_auth.sh
```

Result:

- Chromium and applicable mobile-Chromium project: `6 passed`, `3 skipped`;
- primary administrator/library/browse/search/settings/cache/session vertical
  slice passed against a fresh database and real backend;
- real scanned video storyboard flow passed;
- media path and SHA-256 comparisons passed;
- the script printed `chromium browser e2e suite passed`.

The skipped cases are desktop/mobile applicability skips already declared by
the media interaction matrix, not skipped UIF-403 steps.

