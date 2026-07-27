export const paths = {
  root: "/",
  setup: "/setup/admin",
  login: "/login",
  libraries: "/settings/libraries",
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
  unavailable: "/system/unavailable",
} as const;
