# FolioPath static UI prototype

This directory is a disposable, frontend-only design prototype. It is isolated
from the production `web/src` import graph and does not call FolioPath APIs.

The prototype demonstrates the complete accepted MVP interface surface with
local fixture data. Use the floating **原型目录** to review all 15 screens and
states:

- adaptive light and dark themes with a remembered local preference;
- first-admin setup, login/session expiry, no-library welcome, and startup failure;
- a three-step approved-directory library wizard and library management dialogs;
- fixed desktop directory sidebar, tablet drawer, and full-width mobile drawer;
- responsive folder and media grids;
- current/recursive browse, search results, empty/offline/error states, and a full viewer;
- scanning, cancellation, offline retained-index, partial-failure, and general settings;
- a resizable, non-modal preview workspace that keeps the gallery usable;
- image detail display plus playable local video fixtures;
- pinned preview behavior where single-click selects and double-click replaces
  the pinned item, plus previous/next navigation and a responsive bottom-dock
  fallback without a page-covering backdrop.

Run it locally with:

```sh
npm install
npm run dev
```

Design references live in `design-references/`, browser acceptance evidence lives
in `qa/`, the route-level review list is `screen-inventory.md`, and the final
comparison report is `design-qa.md`.
