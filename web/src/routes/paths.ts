export const paths = {
  root: "/",
  setup: "/setup/admin",
  login: "/login",
  libraries: "/settings/libraries",
  libraryInlineStatus: (libraryId: string) =>
    `/settings/libraries?${new URLSearchParams({ status: libraryId }).toString()}`,
  libraryScanRecords: (libraryId?: string) =>
    `/settings/libraries?${new URLSearchParams({
      view: "scans",
      ...(libraryId ? { libraryId } : {}),
    }).toString()}`,
  libraryProcessingResults: (libraryId?: string) =>
    `/settings/libraries?${new URLSearchParams({
      view: "results",
      ...(libraryId ? { libraryId } : {}),
    }).toString()}`,
  libraryStatus: (libraryId: string) => `/settings/libraries/${libraryId}/status`,
  libraryStatusPattern: "/settings/libraries/:libraryId/status",
  newLibrary: "/settings/libraries/new",
  browse: (libraryId: string, directoryId?: string) =>
    `/libraries/${encodeURIComponent(libraryId)}/browse${
      directoryId ? `/${encodeURIComponent(directoryId)}` : ""
    }`,
  browsePattern: "/libraries/:libraryId/browse/:directoryId?",
  search: "/search",
  librarySearch: (libraryId: string) =>
    `/libraries/${encodeURIComponent(libraryId)}/search`,
  librarySearchPattern: "/libraries/:libraryId/search",
  media: (libraryId: string, assetId: string, returnTo?: string) => {
    const pathname = `/libraries/${encodeURIComponent(libraryId)}/media/${encodeURIComponent(assetId)}`;
    return returnTo
      ? `${pathname}?${new URLSearchParams({ from: returnTo }).toString()}`
      : pathname;
  },
  mediaPattern: "/libraries/:libraryId/media/:assetId",
  generalSettings: "/settings/general",
  storageSettings: "/settings/storage",
  accountSettings: "/settings/account",
  logsSettings: "/settings/logs",
  aboutSettings: "/settings/about",
  generalSettingsForLibrary: (libraryId: string) =>
    `/settings/general?${new URLSearchParams({ libraryId }).toString()}`,
  unavailable: "/system/unavailable",
} as const;
