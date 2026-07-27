export const paths = {
  root: "/",
  setup: "/setup/admin",
  login: "/login",
  libraries: "/settings/libraries",
  newLibrary: "/settings/libraries/new",
  browse: "/libraries/:libraryId/browse/:directoryId?",
  search: "/search",
  media: "/media/:assetId",
  generalSettings: "/settings/general",
  unavailable: "/system/unavailable",
} as const;
