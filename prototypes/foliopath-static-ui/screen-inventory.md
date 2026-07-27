# FolioPath MVP prototype screen inventory

This inventory records the implemented review surface for the complete static
prototype. It is derived from `docs/product-requirements.md`,
`docs/user-flows.md`, and `docs/ui-design.md`; it does not add backend contracts
or claim that the corresponding production feature exists.

| Group | Review screen or state | Prototype route | MVP source |
| --- | --- | --- | --- |
| Start | Create the first administrator | `/setup/admin` | FR-AUTH-001, flow 1 |
| Start | Sign in and session-expired messaging | `/login` | FR-AUTH-001, flow 1 |
| Start | No-library welcome and Docker mount help | `/welcome` | FR-DEP-004, flow 1 |
| Libraries | New-library details, approved path picker, and review | `/settings/libraries/new` | FR-LIB-001/004/005, flow 2 |
| Browse | Current-directory gallery and non-modal preview | `/libraries/family/browse/kyoto` | FR-BRW-001–009, flow 4 |
| Browse | Recursive gallery with source paths | `/libraries/family/browse/kyoto?recursive=1` | FR-BRW-003/009, flow 4 |
| Search | Results with scope, type, date, and source paths | `/libraries/family/search?q=京都` | FR-SRH-001–004, flow 5 |
| Search | Empty, failed, and offline search states | `/libraries/family/search/empty` | FR-BRW-007, flow 5 |
| Media | Explicit full viewer with basic information | `/libraries/family/media/pagoda` | FR-MED-004–007, flow 6 |
| Status | First/full scan detail and cooperative cancellation | `/status/scanning` | FR-SCN-001/003/008, flow 3 |
| Status | Offline library with retained index | `/status/offline` | FR-LIB-006, flow 3 |
| Status | Partial failure and interrupted scan | `/status/error` | FR-SCN-003/004, flow 3 |
| Settings | Library list, rename, scan actions, and removal confirmation | `/settings/libraries` | FR-LIB-006–008, flow 7 |
| Settings | Scan schedule, cache quota, language, theme, session | `/settings/general` | FR-SCN-007, FR-MED-008, FR-UI-006 |
| System | Data/startup failure and API-disconnected recovery | `/system/unavailable` | FR-DEP-004, FR-BRW-007 |

All routes are available from the floating prototype-only screen directory. Each
production route must support the shared light/dark tokens. Browse, search,
settings, viewer, and status routes must remain usable at desktop, tablet, and
mobile breakpoints. The directory is not part of the FolioPath product
information architecture.
